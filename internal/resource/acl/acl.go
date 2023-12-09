package acl

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"go.ytsaurus.tech/yt/go/yt"
)

const (
	InheritanceModeObjectAndDescendants     string = "object_and_descendants"
	InheritanceModeObjectOnly               string = "object_only"
	InheritanceModeDescendantsOnly          string = "descendants_only"
	InheritanceModeImmediateDescendantsOnly string = "immediate_descendants_only"
)

type ACEModel struct {
	Action          types.String `tfsdk:"action"`
	Subjects        types.Set    `tfsdk:"subjects"`
	Permissions     types.Set    `tfsdk:"permissions"`
	InheritanceMode types.String `tfsdk:"inheritance_mode"`
}

func toYTsaurusACE(ace ACEModel) (yt.ACE, diag.Diagnostics) {
	ytACE := yt.ACE{}
	var diags diag.Diagnostics

	ytACE.Action = yt.SecurityAction(ace.Action.ValueString())
	ytACE.InheritanceMode = ace.InheritanceMode.ValueString()

	if !ace.Subjects.IsUnknown() {
		ytACE.Subjects = make([]string, 0, len(ace.Subjects.Elements()))
		diags.Append(ace.Subjects.ElementsAs(context.TODO(), &ytACE.Subjects, false)...)
	}

	if !ace.Permissions.IsUnknown() {
		permissions := make([]string, 0, len(ace.Permissions.Elements()))
		diags.Append(ace.Permissions.ElementsAs(context.TODO(), &permissions, false)...)
		for _, p := range permissions {
			ytACE.Permissions = append(ytACE.Permissions, yt.Permission(p))
		}
	}

	return ytACE, diags
}

func toACEModel(ytACE yt.ACE) (ACEModel, diag.Diagnostics) {
	ace := ACEModel{}
	var diags diag.Diagnostics

	ace.Action = types.StringValue(string(ytACE.Action))
	ace.InheritanceMode = types.StringValue(ytACE.InheritanceMode)

	ace.Subjects, diags = types.SetValueFrom(context.TODO(), types.StringType, ytACE.Subjects)

	permissions := make([]string, 0, len(ytACE.Permissions))
	for _, p := range ytACE.Permissions {
		permissions = append(permissions, string(p))
	}
	var valDiags diag.Diagnostics
	ace.Permissions, valDiags = types.SetValueFrom(context.TODO(), types.StringType, permissions)
	diags.Append(valDiags...)

	return ace, diags
}

type ACLModel []ACEModel

func ToYTsaurusACL(acl ACLModel) ([]yt.ACE, diag.Diagnostics) {
	var ytACL []yt.ACE
	var diags diag.Diagnostics

	for _, ace := range acl {
		ytACE, aceDiags := toYTsaurusACE(ace)
		ytACL = append(ytACL, ytACE)
		diags.Append(aceDiags...)
	}

	return ytACL, diags
}

func ToACLModel(ytACL []yt.ACE) ACLModel {
	var acl ACLModel
	var diags diag.Diagnostics

	for _, ytACE := range ytACL {
		ace, aceDiags := toACEModel(ytACE)
		acl = append(acl, ace)
		diags.Append(aceDiags...)
	}

	return acl
}

var ACLSchema = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"action": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.OneOf(
					string(yt.ActionAllow),
					string(yt.ActionDeny),
				),
			},
			Description: "Either allow (allowing entry) or deny (denying entry).",
		},
		"subjects": schema.SetAttribute{
			Required:    true,
			ElementType: types.StringType,
			Description: "A list of names of subjects (users or groups) to which the entry applies.",
		},
		"permissions": schema.SetAttribute{
			Required:    true,
			ElementType: types.StringType,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(
					stringvalidator.OneOf(
						string(yt.PermissionRead),
						string(yt.PermissionWrite),
						string(yt.PermissionUse),
						string(yt.PermissionAdminister),
						string(yt.PermissionCreate),
						string(yt.PermissionRemove),
						string(yt.PermissionMount),
						string(yt.PermissionManage),
						string(yt.PermissionModifyChildren),
					),
				),
			},
			Description: `A list of access types also called permissions.
Supported permissions:
  - read - Means reading a value or getting information about an object or its attributes
  - write - Means changing an object's state or its attributes
  - use - Applies to accounts, pools, and bundles and means usage (that is, the ability to insert new objects into the quota of a given account, run operations in a pool, or move a dynamic table to a bundle)
  - administer - Means changing the object access descriptor
  - create - Applies only to schemas and means creating objects of this type
  - remove - Means removing an object
  - mount - Means mounting, unmounting, remounting, and resharding a dynamic table
  - manage - Applies only to operations (not to Cypress nodes) and means managing that operation or its jobs
`,
		},
		"inheritance_mode": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString(InheritanceModeObjectAndDescendants),
			Validators: []validator.String{
				stringvalidator.OneOf(
					InheritanceModeObjectAndDescendants,
					InheritanceModeObjectOnly,
					InheritanceModeImmediateDescendantsOnly,
					InheritanceModeDescendantsOnly,
				),
			},
			Description: `The inheritance mode of this ACE, by default.
Can be:
  - object_only - The object_only value means that this entry affects only the object itself
  - object_and_descendants - The object_and_descendants value means that this entry affects the object and all its descendants, including indirect ones
  - descendants_only - The descendants_only value means that this entry affects only descendants, including indirect ones. 
  - immediate_descendants_only - The immediate_descendants_only value means that this entry affects only direct descendants (sons)
			`,
		},
	},
}
