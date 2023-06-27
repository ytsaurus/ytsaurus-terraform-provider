package acc

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"
	"go.ytsaurus.tech/yt/go/yt/ythttp"

	tfprovider "github.com/hashicorp/terraform-plugin-framework/provider"

	"terraform-provider-ytsaurus/internal/provider"
	"terraform-provider-ytsaurus/internal/set"
)

const (
	localDockerYTCluster = "localhost:8000"
)

func accGetYTLocalDockerProviderConfig() string {
	return fmt.Sprintf(`
	provider "ytsaurus" {
		cluster = "%s"
	}
	`, localDockerYTCluster)
}

func accCheckYTsaurusObjectDestroyed(sp string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := ypath.Path(sp)
		ok, err := testYTClient.NodeExists(ctx, p, nil)
		if err != nil {
			return err
		}
		if ok {
			return fmt.Errorf("object %q unexpectedly exists", p.String())
		}
		return nil
	}
}

// func accCheckYTsaurusObjectExists(objectCypressPath string) (bool, error) {
// 	p := ypath.Path(objectCypressPath)
// 	return testYTClient.NodeExists(ctx, p, nil)
// }

// func accGetYTsaurusStringAttribute(objectCypressPath, attributeName string) string {
// 	var result string
// 	p := ypath.Path(objectCypressPath).Attr(attributeName)
// 	if err := testYTClient.GetNode(ctx, p, &result, nil); err != nil {
// 		panic(err.Error())
// 	}
// 	return result
// }

func accCompareYTsaurusAttribute(objectCypressPath, attributeName string, value, result interface{}) error {
	p := ypath.Path(objectCypressPath).Attr(attributeName)
	if err := testYTClient.GetNode(ctx, p, &result, nil); err != nil {
		return err
	}
	if result != value {
		return fmt.Errorf("YTsaurus %q expected %v, got %v", p, value, result)
	}
	return nil
}

func accCheckYTsaurusBoolAttribute(objectCypressPath, attributeName string, value bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var result bool
		return accCompareYTsaurusAttribute(objectCypressPath, attributeName, value, result)
	}
}

func accCheckYTsaurusStringAttribute(objectCypressPath, attributeName, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var result string
		return accCompareYTsaurusAttribute(objectCypressPath, attributeName, value, result)
	}
}

func accCheckYTsaurusInt64Attribute(objectCypressPath, attributeName string, value int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var result int64
		return accCompareYTsaurusAttribute(objectCypressPath, attributeName, value, result)
	}
}

func accCheckYTsaurusUInt64Attribute(objectCypressPath, attributeName string, value uint64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var result int64
		return accCompareYTsaurusAttribute(objectCypressPath, attributeName, value, result)
	}
}

func accCheckYTsaurusListAttribute(objectCypressPath, attributeName string, value []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := ypath.Path(objectCypressPath).Attr(attributeName)
		var result []string
		if err := testYTClient.GetNode(ctx, p, &result, nil); err != nil {
			return err
		}

		sort.Strings(result)
		sort.Strings(value)

		if !reflect.DeepEqual(result, value) {
			return fmt.Errorf("YTsaurus %q expected %q, got %q", p.String(), value, result)
		}
		return nil
	}
}

func accCheckYTsaurusUserMemberOfAttribute(objectCypressPath string, value []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := ypath.Path(objectCypressPath).Attr("member_of")
		var result []string
		if err := testYTClient.GetNode(ctx, p, &result, nil); err != nil {
			return err
		}
		resultSet := set.ToStringSet(result)
		valueSet := set.ToStringSet(value)

		if len(valueSet.Difference(resultSet)) > 0 {
			return fmt.Errorf("YTsaurus %q expected %q, got %q", p.String(), valueSet, resultSet)
		}

		return nil
	}
}

type nodeDynConfigTabletNode struct {
	TabletDynamicMemory int64 `yson:"tablet_dynamic_memory"`
	TabletStaticMemory  int64 `yson:"tablet_static_memory"`
	Slots               int64 `yson:"slots"`
}

type nodeDynConfig struct {
	ConfigAnnotation string                  `yson:"config_annotation"`
	TabletNode       nodeDynConfigTabletNode `yson:"tablet_node"`
}

func accDynConfigReconfigureSlotsOnTabletNodes() {
	config := nodeDynConfig{
		ConfigAnnotation: "default",
		TabletNode: nodeDynConfigTabletNode{
			TabletDynamicMemory: 524288000,
			TabletStaticMemory:  1073741824,
			Slots:               2,
		},
	}
	p := ypath.Path("//sys/cluster_nodes/@config/%true")
	if err := testYTClient.SetNode(ctx, p, config, nil); err != nil {
		panic(err.Error())
	}
}

func accCheckYTsaurusACLAttribute(objectCypressPath string, value []yt.ACE) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := ypath.Path(objectCypressPath).Attr("acl")
		var result []yt.ACE
		if err := testYTClient.GetNode(ctx, p, &result, nil); err != nil {
			return err
		}

		if len(result) == 0 && len(value) == 0 {
			return nil
		}

		if len(result) != len(value) {
			return fmt.Errorf("YTsaurus %q expected %v, got %v", p.String(), value, result)
		}

		for i := 0; i < len(result); i++ {
			sort.Strings(result[i].Subjects)
			sort.Strings(value[i].Subjects)

			sort.Strings(result[i].Permissions)
			sort.Strings(value[i].Permissions)
		}

		if !reflect.DeepEqual(result, value) {
			return fmt.Errorf("YTsaurus %q expected %v, got %v", p.String(), value, result)
		}
		return nil
	}
}

func accAddACLConfig(acl []yt.ACE) string {
	config := ""
	if len(acl) == 0 {
		return config
	}
	config += `
		acl = [`
	for _, ace := range acl {
		config += `
			{`
		config += fmt.Sprintf(`
				action = %q`, ace.Action)

		config += `
				subjects = [`
		for _, s := range ace.Subjects {
			config += fmt.Sprintf(`
					%q,`, s)
		}
		config += `
				]`

		config += `
				permissions = [`
		for _, p := range ace.Permissions {
			config += fmt.Sprintf(`
					%q,`, p)
		}
		config += `
				]`

		if len(ace.InheritanceMode) > 0 {
			config += fmt.Sprintf(`
				inheritance_mode = %q`, ace.InheritanceMode)
		}
		config += `
			},`
	}
	config += `
		]`

	return config
}

var (
	Provider                 tfprovider.Provider
	ProtoV6ProviderFactories map[string]func() (tfprotov6.ProviderServer, error)
	testYTClient             yt.Client
	ctx                      context.Context
	emptyACL                 []yt.ACE
)

func init() {
	Provider = provider.New()
	ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"ytsaurus": providerserver.NewProtocol6WithError(Provider),
	}

	testYTClient, _ = ythttp.NewClient(&yt.Config{Proxy: localDockerYTCluster})
	ctx = context.Background()
	emptyACL = []yt.ACE{}
}
