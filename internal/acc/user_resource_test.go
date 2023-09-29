package acc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-ytsaurus/internal/resource/user"
)

func TestUserResourceCreateAndUpdate(t *testing.T) {
	resourceID := "testuser"
	testUserName := resourceID
	testUserNameRenamed := "testuser_renamed"
	testUserYTCypressPath := fmt.Sprintf("//sys/users/%s", testUserName)
	testUserYTCypressPathRenamed := fmt.Sprintf("//sys/users/%s", testUserNameRenamed)
	testMemberOf := []string{"admins", "devs"}
	testMemberOfValues := []attr.Value{}
	for _, g := range testMemberOf {
		testMemberOfValues = append(testMemberOfValues, types.StringValue(g))
	}
	testMemberOfExpected := []string{"devs", "admins", "users"}

	configEmpty := user.UserModel{}

	configCreateWithOnlyName := user.UserModel{
		Name: types.StringValue(testUserName),
	}

	configRenameUser := user.UserModel{
		Name: types.StringValue(testUserNameRenamed),
	}

	configAddMemberOf := user.UserModel{
		Name:     types.StringValue(testUserName),
		MemberOf: types.SetValueMust(types.StringType, testMemberOfValues),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testUserYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config:      accGetYTLocalDockerProviderConfig() + accResourceYtsaurusUserConfig(resourceID, configEmpty),
				ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusUserConfig(resourceID, configCreateWithOnlyName),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testUserYTCypressPath, "name", testUserName),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusUserConfig(resourceID, configRenameUser),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testUserYTCypressPathRenamed, "name", testUserNameRenamed),
				),
			},
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusUserConfig(resourceID, configAddMemberOf),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testUserYTCypressPath, "name", testUserName),
					accCheckYTsaurusUserMemberOfAttribute(testUserYTCypressPath, testMemberOfExpected),
				),
			},
		},
	})
}

func TestUserResourceCreateWithAllOptions(t *testing.T) {
	resourceID := "testuser"
	testUserName := resourceID
	testUserYTCypressPath := fmt.Sprintf("//sys/users/%s", testUserName)
	testMemberOf := []string{"admins", "devs"}
	testMemberOfValues := []attr.Value{}
	for _, g := range testMemberOf {
		testMemberOfValues = append(testMemberOfValues, types.StringValue(g))
	}
	testMemberOfExpected := []string{"devs", "admins", "users"}

	configCreate := user.UserModel{
		Name:     types.StringValue(testUserName),
		MemberOf: types.SetValueMust(types.StringType, testMemberOfValues),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             accCheckYTsaurusObjectDestroyed(testUserYTCypressPath),
		Steps: []resource.TestStep{
			{
				Config: accGetYTLocalDockerProviderConfig() + accResourceYtsaurusUserConfig(resourceID, configCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					accCheckYTsaurusStringAttribute(testUserYTCypressPath, "name", testUserName),
					accCheckYTsaurusUserMemberOfAttribute(testUserYTCypressPath, testMemberOfExpected),
				),
			},
		},
	})
}

// func accResourceYtsaurusUserConfig(resource, id, name string, memberOf []string) string {
func accResourceYtsaurusUserConfig(id string, m user.UserModel) string {
	config := fmt.Sprintf(`
	resource "ytsaurus_user" %q {`, id)

	if !m.Name.IsNull() {
		config += fmt.Sprintf(`
		name = %s`, m.Name)
	}

	if !m.MemberOf.IsNull() {
		var memberOf []string
		m.MemberOf.ElementsAs(ctx, &memberOf, false)
		config += `
		member_of = [`
		for _, g := range memberOf {
			config += fmt.Sprintf(`
			%q,`, g)
		}
		config += `
		]`
	}

	config += `
	}`

	return config
}
