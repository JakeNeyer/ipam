#!/usr/bin/env bash
# Run Terraform (plan or apply) against a running IPAM instance using the local provider.
#
# Usage: ./hack/terraform-example.sh [plan|apply|destroy]
#   plan   - terraform plan (default; shows what would be created)
#   apply  - terraform apply (creates resources in IPAM)
#   destroy - terraform destroy (removes resources from IPAM)
#
# Environment:
#   IPAM_ENDPOINT - Base URL of the IPAM API (e.g. http://localhost:8011). Required.
#   IPAM_TOKEN    - API token (create in Admin > API tokens). Required.
#
# Requires: Go, Terraform CLI. The script builds the provider from terraform-provider-ipam/
# and uses a dev override so no provider install is needed.
#
# Example:
#   IPAM_ENDPOINT=http://localhost:8011 IPAM_TOKEN=your-token ./hack/terraform-example.sh plan
#   IPAM_ENDPOINT=http://localhost:8011 IPAM_TOKEN=your-token ./hack/terraform-example.sh apply
#   IPAM_ENDPOINT=http://localhost:8011 IPAM_TOKEN=your-token ./hack/terraform-example.sh destroy

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROVIDER_DIR="${REPO_ROOT}/terraform-provider-ipam"
EXAMPLE_DIR="${REPO_ROOT}/hack/terraform-example"
ACTION="${1:-plan}"

if [ "$ACTION" != "plan" ] && [ "$ACTION" != "apply" ] && [ "$ACTION" != "destroy" ]; then
  echo "Usage: $0 [plan|apply|destroy]"
  exit 1
fi

if [ -z "$IPAM_ENDPOINT" ] || [ -z "$IPAM_TOKEN" ]; then
  echo "Error: IPAM_ENDPOINT and IPAM_TOKEN must be set."
  echo "Example: IPAM_ENDPOINT=http://localhost:8011 IPAM_TOKEN=your-token $0 plan"
  exit 1
fi

# Build the provider binary (must satisfy main.tf version ">= 0.1")
echo "Building IPAM provider..."
if ! (cd "$PROVIDER_DIR" && go build -o terraform-provider-ipam_0.1.0 .); then
  echo "Failed to build provider. Is Go installed and terraform-provider-ipam present?"
  exit 1
fi

# Terraform -plugin-dir expects a filesystem mirror layout:
#   <plugin-dir>/registry.terraform.io/<namespace>/<name>/<version>/<platform>/terraform-provider-<name>_<version>
# Create that structure so init finds the provider without querying the registry.
PROVIDER_VERSION="0.1.0"
PLUGIN_PLATFORM="$(go env GOOS)_$(go env GOARCH)"
PLUGIN_ROOT="${EXAMPLE_DIR}/.terraform.d/plugins"
PLUGIN_PATH="${PLUGIN_ROOT}/registry.terraform.io/jakeneyer/ipam/${PROVIDER_VERSION}/${PLUGIN_PLATFORM}"
mkdir -p "$PLUGIN_PATH"
cp "$PROVIDER_DIR/terraform-provider-ipam_${PROVIDER_VERSION}" "$PLUGIN_PATH/"

export TF_VAR_ipam_endpoint="$IPAM_ENDPOINT"
export TF_VAR_ipam_token="$IPAM_TOKEN"

cd "$EXAMPLE_DIR"
# When using -plugin-dir with a local build, the lock file may have checksums from a different
# platform or an older build; remove it so init records the current binary and avoids mismatch errors.
rm -f .terraform.lock.hcl
echo "Running terraform init..."
# -plugin-dir points at a directory with registry layout so Terraform finds the provider locally (no registry query)
terraform init -plugin-dir="$PLUGIN_ROOT"
echo "Running terraform $ACTION..."
terraform "$ACTION"
