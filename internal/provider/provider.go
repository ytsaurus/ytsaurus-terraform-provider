package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"go.ytsaurus.tech/yt/go/yt"
	"go.ytsaurus.tech/yt/go/yt/ythttp"

	"terraform-provider-ytsaurus/internal/resource/account"
	"terraform-provider-ytsaurus/internal/resource/group"
	"terraform-provider-ytsaurus/internal/resource/mapnode"
	"terraform-provider-ytsaurus/internal/resource/medium"
	"terraform-provider-ytsaurus/internal/resource/schedulerpool"
	"terraform-provider-ytsaurus/internal/resource/tabletcellbundle"
	"terraform-provider-ytsaurus/internal/resource/user"
)

type ytsaurusProvider struct{}

type ytsaurusProviderModel struct {
	Cluster   types.String `tfsdk:"cluster"`
	Token     types.String `tfsdk:"token"`
	UseTLS    types.Bool   `tfsdk:"use_tls"`
	ClusterCA types.String `tfsdk:"cluster_ca"`
}

var (
	_ provider.Provider = &ytsaurusProvider{}
)

func New() provider.Provider {
	return &ytsaurusProvider{}
}

func (p *ytsaurusProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ytsaurus"
}

func (p *ytsaurusProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster": schema.StringAttribute{
				Required:    true,
				Description: "The YTsaurus cluster FQDN.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Description: "Admin's token. Use YT_TOKEN_PATH environment variable instead.",
			},
			"use_tls": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to use TLS connection.",
			},
			"cluster_ca": schema.StringAttribute{
				Optional:    true,
				Description: "PEM-encoded root certificates bundle for TLS.",
			},
		},
	}
}

func (p *ytsaurusProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ytsaurusProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientConfig := &yt.Config{
		Proxy: config.Cluster.ValueString(),
	}
	if !config.Token.IsNull() {
		clientConfig.Token = config.Token.ValueString()
	} else {
		clientConfig.ReadTokenFromFile = true
	}
	if config.UseTLS.ValueBool() {
		clientConfig.UseTLS = true
	}
	if !config.ClusterCA.IsNull() {
		clientConfig.Certs = []byte(config.ClusterCA.ValueString())
	}

	client, err := ythttp.NewClient(clientConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create YTsaurus Client",
			fmt.Sprintf(
				"An unexpected error occurred when creating the YTsaurus client. Error: %q",
				err.Error(),
			),
		)
	}

	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *ytsaurusProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *ytsaurusProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		group.NewGroupResource,
		user.NewUserResource,
		account.NewAccountResource,
		medium.NewMediumResource,
		mapnode.NewGroupResource,
		tabletcellbundle.NewTabletCellBundleResource,
		schedulerpool.NewSchedulerPoolResource,
	}
}
