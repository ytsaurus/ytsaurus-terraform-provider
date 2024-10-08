package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	Cluster                         types.String `tfsdk:"cluster"`
	Token                           types.String `tfsdk:"token"`
	UseTLS                          types.Bool   `tfsdk:"use_tls"`
	CertificateAuthorityCertificate types.String `tfsdk:"ca_certificate"`
}

const (
	HTTP  = "http://"
	HTTPS = "https://"
)

var (
	_ provider.Provider = &ytsaurusProvider{}
)

func cleanUpPrefix(cluster string) string {
	for _, p := range []string{HTTP, HTTPS} {
		if strings.HasPrefix(cluster, p) {
			return cluster[len(p):]
		}
	}
	return cluster
}

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
				Description: "Enable TLS for cluster's connection",
			},
			"ca_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "CA certificates in PEM format",
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
		Proxy:  cleanUpPrefix(config.Cluster.ValueString()),
		UseTLS: config.UseTLS.ValueBool() || strings.HasPrefix(config.Cluster.ValueString(), HTTPS),
	}

	if !config.CertificateAuthorityCertificate.IsNull() {
		clientConfig.CertificateAuthorityData = []byte(config.CertificateAuthorityCertificate.ValueString())
	}

	if !config.Token.IsNull() {
		clientConfig.Token = config.Token.ValueString()
	} else {
		if token := os.Getenv("YT_TOKEN"); token != "" {
			clientConfig.Token = token
		} else {
			clientConfig.ReadTokenFromFile = true
		}
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
