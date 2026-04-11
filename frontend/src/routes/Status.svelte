<script lang="ts">
  import { useRuntimeStatus } from '../lib/api/queries';
  import { Badge, Card, Separator } from '../lib/components/ui';

  const statusQuery = useRuntimeStatus();

  function formatTime(date: Date): string {
    return date.toLocaleTimeString();
  }
</script>

<div class="space-y-6">
  <div class="flex flex-wrap items-center justify-between gap-2">
    <h1 class="text-2xl font-semibold tracking-tight">Runtime Status</h1>
    <span class="text-sm text-slate-500">Last updated: {statusQuery.data ? formatTime(new Date()) : '-'}</span>
  </div>

  {#if statusQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if statusQuery.isError}
    <p class="text-rose-600">Failed to load status</p>
  {:else if statusQuery.data}
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <Card class="p-4">
        <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Overall Status</p>
        <div class="mt-2">
          <Badge variant={statusQuery.data.status === 'ok' ? 'success' : statusQuery.data.status === 'degraded' ? 'warning' : 'destructive'}>
            {statusQuery.data.status}
          </Badge>
        </div>
      </Card>

      <Card class="p-4">
        <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Queue</p>
        <p class="mt-2 text-2xl font-semibold">{statusQuery.data.queue_size}</p>
        <p class="text-sm text-slate-500">jobs pending</p>
      </Card>

      <Card class="p-4">
        <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Active Schedules</p>
        <p class="mt-2 text-2xl font-semibold">{statusQuery.data.schedule_count}</p>
        <p class="text-sm text-slate-500">configured</p>
      </Card>

      <Card class="p-4">
        <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Scheduler</p>
        <p class="mt-2 text-2xl font-semibold">{statusQuery.data.scheduler_ready ? 'Ready' : 'Initializing'}</p>
        <p class="text-sm text-slate-500">{statusQuery.data.scheduler_ready ? 'Ready to enqueue jobs' : 'Booting'}</p>
      </Card>

      <Card class="p-4 sm:col-span-2 xl:col-span-4">
        <div class="flex items-center justify-between gap-2">
          <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Runner</p>
          <Badge variant={statusQuery.data.runner_active ? 'success' : 'secondary'}>{statusQuery.data.runner_active ? 'Active' : 'Idle'}</Badge>
        </div>
        <Separator class="my-3" />
        <p class="text-sm text-slate-500">Background job runner state is managed in-process.</p>
      </Card>
    </div>

    <Card class="p-4">
      <h2 class="text-lg font-semibold">About Walens</h2>
      <p class="mt-2 text-sm text-slate-500">Walens is a self-hosted wallpaper collection manager. It fetches images from configured sources on a schedule and delivers them to your devices.</p>
    </Card>
  {/if}
</div>
