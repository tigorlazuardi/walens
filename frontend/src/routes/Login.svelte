<script lang="ts">
  import { getRuntimeConfig, buildApiUrl } from '../lib/runtime';
  import { login as apiLogin, popRedirectAfterLogin } from '../lib/api/client';
  import { QueryClient } from '../lib/query-client';

  const config = getRuntimeConfig();

  // Check for redirect target from query param
  const urlParams = new URLSearchParams(window.location.search);
  const redirectTarget = urlParams.get('redirect') || '/';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleLogin(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;

    try {
      await apiLogin(username, password);
      // Clear stale auth cache so runtime status refetches correctly
      QueryClient.invalidateQueries({ queryKey: ['runtimeStatus'] });
      // Redirect to the originally requested path
      window.location.href = config.basePath + redirectTarget;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-page">
  <div class="login-card">
    <h1>Walens</h1>
    <p class="subtitle">Wallpaper Manager</p>

    <form onsubmit={handleLogin}>
      {#if error}
        <div class="error">{error}</div>
      {/if}

      <div class="field">
        <label for="username">Username</label>
        <input
          type="text"
          id="username"
          bind:value={username}
          autocomplete="username"
          required
          disabled={loading}
        />
      </div>

      <div class="field">
        <label for="password">Password</label>
        <input
          type="password"
          id="password"
          bind:value={password}
          autocomplete="current-password"
          required
          disabled={loading}
        />
      </div>

      <button type="submit" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign In'}
      </button>
    </form>
  </div>
</div>

<style>
  .login-page {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background: #f5f5f5;
    padding: 1rem;
  }

  .login-card {
    background: white;
    padding: 2rem;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    width: 100%;
    max-width: 320px;
  }

  h1 {
    margin: 0;
    font-size: 1.5rem;
    text-align: center;
    color: #333;
  }

  .subtitle {
    text-align: center;
    color: #666;
    margin: 0.5rem 0 1.5rem;
    font-size: 0.9rem;
  }

  .error {
    background: #fee;
    border: 1px solid #fcc;
    color: #c33;
    padding: 0.75rem;
    border-radius: 4px;
    margin-bottom: 1rem;
    font-size: 0.875rem;
  }

  .field {
    margin-bottom: 1rem;
  }

  label {
    display: block;
    font-size: 0.875rem;
    color: #555;
    margin-bottom: 0.25rem;
  }

  input {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box;
  }

  input:focus {
    outline: none;
    border-color: #007bff;
  }

  input:disabled {
    background: #f9f9f9;
  }

  button {
    width: 100%;
    padding: 0.75rem;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 1rem;
    cursor: pointer;
  }

  button:hover:not(:disabled) {
    background: #0056b3;
  }

  button:disabled {
    background: #ccc;
    cursor: not-allowed;
  }
</style>
