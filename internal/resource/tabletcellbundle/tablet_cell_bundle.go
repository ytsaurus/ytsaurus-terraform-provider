package tabletcellbundle

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/resource/acl"
	"terraform-provider-ytsaurus/internal/ytsaurus"
)

const (
	defaultChangelogWriteQuorum            = 2
	defaultChangelogReadQuorum             = 2
	defaultChangelogReplicationFactor      = 3
	defaultSnapshotReplicationFactor       = 3
	defaultMaxReplicationFactor            = 20
	defaultPreferLocalHostForDynamicTables = true

	// FIXME!
	nodeTabletCell yt.NodeType = "tablet_cell"
)

type tabletCellBundleResource struct {
	client yt.Client
}

type TabletCellBundleOptionsModel struct {
	ChangelogAccount           types.String `tfsdk:"changelog_account"`
	ChangelogWriteQuorum       types.Int64  `tfsdk:"changelog_write_quorum"`
	ChangelogReadQuorum        types.Int64  `tfsdk:"changelog_read_quorum"`
	ChangelogReplicationFactor types.Int64  `tfsdk:"changelog_replication_factor"`
	ChangelogPrimaryMedium     types.String `tfsdk:"changelog_primary_medium"`
	SnapshotAccount            types.String `tfsdk:"snapshot_account"`
	SnapshotReplicationFactor  types.Int64  `tfsdk:"snapshot_replication_factor"`
	SnapshotPrimaryMedium      types.String `tfsdk:"snapshot_primary_medium"`
}

func toTabletCellBundleOptionsModel(o *ytsaurus.TabletCellBundleOptions) *TabletCellBundleOptionsModel {
	if o != nil {
		return &TabletCellBundleOptionsModel{
			ChangelogAccount:           types.StringValue(o.ChangelogAccount),
			ChangelogWriteQuorum:       types.Int64Value(o.ChangelogWriteQuorum),
			ChangelogReadQuorum:        types.Int64Value(o.ChangelogReadQuorum),
			ChangelogReplicationFactor: types.Int64Value(o.ChangelogReplicationFactor),
			ChangelogPrimaryMedium:     types.StringValue(o.ChangelogPrimaryMedium),
			SnapshotAccount:            types.StringValue(o.SnapshotAccount),
			SnapshotReplicationFactor:  types.Int64Value(o.SnapshotReplicationFactor),
			SnapshotPrimaryMedium:      types.StringValue(o.SnapshotPrimaryMedium),
		}
	} else {
		return nil
	}
}

type TabletCellBundleModel struct {
	ID              types.String                  `tfsdk:"id"`
	Name            types.String                  `tfsdk:"name"`
	NodeTagFilter   types.String                  `tfsdk:"node_tag_filter"`
	TabletCellCount types.Int64                   `tfsdk:"tablet_cell_count"`
	ACL             acl.ACLModel                  `tfsdk:"acl"`
	Options         *TabletCellBundleOptionsModel `tfsdk:"options"`
}

func toTabletCellBundleModel(b ytsaurus.TabletCellBundle) TabletCellBundleModel {
	bundle := TabletCellBundleModel{
		ID:              types.StringValue(b.ID),
		Name:            types.StringValue(b.Name),
		TabletCellCount: types.Int64Value(b.TabletCellCount),
		ACL:             acl.ToACLModel(b.ACL),
		Options:         toTabletCellBundleOptionsModel(b.Options),
	}

	if len(b.NodeTagFilter) > 0 {
		bundle.NodeTagFilter = types.StringValue(b.NodeTagFilter)
	} else {
		bundle.NodeTagFilter = types.StringNull()
	}

	return bundle
}

func toYTsaurusTabletCellBundleOptions(o *TabletCellBundleOptionsModel) *ytsaurus.TabletCellBundleOptions {
	if o != nil {
		return &ytsaurus.TabletCellBundleOptions{
			ChangelogAccount:           o.ChangelogAccount.ValueString(),
			ChangelogWriteQuorum:       o.ChangelogWriteQuorum.ValueInt64(),
			ChangelogReadQuorum:        o.ChangelogReadQuorum.ValueInt64(),
			ChangelogReplicationFactor: o.ChangelogReplicationFactor.ValueInt64(),
			ChangelogPrimaryMedium:     o.ChangelogPrimaryMedium.ValueString(),
			SnapshotAccount:            o.SnapshotAccount.ValueString(),
			SnapshotReplicationFactor:  o.SnapshotReplicationFactor.ValueInt64(),
			SnapshotPrimaryMedium:      o.SnapshotPrimaryMedium.ValueString(),
		}
	} else {
		return nil
	}
}

func toYTsaurusTabletCellBundle(b TabletCellBundleModel) ytsaurus.TabletCellBundle {
	return ytsaurus.TabletCellBundle{
		ID:              b.ID.ValueString(),
		Name:            b.Name.ValueString(),
		NodeTagFilter:   b.NodeTagFilter.ValueString(),
		TabletCellCount: b.TabletCellCount.ValueInt64(),
		ACL:             acl.ToYTsaurusACL(b.ACL),
		Options:         toYTsaurusTabletCellBundleOptions(b.Options),
	}
}

