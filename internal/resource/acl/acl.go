package acl

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

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
	Action          types.String   `tfsdk:"action"`
	Subjects        []types.String `tfsdk:"subjects"`
	Permissions     []types.String `tfsdk:"permissions"`
	InheritanceMode types.String   `tfsdk:"inheritance_mode"`
}

func toYTsaurusACE(ace ACEModel) yt.ACE {
	ytACE := yt.ACE{}

	ytACE.Action = yt.SecurityAction(ace.Action.ValueString())
	ytACE.InheritanceMode = ace.InheritanceMode.ValueString()
	for _, s := range ace.Subjects {
		ytACE.Subjects = append(ytACE.Subjects, s.ValueString())
	}
	for _, p := range ace.Permissions {
		ytACE.Permissions = append(ytACE.Permissions, yt.Permission(p.ValueString()))
	}

	return ytACE
}

func toACEModel(ytACE yt.ACE) ACEModel {
	ace := ACEModel{}
	ace.Action = types.StringValue(string(ytACE.Action))
	ace.InheritanceMode = types.StringValue(ytACE.InheritanceMode)
	for _, s := range ytACE.Subjects {
		ace.Subjects = append(ace.Subjects, types.StringValue(s))
	}
	for _, p := range ytACE.Permissions {
		ace.Permissions = append(ace.Permissions, types.StringValue(string(p)))
	}
	return ace
}

type ACLModel []ACEModel

func ToYTsaurusACL(acl ACLModel) []yt.ACE {
	var ytACL []yt.ACE
	for _, ace := range acl {
		ytACL = append(ytACL, toYTsaurusACE(ace))
	}
	return ytACL
}

func ToACLModel(ytACL []yt.ACE) ACLModel {
	var acl ACLModel
	for _, ytACE := range ytACL {
		acl = append(acl, toACEModel(ytACE))
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
