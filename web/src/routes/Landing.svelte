<script>
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import SocialIcons from '@rodneylab/svelte-social-icons'
  import { theme } from '../lib/theme.js'

  const base = (import.meta.env.BASE_URL || '/').replace(/\/+$/, '') + '/'
  const githubUrl = 'https://github.com/JakeNeyer/ipam'
  const githubApiUrl = 'https://api.github.com/repos/JakeNeyer/ipam'

  let githubStats = null

  /** @type {'azure' | 'aws' | 'gcp'} */
  let terraformTab = 'aws'

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
  })
</script>

<div class="landing">
  <header class="landing-header">
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

  <section class="hero">
    <div class="hero-bg" aria-hidden="true"></div>
    <div class="hero-bg-title" aria-hidden="true"></div>
    <div class="hero-content">
      <h1 class="hero-title">IP address management, simplified</h1>
      <p class="hero-subtitle">
        Stop using spreadsheets to manage enterprise networks.
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

  <section id="features" class="section features">
    <h2 class="section-title">Core features</h2>
    <p class="section-desc">Environments, blocks, allocations, reserved blocks, a CIDR wizard, subnet calculator, network advisor, and diagram export.</p>
    <div class="features-grid">
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
            <polyline points="9 22 9 12 15 12 15 22" />
          </svg>
        </div>
        <h3 class="feature-title">Environments</h3>
        <p class="feature-desc">Group network blocks by environment (e.g. staging, production).</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="6" width="20" height="12" rx="2" />
            <path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01" />
          </svg>
        </div>
        <h3 class="feature-title">Network blocks</h3>
        <p class="feature-desc">Define ranges of IP address as network blocks.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
            <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
            <line x1="12" y1="22.08" x2="12" y2="12" />
          </svg>
        </div>
        <h3 class="feature-title">Allocations</h3>
        <p class="feature-desc">Allocate CIDR ranges as in-use networks.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon-wrap">
          <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3" />
            <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
          </svg>
        </div>
        <h3 class="feature-title">CIDR wizard</h3>
        <p class="feature-desc">Easily design networks with suggestions for optimal space usage.</p>
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
            <rect x="3" y="3" width="7" height="7" rx="1" />
            <rect x="14" y="3" width="7" height="7" rx="1" />
            <rect x="8.5" y="14" width="7" height="7" rx="1" />
            <path d="M10 7h4M17.5 10v4M6.5 10v4M10 17h4" />
          </svg>
        </div>
        <h3 class="feature-title">Network Diagram Export</h3>
        <p class="feature-desc">Generate draw.io-compatible diagrams of your network topology.</p>
      </div>
    </div>
    <div class="coming-soon-wrap">
      <p class="coming-soon-label">Coming soon</p>
      <div class="coming-soon-grid">
        <div class="feature-card feature-card-coming-soon">
          <div class="feature-icon-wrap">
            <svg class="feature-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 7h18M3 12h18M3 17h18" />
              <path d="M7 3v18M12 3v18M17 3v18" />
            </svg>
          </div>
          <h3 class="feature-title">Cloud Provider Inventory</h3>
          <p class="feature-desc">Track networks by plugging in directly to your cloud provider.</p>
        </div>
      </div>
    </div>
  </section>

  <section id="command-palette" class="section command-palette-section">
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
            <text x="76" y="57" fill="var(--text-muted)" font-size="12" font-family="system-ui, sans-serif">Search environments, blocks, allocations...</text>
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

  <section id="api" class="section api-section">
    <div class="api-bg" aria-hidden="true"></div>
    <div class="api-content">
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
          <rect x="290" y="12" width="120" height="32" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="28" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Environments</text>
          <rect x="290" y="48" width="120" height="32" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="64" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Network Blocks</text>
          <rect x="290" y="84" width="120" height="32" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="100" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Allocations</text>
          <rect x="290" y="120" width="120" height="32" rx="4" fill="url(#apiClientGrad)" stroke="var(--accent)" stroke-width="1" />
          <text x="350" y="136" text-anchor="middle" fill="var(--text)" font-size="8" font-weight="600">Reserved Blocks</text>
        </svg>
      </div>
    </div>
  </section>

  <section id="terraform" class="section terraform-section">
    <div class="terraform-bg" aria-hidden="true"></div>
    <div class="terraform-content">
      <div class="terraform-header">
        <h2 class="section-title terraform-title">Terraform provider</h2>
      </div>
      <p class="section-desc">Use IPAM with your favorite IaC tooling.</p>
      <div class="terraform-grid">
        <div class="terraform-card">
          <h3 class="terraform-card-title">Resources</h3>
          <ul class="terraform-list">
            <li><code>ipam_environment</code></li>
            <li><code>ipam_block</code></li>
            <li><code>ipam_allocation</code></li>
            <li><code>ipam_reserved_block</code></li>
          </ul>
        </div>
        <div class="terraform-card">
          <h3 class="terraform-card-title">Data sources</h3>
          <ul class="terraform-list">
            <li><code>ipam_environment</code> / <code>ipam_environments</code></li>
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
          <Icon icon="simple-icons:amazonaws" class="terraform-tab-icon" aria-hidden="true" />
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
          <Icon icon="simple-icons:microsoftazure" class="terraform-tab-icon" aria-hidden="true" />
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
          <Icon icon="simple-icons:googlecloud" class="terraform-tab-icon" aria-hidden="true" />
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
{'}'}

