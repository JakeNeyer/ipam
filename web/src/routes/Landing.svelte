<script>
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import SocialIcons from '@rodneylab/svelte-social-icons'
  import { theme } from '../lib/theme.js'

  const base = (import.meta.env.BASE_URL || '/').replace(/\/+$/, '') + '/'
  const githubUrl = 'https://github.com/JakeNeyer/ipam'
  const githubApiUrl = 'https://api.github.com/repos/JakeNeyer/ipam'
  const terraformRegistryUrl = 'https://registry.terraform.io/providers/JakeNeyer/ipam/latest/docs'

  let githubStats = null

  /** @type {'azure' | 'aws' | 'gcp'} */
  let terraformTab = 'aws'

  /** @type {Set<string>} sections that have entered the viewport */
  let visibleSections = new Set(['hero'])

  /** @type {boolean} true when user has scrolled past hero */
  let headerScrolled = false

  function toggleTheme() {
    theme.set($theme === 'dark' ? 'light' : 'dark')
  }

  function goLogin() {
    window.location.hash = 'login'
    window.dispatchEvent(new HashChangeEvent('hashchange'))
  }

  function formatCount(n) {
    if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k'
    return String(n)
  }

  onMount(() => {
    fetch(githubApiUrl)
      .then((r) => r.ok ? r.json() : Promise.reject(new Error('Not ok')))
      .then((data) => {
        githubStats = {
          stars: data.stargazers_count ?? 0,
          forks: data.forks_count ?? 0,
        }
      })
      .catch(() => {})

    const observer = new IntersectionObserver(
      (entries) => {
        let changed = false
        entries.forEach((entry) => {
          const id = entry.target.getAttribute('data-animate-id')
          if (!id) return
          if (entry.isIntersecting && !visibleSections.has(id)) {
            visibleSections.add(id)
            changed = true
          }
        })
        if (changed) visibleSections = new Set(visibleSections)
      },
      { rootMargin: '-8% 0px -8% 0px', threshold: 0 }
    )
    const el = document.querySelectorAll('[data-animate-id]')
    el.forEach((node) => observer.observe(node))

    const onScroll = () => {
      headerScrolled = window.scrollY > 60
    }
    window.addEventListener('scroll', onScroll, { passive: true })
    onScroll()

    return () => {
      observer.disconnect()
      window.removeEventListener('scroll', onScroll)
    }
  })
</script>

