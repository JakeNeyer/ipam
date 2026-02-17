# Environments

Environments are logical groupings (e.g. `Production`, `Staging`) that organize your network blocks. Each environment has at least one pool — a CIDR range that blocks in that environment draw from (hierarchy: **Environment** → Pools → **Blocks** → **Allocations**).

<p class="docs-screenshot">
<img src="/images/environments-light.png" alt="Environments page (light mode)" class="screenshot-light" />
<img src="/images/environments-dark.png" alt="Environments page (dark mode)" class="screenshot-dark" />
</p>

- **View** — See all environments; expand a row to view CIDR pools and **Blocks without pool**, then expand a pool or block to see its blocks or allocations.
- **Create** — Add an environment with a name and a required pool (pool name + CIDR). Every environment must have a pool.
- Pools — In the expanded row, add, edit, or delete pools. Pools are CIDR ranges that blocks in that environment draw from; block CIDRs must be contained in a pool’s CIDR. Pools in the same environment cannot overlap.
- **Edit / Delete** — Rename an environment or delete it (and its pools and blocks) from the actions menu.
- **Quick access** — Search for an environment or pool in the command palette (`⌘K` / `Ctrl+K`) to jump straight to it.
