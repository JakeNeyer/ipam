# Terraform Provider for IPAM

This [Terraform](https://www.terraform.io) provider manages resources in an [IPAM](https://github.com/JakeNeyer/ipam) instance: environments, network blocks, allocations, and reserved blocks. It uses the IPAM HTTP API with Bearer token authentication.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.22 (for building from source)
- IPAM server with API token (create tokens in the IPAM UI under **Admin**)

## Authentication

The provider uses an **API token** (Bearer token). Create a token in the IPAM web UI: **Admin** → **API tokens** → **Create token**. Use the token value in the provider block (keep it secret, e.g. via environment variables).

```hcl
provider "ipam" {
  endpoint = "https://ipam.example.com"  # Base URL of the IPAM server (no trailing slash)
  token   = var.ipam_token               # Or env: TF_VAR_ipam_token
}
```

## Building and Installing

From the `terraform-provider-ipam` directory:

```bash
go build -o terraform-provider-ipam
```

Install for local use (Terraform 1.0+):

- Create `~/.terraform.d/plugins/<host>/<namespace>/ipam/<version>/<arch>/` (e.g. `~/.terraform.d/plugins/localhost/jakeneyer/ipam/0.1.0/darwin_arm64/`).
- Copy the binary there and name it `terraform-provider-ipam_<version>` (e.g. `terraform-provider-ipam_0.1.0`).

Or use a [development override](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider-versioning#development-overrides):

```bash
# In your Terraform config directory
export TF_CLI_CONFIG_FILE=dev_overrides.tfrc
# dev_overrides.tfrc:
# provider_installation {
#   dev_overrides {
#     "jakeneyer/ipam" = "/path/to/terraform-provider-ipam"
#   }
#   direct {}
# }
```

## Resources

| Resource | Description |
|----------|-------------|
| `ipam_environment` | Create and manage an environment (e.g. prod, staging). |
| `ipam_reserved_block` | Reserve a CIDR range so it cannot be used as a block or allocation (admin only). Changing `cidr` forces replacement. |
| `ipam_block` | Create and manage a network block (CIDR assigned to an environment). Changing `cidr` forces replacement. |
| `ipam_allocation` | Create and manage an allocation (subnet within a block). Changing `block_name` or `cidr` forces replacement. |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `ipam_environment` | Fetch a single environment by ID. |
| `ipam_environments` | List environments with optional `name` filter. |
| `ipam_reserved_block` | Fetch a single reserved block by ID (admin only). |
| `ipam_reserved_blocks` | List all reserved blocks (admin only). |
| `ipam_block` | Fetch a single network block by ID. |
| `ipam_blocks` | List blocks with optional `name`, `environment_id`, `orphaned_only` filters. |
| `ipam_allocation` | Fetch a single allocation by ID. |
| `ipam_allocations` | List allocations with optional `name`, `block_name` filters. |

## Example

```hcl
terraform {
  required_providers {
    ipam = {
      source  = "jakeneyer/ipam"
      version = "~> 0.1"
    }
  }
}

provider "ipam" {
  endpoint = "http://localhost:8080"
  token   = var.ipam_api_token
}

resource "ipam_environment" "prod" {
  name = "prod"
}

resource "ipam_block" "prod_vpc" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.prod.id
}

resource "ipam_allocation" "region_a" {
  name       = "region-a"
  block_name = ipam_block.prod_vpc.name
  cidr       = "10.0.0.0/16"
}

data "ipam_environments" "all" {}

output "environment_ids" {
  value = [for e in data.ipam_environments.all.environments : e.id]
}
```

## Documentation

Documentation for the provider, resources, and data sources lives in `docs/` and follows [HashiCorp's Terraform provider documentation](https://developer.hashicorp.com/terraform/registry/providers/docs) practices:

- **Schema descriptions** — All provider, resource, and data source attributes have `MarkdownDescription` in code for tooling and IDE help.
- **Examples** — Example configurations are in `examples/provider/`, `examples/resources/<type>/`, and `examples/data-sources/<type>/`.
- **Import** — Each resource has an `examples/resources/<type>/import.sh` with the `terraform import` command.
- **Generated docs** — You can regenerate `docs/` from the provider schema and examples using [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs):

  ```bash
  cd tools && go mod tidy && go generate .
  ```

  This requires [Terraform](https://developer.hashicorp.com/terraform/downloads) installed and a buildable provider. The `docs/` in this repo are hand-written to match the generated format so they work without running the generator.

## Development

- Run unit tests (no live server):  
  `go test ./internal/client/ -v` and `go test ./internal/provider/ -v -short`
- Run acceptance tests (requires a running IPAM server and admin API token):  
  `IPAM_ENDPOINT=http://localhost:8011 IPAM_TOKEN=your-token go test -v -count=1 -run TestAcc ./internal/provider/...`

  **Fixtures:** Copy `env.example` to `.env`, set `IPAM_TOKEN`, then `source .env` and run the test command above. Start the IPAM server from the repo root with `go run .` (or use the hack/terraform-fixtures flow; see `hack/terraform-example/FIXTURES.md`).

  Acceptance tests cover:
  - **TestAccProviderConfig** — provider config and `ipam_environments` data source
  - **TestAccEnvironmentResource** — create, update name, import
  - **TestAccBlockResource** — create block in environment, update name, import
  - **TestAccAllocationResource** — create allocation in block, update name, import
  - **TestAccReservedBlockResource** — create reserved block (admin token required), import
  - **TestAccDataSources** — single and list data sources for environment, block, allocation

  Ensure `IPAM_TOKEN` does not contain double quotes (`"`) to avoid breaking HCL. Reserved-block tests require an admin token.

## References

- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [Terraform Plugin Framework tutorials](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework)
