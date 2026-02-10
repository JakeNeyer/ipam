# Hack scripts

Helper scripts for development and testing.

## seed.sh

Seeds a running IPAM API with test data (environments, blocks, allocations, reserved blocks).

```bash
./hack/seed.sh [BASE_URL] [API_TOKEN]
# Example: ./hack/seed.sh http://localhost:8011 your-api-token
```

Requires: curl, python3. Create an API token in the IPAM UI under Admin > API tokens.

## terraform-example.sh

Runs Terraform (plan, apply, or destroy) against a running IPAM instance using the local `terraform-provider-ipam` from this repo.

```bash
IPAM_ENDPOINT=http://localhost:8011 IPAM_TOKEN=your-token ./hack/terraform-example.sh [plan|apply|destroy]
```

- **plan** (default) – shows what would be created (environment, block, allocation).
- **apply** – creates the resources in IPAM.
- **destroy** – removes the resources from IPAM.

Requires: Go, Terraform CLI. The script builds the provider from `terraform-provider-ipam/` and uses a dev override so no provider install is needed.

### Terraform fixtures

The directory `hack/terraform-example/` includes fixtures for running the script against a live IPAM server:

- **env.example** – Copy to `.env` and set `IPAM_ENDPOINT` and `IPAM_TOKEN`; then `source .env` and run the script.
- **terraform.tfvars.example** – Copy to `terraform.tfvars` for direct `terraform plan/apply/destroy -var-file=terraform.tfvars`.
- **FIXTURES.md** – Full instructions: start IPAM server, create API token, set variables, run plan/apply/destroy.
