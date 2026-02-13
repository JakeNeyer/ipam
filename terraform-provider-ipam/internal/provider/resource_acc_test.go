package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint := os.Getenv("IPAM_ENDPOINT")
	token := os.Getenv("IPAM_TOKEN")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_environment" "acc" {
  name = "acc-env"
  pools = [
    { name = "acc-pool", cidr = "10.0.0.0/8" }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ipam_environment.acc", "id"),
					resource.TestCheckResourceAttr("ipam_environment.acc", "name", "acc-env"),
				),
			},
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_environment" "acc" {
  name = "acc-env-updated"
  pools = [
    { name = "acc-pool", cidr = "10.0.0.0/8" }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ipam_environment.acc", "id"),
					resource.TestCheckResourceAttr("ipam_environment.acc", "name", "acc-env-updated"),
				),
			},
			{
				ResourceName:      "ipam_environment.acc",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBlockResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint := os.Getenv("IPAM_ENDPOINT")
	token := os.Getenv("IPAM_TOKEN")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_environment" "acc" {
  name = "acc-block-env"
  pools = [
    { name = "acc-pool", cidr = "10.0.0.0/8" }
  ]
}

data "ipam_pools" "acc" {
  environment_id = ipam_environment.acc.id
}

resource "ipam_block" "acc" {
  name           = "acc-block"
  cidr           = "10.100.0.0/24"
  environment_id = ipam_environment.acc.id
  pool_id        = ipam_environment.acc.pool_ids[0]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ipam_block.acc", "id"),
					resource.TestCheckResourceAttr("ipam_block.acc", "name", "acc-block"),
					resource.TestCheckResourceAttr("ipam_block.acc", "cidr", "10.100.0.0/24"),
					resource.TestCheckResourceAttr("ipam_block.acc", "total_ips", "256"),
					resource.TestCheckResourceAttr("ipam_block.acc", "used_ips", "0"),
					resource.TestCheckResourceAttr("ipam_block.acc", "available_ips", "256"),
					resource.TestCheckResourceAttrPair("ipam_block.acc", "environment_id", "ipam_environment.acc", "id"),
				),
			},
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_environment" "acc" {
  name = "acc-block-env"
  pools = [
    { name = "acc-pool", cidr = "10.0.0.0/8" }
  ]
}

data "ipam_pools" "acc" {
  environment_id = ipam_environment.acc.id
}

resource "ipam_block" "acc" {
  name           = "acc-block-renamed"
  cidr           = "10.100.0.0/24"
  environment_id = ipam_environment.acc.id
  pool_id        = ipam_environment.acc.pool_ids[0]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ipam_block.acc", "name", "acc-block-renamed"),
				),
			},
			{
				ResourceName:      "ipam_block.acc",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAllocationResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint := os.Getenv("IPAM_ENDPOINT")
	token := os.Getenv("IPAM_TOKEN")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_environment" "acc" {
  name = "acc-alloc-env"
  pools = [
    { name = "acc-pool", cidr = "10.0.0.0/8" }
  ]
}

data "ipam_pools" "acc" {
  environment_id = ipam_environment.acc.id
}

resource "ipam_block" "acc" {
  name           = "acc-alloc-block"
  cidr           = "10.101.0.0/24"
  environment_id = ipam_environment.acc.id
  pool_id        = ipam_environment.acc.pool_ids[0]
}

resource "ipam_allocation" "acc" {
  name       = "acc-alloc"
  block_name = ipam_block.acc.name
  cidr       = "10.101.0.0/26"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ipam_allocation.acc", "id"),
					resource.TestCheckResourceAttr("ipam_allocation.acc", "name", "acc-alloc"),
					resource.TestCheckResourceAttr("ipam_allocation.acc", "block_name", "acc-alloc-block"),
					resource.TestCheckResourceAttr("ipam_allocation.acc", "cidr", "10.101.0.0/26"),
				),
			},
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_environment" "acc" {
  name = "acc-alloc-env"
  pools = [
    { name = "acc-pool", cidr = "10.0.0.0/8" }
  ]
}

data "ipam_pools" "acc" {
  environment_id = ipam_environment.acc.id
}

resource "ipam_block" "acc" {
  name           = "acc-alloc-block"
  cidr           = "10.101.0.0/24"
  environment_id = ipam_environment.acc.id
  pool_id        = ipam_environment.acc.pool_ids[0]
}

