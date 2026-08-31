package schedulerpool

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

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
	GuaranteeType           types.String `tfsdk:"guarantee_type"`
	ResourceFlow            types.Object `tfsdk:"resource_flow"`
	BurstGuaranteeResources types.Object `tfsdk:"burst_guarantee_resources"`
}

type SchedulerPoolModel struct {
	ID                        types.String  `tfsdk:"id"`
	Name                      types.String  `tfsdk:"name"`
	PoolTree                  types.String  `tfsdk:"pool_tree"`
	ACL                       types.List    `tfsdk:"acl"`
	ParentName                types.String  `tfsdk:"parent_name"`
	MaxRunningOperationCount  types.Int64   `tfsdk:"max_running_operation_count"`
	MaxOperationCount         types.Int64   `tfsdk:"max_operation_count"`
	StrongGuaranteeResources  types.Object  `tfsdk:"strong_guarantee_resources"`
	IntegralGuarantees        types.Object  `tfsdk:"integral_guarantees"`
	ResourceLimits            types.Object  `tfsdk:"resource_limits"`
	ForbidImmediateOperations types.Bool    `tfsdk:"forbid_immediate_operations"`
	Weight                    types.Float64 `tfsdk:"weight"`
	Mode                      types.String  `tfsdk:"mode"`
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

// flattenSchedulerPoolResources performs YT to Terraform conversion for SchedulerPoolResourcesModel.
// Direction: YT -> Terraform
func flattenResources(ctx context.Context, resources ytsaurus.SchedulerPoolResources) (types.Object, diag.Diagnostics) {
	attrTypes := map[string]attr.Type{
		"cpu":    types.Int64Type,
		"memory": types.Int64Type,
	}

	cpu := types.Int64Null()
	if resources.CPU != nil {
		cpu = types.Int64Value(*resources.CPU)
	}

	memory := types.Int64Null()
	if resources.Memory != nil {
		memory = types.Int64Value(*resources.Memory)
	}

	return types.ObjectValueFrom(ctx, attrTypes, SchedulerPoolResourcesModel{
		CPU:    cpu,
		Memory: memory,
	})
}

// flattenIntegralGuarantees performs YT to Terraform conversion for SchedulerPoolIntegralGuaranteesModel.
// Direction: YT -> Terraform
func flattenIntegralGuarantees(ctx context.Context, integralGuarantees ytsaurus.SchedulerPoolIntegralGuarantees) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrTypes := map[string]attr.Type{
		"guarantee_type": types.StringType,
		"resource_flow": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"cpu":    types.Int64Type,
				"memory": types.Int64Type,
			},
		},
		"burst_guarantee_resources": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"cpu":    types.Int64Type,
				"memory": types.Int64Type,
			},
		},
	}

	resourcesAttrTypes := map[string]attr.Type{
		"cpu":    types.Int64Type,
		"memory": types.Int64Type,
	}

	var d diag.Diagnostics

	guaranteeType := types.StringNull()
	if integralGuarantees.GuaranteeType != nil {
		guaranteeType = types.StringValue(*integralGuarantees.GuaranteeType)
	}

	resourceFlow := types.ObjectNull(resourcesAttrTypes)
	if integralGuarantees.ResourceFlow != nil {
		resourceFlow, d = flattenResources(ctx, *integralGuarantees.ResourceFlow)
		diags.Append(d...)
	}

	burstGuaranteeResources := types.ObjectNull(resourcesAttrTypes)
	if integralGuarantees.BurstGuaranteeResources != nil {
		burstGuaranteeResources, d = flattenResources(ctx, *integralGuarantees.BurstGuaranteeResources)
		diags.Append(d...)
	}

	if diags.HasError() {
		return types.ObjectUnknown(attrTypes), diags
	}

	return types.ObjectValueFrom(ctx, attrTypes, SchedulerPoolIntegralGuaranteesModel{
		GuaranteeType:           guaranteeType,
		ResourceFlow:            resourceFlow,
		BurstGuaranteeResources: burstGuaranteeResources,
	})
}

