package account

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	defaultTabletCount        = 0
	defaultTabletStaticMemory = 0
)

type accountResource struct {
	client yt.Client
}

type AccountResourceLimitsModel struct {
	NodeCount          types.Int64            `tfsdk:"node_count"`
	ChunkCount         types.Int64            `tfsdk:"chunk_count"`
	TabletCount        types.Int64            `tfsdk:"tablet_count"`
	TabletStaticMemory types.Int64            `tfsdk:"tablet_static_memory"`
	DiskSpacePerMedium map[string]types.Int64 `tfsdk:"disk_space_per_medium"`
}

func toResourceLimitsModel(r *ytsaurus.AccountResourceLimits) *AccountResourceLimitsModel {
	if r == nil {
		return nil
	}
	resourceLimit := AccountResourceLimitsModel{
		NodeCount:          types.Int64Value(r.NodeCount),
		ChunkCount:         types.Int64Value(r.ChunkCount),
		TabletCount:        types.Int64Value(r.TabletCount),
		TabletStaticMemory: types.Int64Value(r.TabletStaticMemory),
		DiskSpacePerMedium: make(map[string]types.Int64),
	}
	for k, v := range r.DiskSpacePerMedium {
		resourceLimit.DiskSpacePerMedium[k] = types.Int64Value(v)
	}
	return &resourceLimit
}

func toYTsaurusAccountResourceLimits(r *AccountResourceLimitsModel) *ytsaurus.AccountResourceLimits {
	if r == nil {
		return nil
	}
	resourceLimits := ytsaurus.AccountResourceLimits{
		NodeCount:          r.NodeCount.ValueInt64(),
		ChunkCount:         r.ChunkCount.ValueInt64(),
		TabletCount:        r.TabletCount.ValueInt64(),
		TabletStaticMemory: r.TabletStaticMemory.ValueInt64(),
		DiskSpacePerMedium: make(map[string]int64),
	}
	for k, v := range r.DiskSpacePerMedium {
		resourceLimits.DiskSpacePerMedium[k] = v.ValueInt64()
	}
	return &resourceLimits
}

type AccountModel struct {
	ID             types.String                `tfsdk:"id"`
	Name           types.String                `tfsdk:"name"`
	InheritACL     types.Bool                  `tfsdk:"inherit_acl"`
	ACL            acl.ACLModel                `tfsdk:"acl"`
	ParentName     types.String                `tfsdk:"parent_name"`
	ResourceLimits *AccountResourceLimitsModel `tfsdk:"resource_limits"`
}

func toAccountModel(a ytsaurus.Account) AccountModel {
	account := AccountModel{
		ID:             types.StringValue(a.ID),
		Name:           types.StringValue(a.Name),
		InheritACL:     types.BoolValue(a.InheritACL),
		ACL:            acl.ToACLModel(a.ACL),
		ResourceLimits: toResourceLimitsModel(a.ResourceLimits),
	}

	if a.ParentName == "root" {
		account.ParentName = types.StringNull()
	} else {
		account.ParentName = types.StringValue(a.ParentName)
	}

	return account
}

func toYTsaurusAccount(a AccountModel) ytsaurus.Account {
	return ytsaurus.Account{
		Name:           a.Name.ValueString(),
		InheritACL:     a.InheritACL.ValueBool(),
		ACL:            acl.ToYTsaurusACL(a.ACL),
		ResourceLimits: toYTsaurusAccountResourceLimits(a.ResourceLimits),
		ParentName:     a.ParentName.ValueString(),
	}
}

var (
	_ resource.Resource                = &accountResource{}
	_ resource.ResourceWithConfigure   = &accountResource{}
	_ resource.ResourceWithImportState = &accountResource{}
)

func NewAccountResource() resource.Resource {
	return &accountResource{}
}

