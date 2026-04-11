<script lang="ts">
  import Layout from './components/Layout.svelte';
  import { getRuntimeConfig } from './lib/runtime';
  import { normalizeAppPath, stripBasePath } from './lib/path';

  const config = getRuntimeConfig();

  const routeModules = (import.meta as ImportMeta & {
    glob: (pattern: string) => Record<string, () => Promise<{ default: any }>>;
  }).glob('./routes/*.svelte');
  const routeFiles: Record<string, string> = {
    '/': './routes/Home.svelte',
    '/login': './routes/Login.svelte',
    '/devices': './routes/Devices.svelte',
    '/sources': './routes/Sources.svelte',
    '/schedules': './routes/Schedules.svelte',
    '/subscriptions': './routes/Subscriptions.svelte',
    '/images': './routes/Images.svelte',
    '/jobs': './routes/Jobs.svelte',
    '/status': './routes/Status.svelte',
    '/settings': './routes/Settings.svelte',
  };

  let currentPath = $state(stripBasePath(window.location.pathname, config.basePath));

  let RouteComponent: any = $state(null);
  let loading = $state(true);

  function navigate(path: string) {
    const normalized = normalizeAppPath(path);
    const fullPath = config.basePath === '/' ? normalized : `${config.basePath}${normalized}`;
    window.history.pushState({}, '', fullPath);
    currentPath = normalized;
    loadRoute(normalized);
  }

  async function loadRoute(path: string) {
    const loader = routeModules[routeFiles[normalizeAppPath(path)]] as undefined | (() => Promise<{ default: any }>);
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

  $effect(() => {
    const handlePop = () => {
      currentPath = stripBasePath(window.location.pathname, config.basePath);
      loadRoute(currentPath);
    };
    window.addEventListener('popstate', handlePop);
    return () => window.removeEventListener('popstate', handlePop);
  });

  $effect(() => {
    loadRoute(currentPath);
  });
</script>

<Layout {navigate} {currentPath}>
  {#snippet children()}
    {#if loading}
      <div class="p-8 text-center text-slate-500">Loading...</div>
    {:else if RouteComponent}
      <RouteComponent />
    {:else}
      <div class="flex flex-col items-center justify-center gap-2 p-8 text-center text-slate-500">
        <h2 class="text-4xl font-semibold text-slate-300">404</h2>
        <p>Page not found</p>
      </div>
    {/if}
  {/snippet}
</Layout>
