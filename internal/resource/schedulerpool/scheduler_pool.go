package schedulerpool

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	minWeight                   = 1.0
	minMaxRunningOperationCount = 1
	minMaxOperationCount        = 1
)

type schedulerPoolResource struct {
	client yt.Client
}

type SchedulerPoolResourcesModel struct {
	CPU    types.Int64 `tfsdk:"cpu"`
	Memory types.Int64 `tfsdk:"memory"`
}

type SchedulerPoolIntegralGuaranteesModel struct {
	GuaranteeType           types.String                 `tfsdk:"guarantee_type"`
	ResourceFlow            *SchedulerPoolResourcesModel `tfsdk:"resource_flow"`
	BurstGuaranteeResources *SchedulerPoolResourcesModel `tfsdk:"burst_guarantee_resources"`
}

type SchedulerPoolModel struct {
	ID                        types.String                          `tfsdk:"id"`
	Name                      types.String                          `tfsdk:"name"`
	PoolTree                  types.String                          `tfsdk:"pool_tree"`
	ACL                       acl.ACLModel                          `tfsdk:"acl"`
	ParentName                types.String                          `tfsdk:"parent_name"`
	MaxRunningOperationCount  types.Int64                           `tfsdk:"max_running_operation_count"`
	MaxOperationCount         types.Int64                           `tfsdk:"max_operation_count"`
	StrongGuaranteeResources  *SchedulerPoolResourcesModel          `tfsdk:"strong_guarantee_resources"`
	IntegralGuarantees        *SchedulerPoolIntegralGuaranteesModel `tfsdk:"integral_guarantees"`
	ResourceLimits            *SchedulerPoolResourcesModel          `tfsdk:"resource_limits"`
	ForbidImmediateOperations types.Bool                            `tfsdk:"forbid_immediate_operations"`
	Weight                    types.Float64                         `tfsdk:"weight"`
	Mode                      types.String                          `tfsdk:"mode"`
}

func toSchedulerPoolIntegralGuaranteesModel(g *ytsaurus.SchedulerPoolIntegralGuarantees) *SchedulerPoolIntegralGuaranteesModel {
	if g != nil {
		return &SchedulerPoolIntegralGuaranteesModel{
			GuaranteeType:           types.StringPointerValue(g.GuaranteeType),
			ResourceFlow:            toSchedulerPoolResourcesModel(g.ResourceFlow),
			BurstGuaranteeResources: toSchedulerPoolResourcesModel(g.BurstGuaranteeResources),
		}
	} else {
		return nil
	}
}

func toYTsaurusSchedulerPoolIntegralGuarantees(g *SchedulerPoolIntegralGuaranteesModel) *ytsaurus.SchedulerPoolIntegralGuarantees {
	if g != nil {
		return &ytsaurus.SchedulerPoolIntegralGuarantees{
			GuaranteeType:           g.GuaranteeType.ValueStringPointer(),
			ResourceFlow:            toYTsaurusSchedulerPoolResources(g.ResourceFlow),
			BurstGuaranteeResources: toYTsaurusSchedulerPoolResources(g.BurstGuaranteeResources),
		}
	} else {
		return nil
	}
}

func toSchedulerPoolResourcesModel(r *ytsaurus.SchedulerPoolResources) *SchedulerPoolResourcesModel {
	if r != nil {
		return &SchedulerPoolResourcesModel{
			CPU:    types.Int64PointerValue(r.CPU),
			Memory: types.Int64PointerValue(r.Memory),
		}
	} else {
		return nil
	}
}

func toYTsaurusSchedulerPoolResources(r *SchedulerPoolResourcesModel) *ytsaurus.SchedulerPoolResources {
	if r != nil {
		return &ytsaurus.SchedulerPoolResources{
			CPU:    r.CPU.ValueInt64Pointer(),
			Memory: r.Memory.ValueInt64Pointer(),
		}
	} else {
		return nil
	}
}

