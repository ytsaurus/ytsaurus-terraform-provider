package mapnode

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/resource/acl"
	"terraform-provider-ytsaurus/internal/ytsaurus"
)

type mapNodeResource struct {
	client yt.Client
}

var (
	_ resource.Resource                = &mapNodeResource{}
	_ resource.ResourceWithConfigure   = &mapNodeResource{}
	_ resource.ResourceWithImportState = &mapNodeResource{}
)

type MapNodeModel struct {
	ID         types.String `tfsdk:"id"`
	Path       types.String `tfsdk:"path"`
	Account    types.String `tfsdk:"account"`
	InheritACL types.Bool   `tfsdk:"inherit_acl"`
	ACL        acl.ACLModel `tfsdk:"acl"`
}

func toMapNodeModel(m ytsaurus.MapNode) MapNodeModel {
	return MapNodeModel{
		ID:         types.StringValue(m.ID),
		Path:       types.StringValue(m.Path),
		Account:    types.StringValue(m.Account),
		InheritACL: types.BoolValue(m.InheritACL),
		ACL:        acl.ToACLModel(m.ACL),
	}
}

func toYTsaurusMapNode(m MapNodeModel) ytsaurus.MapNode {
	return ytsaurus.MapNode{
		Path:       m.Path.ValueString(),
		Account:    m.Account.ValueString(),
		InheritACL: m.InheritACL.ValueBool(),
		ACL:        acl.ToYTsaurusACL(m.ACL),
	}
}

func NewGroupResource() resource.Resource {
	return &mapNodeResource{}
}

func (r *mapNodeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_map_node"
}

func (r *mapNodeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ObjectID in YTsaurus cluster, can be found in object's @id attribute.",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Node absolute path.",
			},
			"account": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Account used to keep track of the resources being used by a specific node.",
			},
			"inherit_acl": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Enable or disable ACL inheritance from object's parents.",
			},
			"acl": schema.ListNestedAttribute{
				Optional:     true,
				NestedObject: acl.ACLSchema,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Description: "A list of ACE records. More information: https://ytsaurus.tech/docs/en/user-guide/storage/access-control.",
			},
		},
	}
}

func (r *mapNodeResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(yt.Client)
}

func (r *mapNodeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MapNodeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytMapNode := toYTsaurusMapNode(plan)
	createOptions := &yt.CreateNodeOptions{
		Attributes: map[string]interface{}{
			"acl":                ytMapNode.ACL,
			"inherit_acl":        ytMapNode.InheritACL,
			"terraform_resource": true,
		},
	}
	if ytMapNode.Account != "" {
		createOptions.Attributes["account"] = ytMapNode.Account
	}

	p := ypath.Path(ytMapNode.Path)
	id, err := r.client.CreateNode(ctx, p, yt.NodeMap, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating map_node",
			fmt.Sprintf(
				"Could not create map_node %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}
	ytMapNode.ID = id.String()

	if ytMapNode.Account == "" {
		if err := r.client.GetNode(ctx, p.Attr("account"), &ytMapNode.Account, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error creating map_node",
				fmt.Sprintf(
					"Could not read 'account' attribute, unexpected error: %q",
					err.Error(),
				),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, toMapNodeModel(ytMapNode))...)
}

func (r *mapNodeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var mapNode ytsaurus.MapNode
	if err := ytsaurus.GetObjectByID(ctx, r.client, objectID, &mapNode); err != nil {
		resp.Diagnostics.AddError(
			"Error reading map_node",
			fmt.Sprintf(
				"Could not read map_node with id %q, unexpected error: %q",
				objectID,
				err.Error(),
			),
		)
		return
	}

	p := ypath.Path(fmt.Sprintf("#%s", objectID)).Attr("path")
	if err := r.client.GetNode(ctx, p, &mapNode.Path, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error reading map_node @path attribute",
			fmt.Sprintf(
				"Could not read map_node %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}

	state := toMapNodeModel(mapNode)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mapNodeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MapNodeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state MapNodeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Path.Equal(state.Path) {
		resp.Diagnostics.AddError(
			"Error updating map_node attributes",
			"Builtin attribute 'path' cannot be updated",
		)
		return
	}

	ytMapNode := toYTsaurusMapNode(plan)
	p := ypath.Path(fmt.Sprintf("#%s", state.ID.ValueString()))
	attributeUpdates := map[string]interface{}{
		"account":     ytMapNode.Account,
		"acl":         ytMapNode.ACL,
		"inherit_acl": ytMapNode.InheritACL,
	}
	for k, v := range attributeUpdates {
		if err := r.client.SetNode(ctx, p.Attr(k), v, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error updating map_node attributes",
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

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *mapNodeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MapNodeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := ypath.Path(state.Path.ValueString())
	if err := r.client.RemoveNode(ctx, p, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting map_node",
			fmt.Sprintf(
				"Could not delete map_node %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}
}

func (r *mapNodeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
