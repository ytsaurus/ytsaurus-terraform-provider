package group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/ytsaurus"
)

type groupResource struct {
	client yt.Client
}

type GroupModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func toGroupModel(g ytsaurus.Group) GroupModel {
	return GroupModel{
		ID:   types.StringValue(g.ID),
		Name: types.StringValue(g.Name),
	}
}

func toYTsaurusGroup(g GroupModel) ytsaurus.Group {
	return ytsaurus.Group{
		Name: g.Name.ValueString(),
	}
}

var (
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(yt.Client)
}

func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Groups are containers for both users and other groups and mainly are used as an ACL subjects. 

More information:
https://ytsaurus.tech/docs/en/user-guide/storage/access-control#users_groups

	Attention!
	Users and groups are located in the same namespace, which means that their names must not coincide.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ObjectID in YTsaurus cluster, can be found in object's @id attribute.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "YTsaurus group name.",
			},
		},
	}
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytGroup := toYTsaurusGroup(plan)
	createOptions := &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"name":               ytGroup.Name,
			"terraform_resource": true,
		},
	}
	id, err := r.client.CreateObject(ctx, yt.NodeGroup, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating group",
			fmt.Sprintf(
				"Could not create group %q, unexpected error: %q",
				plan.Name.ValueString(),
				err.Error(),
			),
		)
		return
	}

	plan.ID = types.StringValue(id.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group ytsaurus.Group
	if err := ytsaurus.GetObjectByID(ctx, r.client, objectID, &group); err != nil {
		resp.Diagnostics.AddError(
			"Error reading group",
			fmt.Sprintf(
				"Could not read group by id %q, unexpected error: %q",
				objectID,
				err.Error(),
			),
		)
		return
	}

	state := toGroupModel(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytGroup := toYTsaurusGroup(plan)
	p := ypath.Path(fmt.Sprintf("#%s", objectID)).Attr("name")
	if err := r.client.SetNode(ctx, p, ytGroup.Name, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error updating group 'name' attribute",
			fmt.Sprintf(
				"Could not set node %q to %q, unexpected error: %q",
				p.String(),
				ytGroup.Name,
				err.Error(),
			),
		)
		return
	}

	plan.ID = types.StringValue(objectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytGroup := toYTsaurusGroup(state)
	p := ypath.Path(fmt.Sprintf("//sys/groups/%s", ytGroup.Name))
	if err := r.client.RemoveNode(ctx, p, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group",
			fmt.Sprintf(
				"Could not delete node %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
