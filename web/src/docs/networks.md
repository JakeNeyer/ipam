# Networks

The Networks page shows all network blocks and allocations. You can filter by environment and create new blocks or allocations.

## Environment filter

Use the **Environment** dropdown to show only blocks in a given environment, or "All" to see every block (including orphaned ones).

## Network blocks

The blocks table lists name, environment, CIDR, total IPs, used, available, and usage percentage. Use the actions menu (⋮) to edit or delete a block. Click **Create block** to add a new block: you can use the CIDR wizard to get a suggested range that doesn't overlap existing blocks in the chosen environment, or type a CIDR manually.

![CIDR wizard for creating a new network block](/images/cidr-wizard.png)

## Allocations

The allocations table lists each subnet (name, block, CIDR). Create allocations with **Create allocation**: pick a block, then either use a suggested CIDR or enter one manually. Allocations must fall within their block's range and not overlap other allocations in the same block.

## Opening a specific block

When you search for a block or allocation in the command palette and select a result, the Networks page opens with that block in focus (and the block list scoped accordingly).
