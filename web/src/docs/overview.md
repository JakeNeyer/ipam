# IPAM User Guide

IPAM is an IP Address Management application for tracking **environments**, pools, **network blocks**, and **allocations**. Hierarchy: **Environment** → Pools → **Blocks** → **Allocations**. Pools are CIDR ranges that blocks in an environment draw from; every environment has at least one pool. The app organizes enterprise networks in one place and works with infrastructure tooling such as Terraform.

## Data model

The IPAM data model has four main resource types, arranged in a hierarchy:

- **Environment** — A logical grouping (e.g. Production, Staging) that owns one or more pools. Environments are the top-level container.

- **Pool** — A CIDR range (e.g. `10.0.0.0/8`) that defines the address space for an environment. Every environment has at least one pool. Network blocks in that environment must have a CIDR that lies entirely within one of the environment’s pools.

- **Network block** — A CIDR range (e.g. `10.0.0.0/16`) that represents a block of IP addresses. Each block is assigned to an environment and optionally to a specific pool within that environment. These could be VLANs, AWS VPCs, Azure VNETs, etc.

- **Allocation** — A subnet carved out of a block (e.g. `10.0.1.0/24`) that represents actual usage.

**Reserved blocks** are CIDR ranges (e.g. `10.255.0.0/16`) that are set aside so they cannot be used for network blocks or allocations. Use them for future expansion, DMZ space, or any ranges you want to keep off-limits. Reserved ranges appear on the Networks page and the Dashboard resource graph. Only admin users can manage reserved blocks.

**Orphaned blocks** are network blocks that are not assigned to any environment or pool. They can be created when you add a block without selecting a pool; they still belong to an organization and can contain allocations. Orphaned blocks appear in a separate “Orphaned” section on the Dashboard resource graph and can be filtered on the Networks page. Assigning a block to a pool moves it into an environment and removes it from the orphaned set.

## Sections

- [Getting started](#docs/getting-started) — Docker setup, REST API, and Terraform examples.
- [Environments](#docs/environments) — Manage environments and their pools.
- [Networks](#docs/networks) — Manage pools, network blocks, and allocations.
- [Integrations](#docs/integrations) — Connect cloud providers and sync pools, blocks, and allocations. See [AWS](#docs/integrations/aws) for the AWS data model.
- [Command palette](#docs/command-palette) — Quick search and navigation (`⌘K` / `Ctrl+K`).
- [CIDR wizard](#docs/cidr-wizard) — Suggested CIDR ranges for blocks (based on pool) and allocations.
- [Network Advisor](#docs/network-advisor) — Plan and generate an IP strategy.
- [Subnet calculator](#docs/subnet-calculator) — Explore subnets without creating resources.
- [Reserved blocks](#docs/reserved-blocks) — Reserved CIDR ranges (admin only).
- [Admin](#docs/admin) — Manage users, organizations, signup links, and API tokens (admin only).
