// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	api "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConsulKeyResource{}
var _ resource.ResourceWithImportState = &ConsulKeyResource{}

func NewConsulKeyResource() resource.Resource {
	return &ConsulKeyResource{}
}

// ConsulKeyResource defines the resource implementation.
type ConsulKeyResource struct {
	client *api.Client
}

// ConsulKeyResourceModel describes the resource data model.
type ConsulKeyResourceModel struct {
	Path   types.String `tfsdk:"path"`
	Value  types.String `tfsdk:"value"`
	Delete types.Bool   `tfsdk:"delete"`
	Id     types.String `tfsdk:"id"`
}

func (r *ConsulKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_consul_key"
}

func (r *ConsulKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "This resource allows you to manage keys in Consul KV store.",

		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "The path to the key in the Consul KV store",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value to set for the key in the Consul KV store",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"delete": schema.BoolAttribute{
				MarkdownDescription: "Whether to delete the key from the Consul KV store",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for the exported service",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ConsulKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConsulKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConsulKeyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.KV().Put(&api.KVPair{
		Key:   data.Path.ValueString(),
		Value: []byte(data.Value.ValueString()),
	}, nil)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to write key, got error: %s", err))
		return
	}

	data.Id = types.StringValue(data.Path.ValueString())

	tflog.Debug(ctx, "exported service")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsulKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConsulKeyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	key, _, err := r.client.KV().Get(data.Path.ValueString(), nil)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read key, got error: %s", err))
		return
	}

	if key == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Value = types.StringValue(string(key.Value))
	data.Id = types.StringValue(data.Path.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsulKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConsulKeyResourceModel
	var oldData ConsulKeyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &oldData)...)

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if oldData.Delete.ValueBool() {
		_, err := r.client.KV().Delete(data.Path.ValueString(), nil)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete key, got error: %s", err))
			return
		}
	}

	_, err := r.client.KV().Put(&api.KVPair{
		Key:   data.Path.ValueString(),
		Value: []byte(data.Value.ValueString()),
	}, nil)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to write key, got error: %s", err))
		return
	}

	data.Id = types.StringValue(data.Path.ValueString())

	tflog.Debug(ctx, "exported service")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsulKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConsulKeyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Delete.ValueBool() {
		_, err := r.client.KV().Delete(data.Path.ValueString(), nil)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete key, got error: %s", err))
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *ConsulKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
