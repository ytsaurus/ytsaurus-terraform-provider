package acc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/resource/acl"
	"terraform-provider-ytsaurus/internal/resource/tabletcellbundle"
)

func TestTabletCellBundleResourceMisconfigurations(t *testing.T) {
	resourceID := "fakebundle"
	testTabletCellBundleName := resourceID
	testtestTabletCellBundleYTCypressPath := fmt.Sprintf("//sys/tablet_cell_bundles/%s", testTabletCellBundleName)
	testChangelogAccount := "tmp"
	testSnapshotAccount := "tmp"
	testChangelogPrimaryMedium := "default"
	testSnapshotPrimaryMedium := "default"

	configEmpty := tabletcellbundle.TabletCellBundleModel{}

	configOnlyName := tabletcellbundle.TabletCellBundleModel{
		Name: types.StringValue(testTabletCellBundleName),
	}

	configWithoutOptions := tabletcellbundle.TabletCellBundleModel{
		Name:            types.StringValue(testTabletCellBundleName),
		TabletCellCount: types.Int64Value(0),
	}

	configWithEmptyOptions := tabletcellbundle.TabletCellBundleModel{
		Name:            types.StringValue(testTabletCellBundleName),
		TabletCellCount: types.Int64Value(0),
		Options:         &tabletcellbundle.TabletCellBundleOptionsModel{},
	}

	configWithEmptyNodeTagFilter := tabletcellbundle.TabletCellBundleModel{
		Name:            types.StringValue(testTabletCellBundleName),
		TabletCellCount: types.Int64Value(0),
		NodeTagFilter:   types.StringValue(""),
		Options: &tabletcellbundle.TabletCellBundleOptionsModel{
			ChangelogAccount:       types.StringValue(testChangelogAccount),
			SnapshotAccount:        types.StringValue(testSnapshotAccount),
			ChangelogPrimaryMedium: types.StringValue(testChangelogPrimaryMedium),
			SnapshotPrimaryMedium:  types.StringValue(testSnapshotPrimaryMedium),
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testtestTabletCellBundleYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusTabletCellBundleConfig(resourceID, configEmpty),
				ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusTabletCellBundleConfig(resourceID, configOnlyName),
				ExpectError: regexp.MustCompile(`The argument "tablet_cell_count" is required, but no definition was found.`),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusTabletCellBundleConfig(resourceID, configWithoutOptions),
				ExpectError: regexp.MustCompile(`The argument "options" is required, but no definition was found.`),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusTabletCellBundleConfig(resourceID, configWithEmptyOptions),
				ExpectError: regexp.MustCompile(`Inappropriate value for attribute "options": attributes "changelog_account",\n"changelog_primary_medium", "snapshot_account", and "snapshot_primary_medium"\nare required.`),
			},
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusTabletCellBundleConfig(resourceID, configWithEmptyNodeTagFilter),
				ExpectError: regexp.MustCompile("Attribute node_tag_filter string length must be at least 1, got: 0"),
			},
		},
	})
}

