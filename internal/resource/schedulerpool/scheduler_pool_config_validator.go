package schedulerpool

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type schedulerPoolResourceConfigValidator struct{}

var _ resource.ConfigValidator = &schedulerPoolResourceConfigValidator{}

func (v schedulerPoolResourceConfigValidator) Description(_ context.Context) string {
	return ""
}

func (v schedulerPoolResourceConfigValidator) MarkdownDescription(_ context.Context) string {
	return ""
}

func (v schedulerPoolResourceConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config SchedulerPoolModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ytSchedulerPool, diags := toYTsaurusSchedulerPool(config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if ytSchedulerPool.MaxRunningOperationCount != nil &&
		ytSchedulerPool.MaxOperationCount != nil &&
		*ytSchedulerPool.MaxRunningOperationCount > *ytSchedulerPool.MaxOperationCount {
		resp.Diagnostics.AddError(
			"Scheduler pool configuration error",
			fmt.Sprintf(
				"%q must be greater that or equal to %q, but %d < %d",
				"max_operation_count",
				"max_running_operation_count",
				*ytSchedulerPool.MaxOperationCount,
				*ytSchedulerPool.MaxRunningOperationCount,
			),
		)
		return
	}
}
