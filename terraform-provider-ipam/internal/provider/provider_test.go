package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProvider(t *testing.T) {
	p := New("test")()
	if p == nil {
		t.Fatal("New returned nil")
	}
}

// TestAccProviderConfig runs a minimal config that requires a live IPAM server.
func TestAccProviderConfig(t *testing.T) {
	testAccPreCheck(t)
	endpoint := os.Getenv("IPAM_ENDPOINT")
	token := os.Getenv("IPAM_TOKEN")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(endpoint, token) + `
data "ipam_environments" "all" {}
`,
			},
		},
	})
}