var (
	_ resource.Resource                = &tabletCellBundleResource{}
	_ resource.ResourceWithConfigure   = &tabletCellBundleResource{}
	_ resource.ResourceWithImportState = &tabletCellBundleResource{}
)

func NewTabletCellBundleResource() resource.Resource {
	return &tabletCellBundleResource{}
}

func (r *tabletCellBundleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tablet_cell_bundle"
}

func (r *tabletCellBundleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Tablet cell bundles are needed to combine cells on the basis of common settings.

More information:
https://ytsaurus.tech/docs/en/user-guide/dynamic-tables/concepts`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ObjectID in the YTsaurus cluster, can be found in an object's @id attribute.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "YTsaurus tablet_cell_bundle name.",
			},
			"node_tag_filter": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "An attribute to select cluster nodes for tablet cells for this bundle.",
			},
			"tablet_cell_count": schema.Int64Attribute{
				Required:    true,
				Description: "Number of cells in a bundle.",
			},
			"acl": schema.ListNestedAttribute{
				Optional:     true,
				NestedObject: acl.ACLSchema,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Description: "A list of ACE records. More information: https://ytsaurus.tech/docs/en/user-guide/storage/access-control.",
			},
			"options": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Tablet_cell_bundle configuration options.",
				Attributes: map[string]schema.Attribute{
					"changelog_account": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						Description: "An account to store tablet_cell_bundle's changelogs.",
					},
					"changelog_write_quorum": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultChangelogWriteQuorum),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Minimum number of changelog's replicas to consider the changelog successfully written.",
					},
					"changelog_read_quorum": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultChangelogReadQuorum),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Minimum available replica count such that the changelog can be read.",
					},
					"changelog_replication_factor": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultChangelogReplicationFactor),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "How many replicas should be stored for the changelog.",
					},
					"changelog_primary_medium": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						Description: "A medium to store the changelog.",
					},
					"snapshot_account": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						Description: "An account to store tablet_cell_bundle's snapshots.",
					},
					"snapshot_replication_factor": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultSnapshotReplicationFactor),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "How many replicas should be stored for the snapshot.",
					},
					"snapshot_primary_medium": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						Description: "A medium to store the snapshot.",
					},
				},
			},
		},
	}
}

func (r *tabletCellBundleResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(yt.Client)
}

func (r *tabletCellBundleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TabletCellBundleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytTabletCellBundle := toYTsaurusTabletCellBundle(plan)
	createOptions := &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"name":               ytTabletCellBundle.Name,
			"terraform_resource": true,
			"acl":                ytTabletCellBundle.ACL,
			"options":            ytTabletCellBundle.Options,
		},
	}
	if len(ytTabletCellBundle.NodeTagFilter) > 0 {
		createOptions.Attributes["node_tag_filter"] = ytTabletCellBundle.NodeTagFilter
	}

	id, err := r.client.CreateObject(ctx, yt.NodeTabletCellBundle, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating tablet_cell_bundle",
			fmt.Sprintf(
				"Could not create tablet_cell_bundle %q, unexpected error: %q",
				ytTabletCellBundle.Name,
				err.Error(),
			),
		)
		return
	}

	createOptions = &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"tablet_cell_bundle": ytTabletCellBundle.Name,
		},
	}
	for i := 0; i < int(ytTabletCellBundle.TabletCellCount); i++ {
		_, err := r.client.CreateObject(ctx, nodeTabletCell, createOptions)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating tablet_cell_bundle",
				fmt.Sprintf(
					"Could not create tablet_cell, unexpected error: %q",
					err.Error(),
				),
			)
			return
		}
	}

	ytTabletCellBundle.ID = id.String()
	resp.Diagnostics.Append(resp.State.Set(ctx, toTabletCellBundleModel(ytTabletCellBundle))...)
}

