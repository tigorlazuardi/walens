<script lang="ts">
  import { useJobs } from '../lib/api/queries';
  import type { JobStatus, JobType } from '../lib/api/types';
  import { Badge, Button, Card, Select } from '../lib/components/ui';

  const jobsQuery = useJobs(() => ({}));

  let filterStatus = $state<JobStatus | ''>('');
  let filterType = $state<JobType | ''>('');

  function statusVariant(status: JobStatus) {
    switch (status) {
      case 'queued': return 'secondary';
      case 'running': return 'warning';
      case 'succeeded': return 'success';
      case 'failed': return 'destructive';
      default: return 'outline';
    }
  }

  function formatDate(dateStr: string): string {
    try { return new Date(dateStr).toLocaleString(); } catch { return dateStr; }
  }

  function formatDuration(ms: number): string {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  }

  let filteredJobs = $derived((jobsQuery.data?.items ?? []).filter((job) => {
    if (filterStatus && job.status !== filterStatus) return false;
    if (filterType && job.job_type !== filterType) return false;
    return true;
  }));
</script>

<div class="space-y-4">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <h1 class="text-2xl font-semibold tracking-tight">Jobs</h1>
    <Button variant="outline" onclick={() => jobsQuery.refetch()}>Refresh</Button>
  </div>

  <div class="flex flex-wrap gap-3">
    <Select class="w-44" bind:value={filterStatus}>
      <option value="">All Statuses</option>
      <option value="queued">Queued</option>
      <option value="running">Running</option>
      <option value="succeeded">Succeeded</option>
      <option value="failed">Failed</option>
      <option value="cancelled">Cancelled</option>
    </Select>
    <Select class="w-44" bind:value={filterType}>
      <option value="">All Types</option>
      <option value="source_sync">Source Sync</option>
      <option value="source_download">Source Download</option>
    </Select>
  </div>

  {#if jobsQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if jobsQuery.isError}
    <p class="text-rose-600">Failed to load jobs</p>
  {:else if jobsQuery.data}
    <div class="grid gap-4">
      {#each filteredJobs as job}
        <Card class={job.status === 'failed' ? 'border-rose-300 p-4' : 'p-4'}>
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="font-semibold capitalize">{job.job_type.replace('_', ' ')}</p>
              {#if job.source_name}<p class="text-sm text-slate-500">Source: {job.source_name}</p>{/if}
            </div>
            <Badge variant={statusVariant(job.status)}>{job.status}</Badge>
          </div>

          <div class="mt-4 grid gap-2 text-sm text-slate-600 sm:grid-cols-2 lg:grid-cols-4">
            <p>Trigger: {job.trigger_kind}</p>
            <p>Created: {formatDate(job.created_at)}</p>
            {#if job.started_at}<p>Started: {formatDate(job.started_at)}</p>{/if}
            {#if job.finished_at}<p>Finished: {formatDate(job.finished_at)} ({formatDuration(job.duration_ms || 0)})</p>{/if}
          </div>

          <div class="mt-4 grid grid-cols-4 gap-3 text-center text-sm">
            <div><div class="text-lg font-semibold">{job.requested_image_count}</div><div class="text-xs text-slate-500">Requested</div></div>
            <div><div class="text-lg font-semibold">{job.downloaded_image_count}</div><div class="text-xs text-slate-500">Downloaded</div></div>
            <div><div class="text-lg font-semibold">{job.stored_image_count}</div><div class="text-xs text-slate-500">Stored</div></div>
            <div><div class="text-lg font-semibold">{job.skipped_image_count}</div><div class="text-xs text-slate-500">Skipped</div></div>
          </div>

          {#if job.error_message}
            <p class="mt-4 rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">{job.error_message}</p>
          {/if}
        </Card>
      {:else}
        <p class="py-10 text-center text-sm italic text-slate-500">No jobs found.</p>
      {/each}
    </div>
  {/if}
</div>
