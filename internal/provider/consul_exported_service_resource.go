// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sync"

	api "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConsulExportedServiceResource{}
var _ resource.ResourceWithImportState = &ConsulExportedServiceResource{}

// Allows for modification of exported-service only once at a time
var exportedServiceLock sync.Mutex

func readExportedServices(client *api.Client) *api.ExportedServicesConfigEntry {
	configEntry, _, err := client.ConfigEntries().Get("exported-services", "default", nil)

	if err != nil {
		return &api.ExportedServicesConfigEntry{
			Name: "default",
		}
	}

	return configEntry.(*api.ExportedServicesConfigEntry)
}

func writeExportedServices(client *api.Client, configEntry *api.ExportedServicesConfigEntry) error {
	var err error

	if len(configEntry.Services) == 0 {
		_, err = client.ConfigEntries().Delete("exported-services", "default", nil)
	} else {
		_, _, err = client.ConfigEntries().Set(configEntry, nil)
	}

	return err
}

func NewConsulExportedServiceResource() resource.Resource {
	return &ConsulExportedServiceResource{}
}

// ConsulExportedServiceResource defines the resource implementation.
type ConsulExportedServiceResource struct {
	client *api.Client
}

// ConsulExportedServiceResourceModel describes the resource data model.
type ConsulExportedServiceResourceModel struct {
	PeerName        types.String `tfsdk:"peer_name"`
	ServiceToExport types.String `tfsdk:"service_to_export"`
	Id              types.String `tfsdk:"id"`
}

func (r *ConsulExportedServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_consul_exported_service"
}

func (r *ConsulExportedServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Consul exported service resource",

		Attributes: map[string]schema.Attribute{
			"peer_name": schema.StringAttribute{
				MarkdownDescription: "Name of the peer to export the service to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_to_export": schema.StringAttribute{
				MarkdownDescription: "The name of the service to export",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Exported peer identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ConsulExportedServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	createClient := req.ProviderData.(func(diagnostics *diag.Diagnostics) (*api.Client, error))

	client, err := createClient(&resp.Diagnostics)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ConsulExportedServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConsulExportedServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	exportedServiceLock.Lock()
	defer exportedServiceLock.Unlock()

	exportedServiceConfigEntry := readExportedServices(r.client)

	inserted := false

	newConsumer := api.ServiceConsumer{
		Peer: data.PeerName.ValueString(),
	}

	for idx := range exportedServiceConfigEntry.Services {
		if exportedServiceConfigEntry.Services[idx].Name == data.ServiceToExport.ValueString() {
			exportedServiceConfigEntry.Services[idx].Consumers = append(exportedServiceConfigEntry.Services[idx].Consumers, newConsumer)
			inserted = true
		}
	}

	if !inserted {
		exportedServiceConfigEntry.Services = append(exportedServiceConfigEntry.Services, api.ExportedService{
			Name: data.ServiceToExport.ValueString(),
			Consumers: []api.ServiceConsumer{
				newConsumer,
			},
		})
	}

	err := writeExportedServices(r.client, exportedServiceConfigEntry)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to write exported services, got error: %s", err))
		return
	}

	data.Id = types.StringValue(fmt.Sprintf("%s_%s", data.PeerName.ValueString(), data.ServiceToExport.ValueString()))

	tflog.Debug(ctx, "exported service")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsulExportedServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConsulExportedServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	exportedServiceConfigEntry := readExportedServices(r.client)

	for _, service := range exportedServiceConfigEntry.Services {
		if service.Name == data.ServiceToExport.ValueString() {
			for _, consumer := range service.Consumers {
				if consumer.Peer == data.PeerName.ValueString() {
					data.Id = types.StringValue(fmt.Sprintf("%s_%s", data.PeerName.ValueString(), data.ServiceToExport.ValueString()))
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *ConsulExportedServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConsulExportedServiceResourceModel
	var oldData ConsulExportedServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &oldData)...)

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	exportedServiceLock.Lock()
	defer exportedServiceLock.Unlock()

	exportedServiceConfigEntry := readExportedServices(r.client)

	removeExportedService(exportedServiceConfigEntry, oldData.ServiceToExport.ValueString(), oldData.PeerName.ValueString())

	newConsumer := api.ServiceConsumer{
		Peer: data.PeerName.ValueString(),
	}

	inserted := false

	for idx := range exportedServiceConfigEntry.Services {
		if exportedServiceConfigEntry.Services[idx].Name == data.ServiceToExport.ValueString() {
			exportedServiceConfigEntry.Services[idx].Consumers = append(exportedServiceConfigEntry.Services[idx].Consumers, newConsumer)
			inserted = true
		}
	}

	if !inserted {
		exportedServiceConfigEntry.Services = append(exportedServiceConfigEntry.Services, api.ExportedService{
			Name: data.ServiceToExport.ValueString(),
			Consumers: []api.ServiceConsumer{
				newConsumer,
			},
		})
	}

	err := writeExportedServices(r.client, exportedServiceConfigEntry)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to write exported services, got error: %s", err))
		return
	}

	data.Id = types.StringValue(fmt.Sprintf("%s_%s", data.PeerName.ValueString(), data.ServiceToExport.ValueString()))

	tflog.Debug(ctx, "exported service")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsulExportedServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConsulExportedServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	exportedServiceLock.Lock()
	defer exportedServiceLock.Unlock()

	exportedServiceConfigEntry := readExportedServices(r.client)

	removeExportedService(exportedServiceConfigEntry, data.ServiceToExport.ValueString(), data.PeerName.ValueString())

	err := writeExportedServices(r.client, exportedServiceConfigEntry)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to write exported services, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func removeExportedService(exportedServiceConfigEntry *api.ExportedServicesConfigEntry, serviceToRremove, consumerToRemove string) {
	var serviceToRemoveIdx int

	for idx_services, service := range exportedServiceConfigEntry.Services {
		if service.Name == serviceToRremove {
			serviceToRemoveIdx = idx_services
			break
		}
	}

	var consumerToRemoveIdx int

	for idx_consumers, consumer := range exportedServiceConfigEntry.Services[serviceToRemoveIdx].Consumers {
		if consumer.Peer == consumerToRemove {
			consumerToRemoveIdx = idx_consumers
			break
		}
	}

	newConsumers := append(exportedServiceConfigEntry.Services[serviceToRemoveIdx].Consumers[:consumerToRemoveIdx], exportedServiceConfigEntry.Services[serviceToRemoveIdx].Consumers[consumerToRemoveIdx+1:]...)

	if len(newConsumers) == 0 {
		exportedServiceConfigEntry.Services = append(exportedServiceConfigEntry.Services[:serviceToRemoveIdx], exportedServiceConfigEntry.Services[serviceToRemoveIdx+1:]...)
	} else {
		exportedServiceConfigEntry.Services[serviceToRemoveIdx].Consumers = newConsumers
	}
}

func (r *ConsulExportedServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
