package group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/set"
	"terraform-provider-ytsaurus/internal/ytsaurus"
)

type groupResource struct {
	client yt.Client
}

type GroupModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	MemberOf types.Set    `tfsdk:"member_of"`
}

func toYTsaurusGroup(ctx context.Context, g GroupModel) (ytsaurus.Group, diag.Diagnostics) {
	var memberOf []string
	diags := g.MemberOf.ElementsAs(ctx, &memberOf, false)

	ytGroup := ytsaurus.Group{
		Name:     g.Name.ValueString(),
		MemberOf: &memberOf,
	}

	return ytGroup, diags
}

func toGroupModel(g ytsaurus.Group) GroupModel {
	group := GroupModel{
		ID:   types.StringValue(g.ID),
		Name: types.StringValue(g.Name),
	}

	if g.MemberOf != nil && len(*g.MemberOf) > 0 {
		var memberOf []attr.Value
		for _, m := range *g.MemberOf {
			memberOf = append(memberOf, types.StringValue(m))
		}
		group.MemberOf = types.SetValueMust(types.StringType, memberOf)
	} else {
		group.MemberOf = types.SetNull(types.StringType)
	}
	return group
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
Groups are containers for both users and other groups and are mainly used as ACL subjects. 

More information:
https://ytsaurus.tech/docs/en/user-guide/storage/access-control#users_groups

	Attention!
	Users and groups are located in the same namespace, which means that their names must not coincide.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ObjectID in the YTsaurus cluster, can be found in an object's @id attribute.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "YTsaurus group name.",
			},
			"member_of": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				Description: "A set of groups that this object belongs to.",
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

	ytGroup, diags := toYTsaurusGroup(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	for _, groupName := range *ytGroup.MemberOf {
		if err := r.client.AddMember(ctx, groupName, ytGroup.Name, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error adding user to group",
				fmt.Sprintf(
					"Could not add user %q to the group %q, unexpected error: %q",
					ytGroup.Name,
					groupName,
					err.Error(),
				),
			)
		}
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

	var ytGroup ytsaurus.Group
	exists, err := ytsaurus.ObjectExistsByID(ctx, r.client, objectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading group",
			fmt.Sprintf(
				"Could not check group by id %q, unexpected error: %q",
				objectID,
				err.Error(),
			),
		)
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := ytsaurus.GetObjectByID(ctx, r.client, objectID, &ytGroup); err != nil {
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

	if ytGroup.MemberOf != nil {
		var memberOfWithoutBuiltin []string
		for _, u := range *ytGroup.MemberOf {
			switch u {
			case "users":
				continue
			default:
				memberOfWithoutBuiltin = append(memberOfWithoutBuiltin, u)
			}
		}
		ytGroup.MemberOf = &memberOfWithoutBuiltin
	}

	state := toGroupModel(ytGroup)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytUserPlan, diags := toYTsaurusGroup(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytUserState, diags := toYTsaurusGroup(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytGroup, diags := toYTsaurusGroup(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	stateMemberOfSet := set.ToStringSet(*ytUserState.MemberOf)
	planMemberOfSet := set.ToStringSet(*ytUserPlan.MemberOf)

	removeMemberFromGroups := stateMemberOfSet.Difference(planMemberOfSet)
	for _, groupName := range removeMemberFromGroups {
		if err := r.client.RemoveMember(ctx, groupName, plan.Name.ValueString(), nil); err != nil {
			resp.Diagnostics.AddError(
				"Error removing user from group",
				fmt.Sprintf(
					"Could not remove %q user from %q group, unexpected error: %q",
					plan.Name.ValueString(),
					groupName,
					err.Error(),
				),
			)
			return
		}
	}

	addMemberToGroups := planMemberOfSet.Difference(stateMemberOfSet)
	for _, groupName := range addMemberToGroups {
		if err := r.client.AddMember(ctx, groupName, plan.Name.ValueString(), nil); err != nil {
			resp.Diagnostics.AddError(
				"Error removing user from group",
				fmt.Sprintf(
					"Could not add %q user to %q group, unexpected error: %q",
					plan.Name.ValueString(),
					groupName,
					err.Error(),
				),
			)
			return
		}
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

	ytGroup, diags := toYTsaurusGroup(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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