// flattenSchedulerPool performs YT to Terraform conversion for SchedulerPoolModel.
// Direction: YT -> Terraform
func flattenSchedulerPool(ctx context.Context, schedulerPool ytsaurus.SchedulerPool) (SchedulerPoolModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var d diag.Diagnostics

	acl, d := acl.FlattenACL(ctx, schedulerPool.ACL)
	diags.Append(d...)
	if diags.HasError() {
		return SchedulerPoolModel{}, diags
	}

	strongGuaranteeResourcesObj := types.ObjectNull(map[string]attr.Type{
		"cpu":    types.Int64Type,
		"memory": types.Int64Type,
	})
	if schedulerPool.StrongGuaranteeResources != nil {
		strongGuaranteeResourcesObj, d = flattenResources(ctx, *schedulerPool.StrongGuaranteeResources)
		diags.Append(d...)
		if diags.HasError() {
			return SchedulerPoolModel{}, diags
		}
	}

	resourceLimitsObj := types.ObjectNull(map[string]attr.Type{
		"cpu":    types.Int64Type,
		"memory": types.Int64Type,
	})
	if schedulerPool.ResourceLimits != nil {
		resourceLimitsObj, d = flattenResources(ctx, *schedulerPool.ResourceLimits)
		diags.Append(d...)
		if diags.HasError() {
			return SchedulerPoolModel{}, diags
		}
	}

	integralGuarantees := types.ObjectNull(map[string]attr.Type{
		"guarantee_type": types.StringType,
		"resource_flow": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"cpu":    types.Int64Type,
				"memory": types.Int64Type,
			},
		},
		"burst_guarantee_resources": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"cpu":    types.Int64Type,
				"memory": types.Int64Type,
			},
		},
	})
	if schedulerPool.IntegralGuarantees != nil {
		integralGuarantees, d = flattenIntegralGuarantees(ctx, *schedulerPool.IntegralGuarantees)
		diags.Append(d...)
		if diags.HasError() {
			return SchedulerPoolModel{}, diags
		}
	}

	return SchedulerPoolModel{
		ID:                        types.StringValue(schedulerPool.ID),
		Name:                      types.StringValue(schedulerPool.Name),
		ParentName:                types.StringPointerValue(schedulerPool.ParentName),
		ACL:                       acl,
		MaxRunningOperationCount:  types.Int64PointerValue(schedulerPool.MaxRunningOperationCount),
		MaxOperationCount:         types.Int64PointerValue(schedulerPool.MaxOperationCount),
		IntegralGuarantees:        integralGuarantees,
		StrongGuaranteeResources:  strongGuaranteeResourcesObj,
		ResourceLimits:            resourceLimitsObj,
		Weight:                    types.Float64PointerValue(schedulerPool.Weight),
		Mode:                      types.StringPointerValue(schedulerPool.Mode),
		ForbidImmediateOperations: types.BoolPointerValue(schedulerPool.ForbidImmediateOperations),
	}, diags
}

// expandResources performs Terraform to YT conversion for SchedulerPoolResources.
// Direction: Terraform -> YT
func expandResources(ctx context.Context, obj types.Object) (*ytsaurus.SchedulerPoolResources, diag.Diagnostics) {
	var diags diag.Diagnostics

	if obj.IsNull() {
		return nil, diags
	}

	var resourcesModel SchedulerPoolResourcesModel
	d := obj.As(ctx, &resourcesModel, basetypes.ObjectAsOptions{})
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	resources := ytsaurus.SchedulerPoolResources{
		CPU:    resourcesModel.CPU.ValueInt64Pointer(),
		Memory: resourcesModel.Memory.ValueInt64Pointer(),
	}

	return &resources, diags
}

func expandResourcesV2(ctx context.Context, obj types.Object) (map[string]int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	if obj.IsNull() {
		return nil, diags
	}

	var model SchedulerPoolResourcesModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	result := map[string]int64{}

	if !model.CPU.IsNull() {
		result["cpu"] = model.CPU.ValueInt64()
	}
	if !model.Memory.IsNull() {
		result["memory"] = model.Memory.ValueInt64()
	}

	return result, diags
}

