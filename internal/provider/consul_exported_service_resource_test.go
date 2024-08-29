// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConsulExportedServiceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConsulExportedServiceResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "peer_name", "invalid-peer"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "service_to_export", "invalid-service-one"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "id", "invalid-peer_invalid-service-one"),
				),
			},
			// ImportState testing
			// {
			// 	ResourceName:      "utils_consul_exported_service.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// Update and Read testing
			{
				Config: testAccConsulExportedServiceResourceConfig("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "service_to_export", "invalid-service-two"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "id", "invalid-peer_invalid-service-two"),
				),
			},
			// Delete testing
		},
	})
}

func testAccConsulExportedServiceResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "utils_consul_exported_service" "test" {
	peer_name = "invalid-peer"
	service_to_export = "invalid-service-%[1]s"
}
`, configurableAttribute)
}

func TestAccConsulExportedServiceResourceMultiple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConsulExportedServiceResourceConfigMultiple("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "peer_name", "invalid-peer"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "service_to_export", "invalid-service-one"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "id", "invalid-peer_invalid-service-one"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test2", "peer_name", "invalid-peer2"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test2", "service_to_export", "invalid-service-one"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test2", "id", "invalid-peer2_invalid-service-one"),
				),
			},
			// ImportState testing
			// {
			// 	ResourceName:      "utils_consul_exported_service.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// Update and Read testing
			{
				Config: testAccConsulExportedServiceResourceConfigMultiple("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "service_to_export", "invalid-service-two"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test", "id", "invalid-peer_invalid-service-two"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test2", "service_to_export", "invalid-service-two"),
					resource.TestCheckResourceAttr("utils_consul_exported_service.test2", "id", "invalid-peer2_invalid-service-two"),
				),
			},
			// Delete testing
		},
	})
}

func testAccConsulExportedServiceResourceConfigMultiple(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "utils_consul_exported_service" "test" {
	peer_name = "invalid-peer"
	service_to_export = "invalid-service-%[1]s"
}

resource "utils_consul_exported_service" "test2" {
	peer_name = "invalid-peer2"
	service_to_export = "invalid-service-%[1]s"
}
`, configurableAttribute)
}
