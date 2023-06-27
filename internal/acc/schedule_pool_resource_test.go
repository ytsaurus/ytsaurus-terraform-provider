package acc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/resource/acl"
	"terraform-provider-ytsaurus/internal/resource/schedulerpool"
)

func TestSchedulerPoolResourceMisconfigurations(t *testing.T) {
	resourceID := "fakepool"
	testSchedulerPoolName := resourceID
	testSchedulerPoolYTCypressPath := fmt.Sprintf("//sys/pool_trees/default/%s", testSchedulerPoolName)
	testPoolTree := "default"
	testMaxOperationCount := int64(10)
	testMaxRunningOperationCount := int64(10)

	configWithoutPoolTree := schedulerpool.SchedulerPoolModel{
		Name: types.StringValue(testSchedulerPoolName),
	}

	configMaxRunningOperationCountShouldBePositive := schedulerpool.SchedulerPoolModel{
		Name:                     types.StringValue(testSchedulerPoolName),
		PoolTree:                 types.StringValue(testPoolTree),
		MaxRunningOperationCount: types.Int64Value(0),
	}

	configMaxOperationCountShouldBePositive := schedulerpool.SchedulerPoolModel{
		Name:              types.StringValue(testSchedulerPoolName),
		PoolTree:          types.StringValue(testPoolTree),
		MaxOperationCount: types.Int64Value(0),
	}

	configParentNameNotEmpty := schedulerpool.SchedulerPoolModel{
		Name:       types.StringValue(testSchedulerPoolName),
		PoolTree:   types.StringValue(testPoolTree),
		ParentName: types.StringValue(`""`),
	}

	configWeightShouldBePositive := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
		Weight:   types.Float64Value(0),
	}

	configModeOneOf := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
		Mode:     types.StringValue("fake"),
	}

	configMaxOperationCountGreaterThenMaxRunningOperationCount := schedulerpool.SchedulerPoolModel{
		Name:                     types.StringValue(testSchedulerPoolName),
		PoolTree:                 types.StringValue(testPoolTree),
		MaxRunningOperationCount: types.Int64Value(testMaxRunningOperationCount),
		MaxOperationCount:        types.Int64Value(testMaxOperationCount - 1),
	}

	configResourceLimitsNotEmpty := schedulerpool.SchedulerPoolModel{
		Name:           types.StringValue(testSchedulerPoolName),
		PoolTree:       types.StringValue(testPoolTree),
		ResourceLimits: &schedulerpool.SchedulerPoolResourcesModel{},
	}

	configStrongGuaranteeResourcesNotEmpty := schedulerpool.SchedulerPoolModel{
		Name:                     types.StringValue(testSchedulerPoolName),
		PoolTree:                 types.StringValue(testPoolTree),
		StrongGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{},
	}

	configIntegralGuaranteesGuaranteeTypeOneOf := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
		IntegralGuarantees: &schedulerpool.SchedulerPoolIntegralGuaranteesModel{
			GuaranteeType: types.StringValue("fake"),
		},
	}

	configIntegralGuaranteesNotEmpty := schedulerpool.SchedulerPoolModel{
		Name:               types.StringValue(testSchedulerPoolName),
		PoolTree:           types.StringValue(testPoolTree),
		IntegralGuarantees: &schedulerpool.SchedulerPoolIntegralGuaranteesModel{},
	}

	configIntegralGuaranteesResourceFlowNotEmpty := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
		IntegralGuarantees: &schedulerpool.SchedulerPoolIntegralGuaranteesModel{
			ResourceFlow: &schedulerpool.SchedulerPoolResourcesModel{},
		},
	}

	configIntegralGuaranteesBurstGuaranteeResourcesNotEmpty := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
		IntegralGuarantees: &schedulerpool.SchedulerPoolIntegralGuaranteesModel{
			BurstGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testSchedulerPoolYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configWithoutPoolTree),
				ExpectError: regexp.MustCompile("The argument \"pool_tree\" is required, but no definition was found."),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configMaxRunningOperationCountShouldBePositive),
				ExpectError: regexp.MustCompile("Attribute max_running_operation_count value must be at least 1, got: 0"),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configMaxOperationCountShouldBePositive),
				ExpectError: regexp.MustCompile("Attribute max_operation_count value must be at least 1, got: 0"),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configParentNameNotEmpty),
				ExpectError: regexp.MustCompile("Attribute parent_name string length must be at least 1, got: 0"),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configWeightShouldBePositive),
				ExpectError: regexp.MustCompile("Attribute weight value must be at least 1.000000, got: 0.000000"),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configModeOneOf),
				ExpectError: regexp.MustCompile(`Attribute mode value must be one of: \["\\"fair_share\\"" "\\"fifo\\""\], got:`),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configMaxOperationCountGreaterThenMaxRunningOperationCount),
				ExpectError: regexp.MustCompile("\"max_operation_count\" must be greater that or equal to\n\"max_running_operation_count\", but 9 < 10"),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configResourceLimitsNotEmpty),
				ExpectError: regexp.MustCompile("Attribute \"resource_limits.(cpu|memory)\" must be specified when \"resource_limits\""),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configStrongGuaranteeResourcesNotEmpty),
				ExpectError: regexp.MustCompile("Attribute \"strong_guarantee_resources.(cpu|memory)\" must be specified when\n? ?\"strong_guarantee_resources\""),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configIntegralGuaranteesGuaranteeTypeOneOf),
				ExpectError: regexp.MustCompile(`Attribute integral_guarantees.guarantee_type value must be one of:\n\["\\"burst\\"" "\\"relaxed\\"" "\\"none\\""\]`),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configIntegralGuaranteesNotEmpty),
				ExpectError: regexp.MustCompile("Attribute \"integral_guarantees.(cpu|memory|resource_flow)\" must be specified\n? ?when\n? ?\"integral_guarantees\""),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configIntegralGuaranteesResourceFlowNotEmpty),
				ExpectError: regexp.MustCompile("Attribute \"integral_guarantees.resource_flow.(cpu|memory)\" must be specified when\n\"integral_guarantees.resource_flow\" is specified"),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configIntegralGuaranteesBurstGuaranteeResourcesNotEmpty),
				ExpectError: regexp.MustCompile("Attribute \"integral_guarantees.burst_guarantee_resources.(cpu|memory)\" must be\nspecified when \"integral_guarantees.burst_guarantee_resources\" is specified"),
			},
		},
	})
}

