<script>
  import { onMount } from 'svelte'
  import { theme } from './lib/theme.js'
  import { checkAuth, authChecked, user, setupRequired, checkSetupRequired, logout, selectedOrgForGlobalAdmin, selectedOrgNameForGlobalAdmin, isGlobalAdmin } from './lib/auth.js'
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
  import Signup from './routes/Signup.svelte'
  import Admin from './routes/Admin.svelte'
  import ReservedBlocks from './routes/ReservedBlocks.svelte'
  import SubnetCalculator from './routes/SubnetCalculator.svelte'
  import NetworkAdvisor from './routes/NetworkAdvisor.svelte'
  import Landing from './routes/Landing.svelte'
  import GlobalAdminDashboard from './routes/GlobalAdminDashboard.svelte'

  let route = 'landing'
  let showTour = false
  let tourStep = 0
  let routeEnvironmentId = null
  let routeEnvId = null
  let routeOrphanedOnly = false
  let routeBlockName = null
  let routeAllocationName = null
  let routeCreateEnv = false
  let routeCreateBlock = false
  let routeCreateAllocation = false
  let routeDocsPage = ''
  let routeSignupToken = ''
  let paletteOpen = false

  function go(path, environmentId = null, opts = {}) {
    route = path
    routeEnvironmentId = environmentId
    routeOrphanedOnly = false
    routeBlockName = opts.block ?? null
    routeAllocationName = opts.allocation ?? null
    routeCreateEnv = opts.create === true
    routeCreateBlock = opts.createBlock === true
    routeCreateAllocation = opts.createAllocation === true
    routeEnvId = opts.env ?? null
    if (path === 'dashboard') {
      window.location.hash = 'dashboard'
    } else if (path === 'networks') {
      const params = new URLSearchParams()
      if (environmentId) params.set('environment', environmentId)
      if (opts.block) params.set('block', opts.block)
      if (opts.allocation) params.set('allocation', opts.allocation)
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
    } else if (path === 'global-admin') {
      window.location.hash = 'global-admin'
    } else if (path === 'admin') {
      window.location.hash = 'admin'
    } else if (path === 'reserved-blocks') {
      window.location.hash = 'reserved-blocks'
    } else if (path === 'subnet-calculator') {
      window.location.hash = 'subnet-calculator'
    } else if (path === 'network-advisor') {
      window.location.hash = 'network-advisor'
    } else {
      window.location.hash = path
    }
  }

  function parseHash() {
    const raw = (window.location.hash || '#').slice(1) || ''
    const [path, query] = raw.split('?')
    if (path === '' || path === 'landing' || path === 'features' || path === 'terraform' || path === 'api' || path === 'user-guide') {
      route = 'landing'
      return
    }
    if (path === 'login') {
      route = 'login'
      return
    }
    if (path === 'setup') {
      route = 'setup'
      return
    }
    if (path === 'signup') {
      route = 'signup'
      routeSignupToken = ''
      if (query) {
        const params = new URLSearchParams(query)
        routeSignupToken = params.get('token') || ''
      }
      return
    }
    if (path === 'environments' || path === 'networks') {
      route = path
      routeEnvironmentId = null
      routeEnvId = null
      routeOrphanedOnly = false
      routeBlockName = null
      routeAllocationName = null
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
          const allocationName = params.get('allocation')
          routeAllocationName = allocationName || null
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
    } else if (path === 'global-admin') {
      route = 'global-admin'
      selectedOrgForGlobalAdmin.set(null)
      selectedOrgNameForGlobalAdmin.set(null)
    } else if (path === 'admin') {
      route = 'admin'
      selectedOrgForGlobalAdmin.set(null)
      selectedOrgNameForGlobalAdmin.set(null)
    } else if (path === 'reserved-blocks') {
      route = 'reserved-blocks'
    } else if (path === 'subnet-calculator') {
      route = 'subnet-calculator'
    } else if (path === 'network-advisor') {
      route = 'network-advisor'
    } else {
      route = 'dashboard'
      routeEnvironmentId = null
      routeEnvId = null
      routeOrphanedOnly = false
      routeBlockName = null
      routeAllocationName = null
      routeCreateEnv = false
      routeCreateBlock = false
      routeCreateAllocation = false
    }
  }

  function handlePaletteNavigate(e) {
    const { path, block, allocation, environmentId } = e.detail || {}
    if (path === 'environments') go('environments', null, environmentId ? { env: environmentId } : {})
    else if (path === 'networks') go('networks', null, { block: block ?? undefined, allocation: allocation ?? undefined })
    else if (path === 'dashboard') go('dashboard')
    else if (path === 'docs') go('docs', null, { page: (e.detail && e.detail.page) || '' })
    else if (path === 'admin') go('admin')
    else if (path === 'reserved-blocks') go('reserved-blocks')
    else if (path === 'subnet-calculator') go('subnet-calculator')
    else if (path === 'network-advisor') go('network-advisor')
  }

  function handlePaletteCreate(e) {
    const action = e.detail?.action
    if (action === 'create-env') go('environments', null, { create: true })
    else if (action === 'create-block') go('networks', null, { createBlock: true })
    else if (action === 'create-alloc') go('networks', null, { createAllocation: true })
  }

  async function handleLogout() {
    await logout()
    window.location.hash = ''
  }

  if (typeof window !== 'undefined') {
    parseHash()
  }

  $: if ($authChecked && $user && (route === 'admin' || route === 'reserved-blocks') && $user.role !== 'admin') {
    window.location.hash = ''
  }
  $: if ($authChecked && $user && isGlobalAdmin($user) && route === 'dashboard' && !$selectedOrgForGlobalAdmin) {
    window.location.hash = 'global-admin'
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

  function toggleTheme() {
    theme.set($theme === 'dark' ? 'light' : 'dark')
  }
</script>

{#if !$authChecked}
  <div class="app loading" role="presentation">
    <div class="loading-message">Loading…</div>
  </div>
{:else if route === 'landing' && !$user}
  {#if $setupRequired === null}
    <div class="app loading" role="presentation">
      <div class="loading-message">Loading…</div>
    </div>
  {:else if $setupRequired}
    <Setup />
  {:else}
    <Landing />
  {/if}
{:else if route === 'docs'}
  <Docs currentPage={routeDocsPage} />
{:else if !$user}
  {#if route === 'signup'}
    <Signup token={routeSignupToken} />
  {:else if route === 'login'}
    <Login />
  {:else if route === 'setup' || ($setupRequired !== null && $setupRequired)}
    <Setup />
  {:else if $setupRequired === null}
    <div class="app loading" role="presentation">
      <div class="loading-message">Loading…</div>
    </div>
  {:else}
    <Login />
  {/if}
{:else}
  <div class="app" role="presentation">
    <Nav
      current={route}
      currentUser={$user}
      selectedOrgIdFromParent={$selectedOrgForGlobalAdmin}
      on:nav={(e) => go(e.detail)}
      on:logout={handleLogout}
    />
    <main class="main" data-tour="tour-command-palette">
      {#if isGlobalAdmin($user) && route === 'global-admin'}
        <GlobalAdminDashboard
          on:selectOrg={(e) => {
            selectedOrgForGlobalAdmin.set(e.detail.id)
            selectedOrgNameForGlobalAdmin.set(e.detail.name)
            go('dashboard')
          }}
        />
      {:else if route === 'dashboard'}
        <Dashboard
          on:envBlocks={(e) => go('networks', e.detail)}
          on:viewOrphaned={() => { window.location.hash = 'networks?orphaned=1'; parseHash() }}
          on:viewBlock={(e) => { window.location.hash = 'networks?block=' + encodeURIComponent(e.detail); parseHash() }}
          on:viewAllocation={(e) => { window.location.hash = 'networks?allocation=' + encodeURIComponent(e.detail); parseHash() }}
        />
      {:else if route === 'environments'}
        <Environments openCreateFromQuery={routeCreateEnv} openEnvironmentId={routeEnvId} on:clearCreateQuery={() => { routeCreateEnv = false; if (window.location.hash.includes('create=1')) { window.location.hash = 'environments' } }} />
      {:else if route === 'networks'}
        <Networks
          environmentId={routeEnvironmentId}
          orphanedOnly={routeOrphanedOnly}
          blockNameFilter={routeBlockName}
          allocationFilter={routeAllocationName}
          openCreateBlockFromQuery={routeCreateBlock}
          openCreateAllocationFromQuery={routeCreateAllocation}
          on:clearEnv={() => { routeEnvironmentId = null; routeOrphanedOnly = false; routeBlockName = null; routeAllocationName = null; window.location.hash = 'networks' }}
          on:setEnv={(e) => go('networks', e.detail)}
          on:setBlockFilter={(e) => { const block = e.detail?.block ?? null; routeAllocationName = null; const params = new URLSearchParams(); if (routeEnvironmentId) params.set('environment', routeEnvironmentId); if (routeOrphanedOnly) params.set('orphaned', '1'); if (block) params.set('block', block); window.location.hash = 'networks' + (params.toString() ? '?' + params.toString() : ''); parseHash(); }}
          on:setAllocationFilter={(e) => { const allocation = e.detail?.allocation ?? null; const params = new URLSearchParams(); if (routeEnvironmentId) params.set('environment', routeEnvironmentId); if (routeOrphanedOnly) params.set('orphaned', '1'); if (routeBlockName) params.set('block', routeBlockName); if (allocation) params.set('allocation', allocation); window.location.hash = 'networks' + (params.toString() ? '?' + params.toString() : ''); parseHash(); }}
          on:clearCreateQuery={() => { routeCreateBlock = false; routeCreateAllocation = false; const h = window.location.hash; if (h.includes('createBlock=1') || h.includes('createAllocation=1')) { const p = new URLSearchParams((h.split('?')[1] || '')); p.delete('createBlock'); p.delete('createAllocation'); const q = p.toString(); window.location.hash = 'networks' + (q ? '?' + q : '') } }}
        />
      {:else if route === 'admin'}
        <Admin />
      {:else if route === 'reserved-blocks'}
        <ReservedBlocks />
      {:else if route === 'subnet-calculator'}
        <SubnetCalculator />
      {:else if route === 'network-advisor'}
        <NetworkAdvisor />
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

  /* Floating theme toggle (bottom right), same as Landing and Docs */
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

  .main {
    flex: 1;
    min-width: 0;
    padding: 1.5rem 1.5rem 2rem;
    max-width: 1120px;
    width: 100%;
    overflow: auto;
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
  /* Uniform buttons: same style, size, color across the UI. Primary = blue text + blue border. */
  :global(.btn) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    font-family: var(--font-sans);
    color: var(--text);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s, color 0.15s;
  }
  :global(.btn:hover:not(:disabled)) {
    background: var(--surface-elevated);
    border-color: var(--text-muted);
  }
  :global(.btn:disabled) {
    opacity: 0.6;
    cursor: not-allowed;
  }
  :global(.btn-primary) {
    color: var(--accent);
    border-color: var(--accent);
    background: var(--surface);
  }
  :global(.btn-primary:hover:not(:disabled)) {
    background: var(--accent-dim);
    border-color: var(--accent);
    color: var(--accent);
  }
  :global(.btn-small) {
    padding: 0.35rem 0.65rem;
    font-size: 0.85rem;
  }
  :global(.btn-danger) {
    color: var(--danger, #dc2626);
    border-color: var(--border);
    background: var(--surface);
  }
  :global(.btn-danger:hover:not(:disabled)) {
    background: rgba(220, 38, 38, 0.08);
    border-color: var(--danger, #dc2626);
    color: var(--danger, #dc2626);
  }
  :global(.btn-outline-danger) {
    color: var(--danger, #dc2626);
    border-color: var(--danger, #dc2626);
    border-style: dashed;
    background: transparent;
  }
  :global(.btn-outline-danger:hover:not(:disabled)) {
    background: rgba(220, 38, 38, 0.08);
    border-style: solid;
  }
  :global(.table-empty-cell) {
    color: var(--text-muted);
    padding: 1.5rem 1rem;
    text-align: center;
  }
  /* Standardized page headers: same height, title size, and spacing on every page */
  :global(.page-header) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1.5rem;
    min-height: 2.25rem;
  }
  :global(.page-header-text) {
    flex: 1;
    min-width: 0;
  }
  :global(.page-title) {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 600;
    letter-spacing: -0.02em;
    color: var(--text);
    line-height: 1.3;
  }
  :global(.page-desc) {
    margin: 0.25rem 0 0;
    font-size: 0.9rem;
    color: var(--text-muted);
    line-height: 1.4;
  }
</style>
