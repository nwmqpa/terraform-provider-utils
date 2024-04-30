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

var singleIntentionMutexes map[string]*sync.Mutex
var singleIntentionMutexesLock sync.Mutex

func getMutexForSingleIntention(id string) *sync.Mutex {
	singleIntentionMutexesLock.Lock()
	defer singleIntentionMutexesLock.Unlock()

	if singleIntentionMutexes == nil {
		singleIntentionMutexes = make(map[string]*sync.Mutex)
	}

	if _, ok := singleIntentionMutexes[id]; !ok {
		singleIntentionMutexes[id] = &sync.Mutex{}
	}

	mutexToHangOn := singleIntentionMutexes[id]

	return mutexToHangOn
}

func readServiceIntentions(client *api.Client, serviceName string) *api.ServiceIntentionsConfigEntry {
	configEntry, _, err := client.ConfigEntries().Get("service-intentions", serviceName, nil)

	if err != nil {
		return &api.ServiceIntentionsConfigEntry{
			Kind: "service-intentions",
			Name: serviceName,
		}
	}

	return configEntry.(*api.ServiceIntentionsConfigEntry)
}

func writeServiceIntentions(client *api.Client, configEntry *api.ServiceIntentionsConfigEntry) error {
	var err error

	if len(configEntry.Sources) == 0 {
		_, err = client.ConfigEntries().Delete("service-intentions", configEntry.Name, nil)
	} else {
		_, _, err = client.ConfigEntries().Set(configEntry, nil)
	}

	return err
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConsulSingleIntentionResource{}
var _ resource.ResourceWithImportState = &ConsulSingleIntentionResource{}

// Allows for modification of exported-service only once at a time

func NewConsulSingleIntentionResource() resource.Resource {
	return &ConsulSingleIntentionResource{}
}

// ConsulSingleIntentionResource defines the resource implementation.
type ConsulSingleIntentionResource struct {
	client *api.Client
}

// ConsulSingleIntentionResourceModel describes the resource data model.
type ConsulSingleIntentionResourceModel struct {
	DestinationService types.String `tfsdk:"destination_service"`
	SourceService      types.String `tfsdk:"source_service"`
	SourcePeer         types.String `tfsdk:"source_peer"`
	Id                 types.String `tfsdk:"id"`
}

func (r *ConsulSingleIntentionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_consul_single_intention"
}

func (r *ConsulSingleIntentionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Consul exported service resource",

		Attributes: map[string]schema.Attribute{
			"destination_service": schema.StringAttribute{
				MarkdownDescription: "The name of the destination service",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_service": schema.StringAttribute{
				MarkdownDescription: "The name of the source service",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_peer": schema.StringAttribute{
				MarkdownDescription: "The name of the source peer",
				Optional:            true,
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

func (r *ConsulSingleIntentionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConsulSingleIntentionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConsulSingleIntentionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	singleIntentionMutex := getMutexForSingleIntention(data.DestinationService.ValueString())

	singleIntentionMutex.Lock()
	defer singleIntentionMutex.Unlock()

	serviceIntentionsConfigEntry := readServiceIntentions(r.client, data.DestinationService.ValueString())

	if data.SourcePeer.IsNull() {
		serviceIntentionsConfigEntry.Sources = append(serviceIntentionsConfigEntry.Sources, &api.SourceIntention{
			Name:       data.SourceService.ValueString(),
			Action:     api.IntentionActionAllow,
			Precedence: 9,
			Type:       api.IntentionSourceConsul,
		})
	} else {
		serviceIntentionsConfigEntry.Sources = append(serviceIntentionsConfigEntry.Sources, &api.SourceIntention{
			Name:       data.SourceService.ValueString(),
			Peer:       data.SourcePeer.ValueString(),
			Action:     api.IntentionActionAllow,
			Precedence: 9,
			Type:       api.IntentionSourceConsul,
		})
	}

	err := writeServiceIntentions(r.client, serviceIntentionsConfigEntry)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to write services intentions, got error: %s", err))
		return
	}

	if !data.SourcePeer.IsNull() {
		data.Id = types.StringValue(fmt.Sprintf("%s_%s_%s", data.DestinationService.ValueString(), data.SourceService.ValueString(), data.SourcePeer.ValueString()))
	} else {
		data.Id = types.StringValue(fmt.Sprintf("%s_%s", data.DestinationService.ValueString(), data.SourceService.ValueString()))
	}

	tflog.Trace(ctx, "exported service")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsulSingleIntentionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConsulSingleIntentionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceIntentionsConfigEntry := readServiceIntentions(r.client, data.DestinationService.ValueString())

	for _, source := range serviceIntentionsConfigEntry.Sources {
		if data.SourcePeer.IsNull() {
			if source.Name == data.SourceService.ValueString() {
				data.Id = types.StringValue(fmt.Sprintf("%s_%s", data.DestinationService.ValueString(), data.SourceService.ValueString()))
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		} else {
			if source.Name == data.SourceService.ValueString() && source.Peer == data.SourcePeer.ValueString() {
				data.Id = types.StringValue(fmt.Sprintf("%s_%s_%s", data.DestinationService.ValueString(), data.SourceService.ValueString(), data.SourcePeer.ValueString()))
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *ConsulSingleIntentionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConsulSingleIntentionResourceModel
	var oldData ConsulSingleIntentionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &oldData)...)

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	singleIntentionMutex := getMutexForSingleIntention(data.DestinationService.ValueString())

	singleIntentionMutex.Lock()
	defer singleIntentionMutex.Unlock()

	serviceIntentionsConfigEntry := readServiceIntentions(r.client, data.DestinationService.ValueString())

	sourceToRemove := -1

	for i, source := range serviceIntentionsConfigEntry.Sources {
		if oldData.SourcePeer.IsNull() {
			if source.Name == oldData.SourceService.ValueString() {
				sourceToRemove = i
				break
			}
		} else {
			if source.Name == oldData.SourceService.ValueString() && source.Peer == oldData.SourcePeer.ValueString() {
				sourceToRemove = i
				break
			}
		}
	}

	if sourceToRemove != -1 {
		serviceIntentionsConfigEntry.Sources = append(serviceIntentionsConfigEntry.Sources[:sourceToRemove], serviceIntentionsConfigEntry.Sources[sourceToRemove+1:]...)
	}

	if data.SourcePeer.IsNull() {
		serviceIntentionsConfigEntry.Sources = append(serviceIntentionsConfigEntry.Sources, &api.SourceIntention{
			Name:       data.SourceService.ValueString(),
			Action:     api.IntentionActionAllow,
			Precedence: 9,
			Type:       api.IntentionSourceConsul,
		})
	} else {
		serviceIntentionsConfigEntry.Sources = append(serviceIntentionsConfigEntry.Sources, &api.SourceIntention{
			Name:       data.SourceService.ValueString(),
			Peer:       data.SourcePeer.ValueString(),
			Action:     api.IntentionActionAllow,
			Precedence: 9,
			Type:       api.IntentionSourceConsul,
		})
	}

	err := writeServiceIntentions(r.client, serviceIntentionsConfigEntry)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to write services intentions, got error: %s", err))
		return
	}

	if !data.SourcePeer.IsNull() {
		data.Id = types.StringValue(fmt.Sprintf("%s_%s_%s", data.DestinationService.ValueString(), data.SourceService.ValueString(), data.SourcePeer.ValueString()))
	} else {
		data.Id = types.StringValue(fmt.Sprintf("%s_%s", data.DestinationService.ValueString(), data.SourceService.ValueString()))
	}

	tflog.Trace(ctx, "exported service")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsulSingleIntentionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConsulSingleIntentionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	singleIntentionMutex := getMutexForSingleIntention(data.DestinationService.ValueString())

	singleIntentionMutex.Lock()
	defer singleIntentionMutex.Unlock()

	serviceIntentionsConfigEntry := readServiceIntentions(r.client, data.DestinationService.ValueString())

	sourceToRemove := -1

	for i, source := range serviceIntentionsConfigEntry.Sources {
		if data.SourcePeer.IsNull() {
			if source.Name == data.SourceService.ValueString() {
				sourceToRemove = i
				break
			}
		} else {
			if source.Name == data.SourceService.ValueString() && source.Peer == data.SourcePeer.ValueString() {
				sourceToRemove = i
				break
			}
		}
	}

	if sourceToRemove != -1 {
		serviceIntentionsConfigEntry.Sources = append(serviceIntentionsConfigEntry.Sources[:sourceToRemove], serviceIntentionsConfigEntry.Sources[sourceToRemove+1:]...)
	}

	err := writeServiceIntentions(r.client, serviceIntentionsConfigEntry)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to write exported services, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ConsulSingleIntentionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
