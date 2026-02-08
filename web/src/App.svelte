<script>
  import { onMount } from 'svelte'
  import './lib/theme.js'
  import { checkAuth, authChecked, user, setupRequired, checkSetupRequired } from './lib/auth.js'
  import { completeTour } from './lib/api.js'
  import Nav from './lib/Nav.svelte'
  import CommandPalette from './lib/CommandPalette.svelte'
  import Tour from './lib/Tour.svelte'
  import { tourSteps } from './lib/tourSteps.js'
  import Dashboard from './routes/Dashboard.svelte'
  import Environments from './routes/Environments.svelte'
  import Networks from './routes/Networks.svelte'
  import Docs from './routes/Docs.svelte'
  import Login from './routes/Login.svelte'
  import Setup from './routes/Setup.svelte'
  import Admin from './routes/Admin.svelte'

  let route = 'dashboard'
  let showTour = false
  let tourStep = 0
  let routeEnvironmentId = null
  let routeEnvId = null
  let routeOrphanedOnly = false
  let routeBlockName = null
  let routeCreateEnv = false
  let routeCreateBlock = false
  let routeCreateAllocation = false
  let routeDocsPage = ''
  let paletteOpen = false

  function go(path, environmentId = null, opts = {}) {
    route = path
    routeEnvironmentId = environmentId
    routeOrphanedOnly = false
    routeBlockName = opts.block ?? null
    routeCreateEnv = opts.create === true
    routeCreateBlock = opts.createBlock === true
    routeCreateAllocation = opts.createAllocation === true
    routeEnvId = opts.env ?? null
    if (path === 'dashboard') {
      window.location.hash = ''
    } else if (path === 'networks') {
      const params = new URLSearchParams()
      if (environmentId) params.set('environment', environmentId)
      if (opts.block) params.set('block', opts.block)
      if (opts.createBlock) params.set('createBlock', '1')
      if (opts.createAllocation) params.set('createAllocation', '1')
      const q = params.toString()
      window.location.hash = 'networks' + (q ? '?' + q : '')
    } else if (path === 'environments') {
      const params = new URLSearchParams()
      if (opts.create) params.set('create', '1')
      if (opts.env) params.set('env', opts.env)
      const q = params.toString()
      window.location.hash = 'environments' + (q ? '?' + q : '')
    } else if (path === 'docs') {
      window.location.hash = opts.page ? 'docs/' + opts.page : 'docs'
    } else if (path === 'admin') {
      window.location.hash = 'admin'
    } else {
      window.location.hash = path
    }
  }

  function parseHash() {
    const raw = (window.location.hash || '#').slice(1) || 'dashboard'
    const [path, query] = raw.split('?')
    if (path === 'environments' || path === 'networks') {
      route = path
      routeEnvironmentId = null
      routeEnvId = null
      routeOrphanedOnly = false
      routeBlockName = null
      routeCreateEnv = false
      routeCreateBlock = false
      routeCreateAllocation = false
      if (query) {
        const params = new URLSearchParams(query)
        if (path === 'networks') {
          const envId = params.get('environment')
          if (envId) routeEnvironmentId = envId
          routeOrphanedOnly = params.get('orphaned') === '1' || params.get('orphaned') === 'true'
          const blockName = params.get('block')
          routeBlockName = blockName || null
          routeCreateBlock = params.get('createBlock') === '1'
          routeCreateAllocation = params.get('createAllocation') === '1'
        } else if (path === 'environments') {
          routeCreateEnv = params.get('create') === '1'
          const envId = params.get('env')
          routeEnvId = envId || null
        }
      }
    } else if (path === 'docs' || path.startsWith('docs/')) {
      route = 'docs'
      routeDocsPage = path === 'docs' ? '' : path.slice(5)
    } else if (path === 'admin') {
      route = 'admin'
    } else {
      route = 'dashboard'
      routeEnvironmentId = null
      routeEnvId = null
      routeOrphanedOnly = false
      routeBlockName = null
      routeCreateEnv = false
      routeCreateBlock = false
      routeCreateAllocation = false
    }
  }

  function handlePaletteNavigate(e) {
    const { path, block, environmentId } = e.detail || {}
    if (path === 'environments') go('environments', null, environmentId ? { env: environmentId } : {})
    else if (path === 'networks') go('networks', null, block ? { block } : {})
    else if (path === 'dashboard') go('dashboard')
    else if (path === 'docs') go('docs', null, { page: (e.detail && e.detail.page) || '' })
    else if (path === 'admin') go('admin')
  }

  function handlePaletteCreate(e) {
    const action = e.detail?.action
    if (action === 'create-env') go('environments', null, { create: true })
    else if (action === 'create-block') go('networks', null, { createBlock: true })
    else if (action === 'create-alloc') go('networks', null, { createAllocation: true })
  }

  async function handleLogout() {
    const { logout } = await import('./lib/auth.js')
    await logout()
    window.location.hash = ''
  }

  if (typeof window !== 'undefined') {
    parseHash()
  }

  $: if ($authChecked && $user && route === 'admin' && $user.role !== 'admin') {
    window.location.hash = ''
  }

  let setupCheckRequested = false
  $: if ($authChecked && !$user && $setupRequired === null && !setupCheckRequested) {
    setupCheckRequested = true
    checkSetupRequired()
  }

  onMount(() => {
    checkAuth()
    parseHash()
    const handler = () => parseHash()
    window.addEventListener('hashchange', handler)
    function keydown(e) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        paletteOpen = !paletteOpen
      }
    }
    window.addEventListener('keydown', keydown)
    return () => {
      window.removeEventListener('hashchange', handler)
      window.removeEventListener('keydown', keydown)
    }
  })

  let tourOfferChecked = false
  $: if ($authChecked && $user && !showTour && !tourOfferChecked) {
    tourOfferChecked = true
    if (!$user.tour_completed) {
      tourStep = 0
      showTour = true
    }
  }

  async function onTourDone() {
    showTour = false
    try {
      await completeTour()
      user.update((u) => (u ? { ...u, tour_completed: true } : u))
    } catch (_) {}
  }

  function onTourStep(e) {
    tourStep = e.detail?.index ?? 0
  }
