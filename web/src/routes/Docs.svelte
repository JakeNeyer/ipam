<script>
  import DocsViewer from '../lib/DocsViewer.svelte'
  import overviewMd from '../docs/overview.md?raw'
  import gettingStartedMd from '../docs/getting-started.md?raw'
  import dashboardMd from '../docs/dashboard.md?raw'
  import environmentsMd from '../docs/environments.md?raw'
  import networksMd from '../docs/networks.md?raw'
  import commandPaletteMd from '../docs/command-palette.md?raw'

  export let currentPage = ''

  const PAGES = [
    { id: '', label: 'Overview' },
    { id: 'getting-started', label: 'Getting started' },
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'environments', label: 'Environments' },
    { id: 'networks', label: 'Networks' },
    { id: 'command-palette', label: 'Command palette' },
  ]

  const CONTENT = {
    '': overviewMd,
    'getting-started': gettingStartedMd,
    'dashboard': dashboardMd,
    'environments': environmentsMd,
    'networks': networksMd,
    'command-palette': commandPaletteMd,
  }

  $: normalizedPage = PAGES.some((p) => p.id === currentPage) ? currentPage : ''
  $: markdownContent = CONTENT[normalizedPage] ?? ''
</script>

<div class="docs-full">
  <aside class="docs-sidebar">
    <a href="#" class="docs-back" onclick={(e) => { e.preventDefault(); window.location.hash = '' }}>← Back to IPAM</a>
    <h2 class="docs-sidebar-title">User guide</h2>
    <nav class="docs-nav" aria-label="Documentation">
      <ul>
        {#each PAGES as page}
          <li>
            <a
              href="#docs{page.id ? '/' + page.id : ''}"
              class="docs-nav-link"
              class:active={normalizedPage === page.id}
            >
              {page.label}
            </a>
          </li>
        {/each}
      </ul>
    </nav>
    <p class="docs-sidebar-note">
      API reference is available from <strong>Settings → API docs</strong>.
    </p>
  </aside>
  <article class="docs-content">
    <DocsViewer content={markdownContent} />
  </article>
</div>

<style>
  .docs-full {
    display: flex;
    gap: 2rem;
    width: 100%;
    min-height: 100vh;
    height: 100vh;
    padding: 1.5rem 1.5rem 2rem;
    box-sizing: border-box;
    background: var(--bg);
    color: var(--text);
    overflow: auto;
  }
  .docs-sidebar {
    flex-shrink: 0;
    width: 12rem;
    padding: 0.5rem 0;
    border-right: 1px solid var(--border);
  }
  .docs-back {
    display: block;
    margin-bottom: 1rem;
    font-size: 0.9rem;
    color: var(--accent);
    text-decoration: none;
  }
  .docs-back:hover {
    text-decoration: underline;
  }
  .docs-sidebar-title {
    margin: 0 0 0.75rem 0;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
  }
  .docs-nav ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .docs-nav li {
    margin: 0;
  }
  .docs-nav-link {
    display: block;
    padding: 0.35rem 0;
    font-size: 0.9rem;
    color: var(--text-muted);
    text-decoration: none;
    border-radius: var(--radius);
  }
  .docs-nav-link:hover {
    color: var(--text);
  }
  .docs-nav-link.active {
    color: var(--accent);
    font-weight: 500;
  }
  .docs-sidebar-note {
    margin: 1.5rem 0 0 0;
    padding-top: 1rem;
    border-top: 1px solid var(--border);
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .docs-content {
    flex: 1;
    min-width: 0;
  }
</style>
