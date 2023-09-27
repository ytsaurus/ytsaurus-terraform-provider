package ytsaurus

import (
	"go.ytsaurus.tech/yt/go/yt"
)

type Group struct {
	ID   string `yson:"id"`
	Name string `yson:"name"`
}

type User struct {
	ID       string    `yson:"id"`
	Name     string    `yson:"name"`
	MemberOf *[]string `yson:"member_of"`
}

type AccountResourceLimits struct {
	NodeCount          int64            `yson:"node_count"`
	ChunkCount         int64            `yson:"chunk_count"`
	TabletCount        int64            `yson:"tablet_count"`
	TabletStaticMemory int64            `yson:"tablet_static_memory"`
	DiskSpacePerMedium map[string]int64 `yson:"disk_space_per_medium"`
}

type Account struct {
	ID             string                 `yson:"id"`
	Name           string                 `yson:"name"`
	ResourceLimits *AccountResourceLimits `yson:"resource_limits"`
	InheritACL     bool                   `yson:"inherit_acl"`
	ACL            []yt.ACE               `yson:"acl"`
	ParentName     string                 `yson:"parent_name"`
}

type Medium struct {
	ID                  string       `yson:"id"`
	Name                string       `yson:"name"`
	ACL                 []yt.ACE     `yson:"acl"`
	Config              MediumConfig `yson:"config"`
	DiskFamilyWhitelist *[]string    `yson:"disk_family_whitelist"`
}

type MediumConfig struct {
	MaxErasureReplicasPerRack       int64 `yson:"max_erasure_replicas_per_rack"`
	MaxJournalReplicasPerRack       int64 `yson:"max_journal_replicas_per_rack"`
	MaxRegularReplicasPerRack       int64 `yson:"max_regular_replicas_per_rack"`
	MaxReplicasPerRack              int64 `yson:"max_replicas_per_rack"`
	MaxReplicationFactor            int64 `yson:"max_replication_factor"`
	PreferLocalHostForDynamicTables bool  `yson:"prefer_local_host_for_dynamic_tables"`
}

type MapNode struct {
	ID         string   `yson:"id"`
	Path       string   `yson:"path"`
	Account    string   `yson:"account"`
	InheritACL bool     `yson:"inherit_acl"`
	ACL        []yt.ACE `yson:"acl"`
}

type TabletCellBundleOptions struct {
	ChangelogAccount           string `yson:"changelog_account"`
	ChangelogWriteQuorum       int64  `yson:"changelog_write_quorum"`
	ChangelogReadQuorum        int64  `yson:"changelog_read_quorum"`
	ChangelogReplicationFactor int64  `yson:"changelog_replication_factor"`
	ChangelogPrimaryMedium     string `yson:"changelog_primary_medium"`
	SnapshotAccount            string `yson:"snapshot_account"`
	SnapshotReplicationFactor  int64  `yson:"snapshot_replication_factor"`
	SnapshotPrimaryMedium      string `yson:"snapshot_primary_medium"`
}

type TabletCellBundle struct {
	ID              string                   `yson:"id"`
	Name            string                   `yson:"name"`
	NodeTagFilter   string                   `yson:"node_tag_filter"`
	TabletCellCount int64                    `yson:"tablet_cell_count"`
	ACL             []yt.ACE                 `yson:"acl"`
	Options         *TabletCellBundleOptions `yson:"options"`
}

type TabletCellBundleArea struct {
	ID            string `yson:"id"`
	CellCount     int64  `yson:"cell_count"`
	NodeTagFilter string `yson:"node_tag_filter"`
}

type TabletCellBundleAreas map[string]TabletCellBundleArea

type SchedulerPoolIntegralGuarantees struct {
	GuaranteeType           *string                 `yson:"guarantee_type"`
	ResourceFlow            *SchedulerPoolResources `yson:"resource_flow"`
	BurstGuaranteeResources *SchedulerPoolResources `yson:"burst_guarantee_resources"`
}

type SchedulerPoolResources struct {
	CPU    *int64 `yson:"cpu"`
	Memory *int64 `yson:"memory"`
}

type SchedulerPool struct {
	ID                        string                           `yson:"id"`
	Name                      string                           `yson:"name"`
	ACL                       []yt.ACE                         `yson:"acl"`
	Path                      string                           `yson:"path"`
	ParentName                *string                          `yson:"parent_name"`
	MaxRunningOperationCount  *int64                           `yson:"max_running_operation_count"`
	MaxOperationCount         *int64                           `yson:"max_operation_count"`
	IntegralGuarantees        *SchedulerPoolIntegralGuarantees `yson:"integral_guarantees"`
	StrongGuaranteeResources  *SchedulerPoolResources          `yson:"strong_guarantee_resources"`
	ResourceLimits            *SchedulerPoolResources          `yson:"resource_limits"`
	Weight                    *float64                         `yson:"weight"`
	Mode                      *string                          `yson:"mode"`
	ForbidImmediateOperations *bool                            `yson:"forbid_immediate_operations"`
}