resource "ipam_block" "main" {'{'}
  name           = "main-vpc"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.prod.id
{'}'}

resource "ipam_allocation" "app" {'{'}
  name       = "app-subnet"
  block_name = ipam_block.main.name
  cidr       = "10.0.1.0/24"
{'}'}

# Use IPAM block CIDR for VPC, allocation for subnet
resource "aws_vpc" "main" {'{'}
  cidr_block           = ipam_block.main.cidr
  enable_dns_hostnames = true
{'}'}

resource "aws_subnet" "app" {'{'}
  vpc_id     = aws_vpc.main.id
  cidr_block = ipam_allocation.app.cidr
  availability_zone = "us-east-1a"
{'}'}</code></pre></div>
        {:else if terraformTab === 'azure'}
          <div id="terraform-panel-azure" role="tabpanel" aria-labelledby="terraform-tab-azure" tabindex="0" class="terraform-panel"><pre class="terraform-code"><code>provider "ipam" {'{'}
  endpoint = "https://ipam.example.com"
  token    = var.ipam_token
{'}'}

resource "ipam_environment" "prod" {'{'}
  name = "production"
{'}'}

resource "ipam_block" "main" {'{'}
  name           = "main-vnet"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.prod.id
{'}'}

resource "ipam_allocation" "app" {'{'}
  name       = "app-subnet"
  block_name = ipam_block.main.name
  cidr       = "10.0.1.0/24"
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
{'}'}

resource "ipam_block" "main" {'{'}
  name           = "main-network"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.prod.id
{'}'}

