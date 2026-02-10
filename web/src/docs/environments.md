# Environments

The Environments page lists all of your environments. Environments are logical groupings (e.g. Production, Staging, Development) that help you organize network blocks. Each environment can contain multiple blocks; blocks not assigned to any environment appear as orphaned on the Networks page. Use this page to create environments, see which blocks belong to each, and edit or delete environments.

<p class="docs-screenshot">
<img src="/images/environments-light.png" alt="Environments page (light mode)" class="screenshot-light" />
<img src="/images/environments-dark.png" alt="Environments page (dark mode)" class="screenshot-dark" />
</p>

## Viewing environments

The table shows each environment’s name and ID. Click a row to expand it and see its network blocks, including CIDR, total IPs, used IPs, and any allocations within each block. Expanding a block shows allocation details so you can see the full hierarchy (environment → block → allocation) in one place. This view is useful for understanding how much space each environment has and how it’s used.

## Creating an environment

Click **Create environment** to add a new environment. Give it a name (e.g. "Production" or "Staging"). You can optionally create an initial network block at the same time by providing a block name and CIDR. You can also start from the command palette (⌘K or Ctrl+K) by choosing "Create environment"; the Environments page opens with the create form ready.

## Editing and deleting

Use the actions menu (⋮) on a row to **Edit** the environment name or **Delete** the environment. Editing only changes the display name. Deleting an environment removes the environment and all blocks that belong to it; those blocks are not moved to another environment. Use delete when you are consolidating or retiring an environment and have already moved or removed its blocks as needed.

## Opening a specific environment

If you search for an environment in the command palette (⌘K or Ctrl+K) and select it, the Environments page opens showing only that environment, with a link to return to "All environments". This makes it easy to jump to one environment when you have many.
