package acc

import (
	"regexp"
	"terraform-provider-ytsaurus/internal/resource/acl"
	"terraform-provider-ytsaurus/internal/resource/mapnode"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"go.ytsaurus.tech/yt/go/yt"
)

func TestACLSchemaMisconfigurations(t *testing.T) {
	resourceID := "acltest"
	testMapNodePath := "/tmp/acltest"
	testMapNodeTmpAccount := "tmp"
	inheritAcl := true

	badActionACL := []yt.ACE{
		{
			Action:          "fake",
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionRead},
			InheritanceMode: "object_and_descendants",
		},
	}
	badPermissionsACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{"fake"},
			InheritanceMode: "object_and_descendants",
		},
	}

	mapNodeConfigBadActionACL := mapnode.MapNodeModel{
		Path:       types.StringValue(testMapNodePath),
		Account:    types.StringValue(testMapNodeTmpAccount),
		InheritACL: types.BoolValue(inheritAcl),
		ACL:        acl.ToACLModel(badActionACL),
	}

	mapNodeConfigBadPermissionsACL := mapnode.MapNodeModel{
		Path:       types.StringValue(testMapNodePath),
		Account:    types.StringValue(testMapNodeTmpAccount),
		InheritACL: types.BoolValue(inheritAcl),
		ACL:        acl.ToACLModel(badPermissionsACL),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() +
					accResourceYtsaurusMapNodeConfig(resourceID, mapNodeConfigBadActionACL),
				ExpectError: regexp.MustCompile(`action value must be one of: \["\\"allow\\"" "\\"deny\\""\]`),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() +
					accResourceYtsaurusMapNodeConfig(resourceID, mapNodeConfigBadPermissionsACL),
				ExpectError: regexp.MustCompile(`permissions\[Value\("fake"\)\] value must be one of:`),
			},
		},
	})

}
