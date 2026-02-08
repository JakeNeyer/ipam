# Getting started

This guide introduces the main concepts and how to use the app.

## Concepts

- **Environments** — Logical groupings (e.g. Production, Staging). Each environment can contain multiple network blocks.
- **Network blocks** — CIDR ranges (e.g. `10.1.0.0/16`) that belong to an environment or are orphaned. They define a pool of IP addresses.
- **Allocations** — Subnets carved out of a block (e.g. `10.1.0.0/24`). They represent actual usage within a block.

## First steps

1. Open **Environments** and create an environment (e.g. "Production" or "Staging").
2. Open **Networks** and create a network block: choose a CIDR and assign it to an environment.
3. Create an allocation from a block to reserve a subnet (e.g. for a VPC or application).
4. Use the **Dashboard** to see utilization and the resource graph.