func expandIntegralGuaranteesV2(ctx context.Context, obj types.Object) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if obj.IsNull() {
		return nil, diags
	}

	var model SchedulerPoolIntegralGuaranteesModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	result := map[string]interface{}{}

	if !model.GuaranteeType.IsNull() {
		result["guarantee_type"] = model.GuaranteeType.ValueString()
	}

	if !model.ResourceFlow.IsNull() {
		resourceFlow, diags := expandResourcesV2(ctx, model.ResourceFlow)
		if diags.HasError() {
			return nil, diags
		}
		if resourceFlow != nil {
			result["resource_flow"] = resourceFlow
		}
	}

	if !model.BurstGuaranteeResources.IsNull() {
		burstGuaranteeResources, diags := expandResourcesV2(ctx, model.BurstGuaranteeResources)
		if diags.HasError() {
			return nil, diags
		}
		if burstGuaranteeResources != nil {
			result["burst_guarantee_resources"] = burstGuaranteeResources
		}
	}

	return result, diags
}

// expandIntegralGuarantees performs Terraform to YT conversion for SchedulerPoolResources.
// Direction: Terraform -> YT
func expandIntegralGuarantees(ctx context.Context, obj types.Object) (*ytsaurus.SchedulerPoolIntegralGuarantees, diag.Diagnostics) {
	var diags diag.Diagnostics

	if obj.IsNull() {
		return nil, diags
	}

	var integralGuaranteesModel SchedulerPoolIntegralGuaranteesModel
	d := obj.As(ctx, &integralGuaranteesModel, basetypes.ObjectAsOptions{})
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	resourceFlow, d := expandResources(ctx, integralGuaranteesModel.ResourceFlow)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	burstGuaranteeResources, d := expandResources(ctx, integralGuaranteesModel.BurstGuaranteeResources)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	integralGuarantees := ytsaurus.SchedulerPoolIntegralGuarantees{
		GuaranteeType:           integralGuaranteesModel.GuaranteeType.ValueStringPointer(),
		ResourceFlow:            resourceFlow,
		BurstGuaranteeResources: burstGuaranteeResources,
	}

	return &integralGuarantees, diags
}

// expandSchedulerPool performs Terraform to YT conversion for SchedulerPool resources.
// Direction: Terraform -> YT
func expandSchedulerPool(ctx context.Context, schedulerPoolModel SchedulerPoolModel) (ytsaurus.SchedulerPool, diag.Diagnostics) {
	var diags diag.Diagnostics

	acl, d := acl.ExpandACL(ctx, schedulerPoolModel.ACL)
	diags.Append(d...)
	if diags.HasError() {
		return ytsaurus.SchedulerPool{}, diags
	}

	strongGuaranteeResources, d := expandResources(ctx, schedulerPoolModel.StrongGuaranteeResources)
	diags.Append(d...)
	if diags.HasError() {
		return ytsaurus.SchedulerPool{}, diags
	}

	resourceLimits, d := expandResources(ctx, schedulerPoolModel.ResourceLimits)
	diags.Append(d...)
	if diags.HasError() {
		return ytsaurus.SchedulerPool{}, diags
	}

	integralGuarantees, d := expandIntegralGuarantees(ctx, schedulerPoolModel.IntegralGuarantees)
	diags.Append(d...)
	if diags.HasError() {
		return ytsaurus.SchedulerPool{}, diags
	}

	return ytsaurus.SchedulerPool{
		ID:                        schedulerPoolModel.ID.ValueString(),
		Name:                      schedulerPoolModel.Name.ValueString(),
		ACL:                       acl,
		ParentName:                schedulerPoolModel.ParentName.ValueStringPointer(),
		MaxRunningOperationCount:  schedulerPoolModel.MaxRunningOperationCount.ValueInt64Pointer(),
		MaxOperationCount:         schedulerPoolModel.MaxOperationCount.ValueInt64Pointer(),
		IntegralGuarantees:        integralGuarantees,
		StrongGuaranteeResources:  strongGuaranteeResources,
		ResourceLimits:            resourceLimits,
		Weight:                    schedulerPoolModel.Weight.ValueFloat64Pointer(),
		Mode:                      schedulerPoolModel.Mode.ValueStringPointer(),
		ForbidImmediateOperations: schedulerPoolModel.ForbidImmediateOperations.ValueBoolPointer(),
		Path:                      fmt.Sprintf("//sys/pool_trees/%s/%s", schedulerPoolModel.PoolTree.ValueString(), schedulerPoolModel.Name.ValueString()),
	}, diags
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
	resourceLimits.Description = "The resource_limits option describes limits for different resources in a given pool."

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
				Description: "ObjectID in the YTsaurus cluster, can be found in an object's @id attribute.",
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
				Description: "A name of the parent pool in the same pool_tree.",
			},
			"max_running_operation_count": schema.Int64Attribute{
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(minMaxRunningOperationCount),
				},
				Description: "Maximum number of operations in the running state.",
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

