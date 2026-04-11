<script lang="ts">
  import { getRuntimeConfig } from '../lib/runtime';
  import { stripBasePath } from '../lib/path';
  import { logout, setRedirectAfterLogin } from '../lib/api/client';
  import { useRuntimeStatus } from '../lib/api/queries';
  import { Button, Card } from '../lib/components/ui';
  import type { Snippet } from 'svelte';

  interface Props {
    navigate: (path: string) => void;
    currentPath: string;
    children: Snippet;
  }

  let { navigate, currentPath, children }: Props = $props();

  const config = getRuntimeConfig();
  const statusQuery = useRuntimeStatus();

  let authEnabled = $derived(statusQuery.data?.auth_enabled ?? false);
  let isLoginRoute = $derived(stripBasePath(currentPath, config.basePath) === '/login');
  let showLogin = $derived(authEnabled && !isLoginRoute);

  function handleNav(path: string) {
    navigate(path);
  }

  async function handleLogout() {
    await logout();
    navigate('/');
  }

  function isActive(path: string): boolean {
    const stripped = stripBasePath(currentPath, config.basePath) || '/';
    return stripped === path;
  }

  function handleLoginNav(e: Event) {
    e.preventDefault();
    const target = stripBasePath(currentPath, config.basePath) || '/';
    setRedirectAfterLogin(target);
    navigate('/login');
  }
</script>

<div class="min-h-screen bg-slate-50 text-slate-950">
  {#if showLogin}
    <div class="flex min-h-screen items-center justify-center p-4">
      <Card class="w-full max-w-sm p-6 text-center">
        <p class="text-sm text-slate-500">Authentication required.</p>
        <div class="mt-4">
          <Button onclick={handleLoginNav}>Go to Login</Button>
        </div>
      </Card>
    </div>
  {:else}
    <header class="sticky top-0 z-20 border-b border-slate-200 bg-white/95 backdrop-blur">
      <div class="mx-auto flex max-w-7xl flex-wrap items-center gap-3 px-4 py-3">
        <button class="text-lg font-semibold tracking-tight" onclick={() => handleNav('/')}>Walens</button>
        <nav class="flex flex-1 gap-2 overflow-x-auto">
          {#each [
            ['/', 'Home'],
            ['/devices', 'Devices'],
            ['/sources', 'Sources'],
            ['/schedules', 'Schedules'],
            ['/subscriptions', 'Subscriptions'],
            ['/images', 'Images'],
            ['/jobs', 'Jobs'],
            ['/status', 'Status'],
            ['/settings', 'Settings'],
          ] as [path, label]}
            <Button variant={isActive(path) ? 'default' : 'ghost'} size="sm" class="whitespace-nowrap" onclick={() => handleNav(path)}>
              {label}
            </Button>
          {/each}
        </nav>
        {#if authEnabled}
          <Button variant="outline" size="sm" onclick={handleLogout}>Logout</Button>
        {/if}
      </div>
    </header>

    <main class="mx-auto max-w-7xl p-4 sm:p-6">
      {@render children()}
    </main>
  {/if}
</div>
