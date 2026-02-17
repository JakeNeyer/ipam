<script>
  import DocsViewer from '../lib/DocsViewer.svelte'
  import { theme } from '../lib/theme.js'
  import overviewMd from '../docs/overview.md?raw'
  import gettingStartedMd from '../docs/getting-started.md?raw'
  import environmentsMd from '../docs/environments.md?raw'
  import networksMd from '../docs/networks.md?raw'
  import commandPaletteMd from '../docs/command-palette.md?raw'
  import cidrWizardMd from '../docs/cidr-wizard.md?raw'
  import reservedBlocksMd from '../docs/reserved-blocks.md?raw'
  import subnetCalculatorMd from '../docs/subnet-calculator.md?raw'
  import networkAdvisorMd from '../docs/network-advisor.md?raw'
  import adminMd from '../docs/admin.md?raw'
  import integrationsMd from '../docs/integrations.md?raw'
  import integrationsAwsMd from '../docs/integrations/aws.md?raw'

  export let currentPage = ''

  /** Nav tree: top-level items; optional `children` for sub-pages. */
  const NAV = [
    { id: '', label: 'Overview' },
    { id: 'getting-started', label: 'Getting started' },
    { id: 'environments', label: 'Environments' },
    {
      id: 'networks',
      label: 'Networks',
      children: [
        { id: 'cidr-wizard', label: 'CIDR wizard' },
      ],
    },
    {
      id: 'integrations',
      label: 'Integrations',
      children: [
        { id: 'integrations/aws', label: 'AWS' },
      ],
    },
    { id: 'command-palette', label: 'Command palette' },
    { id: 'network-advisor', label: 'Network Advisor' },
    { id: 'subnet-calculator', label: 'Subnet calculator' },
    { id: 'reserved-blocks', label: 'Reserved blocks' },
    { id: 'admin', label: 'Admin' },
  ]

  function allPageIds() {
    const ids = []
    for (const item of NAV) {
      ids.push(item.id)
      if (item.children) for (const c of item.children) ids.push(c.id)
    }
    return ids
  }

  const CONTENT = {
    '': overviewMd,
    'getting-started': gettingStartedMd,
    'environments': environmentsMd,
    'networks': networksMd,
    'integrations': integrationsMd,
    'integrations/aws': integrationsAwsMd,
    'command-palette': commandPaletteMd,
    'cidr-wizard': cidrWizardMd,
    'network-advisor': networkAdvisorMd,
    'subnet-calculator': subnetCalculatorMd,
    'reserved-blocks': reservedBlocksMd,
    'admin': adminMd,
  }

  $: normalizedPage = allPageIds().includes(currentPage) ? currentPage : ''
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
      <ul class="docs-nav-list">
        {#each NAV as item}
          <li class="docs-nav-item">
            <a
              href="#docs{item.id ? '/' + item.id : ''}"
              class="docs-nav-link"
              class:active={normalizedPage === item.id}
            >
              {item.label}
            </a>
            {#if item.children && item.children.length > 0}
              <ul class="docs-nav-sublist">
                {#each item.children as child}
                  <li class="docs-nav-subitem">
                    <a
                      href="#docs/{child.id}"
                      class="docs-nav-link docs-nav-sublink"
                      class:active={normalizedPage === child.id}
                    >
                      {child.label}
                    </a>
                  </li>
                {/each}
              </ul>
            {/if}
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
    gap: 0;
    width: 100%;
    min-height: 100vh;
    height: 100vh;
    padding: 0;
    box-sizing: border-box;
    background: var(--bg);
    color: var(--text);
    overflow: auto;
  }
  .docs-sidebar {
    flex-shrink: 0;
    width: 14rem;
    padding: 2rem 1.5rem 2rem 1.75rem;
    background: transparent;
  }
  .docs-back {
    display: inline-block;
    margin-bottom: 2rem;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-muted);
    text-decoration: none;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    font-family: inherit;
    letter-spacing: 0.02em;
    transition: color 0.15s ease;
  }
  .docs-back:hover {
    color: var(--accent);
  }
  .floating-theme-btn {
    position: fixed;
    bottom: 1.5rem;
    right: 1.5rem;
    z-index: 100;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    padding: 0;
    border: none;
    border-radius: 50%;
    background: var(--surface);
    color: var(--text-muted);
    cursor: pointer;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
    transition: color 0.15s ease, background 0.15s ease, box-shadow 0.15s ease;
  }
  .floating-theme-btn:hover {
    color: var(--text);
    background: var(--surface-elevated, var(--surface));
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  }
  .floating-theme-btn .floating-theme-icon {
    width: 1.125rem;
    height: 1.125rem;
  }
  .docs-sidebar-title {
    margin: 0 0 1rem 0;
    font-size: 0.6875rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--text-muted);
  }
  .docs-nav-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .docs-nav-item {
    margin: 0;
  }
  .docs-nav-sublist {
    list-style: none;
    margin: 0 0 0.5rem 0;
    padding: 0 0 0 1rem;
    border-left: 1px solid var(--border);
  }
  .docs-nav-subitem {
    margin: 0;
  }
  .docs-nav-link {
    display: block;
    padding: 0.4rem 0;
    font-size: 0.875rem;
    font-weight: 400;
    color: var(--text-muted);
    text-decoration: none;
    border-radius: 4px;
    transition: color 0.12s ease;
  }
  .docs-nav-sublink {
    padding: 0.25rem 0;
    font-size: 0.8125rem;
  }
  .docs-nav-link:hover {
    color: var(--text);
  }
  .docs-nav-link.active {
    color: var(--accent);
    font-weight: 500;
  }
  .docs-sidebar-note {
    margin: 2rem 0 0 0;
    padding-top: 1.25rem;
    border-top: 1px solid var(--border);
    font-size: 0.75rem;
    line-height: 1.45;
    color: var(--text-muted);
  }
  .docs-content {
    flex: 1;
    min-width: 0;
    padding: 2rem 2.5rem 3rem;
    border-left: 1px solid var(--border);
    overflow-y: auto;
  }
</style>
