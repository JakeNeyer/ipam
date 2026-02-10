<script>
  import { createEventDispatcher } from 'svelte'
  import Icon from '@iconify/svelte'
  import { theme } from './theme.js'
  export let current = 'dashboard'
  export let currentUser = null
  const dispatch = createEventDispatcher()

  const COLLAPSED_KEY = 'ipam-nav-collapsed'
  let collapsed = false
  let settingsOpen = false

  if (typeof window !== 'undefined') {
    try {
      const stored = localStorage.getItem(COLLAPSED_KEY)
      collapsed = stored === 'true'
    } catch (_) {}
  }

  function toggleCollapse() {
    collapsed = !collapsed
    try {
      localStorage.setItem(COLLAPSED_KEY, String(collapsed))
    } catch (_) {}
  }

  function closeSettings(e) {
    if (settingsOpen && e.target && !e.target.closest('.settings-area')) settingsOpen = false
  }

  const links = [
    { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
    { id: 'environments', label: 'Environments', icon: 'lucide:layers' },
    { id: 'networks', label: 'Networks', icon: 'lucide:network' },
    { id: 'subnet-calculator', label: 'Subnet calculator', icon: 'lucide:calculator' },
  ]

  let hoveredLabel = null
</script>

<svelte:window on:click={closeSettings} />

<nav class="nav" class:collapsed>
  {#if !collapsed}
    <div class="brand">
      <img src={$theme === 'light' ? '/images/logo-light.svg' : '/images/logo.svg'} alt="IPAM" class="logo" />
    </div>
  {/if}
  <ul class="links">
    {#each links as link}
      <li>
        <button
          class="link"
          class:active={current === link.id}
          data-tour="tour-nav-{link.id}"
          on:click={() => dispatch('nav', link.id)}
          on:mouseenter={() => (hoveredLabel = collapsed ? link.label : null)}
          on:mouseleave={() => (hoveredLabel = null)}
          title={collapsed ? link.label : ''}
          aria-label={link.label}
        >
          <span class="icon"><Icon icon={link.icon} width="1.25em" height="1.25em" /></span>
          {#if !collapsed}
            <span class="label">{link.label}</span>
          {:else if hoveredLabel === link.label}
            <span class="nav-tooltip" role="tooltip">{link.label}</span>
          {/if}
        </button>
      </li>
    {/each}
    {#if currentUser?.role === 'admin'}
      <li>
        <button
          class="link"
          class:active={current === 'reserved-blocks'}
          on:click={() => dispatch('nav', 'reserved-blocks')}
          on:mouseenter={() => (hoveredLabel = collapsed ? 'Reserved blocks' : null)}
          on:mouseleave={() => (hoveredLabel = null)}
          title={collapsed ? 'Reserved blocks' : ''}
          aria-label="Reserved blocks"
        >
          <span class="icon"><Icon icon="lucide:ban" width="1.25em" height="1.25em" /></span>
          {#if !collapsed}
            <span class="label">Reserved blocks</span>
          {:else if hoveredLabel === 'Reserved blocks'}
            <span class="nav-tooltip" role="tooltip">Reserved blocks</span>
          {/if}
        </button>
      </li>
      <li>
        <button
          class="link"
          class:active={current === 'admin'}
          data-tour="tour-nav-admin"
          on:click={() => dispatch('nav', 'admin')}
          on:mouseenter={() => (hoveredLabel = collapsed ? 'Admin' : null)}
          on:mouseleave={() => (hoveredLabel = null)}
          title={collapsed ? 'Admin' : ''}
          aria-label="Admin"
        >
          <span class="icon"><Icon icon="lucide:shield" width="1.25em" height="1.25em" /></span>
          {#if !collapsed}
            <span class="label">Admin</span>
          {:else if hoveredLabel === 'Admin'}
            <span class="nav-tooltip" role="tooltip">Admin</span>
          {/if}
        </button>
      </li>
    {/if}
  </ul>
  <div class="nav-footer">
    <button
      type="button"
      class="collapse-btn"
      on:click={toggleCollapse}
      title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
    >
      <span class="collapse-icon" aria-hidden="true"><Icon icon={collapsed ? 'lucide:chevron-right' : 'lucide:chevron-left'} width="1em" height="1em" /></span>
      {#if !collapsed}
        <span class="collapse-label">Collapse</span>
      {/if}
    </button>
    <div class="settings-area">
      <button
        type="button"
        class="settings-trigger"
        data-tour="tour-nav-settings"
        on:click={() => (settingsOpen = !settingsOpen)}
        title="Settings"
        aria-label="Settings"
        aria-expanded={settingsOpen}
        aria-haspopup="true"
      >
        <span class="settings-icon" aria-hidden="true"><Icon icon="lucide:settings" width="1.25em" height="1.25em" /></span>
        {#if !collapsed}
          <span class="settings-label">Settings</span>
        {/if}
      </button>
      {#if settingsOpen}
        <div class="settings-popover" role="menu">
          <a
            href="#docs"
            class="settings-item"
            role="menuitem"
          >
            User guide
          </a>
          <a
            href="/docs"
            target="_blank"
            rel="noopener noreferrer"
            class="settings-item"
            role="menuitem"
          >
            API docs
          </a>
          <button
            type="button"
            class="settings-item"
            role="menuitem"
            on:click={() => { dispatch('logout'); settingsOpen = false }}
          >
            <Icon icon="lucide:log-out" width="1em" height="1em" /> Sign out
          </button>
        </div>
      {/if}
    </div>
  </div>
</nav>

<style>
  .nav {
    display: flex;
    flex-direction: column;
    width: 12rem;
    min-width: 12rem;
    height: 100vh;
    min-height: 0;
    padding: 1rem 0.75rem;
    background: var(--surface);
    border-right: 1px solid var(--border);
    flex-shrink: 0;
    transition: width 0.2s ease, min-width 0.2s ease;
  }
  .nav.collapsed {
    width: 3.5rem;
    min-width: 3.5rem;
    padding-left: 0.5rem;
    padding-right: 0.5rem;
  }
  .brand {
    display: flex;
    align-items: center;
    padding: 0 0.5rem 1rem;
    margin-bottom: 0.5rem;
    border-bottom: 1px solid var(--border);
  }
  .logo {
    display: block;
    height: 3.25rem;
    width: auto;
    object-fit: contain;
  }
  .links {
    display: flex;
    flex-direction: column;
    list-style: none;
    margin: 0;
    padding: 0;
    gap: 0.125rem;
    flex: 1;
    min-height: 0;
  }
  .link {
    position: relative;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    background: transparent;
    border: none;
    border-radius: var(--radius);
    color: var(--text-muted);
    font-family: var(--font-sans);
    font-size: 0.875rem;
    font-weight: 500;
    text-align: left;
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }
  .nav.collapsed .link {
    padding: 0.5rem;
    justify-content: center;
  }
  .nav-tooltip {
    position: absolute;
    left: 100%;
    top: 50%;
    transform: translateY(-50%);
    margin-left: 0.5rem;
    padding: 0.25rem 0.5rem;
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text);
    font-size: 0.8rem;
    font-weight: 500;
    white-space: nowrap;
    box-shadow: var(--shadow-md);
    pointer-events: none;
    z-index: 200;
  }
  .link:hover {
    color: var(--text);
    background: var(--accent-dim);
  }
  .link.active {
    color: var(--accent);
    background: var(--accent-dim);
  }
  .icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    opacity: 0.9;
    flex-shrink: 0;
  }
  .icon :global(svg) {
    flex-shrink: 0;
  }
  .label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .nav-footer {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    margin-top: 0.5rem;
    padding-top: 0.5rem;
    border-top: 1px solid var(--border);
  }
  .collapse-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    background: transparent;
    border: none;
    border-radius: var(--radius);
    color: var(--text-muted);
    font-family: var(--font-sans);
    font-size: 0.875rem;
    font-weight: 500;
    text-align: left;
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }
  .nav.collapsed .collapse-btn {
    padding: 0.5rem;
    justify-content: center;
  }
  .collapse-btn:hover {
    color: var(--text);
    background: var(--accent-dim);
  }
  .collapse-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
    flex-shrink: 0;
  }
  .collapse-label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .settings-area {
    position: relative;
  }
  .settings-trigger {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    background: transparent;
    border: none;
    border-radius: var(--radius);
    color: var(--text-muted);
    font-family: var(--font-sans);
    font-size: 0.875rem;
    font-weight: 500;
    text-align: left;
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }
  .nav.collapsed .settings-trigger {
    padding: 0.5rem;
    justify-content: center;
  }
  .settings-trigger:hover {
    color: var(--text);
    background: var(--accent-dim);
  }
  .settings-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .settings-label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .settings-popover {
    position: absolute;
    bottom: calc(100% + 0.5rem);
    left: 0;
    min-width: 9rem;
    padding: 0.2rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
    z-index: 100;
  }
  .nav.collapsed .settings-popover {
    left: calc(100% + 0.5rem);
    bottom: auto;
    top: 0;
  }
  .settings-item {
    display: block;
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: none;
    border-radius: calc(var(--radius) - 2px);
    background: transparent;
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    text-align: left;
    text-decoration: none;
    cursor: pointer;
    transition: background 0.15s;
  }
  .settings-item:hover {
    background: rgba(255, 255, 255, 0.06);
  }
  :global([data-theme='light']) .settings-item:hover {
    background: rgba(0, 0, 0, 0.05);
  }
</style>
