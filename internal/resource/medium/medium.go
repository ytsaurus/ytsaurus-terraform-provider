package medium

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/resource/acl"
	"terraform-provider-ytsaurus/internal/ytsaurus"
)

const (
	defaultMaxErasureReplicasPerRack       = 2147483647
	defaultMaxJournalReplicasPerRack       = 2147483647
	defaultMaxRegularReplicasPerRack       = 2147483647
	defaultMaxReplicasPerRack              = 2147483647
	defaultMaxReplicationFactor            = 20
	defaultPreferLocalHostForDynamicTables = true
)

type mediumResource struct {
	client yt.Client
}

var (
	_ resource.Resource                = &mediumResource{}
	_ resource.ResourceWithConfigure   = &mediumResource{}
	_ resource.ResourceWithImportState = &mediumResource{}
)

type mediumConfigModel struct {
	MaxErasureReplicasPerRack       types.Int64 `tfsdk:"max_erasure_replicas_per_rack"`
	MaxJournalReplicasPerRack       types.Int64 `tfsdk:"max_journal_replicas_per_rack"`
	MaxRegularReplicasPerRack       types.Int64 `tfsdk:"max_regular_replicas_per_rack"`
	MaxReplicasPerRack              types.Int64 `tfsdk:"max_replicas_per_rack"`
	MaxReplicationFactor            types.Int64 `tfsdk:"max_replication_factor"`
	PreferLocalHostForDynamicTables types.Bool  `tfsdk:"prefer_local_host_for_dynamic_tables"`
}

func toYTsaurusMediumConfig(c mediumConfigModel) ytsaurus.MediumConfig {
	return ytsaurus.MediumConfig{
		MaxErasureReplicasPerRack:       c.MaxErasureReplicasPerRack.ValueInt64(),
		MaxJournalReplicasPerRack:       c.MaxJournalReplicasPerRack.ValueInt64(),
		MaxRegularReplicasPerRack:       c.MaxRegularReplicasPerRack.ValueInt64(),
		MaxReplicasPerRack:              c.MaxReplicasPerRack.ValueInt64(),
		MaxReplicationFactor:            c.MaxReplicationFactor.ValueInt64(),
		PreferLocalHostForDynamicTables: c.PreferLocalHostForDynamicTables.ValueBool(),
	}
}

func toMediumConfigModel(c ytsaurus.MediumConfig) *mediumConfigModel {
	return &mediumConfigModel{
		MaxErasureReplicasPerRack:       types.Int64Value(c.MaxErasureReplicasPerRack),
		MaxJournalReplicasPerRack:       types.Int64Value(c.MaxJournalReplicasPerRack),
		MaxRegularReplicasPerRack:       types.Int64Value(c.MaxRegularReplicasPerRack),
		MaxReplicasPerRack:              types.Int64Value(c.MaxReplicasPerRack),
		MaxReplicationFactor:            types.Int64Value(c.MaxReplicationFactor),
		PreferLocalHostForDynamicTables: types.BoolValue(c.PreferLocalHostForDynamicTables),
	}
}

type mediumModel struct {
	ID                  types.String       `tfsdk:"id"`
	Name                types.String       `tfsdk:"name"`
	ACL                 acl.ACLModel       `tfsdk:"acl"`
	DiskFamilyWhitelist types.List         `tfsdk:"disk_family_whitelist"`
	Config              *mediumConfigModel `tfsdk:"config"`
}

func toYTsaurusMedium(ctx context.Context, m mediumModel) (ytsaurus.Medium, diag.Diagnostics) {
	var diskFamilyWhitelist []string
	diags := m.DiskFamilyWhitelist.ElementsAs(ctx, &diskFamilyWhitelist, false)
	acl, valDiags := acl.ToYTsaurusACL(m.ACL)
	diags.Append(valDiags...)

	ytMedium := ytsaurus.Medium{
		Name:                m.Name.ValueString(),
		DiskFamilyWhitelist: &diskFamilyWhitelist,
		ACL:                 acl,
	}

	if m.Config != nil {
		ytMedium.Config = toYTsaurusMediumConfig(*m.Config)
	} else {
		ytMedium.Config.MaxErasureReplicasPerRack = defaultMaxErasureReplicasPerRack
		ytMedium.Config.MaxJournalReplicasPerRack = defaultMaxJournalReplicasPerRack
		ytMedium.Config.MaxRegularReplicasPerRack = defaultMaxRegularReplicasPerRack
		ytMedium.Config.MaxReplicasPerRack = defaultMaxReplicasPerRack
		ytMedium.Config.MaxReplicationFactor = defaultMaxReplicationFactor
		ytMedium.Config.PreferLocalHostForDynamicTables = defaultPreferLocalHostForDynamicTables
	}

	return ytMedium, diags
}

