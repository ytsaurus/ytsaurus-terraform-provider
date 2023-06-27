package acc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"go.ytsaurus.tech/yt/go/yt"

	"terraform-provider-ytsaurus/internal/resource/acl"
	"terraform-provider-ytsaurus/internal/resource/mapnode"
)

func TestMapNodeResourceCreateAndUpdate(t *testing.T) {
	resourceID := "projecthome"
	testMapNodePath := "//home/fakeproject"
	testMapNodeDefaultAccount := "default"
	testMapNodeSysAccount := "sys"
	inheritAclTrue := true
	inheritAclFalse := false

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionRead},
			InheritanceMode: "object_and_descendants",
		},
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"admins"},
			Permissions:     []string{yt.PermissionAdminister, yt.PermissionRead, yt.PermissionWrite, yt.PermissionRemove},
			InheritanceMode: "object_and_descendants",
		},
	}

	configEmpty := mapnode.MapNodeModel{}

	configCreateWithoutOptions := mapnode.MapNodeModel{
		Path: types.StringValue(testMapNodePath),
	}

	configCreateUpdate := mapnode.MapNodeModel{
		Path:       types.StringValue(testMapNodePath),
		InheritACL: types.BoolValue(inheritAclFalse),
		Account:    types.StringValue(testMapNodeSysAccount),
		ACL:        acl.ToACLModel(testACL),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testMapNodePath),
		Steps: []resource.TestStep{
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusMapNodeConfig(resourceID, configEmpty),
				ExpectError: regexp.MustCompile(`The argument "path" is required, but no definition was found.`),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusMapNodeConfig(resourceID, configCreateWithoutOptions),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testMapNodePath, "path", testMapNodePath),
					accCheckYTsaurusStringAttribute(testMapNodePath, "account", testMapNodeDefaultAccount),
					accCheckYTsaurusBoolAttribute(testMapNodePath, "inherit_acl", inheritAclTrue),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusMapNodeConfig(resourceID, configCreateUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testMapNodePath, "path", testMapNodePath),
					accCheckYTsaurusStringAttribute(testMapNodePath, "account", testMapNodeSysAccount),
					accCheckYTsaurusBoolAttribute(testMapNodePath, "inherit_acl", inheritAclFalse),
					accCheckYTsaurusACLAttribute(testMapNodePath, testACL),
				),
			},
		},
	})
}

func TestMapNodeResourceCreateWithAllOptions(t *testing.T) {
	resourceID := "projecthome"
	testMapNodePath := "//home/fakeproject"
	testMapNodeSysAccount := "sys"
	inheritAclFalse := false

	testACL := []yt.ACE{
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"users"},
			Permissions:     []string{yt.PermissionRead},
			InheritanceMode: "object_and_descendants",
		},
		{
			Action:          yt.ActionAllow,
			Subjects:        []string{"admins"},
			Permissions:     []string{yt.PermissionAdminister, yt.PermissionRead, yt.PermissionWrite, yt.PermissionRemove},
			InheritanceMode: "object_and_descendants",
		},
	}

	configCreate := mapnode.MapNodeModel{
		Path:       types.StringValue(testMapNodePath),
		InheritACL: types.BoolValue(inheritAclFalse),
		Account:    types.StringValue(testMapNodeSysAccount),
		ACL:        acl.ToACLModel(testACL),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testMapNodePath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusMapNodeConfig(resourceID, configCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testMapNodePath, "path", testMapNodePath),
					accCheckYTsaurusStringAttribute(testMapNodePath, "account", testMapNodeSysAccount),
					accCheckYTsaurusBoolAttribute(testMapNodePath, "inherit_acl", inheritAclFalse),
					accCheckYTsaurusACLAttribute(testMapNodePath, testACL),
				),
			},
		},
	})
}

// func accResourceYtsaurusMapNodeConfig(resource, id, path, account string, inheritAcl bool, acl []yt.ACE) string {
func accResourceYtsaurusMapNodeConfig(id string, m mapnode.MapNodeModel) string {
	config := fmt.Sprintf(`
		resource "ytsaurus_map_node" %q {`, id)

	if !m.Path.IsNull() {
		config += fmt.Sprintf(`
			path = %q`, m.Path.ValueString())
	}

	if !m.Account.IsNull() {
		config += fmt.Sprintf(`
		account = %q`, m.Account.ValueString())
	}

	if !m.InheritACL.IsNull() {
		config += fmt.Sprintf(`
		inherit_acl = %t`, m.InheritACL.ValueBool())
	}

	acl := acl.ToYTsaurusACL(m.ACL)
	if len(acl) > 0 {
		config += accAddACLConfig(acl)
	}

	config += `
	}`

	return config
}