func TestSchedulerPoolResourceCreateAndUpdate(t *testing.T) {
	resourceID := "fakepool"
	testSchedulerPoolName := resourceID
	testSchedulerPoolYTCypressPath := fmt.Sprintf("//sys/pool_trees/default/%s", testSchedulerPoolName)
	testPoolTree := "default"
	testForbidImmediateOperations := true
	testMaxOperationCount := int64(10)
	testMaxRunningOperationCount := int64(10)
	testSchedulerPoolResourcesModelCPU := int64(1)
	testSchedulerPoolResourcesModelMemory := int64(1 * 1024 * 1024)
	testGuaranteeTypeBurst := "burst"
	testGuaranteeTypeRelaxed := "relaxed"

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionUse},
			InheritanceMode: "object_and_descendants",
		},
	}

	configCreateWithMinimalOptions := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
	}

	configUpdateAllAttributes := schedulerpool.SchedulerPoolModel{
		Name:                      types.StringValue(testSchedulerPoolName),
		PoolTree:                  types.StringValue(testPoolTree),
		ACL:                       acl.ToACLModel(testACL),
		MaxRunningOperationCount:  types.Int64Value(testMaxRunningOperationCount),
		MaxOperationCount:         types.Int64Value(testMaxOperationCount),
		ForbidImmediateOperations: types.BoolValue(testForbidImmediateOperations),
		ResourceLimits: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		StrongGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
	}

	configIntegralGuaranteesBurst := schedulerpool.SchedulerPoolModel{
		Name:                      types.StringValue(testSchedulerPoolName),
		PoolTree:                  types.StringValue(testPoolTree),
		ACL:                       acl.ToACLModel(testACL),
		MaxRunningOperationCount:  types.Int64Value(testMaxRunningOperationCount),
		MaxOperationCount:         types.Int64Value(testMaxOperationCount),
		ForbidImmediateOperations: types.BoolValue(testForbidImmediateOperations),
		ResourceLimits: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		StrongGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		IntegralGuarantees: &schedulerpool.SchedulerPoolIntegralGuaranteesModel{
			GuaranteeType: types.StringValue(testGuaranteeTypeBurst),
			BurstGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{
				CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
				Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
			},
		},
	}

	configIntegralGuaranteesRelaxed := schedulerpool.SchedulerPoolModel{
		Name:                      types.StringValue(testSchedulerPoolName),
		PoolTree:                  types.StringValue(testPoolTree),
		ACL:                       acl.ToACLModel(testACL),
		MaxRunningOperationCount:  types.Int64Value(testMaxRunningOperationCount),
		MaxOperationCount:         types.Int64Value(testMaxOperationCount),
		ForbidImmediateOperations: types.BoolValue(testForbidImmediateOperations),
		ResourceLimits: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		StrongGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		IntegralGuarantees: &schedulerpool.SchedulerPoolIntegralGuaranteesModel{
			GuaranteeType: types.StringValue(testGuaranteeTypeRelaxed),
			ResourceFlow: &schedulerpool.SchedulerPoolResourcesModel{
				CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
				Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testSchedulerPoolYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configCreateWithMinimalOptions),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "name", testSchedulerPoolName),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configUpdateAllAttributes),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "name", testSchedulerPoolName),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_operation_count", testMaxOperationCount),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_running_operation_count", testMaxRunningOperationCount),
					accCheckYTsaurusACLAttribute(testSchedulerPoolYTCypressPath, testACL),
					accCheckYTsaurusBoolAttribute(testSchedulerPoolYTCypressPath, "forbid_immediate_operations", testForbidImmediateOperations),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/memory", testSchedulerPoolResourcesModelMemory),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configIntegralGuaranteesBurst),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "name", testSchedulerPoolName),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_operation_count", testMaxOperationCount),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_running_operation_count", testMaxRunningOperationCount),
					accCheckYTsaurusACLAttribute(testSchedulerPoolYTCypressPath, testACL),
					accCheckYTsaurusBoolAttribute(testSchedulerPoolYTCypressPath, "forbid_immediate_operations", testForbidImmediateOperations),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "integral_guarantees/guarantee_type", testGuaranteeTypeBurst),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "integral_guarantees/burst_guarantee_resources/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "integral_guarantees/burst_guarantee_resources/memory", testSchedulerPoolResourcesModelMemory),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configIntegralGuaranteesRelaxed),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "name", testSchedulerPoolName),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_operation_count", testMaxOperationCount),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_running_operation_count", testMaxRunningOperationCount),
					accCheckYTsaurusACLAttribute(testSchedulerPoolYTCypressPath, testACL),
					accCheckYTsaurusBoolAttribute(testSchedulerPoolYTCypressPath, "forbid_immediate_operations", testForbidImmediateOperations),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "integral_guarantees/guarantee_type", testGuaranteeTypeRelaxed),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "integral_guarantees/resource_flow/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "integral_guarantees/resource_flow/memory", testSchedulerPoolResourcesModelMemory),
				),
			},
		},
	})
}