func (r *schedulerPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SchedulerPoolModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createOptions := &yt.CreateObjectOptions{
		Attributes: map[string]interface{}{
			"name":               plan.Name.ValueString(),
			"pool_tree":          plan.PoolTree.ValueString(),
			"terraform_resource": true,
		},
	}

	if !plan.ACL.IsNull() {
		ytACL, diags := acl.ExpandACL(ctx, plan.ACL)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createOptions.Attributes["acl"] = ytACL
	}
	if !plan.ParentName.IsNull() {
		createOptions.Attributes["parent_name"] = plan.ParentName.ValueString()
	}
	if !plan.MaxOperationCount.IsNull() {
		createOptions.Attributes["max_operation_count"] = plan.MaxOperationCount.ValueInt64()
	}
	if !plan.MaxRunningOperationCount.IsNull() {
		createOptions.Attributes["max_running_operation_count"] = plan.MaxRunningOperationCount.ValueInt64()
	}
	if !plan.Weight.IsNull() {
		createOptions.Attributes["weight"] = plan.Weight.ValueFloat64()
	}
	if !plan.Mode.IsNull() {
		createOptions.Attributes["mode"] = plan.Mode.ValueString()
	}
	if !plan.ForbidImmediateOperations.IsNull() {
		createOptions.Attributes["forbid_immediate_operations"] = plan.ForbidImmediateOperations.ValueBool()
	}

	resourceLimits, diags := expandResourcesV2(ctx, plan.ResourceLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if resourceLimits != nil {
		createOptions.Attributes["resource_limits"] = resourceLimits
	}

	strongGuaranteeResources, diags := expandResourcesV2(ctx, plan.StrongGuaranteeResources)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if strongGuaranteeResources != nil {
		createOptions.Attributes["strong_guarantee_resources"] = strongGuaranteeResources
	}

	integralGuarantees, diags := expandIntegralGuaranteesV2(ctx, plan.IntegralGuarantees)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if integralGuarantees != nil {
		createOptions.Attributes["integral_guarantees"] = integralGuarantees
	}

	id, err := r.client.CreateObject(ctx, yt.NodeSchedulerPool, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating scheduler_pool",
			fmt.Sprintf(
				"Could not create scheduler_pool %q, unexpected error: %q",
				plan.Name.ValueString(),
				err.Error(),
			),
		)
		return
	}

	plan.ID = types.StringValue(id.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *schedulerPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var objectID string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &objectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ytSchedulerPool ytsaurus.SchedulerPool
	exists, err := ytsaurus.ObjectExistsByID(ctx, r.client, objectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading scheduler_pool",
			fmt.Sprintf(
				"Could not check scheduler_pool by id %q, unexpected error: %q",
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

	// pools without ParentName store yt cypress id to pool_tree object in @parent_name attribute, like #x-xxxx-xxxx-xxxxxx
	if ytSchedulerPool.ParentName != nil {
		if strings.HasPrefix(*ytSchedulerPool.ParentName, "#") {
			ytSchedulerPool.ParentName = nil
		}
	}

	state, d := flattenSchedulerPool(ctx, ytSchedulerPool)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	parts := strings.Split(ytSchedulerPool.Path, "pool_trees/")
	if len(parts) == 2 {
		state.PoolTree = types.StringValue(strings.Split(parts[1], "/")[0])
	}

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

	ytSchedulerPoolPlan, d := expandSchedulerPool(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytSchedulerPoolState, d := expandSchedulerPool(ctx, state)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	attributeUpdates := map[string]interface{}{
		"name": ytSchedulerPoolPlan.Name,
	}

	if !plan.ACL.Equal(state.ACL) {
		attributeUpdates["acl"] = ytSchedulerPoolPlan.ACL
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
	state, d = flattenSchedulerPool(ctx, ytSchedulerPoolPlan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.PoolTree = plan.PoolTree

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *schedulerPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SchedulerPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytSchedulerPool, d := expandSchedulerPool(ctx, state)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

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
