<script>
  import { theme } from '../lib/theme.js'
  import { setup as apiSetup, login } from '../lib/api.js'
  import { user, setupRequired } from '../lib/auth.js'

  let email = ''
  let password = ''
  let confirmPassword = ''
  let error = ''
  let submitting = false

  async function handleSubmit(e) {
    e.preventDefault()
    error = ''
    if (!email.trim() || !password || !confirmPassword) {
      error = 'All fields are required.'
      return
    }
    if (password !== confirmPassword) {
      error = 'Passwords do not match.'
      return
    }
    if (password.length < 6) {
      error = 'Password must be at least 6 characters.'
      return
    }
    submitting = true
    const trimmedEmail = email.trim()
    try {
      // Step 1: create initial admin (POST /api/setup)
      await apiSetup(trimmedEmail, password)
      console.debug('[setup] account created, logging in…')
    } catch (err) {
      console.error('[setup] create account failed:', err?.message ?? err, err)
      error = err?.message ? `Could not create account: ${err.message}` : 'Could not create account.'
      submitting = false
      return
    }
    try {
      // Step 2: log in to get session (POST /api/auth/login)
      const u = await login(trimmedEmail, password)
      if (u) {
        user.set(u)
        setupRequired.set(false)
        console.debug('[setup] logged in successfully')
      } else {
        console.warn('[setup] login returned no user; check Network tab for POST /api/auth/login response')
        error = 'Account created. Please sign in with your email and password.'
      }
    } catch (err) {
      console.error('[setup] login after setup failed:', err?.message ?? err, err)
      error = err?.message ? `Account created but sign-in failed: ${err.message}` : 'Account created. Please sign in with your email and password.'
    } finally {
      submitting = false
    }
  }
</script>

<div class="setup-page">
  <div class="setup-card">
    <img src={$theme === 'light' ? '/images/logo-light.svg' : '/images/logo.svg'} alt="IPAM" class="setup-logo" />
    <h1 class="setup-title">Setup</h1>
    <p class="setup-subtitle">Create the initial admin account to get started.</p>
    <form class="setup-form" on:submit={handleSubmit}>
      {#if error}
        <div class="setup-error" role="alert">{error}</div>
      {/if}
      <label class="setup-label">
        <span>Admin email</span>
        <input type="email" bind:value={email} placeholder="admin@example.com" autocomplete="email" disabled={submitting} />
      </label>
      <label class="setup-label">
        <span>Password</span>
        <input type="password" bind:value={password} placeholder="At least 6 characters" autocomplete="new-password" disabled={submitting} />
      </label>
      <label class="setup-label">
        <span>Confirm password</span>
        <input type="password" bind:value={confirmPassword} placeholder="Confirm password" autocomplete="new-password" disabled={submitting} />
      </label>
      <button type="submit" class="btn btn-primary setup-submit" disabled={submitting}>
        {submitting ? 'Creating account…' : 'Create admin account'}
      </button>
    </form>
  </div>
</div>

<style>
  .setup-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1.5rem;
    background: var(--bg);
    color: var(--text);
  }
  .setup-card {
    width: 100%;
    max-width: 22rem;
    padding: 2rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
  }
  .setup-logo {
    display: block;
    width: 100%;
    height: auto;
    margin: 0 auto 1rem;
    object-fit: contain;
  }
  .setup-title {
    margin: 0 0 0.25rem 0;
    font-size: 1.5rem;
    font-weight: 600;
  }
  .setup-subtitle {
    margin: 0 0 1.5rem 0;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .setup-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .setup-error {
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    color: var(--danger);
    background: rgba(239, 68, 68, 0.1);
    border-radius: var(--radius);
  }
  .setup-label {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .setup-label input {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 1rem;
  }
  .setup-label input:focus {
    outline: none;
    border-color: var(--accent);
  }
  .setup-submit {
    margin-top: 0.5rem;
  }
</style>