func TestSchedulerPoolResourceCreateWithAllAttributesIntegralGuaranteesBurst(t *testing.T) {
	resourceID := "fakepool"
	testSchedulerPoolName := resourceID
	testSchedulerPoolYTCypressPath := fmt.Sprintf("//sys/pool_trees/default/%s", testSchedulerPoolName)
	testPoolTree := "default"
	testForbidImmediateOperations := true
	testMaxOperationCount := int64(10)
	testMaxRunningOperationCount := int64(10)
	testSchedulerPoolResourcesModelCPU := int64(1)
	testSchedulerPoolResourcesModelMemory := int64(1 * 1024 * 1024)
	testGuaranteeTypeBurst := "burst"

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionUse},
			InheritanceMode: "object_and_descendants",
		},
	}

	configIntegralGuaranteesBurst := schedulerpool.SchedulerPoolModel{
		Name:                      types.StringValue(testSchedulerPoolName),
		PoolTree:                  types.StringValue(testPoolTree),
		ACL:                       acl.ToACLModel(testACL),
		MaxRunningOperationCount:  types.Int64Value(testMaxRunningOperationCount),
		MaxOperationCount:         types.Int64Value(testMaxOperationCount),
		ForbidImmediateOperations: types.BoolValue(testForbidImmediateOperations),
		ResourceLimits: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		StrongGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		IntegralGuarantees: &schedulerpool.SchedulerPoolIntegralGuaranteesModel{
			GuaranteeType: types.StringValue(testGuaranteeTypeBurst),
			BurstGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{
				CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
				Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testSchedulerPoolYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configIntegralGuaranteesBurst),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "name", testSchedulerPoolName),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_operation_count", testMaxOperationCount),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_running_operation_count", testMaxRunningOperationCount),
					accCheckYTsaurusACLAttribute(testSchedulerPoolYTCypressPath, testACL),
					accCheckYTsaurusBoolAttribute(testSchedulerPoolYTCypressPath, "forbid_immediate_operations", testForbidImmediateOperations),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "integral_guarantees/guarantee_type", testGuaranteeTypeBurst),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "integral_guarantees/burst_guarantee_resources/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "integral_guarantees/burst_guarantee_resources/memory", testSchedulerPoolResourcesModelMemory),
				),
			},
		},
	})
}

