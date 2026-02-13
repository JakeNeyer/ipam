# Networks

The Networks page is where you manage **pools**, network blocks, and allocations day-to-day. Hierarchy: **Pool → Blocks → Allocations** (pools belong to an environment).

<p class="docs-screenshot">
<img src="/images/networks-light.png" alt="Networks page (light mode)" class="screenshot-light" />
<img src="/images/networks-dark.png" alt="Networks page (dark mode)" class="screenshot-dark" />
</p>

- **Filter** — The **Environment** dropdown is general only: All, Orphaned only, or Unused only. Use the **Pool** dropdown to filter by environment (e.g. “Environment: Prod”) or by a specific pool. You can also filter by **block** or **allocation**.
- **Environment pools** — When you select an environment in the Pool filter (e.g. “Environment: Prod”), you can view and manage that environment’s pools. Create, edit, or delete pools. Pools in the same environment cannot overlap.
- **Network blocks** — Create, edit, and delete CIDR ranges assigned to environments. Optionally assign a block to a pool; the block’s CIDR must be contained in the pool’s CIDR. The CIDR wizard suggests non-overlapping ranges.
- **Allocations** — Carve subnets out of blocks (e.g. `/24` within a `/16`). Allocations must fit within their block and cannot overlap. The wizard suggests the next available range.
- **Quick access** — Search for a pool, block, or allocation in the command palette (`⌘K` / `Ctrl+K`) to jump directly to it.
