# Integrations

Integrations connect cloud providers to IPAM so you can **sync** pools, network blocks, and allocations from the cloud into your organization. Synced resources appear in the environment you choose when creating the integration. You can run sync on demand or on a schedule.

## Supported providers

- **AWS** — Sync AWS IPAM pools, VPCs (as blocks), and subnets (as allocations). Read-only or read-write. See [AWS data model](#docs/integrations/aws) for how AWS resources map to IPAM.
- **Azure** — Coming soon.
- **GCP** — Coming soon.

## Concepts

- **Integration** — A connection to one cloud provider (e.g. one AWS account/region + IPAM scope), tied to one environment. Each integration has a name, sync mode (read-only or read-write), and optional conflict resolution (cloud vs IPAM).
- **Sync** — Pulls the current set of pools, blocks, and allocations from the cloud and creates or updates matching resources in IPAM. Optionally (read-write) pushes changes from IPAM back to the cloud.
- **Environment** — Synced pools and blocks are attached to the environment you select when creating the integration. Use the same environment as your non-synced pools, or a dedicated one (e.g. “AWS Production”).

## Data model mapping

Each provider maps its own resources to IPAM’s hierarchy:

| IPAM       | Purpose                          |
| ---------- | -------------------------------- |
| **Pool**   | Top-level CIDR container (e.g. AWS IPAM pool). |
| **Block**  | CIDR range (e.g. VPC in AWS).    |
| **Allocation** | Subnet carved from a block (e.g. AWS subnet). |

Provider-specific mapping and identifiers are described on each integration’s page:

- [AWS](#docs/integrations/aws) — IPAM pools → Pools; VPCs → Blocks; Subnets → Allocations.

## Where to configure

Use **Integrations** in the app to add, edit, sync, or remove connections. Setup and authentication (e.g. AWS credential chain, IAM permissions) are described in the add-integration flow and in each provider’s data model page.
