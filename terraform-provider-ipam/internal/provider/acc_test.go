package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used for acceptance tests.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ipam": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck skips acceptance tests if IPAM_ENDPOINT or IPAM_TOKEN are not set.
// Token must not contain double quotes (") to avoid breaking HCL config.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping acceptance test in short mode")
	}
	if v := os.Getenv("IPAM_ENDPOINT"); v == "" {
		t.Skip("set IPAM_ENDPOINT to run acceptance tests")
	}
	if v := os.Getenv("IPAM_TOKEN"); v == "" {
		t.Skip("set IPAM_TOKEN to run acceptance tests")
	}
}

func testAccProviderConfig(endpoint, token string) string {
	return `
provider "ipam" {
  endpoint = "` + endpoint + `"
  token   = "` + token + `"
}
`
}