resource "ipam_allocation" "app" {'{'}
  name       = "app-subnet"
  block_name = ipam_block.main.name
  cidr       = "10.0.1.0/24"
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

  <section id="user-guide" class="section docs-section">
    <h2 class="section-title">User guide</h2>
    <div class="docs-section-links">
      <a href="#docs" class="docs-section-link">Overview</a>
      <a href="#docs/getting-started" class="docs-section-link">Getting started</a>
      <a href="#docs/environments" class="docs-section-link">Environments</a>
      <a href="#docs/networks" class="docs-section-link">Networks</a>
      <a href="#docs/command-palette" class="docs-section-link">Command palette</a>
      <a href="#docs/cidr-wizard" class="docs-section-link">CIDR wizard</a>
      <a href="#docs/network-advisor" class="docs-section-link">Network Advisor</a>
      <a href="#docs/subnet-calculator" class="docs-section-link">Subnet calculator</a>
      <a href="#docs/reserved-blocks" class="docs-section-link">Reserved blocks</a>
    </div>
    <p class="docs-section-cta-wrap">
      <a href="#docs" class="docs-section-cta">Read the full user guide →</a>
    </p>
  </section>

  <section class="section cta-section">
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
      <p class="footer-copy">IP address management for environments, blocks, and allocations.</p>
      <a href={githubUrl} target="_blank" rel="noopener noreferrer" class="github-link github-link-footer">
      <span class="github-link-icon">
        <SocialIcons alt="" network="github" width={32} height={32} fgColor="currentColor" bgColor="transparent" />
      </span>
      <span class="github-link-label">View on GitHub</span>
      {#if githubStats}
        <span class="github-link-count"> · {formatCount(githubStats.stars)}</span>
      {/if}
    </a>
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
  .landing {
    min-height: 100vh;
    background: var(--bg);
    color: var(--text);
  }

  .landing-header {
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
  }

  :global(.dark) .landing-header {
    box-shadow: 0 1px 0 rgba(0, 0, 0, 0.15);
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

  .hero-title {
    font-size: clamp(2rem, 5vw, 3rem);
    font-weight: 700;
    line-height: 1.2;
    margin: 0 0 1rem;
    letter-spacing: -0.02em;
  }

  .hero-subtitle {
    font-size: 1.125rem;
    color: var(--text-muted);
    line-height: 1.6;
    margin: 0 0 2rem;
  }

  .hero-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    justify-content: center;
  }

  .btn-hero-primary {
    display: inline-block;
    padding: 0.6rem 1.25rem;
    border-radius: var(--radius);
    background: var(--accent-dim);
    color: var(--accent);
    text-decoration: none;
    font-weight: 500;
    font-size: 0.9375rem;
    border: 1px solid transparent;
    transition: background 0.15s, border-color 0.15s;
  }

  .btn-hero-primary:hover {
    background: var(--accent-dim);
    border-color: var(--accent);
  }

  .btn-hero-secondary {
    padding: 0.6rem 1.25rem;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text-muted);
    border: 1px solid var(--border);
    font-weight: 500;
    font-size: 0.9375rem;
    cursor: pointer;
    transition: border-color 0.15s, color 0.15s;
  }

  .btn-hero-secondary:hover {
    border-color: var(--text-muted);
    color: var(--text);
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
  .command-palette-wrap {
    position: relative;
    max-width: 560px;
    margin: 2.5rem auto 0;
    padding: 0 1rem;
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
    padding: 4rem 1.5rem;
    max-width: 1200px;
    margin: 0 auto;
  }

  .section-title {
    font-size: 1.75rem;
    font-weight: 700;
    text-align: center;
    margin: 0 0 0.5rem;
  }

  .section-desc {
    text-align: center;
    color: var(--text-muted);
    max-width: 560px;
    margin: 0 auto 2.5rem;
    line-height: 1.6;
  }

  /* Features */
  .features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1.5rem;
  }

  .feature-card {
    padding: 1.5rem;
    border-radius: var(--radius);
    background: var(--surface);
    border: 1px solid var(--border);
    box-shadow: var(--shadow-sm);
  }

  .feature-icon-wrap {
    width: 48px;
    height: 48px;
    border-radius: var(--radius);
    background: var(--accent-dim);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 1rem;
  }

  .feature-icon {
    width: 24px;
    height: 24px;
    color: var(--accent);
  }

  .feature-title {
    font-size: 1.125rem;
    font-weight: 600;
    margin: 0 0 0.5rem;
  }

  .feature-desc {
    font-size: 0.9rem;
    color: var(--text-muted);
    line-height: 1.5;
    margin: 0;
  }

  .coming-soon-wrap {
    margin-top: 2rem;
  }

  .coming-soon-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1.5rem;
  }

  .coming-soon-label {
    margin: 0 0 0.75rem;
    text-align: left;
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--accent);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .feature-card-coming-soon {
    border-style: dashed;
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

  .api-diagram {
    margin-bottom: 2rem;
    padding: 1.5rem;
    border-radius: var(--radius);
    filter: drop-shadow(0 8px 24px rgba(0, 0, 0, 0.12)) drop-shadow(0 2px 8px rgba(0, 0, 0, 0.08));
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

  .terraform-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    margin-bottom: 0.5rem;
  }

  .terraform-title {
    margin: 0;
  }

  .terraform-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 1.5rem;
    margin-bottom: 2rem;
  }

  .terraform-card {
    padding: 1.25rem;
    border-radius: var(--radius);
    background: var(--surface);
    border: 1px solid var(--border);
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

  .docs-section-links {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 0.75rem;
    max-width: 720px;
    margin: 0 auto 1.5rem;
  }

  .docs-section-link {
    display: block;
    padding: 0.6rem 0.75rem;
    border-radius: var(--radius);
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 500;
    transition: border-color 0.15s, background 0.15s;
  }

  .docs-section-link:hover {
    border-color: var(--accent);
    background: var(--accent-dim);
    color: var(--text);
  }

  .docs-section-cta-wrap {
    text-align: center;
    margin: 0;
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

  .cta-bg {
    position: absolute;
    inset: 0;
    background: radial-gradient(ellipse 70% 50% at 50% 100%, var(--accent-dim) 0%, transparent 70%);
    pointer-events: none;
  }

  .cta-content {
    position: relative;
  }

  .cta-title {
    font-size: 1.75rem;
    font-weight: 700;
    margin: 0 0 0.5rem;
  }

  .cta-desc {
    color: var(--text-muted);
    margin: 0 0 1.5rem;
  }

  .btn-cta {
    padding: 0.6rem 1.25rem;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text-muted);
    border: 1px solid var(--border);
    font-weight: 500;
    font-size: 0.9375rem;
    cursor: pointer;
    transition: border-color 0.15s, color 0.15s;
  }

  .btn-cta:hover {
    border-color: var(--text-muted);
    color: var(--text);
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

  .footer-copy {
    font-size: 0.875rem;
    color: var(--text-muted);
    margin: 0;
  }

  .github-link-footer {
    margin-top: 1rem;
  }
</style>
