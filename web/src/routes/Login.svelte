<script>
  import { onMount } from 'svelte'
  import { theme } from '../lib/theme.js'
  import { getAuthConfig, login } from '../lib/api.js'
  import { user } from '../lib/auth.js'
  import ErrorModal from '../lib/ErrorModal.svelte'

  let email = ''
  let password = ''
  let error = ''
  let submitting = false
  let githubOAuthEnabled = false
  let configLoaded = false
  let hasOAuthRedirectError = false
  let showErrorModal = false

  onMount(() => {
    const hash = (window.location.hash || '#').slice(1) || ''
    const q = hash.indexOf('?')
    const params = new URLSearchParams(q >= 0 ? hash.slice(q) : window.location.search)
    const err = params.get('error')
    if (err) {
      error = decodeURIComponent(err)
      hasOAuthRedirectError = true
      showErrorModal = true
    }
    getAuthConfig()
      .then((c) => {
        githubOAuthEnabled = c?.githubOAuthEnabled === true
        configLoaded = true
      })
      .catch(() => {
        configLoaded = true
      })
  })

  function closeErrorModal() {
    showErrorModal = false
    const hash = (window.location.hash || '#').slice(1) || ''
    const q = hash.indexOf('?')
    if (q >= 0) {
      const params = new URLSearchParams(hash.slice(q))
      params.delete('error')
      const rest = params.toString()
      const baseHash = hash.slice(0, q)
      window.history.replaceState({}, '', window.location.pathname + '#' + baseHash + (rest ? '?' + rest : ''))
    }
  }

  function signInWithGitHub() {
    const base = window.location.origin + window.location.pathname.replace(/\/$/, '') || ''
    window.location.href = base + '/api/auth/oauth/github/start'
  }

  async function handleSubmit(e) {
    e.preventDefault()
    error = ''
    if (!email.trim() || !password) {
      error = 'Email and password are required.'
      return
    }
    submitting = true
    try {
      const u = await login(email.trim(), password)
      if (u) {
        user.set(u)
      } else {
        error = 'Invalid email or password.'
      }
    } catch (err) {
      error = err.message || 'Login failed.'
    } finally {
      submitting = false
    }
  }
</script>

{#if showErrorModal && error}
  <ErrorModal message={error} on:close={closeErrorModal} />
{/if}
<div class="login-page">
  <div class="login-card">
    <img src={$theme === 'light' ? '/images/logo-light.svg' : '/images/logo.svg'} alt="IPAM" class="login-logo" />
    <p class="login-subtitle">Sign in to continue</p>
    <form class="login-form" on:submit={handleSubmit}>
      {#if error}
        <div class="login-error" role="alert">{error}</div>
      {/if}
      {#if !configLoaded}
        <p class="login-muted">Loading…</p>
      {:else if githubOAuthEnabled || hasOAuthRedirectError}
        <button type="button" class="btn btn-primary login-github" on:click={signInWithGitHub} disabled={submitting}>
          Sign in with GitHub
        </button>
      {:else}
        <label class="login-label" for="login-email">
          <span>Email</span>
          <input
            id="login-email"
            name="email"
            type="email"
            bind:value={email}
            placeholder="admin@localhost"
            autocomplete="email"
            disabled={submitting}
          />
        </label>
        <label class="login-label" for="login-password">
          <span>Password</span>
          <input
            id="login-password"
            name="password"
            type="password"
            bind:value={password}
            placeholder="Password"
            autocomplete="current-password"
            disabled={submitting}
          />
        </label>
        <button type="submit" class="btn btn-primary login-submit" disabled={submitting}>
          {submitting ? 'Signing in…' : 'Sign in'}
        </button>
      {/if}
    </form>
  </div>
</div>

<style>
  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1.5rem;
    background: var(--bg);
    color: var(--text);
  }
  .login-card {
    width: 100%;
    max-width: 22rem;
    padding: 2rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
  }
  .login-logo {
    display: block;
    width: 100%;
    height: auto;
    max-height: 9rem;
    margin: 0 auto 1rem;
    object-fit: contain;
  }
  .login-subtitle {
    margin: 0 0 1.5rem 0;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .login-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .login-error {
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    color: var(--danger);
    background: rgba(239, 68, 68, 0.1);
    border-radius: var(--radius);
  }
  .login-label {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .login-label input {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 1rem;
  }
  .login-label input:focus {
    outline: none;
    border-color: var(--accent);
  }
  .login-submit {
    margin-top: 0.5rem;
  }
  .login-muted {
    margin: 0;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .login-github {
    width: 100%;
    border: 1px solid var(--border);
  }
  .login-divider {
    margin: 0;
    font-size: 0.85rem;
    color: var(--text-muted);
    text-align: center;
  }
</style>
