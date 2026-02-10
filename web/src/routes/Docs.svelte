<script>
  import DocsViewer from '../lib/DocsViewer.svelte'
  import { theme } from '../lib/theme.js'
  import overviewMd from '../docs/overview.md?raw'
  import gettingStartedMd from '../docs/getting-started.md?raw'
  import dashboardMd from '../docs/dashboard.md?raw'
  import environmentsMd from '../docs/environments.md?raw'
  import networksMd from '../docs/networks.md?raw'
  import commandPaletteMd from '../docs/command-palette.md?raw'
  import cidrWizardMd from '../docs/cidr-wizard.md?raw'
  import adminMd from '../docs/admin.md?raw'
  import reservedBlocksMd from '../docs/reserved-blocks.md?raw'
  import subnetCalculatorMd from '../docs/subnet-calculator.md?raw'

  export let currentPage = ''

  const PAGES = [
    { id: '', label: 'Overview' },
    { id: 'getting-started', label: 'Getting started' },
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'environments', label: 'Environments' },
    { id: 'networks', label: 'Networks' },
    { id: 'command-palette', label: 'Command palette' },
    { id: 'cidr-wizard', label: 'CIDR wizard' },
    { id: 'subnet-calculator', label: 'Subnet calculator' },
    { id: 'admin', label: 'Admin' },
    { id: 'reserved-blocks', label: 'Reserved blocks' },
  ]

  const CONTENT = {
    '': overviewMd,
    'getting-started': gettingStartedMd,
    'dashboard': dashboardMd,
    'environments': environmentsMd,
    'networks': networksMd,
    'command-palette': commandPaletteMd,
    'cidr-wizard': cidrWizardMd,
    'subnet-calculator': subnetCalculatorMd,
    'admin': adminMd,
    'reserved-blocks': reservedBlocksMd,
  }

  $: normalizedPage = PAGES.some((p) => p.id === currentPage) ? currentPage : ''
  $: markdownContent = CONTENT[normalizedPage] ?? ''

  function toggleTheme() {
    theme.set($theme === 'dark' ? 'light' : 'dark')
  }
</script>

<div class="docs-full">
  <aside class="docs-sidebar">
    <button type="button" class="docs-back" on:click={() => { window.location.hash = '' }}>← Back to IPAM</button>
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
  <button
    type="button"
    class="floating-theme-btn"
    on:click={toggleTheme}
    title={$theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
    aria-label={$theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
  >
    {#if $theme === 'dark'}
      <svg class="floating-theme-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <circle cx="12" cy="12" r="5" />
        <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
      </svg>
    {:else}
      <svg class="floating-theme-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
      </svg>
    {/if}
  </button>
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
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    font-family: inherit;
    text-align: left;
  }
  .docs-back:hover {
    text-decoration: underline;
  }
  /* Floating theme toggle (bottom right) */
  .floating-theme-btn {
    position: fixed;
    bottom: 1.5rem;
    right: 1.5rem;
    z-index: 100;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 3rem;
    height: 3rem;
    padding: 0;
    border: none;
    border-radius: 50%;
    background: var(--surface);
    color: var(--text-muted);
    cursor: pointer;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15), 0 0 0 1px var(--border);
    transition: color 0.15s, background 0.15s, box-shadow 0.15s;
  }
  .floating-theme-btn:hover {
    color: var(--text);
    background: var(--surface-elevated, var(--surface));
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2), 0 0 0 1px var(--border);
  }
  .floating-theme-btn .floating-theme-icon {
    width: 1.375rem;
    height: 1.375rem;
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
