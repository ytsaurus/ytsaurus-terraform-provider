package objectschema

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/resource/acl"
	"terraform-provider-ytsaurus/internal/ytsaurus"
)

const (
	ACL_ATTR           = "acl"
	BEFORE_TF_ACL_ATTR = "before_tf_acl"
)

type objectSchemaResource struct {
	client yt.Client
}

var (
	_ resource.Resource                = &objectSchemaResource{}
	_ resource.ResourceWithConfigure   = &objectSchemaResource{}
	_ resource.ResourceWithImportState = &objectSchemaResource{}
)

type ObjectSchemaModel struct {
	ID  types.String `tfsdk:"id"`
	ACL acl.ACLModel `tfsdk:"acl"`
}

func toYTsaurusObjectSchema(m ObjectSchemaModel) (ytsaurus.ObjectSchema, diag.Diagnostics) {
	acl, diags := acl.ToYTsaurusACL(m.ACL)
	return ytsaurus.ObjectSchema{
		ID:  m.ID.ValueString(),
		ACL: acl,
	}, diags
}

func toObjectSchemaModel(m ytsaurus.ObjectSchema) ObjectSchemaModel {
	return ObjectSchemaModel{
		ID:  types.StringValue(m.ID),
		ACL: acl.ToACLModel(m.ACL),
	}
}

func NewObjectSchemaResource() resource.Resource {
	return &objectSchemaResource{}
}

func (r *objectSchemaResource) copyAttributeToAttribute(ctx context.Context, p ypath.Path, srcAttr, dstAttr string, ptrBuff any) error {
	if err := r.client.GetNode(ctx, p.Attr(srcAttr), ptrBuff, nil); err != nil {
		return err
	}
	return r.client.SetNode(ctx, p.Attr(dstAttr), ptrBuff, nil)
}

func (r *objectSchemaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_schema"
}

func (r *objectSchemaResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(yt.Client)
}

func (r *objectSchemaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
YT schemas are the parents for various cypress objects and are needed to calculate @effective_acl.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    false,
				Required:    true,
				Description: "ObjectID in the YTsaurus cluster, can be found in an object's @id attribute.",
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

func (r *objectSchemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ObjectSchemaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytObjectSchema, diags := toYTsaurusObjectSchema(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := ypath.Path(fmt.Sprintf("#%s", plan.ID.ValueString()))
	if ok, err := r.client.NodeExists(ctx, p, nil); err != nil || !ok {
		if err != nil {
			resp.Diagnostics.AddError(
				"Error importing schema",
				fmt.Sprintf(
					"Could not import schema %q, unexpected error: %q",
					p,
					err.Error(),
				),
			)
			return
		}
		if !ok {
			resp.Diagnostics.AddError(
				"Error importing schema",
				fmt.Sprintf(
					"The schema %q does not exist",
					p,
				),
			)
			return
		}
	}

	if err := r.copyAttributeToAttribute(ctx, p, ACL_ATTR, BEFORE_TF_ACL_ATTR, new([]yt.ACE)); err != nil {
		resp.Diagnostics.AddError(
			"Error importing schema",
			fmt.Sprintf(
				"Could copy attr %q to %q, unexpected error: %q",
				p.Attr("acl"),
				p.Attr("before_tf_acl"),
				err,
			),
		)
		return
	}

	if err := r.client.SetNode(ctx, p.Attr(ACL_ATTR), ytObjectSchema.ACL, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error importing schema",
			fmt.Sprintf(
				"Could not set attr %q, unexpected error: %q",
				p.Attr("acl"),
				err,
			),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, toObjectSchemaModel(ytObjectSchema))...)
}

func (r *objectSchemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ytObjectSchema ytsaurus.ObjectSchema
	exists, err := ytsaurus.ObjectExistsByID(ctx, r.client, objectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading schema",
			fmt.Sprintf(
				"Could not check schema with id %q, unexpected error: %q",
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

	if err := ytsaurus.GetObjectByID(ctx, r.client, objectID, &ytObjectSchema); err != nil {
		resp.Diagnostics.AddError(
			"Error reading schema",
			fmt.Sprintf(
				"Could not read schema with id %q, unexpected error: %q",
				objectID,
				err.Error(),
			),
		)
		return
	}

	state := toObjectSchemaModel(ytObjectSchema)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *objectSchemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ObjectSchemaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state ObjectSchemaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.ID.Equal(state.ID) {
		resp.Diagnostics.AddError(
			"Error updating object schema attributes",
			"Builtin attribute 'id' cannot be updated",
		)
		return
	}

	ytObjectSchema, diags := toYTsaurusObjectSchema(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := ypath.Path(fmt.Sprintf("#%s", plan.ID.ValueString()))
	if err := r.client.SetNode(ctx, p.Attr(ACL_ATTR), ytObjectSchema.ACL, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error updating object schema attributes",
			fmt.Sprintf(
				"Could not set %q to '%v', unexpected error: %q",
				p.Attr(ACL_ATTR).String(),
				ytObjectSchema.ACL,
				err.Error(),
			),
		)
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *objectSchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ObjectSchemaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := ypath.Path(fmt.Sprintf("#%s", state.ID.ValueString()))
	ok, err := r.client.NodeExists(ctx, p.Attr(BEFORE_TF_ACL_ATTR), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting schema",
			fmt.Sprintf(
				"Could not get attr %q, unexpected error: %q",
				p.Attr(BEFORE_TF_ACL_ATTR),
				err,
			),
		)
		return
	}

	if ok {
		if err := r.copyAttributeToAttribute(ctx, p, BEFORE_TF_ACL_ATTR, ACL_ATTR, new([]yt.ACE)); err != nil {
			resp.Diagnostics.AddError(
				"Error deleting schema",
				fmt.Sprintf(
					"Could not copy attr %q to %q, unexpected error: %q",
					p.Attr(BEFORE_TF_ACL_ATTR),
					p.Attr(ACL_ATTR),
					err,
				),
			)
			return
		}
		if err := r.client.RemoveNode(ctx, p.Attr(BEFORE_TF_ACL_ATTR), nil); err != nil {
			resp.Diagnostics.AddError(
				"Error deleting schema",
				fmt.Sprintf(
					"Could not remove attr %q, unexpected error: %q",
					p.Attr(BEFORE_TF_ACL_ATTR),
					err,
				),
			)
			return
		}
	}
}

func (r *objectSchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