func toSchedulerPoolModel(p ytsaurus.SchedulerPool) SchedulerPoolModel {
	model := SchedulerPoolModel{
		ID:                        types.StringValue(p.ID),
		Name:                      types.StringValue(p.Name),
		ACL:                       acl.ToACLModel(p.ACL),
		MaxRunningOperationCount:  types.Int64PointerValue(p.MaxRunningOperationCount),
		MaxOperationCount:         types.Int64PointerValue(p.MaxOperationCount),
		IntegralGuarantees:        toSchedulerPoolIntegralGuaranteesModel(p.IntegralGuarantees),
		StrongGuaranteeResources:  toSchedulerPoolResourcesModel(p.StrongGuaranteeResources),
		ResourceLimits:            toSchedulerPoolResourcesModel(p.ResourceLimits),
		Weight:                    types.Float64PointerValue(p.Weight),
		Mode:                      types.StringPointerValue(p.Mode),
		ForbidImmediateOperations: types.BoolPointerValue(p.ForbidImmediateOperations),
	}

	if p.ParentName != nil && strings.HasPrefix(*p.ParentName, "#") {
		model.ParentName = types.StringNull()
	} else {
		model.ParentName = types.StringPointerValue(p.ParentName)
	}

	if len(p.Path) > 0 {
		p, ok := strings.CutPrefix(p.Path, "//sys/pool_trees/")
		if !ok {
			return model
		}

		poolTree, _, ok := strings.Cut(p, "/")
		if !ok {
			return model
		}
		model.PoolTree = types.StringValue(poolTree)
	}

	return model
}

func toYTsaurusSchedulerPool(p SchedulerPoolModel) ytsaurus.SchedulerPool {
	return ytsaurus.SchedulerPool{
		ID:                        p.ID.ValueString(),
		Name:                      p.Name.ValueString(),
		ACL:                       acl.ToYTsaurusACL(p.ACL),
		ParentName:                p.ParentName.ValueStringPointer(),
		MaxRunningOperationCount:  p.MaxRunningOperationCount.ValueInt64Pointer(),
		MaxOperationCount:         p.MaxOperationCount.ValueInt64Pointer(),
		IntegralGuarantees:        toYTsaurusSchedulerPoolIntegralGuarantees(p.IntegralGuarantees),
		StrongGuaranteeResources:  toYTsaurusSchedulerPoolResources(p.StrongGuaranteeResources),
		ResourceLimits:            toYTsaurusSchedulerPoolResources(p.ResourceLimits),
		Weight:                    p.Weight.ValueFloat64Pointer(),
		Mode:                      p.Mode.ValueStringPointer(),
		ForbidImmediateOperations: p.ForbidImmediateOperations.ValueBoolPointer(),
		Path:                      fmt.Sprintf("//sys/pool_trees/%s/%s", p.PoolTree.ValueString(), p.Name.ValueString()),
	}
}

func ytSchedulerPoolResourcesToMap(r *ytsaurus.SchedulerPoolResources) map[string]int64 {
	m := map[string]int64{}
	if r != nil {
		if r.CPU != nil {
			m["cpu"] = *r.CPU
		}
		if r.Memory != nil {
			m["memory"] = *r.Memory
		}
	}
	return m
}

func ytSchedulerPoolIntegralGuaranteesToMap(g *ytsaurus.SchedulerPoolIntegralGuarantees) map[string]interface{} {
	m := make(map[string]interface{})
	if g != nil {
		if g.GuaranteeType != nil {
			m["guarantee_type"] = *g.GuaranteeType
		}
		if g.ResourceFlow != nil {
			m["resource_flow"] = ytSchedulerPoolResourcesToMap(g.ResourceFlow)
		}
		if g.BurstGuaranteeResources != nil {
			m["burst_guarantee_resources"] = ytSchedulerPoolResourcesToMap(g.BurstGuaranteeResources)
		}
	}
	return m
}

var (
	_ resource.Resource                     = &schedulerPoolResource{}
	_ resource.ResourceWithConfigure        = &schedulerPoolResource{}
	_ resource.ResourceWithImportState      = &schedulerPoolResource{}
	_ resource.ResourceWithConfigValidators = &schedulerPoolResource{}
)

func NewSchedulerPoolResource() resource.Resource {
	return &schedulerPoolResource{}
}

func (r *schedulerPoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduler_pool"
}