func toMediumModel(m ytsaurus.Medium) mediumModel {

	medium := mediumModel{
		ID:     types.StringValue(m.ID),
		Name:   types.StringValue(m.Name),
		ACL:    acl.ToACLModel(m.ACL),
		Config: toMediumConfigModel(m.Config),
	}

	if m.DiskFamilyWhitelist != nil {
		var diskFamilyWhitelist []attr.Value
		for _, d := range *m.DiskFamilyWhitelist {
			diskFamilyWhitelist = append(diskFamilyWhitelist, types.StringValue(d))
		}
		medium.DiskFamilyWhitelist = types.ListValueMust(types.StringType, diskFamilyWhitelist)
	} else {
		medium.DiskFamilyWhitelist = types.ListNull(types.StringType)
	}

	return medium
}

func NewMediumResource() resource.Resource {
	return &mediumResource{}
}

func (r *mediumResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_medium"
}

func (r *mediumResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(yt.Client)
}

func (r *mediumResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Different media types (HDD, SDD, RAM) are logically combined into special entities referred to as media.

More information:
https://ytsaurus.tech/docs/en/user-guide/storage/media`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ObjectID in the YTsaurus cluster, can be found in an object's @id attribute.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "YTsaurus medium name.",
			},
			"acl": schema.ListNestedAttribute{
				Optional:     true,
				NestedObject: acl.ACLSchema,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Description: "A list of ACE records. More information: https://ytsaurus.tech/docs/en/user-guide/storage/access-control.",
			},
			"disk_family_whitelist": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of disk_families allowed for the medium.",
			},
			"config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "The medium options.",
				Attributes: map[string]schema.Attribute{
					"max_erasure_replicas_per_rack": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultMaxErasureReplicasPerRack),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Maximum number of erasure chunk replicas to store in a rack.",
					},
					"max_journal_replicas_per_rack": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultMaxJournalReplicasPerRack),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Maximum number of journal chunk replicas to store in a rack.",
					},
					"max_regular_replicas_per_rack": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultMaxRegularReplicasPerRack),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Maximum number of regular chunk replicas to store in a rack.",
					},
					"max_replicas_per_rack": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultMaxReplicasPerRack),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Maximum number of replicas to store in a rack.",
					},
					"max_replication_factor": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultMaxReplicationFactor),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Maximum replication factor for the medium.",
					},
					"prefer_local_host_for_dynamic_tables": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(defaultPreferLocalHostForDynamicTables),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
						Description: "Prefer to store dynamic table data on local disks.",
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *mediumResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mediumModel
	req.Plan.Get(ctx, &plan)
	if resp.Diagnostics.HasError() {
		return
	}

	ytMedium, diags := toYTsaurusMedium(ctx, plan)
	resp.Diagnostics.Append(diags...)

	createOptions := &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"name":               ytMedium.Name,
			"acl":                ytMedium.ACL,
			"config":             ytMedium.Config,
			"terraform_resource": true,
		},
	}
	if !plan.DiskFamilyWhitelist.IsNull() {
		createOptions.Attributes["disk_family_whitelist"] = ytMedium.DiskFamilyWhitelist
	}

	id, err := r.client.CreateObject(ctx, yt.NodeDomesticMedium, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating medium",
			fmt.Sprintf(
				"Could not create medium %q, unexpected error: %q",
				plan.Name.ValueString(),
				err.Error(),
			),
		)
		return
	}

	plan.ID = types.StringValue(id.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *mediumResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var medium ytsaurus.Medium
	if err := ytsaurus.GetObjectByID(ctx, r.client, objectID, &medium); err != nil {
		resp.Diagnostics.AddError(
			"Error reading medium",
			fmt.Sprintf(
				"Could not read medium by id %q, unexpected error: %q",
				objectID,
				err.Error(),
			),
		)
		return
	}

	state := toMediumModel(medium)

	// Reset state.Config to nil if it wasn't configured in .tf file
	var currentState mediumModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if currentState.Config == nil {
		state.Config = nil
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mediumResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state mediumModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var plan mediumModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytMedium, diags := toYTsaurusMedium(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := ypath.Path(fmt.Sprintf("#%s", state.ID.ValueString()))
	attributeUpdates := map[string]interface{}{
		"name":   ytMedium.Name,
		"acl":    ytMedium.ACL,
		"config": ytMedium.Config,
	}
	if !plan.DiskFamilyWhitelist.IsNull() {
		attributeUpdates["disk_family_whitelist"] = ytMedium.DiskFamilyWhitelist
	}

	for k, v := range attributeUpdates {
		if err := r.client.SetNode(ctx, p.Attr(k), v, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error updating medium attributes",
				fmt.Sprintf(
					"Could not set node %q to '%v', unexpected error: %q",
					p.Attr(k).String(),
					v,
					err.Error(),
				),
			)
			return
		}
	}

	if plan.DiskFamilyWhitelist.IsNull() {
		attrPath := p.Attr("disk_family_whitelist")
		if err := ytsaurus.RemoveIfExists(ctx, r.client, attrPath); err != nil {
			resp.Diagnostics.AddError(
				"Error updating medium attributes",
				fmt.Sprintf(
					"Could not remove attribute %q, unexpected error: %q",
					attrPath.String(),
					err.Error(),
				),
			)
			return
		}
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *mediumResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Error deleting medium",
		"Could not delete medium, media can't be deleted after creation, only renamed",
	)
}

func (r *mediumResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