<div class="landing">
  <header class="landing-header" class:scrolled={headerScrolled}>
    <a href="#landing" class="landing-logo-wrap">
      <img src="{base}images/{$theme === 'light' ? 'logo-light.svg' : 'logo.svg'}" alt="IPAM" class="landing-logo" />
    </a>
    <div class="landing-header-actions">
      <a href={githubUrl} target="_blank" rel="noopener noreferrer" class="github-link github-link-icon-only" aria-label="Star on GitHub">
        <span class="github-link-icon">
          <SocialIcons alt="" network="github" width={32} height={32} fgColor="currentColor" bgColor="transparent" />
        </span>
      </a>
      <button type="button" class="landing-cta-btn" on:click={goLogin}>Log in</button>
    </div>
  </header>

  <section class="hero" data-animate-id="hero" class:visible={visibleSections.has('hero')}>
    <div class="hero-bg" aria-hidden="true"></div>
    <div class="hero-bg-title" aria-hidden="true"></div>
    <div class="hero-content">
      <p class="hero-eyebrow">IP address management</p>
      <h1 class="hero-title">IP address management, simplified</h1>
      <p class="hero-subtitle">
        A simple tool for advanced networks.
      </p>
      <div class="hero-actions">
        <a href="#features" class="btn-hero-primary">See how it works</a>
        <button type="button" class="btn-hero-secondary" on:click={goLogin}>Get started</button>
      </div>
    </div>
    <div class="hero-dashboard-wrap" aria-hidden="true">
      <div class="hero-dashboard-glow"></div>
      <div class="hero-dashboard-frame">
        <div class="hero-dashboard-titlebar">
          <div class="hero-dashboard-dots">
            <span></span><span></span><span></span>
          </div>
          <div class="hero-dashboard-url">ipam / dashboard</div>
        </div>
        <div class="hero-dashboard-inner">
          <img
            src="{base}images/dashboard-light.png"
            alt="IPAM Dashboard (light mode)"
            class="hero-dashboard-img hero-dashboard-img-light"
          />
          <img
            src="{base}images/dashboard-dark.png"
            alt="IPAM Dashboard (dark mode)"
            class="hero-dashboard-img hero-dashboard-img-dark"
          />
        </div>
      </div>
    </div>
  </section>

  <section id="features" class="section features" data-animate-id="features" class:visible={visibleSections.has('features')}>
    <p class="section-eyebrow">Core features</p>
    <h2 class="section-title">Everything you need to get the job done</h2>
    <p class="section-desc">Plan, organize, sync, and track networks—all in one place.</p>
    <div class="features-grid">
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2L2 7l10 5 10-5L12 2z" />
            <path d="M2 17l10 5 10-5" />
            <path d="M2 12l10 5 10-5" />
          </svg>
        </div>
        <h3 class="feature-title">Environments, Pools, Network blocks & Allocations</h3>
        <p class="feature-desc">Organize by environments (e.g. staging, production); define CIDR pools per environment; create network blocks that draw from pools; allocate subnets within blocks as in-use networks.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
            <path d="M7 11V7a5 5 0 0 1 10 0v4" />
          </svg>
        </div>
        <h3 class="feature-title">Reserved blocks</h3>
        <p class="feature-desc">Wall off address ranges from being used.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z" />
          </svg>
        </div>
        <h3 class="feature-title">Cloud provider integration</h3>
        <p class="feature-desc">Sync IPAM with cloud providers such as AWS VPC IPAM: import pools and allocations, keep them in sync, and manage from one place.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2z" />
            <path d="M2 12h20" />
            <path d="M12 2c3.5 3 5.5 7 5.5 10s-2 7-5.5 10c-3.5-3-5.5-7-5.5-10S8.5 5 12 2z" />
          </svg>
        </div>
        <h3 class="feature-title">IPv4 + IPv6 support</h3>
        <p class="feature-desc">Plan, allocate, and size networks across IPv4 and IPv6 (including ULA).</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 12h3l2-4 4 8 2-4h7" />
            <circle cx="5" cy="12" r="2" />
            <circle cx="12" cy="16" r="2" />
            <circle cx="19" cy="12" r="2" />
          </svg>
        </div>
        <h3 class="feature-title">Network Advisor</h3>
        <p class="feature-desc">Plan and optimize IP allocation with a step-by-step wizard.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" />
            <line x1="3" y1="9" x2="21" y2="9" />
            <line x1="3" y1="15" x2="21" y2="15" />
            <line x1="9" y1="3" x2="9" y2="21" />
            <line x1="15" y1="3" x2="15" y2="21" />
          </svg>
        </div>
        <h3 class="feature-title">Subnet calculator</h3>
        <p class="feature-desc">Split and join subnets in a table; plan CIDRs without creating resources.</p>
      </div>
    </div>
  </section>

  <section id="integrations" class="section integrations-section" data-animate-id="integrations" class:visible={visibleSections.has('integrations')}>
    <p class="section-eyebrow">Cloud native</p>
    <h2 class="section-title">One IPAM. Every cloud.</h2>
    <p class="section-desc">Sync environments, pools, and allocations with AWS VPC IPAM, Azure Virtual Network Manager, Google Cloud VPC, and on-premises networks. Single source of truth across hybrid and multi-cloud.</p>
    <div class="integrations-diagram-wrap" aria-hidden="true">
      <div class="integrations-glow"></div>
      <div class="integrations-diagram-inner">
        <svg class="integrations-svg" viewBox="0 0 820 520" fill="none" xmlns="http://www.w3.org/2000/svg">
          <defs>
            <linearGradient id="ipamHubGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="var(--accent)" stop-opacity="0.3" />
              <stop offset="100%" stop-color="var(--accent)" stop-opacity="0.06" />
            </linearGradient>
            <filter id="integrationsGlow" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="6" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>
          <!-- Connection paths: balanced cross, node edge to hub edge (cards 24% × 24%) -->
          <g class="integrations-connectors">
            <path class="integration-line integration-line-onprem" d="M 213 260 Q 261 260 310 260" fill="none" stroke="var(--accent)" stroke-width="3" stroke-dasharray="12 18" stroke-linecap="round" opacity="0.75" />
            <path class="integration-line integration-line-aws" d="M 410 135 Q 410 167 410 200" fill="none" stroke="var(--accent)" stroke-width="3" stroke-dasharray="12 18" stroke-linecap="round" opacity="0.75" />
            <path class="integration-line integration-line-azure" d="M 607 260 Q 558 260 510 260" fill="none" stroke="var(--accent)" stroke-width="3" stroke-dasharray="12 18" stroke-linecap="round" opacity="0.75" />
            <path class="integration-line integration-line-gcp" d="M 410 385 Q 410 352 410 320" fill="none" stroke="var(--accent)" stroke-width="3" stroke-dasharray="12 18" stroke-linecap="round" opacity="0.75" />
          </g>
          <!-- Data flow dots -->
          <circle class="integration-dot integration-dot-onprem" r="6" fill="var(--accent)">
            <animateMotion dur="2.2s" repeatCount="indefinite" path="M 213 260 Q 261 260 310 260" />
          </circle>
          <circle class="integration-dot integration-dot-onprem-2" r="5" fill="var(--accent)" opacity="0.7">
            <animateMotion dur="2.2s" repeatCount="indefinite" path="M 213 260 Q 261 260 310 260" begin="0.7s" />
          </circle>
          <circle class="integration-dot integration-dot-aws" r="6" fill="var(--accent)">
            <animateMotion dur="2.2s" repeatCount="indefinite" path="M 410 135 Q 410 167 410 200" />
          </circle>
          <circle class="integration-dot integration-dot-aws-2" r="5" fill="var(--accent)" opacity="0.7">
            <animateMotion dur="2.2s" repeatCount="indefinite" path="M 410 135 Q 410 167 410 200" begin="0.7s" />
          </circle>
          <circle class="integration-dot integration-dot-azure" r="6" fill="var(--accent)">
            <animateMotion dur="2.2s" repeatCount="indefinite" path="M 607 260 Q 558 260 510 260" />
          </circle>
          <circle class="integration-dot integration-dot-azure-2" r="5" fill="var(--accent)" opacity="0.7">
            <animateMotion dur="2.2s" repeatCount="indefinite" path="M 607 260 Q 558 260 510 260" begin="0.7s" />
          </circle>
          <circle class="integration-dot integration-dot-gcp" r="6" fill="var(--accent)">
            <animateMotion dur="2.2s" repeatCount="indefinite" path="M 410 385 Q 410 352 410 320" />
          </circle>
          <circle class="integration-dot integration-dot-gcp-2" r="5" fill="var(--accent)" opacity="0.7">
            <animateMotion dur="2.2s" repeatCount="indefinite" path="M 410 385 Q 410 352 410 320" begin="0.7s" />
          </circle>
          <!-- Central IPAM hub -->
          <g class="integration-hub">
            <rect x="310" y="200" width="200" height="120" rx="24" fill="url(#ipamHubGrad)" stroke="var(--accent)" stroke-width="3" filter="url(#integrationsGlow)" class="integration-hub-bg" />
            <text x="410" y="248" text-anchor="middle" fill="var(--accent)" font-size="22" font-weight="800" font-family="system-ui, sans-serif">IPAM</text>
            <text x="410" y="275" text-anchor="middle" fill="var(--text-muted)" font-size="12" font-family="system-ui, sans-serif">Sync · Allocate · Track</text>
          </g>
        </svg>
        <!-- On-prem and cloud provider cards (overlaid) -->
        <div class="integration-node-card integration-node-onprem">
          <span class="integration-node-icon integration-node-icon-onprem" aria-hidden="true">
            <Icon icon="lucide:server" />
          </span>
          <span class="integration-node-title">On-premises</span>
          <span class="integration-node-subtitle">Data center / private</span>
        </div>
        <div class="integration-node-card integration-node-aws">
          <span class="integration-node-icon integration-node-icon-aws" aria-hidden="true">
            <Icon icon="simple-icons:amazonaws" />
          </span>
          <span class="integration-node-title">AWS</span>
          <span class="integration-node-subtitle">VPC IPAM</span>
        </div>
        <div class="integration-node-card integration-node-azure">
          <span class="integration-coming-soon">Coming soon</span>
          <span class="integration-node-icon integration-node-icon-azure" aria-hidden="true">
            <Icon icon="simple-icons:microsoftazure" />
          </span>
          <span class="integration-node-title">Azure</span>
          <span class="integration-node-subtitle">Virtual Network Manager</span>
        </div>
        <div class="integration-node-card integration-node-gcp">
          <span class="integration-coming-soon">Coming soon</span>
          <span class="integration-node-icon integration-node-icon-gcp" aria-hidden="true">
            <Icon icon="simple-icons:googlecloud" />
          </span>
          <span class="integration-node-title">Google Cloud</span>
          <span class="integration-node-subtitle">VPC address management</span>
        </div>
      </div>
    </div>
  </section>

  <section id="command-palette" class="section command-palette-section" data-animate-id="command-palette" class:visible={visibleSections.has('command-palette')}>
    <p class="section-eyebrow">Productivity</p>
    <h2 class="section-title">Command palette</h2>
    <p class="section-desc">Search, navigate, and create from anywhere. Press ⌘K or Ctrl+K.</p>
    <div class="command-palette-wrap" aria-hidden="true">
      <div class="command-palette-glow"></div>
      <div class="command-palette-frame">
        <div class="command-palette-titlebar">
          <div class="command-palette-dots">
            <span></span><span></span><span></span>
          </div>
          <div class="command-palette-hint">⌘K · Command palette</div>
        </div>
        <div class="command-palette-inner">
          <svg
            class="command-palette-svg"
            viewBox="0 0 400 280"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            aria-hidden="true"
          >
            <defs>
              <linearGradient id="paletteBg" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" stop-color="var(--surface-elevated, var(--surface))" />
                <stop offset="100%" stop-color="var(--surface)" />
              </linearGradient>
            </defs>
            <!-- Modal panel -->
            <rect x="24" y="16" width="352" height="248" rx="10" fill="url(#paletteBg)" stroke="var(--border)" stroke-width="1.5" />
            <!-- Search bar -->
            <rect x="40" y="32" width="320" height="40" rx="8" fill="var(--bg)" stroke="var(--border)" stroke-width="1" />
            <text x="56" y="57" fill="var(--text-muted)" font-size="12" font-family="system-ui, sans-serif">⌘</text>
            <text x="76" y="57" fill="var(--text-muted)" font-size="12" font-family="system-ui, sans-serif">Search environments, pools, blocks, allocations...</text>
            <!-- Command list -->
            <rect x="40" y="88" width="320" height="32" rx="6" fill="var(--accent-dim)" stroke="var(--accent)" stroke-width="1" opacity="0.9" />
            <text x="56" y="108" fill="var(--accent)" font-size="12" font-weight="600" font-family="system-ui, sans-serif">Go to Dashboard</text>
            <text x="56" y="132" fill="var(--text)" font-size="12" font-family="system-ui, sans-serif">Go to Environments</text>
            <text x="56" y="156" fill="var(--text)" font-size="12" font-family="system-ui, sans-serif">Go to Networks</text>
            <text x="56" y="180" fill="var(--text)" font-size="12" font-family="system-ui, sans-serif">Go to Docs</text>
            <text x="56" y="204" fill="var(--text)" font-size="12" font-family="system-ui, sans-serif">Create environment</text>
            <text x="56" y="228" fill="var(--text)" font-size="12" font-family="system-ui, sans-serif">Create network block</text>
            <!-- Footer -->
            <line x1="40" y1="248" x2="360" y2="248" stroke="var(--border)" stroke-width="1" />
            <text x="56" y="260" fill="var(--text-muted)" font-size="10" font-family="system-ui, sans-serif">↑↓ Navigate</text>
            <text x="200" y="260" fill="var(--text-muted)" font-size="10" font-family="system-ui, sans-serif" text-anchor="middle">↵ Select</text>
            <text x="344" y="260" fill="var(--text-muted)" font-size="10" font-family="system-ui, sans-serif" text-anchor="end">Esc</text>
          </svg>
        </div>
      </div>
    </div>
  </section>

  <section id="api" class="section api-section" data-animate-id="api" class:visible={visibleSections.has('api')}>
    <div class="api-bg" aria-hidden="true"></div>
    <div class="api-content">
      <p class="section-eyebrow">Developers</p>
      <h2 class="section-title">API</h2>
      <p class="section-desc">Full API for all IPAM resources. Use it from scripts, CI/CD, and the Terraform provider.</p>
      <div class="api-diagram" aria-hidden="true">
        <svg class="api-flow-svg" viewBox="0 0 420 160" fill="none" xmlns="http://www.w3.org/2000/svg">
          <defs>
            <linearGradient id="apiClientGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="var(--accent)" stop-opacity="0.2" />
              <stop offset="100%" stop-color="var(--accent)" stop-opacity="0.05" />
            </linearGradient>
            <marker id="apiArrow" markerWidth="8" markerHeight="8" refX="6" refY="4" orient="auto">
              <path d="M0 0 L8 4 L0 8 Z" fill="var(--accent)" />
            </marker>
          </defs>
          <rect x="20" y="20" width="80" height="120" rx="6" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="60" y="55" text-anchor="middle" fill="var(--text)" font-size="10" font-weight="600">Scripts</text>
          <text x="60" y="80" text-anchor="middle" fill="var(--text)" font-size="10" font-weight="600">CI/CD</text>
          <text x="60" y="105" text-anchor="middle" fill="var(--text)" font-size="10" font-weight="600">Terraform</text>
          <path d="M100 80 L140 80" stroke="var(--accent)" stroke-width="2" marker-end="url(#apiArrow)" />
          <rect x="150" y="50" width="100" height="60" rx="6" fill="var(--surface)" stroke="var(--accent)" stroke-width="1.5" />
          <text x="200" y="75" text-anchor="middle" fill="var(--text)" font-size="11" font-weight="600">API</text>
          <text x="200" y="92" text-anchor="middle" fill="var(--text-muted)" font-size="9">JSON</text>
          <path d="M250 80 L285 80" stroke="var(--accent)" stroke-width="2" />
          <rect x="290" y="8" width="120" height="26" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="24" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Environments</text>
          <rect x="290" y="38" width="120" height="26" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="54" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Pools</text>
          <rect x="290" y="68" width="120" height="26" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="84" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Network Blocks</text>
          <rect x="290" y="98" width="120" height="26" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="114" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Allocations</text>
          <rect x="290" y="128" width="120" height="26" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="144" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Reserved Blocks</text>
        </svg>
      </div>
    </div>
  </section>

  <section id="terraform" class="section terraform-section" data-animate-id="terraform" class:visible={visibleSections.has('terraform')}>
    <div class="terraform-bg" aria-hidden="true"></div>
    <div class="terraform-content">
      <div class="terraform-header">
        <p class="section-eyebrow">Infrastructure as code</p>
        <div class="terraform-title-row">
          <img src="{base}images/terraform.svg" alt="Terraform" class="terraform-logo" />
          <h2 class="section-title terraform-title">Terraform provider</h2>
        </div>
      </div>
      <p class="section-desc">Use IPAM with your favorite IaC tooling.</p>
      <div class="terraform-registry-wrap">
        <a href={terraformRegistryUrl} target="_blank" rel="noopener noreferrer" class="terraform-registry-link">
          <img src="{base}images/terraform.svg" alt="" class="terraform-registry-icon" aria-hidden="true" />
          <span>View on Terraform Registry</span>
          <svg class="terraform-registry-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path d="M7 17L17 7M17 7H7M17 7v10" />
          </svg>
        </a>
      </div>
      <div class="terraform-grid">
        <div class="terraform-card">
          <h3 class="terraform-card-title">Resources</h3>
          <ul class="terraform-list">
            <li><code>ipam_environment</code></li>
            <li><code>ipam_pool</code></li>
            <li><code>ipam_block</code></li>
            <li><code>ipam_allocation</code></li>
            <li><code>ipam_reserved_block</code></li>
          </ul>
        </div>
        <div class="terraform-card">
          <h3 class="terraform-card-title">Data sources</h3>
          <ul class="terraform-list">
            <li><code>ipam_environment</code> / <code>ipam_environments</code></li>
            <li><code>ipam_pool</code> / <code>ipam_pools</code></li>
            <li><code>ipam_block</code> / <code>ipam_blocks</code></li>
            <li><code>ipam_allocation</code> / <code>ipam_allocations</code></li>
            <li><code>ipam_reserved_block</code> / <code>ipam_reserved_blocks</code></li>
          </ul>
        </div>
      </div>
      <div class="terraform-tabs" role="tablist" aria-label="Cloud provider">
        <button
          type="button"
          class="terraform-tab"
          class:active={terraformTab === 'aws'}
          role="tab"
          aria-selected={terraformTab === 'aws'}
          aria-controls="terraform-panel-aws"
          id="terraform-tab-aws"
          on:click={() => (terraformTab = 'aws')}
        >
          <span class="terraform-tab-icon"><Icon icon="simple-icons:amazonaws" aria-hidden="true" /></span>
          <span>AWS</span>
        </button>
        <button
          type="button"
          class="terraform-tab"
          class:active={terraformTab === 'azure'}
          role="tab"
          aria-selected={terraformTab === 'azure'}
          aria-controls="terraform-panel-azure"
          id="terraform-tab-azure"
          on:click={() => (terraformTab = 'azure')}
        >
          <span class="terraform-tab-icon"><Icon icon="simple-icons:microsoftazure" aria-hidden="true" /></span>
          <span>Azure</span>
        </button>
        <button
          type="button"
          class="terraform-tab"
          class:active={terraformTab === 'gcp'}
          role="tab"
          aria-selected={terraformTab === 'gcp'}
          aria-controls="terraform-panel-gcp"
          id="terraform-tab-gcp"
          on:click={() => (terraformTab = 'gcp')}
        >
          <span class="terraform-tab-icon"><Icon icon="simple-icons:googlecloud" aria-hidden="true" /></span>
          <span>GCP</span>
        </button>
      </div>
      <div class="terraform-snippet">
        {#if terraformTab === 'aws'}
          <div id="terraform-panel-aws" role="tabpanel" aria-labelledby="terraform-tab-aws" tabindex="0" class="terraform-panel"><pre class="terraform-code"><code>provider "ipam" {'{'}
  endpoint = "https://ipam.example.com"
  token    = var.ipam_token
{'}'}

resource "ipam_environment" "prod" {'{'}
  name = "production"
  pools = [
    {'{'} name = "prod-pool", cidr = "10.0.0.0/8" {'}'}
  ]
{'}'}

resource "ipam_block" "main" {'{'}
  name           = "main-vpc"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.prod.id
  pool_id        = ipam_environment.prod.pool_ids[0]
{'}'}

# Auto-allocate: next available /24 in the block
resource "ipam_allocation" "app" {'{'}
  name           = "app-subnet"
  block_name     = ipam_block.main.name
  prefix_length  = 24
{'}'}

# Use IPAM block CIDR for VPC, allocation for subnet
resource "aws_vpc" "main" {'{'}
  cidr_block           = ipam_block.main.cidr
  enable_dns_hostnames = true
{'}'}

resource "aws_subnet" "app" {'{'}
  vpc_id            = aws_vpc.main.id
  cidr_block        = ipam_allocation.app.cidr
  availability_zone = "us-east-1a"
{'}'}</code></pre></div>
        {:else if terraformTab === 'azure'}
          <div id="terraform-panel-azure" role="tabpanel" aria-labelledby="terraform-tab-azure" tabindex="0" class="terraform-panel"><pre class="terraform-code"><code>provider "ipam" {'{'}
  endpoint = "https://ipam.example.com"
  token    = var.ipam_token
{'}'}

resource "ipam_environment" "prod" {'{'}
  name = "production"
  pools = [
    {'{'} name = "prod-pool", cidr = "10.0.0.0/8" {'}'}
  ]
{'}'}

resource "ipam_block" "main" {'{'}
  name           = "main-vnet"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.prod.id
  pool_id        = ipam_environment.prod.pool_ids[0]
{'}'}

# Auto-allocate: next available /24 in the block
resource "ipam_allocation" "app" {'{'}
  name           = "app-subnet"
  block_name     = ipam_block.main.name
  prefix_length  = 24
{'}'}

# Use IPAM block for VNet address space, allocation for subnet
resource "azurerm_virtual_network" "main" {'{'}
  name                = "main-vnet"
  address_space       = [ipam_block.main.cidr]
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
{'}'}

resource "azurerm_subnet" "app" {'{'}
  name                 = "app"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = [ipam_allocation.app.cidr]
{'}'}</code></pre></div>
        {:else}
          <div id="terraform-panel-gcp" role="tabpanel" aria-labelledby="terraform-tab-gcp" tabindex="0" class="terraform-panel"><pre class="terraform-code"><code>provider "ipam" {'{'}
  endpoint = "https://ipam.example.com"
  token    = var.ipam_token
{'}'}

resource "ipam_environment" "prod" {'{'}
  name = "production"
  pools = [
    {'{'} name = "prod-pool", cidr = "10.0.0.0/8" {'}'}
  ]
{'}'}

resource "ipam_block" "main" {'{'}
  name           = "main-network"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.prod.id
  pool_id        = ipam_environment.prod.pool_ids[0]
{'}'}

# Auto-allocate: next available /24 in the block
resource "ipam_allocation" "app" {'{'}
  name           = "app-subnet"
  block_name     = ipam_block.main.name
  prefix_length  = 24
{'}'}

# Use IPAM allocation CIDRs for GCP subnets (custom mode)
resource "google_compute_network" "main" {'{'}
  name                    = "main-network"
  auto_create_subnetworks = false
{'}'}

resource "google_compute_subnetwork" "app" {'{'}
  name          = "app"
  ip_cidr_range = ipam_allocation.app.cidr
  region        = "us-central1"
  network       = google_compute_network.main.id
{'}'}</code></pre></div>
        {/if}
      </div>
    </div>
  </section>

  <section id="user-guide" class="section docs-section" data-animate-id="docs" class:visible={visibleSections.has('docs')}>
    <p class="section-eyebrow">Documentation</p>
    <h2 class="section-title">User guide</h2>
    <div class="docs-section-links">
      <a href="#docs" class="docs-section-link">Overview</a>
      <a href="#docs/getting-started" class="docs-section-link">Getting started</a>
      <a href="#docs/environments" class="docs-section-link">Environments</a>
      <a href="#docs/networks" class="docs-section-link">Networks</a>
      <a href="#docs/integrations" class="docs-section-link">Integrations</a>
      <a href="#docs/integrations/aws" class="docs-section-link">Integrations — AWS</a>
      <a href="#docs/command-palette" class="docs-section-link">Command palette</a>
      <a href="#docs/cidr-wizard" class="docs-section-link">CIDR wizard</a>
      <a href="#docs/network-advisor" class="docs-section-link">Network Advisor</a>
      <a href="#docs/subnet-calculator" class="docs-section-link">Subnet calculator</a>
      <a href="#docs/reserved-blocks" class="docs-section-link">Reserved blocks</a>
      <a href="#docs/admin" class="docs-section-link">Admin</a>
    </div>
    <p class="docs-section-cta-wrap">
      <a href="#docs" class="docs-section-cta">Read the full user guide →</a>
    </p>
  </section>

  <section class="section cta-section" data-animate-id="cta" class:visible={visibleSections.has('cta')}>
    <div class="cta-bg" aria-hidden="true"></div>
    <div class="cta-content">
      <h2 class="cta-title">Ready to manage your IP space?</h2>
      <p class="cta-desc">Log in to the dashboard or run the Terraform provider against your IPAM instance.</p>
      <button type="button" class="btn-cta" on:click={goLogin}>Log in to IPAM</button>
    </div>
  </section>

  <footer class="landing-footer">
    <div class="footer-inner">
      <img src="{base}images/{$theme === 'light' ? 'logo-light.svg' : 'logo.svg'}" alt="IPAM" class="footer-logo" />

      <div class="footer-links">
        <a href={githubUrl} target="_blank" rel="noopener noreferrer" class="github-link github-link-footer">
          <span class="github-link-icon">
            <SocialIcons alt="" network="github" width={32} height={32} fgColor="currentColor" bgColor="transparent" />
          </span>
          <span class="github-link-label">View on GitHub</span>
          {#if githubStats}
            <span class="github-link-count"> · {formatCount(githubStats.stars)}</span>
          {/if}
        </a>
        <a href={terraformRegistryUrl} target="_blank" rel="noopener noreferrer" class="github-link github-link-footer">
          <span class="github-link-icon">
            <img src="{base}images/terraform.svg" alt="" class="footer-terraform-icon" />
          </span>
          <span class="github-link-label">Terraform Registry</span>
        </a>
      </div>
    </div>
  </footer>
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
  /* Scroll-triggered animations */
  @keyframes fadeInUp {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .landing {
    min-height: 100vh;
    background: var(--bg);
    color: var(--text);
  }

  .landing-header {
    position: sticky;
    top: 0;
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.5rem;
    max-width: 1200px;
    margin: 0 auto;
    background: radial-gradient(
      ellipse 120% 120% at 50% 100%,
      var(--accent-dim) 0%,
      transparent 55%
    );
    border-bottom: 1px solid transparent;
    box-shadow: 0 1px 0 rgba(255, 255, 255, 0.04);
    transition: background 0.25s ease, box-shadow 0.25s ease;
  }

  .landing-header.scrolled {
    background: var(--bg);
    box-shadow: 0 1px 0 var(--border), 0 4px 20px rgba(0, 0, 0, 0.06);
  }

  :global(.dark) .landing-header {
    box-shadow: 0 1px 0 rgba(0, 0, 0, 0.15);
  }

  :global(.dark) .landing-header.scrolled {
    box-shadow: 0 1px 0 var(--border), 0 4px 24px rgba(0, 0, 0, 0.3);
  }

  .landing-logo-wrap {
    display: flex;
    align-items: center;
  }

  .landing-logo {
    height: 2.25rem;
    width: auto;
    object-fit: contain;
  }

  .landing-header-actions {
    display: flex;
    align-items: center;
    gap: 1rem;
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

  /* GitHub-style Star button (dark segmented) */
  .github-link {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    text-decoration: none;
    color: var(--text-muted);
    font-size: 0.875rem;
    font-weight: 500;
    line-height: 1;
    transition: color 0.15s;
  }

  .github-link:hover {
    color: var(--text);
  }

  .landing-header-actions .github-link {
    color: var(--text);
    opacity: 0.88;
  }

  .landing-header-actions .github-link:hover {
    opacity: 1;
  }

  .github-link-icon {
    display: inline-flex;
    flex-shrink: 0;
    line-height: 0;
  }

  .github-link-icon :global(.social-icon),
  .github-link-icon :global(svg) {
    width: 2rem;
    height: 2rem;
    fill: currentColor;
  }

  .github-link-label {
    white-space: nowrap;
  }

  .github-link-count {
    opacity: 0.85;
  }

  .github-link-footer .github-link-label {
    font-size: 0.8125rem;
  }

  .landing-cta-btn {
    padding: 0.45rem 0.9rem;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text-muted);
    border: 1px solid var(--border);
    font-weight: 500;
    font-size: 0.875rem;
    cursor: pointer;
    transition: border-color 0.15s, color 0.15s;
  }

  .landing-cta-btn:hover {
    border-color: var(--text-muted);
    color: var(--text);
  }

  .landing-header-actions .landing-cta-btn {
    color: var(--text);
    border-color: var(--text-muted);
    opacity: 0.95;
  }

  .landing-header-actions .landing-cta-btn:hover {
    opacity: 1;
  }

  /* Hero */
  .hero {
    position: relative;
    padding: 4rem 1.5rem 5rem;
    max-width: 1200px;
    margin: 0 auto;
    overflow: hidden;
  }

  .hero-bg {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 120% 80% at 50% -20%, var(--accent-dim) 0%, transparent 50%),
      radial-gradient(ellipse 100% 70% at 50% 0%, var(--accent-dim) 0%, transparent 55%);
    pointer-events: none;
  }

  .hero-bg-title {
    position: absolute;
    inset: 0;
    background: radial-gradient(
      ellipse 90% 60% at 50% 35%,
      var(--accent-dim) 0%,
      transparent 60%
    );
    pointer-events: none;
  }

  .hero-content {
    position: relative;
    text-align: center;
    max-width: 640px;
    margin: 0 auto;
  }

  .hero.visible .hero-eyebrow {
    animation: fadeInUp 0.6s ease-out forwards;
  }

  .hero.visible .hero-title {
    animation: fadeInUp 0.6s ease-out 0.08s both;
  }

  .hero.visible .hero-subtitle {
    animation: fadeInUp 0.6s ease-out 0.16s both;
  }

  .hero.visible .hero-actions {
    animation: fadeInUp 0.6s ease-out 0.24s both;
  }

  .hero.visible .hero-dashboard-wrap {
    animation: fadeInUp 0.8s ease-out 0.35s both;
  }

  .hero-eyebrow {
    font-size: 0.8125rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--accent);
    margin: 0 0 0.75rem;
    opacity: 0;
  }

  .hero-title {
    font-size: clamp(2.25rem, 5.5vw, 3.25rem);
    font-weight: 700;
    line-height: 1.15;
    margin: 0 0 1rem;
    letter-spacing: -0.03em;
    opacity: 0;
  }

  .hero-subtitle {
    font-size: 1.2rem;
    color: var(--text-muted);
    line-height: 1.6;
    margin: 0 0 2rem;
    max-width: 28em;
    margin-left: auto;
    margin-right: auto;
    opacity: 0;
  }

  .hero-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    justify-content: center;
    opacity: 0;
  }

  .hero-dashboard-wrap {
    opacity: 0;
  }

  .btn-hero-primary {
    display: inline-block;
    padding: 0.75rem 1.5rem;
    border-radius: 999px;
    background: var(--accent);
    color: var(--bg);
    text-decoration: none;
    font-weight: 600;
    font-size: 0.9375rem;
    border: none;
    transition: transform 0.2s ease, box-shadow 0.2s ease, opacity 0.2s;
  }

  .btn-hero-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 8px 24px var(--accent-dim);
  }

  .btn-hero-secondary {
    padding: 0.75rem 1.5rem;
    border-radius: 999px;
    background: transparent;
    color: var(--text);
    border: 2px solid var(--border);
    font-weight: 600;
    font-size: 0.9375rem;
    cursor: pointer;
    transition: border-color 0.2s, color 0.2s, transform 0.2s ease;
  }

  .btn-hero-secondary:hover {
    border-color: var(--accent);
    color: var(--accent);
    transform: translateY(-1px);
  }

  /* Dashboard screenshot (Netmaker-style) */
  .hero-dashboard-wrap {
    position: relative;
    max-width: 900px;
    margin: 3rem auto 0;
    padding: 0 1rem;
  }

  .hero-dashboard-glow {
    position: absolute;
    inset: -20%;
    background: radial-gradient(
      ellipse 70% 50% at 50% 50%,
      var(--accent-dim) 0%,
      transparent 70%
    );
    pointer-events: none;
    opacity: 0.8;
  }

  .hero-dashboard-frame {
    position: relative;
    border-radius: 12px;
    overflow: hidden;
    box-shadow:
      0 4px 6px rgba(0, 0, 0, 0.07),
      0 10px 40px rgba(0, 0, 0, 0.12),
      0 0 0 1px var(--border);
    background: var(--surface);
  }

  .hero-dashboard-titlebar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    background: var(--surface-elevated, var(--surface));
    border-bottom: 1px solid var(--border);
  }

  .hero-dashboard-dots {
    display: flex;
    gap: 6px;
  }

  .hero-dashboard-dots span {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--text-muted);
    opacity: 0.5;
  }

  .hero-dashboard-dots span:nth-child(1) { background: #ff5f57; opacity: 1; }
  .hero-dashboard-dots span:nth-child(2) { background: #febc2e; opacity: 1; }
  .hero-dashboard-dots span:nth-child(3) { background: #28c840; opacity: 1; }

  .hero-dashboard-url {
    flex: 1;
    text-align: center;
    font-size: 0.75rem;
    font-family: var(--font-mono);
    color: var(--text-muted);
  }

  .hero-dashboard-inner {
    position: relative;
    overflow: hidden;
    max-height: 420px;
  }

  .hero-dashboard-img {
    width: 100%;
    height: auto;
    display: block;
    vertical-align: top;
  }

  .hero-dashboard-img-dark {
    display: none;
  }

  .hero-dashboard-img-light {
    display: block;
  }

  :global(html.dark) .hero-dashboard-inner .hero-dashboard-img-light {
    display: none;
  }

  :global(html.dark) .hero-dashboard-inner .hero-dashboard-img-dark {
    display: block;
  }

  /* Command palette section */
  .command-palette-section.visible .command-palette-wrap {
    animation: fadeInUp 0.6s ease-out 0.1s both;
  }

  .command-palette-wrap {
    position: relative;
    max-width: 560px;
    margin: 2.5rem auto 0;
    padding: 0 1rem;
    opacity: 0;
  }

  .command-palette-glow {
    position: absolute;
    inset: -15%;
    background: radial-gradient(
      ellipse 70% 50% at 50% 50%,
      var(--accent-dim) 0%,
      transparent 65%
    );
    pointer-events: none;
    opacity: 0.6;
  }

  .command-palette-frame {
    position: relative;
    border-radius: 12px;
    overflow: hidden;
    box-shadow:
      0 4px 6px rgba(0, 0, 0, 0.07),
      0 10px 40px rgba(0, 0, 0, 0.12),
      0 0 0 1px var(--border);
    background: var(--surface);
  }

  .command-palette-titlebar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 1rem;
    background: var(--surface-elevated, var(--surface));
    border-bottom: 1px solid var(--border);
  }

  .command-palette-dots {
    display: flex;
    gap: 6px;
  }

  .command-palette-dots span {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--text-muted);
    opacity: 0.5;
  }

  .command-palette-dots span:nth-child(1) { background: #ff5f57; opacity: 1; }
  .command-palette-dots span:nth-child(2) { background: #febc2e; opacity: 1; }
  .command-palette-dots span:nth-child(3) { background: #28c840; opacity: 1; }

  .command-palette-hint {
    flex: 1;
    text-align: center;
    font-size: 0.7rem;
    font-family: var(--font-mono);
    color: var(--text-muted);
  }

  .command-palette-inner {
    position: relative;
    overflow: hidden;
    padding: 1rem 1.5rem 1.5rem;
  }

  .command-palette-svg {
    width: 100%;
    max-width: 400px;
    height: auto;
    display: block;
    margin: 0 auto;
  }

  /* Section */
  .section {
    padding: 5rem 1.5rem;
    max-width: 1200px;
    margin: 0 auto;
  }

  .section-eyebrow {
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.14em;
    color: var(--accent);
    text-align: center;
    margin: 0 0 0.5rem;
    opacity: 0;
  }

  .section.visible .section-eyebrow {
    animation: fadeInUp 0.5s ease-out forwards;
  }

  .section-title {
    font-size: clamp(1.75rem, 4vw, 2.25rem);
    font-weight: 700;
    text-align: center;
    margin: 0 0 0.5rem;
    letter-spacing: -0.02em;
    opacity: 0;
  }

  .section.visible .section-title {
    animation: fadeInUp 0.5s ease-out 0.06s both;
  }

  .section-desc {
    text-align: center;
    color: var(--text-muted);
    max-width: 560px;
    margin: 0 auto 2.5rem;
    line-height: 1.65;
    font-size: 1rem;
    opacity: 0;
  }

  .section.visible .section-desc {
    animation: fadeInUp 0.5s ease-out 0.12s both;
  }

  /* Features */
  .features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1.5rem;
  }

  .feature-card {
    padding: 1.75rem;
    border-radius: 16px;
    background: var(--surface);
    border: 1px solid var(--border);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
    transition: transform 0.25s ease, box-shadow 0.25s ease, border-color 0.2s;
    opacity: 0;
  }

  .features.visible .feature-card {
    animation: fadeInUp 0.5s ease-out both;
  }

  .features.visible .feature-card:nth-child(1) { animation-delay: 0.12s; }
  .features.visible .feature-card:nth-child(2) { animation-delay: 0.16s; }
  .features.visible .feature-card:nth-child(3) { animation-delay: 0.2s; }
  .features.visible .feature-card:nth-child(4) { animation-delay: 0.24s; }
  .features.visible .feature-card:nth-child(5) { animation-delay: 0.28s; }
  .features.visible .feature-card:nth-child(6) { animation-delay: 0.32s; }
  .features.visible .feature-card:nth-child(7) { animation-delay: 0.36s; }
  .features.visible .feature-card:nth-child(8) { animation-delay: 0.4s; }

  .feature-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--border);
    border-color: var(--accent);
  }

  :global(.dark) .feature-card:hover {
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25), 0 0 0 1px var(--accent);
  }

  .feature-icon-wrap {
    width: 52px;
    height: 52px;
    border-radius: 14px;
    background: var(--accent-dim);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 1.25rem;
    transition: background 0.2s, transform 0.2s;
  }

  .feature-card:hover .feature-icon-wrap {
    background: var(--accent);
    transform: scale(1.05);
  }

  .feature-card:hover .feature-icon {
    color: var(--bg);
  }

  .feature-icon {
    width: 26px;
    height: 26px;
    color: var(--accent);
    transition: color 0.2s;
  }

  .feature-title {
    font-size: 1.125rem;
    font-weight: 600;
    margin: 0 0 0.5rem;
    letter-spacing: -0.01em;
  }

  .feature-desc {
    font-size: 0.9rem;
    color: var(--text-muted);
    line-height: 1.55;
    margin: 0;
  }

  /* Integrations section - IPAM ↔ AWS / Azure / GCP (Orizon-style animations) */
  @keyframes integrationFlow {
    to { stroke-dashoffset: -60; }
  }

  @keyframes hubFloat {
    0%, 100% { transform: translateY(0) scale(1); opacity: 1; }
    50% { transform: translateY(-6px) scale(1.02); opacity: 0.98; }
  }

  @keyframes cardReveal {
    from {
      opacity: 0;
      transform: translateY(20px) scale(0.96);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  @keyframes cardFloat {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-4px); }
  }

  .integrations-section {
    position: relative;
  }

  .integrations-section.visible .integrations-diagram-wrap {
    animation: fadeInUp 0.8s ease-out 0.1s both;
  }

  .integrations-section.visible .integration-node-card {
    animation: cardReveal 0.6s cubic-bezier(0.22, 1, 0.36, 1) both;
  }

  .integrations-section.visible .integration-node-onprem {
    animation-delay: 0.25s;
  }

  .integrations-section.visible .integration-node-aws {
    animation-delay: 0.4s;
  }

  .integrations-section.visible .integration-node-azure {
    animation-delay: 0.55s;
  }

  .integrations-section.visible .integration-node-gcp {
    animation-delay: 0.7s;
  }

  /* Gentle float on cards (Orizon-style), staggered phase */
  .integrations-section.visible .integration-node-card:nth-child(2) {
    animation-name: cardReveal, cardFloat;
    animation-duration: 0.6s, 4s;
    animation-delay: 0.25s, 0.4s;
    animation-timing-function: cubic-bezier(0.22, 1, 0.36, 1), ease-in-out;
    animation-iteration-count: 1, infinite;
    animation-fill-mode: both, both;
  }

  .integrations-section.visible .integration-node-card:nth-child(3) {
    animation-name: cardReveal, cardFloat;
    animation-duration: 0.6s, 4s;
    animation-delay: 0.4s, 0.6s;
    animation-timing-function: cubic-bezier(0.22, 1, 0.36, 1), ease-in-out;
    animation-iteration-count: 1, infinite;
    animation-fill-mode: both, both;
  }

  .integrations-section.visible .integration-node-card:nth-child(4) {
    animation-name: cardReveal, cardFloat;
    animation-duration: 0.6s, 4s;
    animation-delay: 0.55s, 0.8s;
    animation-timing-function: cubic-bezier(0.22, 1, 0.36, 1), ease-in-out;
    animation-iteration-count: 1, infinite;
    animation-fill-mode: both, both;
  }

  .integrations-section.visible .integration-node-card:nth-child(5) {
    animation-name: cardReveal, cardFloat;
    animation-duration: 0.6s, 4s;
    animation-delay: 0.7s, 1s;
    animation-timing-function: cubic-bezier(0.22, 1, 0.36, 1), ease-in-out;
    animation-iteration-count: 1, infinite;
    animation-fill-mode: both, both;
  }

  .integrations-diagram-wrap {
    position: relative;
    max-width: 900px;
    margin: 2.5rem auto 0;
    padding: 1.5rem;
    opacity: 0;
  }

  .integrations-glow {
    position: absolute;
    inset: -20%;
    background: radial-gradient(
      ellipse 80% 60% at 50% 50%,
      var(--accent-dim) 0%,
      transparent 60%
    );
    pointer-events: none;
    opacity: 0.7;
  }

  .integrations-diagram-inner {
    position: relative;
    width: 100%;
    aspect-ratio: 820 / 520;
    max-width: 820px;
    margin: 0 auto;
  }

  .integrations-svg {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    display: block;
  }

  .integrations-connectors .integration-line {
    stroke-dashoffset: 0;
    animation: integrationFlow 1.5s linear infinite;
  }

  .integrations-connectors .integration-line-onprem {
    animation-delay: 0.25s;
  }

  .integrations-connectors .integration-line-azure {
    animation-delay: 0.5s;
  }

  .integrations-connectors .integration-line-gcp {
    animation-delay: 1s;
  }

  .integration-dot {
    opacity: 0.9;
  }

  .integration-hub-bg {
    animation: hubFloat 4s ease-in-out infinite;
  }

  /* Cloud provider overlay cards with logos */
  .integration-coming-soon {
    position: absolute;
    top: 0.25rem;
    right: 0.25rem;
    padding: 0.15rem 0.4rem;
    border-radius: 4px;
    font-size: 0.55rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    color: var(--accent);
    background: var(--accent-dim);
    border: 1px solid var(--accent);
    line-height: 1;
    white-space: nowrap;
  }

  .integration-node-card {
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.4rem;
    padding: 0.75rem 1rem;
    padding-top: 1.5rem;
    border-radius: 18px;
    border: 2px solid;
    background: var(--surface);
    box-shadow: 0 6px 24px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--border);
    transition: transform 0.3s cubic-bezier(0.22, 1, 0.36, 1), box-shadow 0.3s ease;
    opacity: 0;
  }

  .integration-node-card:hover {
    transform: translateY(-6px) scale(1.03);
    box-shadow: 0 12px 36px rgba(0, 0, 0, 0.12), 0 0 0 1px var(--border);
  }

  :global(.dark) .integration-node-card {
    box-shadow: 0 6px 28px rgba(0, 0, 0, 0.28), 0 0 0 1px var(--border);
  }

  :global(.dark) .integration-node-card:hover {
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4), 0 0 0 1px var(--border);
  }

  /* Balanced cross: all nodes equal size (24% × 24%) */
  .integration-node-onprem {
    left: 2%;
    top: 38%;
    width: 24%;
    height: 24%;
    border-color: var(--text-muted);
    background: linear-gradient(135deg, rgba(128, 128, 128, 0.12) 0%, rgba(128, 128, 128, 0.03) 100%);
  }

  .integration-node-aws {
    left: 38%;
    top: 2%;
    width: 24%;
    height: 24%;
    border-color: #FF9900;
    background: linear-gradient(135deg, rgba(255, 153, 0, 0.1) 0%, rgba(255, 153, 0, 0.02) 100%);
  }

  .integration-node-azure {
    left: 74%;
    top: 38%;
    width: 24%;
    height: 24%;
    border-color: #0078D4;
    background: linear-gradient(135deg, rgba(0, 120, 212, 0.1) 0%, rgba(0, 120, 212, 0.02) 100%);
  }

  .integration-node-gcp {
    left: 38%;
    top: 74%;
    width: 24%;
    height: 24%;
    border-color: #4285F4;
    background: linear-gradient(135deg, rgba(66, 133, 244, 0.1) 0%, rgba(52, 168, 83, 0.04) 100%);
  }

  .integration-node-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    flex-shrink: 0;
  }

  .integration-node-icon :global(svg) {
    width: 100%;
    height: 100%;
  }

  .integration-node-aws .integration-node-icon {
    color: #FF9900;
  }

  .integration-node-azure .integration-node-icon {
    color: #0078D4;
  }

  .integration-node-gcp .integration-node-icon {
    color: #4285F4;
  }

  .integration-node-onprem .integration-node-icon {
    color: var(--text-muted);
  }

  .integration-node-title {
    font-size: 1rem;
    font-weight: 700;
    color: var(--text);
    line-height: 1.2;
  }

  .integration-node-subtitle {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-weight: 500;
    line-height: 1.2;
    text-align: center;
  }

  /* API section */
  .api-section {
    position: relative;
    border-radius: 1rem;
    overflow: hidden;
  }

  .api-bg {
    position: absolute;
    inset: 0;
    background: linear-gradient(180deg, transparent 0%, var(--accent-dim) 30%, transparent 70%);
    pointer-events: none;
  }

  .api-content {
    position: relative;
  }

  .api-section.visible .api-diagram {
    animation: fadeInUp 0.6s ease-out 0.15s both;
  }

  .api-diagram {
    margin-bottom: 2rem;
    padding: 1.5rem;
    border-radius: var(--radius);
    filter: drop-shadow(0 8px 24px rgba(0, 0, 0, 0.12)) drop-shadow(0 2px 8px rgba(0, 0, 0, 0.08));
    opacity: 0;
  }

  :global(.dark) .api-diagram {
    filter: drop-shadow(0 8px 32px rgba(0, 0, 0, 0.4)) drop-shadow(0 2px 12px rgba(0, 0, 0, 0.2));
  }

  .api-flow-svg {
    width: 100%;
    max-width: 600px;
    height: auto;
    margin: 0 auto;
    display: block;
  }

  /* Terraform */
  .terraform-section {
    position: relative;
    border-radius: 1rem;
    overflow: hidden;
  }

  .terraform-bg {
    position: absolute;
    inset: 0;
    background: linear-gradient(135deg, var(--accent-dim) 0%, transparent 50%);
    pointer-events: none;
  }

  .terraform-content {
    position: relative;
  }

  .terraform-section.visible .terraform-grid,
  .terraform-section.visible .terraform-tabs,
  .terraform-section.visible .terraform-snippet {
    animation: fadeInUp 0.5s ease-out both;
  }

  .terraform-section.visible .terraform-grid { animation-delay: 0.08s; }
  .terraform-section.visible .terraform-tabs { animation-delay: 0.14s; opacity: 0; }
  .terraform-section.visible .terraform-snippet { animation-delay: 0.2s; opacity: 0; }

  .terraform-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    margin-bottom: 0.5rem;
  }

  .terraform-title-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    justify-content: center;
  }

  .terraform-logo {
    width: 2.5rem;
    height: 2.5rem;
    flex-shrink: 0;
  }

  .terraform-title {
    margin: 0;
  }

  .terraform-registry-wrap {
    text-align: center;
    margin-bottom: 2rem;
    opacity: 0;
  }

  .terraform-section.visible .terraform-registry-wrap {
    animation: fadeInUp 0.5s ease-out 0.1s both;
  }

  .terraform-registry-link {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 1.25rem;
    border-radius: 999px;
    background: var(--surface);
    border: 1.5px solid var(--border);
    color: var(--text);
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 600;
    transition: border-color 0.2s, background 0.2s, transform 0.2s ease, box-shadow 0.2s;
  }

  .terraform-registry-link:hover {
    border-color: #5c4ee5;
    background: rgba(92, 78, 229, 0.08);
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(92, 78, 229, 0.15);
    color: var(--text);
  }

  .terraform-registry-icon {
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
  }

  .terraform-registry-arrow {
    width: 1rem;
    height: 1rem;
    flex-shrink: 0;
    opacity: 0.6;
    transition: opacity 0.15s, transform 0.15s;
  }

  .terraform-registry-link:hover .terraform-registry-arrow {
    opacity: 1;
    transform: translate(1px, -1px);
  }

  .terraform-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 1.5rem;
    margin-bottom: 2rem;
    opacity: 0;
  }

  .terraform-tabs {
    opacity: 0;
  }

  .terraform-snippet {
    opacity: 0;
  }

  .terraform-card {
    padding: 1.5rem;
    border-radius: 14px;
    background: var(--surface);
    border: 1px solid var(--border);
    transition: transform 0.2s ease, border-color 0.2s, box-shadow 0.2s;
  }

  .terraform-card:hover {
    transform: translateY(-2px);
    border-color: var(--accent);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.06);
  }

  .terraform-card-title {
    font-size: 1rem;
    font-weight: 600;
    margin: 0 0 0.75rem;
  }

  .terraform-list {
    list-style: none;
    margin: 0;
    padding: 0;
    font-size: 0.875rem;
    color: var(--text-muted);
  }

  .terraform-list li {
    padding: 0.25rem 0;
  }

  .terraform-list code {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--accent);
  }

  .terraform-tabs {
    display: flex;
    gap: 0.25rem;
    margin-bottom: 0;
    padding: 0.5rem 0.5rem 0 0.5rem;
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-bottom: none;
    border-radius: var(--radius) var(--radius) 0 0;
  }

  .terraform-tab {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-muted);
    background: transparent;
    border: none;
    border-radius: var(--radius) var(--radius) 0 0;
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }

  .terraform-tab-icon {
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
    color: inherit;
  }

  .terraform-tab:hover {
    color: var(--text);
    background: var(--surface);
  }

  .terraform-tab.active {
    color: var(--accent);
    background: var(--surface);
    border-bottom: 2px solid var(--surface);
    margin-bottom: -1px;
  }

  .terraform-snippet {
    border-radius: 0 0 var(--radius) var(--radius);
    overflow: hidden;
    border: 1px solid var(--border);
    background: var(--surface-elevated);
  }

  .terraform-code {
    margin: 0;
    padding: 1.25rem;
    font-size: 0.8rem;
    font-family: var(--font-mono);
    line-height: 1.5;
    overflow-x: auto;
  }

  .terraform-code code {
    color: var(--text);
  }

  /* CTA */
  /* Docs / User guide section */
  .docs-section {
    position: relative;
  }

  .docs-section.visible .docs-section-links,
  .docs-section.visible .docs-section-cta-wrap {
    animation: fadeInUp 0.5s ease-out 0.1s both;
  }

  .docs-section.visible .docs-section-cta-wrap {
    animation-delay: 0.18s;
  }

  .docs-section-links {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 0.75rem;
    max-width: 720px;
    margin: 0 auto 1.5rem;
    opacity: 0;
  }

  .docs-section-link {
    display: block;
    padding: 0.7rem 1rem;
    border-radius: 12px;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 500;
    transition: border-color 0.2s, background 0.2s, transform 0.2s ease;
  }

  .docs-section-link:hover {
    border-color: var(--accent);
    background: var(--accent-dim);
    color: var(--text);
    transform: translateY(-2px);
  }

  .docs-section-cta-wrap {
    text-align: center;
    margin: 0;
    opacity: 0;
  }

  .docs-section-cta {
    display: inline-block;
    padding: 0.5rem 1rem;
    border-radius: var(--radius);
    background: transparent;
    color: var(--accent);
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 500;
    border: 1px solid var(--border);
    transition: border-color 0.15s, background 0.15s;
  }

  .docs-section-cta:hover {
    border-color: var(--accent);
    background: var(--accent-dim);
  }

  .cta-section {
    position: relative;
    text-align: center;
  }

  .cta-section.visible .cta-content {
    animation: fadeInUp 0.6s ease-out 0.1s both;
  }

  .cta-bg {
    position: absolute;
    inset: 0;
    background: radial-gradient(ellipse 70% 50% at 50% 100%, var(--accent-dim) 0%, transparent 70%);
    pointer-events: none;
  }

  .cta-content {
    position: relative;
    opacity: 0;
  }

  .cta-title {
    font-size: clamp(1.75rem, 4vw, 2.25rem);
    font-weight: 700;
    margin: 0 0 0.5rem;
    letter-spacing: -0.02em;
  }

  .cta-desc {
    color: var(--text-muted);
    margin: 0 0 1.5rem;
    font-size: 1.05rem;
  }

  .btn-cta {
    padding: 0.75rem 1.5rem;
    border-radius: 999px;
    background: var(--accent);
    color: var(--bg);
    border: none;
    font-weight: 600;
    font-size: 0.9375rem;
    cursor: pointer;
    transition: transform 0.2s ease, box-shadow 0.2s ease, opacity 0.2s;
  }

  .btn-cta:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px var(--accent-dim);
  }

  /* Footer */
  .landing-footer {
    padding: 2rem 1.5rem;
    border-top: 1px solid var(--border);
    text-align: center;
  }

  .footer-inner {
    max-width: 1200px;
    margin: 0 auto;
  }

  .footer-logo {
    height: 1.75rem;
    width: auto;
    opacity: 0.8;
    margin-bottom: 0.5rem;
  }


  .footer-links {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1.5rem;
    margin-top: 1rem;
    flex-wrap: wrap;
  }

  .footer-terraform-icon {
    width: 1.5rem;
    height: 1.5rem;
  }
</style>
