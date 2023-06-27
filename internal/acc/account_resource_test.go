package acc

import (
	"fmt"
	"regexp"

	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/resource/account"
	"terraform-provider-ytsaurus/internal/resource/acl"
)

func TestAccountResourceCreateAndUpdate(t *testing.T) {

	resourceID := "testaccount"
	testAccountName := resourceID
	testAccountNameRenamed := "testaccount_renamed"
	testAccountYTCypressPath := fmt.Sprintf("//sys/accounts/%s", testAccountName)
	testAccountYTCypressPathRenamed := fmt.Sprintf("//sys/accounts/%s", testAccountNameRenamed)

	testNodeCount := int64(1000)
	testChunkCount := int64(1000)
	testTabletCount := int64(10)
	testTabletStaticMemory := int64(10000)
	testDefaultMedium := "default"
	testDefaultMediumSize := int64(1000000)

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionUse},
			InheritanceMode: "object_and_descendants",
		},
	}

	configEmpty := account.AccountModel{}

	configWithoutResourceLimits := account.AccountModel{
		Name: types.StringValue(testAccountName),
	}

	configWithEmptyResourceLimits := account.AccountModel{
		Name:           types.StringValue(testAccountName),
		ResourceLimits: &account.AccountResourceLimitsModel{},
	}

	configCreate := account.AccountModel{
		Name: types.StringValue(testAccountName),
		ResourceLimits: &account.AccountResourceLimitsModel{
			ChunkCount: types.Int64Value(testChunkCount),
			NodeCount:  types.Int64Value(testNodeCount),
			DiskSpacePerMedium: map[string]basetypes.Int64Value{
				testDefaultMedium: types.Int64Value(testDefaultMediumSize),
			},
		},
	}

	configRename := account.AccountModel{
		Name: types.StringValue(testAccountNameRenamed),
		ResourceLimits: &account.AccountResourceLimitsModel{
			ChunkCount: types.Int64Value(testChunkCount),
			NodeCount:  types.Int64Value(testNodeCount),
			DiskSpacePerMedium: map[string]basetypes.Int64Value{
				testDefaultMedium: types.Int64Value(testDefaultMediumSize),
			},
		},
	}

	configUpdate := account.AccountModel{
		Name: types.StringValue(testAccountName),
		ResourceLimits: &account.AccountResourceLimitsModel{
			ChunkCount: types.Int64Value(testChunkCount + 1),
			NodeCount:  types.Int64Value(testNodeCount + 1),
			DiskSpacePerMedium: map[string]basetypes.Int64Value{
				testDefaultMedium: types.Int64Value(testDefaultMediumSize + 1),
			},
			TabletCount:        types.Int64Value(testTabletCount),
			TabletStaticMemory: types.Int64Value(testTabletStaticMemory),
		},
		ACL: acl.ToACLModel(testACL),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testAccountYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceID, configEmpty),
				ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceID, configWithoutResourceLimits),
				ExpectError: regexp.MustCompile(`The argument "resource_limits" is required, but no definition was found.`),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceID, configWithEmptyResourceLimits),
				ExpectError: regexp.MustCompile(`Inappropriate value for attribute "resource_limits": attributes\n"chunk_count", "disk_space_per_medium", and "node_count" are required.`),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceID, configCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testAccountYTCypressPath, "name", testAccountName),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/node_count", testNodeCount),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/chunk_count", testChunkCount),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/disk_space_per_medium/default", testDefaultMediumSize),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceID, configRename),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testAccountYTCypressPathRenamed, "name", testAccountNameRenamed),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceID, configUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testAccountYTCypressPath, "name", testAccountName),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/node_count", testNodeCount+1),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/chunk_count", testChunkCount+1),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/disk_space_per_medium/default", testDefaultMediumSize+1),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/tablet_count", testTabletCount),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/tablet_static_memory", testTabletStaticMemory),
					accCheckYTsaurusACLAttribute(testAccountYTCypressPath, testACL),
				),
			},
		},
	})
}

func TestAccountResourceCreateWithAllOptions(t *testing.T) {

	resourceID := "testaccount"
	testAccountName := resourceID
	testAccountYTCypressPath := fmt.Sprintf("//sys/accounts/%s", testAccountName)

	testNodeCount := int64(1000)
	testChunkCount := int64(1000)
	testTabletCount := int64(10)
	testTabletStaticMemory := int64(10000)
	testDefaultMedium := "default"
	testDefaultMediumSize := int64(1000000)

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionUse},
			InheritanceMode: "object_and_descendants",
		},
	}

	configCreate := account.AccountModel{
		Name: types.StringValue(testAccountName),
		ResourceLimits: &account.AccountResourceLimitsModel{
			ChunkCount: types.Int64Value(testChunkCount),
			NodeCount:  types.Int64Value(testNodeCount),
			DiskSpacePerMedium: map[string]basetypes.Int64Value{
				testDefaultMedium: types.Int64Value(testDefaultMediumSize),
			},
			TabletCount:        types.Int64Value(testTabletCount),
			TabletStaticMemory: types.Int64Value(testTabletStaticMemory),
		},
		ACL: acl.ToACLModel(testACL),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testAccountYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceID, configCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testAccountYTCypressPath, "name", testAccountName),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/node_count", testNodeCount),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/chunk_count", testChunkCount),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/disk_space_per_medium/default", testDefaultMediumSize),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/tablet_count", testTabletCount),
					accCheckYTsaurusInt64Attribute(testAccountYTCypressPath, "resource_limits/tablet_static_memory", testTabletStaticMemory),
					accCheckYTsaurusACLAttribute(testAccountYTCypressPath, testACL),
				),
			},
		},
	})
}