func TestSchedulerPoolResourceCreateWithAllAttributesIntegralGuaranteesRelaxed(t *testing.T) {
	resourceID := "fakepool"
	testSchedulerPoolName := resourceID
	testSchedulerPoolYTCypressPath := fmt.Sprintf("//sys/pool_trees/default/%s", testSchedulerPoolName)
	testPoolTree := "default"
	testForbidImmediateOperations := true
	testMaxOperationCount := int64(10)
	testMaxRunningOperationCount := int64(10)
	testSchedulerPoolResourcesModelCPU := int64(1)
	testSchedulerPoolResourcesModelMemory := int64(1 * 1024 * 1024)
	testGuaranteeTypeRelaxed := "relaxed"

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionUse},
			InheritanceMode: "object_and_descendants",
		},
	}

	configIntegralGuaranteesRelaxed := schedulerpool.SchedulerPoolModel{
		Name:                      types.StringValue(testSchedulerPoolName),
		PoolTree:                  types.StringValue(testPoolTree),
		ACL:                       acl.ToACLModel(testACL),
		MaxRunningOperationCount:  types.Int64Value(testMaxRunningOperationCount),
		MaxOperationCount:         types.Int64Value(testMaxOperationCount),
		ForbidImmediateOperations: types.BoolValue(testForbidImmediateOperations),
		ResourceLimits: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		StrongGuaranteeResources: &schedulerpool.SchedulerPoolResourcesModel{
			CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
			Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
		},
		IntegralGuarantees: &schedulerpool.SchedulerPoolIntegralGuaranteesModel{
			GuaranteeType: types.StringValue(testGuaranteeTypeRelaxed),
			ResourceFlow: &schedulerpool.SchedulerPoolResourcesModel{
				CPU:    types.Int64Value(testSchedulerPoolResourcesModelCPU),
				Memory: types.Int64Value(testSchedulerPoolResourcesModelMemory),
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testSchedulerPoolYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configIntegralGuaranteesRelaxed),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "name", testSchedulerPoolName),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_operation_count", testMaxOperationCount),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "max_running_operation_count", testMaxRunningOperationCount),
					accCheckYTsaurusACLAttribute(testSchedulerPoolYTCypressPath, testACL),
					accCheckYTsaurusBoolAttribute(testSchedulerPoolYTCypressPath, "forbid_immediate_operations", testForbidImmediateOperations),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "resource_limits/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "strong_guarantee_resources/memory", testSchedulerPoolResourcesModelMemory),
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "integral_guarantees/guarantee_type", testGuaranteeTypeRelaxed),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "integral_guarantees/resource_flow/cpu", testSchedulerPoolResourcesModelCPU),
					accCheckYTsaurusInt64Attribute(testSchedulerPoolYTCypressPath, "integral_guarantees/resource_flow/memory", testSchedulerPoolResourcesModelMemory),
				),
			},
		},
	})
}

