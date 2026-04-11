<script>
  // Read runtime config injected by backend
  const config = window.__WALENS__ || {
    basePath: '/',
    apiBase: '/api',
  };

  // Export config for use in child components
  export { config };

  // Simple client-side router honoring basePath
  function getRoutePath() {
    const fullPath = window.location.pathname;
    // Strip basePath from the beginning if present
    if (config.basePath !== '/' && fullPath.startsWith(config.basePath)) {
      return fullPath.slice(config.basePath.length) || '/';
    }
    return fullPath;
  }

  let currentRoute = $state(getRoutePath());

  // Navigation function
  function navigate(path) {
    const fullPath = config.basePath === '/' ? path : config.basePath + path;
    window.history.pushState({}, '', fullPath);
    currentRoute = getRoutePath();
  }

  // Handle popstate (browser back/forward)
  if (typeof window !== 'undefined') {
    window.addEventListener('popstate', () => {
      currentRoute = getRoutePath();
    });
  }

  // Lazy-loaded route components
  const routes = {
    '/': () => import('./routes/Home.svelte'),
    '/sources': () => import('./routes/Sources.svelte'),
    '/devices': () => import('./routes/Devices.svelte'),
    '/images': () => import('./routes/Images.svelte'),
    '/settings': () => import('./routes/Settings.svelte'),
  };

  // Resolve route with dynamic import
  let RouteComponent = $state(null);
  let loading = $state(false);

  async function loadRoute(path) {
    loading = true;
    try {
      // Find matching route or fallback to /
      const handler = routes[path] || routes['/'];
      const module = await handler();
      RouteComponent = module.default;
    } catch (e) {
      console.error('Failed to load route:', e);
      RouteComponent = null;
    } finally {
      loading = false;
    }
  }

  // React to route changes
  $effect(() => {
    loadRoute(currentRoute);
  });

  // Export navigation for use in components
  export { navigate };
</script>

<div class="app">
  <nav>
    <button onclick={() => navigate('/')}>Home</button>
    <button onclick={() => navigate('/sources')}>Sources</button>
    <button onclick={() => navigate('/devices')}>Devices</button>
    <button onclick={() => navigate('/images')}>Images</button>
    <button onclick={() => navigate('/settings')}>Settings</button>
  </nav>

  <main>
    {#if loading}
      <p>Loading...</p>
    {:else if RouteComponent}
      <RouteComponent />
    {:else}
      <p>Route not found</p>
    {/if}
  </main>
</div>

<style>
  .app {
    font-family: sans-serif;
    padding: 1rem;
  }
  nav {
    margin-bottom: 1rem;
  }
  nav button {
    margin-right: 0.5rem;
  }
</style>
