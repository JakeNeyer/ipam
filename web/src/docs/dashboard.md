# Dashboard

The Dashboard gives you a high-level view of all IPAM resources and how they are used.

![Dashboard showing metrics, block utilization, and resource graph](/images/dashboard.png)

## Summary statistics

At the top you'll see counts for environments, network blocks, and allocations, plus total IPs, used IPs, and overall utilization. Use these to quickly see capacity and usage.

## Orphaned blocks

Blocks that are not assigned to any environment appear as **orphaned**. The Dashboard shows an alert when orphaned blocks exist, with a link to the Networks page to view or assign them.

## Block utilization

A list of each network block with a progress bar shows what percentage of its IP space is allocated. This helps you spot under- or over-used blocks.

## Resource graph

The resource graph shows how environments, blocks, and allocations are connected. You can click an environment to go to Networks filtered by that environment, or click a block to focus on it on the Networks page. Orphaned blocks appear under an "Orphaned" node.

## Export CSV

Use **Export CSV** to download a spreadsheet of your blocks and allocations for reporting or auditing.