</script>

{#if !$authChecked}
  <div class="app loading" role="presentation">
    <div class="loading-message">Loading…</div>
  </div>
{:else if !$user}
  {#if $setupRequired === null}
    <div class="app loading" role="presentation">
      <div class="loading-message">Loading…</div>
    </div>
  {:else if $setupRequired}
    <Setup />
  {:else}
    <Login />
  {/if}
{:else if route === 'docs'}
  <Docs currentPage={routeDocsPage} />
{:else}
  <div class="app" role="presentation">
    <Nav current={route} currentUser={$user} on:nav={(e) => go(e.detail)} on:logout={handleLogout} />
    <main class="main" data-tour="tour-command-palette">
      {#if route === 'dashboard'}
        <Dashboard
          on:envBlocks={(e) => go('networks', e.detail)}
          on:viewOrphaned={() => { window.location.hash = 'networks?orphaned=1'; parseHash() }}
          on:viewBlock={(e) => { window.location.hash = 'networks?block=' + encodeURIComponent(e.detail); parseHash() }}
        />
      {:else if route === 'environments'}
        <Environments openCreateFromQuery={routeCreateEnv} openEnvironmentId={routeEnvId} on:clearCreateQuery={() => { routeCreateEnv = false; if (window.location.hash.includes('create=1')) { window.location.hash = 'environments' } }} />
      {:else if route === 'networks'}
        <Networks
          environmentId={routeEnvironmentId}
          orphanedOnly={routeOrphanedOnly}
          blockNameFilter={routeBlockName}
          openCreateBlockFromQuery={routeCreateBlock}
          openCreateAllocationFromQuery={routeCreateAllocation}
          on:clearEnv={() => { routeEnvironmentId = null; routeOrphanedOnly = false; routeBlockName = null; window.location.hash = 'networks' }}
          on:setEnv={(e) => go('networks', e.detail)}
          on:clearCreateQuery={() => { routeCreateBlock = false; routeCreateAllocation = false; const h = window.location.hash; if (h.includes('createBlock=1') || h.includes('createAllocation=1')) { const p = new URLSearchParams((h.split('?')[1] || '')); p.delete('createBlock'); p.delete('createAllocation'); const q = p.toString(); window.location.hash = 'networks' + (q ? '?' + q : '') } }}
        />
      {:else if route === 'admin'}
        <Admin />
      {:else}
        <Dashboard />
      {/if}
    </main>
    <CommandPalette
      open={paletteOpen}
      currentRoute={route}
      currentUser={$user}
      on:close={() => (paletteOpen = false)}
      on:navigate={handlePaletteNavigate}
      on:create={handlePaletteCreate}
    />
    <Tour
      steps={tourSteps}
      open={showTour}
      currentStep={tourStep}
      on:step={onTourStep}
      on:done={onTourDone}
      on:skip={onTourDone}
    />
  </div>
{/if}

<style>
  .app {
    height: 100vh;
    min-height: 0;
    display: flex;
    flex-direction: row;
    overflow: hidden;
    background: var(--bg);
    color: var(--text);
  }
  .app.loading {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background: var(--bg);
    color: var(--text-muted);
  }
  .loading-message {
    font-size: 0.95rem;
  }
  .main {
    flex: 1;
    min-width: 0;
    padding: 1.5rem 1.5rem 2rem;
    max-width: 1120px;
    width: 100%;
    overflow: auto;
  }

  :global(:root) {
    --radius: 10px;
    --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.04);
    --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.06), 0 2px 4px -2px rgba(0, 0, 0, 0.04);
    --font-sans: 'Inter', system-ui, -apple-system, sans-serif;
    --font-mono: 'JetBrains Mono', ui-monospace, monospace;
  }
  :global(:root[data-theme='dark']) {
    --shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.2);
    --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.25);
    --bg: #18181b;
    --surface: #27272a;
    --surface-elevated: #3f3f46;
    --border: #3f3f46;
    --text: #fafafa;
    --text-muted: #a1a1aa;
    --accent: #6366f1;
    --accent-dim: #6366f125;
    --btn-primary-text: #fff;
    --btn-primary-hover-bg: #4f46e5;
    --btn-primary-hover-border: #4f46e5;
    --table-row-hover: rgba(255, 255, 255, 0.03);
    --table-header-bg: rgba(255, 255, 255, 0.04);
    --success: #22c55e;
    --warn: #eab308;
    --danger: #ef4444;
  }
  :global(:root[data-theme='light']) {
    --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.04);
    --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.06), 0 2px 4px -2px rgba(0, 0, 0, 0.04);
    --bg: #f4f4f5;
    --surface: #ffffff;
    --surface-elevated: #ffffff;
    --border: #e4e4e7;
    --text: #18181b;
    --text-muted: #71717a;
    --accent: #6366f1;
    --accent-dim: #6366f115;
    --btn-primary-text: #fff;
    --btn-primary-hover-bg: #4f46e5;
    --btn-primary-hover-border: #4f46e5;
    --table-row-hover: rgba(0, 0, 0, 0.02);
    --table-header-bg: rgba(0, 0, 0, 0.03);
    --success: #16a34a;
    --warn: #ca8a04;
    --danger: #dc2626;
  }
  :global(:root:not([data-theme])) {
    --shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.2);
    --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.25);
    --bg: #18181b;
    --surface: #27272a;
    --surface-elevated: #3f3f46;
    --border: #3f3f46;
    --text: #fafafa;
    --text-muted: #a1a1aa;
    --accent: #6366f1;
    --accent-dim: #6366f125;
    --btn-primary-text: #fff;
    --btn-primary-hover-bg: #4f46e5;
    --btn-primary-hover-border: #4f46e5;
    --table-row-hover: rgba(255, 255, 255, 0.03);
    --table-header-bg: rgba(255, 255, 255, 0.04);
    --success: #22c55e;
    --warn: #eab308;
    --danger: #ef4444;
  }
  :global(*) {
    box-sizing: border-box;
  }
  :global(body) {
    margin: 0;
    font-family: var(--font-sans);
    font-size: 14px;
    line-height: 1.5;
    -webkit-font-smoothing: antialiased;
  }
  :global(.btn) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.5rem 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s, color 0.15s;
  }
  :global(.btn:hover:not(:disabled)) {
    background: var(--surface-elevated);
    border-color: var(--border);
  }
  :global(.btn:disabled) {
    opacity: 0.6;
    cursor: not-allowed;
  }
  :global(.btn-primary) {
    background: var(--accent);
    border-color: var(--accent);
    color: var(--btn-primary-text);
  }
  :global(.btn-primary:hover:not(:disabled)) {
    background: var(--btn-primary-hover-bg);
    border-color: var(--btn-primary-hover-border);
  }
</style>