func TestSchedulerPoolResourceCreateParentChild(t *testing.T) {
	resourceParentID := "fakepool_parent"
	resourceChildID := "fakepool_child"

	testParentSchedulerPoolName := resourceParentID
	testChildSchedulerPoolName := resourceChildID

	testSchedulerPoolParentYTCypressPath := fmt.Sprintf("//sys/pool_trees/default/%s", testParentSchedulerPoolName)
	testSchedulerPoolChildYTCypressPath := fmt.Sprintf("//sys/pool_trees/default/%s/%s", testParentSchedulerPoolName, testChildSchedulerPoolName)
	testSchedulerPoolRemovedParentYTCypressPath := fmt.Sprintf("//sys/pool_trees/default/%s", testChildSchedulerPoolName)
	testPoolTree := "default"
	testParentName := fmt.Sprintf("ytsaurus_scheduler_pool.%s.name", resourceParentID)

	configParent := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testParentSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
	}

	configChild := schedulerpool.SchedulerPoolModel{
		Name:       types.StringValue(testChildSchedulerPoolName),
		PoolTree:   types.StringValue(testPoolTree),
		ParentName: types.StringValue(testParentName),
	}

	configRemoveParent := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testChildSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testSchedulerPoolParentYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceParentID, configParent) + accResourceYtsaurusSchedulerPoolConfig(resourceChildID, configChild),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolParentYTCypressPath, "name", testParentSchedulerPoolName),
					accCheckYTsaurusStringAttribute(testSchedulerPoolChildYTCypressPath, "name", testChildSchedulerPoolName),
					accCheckYTsaurusStringAttribute(testSchedulerPoolChildYTCypressPath, "parent_name", testParentSchedulerPoolName),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceParentID, configParent) + accResourceYtsaurusSchedulerPoolConfig(resourceChildID, configRemoveParent),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolRemovedParentYTCypressPath, "name", testChildSchedulerPoolName),
				),
			},
		},
	})

}

func TestSchedulerPoolResourceCreateAndRename(t *testing.T) {
	resourceID := "fakepool"

	testSchedulerPoolName := resourceID
	testSchedulerPoolNameRenamed := fmt.Sprintf("%s_renamed", resourceID)
	testSchedulerPoolYTCypressPath := fmt.Sprintf("//sys/pool_trees/default/%s", testSchedulerPoolName)
	testSchedulerPoolYTCypressRenamed := fmt.Sprintf("//sys/pool_trees/default/%s", testSchedulerPoolNameRenamed)
	testPoolTree := "default"

	configCreate := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testSchedulerPoolName),
		PoolTree: types.StringValue(testPoolTree),
	}

	configRenamed := schedulerpool.SchedulerPoolModel{
		Name:     types.StringValue(testSchedulerPoolNameRenamed),
		PoolTree: types.StringValue(testPoolTree),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testSchedulerPoolYTCypressRenamed),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressPath, "name", testSchedulerPoolName),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusSchedulerPoolConfig(resourceID, configRenamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testSchedulerPoolYTCypressRenamed, "name", testSchedulerPoolNameRenamed),
				),
			},
		},
	})
}

