// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	api "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure UtilsProvider satisfies various provider interfaces.
var _ provider.Provider = &UtilsProvider{}
var _ provider.ProviderWithFunctions = &UtilsProvider{}

// UtilsProvider defines the provider implementation.
type UtilsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// UtilsProviderModel describes the provider data model.
type UtilsProviderModel struct {
	ConsulClusterAddress types.String `tfsdk:"consul_cluster_address"`
	ConsulClusterScheme  types.String `tfsdk:"consul_cluster_scheme"`
	ConsulToken          types.String `tfsdk:"consul_token"`
	AclAuthMethod        types.String `tfsdk:"acl_auth_method"`
}

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func loginToConsul(httpClient *http.Client, providerModel UtilsProviderModel, diagnostics *diag.Diagnostics) (*api.Client, error) {
	consulAddress := "127.0.0.1:8500"
	consulScheme := "http"
	var consulToken string

	consulHttpAddrEnv := os.Getenv("CONSUL_HTTP_ADDR")

	if providerModel.ConsulClusterAddress.IsNull() {
		if consulHttpAddrEnv != "" && strings.Contains(consulHttpAddrEnv, "://") {
			consulAddress = strings.Split(consulHttpAddrEnv, "://")[1]
		} else if consulHttpAddrEnv != "" {
			consulAddress = consulHttpAddrEnv
		}
	} else {
		consulAddress = providerModel.ConsulClusterAddress.ValueString()
	}

	if providerModel.ConsulClusterScheme.IsNull() {
		if consulHttpAddrEnv != "" && strings.Contains(consulHttpAddrEnv, "://") {
			consulScheme = strings.Split(consulHttpAddrEnv, "://")[0]
		}
	} else {
		consulScheme = providerModel.ConsulClusterScheme.ValueString()
	}

	if providerModel.ConsulToken.IsNull() {
		if os.Getenv("CONSUL_HTTP_TOKEN") != "" {
			consulToken = os.Getenv("CONSUL_HTTP_TOKEN")
		} else {
			diagnostics.AddError("Client Error", "Unable to locate initial consul token")
		}
	} else {
		consulToken = providerModel.ConsulToken.ValueString()
	}

	consulConfig := api.Config{
		Address:    consulAddress,
		Scheme:     consulScheme,
		HttpClient: httpClient,
	}

	client, err := api.NewClient(&consulConfig)

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create consul client, got error: %s", err))
		return nil, err
	}

	var aclToken string

	if IsValidUUID(consulToken) {
		aclToken = consulToken
	} else if !providerModel.AclAuthMethod.IsNull() {
		token, _, err := client.ACL().Login(&api.ACLLoginParams{
			AuthMethod:  providerModel.AclAuthMethod.ValueString(),
			BearerToken: consulToken,
		}, nil)

		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to authenticate to consul, got error: %s", err))
			return nil, err
		}

		aclToken = token.SecretID
	} else {
		diagnostics.AddError("Client Error", "Cannot authenticate using JWT token without acl auth method")
	}

	consulConfig.Token = aclToken

	client, err = api.NewClient(&consulConfig)

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create consul client, got error: %s", err))
		return nil, err
	}

	return client, nil
}

func (p *UtilsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "utils"
	resp.Version = p.version
}

func (p *UtilsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"consul_cluster_address": schema.StringAttribute{
				MarkdownDescription: "The address of the Consul cluster.",
				Optional:            true,
			},
			"consul_cluster_scheme": schema.StringAttribute{
				MarkdownDescription: "The scheme used to connect to the consul cluster. Can be http or https.",
				Optional:            true,
			},
			"consul_token": schema.StringAttribute{
				MarkdownDescription: "The token used to authenticate to the consul cluster. Can be a JWT formatted token or a UUIDv4 secret ID",
				Optional:            true,
			},
			"acl_auth_method": schema.StringAttribute{
				MarkdownDescription: "Auth method used when the token is JWT encoded. Not needed if the token is a UUIDv4 secret ID.",
				Optional:            true,
			},
		},
	}
}

func (p *UtilsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data UtilsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Example client configuration for data sources and resources
	resp.DataSourceData = func(diagnostics *diag.Diagnostics) (*api.Client, error) {
		return loginToConsul(http.DefaultClient, data, diagnostics)
	}
	resp.ResourceData = func(diagnostics *diag.Diagnostics) (*api.Client, error) {
		return loginToConsul(http.DefaultClient, data, diagnostics)
	}
}

func (p *UtilsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewConsulExportedServiceResource,
		NewConsulSingleIntentionResource,
		NewConsulKeyResource,
	}
}

func (p *UtilsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *UtilsProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UtilsProvider{
			version: version,
		}
	}
}