func TestTabletCellBundleResourceCreateAndUpdate(t *testing.T) {
	resourceID := "fakebundle"
	testTabletCellBundleName := resourceID
	testtestTabletCellBundleYTCypressPath := fmt.Sprintf("//sys/tablet_cell_bundles/%s", testTabletCellBundleName)
	testChangelogAccount := "tmp"
	testSnapshotAccount := "tmp"
	testChangelogPrimaryMedium := "default"
	testSnapshotPrimaryMedium := "default"
	testTabletCellCount := int64(1)
	testNodeTagFilter := "localhost"
	testQuorum := int64(1)
	testReplicationFactor := int64(1)

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionUse},
			InheritanceMode: "object_and_descendants",
		},
	}

	configCreate := tabletcellbundle.TabletCellBundleModel{
		Name:            types.StringValue(testTabletCellBundleName),
		TabletCellCount: types.Int64Value(testTabletCellCount),
		Options: &tabletcellbundle.TabletCellBundleOptionsModel{
			ChangelogAccount:           types.StringValue(testChangelogAccount),
			SnapshotAccount:            types.StringValue(testSnapshotAccount),
			ChangelogPrimaryMedium:     types.StringValue(testChangelogPrimaryMedium),
			SnapshotPrimaryMedium:      types.StringValue(testSnapshotPrimaryMedium),
			ChangelogWriteQuorum:       types.Int64Value(testQuorum),
			ChangelogReadQuorum:        types.Int64Value(testQuorum),
			ChangelogReplicationFactor: types.Int64Value(testReplicationFactor),
			SnapshotReplicationFactor:  types.Int64Value(testReplicationFactor),
		},
	}

	configUpdate := tabletcellbundle.TabletCellBundleModel{
		Name:            types.StringValue(testTabletCellBundleName),
		TabletCellCount: types.Int64Value(testTabletCellCount),
		NodeTagFilter:   types.StringValue(testNodeTagFilter),
		ACL:             acl.ToACLModel(testACL),
		Options: &tabletcellbundle.TabletCellBundleOptionsModel{
			ChangelogAccount:           types.StringValue(testChangelogAccount),
			SnapshotAccount:            types.StringValue(testSnapshotAccount),
			ChangelogPrimaryMedium:     types.StringValue(testChangelogPrimaryMedium),
			SnapshotPrimaryMedium:      types.StringValue(testSnapshotPrimaryMedium),
			ChangelogWriteQuorum:       types.Int64Value(testQuorum),
			ChangelogReadQuorum:        types.Int64Value(testQuorum),
			ChangelogReplicationFactor: types.Int64Value(testReplicationFactor),
			SnapshotReplicationFactor:  types.Int64Value(testReplicationFactor),
		},
	}

	accDynConfigReconfigureSlotsOnTabletNodes()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testtestTabletCellBundleYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusTabletCellBundleConfig(resourceID, configCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "name", testTabletCellBundleName),
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "options/changelog_account", testChangelogAccount),
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "options/changelog_primary_medium", testChangelogPrimaryMedium),
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "options/snapshot_account", testSnapshotAccount),
					accCheckYTsaurusInt64Attribute(testtestTabletCellBundleYTCypressPath, "options/changelog_write_quorum", testQuorum),
					accCheckYTsaurusInt64Attribute(testtestTabletCellBundleYTCypressPath, "options/changelog_read_quorum", testQuorum),
					accCheckYTsaurusInt64Attribute(testtestTabletCellBundleYTCypressPath, "options/changelog_replication_factor", testReplicationFactor),
					accCheckYTsaurusInt64Attribute(testtestTabletCellBundleYTCypressPath, "options/snapshot_replication_factor", testReplicationFactor),
					accCheckYTsaurusUInt64Attribute(testtestTabletCellBundleYTCypressPath, "tablet_cell_count", uint64(testTabletCellCount)),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusTabletCellBundleConfig(resourceID, configUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusACLAttribute(testtestTabletCellBundleYTCypressPath, testACL),
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "node_tag_filter", testNodeTagFilter),
				),
			},
		},
	})
}

