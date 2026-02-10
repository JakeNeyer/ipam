# Getting started

This guide introduces the main concepts and how to use the app. IPAM helps you track IP address space by organizing it into environments, blocks, and allocations so you can see what you have, what is in use, and where to allocate next.

## Concepts

- **Environments** — Logical groupings (e.g. Production, Staging, Development). Each environment can contain multiple network blocks. Use environments to separate IP space by team, stage, or region.
- **Network blocks** — CIDR ranges (e.g. `10.1.0.0/16`) that belong to an environment or are orphaned. They define a pool of IP addresses. Blocks are the top-level ranges you subdivide into allocations.
- **Allocations** — Subnets carved out of a block (e.g. `10.1.0.0/24`). They represent actual usage—a VPC, application, or region. The app tracks utilization so you can see how full each block is and avoid overlaps.

## First steps

1. Open **Environments** and create an environment (e.g. "Production" or "Staging").
2. Open **Networks** and create a network block: choose a CIDR and assign it to an environment. The CIDR wizard can suggest a range that does not overlap existing blocks.
3. Create an allocation from a block to reserve a subnet (e.g. for a VPC or application). The wizard can suggest the next available range within the block.
4. Use the **Dashboard** to see utilization and the resource graph. From there you can jump to Networks or Environments by clicking a resource.
