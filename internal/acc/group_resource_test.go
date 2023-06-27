package acc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-ytsaurus/internal/resource/group"
)

func TestGroupResource(t *testing.T) {
	resourceID := "testgroup"

	testGroupName := "testgroup"
	testGroupNameUpdated := "testgroup_renamed"
	testGroupYTCypressPath := fmt.Sprintf("//sys/groups/%s", testGroupName)
	testGroupYTCypressPathUpdated := fmt.Sprintf("//sys/groups/%s", testGroupNameUpdated)

	configEmpty := group.GroupModel{}

	configCreate := group.GroupModel{
		Name: types.StringValue(testGroupName),
	}

	configUpdate := group.GroupModel{
		Name: types.StringValue(testGroupNameUpdated),
	}

	// create test group -> rename test group -> delete test group
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testGroupYTCypressPathUpdated),
		Steps: []resource.TestStep{
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusGroupConfig(resourceID, configEmpty),
				ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusGroupConfig(resourceID, configCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testGroupYTCypressPath, "name", testGroupName),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusGroupConfig(resourceID, configUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testGroupYTCypressPathUpdated, "name", testGroupNameUpdated),
				),
			},
		},
	})
}

func accResourceYtsaurusGroupConfig(id string, m group.GroupModel) string {
	config := fmt.Sprintf(`
	resource "ytsaurus_group" %q {`, id)

	if !m.Name.IsNull() {
		config += fmt.Sprintf(`
		name = %q`, m.Name.ValueString())
	}

	config += `
	}`

	return config
}
