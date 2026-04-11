<script lang="ts">
  import { useConfig, useUpdateConfig } from '../lib/api/queries';
  import type { UpdateConfigRequest } from '../lib/api/types';
  import { Button, Card, Input, Select } from '../lib/components/ui';

  const configQuery = useConfig();
  const updateMutation = useUpdateConfig();

  let dataDir = $state('');
  let logLevel = $state('');
  let saved = $state(false);

  $effect(() => {
    if (configQuery.data) {
      dataDir = configQuery.data.data_dir ?? '';
      logLevel = configQuery.data.log_level ?? '';
    }
  });

  async function handleSave(e: Event) {
    e.preventDefault();
    const body: UpdateConfigRequest = {
      data_dir: dataDir,
      log_level: logLevel,
    };
    await updateMutation.mutateAsync(body);
    saved = true;
    setTimeout(() => {
      saved = false;
    }, 3000);
  }
</script>

<div class="max-w-2xl space-y-6">
  <h1 class="text-2xl font-semibold tracking-tight">Settings</h1>

  {#if configQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if configQuery.isError}
    <p class="text-rose-600">Failed to load settings</p>
  {:else if configQuery.data}
    <Card class="p-6">
      <h2 class="text-lg font-semibold">Storage & Logging</h2>
      <p class="mt-1 text-sm text-slate-500">Configure application storage and logging.</p>

      <form class="mt-6 space-y-4" onsubmit={handleSave}>
        <div class="space-y-2">
          <label for="dataDir" class="text-sm font-medium text-slate-700">Data Directory</label>
          <Input id="dataDir" bind:value={dataDir} placeholder="./data" required />
          <span class="text-xs text-slate-500">Directory where images and thumbnails are stored.</span>
        </div>

        <div class="space-y-2">
          <label for="logLevel" class="text-sm font-medium text-slate-700">Log Level</label>
          <Select id="logLevel" bind:value={logLevel}>
            <option value="debug">Debug</option>
            <option value="info">Info</option>
            <option value="warn">Warn</option>
            <option value="error">Error</option>
          </Select>
          <span class="text-xs text-slate-500">Controls verbosity of application logging.</span>
        </div>

        <div class="flex items-center gap-3 pt-2">
          <Button type="submit" disabled={updateMutation.isPending}>
            {updateMutation.isPending ? 'Saving...' : 'Save Settings'}
          </Button>
          {#if saved}
            <span class="text-sm text-emerald-600">Settings saved!</span>
          {/if}
        </div>
      </form>
    </Card>
  {/if}
</div>