resource "ipam_allocation" "acc" {
  name       = "acc-alloc-updated"
  block_name = ipam_block.acc.name
  cidr       = "10.101.0.0/26"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ipam_allocation.acc", "name", "acc-alloc-updated"),
				),
			},
			{
				ResourceName:      "ipam_allocation.acc",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAllocationAutoResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint := os.Getenv("IPAM_ENDPOINT")
	token := os.Getenv("IPAM_TOKEN")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_environment" "acc" {
  name = "acc-auto-alloc-env"
  pools = [
    { name = "acc-pool", cidr = "10.0.0.0/8" }
  ]
}

data "ipam_pools" "acc" {
  environment_id = ipam_environment.acc.id
}

resource "ipam_block" "acc" {
  name           = "acc-auto-alloc-block"
  cidr           = "10.104.0.0/16"
  environment_id = ipam_environment.acc.id
  pool_id        = ipam_environment.acc.pool_ids[0]
}

resource "ipam_allocation" "acc" {
  name           = "acc-auto-alloc"
  block_name     = ipam_block.acc.name
  prefix_length  = 24
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ipam_allocation.acc", "id"),
					resource.TestCheckResourceAttr("ipam_allocation.acc", "name", "acc-auto-alloc"),
					resource.TestCheckResourceAttr("ipam_allocation.acc", "block_name", "acc-auto-alloc-block"),
					resource.TestCheckResourceAttrSet("ipam_allocation.acc", "cidr"),
					resource.TestCheckResourceAttrWith("ipam_allocation.acc", "cidr", func(value string) error {
						if value == "" {
							return fmt.Errorf("cidr must be set by API for auto allocation")
						}
						if !strings.HasSuffix(value, "/24") {
							return fmt.Errorf("expected /24 allocation, got %s", value)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      "ipam_allocation.acc",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccReservedBlockResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint := os.Getenv("IPAM_ENDPOINT")
	token := os.Getenv("IPAM_TOKEN")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_reserved_block" "acc" {
  name   = "acc-reserved"
  cidr   = "10.200.0.0/24"
  reason = "acceptance test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ipam_reserved_block.acc", "id"),
					resource.TestCheckResourceAttr("ipam_reserved_block.acc", "name", "acc-reserved"),
					resource.TestCheckResourceAttr("ipam_reserved_block.acc", "cidr", "10.200.0.0/24"),
					resource.TestCheckResourceAttr("ipam_reserved_block.acc", "reason", "acceptance test"),
					resource.TestCheckResourceAttrSet("ipam_reserved_block.acc", "created_at"),
				),
			},
			{
				ResourceName:      "ipam_reserved_block.acc",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDataSources(t *testing.T) {
	testAccPreCheck(t)
	endpoint := os.Getenv("IPAM_ENDPOINT")
	token := os.Getenv("IPAM_TOKEN")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(endpoint, token) + `
resource "ipam_environment" "acc" {
  name = "acc-ds-env"
  pools = [
    { name = "acc-pool", cidr = "10.0.0.0/8" }
  ]
}

data "ipam_pools" "acc" {
  environment_id = ipam_environment.acc.id
}

resource "ipam_block" "acc" {
  name           = "acc-ds-block"
  cidr           = "10.102.0.0/24"
  environment_id = ipam_environment.acc.id
  pool_id        = ipam_environment.acc.pool_ids[0]
}

resource "ipam_allocation" "acc" {
  name       = "acc-ds-alloc"
  block_name = ipam_block.acc.name
  cidr       = "10.102.0.0/26"
}

data "ipam_environment" "acc" {
  id = ipam_environment.acc.id
}

data "ipam_environments" "acc" {
  name = "acc-ds"
}

data "ipam_block" "acc" {
  id = ipam_block.acc.id
}

data "ipam_blocks" "acc" {
  environment_id = ipam_environment.acc.id
}

data "ipam_allocation" "acc" {
  id = ipam_allocation.acc.id
}

data "ipam_allocations" "acc" {
  block_name = ipam_block.acc.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.ipam_environment.acc", "name", "acc-ds-env"),
					resource.TestCheckResourceAttrPair("data.ipam_environment.acc", "id", "ipam_environment.acc", "id"),
					resource.TestCheckResourceAttr("data.ipam_block.acc", "name", "acc-ds-block"),
					resource.TestCheckResourceAttr("data.ipam_block.acc", "cidr", "10.102.0.0/24"),
					resource.TestCheckResourceAttr("data.ipam_allocation.acc", "name", "acc-ds-alloc"),
					resource.TestCheckResourceAttr("data.ipam_allocation.acc", "cidr", "10.102.0.0/26"),
				),
			},
		},
	})
}
