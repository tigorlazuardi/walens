<script lang="ts">
  import { getRuntimeConfig } from '../lib/runtime';
  import { logout, setRedirectAfterLogin } from '../lib/api/client';
  import { useRuntimeStatus } from '../lib/api/queries';
  import type { Snippet } from 'svelte';

  interface Props {
    navigate: (path: string) => void;
    currentPath: string;
    children: Snippet;
  }

  let { navigate, currentPath, children }: Props = $props();

  const config = getRuntimeConfig();
  const statusQuery = useRuntimeStatus();

  // Auth is enabled if runtime status says so
  let authEnabled = $derived(statusQuery.data?.auth_enabled ?? false);
  // If not logged in AND auth is enabled, show login
  // But allow /login route through
  let isLoginRoute = $derived(currentPath.replace(config.basePath, '') === '/login');
  let showLogin = $derived(authEnabled && !isLoginRoute);

  function handleNav(path: string) {
    navigate(path);
  }

  async function handleLogout() {
    await logout();
    // After logout, navigate to home
    navigate('/');
  }

  function isActive(path: string): boolean {
    const stripped = currentPath.replace(config.basePath, '') || '/';
    return stripped === path;
  }

  function handleLoginNav(e: Event) {
    e.preventDefault();
    const target = currentPath !== '/' ? currentPath : '/';
    setRedirectAfterLogin(target);
    navigate('/login');
  }
</script>

<div class="app-shell">
  {#if showLogin}
    <!-- Minimal placeholder - Login.svelte is shown via /login route -->
    <!-- This shows only when auth is enabled and user needs to log in -->
    <div class="login-redirect">
      <p>Authentication required.</p>
      <button onclick={handleLoginNav}>Go to Login</button>
    </div>
  {:else}
    <!-- Main app shell -->
    <header class="topnav">
      <div class="nav-brand" role="button" tabindex="0" onclick={() => handleNav('/')}>Walens</div>
      <nav class="nav-links">
        <button class="nav-btn" class:active={isActive('/')} onclick={() => handleNav('/')}>Home</button>
        <button class="nav-btn" class:active={isActive('/devices')} onclick={() => handleNav('/devices')}>Devices</button>
        <button class="nav-btn" class:active={isActive('/sources')} onclick={() => handleNav('/sources')}>Sources</button>
        <button class="nav-btn" class:active={isActive('/schedules')} onclick={() => handleNav('/schedules')}>Schedules</button>
        <button class="nav-btn" class:active={isActive('/subscriptions')} onclick={() => handleNav('/subscriptions')}>Subscriptions</button>
        <button class="nav-btn" class:active={isActive('/images')} onclick={() => handleNav('/images')}>Images</button>
        <button class="nav-btn" class:active={isActive('/jobs')} onclick={() => handleNav('/jobs')}>Jobs</button>
        <button class="nav-btn" class:active={isActive('/status')} onclick={() => handleNav('/status')}>Status</button>
        <button class="nav-btn" class:active={isActive('/settings')} onclick={() => handleNav('/settings')}>Settings</button>
      </nav>
      <div class="nav-right">
        {#if authEnabled}
          <button class="logout-btn" onclick={handleLogout}>Logout</button>
        {/if}
      </div>
    </header>

    <main class="main-content">
      {@render children()}
    </main>
  {/if}
</div>

<style>
  :global(body) {
    margin: 0;
    font-family: system-ui, -apple-system, sans-serif;
    background: #f5f5f5;
  }

  .app-shell {
    min-height: 100vh;
  }

  .login-redirect {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    gap: 1rem;
    color: #666;
  }

  .login-redirect button {
    padding: 0.5rem 1rem;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  /* Top nav */
  .topnav {
    background: white;
    border-bottom: 1px solid #e0e0e0;
    padding: 0 1rem;
    display: flex;
    align-items: center;
    gap: 1rem;
    height: 48px;
  }

  .nav-brand {
    font-weight: 600;
    font-size: 1.1rem;
    color: #333;
    padding-right: 1rem;
    border-right: 1px solid #e0e0e0;
    cursor: pointer;
  }

  .nav-links {
    display: flex;
    gap: 0.25rem;
    flex: 1;
  }

  .nav-btn {
    background: none;
    border: none;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    border-radius: 4px;
    font-size: 0.9rem;
    color: #555;
  }

  .nav-btn:hover {
    background: #f0f0f0;
  }

  .nav-btn.active {
    background: #e3f2fd;
    color: #1976d2;
  }

  .nav-right {
    margin-left: auto;
  }

  .logout-btn {
    background: none;
    border: 1px solid #ddd;
    padding: 0.375rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85rem;
    color: #666;
  }

  .logout-btn:hover {
    background: #f5f5f5;
  }

  .main-content {
    padding: 1rem;
    max-width: 1200px;
    margin: 0 auto;
  }

  /* Mobile responsive */
  @media (max-width: 768px) {
    .topnav {
      flex-wrap: wrap;
      height: auto;
      padding: 0.5rem;
    }

    .nav-links {
      order: 3;
      width: 100%;
      overflow-x: auto;
      padding-top: 0.5rem;
    }

    .nav-btn {
      white-space: nowrap;
      font-size: 0.85rem;
      padding: 0.375rem 0.5rem;
    }

    .main-content {
      padding: 0.5rem;
    }
  }
</style>
