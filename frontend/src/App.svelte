<script lang="ts">
  import Layout from './components/Layout.svelte';
  import { getRuntimeConfig } from './lib/runtime';
  import type { Snippet } from 'svelte';

  const config = getRuntimeConfig();

  // Route definitions - using lazy imports for code splitting
  const routeLoaders: Record<string, () => Promise<{ default: any }>> = {
    '/': () => import('./routes/Home.svelte'),
    '/login': () => import('./routes/Login.svelte'),
    '/devices': () => import('./routes/Devices.svelte'),
    '/sources': () => import('./routes/Sources.svelte'),
    '/schedules': () => import('./routes/Schedules.svelte'),
    '/subscriptions': () => import('./routes/Subscriptions.svelte'),
    '/images': () => import('./routes/Images.svelte'),
    '/jobs': () => import('./routes/Jobs.svelte'),
    '/status': () => import('./routes/Status.svelte'),
    '/settings': () => import('./routes/Settings.svelte'),
  };

  // Current path state
  let currentPath = $state(window.location.pathname.replace(config.basePath, '') || '/');

  // Route component
  let RouteComponent: any = $state(null);
  let loading = $state(true);

  // Navigate function
  function navigate(path: string) {
    const fullPath = config.basePath + path;
    window.history.pushState({}, '', fullPath);
    currentPath = path;
    loadRoute(path);
  }

  // Load route
  async function loadRoute(path: string) {
    const loader = routeLoaders[path];
    if (loader) {
      loading = true;
      try {
        const module = await loader();
        RouteComponent = module.default;
      } catch (e) {
        console.error('Failed to load route:', e);
        RouteComponent = null;
      }
      loading = false;
    } else {
      RouteComponent = null;
      loading = false;
    }
  }

  // Handle popstate (back/forward buttons)
  $effect(() => {
    const handlePop = () => {
      currentPath = window.location.pathname.replace(config.basePath, '') || '/';
      loadRoute(currentPath);
    };
    window.addEventListener('popstate', handlePop);
    return () => window.removeEventListener('popstate', handlePop);
  });

  // Initial route load
  $effect(() => {
    loadRoute(currentPath);
  });

  // Content snippet that renders the current route
  let content: Snippet;
</script>

<Layout {navigate} {currentPath}>
  {#snippet children()}
    {#if loading}
      <div class="loading">Loading...</div>
    {:else if RouteComponent}
      <RouteComponent />
    {:else}
      <div class="not-found">
        <h2>404</h2>
        <p>Page not found</p>
      </div>
    {/if}
  {/snippet}
</Layout>

<style>
  .loading, .not-found {
    padding: 2rem;
    text-align: center;
    color: #666;
  }

  .not-found h2 {
    margin: 0;
    font-size: 3rem;
    color: #ccc;
  }

  .not-found p {
    margin: 0.5rem 0 0;
  }
</style>
