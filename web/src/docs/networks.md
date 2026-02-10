# Networks

The Networks page is where you view and manage network blocks and their allocations. Use it to create blocks (CIDR ranges) and allocations (subnets), filter by environment or block, and edit or delete resources. It is the main workspace for day-to-day IP planning and allocation.

<p class="docs-screenshot">
<img src="/images/networks-light.png" alt="Networks page (light mode)" class="screenshot-light" />
<img src="/images/networks-dark.png" alt="Networks page (dark mode)" class="screenshot-dark" />
</p>

## Environment filter

Use the **Environment** dropdown to show only blocks in a given environment, or choose **All** to see every block (including orphaned ones). The **Block** and **Allocation** dropdowns let you narrow further to a single block or allocation. Filtering keeps the list manageable when you have many resources and helps you focus on one environment or block at a time.

## Network blocks

The blocks table lists each block’s name, environment, CIDR, total IPs, used, available, and usage percentage. Blocks are the top-level CIDR ranges you assign to an environment; they define the pool of addresses you can subdivide into allocations. Use the actions menu (⋮) on a row to **Edit** (change name or environment) or **Delete** a block. Click **Create block** to add a new block: choose an environment, then use the CIDR wizard to get a suggested range that doesn’t overlap existing blocks or reserved ranges in that environment, or enter a CIDR manually. The app prevents overlapping blocks and blocks that conflict with reserved ranges.

## Allocations

The allocations table lists each subnet (name, block, CIDR). Allocations are the subnets you carve out of a block for actual use (e.g. a VPC, application, or region). Click **Create allocation** to add one: pick a block, then use a suggested CIDR from the wizard or enter one manually. Allocations must fall entirely within their block and cannot overlap other allocations in the same block. The wizard suggests the next available range to make it easy to fill a block without gaps or overlaps.

## Finding a specific block or allocation

When you search for a block or allocation in the command palette (⌘K or Ctrl+K) and select a result, the Networks page opens with that block in focus and the block list scoped accordingly. You can also open a block or allocation by clicking it on the Dashboard resource graph.
