# Networks

The Networks page is where you manage network blocks and allocations day-to-day.

<p class="docs-screenshot">
<img src="/images/networks-light.png" alt="Networks page (light mode)" class="screenshot-light" />
<img src="/images/networks-dark.png" alt="Networks page (dark mode)" class="screenshot-dark" />
</p>

- **Filter** — Narrow the view by environment, block, or allocation using the dropdowns.
- **Network blocks** — Create, edit, and delete CIDR ranges assigned to environments. The CIDR wizard suggests non-overlapping ranges.
- **Allocations** — Carve subnets out of blocks (e.g. `/24` within a `/16`). Allocations must fit within their block and cannot overlap. The wizard suggests the next available range.
- **Quick access** — Search for a block or allocation in the command palette (`⌘K` / `Ctrl+K`) to jump directly to it.
