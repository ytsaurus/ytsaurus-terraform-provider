package user

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/set"
	"terraform-provider-ytsaurus/internal/ytsaurus"
)

type userResource struct {
	client yt.Client
}

type UserModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	MemberOf types.Set    `tfsdk:"member_of"`
}

func toYTsaurusUser(ctx context.Context, u UserModel) (ytsaurus.User, diag.Diagnostics) {
	var memberOf []string
	diags := u.MemberOf.ElementsAs(ctx, &memberOf, false)
	ytMedium := ytsaurus.User{
		Name:     u.Name.ValueString(),
		MemberOf: &memberOf,
	}
	return ytMedium, diags
}

func toUserModel(u ytsaurus.User) UserModel {
	user := UserModel{
		ID:   types.StringValue(u.ID),
		Name: types.StringValue(u.Name),
	}

	if u.MemberOf != nil && len(*u.MemberOf) > 0 {
		var memberOf []attr.Value
		for _, m := range *u.MemberOf {
			memberOf = append(memberOf, types.StringValue(m))
		}
		user.MemberOf = types.SetValueMust(types.StringType, memberOf)
	} else {
		user.MemberOf = types.SetNull(types.StringType)
	}
	return user
}

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

func NewUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Users are used for requests authentication and authorization.

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
				Description: "YTsaurus user name.",
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
				Description: "A set of user's groups.",
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(yt.Client)
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytUser, diags := toYTsaurusUser(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createOptions := &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"name":               ytUser.Name,
			"terraform_resource": true,
		},
	}
	id, err := r.client.CreateObject(ctx, yt.NodeUser, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user",
			fmt.Sprintf(
				"Could not create user %q, unexpected error: %q",
				plan.Name.ValueString(),
				err.Error(),
			),
		)
		return
	}

	for _, groupName := range *ytUser.MemberOf {
		if err := r.client.AddMember(ctx, groupName, ytUser.Name, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error adding user to group",
				fmt.Sprintf(
					"Could not add user %q to the group %q, unexpected error: %q",
					ytUser.Name,
					groupName,
					err.Error(),
				),
			)
		}
	}

	plan.ID = types.StringValue(id.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ytUser ytsaurus.User
	if err := ytsaurus.GetObjectByID(ctx, r.client, state.ID.ValueString(), &ytUser); err != nil {
		resp.Diagnostics.AddError(
			"Error reading user",
			fmt.Sprintf(
				"Could not read user %q by id %q, unexpected error: %q",
				state.Name.ValueString(),
				state.ID.ValueString(),
				err.Error(),
			),
		)
		return
	}

	if ytUser.MemberOf != nil {
		var memberOfWithoutBuiltin []string
		for _, u := range *ytUser.MemberOf {
			switch u {
			case "users":
				continue
			default:
				memberOfWithoutBuiltin = append(memberOfWithoutBuiltin, u)
			}
		}
		ytUser.MemberOf = &memberOfWithoutBuiltin
	}

	state = toUserModel(ytUser)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytUserState, diags := toYTsaurusUser(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytUserPlan, diags := toYTsaurusUser(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if ytUserState.Name != ytUserPlan.Name {
		p := ypath.Path(fmt.Sprintf("#%s", state.ID.ValueString())).Attr("name")
		if err := r.client.SetNode(ctx, p, ytUserPlan.Name, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error updating group 'name' attribute",
				fmt.Sprintf(
					"Could not set node %q to %q, unexpected error: %q",
					p.String(),
					ytUserPlan.Name,
					err.Error(),
				),
			)
			return
		}
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

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := ypath.Path(fmt.Sprintf("//sys/users/%s", state.Name.ValueString()))
	if err := r.client.RemoveNode(ctx, p, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting user",
			fmt.Sprintf(
				"Could not delete node %q, unexpected error: %q",
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
				"Error deleting user",
				fmt.Sprintf(
					"Could not delete node %q, unexpected error: %q",
					p.String(),
					err.Error(),
				),
			)
			return
		}
		if !ok {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
