# Terraform example fixtures

This directory and the parent `hack/` scripts provide everything needed to run the Terraform example against a live IPAM instance.

## Prerequisites

- **Go** – to build the IPAM server and the provider
- **Terraform CLI** – 1.x
- **Running IPAM server** – see below

## 1. Start the IPAM server

From the **repository root** (not this directory):

```bash
# In-memory store (no database)
go run .

# Or with Postgres
DATABASE_URL="postgres://user:pass@localhost:5432/ipam?sslmode=disable" go run .
```

Server listens at `http://localhost:8011`. Ensure setup is complete and you have an admin user (visit the UI once if needed).

## 2. Create an API token

1. Open `http://localhost:8011` in a browser and log in as admin.
2. Go to **Admin → API tokens**.
3. Create a token and copy it. The token **must not contain double quotes (`"`)** so it does not break HCL.

## 3. Set fixture variables

Either use the example env file or export variables manually.

### Option A: env.example (recommended for the script)

```bash
# From repo root
cp hack/terraform-example/env.example hack/terraform-example/.env
# Edit .env and set IPAM_TOKEN=your-copied-token

# Load and run (from repo root)
set -a && source hack/terraform-example/.env && set +a
./hack/terraform-example.sh plan
./hack/terraform-example.sh apply
./hack/terraform-example.sh destroy
```

### Option B: Export manually

```bash
export IPAM_ENDPOINT=http://localhost:8011
export IPAM_TOKEN=your-api-token
./hack/terraform-example.sh plan
./hack/terraform-example.sh apply
./hack/terraform-example.sh destroy
```

### Option C: terraform.tfvars (for direct Terraform commands)

If you run `terraform` yourself (e.g. after `terraform init -plugin-dir=...`):

```bash
cp hack/terraform-example/terraform.tfvars.example hack/terraform-example/terraform.tfvars
# Edit terraform.tfvars and set ipam_token

cd hack/terraform-example
# After init with plugin-dir (see terraform-example.sh for plugin layout):
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
terraform destroy -var-file=terraform.tfvars
```

## 4. What the example creates

The example `main.tf` declares:

- **ipam_environment.hack** – name `tf-hack-env`
- **ipam_block.hack** – name `tf-hack-block`, CIDR `10.200.0.0/24`, in that environment
- **ipam_allocation.hack** – name `tf-hack-alloc`, CIDR `10.200.0.0/26`, in that block (explicit CIDR)
- **ipam_allocation.hack_auto** – name `tf-hack-alloc-auto`, auto-allocated /26 via `/api/allocations/auto` (prefix_length)

Run `terraform destroy` when finished to remove these resources from IPAM.

## Fixture files reference

| File | Purpose |
|------|---------|
| `env.example` | Template for `.env`; set `IPAM_ENDPOINT` and `IPAM_TOKEN` for the hack script. |
| `terraform.tfvars.example` | Template for `terraform.tfvars`; set `ipam_endpoint` and `ipam_token` for direct `terraform` commands. |
| `.env` | Local env (gitignored). Copy from `env.example` and fill in. |
| `terraform.tfvars` | Local tfvars (gitignored). Copy from `terraform.tfvars.example` and fill in. |