func (r *tabletCellBundleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ytTabletCellBundle ytsaurus.TabletCellBundle
	if err := ytsaurus.GetObjectByID(ctx, r.client, objectID, &ytTabletCellBundle); err != nil {
		resp.Diagnostics.AddError(
			"Error reading tablet_cell_bundle",
			fmt.Sprintf(
				"Could not read tablet_cell_bundle by id %q, unexpected error: %q",
				objectID,
				err.Error(),
			),
		)
		return
	}

	// FIXME after YT-19060
	var ytTabletCellBundleAreas ytsaurus.TabletCellBundleAreas
	p := ypath.Path(fmt.Sprintf("#%s/@areas", objectID))
	if err := r.client.GetNode(ctx, p, &ytTabletCellBundleAreas, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error reading tablet_cell_bundle",
			fmt.Sprintf(
				"Could not read %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}

	area, ok := ytTabletCellBundleAreas["default"]
	if !ok {
		ytTabletCellBundle.NodeTagFilter = ""
	} else if len(area.NodeTagFilter) == 0 {
		ytTabletCellBundle.NodeTagFilter = ""
	} else {
		ytTabletCellBundle.NodeTagFilter = area.NodeTagFilter
	}

	state := toTabletCellBundleModel(ytTabletCellBundle)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *tabletCellBundleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TabletCellBundleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state TabletCellBundleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytTabletCellBundlePlan := toYTsaurusTabletCellBundle(plan)
	ytTabletCellBundleState := toYTsaurusTabletCellBundle(state)

	attributeUpdates := map[string]interface{}{
		"name":            ytTabletCellBundlePlan.Name,
		"options":         ytTabletCellBundlePlan.Options,
		"node_tag_filter": ytTabletCellBundlePlan.NodeTagFilter,
		"acl":             ytTabletCellBundlePlan.ACL,
	}

	p := ypath.Path(fmt.Sprintf("#%s", ytTabletCellBundleState.ID))
	for k, v := range attributeUpdates {
		if err := r.client.SetNode(ctx, p.Attr(k), v, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error updating tablet_cell_bundle",
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

	if err := r.updateTabletCellCount(ctx, ytTabletCellBundleState.Name, ytTabletCellBundleState.TabletCellCount, ytTabletCellBundlePlan.TabletCellCount); err != nil {
		resp.Diagnostics.AddError(
			"Error updating tablet_cell_bundle",
			fmt.Sprintf(
				"Could update tablet_cell_count attribute, unexpected error: %q",
				err.Error(),
			),
		)
		return
	}

	ytTabletCellBundlePlan.ID = ytTabletCellBundleState.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, toTabletCellBundleModel(ytTabletCellBundlePlan))...)
}

func (r *tabletCellBundleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TabletCellBundleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytTabletCellBundleState := toYTsaurusTabletCellBundle(state)

	if err := r.updateTabletCellCount(ctx, ytTabletCellBundleState.Name, ytTabletCellBundleState.TabletCellCount, 0); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting tablet_cell_bundle",
			fmt.Sprintf(
				"Could delete tablet cells, unexpected error: %q",
				err.Error(),
			),
		)
		return
	}

	var ytTabletCellBundleAreas ytsaurus.TabletCellBundleAreas
	p := ypath.Path(fmt.Sprintf("#%s/@areas", ytTabletCellBundleState.ID))
	if err := r.client.GetNode(ctx, p, &ytTabletCellBundleAreas, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting tablet_cell_bundle",
			fmt.Sprintf(
				"Could not read %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}

	for _, area := range ytTabletCellBundleAreas {
		p := ypath.Path(fmt.Sprintf("#%s", area.ID))
		if err := r.client.RemoveNode(ctx, p, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error deleting tablet_cell_bundle",
				fmt.Sprintf(
					"Could delete area %q, unexpected error: %q",
					area.ID,
					err.Error(),
				),
			)
			return
		}
	}

	p = ypath.Path(fmt.Sprintf("#%s", ytTabletCellBundleState.ID))
	if err := r.client.RemoveNode(ctx, p, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting tablet_cell_bundle",
			fmt.Sprintf(
				"Could delete tablet_cell_bundle %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}

	for {
		ok, err := r.client.NodeExists(ctx, p, nil)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting tablet_cell_bundle",
				fmt.Sprintf(
					"Could delete tablet_cell_bundle %q, unexpected error: %q",
					p.String(),
					err.Error(),
				),
			)
			return
		}
		if ok {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
}

func (r *tabletCellBundleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *tabletCellBundleResource) updateTabletCellCount(ctx context.Context, bundleName string, current int64, expected int64) error {

	cypressPathToBundle := ypath.Path(fmt.Sprintf("//sys/tablet_cell_bundles/%s", bundleName))

	if expected > current {
		createOptions := &yt.CreateObjectOptions{
			Attributes: map[string]interface{}{
				"tablet_cell_bundle": bundleName,
			},
		}
		for i := 0; i < int(expected-current); i++ {
			_, err := r.client.CreateObject(ctx, nodeTabletCell, createOptions)
			if err != nil {
				return err
			}
		}

	} else {
		var tabletCellIDs []string
		if err := r.client.GetNode(ctx, cypressPathToBundle.Attr("tablet_cell_ids"), &tabletCellIDs, nil); err != nil {
			return err
		}

		for i := 0; i < int(current-expected); i++ {
			tabletCell := tabletCellIDs[i]
			p := ypath.Path(fmt.Sprintf("#%s", tabletCell))
			if err := r.client.RemoveNode(ctx, p, nil); err != nil {
				return err
			}
		}
	}

	for {
		var tabletCellCount int64
		if err := r.client.GetNode(ctx, cypressPathToBundle.Attr("tablet_cell_count"), &tabletCellCount, nil); err != nil {
			return err
		}

		if tabletCellCount == expected {
			break
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}
