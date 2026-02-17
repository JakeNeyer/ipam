# AWS integration — data model

The AWS integration syncs **VPC IPAM** resources into IPAM: pools, VPCs (as blocks), and subnets (as allocations). This page describes the high-level data model and how AWS concepts map to IPAM.

## Overview

- **AWS VPC IPAM** — AWS’s built-in IPAM lets you create **scopes**, pools, and **pool allocations** (e.g. VPC CIDRs). Subnets live inside VPCs.
- **IPAM (this app)** — Uses **Environment** → Pools → **Blocks** → **Allocations**. The integration maps AWS resources into this hierarchy so you can view and manage them in one place, and optionally push changes back to AWS (read-write).

## High-level mapping

| IPAM resource | AWS resource | Notes |
| -------------- | ------------ | ----- |
| **Pool** | **IPAM pool** | An AWS IPAM pool (possibly nested under another pool). Synced with its provisioned CIDR and external ID = `IpamPoolId`. Name in IPAM is the pool ID plus optional Name tag (e.g. `ipam-pool-xxx (My Pool)`). |
| **Block** | **VPC** (IPAM pool allocation) | A *pool allocation* of type VPC: the VPC’s primary CIDR. Each such allocation becomes one **block** in IPAM. External ID is the VPC ID. |
| **Allocation** | **Subnet** | Subnets belonging to a synced VPC are synced as **allocations** under the block that represents that VPC. External ID is the subnet ID. |

So the flow is: **IPAM scope → IPAM pools (→ Pools)**; **pool allocations (VPCs) → Blocks**; **VPC subnets → Allocations**.

## Hierarchy in AWS

1. **IPAM scope** — Top-level container in AWS. When creating an integration you can optionally restrict sync to one scope (scope ID). For read-write, a scope ID is required so the app can create new pools in AWS.
2. **IPAM pools** — Can be top-level or nested (child pools). Sync discovers all pools in the scope (or all scopes if scope ID is blank, read-only only) and creates/updates Pools in IPAM. Pool hierarchy (parent/child) is preserved via IPAM’s parent pool.
3. **Pool allocations** — Resources “allocated” from a pool. The integration only treats **VPC** allocations as **blocks**. Other allocation types (e.g. child IPAM pool, EIP) are skipped. Each VPC allocation becomes one block linked to the corresponding synced pool.
4. **Subnets** — Per-VPC. The integration lists subnets for each synced block (VPC) and creates **allocations** for them.

## Identifiers

- **Pool** — `ExternalID` = AWS `IpamPoolId`. Used to match on re-sync and for read-write push.
- **Block** — `ExternalID` = VPC ID (from the pool allocation `ResourceId`). Used to match and to fetch subnets.
- **Allocation** — `ExternalID` = subnet ID. Used to match on re-sync.

## Sync behavior

- **Read-only** — Sync only pulls from AWS. Creates/updates pools, blocks, and allocations in IPAM; removes them if they disappear in AWS (when the provider reports the current set). No scope ID required (can sync all scopes).
- **Read-write** — Same as read-only, plus: creating a pool, block, or allocation in IPAM can create it in AWS. Scope ID is required. Conflict resolution (cloud vs IPAM) decides who wins when the same resource exists in both.

Names in IPAM come from AWS where available: pool Name tag, allocation Description, subnet Name tag; otherwise the external ID is used.
