<script lang="ts">
  import { useRuntimeStatus } from '../lib/api/queries';
  import { Badge, Card } from '../lib/components/ui';

  const statusQuery = useRuntimeStatus();
</script>

<div class="space-y-6">
  <div class="space-y-1">
    <h1 class="text-3xl font-semibold tracking-tight">Walens</h1>
    <p class="text-sm text-slate-500">Wallpaper Manager</p>
  </div>

  {#if statusQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if statusQuery.isError}
    <p class="text-rose-600">Failed to load status</p>
  {:else if statusQuery.data}
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <Card class="p-4">
        <p class="text-xs font-medium uppercase tracking-wide text-slate-500">System</p>
        <div class="mt-2">
          <Badge variant={statusQuery.data.status === 'ok' ? 'success' : statusQuery.data.status === 'degraded' ? 'warning' : 'destructive'}>
            {statusQuery.data.status}
          </Badge>
        </div>
      </Card>
      <Card class="p-4">
        <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Queue</p>
        <p class="mt-2 text-2xl font-semibold">{statusQuery.data.queue_size}</p>
        <p class="text-sm text-slate-500">jobs</p>
      </Card>
      <Card class="p-4">
        <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Schedules</p>
        <p class="mt-2 text-2xl font-semibold">{statusQuery.data.schedule_count}</p>
        <p class="text-sm text-slate-500">active</p>
      </Card>
      <Card class="p-4">
        <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Runner</p>
        <p class="mt-2 text-2xl font-semibold">{statusQuery.data.runner_active ? 'Active' : 'Idle'}</p>
        <p class="text-sm text-slate-500">{statusQuery.data.scheduler_ready ? 'Ready' : 'Initializing'}</p>
      </Card>
    </div>
  {/if}
</div>
