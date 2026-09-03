# Networks

The Networks page is where you manage pools, network blocks, and allocations day-to-day. Hierarchy: Environment → Pool → **Blocks** → **Allocations**.

<p class="docs-screenshot">
<img src="/images/networks-light.png" alt="Networks page (light mode)" class="screenshot-light" />
<img src="/images/networks-dark.png" alt="Networks page (dark mode)" class="screenshot-dark" />
</p>

- **View** — All, Orphaned, or Unused. **Scope** narrows to an environment, a pool, or blocks with no pool.
- **Drill in** — Click a pool, environment, block, or allocation name to focus that part of the hierarchy. Use **← All networks** to clear the scope.
- **Create** — Pool, block, and allocation actions are in the page header. Create forms pre-fill from the current scope when they can.
- **Environment pools** — Pools in the same environment cannot overlap. Nested child pools are indented in the table.
- **Network blocks** — CIDR ranges assigned to a pool. The CIDR wizard suggests non-overlapping ranges.
- **Allocations** — Subnets carved out of blocks. Allocations must fit within their block and cannot overlap.
- **Quick access** — Search for a pool, block, or allocation in the command palette (`⌘K` / `Ctrl+K`) to jump directly to it.