func TestTabletCellBundleResourceCreateWithAllOptions(t *testing.T) {
	resourceID := "fakebundle"
	testTabletCellBundleName := resourceID
	testtestTabletCellBundleYTCypressPath := fmt.Sprintf("//sys/tablet_cell_bundles/%s", testTabletCellBundleName)
	testChangelogAccount := "tmp"
	testSnapshotAccount := "tmp"
	testChangelogPrimaryMedium := "default"
	testSnapshotPrimaryMedium := "default"
	testTabletCellCount := int64(1)
	testNodeTagFilter := "localhost"
	testQuorum := int64(1)
	testReplicationFactor := int64(1)

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionUse},
			InheritanceMode: "object_and_descendants",
		},
	}

	configCreate := tabletcellbundle.TabletCellBundleModel{
		Name:            types.StringValue(testTabletCellBundleName),
		TabletCellCount: types.Int64Value(testTabletCellCount),
		NodeTagFilter:   types.StringValue(testNodeTagFilter),
		ACL:             acl.ToACLModel(testACL),
		Options: &tabletcellbundle.TabletCellBundleOptionsModel{
			ChangelogAccount:           types.StringValue(testChangelogAccount),
			SnapshotAccount:            types.StringValue(testSnapshotAccount),
			ChangelogPrimaryMedium:     types.StringValue(testChangelogPrimaryMedium),
			SnapshotPrimaryMedium:      types.StringValue(testSnapshotPrimaryMedium),
			ChangelogWriteQuorum:       types.Int64Value(testQuorum),
			ChangelogReadQuorum:        types.Int64Value(testQuorum),
			ChangelogReplicationFactor: types.Int64Value(testReplicationFactor),
			SnapshotReplicationFactor:  types.Int64Value(testReplicationFactor),
		},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testtestTabletCellBundleYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusTabletCellBundleConfig(resourceID, configCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "name", testTabletCellBundleName),
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "options/changelog_account", testChangelogAccount),
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "options/changelog_primary_medium", testChangelogPrimaryMedium),
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "options/snapshot_account", testSnapshotAccount),
					accCheckYTsaurusInt64Attribute(testtestTabletCellBundleYTCypressPath, "options/changelog_write_quorum", testQuorum),
					accCheckYTsaurusInt64Attribute(testtestTabletCellBundleYTCypressPath, "options/changelog_read_quorum", testQuorum),
					accCheckYTsaurusInt64Attribute(testtestTabletCellBundleYTCypressPath, "options/changelog_replication_factor", testReplicationFactor),
					accCheckYTsaurusInt64Attribute(testtestTabletCellBundleYTCypressPath, "options/snapshot_replication_factor", testReplicationFactor),
					accCheckYTsaurusUInt64Attribute(testtestTabletCellBundleYTCypressPath, "tablet_cell_count", uint64(testTabletCellCount)),
					accCheckYTsaurusACLAttribute(testtestTabletCellBundleYTCypressPath, testACL),
					accCheckYTsaurusStringAttribute(testtestTabletCellBundleYTCypressPath, "node_tag_filter", testNodeTagFilter),
				),
			},
		},
	})

}

func accResourceYtsaurusTabletCellBundleConfig(id string, m tabletcellbundle.TabletCellBundleModel) string {

	config := fmt.Sprintf(`
	resource "ytsaurus_tablet_cell_bundle" %q {`, id)

	if !m.Name.IsNull() {
		config += fmt.Sprintf(`
		name = %q`, m.Name.ValueString())
	}

	if !m.TabletCellCount.IsNull() {
		config += fmt.Sprintf(`
		tablet_cell_count = %d`, m.TabletCellCount.ValueInt64())
	}

	if !m.NodeTagFilter.IsNull() {
		config += fmt.Sprintf(`
		node_tag_filter = %q`, m.NodeTagFilter.ValueString())
	}

	acl, _ := acl.ToYTsaurusACL(m.ACL)
	if len(acl) > 0 {
		config += accAddACLConfig(acl)
	}

	if m.Options != nil {
		config += `
		options = {`

		if !m.Options.ChangelogAccount.IsNull() {
			config += fmt.Sprintf(`
			changelog_account = %q`, m.Options.ChangelogAccount.ValueString())
		}

		if !m.Options.SnapshotAccount.IsNull() {
			config += fmt.Sprintf(`
			snapshot_account = %q`, m.Options.SnapshotAccount.ValueString())
		}

		if !m.Options.ChangelogPrimaryMedium.IsNull() {
			config += fmt.Sprintf(`
			changelog_primary_medium = %q`, m.Options.ChangelogPrimaryMedium.ValueString())
		}

		if !m.Options.SnapshotPrimaryMedium.IsNull() {
			config += fmt.Sprintf(`
			snapshot_primary_medium = %q`, m.Options.SnapshotPrimaryMedium.ValueString())
		}

		if !m.Options.ChangelogWriteQuorum.IsNull() {
			config += fmt.Sprintf(`
			changelog_write_quorum = %d`, m.Options.ChangelogWriteQuorum.ValueInt64())
		}

		if !m.Options.ChangelogReadQuorum.IsNull() {
			config += fmt.Sprintf(`
			changelog_read_quorum = %d`, m.Options.ChangelogReadQuorum.ValueInt64())
		}

		if !m.Options.ChangelogReplicationFactor.IsNull() {
			config += fmt.Sprintf(`
			changelog_replication_factor = %d`, m.Options.ChangelogReplicationFactor.ValueInt64())
		}

		if !m.Options.SnapshotReplicationFactor.IsNull() {
			config += fmt.Sprintf(`
			snapshot_replication_factor = %d`, m.Options.SnapshotReplicationFactor.ValueInt64())
		}

		config += `
		}`
	}

	config += `
	}`

	return config
}
