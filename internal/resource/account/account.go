package account

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

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
	NodeCount          types.Int64 `tfsdk:"node_count"`
	ChunkCount         types.Int64 `tfsdk:"chunk_count"`
	TabletCount        types.Int64 `tfsdk:"tablet_count"`
	TabletStaticMemory types.Int64 `tfsdk:"tablet_static_memory"`
	DiskSpacePerMedium types.Map   `tfsdk:"disk_space_per_medium"`
}

type AccountModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ACL            types.List   `tfsdk:"acl"`
	ParentName     types.String `tfsdk:"parent_name"`
	ResourceLimits types.Object `tfsdk:"resource_limits"`
	InheritACL     types.Bool   `tfsdk:"inherit_acl"`
}

// flattenAccountResourceLimits performs YT to Terraform conversion for AccountResourceLimits.
// Direction: YT -> Terraform
func flattenAccountResourceLimits(ctx context.Context, limits ytsaurus.AccountResourceLimits) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrTypes := map[string]attr.Type{
		"node_count":            types.Int64Type,
		"chunk_count":           types.Int64Type,
		"tablet_count":          types.Int64Type,
		"tablet_static_memory":  types.Int64Type,
		"disk_space_per_medium": types.MapType{ElemType: types.Int64Type},
	}

	var diskSpacePerMedium types.Map
	if len(limits.DiskSpacePerMedium) == 0 {
		diskSpacePerMedium = types.MapNull(types.Int64Type)
	} else {
		elements := make(map[string]attr.Value, len(limits.DiskSpacePerMedium))
		for medium, limit := range limits.DiskSpacePerMedium {
			elements[medium] = types.Int64Value(limit)
		}
		diskSpacePerMedium, diags = types.MapValueFrom(ctx, types.Int64Type, elements)
		if diags.HasError() {
			return types.ObjectNull(attrTypes), diags
		}
	}

	accountResourceLimitsObj, diags := types.ObjectValueFrom(ctx, attrTypes, AccountResourceLimitsModel{
		NodeCount:          types.Int64Value(limits.NodeCount),
		ChunkCount:         types.Int64Value(limits.ChunkCount),
		TabletCount:        types.Int64Value(limits.TabletCount),
		TabletStaticMemory: types.Int64Value(limits.TabletStaticMemory),
		DiskSpacePerMedium: diskSpacePerMedium,
	})

	return accountResourceLimitsObj, diags
}

// flattenAccount performs YT to Terraform conversion for Account resources.
// Direction: YT -> Terraform
func flattenAccount(ctx context.Context, account ytsaurus.Account) (AccountModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	resourceLimits, diags := flattenAccountResourceLimits(ctx, account.ResourceLimits)
	acl, diags := acl.FlattenACL(ctx, account.ACL)

	accountModel := AccountModel{
		ID:             types.StringValue(account.ID),
		Name:           types.StringValue(account.Name),
		ACL:            acl,
		ResourceLimits: resourceLimits,
		InheritACL:     types.BoolValue(account.InheritACL),
	}
	if account.ParentName == "root" {
		accountModel.ParentName = types.StringNull()
	} else {
		accountModel.ParentName = types.StringValue(account.ParentName)
	}

	return accountModel, diags
}

// expandAccountResourceLimits performs Terraform to YT conversion for AccountResourceLimits resources.
// Direction: Terraform -> YT
func ExpandAccountResourceLimits(ctx context.Context, obj types.Object) (ytsaurus.AccountResourceLimits, diag.Diagnostics) {
	var diags diag.Diagnostics
	var limits ytsaurus.AccountResourceLimits

	if obj.IsNull() {
		diags.AddError(
			"AccountResourceLimits is Null",
			"",
		)
		return ytsaurus.AccountResourceLimits{}, diags
	}

	var limitsModel AccountResourceLimitsModel
	diags = obj.As(ctx, &limitsModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return ytsaurus.AccountResourceLimits{}, diags
	}

	limits.NodeCount = limitsModel.NodeCount.ValueInt64()
	limits.ChunkCount = limitsModel.ChunkCount.ValueInt64()
	limits.TabletCount = limitsModel.TabletCount.ValueInt64()
	limits.TabletStaticMemory = limitsModel.TabletStaticMemory.ValueInt64()
	if !limitsModel.DiskSpacePerMedium.IsNull() {
		elements := make(map[string]types.Int64)
		diags := limitsModel.DiskSpacePerMedium.ElementsAs(ctx, &elements, false)
		if diags.HasError() {
			return ytsaurus.AccountResourceLimits{}, diags
		}

		limits.DiskSpacePerMedium = make(map[string]int64, len(elements))
		for medium, val := range elements {
			limits.DiskSpacePerMedium[medium] = val.ValueInt64()
		}
	}

	return limits, diags
}

// expandAccount performs Terraform to YT conversion for Account resources.
// Direction: Terraform -> YT
func expandAccount(ctx context.Context, accountModel AccountModel) (ytsaurus.Account, diag.Diagnostics) {
	var diags diag.Diagnostics
	if accountModel.ID.IsNull() {
		diags.AddError(
			"accountModel is Null",
			"",
		)
		return ytsaurus.Account{}, diags
	}

	acl, diags := acl.ExpandACL(ctx, accountModel.ACL)
	resourceLimits, diags := ExpandAccountResourceLimits(ctx, accountModel.ResourceLimits)
	return ytsaurus.Account{
		Name:           accountModel.Name.ValueString(),
		ACL:            acl,
		ResourceLimits: resourceLimits,
		InheritACL:     accountModel.InheritACL.ValueBool(),
		ParentName:     accountModel.ParentName.ValueString(),
	}, diags
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
			"inherit_acl": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Enable or disable ACL inheritance from an object's parents.",
			},
			"resource_limits": schema.SingleNestedAttribute{
				Required:    true,
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

	ytAccount, diags := expandAccount(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createOptions := &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"name":               ytAccount.Name,
			"resource_limits":    ytAccount.ResourceLimits,
			"terraform_resource": true,
		},
	}

	if !plan.ACL.IsNull() {
		createOptions.Attributes["acl"] = ytAccount.ACL
	}
	if !plan.ParentName.IsNull() {
		createOptions.Attributes["parent_name"] = ytAccount.ParentName
	}
	if !plan.InheritACL.ValueBool() {
		createOptions.Attributes["inherit_acl"] = ytAccount.InheritACL
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
	exists, err := ytsaurus.ObjectExistsByID(ctx, r.client, objectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading account",
			fmt.Sprintf(
				"Could not check account by id %q, unexpected error: %q",
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

	state, diags := flattenAccount(ctx, ytAccount)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state AccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytAccount, diags := expandAccount(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := ypath.Path(fmt.Sprintf("#%s", objectID))
	attributeUpdates := map[string]interface{}{
		"name":            ytAccount.Name,
		"resource_limits": ytAccount.ResourceLimits,
	}

	if !plan.ACL.Equal(state.ACL) {
		attributeUpdates["acl"] = ytAccount.ACL
	}
	if !plan.InheritACL.Equal(state.InheritACL) {
		attributeUpdates["inherit_acl"] = ytAccount.InheritACL
	}

	if !plan.ParentName.IsNull() {
		attributeUpdates["parent_name"] = ytAccount.ParentName
	} else {
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

	ytAccount, diags := expandAccount(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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
