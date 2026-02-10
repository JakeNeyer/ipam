# Dashboard

The Dashboard is your home view. It gives you a high-level picture of all IPAM resources and how they are used, so you can quickly see capacity, utilization, and relationships without opening each page.

<p class="docs-screenshot">
<img src="/images/dashboard-light.png" alt="Dashboard (light mode)" class="screenshot-light" />
<img src="/images/dashboard-dark.png" alt="Dashboard (dark mode)" class="screenshot-dark" />
</p>

## Summary statistics

At the top of the Dashboard you'll see counts for environments, network blocks, and allocations, plus total IPs, used IPs, and overall utilization. These numbers give you an at-a-glance view of how much IP space you have and how much is in use across the whole system.

## Orphaned blocks

Blocks that are not assigned to any environment appear as **orphaned**. The Dashboard shows an alert when orphaned blocks exist, with a link to the Networks page so you can assign them to an environment or manage them. Keeping blocks assigned helps you organize and filter by environment.

## Block utilization

A list of each network block with a progress bar shows what percentage of its IP space is allocated. Use this to spot blocks that are under-used (plenty of room for new allocations) or nearly full (you may need to plan a new block or reclaim space). Sorting by utilization helps you prioritize where to allocate next.

## Resource graph

The resource graph shows how environments, blocks, and allocations connect. Each environment appears with its blocks underneath, and each block shows its allocations. Click an environment to open the Networks page filtered by that environment; click a block to focus on it; or click an allocation to filter the Networks view to that allocation. Orphaned blocks appear under an "Orphaned" node, and reserved blocks (if any) under "Reserved". The graph helps you see the full hierarchy at once.

## Export CSV

Use **Export CSV** to download a spreadsheet of your blocks and allocations. Use the export for reporting, auditing, or feeding other tools. The file includes block and allocation names, CIDRs, and usage so you can work with the data outside the app.
