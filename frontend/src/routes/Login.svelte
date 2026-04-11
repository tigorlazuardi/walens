<script lang="ts">
  import { getRuntimeConfig } from '../lib/runtime';
  import { joinBasePath } from '../lib/path';
  import { login as apiLogin, popRedirectAfterLogin } from '../lib/api/client';
  import { QueryClient } from '../lib/query-client';
  import { Button, Card, Input } from '../lib/components/ui';

  const config = getRuntimeConfig();

  const urlParams = new URLSearchParams(window.location.search);
  const redirectTarget = urlParams.get('redirect') || popRedirectAfterLogin() || '/';

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
      QueryClient.invalidateQueries({ queryKey: ['runtimeStatus'] });
      window.location.href = joinBasePath(config.basePath, redirectTarget);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="flex min-h-screen items-center justify-center p-4">
  <Card class="w-full max-w-sm p-6">
    <div class="mb-6 text-center">
      <h1 class="text-2xl font-semibold tracking-tight">Walens</h1>
      <p class="text-sm text-slate-500">Wallpaper Manager</p>
    </div>

    <form class="space-y-4" onsubmit={handleLogin}>
      {#if error}
        <div class="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700">{error}</div>
      {/if}

      <div class="space-y-2">
        <label for="username" class="text-sm font-medium text-slate-700">Username</label>
        <Input id="username" bind:value={username} autocomplete="username" required disabled={loading} />
      </div>

      <div class="space-y-2">
        <label for="password" class="text-sm font-medium text-slate-700">Password</label>
        <Input id="password" type="password" bind:value={password} autocomplete="current-password" required disabled={loading} />
      </div>

      <Button class="w-full" type="submit" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign In'}
      </Button>
    </form>
  </Card>
</div>