func (r *schedulerPoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	schedulerPoolResourcesSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"cpu": schema.Int64Attribute{
				Optional:    true,
				Description: "CPU cores limit.",
			},
			"memory": schema.Int64Attribute{
				Optional:    true,
				Description: "Memory limit in bytes.",
			},
		},
		Validators: []validator.Object{
			objectvalidator.Any(
				objectvalidator.AlsoRequires(
					path.Expressions{path.MatchRelative().AtName("cpu")}...),
				objectvalidator.AlsoRequires(
					path.Expressions{path.MatchRelative().AtName("memory")}...),
			),
		},
	}

	strongGuaranteeResources := schedulerPoolResourcesSchema
	strongGuaranteeResources.Description = "The pool's guaranteed resources."

	resourceFlow := schedulerPoolResourcesSchema
	resourceFlow.Description = ""

	burstGuaranteeResources := schedulerPoolResourcesSchema
	burstGuaranteeResources.Description = ""

	resourceLimits := schedulerPoolResourcesSchema
	resourceLimits.Description = "The resource_limits option describes the limits on different resources of a given pool."

	resp.Schema = schema.Schema{
		Description: `
A pool is a container for the CPU and RAM resources that the scheduler uses.

More information:
https://ytsaurus.tech/docs/en/user-guide/data-processing/scheduler/scheduler-and-pools
and
https://ytsaurus.tech/docs/en/user-guide/data-processing/scheduler/pool-settings`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ObjectID in YTsaurus cluster, can be found in object's @id attribute.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "YTsaurus scheduler_poll name.",
			},
			"pool_tree": schema.StringAttribute{
				Required:    true,
				Description: "A pool_tree name for the pool.",
			},
			"acl": schema.ListNestedAttribute{
				Optional:     true,
				NestedObject: acl.ACLSchema,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Description: "A list of ACE records. More information: https://ytsaurus.tech/docs/en/user-guide/storage/access-control.",
			},
			"parent_name": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "A name of parent pool in the same pool_tree.",
			},
			"max_running_operation_count": schema.Int64Attribute{
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(minMaxRunningOperationCount),
				},
				Description: "Maximum number of operations in running state.",
			},
			"max_operation_count": schema.Int64Attribute{
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(minMaxOperationCount),
				},
				Description: "Maximum number of operations in all states.",
			},
			"strong_guarantee_resources": strongGuaranteeResources,
			"integral_guarantees": schema.SingleNestedAttribute{
				Description: `Integral guarantees configuration. More information: https://ytsaurus.tech/docs/en/user-guide/data-processing/scheduler/integral-guarantees.`,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"guarantee_type": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.OneOf(
								"burst",
								"relaxed",
								"none",
							),
						},
						Description: "A guarantee type, can be 'burst' or 'relaxed'.",
					},
					"resource_flow":             resourceFlow,
					"burst_guarantee_resources": burstGuaranteeResources,
				},
				Validators: []validator.Object{
					objectvalidator.Any(
						objectvalidator.AlsoRequires(
							path.Expressions{path.MatchRelative().AtName("guarantee_type")}...),
						objectvalidator.AlsoRequires(
							path.Expressions{path.MatchRelative().AtName("resource_flow")}...),
						objectvalidator.AlsoRequires(
							path.Expressions{path.MatchRelative().AtName("burst_guarantee_resources")}...),
					),
				},
			},
			"resource_limits": resourceLimits,
			"forbid_immediate_operations": schema.BoolAttribute{
				Optional:    true,
				Description: "Prohibits the start of operations directly in the given pool; does not apply to starting operations in subpools.",
			},
			"weight": schema.Float64Attribute{
				Optional: true,
				Validators: []validator.Float64{
					float64validator.AtLeast(minWeight),
				},
				Description: "A real non-negative number, which is responsible for the proportion in which the subtree should be provided with the resources of the parent pool.",
			},
			"mode": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"fair_share",
						"fifo",
					),
				},
				Description: "The scheduling mode. Can be 'fifo' or 'fair_share'.",
			},
		},
	}

}

func (r *schedulerPoolResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(yt.Client)
}

func ppanic(v interface{}) {
	panic(fmt.Sprintf("%v", v))
}

