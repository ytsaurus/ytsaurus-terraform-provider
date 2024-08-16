package schedulerpool

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
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
	var MaxRunningOperationCount, MaxOperationCount *int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("max_running_operation_count"), &MaxRunningOperationCount)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("max_operation_count"), &MaxOperationCount)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if MaxRunningOperationCount != nil &&
		MaxOperationCount != nil &&
		*MaxRunningOperationCount > *MaxOperationCount {
		resp.Diagnostics.AddError(
			"Scheduler pool configuration error",
			fmt.Sprintf(
				"%q must be greater that or equal to %q, but %d < %d",
				"max_operation_count",
				"max_running_operation_count",
				*MaxOperationCount,
				*MaxRunningOperationCount,
			),
		)
		return
	}
}