func TestAccountResourceCreateParentChild(t *testing.T) {

	resourceChildID := "testaccount"
	resourceParentID := "testaccount_father"

	testAccountChildName := resourceChildID
	testAccountParentName := resourceParentID

	testAccountChildYTCypressPath := fmt.Sprintf("//sys/accounts/%s", testAccountChildName)
	testAccountParentYTCypressPath := fmt.Sprintf("//sys/accounts/%s", testAccountParentName)

	testNodeCount := int64(1000)
	testChunkCount := int64(1000)
	testDefaultMedium := "default"
	testDefaultMediumSize := int64(1000000)

	configParent := account.AccountModel{
		Name: types.StringValue(testAccountParentName),
		ResourceLimits: &account.AccountResourceLimitsModel{
			ChunkCount: types.Int64Value(testChunkCount),
			NodeCount:  types.Int64Value(testNodeCount),
			DiskSpacePerMedium: map[string]basetypes.Int64Value{
				testDefaultMedium: types.Int64Value(testDefaultMediumSize),
			},
		},
	}

	configChild := account.AccountModel{
		Name:       types.StringValue(testAccountChildName),
		ParentName: types.StringValue(fmt.Sprintf("ytsaurus_account.%s.name", resourceParentID)),
		ResourceLimits: &account.AccountResourceLimitsModel{
			ChunkCount: types.Int64Value(testChunkCount),
			NodeCount:  types.Int64Value(testNodeCount),
			DiskSpacePerMedium: map[string]basetypes.Int64Value{
				testDefaultMedium: types.Int64Value(testDefaultMediumSize),
			},
		},
	}

	configRemoveParentName := account.AccountModel{
		Name: types.StringValue(testAccountChildName),
		ResourceLimits: &account.AccountResourceLimitsModel{
			ChunkCount: types.Int64Value(testChunkCount),
			NodeCount:  types.Int64Value(testNodeCount),
			DiskSpacePerMedium: map[string]basetypes.Int64Value{
				testDefaultMedium: types.Int64Value(testDefaultMediumSize),
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testAccountParentYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceParentID, configParent) + accResourceYtsaurusAccountConfig(resourceChildID, configChild),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testAccountParentYTCypressPath, "name", testAccountParentName),
					accCheckYTsaurusStringAttribute(testAccountChildYTCypressPath, "name", testAccountChildName),
					accCheckYTsaurusStringAttribute(testAccountChildYTCypressPath, "parent_name", testAccountParentName),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusAccountConfig(resourceParentID, configParent) + accResourceYtsaurusAccountConfig(resourceChildID, configRemoveParentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testAccountParentYTCypressPath, "name", testAccountParentName),
					accCheckYTsaurusStringAttribute(testAccountChildYTCypressPath, "name", testAccountChildName),
					accCheckYTsaurusStringAttribute(testAccountChildYTCypressPath, "parent_name", "root"),
				),
			},
		},
	})
}

func accResourceYtsaurusAccountConfig(id string, m account.AccountModel) string {

	config := fmt.Sprintf(`
	resource "ytsaurus_account" %q {`, id)

	if !m.Name.IsNull() {
		config += fmt.Sprintf(`
		name = %q`, m.Name.ValueString())
	}

	if !m.ParentName.IsNull() {
		config += fmt.Sprintf(`
		parent_name = %s`, m.ParentName.ValueString())
	}

	if m.ResourceLimits != nil {
		config += `
		resource_limits = {`

		if !m.ResourceLimits.NodeCount.IsNull() {
			config += fmt.Sprintf(`
			node_count = %d`, m.ResourceLimits.NodeCount.ValueInt64())
		}

		if !m.ResourceLimits.ChunkCount.IsNull() {
			config += fmt.Sprintf(`
			chunk_count = %d`, m.ResourceLimits.ChunkCount.ValueInt64())
		}

		if !m.ResourceLimits.TabletCount.IsNull() {
			config += fmt.Sprintf(`
			tablet_count = %d`, m.ResourceLimits.TabletCount.ValueInt64())
		}

		if !m.ResourceLimits.TabletStaticMemory.IsNull() {
			config += fmt.Sprintf(`
			tablet_static_memory = %d`, m.ResourceLimits.TabletStaticMemory.ValueInt64())
		}

		if len(m.ResourceLimits.DiskSpacePerMedium) > 0 {
			config += `
			disk_space_per_medium = {`

			for k, v := range m.ResourceLimits.DiskSpacePerMedium {
				config += fmt.Sprintf(`
				%q = %d`, k, v.ValueInt64())
			}

			config += `
			}`
		}

		config += `
		}`
	}

	config += accAddACLConfig(acl.ToYTsaurusACL(m.ACL))

	config += `
	}`

	return config
}