func (r *schedulerPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SchedulerPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytSchedulerPool := toYTsaurusSchedulerPool(plan)
	createOptions := &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"name":               ytSchedulerPool.Name,
			"acl":                ytSchedulerPool.ACL,
			"pool_tree":          plan.PoolTree.ValueString(),
			"terraform_resource": true,
		},
	}

	if ytSchedulerPool.ParentName != nil {
		createOptions.Attributes["parent_name"] = *ytSchedulerPool.ParentName
	}
	if ytSchedulerPool.MaxRunningOperationCount != nil {
		createOptions.Attributes["max_running_operation_count"] = *ytSchedulerPool.MaxRunningOperationCount
	}
	if ytSchedulerPool.MaxOperationCount != nil {
		createOptions.Attributes["max_operation_count"] = *ytSchedulerPool.MaxOperationCount
	}
	if ytSchedulerPool.Weight != nil {
		createOptions.Attributes["weight"] = *ytSchedulerPool.Weight
	}
	if ytSchedulerPool.Mode != nil {
		createOptions.Attributes["mode"] = *ytSchedulerPool.Mode
	}
	if ytSchedulerPool.ForbidImmediateOperations != nil {
		createOptions.Attributes["forbid_immediate_operations"] = *ytSchedulerPool.ForbidImmediateOperations
	}

	resourceLimits := ytSchedulerPoolResourcesToMap(ytSchedulerPool.ResourceLimits)
	if len(resourceLimits) > 0 {
		createOptions.Attributes["resource_limits"] = resourceLimits
	}

	strongGuaranteeResources := ytSchedulerPoolResourcesToMap(ytSchedulerPool.StrongGuaranteeResources)
	if len(strongGuaranteeResources) > 0 {
		createOptions.Attributes["strong_guarantee_resources"] = strongGuaranteeResources
	}

	integralGuarantees := ytSchedulerPoolIntegralGuaranteesToMap(ytSchedulerPool.IntegralGuarantees)
	if ytSchedulerPool.IntegralGuarantees != nil {
		createOptions.Attributes["integral_guarantees"] = integralGuarantees
	}

	id, err := r.client.CreateObject(ctx, yt.NodeSchedulerPool, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating scheduler_pool",
			fmt.Sprintf(
				"Could not create scheduler_pool %q, unexpected error: %q",
				ytSchedulerPool.Name,
				err.Error(),
			),
		)
		return
	}

	ytSchedulerPool.ID = id.String()

	state := toSchedulerPoolModel(ytSchedulerPool)
	state.PoolTree = plan.PoolTree

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *schedulerPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ytSchedulerPool ytsaurus.SchedulerPool
	if err := ytsaurus.GetObjectByID(ctx, r.client, objectID, &ytSchedulerPool); err != nil {
		resp.Diagnostics.AddError(
			"Error reading scheduler_pool",
			fmt.Sprintf(
				"Could not read scheduler_pool by id %q, unexpected error: %q",
				objectID,
				err.Error(),
			),
		)
		return
	}

	state := toSchedulerPoolModel(ytSchedulerPool)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *schedulerPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SchedulerPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state SchedulerPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytSchedulerPoolPlan := toYTsaurusSchedulerPool(plan)
	ytSchedulerPoolState := toYTsaurusSchedulerPool(state)

	attributeUpdates := map[string]interface{}{
		"name": ytSchedulerPoolPlan.Name,
		"acl":  ytSchedulerPoolPlan.ACL,
	}

	if ytSchedulerPoolPlan.ParentName != nil {
		attributeUpdates["parent_name"] = *ytSchedulerPoolPlan.ParentName
	}
	if ytSchedulerPoolPlan.MaxRunningOperationCount != nil {
		attributeUpdates["max_running_operation_count"] = *ytSchedulerPoolPlan.MaxRunningOperationCount
	}
	if ytSchedulerPoolPlan.MaxRunningOperationCount != nil {
		attributeUpdates["max_running_operation_count"] = *ytSchedulerPoolPlan.MaxRunningOperationCount
	}
	if ytSchedulerPoolPlan.MaxOperationCount != nil {
		attributeUpdates["max_operation_count"] = *ytSchedulerPoolPlan.MaxOperationCount
	}
	if ytSchedulerPoolPlan.Weight != nil {
		attributeUpdates["weight"] = *ytSchedulerPoolPlan.Weight
	}
	if ytSchedulerPoolPlan.Mode != nil {
		attributeUpdates["mode"] = *ytSchedulerPoolPlan.Mode
	}
	if ytSchedulerPoolPlan.ForbidImmediateOperations != nil {
		attributeUpdates["forbid_immediate_operations"] = *ytSchedulerPoolPlan.ForbidImmediateOperations
	}

	resourceLimits := ytSchedulerPoolResourcesToMap(ytSchedulerPoolPlan.ResourceLimits)
	if len(resourceLimits) > 0 {
		attributeUpdates["resource_limits"] = resourceLimits
	}

	strongGuaranteeResources := ytSchedulerPoolResourcesToMap(ytSchedulerPoolPlan.StrongGuaranteeResources)
	if len(strongGuaranteeResources) > 0 {
		attributeUpdates["strong_guarantee_resources"] = strongGuaranteeResources
	}

	integralGuarantees := ytSchedulerPoolIntegralGuaranteesToMap(ytSchedulerPoolPlan.IntegralGuarantees)
	if len(integralGuarantees) > 0 {
		attributeUpdates["integral_guarantees"] = integralGuarantees
	}

	p := ypath.Path(fmt.Sprintf("#%s", ytSchedulerPoolState.ID))
	for k, v := range attributeUpdates {
		if err := r.client.SetNode(ctx, p.Attr(k), v, nil); err != nil {
			resp.Diagnostics.AddError(
				"Error updating scheduler_pool",
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

	var attributeToDelete []string

	if ytSchedulerPoolPlan.MaxRunningOperationCount == nil && ytSchedulerPoolState.MaxRunningOperationCount != nil {
		attributeToDelete = append(attributeToDelete, "max_running_operation_count")
	}
	if ytSchedulerPoolPlan.MaxOperationCount == nil && ytSchedulerPoolState.MaxOperationCount != nil {
		attributeToDelete = append(attributeToDelete, "max_operation_count")
	}
	if ytSchedulerPoolPlan.Weight == nil && ytSchedulerPoolState.Weight != nil {
		attributeToDelete = append(attributeToDelete, "weight")
	}
	if ytSchedulerPoolPlan.Mode == nil && ytSchedulerPoolState.Mode != nil {
		attributeToDelete = append(attributeToDelete, "mode")
	}
	if ytSchedulerPoolPlan.ForbidImmediateOperations == nil && ytSchedulerPoolState.ForbidImmediateOperations != nil {
		attributeToDelete = append(attributeToDelete, "forbid_immediate_operations")
	}
	if ytSchedulerPoolPlan.ResourceLimits == nil && ytSchedulerPoolState.ResourceLimits != nil {
		attributeToDelete = append(attributeToDelete, "resource_limits")
	}
	if ytSchedulerPoolPlan.StrongGuaranteeResources == nil && ytSchedulerPoolState.StrongGuaranteeResources != nil {
		attributeToDelete = append(attributeToDelete, "strong_guarantee_resources")
	}
	if ytSchedulerPoolPlan.IntegralGuarantees == nil && ytSchedulerPoolState.IntegralGuarantees != nil {
		attributeToDelete = append(attributeToDelete, "integral_guarantees")
	}

	for _, k := range attributeToDelete {
		if err := r.client.RemoveNode(ctx, p.Attr(k), nil); err != nil {
			resp.Diagnostics.AddError(
				"Error updating scheduler_pool",
				fmt.Sprintf(
					"Could not remove %q, unexpected error: %q",
					p.Attr(k).String(),
					err.Error(),
				),
			)
			return
		}
	}

	if ytSchedulerPoolPlan.ParentName == nil && ytSchedulerPoolState.ParentName != nil {
		p = ypath.Path(fmt.Sprintf("#%s", ytSchedulerPoolState.ID)).Attr("parent_name")
		if err := r.client.SetNode(ctx, p, "<Root>", nil); err != nil {
			resp.Diagnostics.AddError(
				"Error updating scheduler_pool",
				fmt.Sprintf(
					"Could not set node %q to <Root>, unexpected error: %q",
					p.String(),
					err.Error(),
				),
			)
			return
		}
	}

	ytSchedulerPoolPlan.ID = ytSchedulerPoolState.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, toSchedulerPoolModel(ytSchedulerPoolPlan))...)
}

func (r *schedulerPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SchedulerPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytSchedulerPool := toYTsaurusSchedulerPool(state)
	p := ypath.Path(fmt.Sprintf("#%s", ytSchedulerPool.ID))
	if err := r.client.RemoveNode(ctx, p, nil); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting scheduler_pool",
			fmt.Sprintf(
				"Could delete scheduler_pool %q, unexpected error: %q",
				p.String(),
				err.Error(),
			),
		)
		return
	}
}

func (r *schedulerPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *schedulerPoolResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		schedulerPoolResourceConfigValidator{},
	}
}
