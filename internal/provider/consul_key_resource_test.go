// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConsulKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConsulKeyResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_key.test", "path", "test/one"),
					resource.TestCheckResourceAttr("utils_consul_key.test", "value", "test"),
					resource.TestCheckResourceAttr("utils_consul_key.test", "id", "test/one"),
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
				Config: testAccConsulKeyResourceConfig("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_key.test", "path", "test/two"),
					resource.TestCheckResourceAttr("utils_consul_key.test", "value", "test"),
					resource.TestCheckResourceAttr("utils_consul_key.test", "id", "test/two"),
				),
			},
			// Delete testing
		},
	})
}

func testAccConsulKeyResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "utils_consul_key" "test" {
	path = "test/%[1]s"
	value = "test"
}
`, configurableAttribute)
}