func (r *accountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (r *accountResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(yt.Client)
}

func (r *accountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Accounts are used to control and share the cluster's resources between users.

More information:
https://ytsaurus.tech/docs/en/user-guide/storage/accounts
		`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ObjectID in YTsaurus cluster, can be found in object's @id attribute",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "YTsaurus account name",
			},
			"inherit_acl": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Enable or disable ACL inheritance from an object's parents.",
			},
			"acl": schema.ListNestedAttribute{
				Optional:     true,
				NestedObject: acl.ACLSchema,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Description: "A list of ACE records. More information: https://ytsaurus.tech/docs/en/user-guide/storage/access-control",
			},
			"parent_name": schema.StringAttribute{
				Optional:    true,
				Description: "Parent account name",
			},
			"resource_limits": schema.SingleNestedAttribute{
				// Required:    true,
				Optional:    true,
				Description: "Resource limits for the account",
				Attributes: map[string]schema.Attribute{
					"node_count": schema.Int64Attribute{
						Required:    true,
						Description: "Number of Cypress nodes",
					},
					"chunk_count": schema.Int64Attribute{
						Required:    true,
						Description: "Number of chunks",
					},
					"tablet_count": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultTabletCount),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Number of tablets",
					},
					"tablet_static_memory": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(defaultTabletStaticMemory),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Description: "Memory volume for dynamic tables loaded into memory",
					},
					"disk_space_per_medium": schema.MapAttribute{
						Required:    true,
						ElementType: types.Int64Type,
						Description: "Disk space in bytes (for each medium)",
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *accountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytAccount := toYTsaurusAccount(plan)
	createOptions := &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"name":               ytAccount.Name,
			"inherit_acl":        ytAccount.InheritACL,
			"acl":                ytAccount.ACL,
			"terraform_resource": true,
		},
	}

	if ytAccount.ResourceLimits != nil {
		createOptions.Attributes["resource_limits"] = ytAccount.ResourceLimits
	}

	if len(ytAccount.ParentName) > 0 {
		createOptions.Attributes["parent_name"] = ytAccount.ParentName
	}

	id, err := r.client.CreateObject(ctx, yt.NodeAccount, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating account",
			fmt.Sprintf(
				"Could not create account %q, unexpected error: %q",
				ytAccount.Name,
				err.Error(),
			),
		)
		return
	}

	plan.ID = types.StringValue(id.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *accountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ytAccount ytsaurus.Account
	if err := ytsaurus.GetObjectByID(ctx, r.client, objectID, &ytAccount); err != nil {
		resp.Diagnostics.AddError(
			"Error reading account",
			fmt.Sprintf(
				"Could not read account by id %q, unexpected error: %q",
				objectID,
				err.Error(),
			),
		)
		return
	}

	state := toAccountModel(ytAccount)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytAccount := toYTsaurusAccount(plan)
	p := ypath.Path(fmt.Sprintf("#%s", objectID))
	attributeUpdates := map[string]interface{}{
		"inherit_acl": ytAccount.InheritACL,
		"acl":         ytAccount.ACL,
	}

	if ytAccount.Name != "root" {
		attributeUpdates["name"] = ytAccount.Name
	}

	if ytAccount.ResourceLimits != nil {
		attributeUpdates["resource_limits"] = ytAccount.ResourceLimits
	}

	if len(ytAccount.ParentName) > 0 {
		attributeUpdates["parent_name"] = ytAccount.ParentName
	} else if ytAccount.Name != "root" {
		attributeUpdates["parent_name"] = "root"
	}

	for k, v := range attributeUpdates {
		if err := r.client.SetNode(ctx, p.Attr(k), v, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error updating account attributes",
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

	plan.ID = types.StringValue(objectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *accountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytAccount := toYTsaurusAccount(state)
	p := ypath.Path(fmt.Sprintf("//sys/accounts/%s", ytAccount.Name))

	var subNodes []string

	if err := r.client.ListNode(ctx, p, &subNodes, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting account",
			fmt.Sprintf(
				"Could not list path %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}
	if len(subNodes) > 0 {
		resp.Diagnostics.AddError(
			"Error deleting account",
			fmt.Sprintf(
				"Please, remove all subaccounts first: %s",
				strings.Join(subNodes, ","),
			),
		)
		return
	}

	if err := r.client.RemoveNode(ctx, p, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting account",
			fmt.Sprintf(
				"Could not delete account %q, unexpected error: %q",
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
				"Error deleting account",
				fmt.Sprintf(
					"Could not check is %q exist, unexpected error: %q",
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

func (r *accountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
