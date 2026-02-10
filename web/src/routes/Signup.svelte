<script>
  import { onMount } from 'svelte'
  import { theme } from '../lib/theme.js'
  import { validateSignupInvite, registerWithInvite } from '../lib/api.js'
  import { user } from '../lib/auth.js'

  export let token = ''

  let email = ''
  let password = ''
  let confirmPassword = ''
  let error = ''
  let submitting = false
  let validating = true
  let valid = false
  let expiresAt = ''

  onMount(async () => {
    if (!token || !token.trim()) {
      error = 'Invalid signup link. No token provided.'
      validating = false
      return
    }
    try {
      const res = await validateSignupInvite(token.trim())
      valid = res?.valid === true
      if (valid && res.expires_at) expiresAt = res.expires_at
      if (!valid) error = 'This signup link is invalid or has expired.'
    } catch (err) {
      error = err?.message || 'This signup link is invalid or has expired.'
      valid = false
    } finally {
      validating = false
    }
  })

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
    if (password.length < 8) {
      error = 'Password must be at least 8 characters.'
      return
    }
    submitting = true
    const trimmedEmail = email.trim()
    try {
      const data = await registerWithInvite(token.trim(), trimmedEmail, password)
      const u = data?.user
      if (u) {
        user.set(u)
        // Clear one-time invite token from URL and land in the app.
        window.location.hash = 'dashboard'
      } else {
        error = 'Account was created. Please sign in with your email and password.'
      }
    } catch (err) {
      error = err?.message || 'Could not create account.'
    } finally {
      submitting = false
    }
  }
</script>

<div class="signup-page">
  <div class="signup-card">
    <img src={$theme === 'light' ? '/images/logo-light.svg' : '/images/logo.svg'} alt="IPAM" class="signup-logo" />
    <h1 class="signup-title">Create your account</h1>
    <p class="signup-subtitle">You’ve been invited to join. Enter your details below.</p>

    {#if validating}
      <p class="signup-muted">Checking invite link…</p>
    {:else if !valid}
      <div class="signup-error" role="alert">{error}</div>
      <a href="#login" class="signup-link">Sign in</a>
    {:else}
      <form class="signup-form" on:submit={handleSubmit}>
        {#if error}
          <div class="signup-error" role="alert">{error}</div>
        {/if}
        <label class="signup-label">
          <span>Email</span>
          <input type="email" bind:value={email} placeholder="you@example.com" autocomplete="email" disabled={submitting} />
        </label>
        <label class="signup-label">
          <span>Password</span>
          <input type="password" bind:value={password} placeholder="At least 8 characters" autocomplete="new-password" disabled={submitting} />
        </label>
        <label class="signup-label">
          <span>Confirm password</span>
          <input type="password" bind:value={confirmPassword} placeholder="Confirm password" autocomplete="new-password" disabled={submitting} />
        </label>
        <button type="submit" class="btn btn-primary signup-submit" disabled={submitting}>
          {submitting ? 'Creating account…' : 'Create account'}
        </button>
      </form>
    {/if}
  </div>
</div>

<style>
  .signup-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1.5rem;
    background: var(--bg);
    color: var(--text);
  }
  .signup-card {
    width: 100%;
    max-width: 22rem;
    padding: 2rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
  }
  .signup-logo {
    display: block;
    width: 100%;
    height: auto;
    margin: 0 auto 1rem;
    object-fit: contain;
  }
  .signup-title {
    margin: 0 0 0.25rem 0;
    font-size: 1.5rem;
    font-weight: 600;
  }
  .signup-subtitle {
    margin: 0 0 1.5rem 0;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .signup-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .signup-error {
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    color: var(--danger);
    background: rgba(239, 68, 68, 0.1);
    border-radius: var(--radius);
  }
  .signup-muted {
    margin: 0;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .signup-link {
    margin-top: 1rem;
    font-size: 0.9rem;
    color: var(--accent);
  }
  .signup-label {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .signup-label input {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 1rem;
  }
  .signup-label input:focus {
    outline: none;
    border-color: var(--accent);
  }
  .signup-submit {
    margin-top: 0.5rem;
  }
</style>