func accResourceYtsaurusSchedulerPoolConfig(id string, m schedulerpool.SchedulerPoolModel) string {

	config := fmt.Sprintf(`
	resource "ytsaurus_scheduler_pool" %q {
		name = %q`, id, m.Name.ValueString())

	if len(m.PoolTree.ValueString()) > 0 {
		config += fmt.Sprintf(`
		pool_tree = %q`, m.PoolTree.ValueString())
	}

	if !m.ParentName.IsNull() {
		config += fmt.Sprintf(`
		parent_name = %s`, m.ParentName.ValueString())
	}

	if !m.Weight.IsNull() {
		config += fmt.Sprintf(`
		weight = %f`, m.Weight.ValueFloat64())
	}

	if !m.Mode.IsNull() {
		config += fmt.Sprintf(`
		mode = %q`, m.Mode.ValueString())
	}

	if !m.MaxOperationCount.IsNull() {
		config += fmt.Sprintf(`
		max_operation_count = %d`, m.MaxOperationCount.ValueInt64())
	}

	if !m.MaxRunningOperationCount.IsNull() {
		config += fmt.Sprintf(`
		max_running_operation_count = %d`, m.MaxRunningOperationCount.ValueInt64())
	}

	if !m.ForbidImmediateOperations.IsNull() {
		config += fmt.Sprintf(`
		forbid_immediate_operations = %t`, m.ForbidImmediateOperations.ValueBool())
	}

	if m.ResourceLimits != nil {
		config += `
		resource_limits = {`

		if !m.ResourceLimits.CPU.IsNull() {
			config += fmt.Sprintf(`
			cpu = %d`, m.ResourceLimits.CPU.ValueInt64())
		}

		if !m.ResourceLimits.Memory.IsNull() {
			config += fmt.Sprintf(`
			memory = %d`, m.ResourceLimits.Memory.ValueInt64())
		}

		config += `
		}`
	}

	if m.StrongGuaranteeResources != nil {
		config += `
		strong_guarantee_resources = {`

		if !m.StrongGuaranteeResources.CPU.IsNull() {
			config += fmt.Sprintf(`
			cpu = %d`, m.StrongGuaranteeResources.CPU.ValueInt64())
		}

		if !m.StrongGuaranteeResources.Memory.IsNull() {
			config += fmt.Sprintf(`
			memory = %d`, m.StrongGuaranteeResources.Memory.ValueInt64())
		}

		config += `
		}`
	}

	if m.IntegralGuarantees != nil {
		config += `
		integral_guarantees = {`

		if !m.IntegralGuarantees.GuaranteeType.IsNull() {
			config += fmt.Sprintf(`
			guarantee_type = %q`, m.IntegralGuarantees.GuaranteeType.ValueString())
		}

		if m.IntegralGuarantees.ResourceFlow != nil {
			config += `
			resource_flow = {`

			if !m.IntegralGuarantees.ResourceFlow.CPU.IsNull() {
				config += fmt.Sprintf(`
				cpu = %d`, m.IntegralGuarantees.ResourceFlow.CPU.ValueInt64())
			}

			if !m.IntegralGuarantees.ResourceFlow.Memory.IsNull() {
				config += fmt.Sprintf(`
				memory = %d`, m.IntegralGuarantees.ResourceFlow.Memory.ValueInt64())
			}

			config += `
			}`
		}

		if m.IntegralGuarantees.BurstGuaranteeResources != nil {
			config += `
			burst_guarantee_resources = {`

			if !m.IntegralGuarantees.BurstGuaranteeResources.CPU.IsNull() {
				config += fmt.Sprintf(`
				cpu = %d`, m.IntegralGuarantees.BurstGuaranteeResources.CPU.ValueInt64())
			}

			if !m.IntegralGuarantees.BurstGuaranteeResources.Memory.IsNull() {
				config += fmt.Sprintf(`
				memory = %d`, m.IntegralGuarantees.BurstGuaranteeResources.Memory.ValueInt64())
			}

			config += `
			}`
		}

		config += `
		}`
	}

	acl := acl.ToYTsaurusACL(m.ACL)
	if len(acl) > 0 {
		config += accAddACLConfig(acl)
	}

	config += `
	}`

	return config
}
