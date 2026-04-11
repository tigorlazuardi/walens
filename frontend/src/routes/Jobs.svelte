<script lang="ts">
  import { useJobs } from '../lib/api/queries';
  import type { JobStatus, JobType, TriggerKind } from '../lib/api/types';

  const jobsQuery = useJobs(() => ({}));

  let filterStatus = $state<JobStatus | ''>('');
  let filterType = $state<JobType | ''>('');

  function statusClass(status: JobStatus): string {
    switch (status) {
      case 'queued': return 'badge blue';
      case 'running': return 'badge yellow';
      case 'succeeded': return 'badge green';
      case 'failed': return 'badge red';
      case 'cancelled': return 'badge gray';
      default: return 'badge gray';
    }
  }

  function formatDate(dateStr: string): string {
    try {
      return new Date(dateStr).toLocaleString();
    } catch {
      return dateStr;
    }
  }

  function formatDuration(ms: number): string {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Jobs</h1>
    <button class="refresh-btn" onclick={() => jobsQuery.refetch()}>Refresh</button>
  </div>

  <div class="filters">
    <select bind:value={filterStatus}>
      <option value="">All Statuses</option>
      <option value="queued">Queued</option>
      <option value="running">Running</option>
      <option value="succeeded">Succeeded</option>
      <option value="failed">Failed</option>
      <option value="cancelled">Cancelled</option>
    </select>
    <select bind:value={filterType}>
      <option value="">All Types</option>
      <option value="source_sync">Source Sync</option>
      <option value="source_download">Source Download</option>
    </select>
  </div>

  {#if jobsQuery.isLoading}
    <p>Loading...</p>
  {:else if jobsQuery.isError}
    <p class="error">Failed to load jobs</p>
  {:else if jobsQuery.data}
    <div class="job-list">
      {#each jobsQuery.data.items as job}
        <div class="job-card" class:failed={job.status === 'failed'}>
          <div class="job-header">
            <span class="job-type">{job.job_type.replace('_', ' ')}</span>
            <span class={statusClass(job.status)}>{job.status}</span>
          </div>
          <div class="job-info">
            {#if job.source_name}
              <p class="source-name">Source: {job.source_name}</p>
            {/if}
            <p class="trigger">Trigger: {job.trigger_kind}</p>
            <p class="time">Created: {formatDate(job.created_at)}</p>
            {#if job.started_at}
              <p class="time">Started: {formatDate(job.started_at)}</p>
            {/if}
            {#if job.finished_at}
              <p class="time">Finished: {formatDate(job.finished_at)} ({formatDuration(job.duration_ms || 0)})</p>
            {/if}
          </div>
          <div class="job-stats">
            <div class="stat">
              <span class="stat-value">{job.requested_image_count}</span>
              <span class="stat-label">Requested</span>
            </div>
            <div class="stat">
              <span class="stat-value">{job.downloaded_image_count}</span>
              <span class="stat-label">Downloaded</span>
            </div>
            <div class="stat">
              <span class="stat-value">{job.stored_image_count}</span>
              <span class="stat-label">Stored</span>
            </div>
            <div class="stat">
              <span class="stat-value">{job.skipped_image_count}</span>
              <span class="stat-label">Skipped</span>
            </div>
          </div>
          {#if job.error_message}
            <p class="error-message">{job.error_message}</p>
          {/if}
        </div>
      {:else}
        <p class="empty">No jobs found.</p>
      {/each}
    </div>
  {/if}
</div>

<style>
  .page {
    padding: 1rem;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  h1 {
    margin: 0;
    font-size: 1.5rem;
  }

  .filters {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .filters select {
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 0.9rem;
  }

  .refresh-btn {
    background: #f5f5f5;
    border: 1px solid #ddd;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .refresh-btn:hover {
    background: #e8e8e8;
  }

  .job-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .job-card {
    background: white;
    border-radius: 8px;
    padding: 1rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    border-left: 3px solid #ccc;
  }

  .job-card.failed {
    border-left-color: #c62828;
  }

  .job-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .job-type {
    font-weight: 600;
    text-transform: capitalize;
  }

  .badge {
    font-size: 0.7rem;
    padding: 0.125rem 0.5rem;
    border-radius: 3px;
    font-weight: 500;
  }

  .badge.blue {
    background: #e3f2fd;
    color: #1976d2;
  }

  .badge.yellow {
    background: #fff8e1;
    color: #f57c00;
  }

  .badge.green {
    background: #e8f5e9;
    color: #2e7d32;
  }

  .badge.red {
    background: #ffebee;
    color: #c62828;
  }

  .badge.gray {
    background: #f5f5f5;
    color: #666;
  }

  .job-info p {
    margin: 0.2rem 0;
    font-size: 0.85rem;
    color: #666;
  }

  .job-stats {
    display: flex;
    gap: 1rem;
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid #eee;
  }

  .stat {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .stat-value {
    font-size: 1.1rem;
    font-weight: 600;
    color: #333;
  }

  .stat-label {
    font-size: 0.7rem;
    color: #888;
    text-transform: uppercase;
  }

  .error-message {
    margin-top: 0.5rem;
    padding: 0.5rem;
    background: #ffebee;
    border-radius: 4px;
    color: #c62828;
    font-size: 0.85rem;
  }

  .empty {
    color: #888;
    font-style: italic;
    text-align: center;
    padding: 2rem;
  }

  .error {
    color: #c33;
  }
</style>
