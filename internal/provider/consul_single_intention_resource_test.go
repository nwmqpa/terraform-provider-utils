// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConsulSingleIntentionResourceWithoutPeer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConsulSingleIntentionResourceConfigWithoutPeer("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "source_service", "invalid-source-service"),
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "destination_service", "invalid-service-one"),
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "id", "invalid-service-one_invalid-source-service"),
				),
			},
			// ImportState testing
			// {
			// 	ResourceName:      "utils_consul_single_intention.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// Update and Read testing
			{
				Config: testAccConsulSingleIntentionResourceConfigWithoutPeer("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "destination_service", "invalid-service-two"),
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "id", "invalid-service-two_invalid-source-service"),
				),
			},
			// Delete testing
		},
	})
}

func testAccConsulSingleIntentionResourceConfigWithoutPeer(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "utils_consul_single_intention" "test" {
	destination_service = "invalid-service-%[1]s"
	source_service = "invalid-source-service"
}
`, configurableAttribute)
}

func TestAccConsulSingleIntentionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConsulSingleIntentionResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "source_service", "invalid-source-service"),
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "source_peer", "invalid-source-peer"),
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "destination_service", "invalid-service-one"),
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "id", "invalid-service-one_invalid-source-service_invalid-source-peer"),
				),
			},
			// ImportState testing
			// {
			// 	ResourceName:      "utils_consul_single_intention.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// Update and Read testing
			{
				Config: testAccConsulSingleIntentionResourceConfig("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "destination_service", "invalid-service-two"),
					resource.TestCheckResourceAttr("utils_consul_single_intention.test", "id", "invalid-service-two_invalid-source-service_invalid-source-peer"),
				),
			},
			// Delete testing
		},
	})
}

func testAccConsulSingleIntentionResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "utils_consul_single_intention" "test" {
	destination_service = "invalid-service-%[1]s"
	source_service = "invalid-source-service"
	source_peer    = "invalid-source-peer"
}
`, configurableAttribute)
}
