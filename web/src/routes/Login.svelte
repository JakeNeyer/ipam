<script>
  import { theme } from '../lib/theme.js'
  import { login } from '../lib/api.js'
  import { user } from '../lib/auth.js'

  let email = ''
  let password = ''
  let error = ''
  let submitting = false

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

<div class="login-page">
  <div class="login-card">
    <img src={$theme === 'light' ? '/images/logo-light.svg' : '/images/logo.svg'} alt="IPAM" class="login-logo" />
    <p class="login-subtitle">Sign in to continue</p>
    <form class="login-form" on:submit={handleSubmit}>
      {#if error}
        <div class="login-error" role="alert">{error}</div>
      {/if}
      <label class="login-label">
        <span>Email</span>
        <input type="email" bind:value={email} placeholder="admin@localhost" autocomplete="email" disabled={submitting} />
      </label>
      <label class="login-label">
        <span>Password</span>
        <input type="password" bind:value={password} placeholder="Password" autocomplete="current-password" disabled={submitting} />
      </label>
      <button type="submit" class="btn btn-primary login-submit" disabled={submitting}>
        {submitting ? 'Signing in…' : 'Sign in'}
      </button>
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
</style>
