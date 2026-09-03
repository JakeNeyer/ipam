# Environments

Environments are logical groupings (e.g. `Production`, `Staging`) that organize your network blocks. Each environment has at least one pool — a CIDR range that blocks in that environment draw from (hierarchy: **Environment** → Pools → **Blocks** → **Allocations**).

<p class="docs-screenshot">
<img src="/images/environments-light.png" alt="Environments page (light mode)" class="screenshot-light" />
<img src="/images/environments-dark.png" alt="Environments page (dark mode)" class="screenshot-dark" />
</p>

- **View** — See all environments. Click a name to open that environment’s pools. Click a pool to focus it and its blocks. Use **← All environments** to go back.
- **Create** — Add an environment with a name and a required pool (pool name + CIDR). Every environment must have a pool.
- **Pools** — Add, edit, or delete pools from the environment view. Nested child pools are indented. Pools are CIDR ranges that blocks draw from; block CIDRs must be contained in a pool’s CIDR. Pools in the same environment cannot overlap.
- **Blocks** — Blocks without a pool appear on the environment view. Open a pool to see its blocks. Block names go to Networks. **Open in Networks** focuses the same environment or pool there.
- **Edit / Delete** — Rename an environment or delete it (and its pools and blocks) from the actions menu.
- **Quick access** — Search for an environment or pool in the command palette (`⌘K` / `Ctrl+K`) to jump straight to it.
