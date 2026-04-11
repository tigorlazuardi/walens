<script lang="ts">
  import { useRuntimeStatus } from '../lib/api/queries';

  const statusQuery = useRuntimeStatus();

  function formatTime(date: Date): string {
    return date.toLocaleTimeString();
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Runtime Status</h1>
    <span class="updated">Last updated: {statusQuery.data ? formatTime(new Date()) : '-'}</span>
  </div>

  {#if statusQuery.isLoading}
    <p>Loading...</p>
  {:else if statusQuery.isError}
    <p class="error">Failed to load status</p>
  {:else if statusQuery.data}
    <div class="status-grid">
      <div class="status-card" class:ok={statusQuery.data.status === 'ok'} class:degraded={statusQuery.data.status === 'degraded'} class:stopping={statusQuery.data.status === 'stopping'}>
        <h3>Overall Status</h3>
        <p class="status-value">{statusQuery.data.status}</p>
      </div>

      <div class="status-card">
        <h3>Queue</h3>
        <p class="status-value">{statusQuery.data.queue_size}</p>
        <p class="status-note">jobs pending</p>
      </div>

      <div class="status-card">
        <h3>Active Schedules</h3>
        <p class="status-value">{statusQuery.data.schedule_count}</p>
        <p class="status-note">configured</p>
      </div>

      <div class="status-card" class:ready={statusQuery.data.scheduler_ready} class:initializing={!statusQuery.data.scheduler_ready}>
        <h3>Scheduler</h3>
        <p class="status-value">{statusQuery.data.scheduler_ready ? 'Ready' : 'Initializing'}</p>
      </div>

      <div class="status-card" class:active={statusQuery.data.runner_active} class:idle={!statusQuery.data.runner_active}>
        <h3>Runner</h3>
        <p class="status-value">{statusQuery.data.runner_active ? 'Active' : 'Idle'}</p>
      </div>
    </div>

    <div class="info-section">
      <h2>About Walens</h2>
      <p>Walens is a self-hosted wallpaper collection manager. It fetches images from configured sources on a schedule and delivers them to your devices.</p>
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
    margin-bottom: 1.5rem;
  }

  h1 {
    margin: 0;
    font-size: 1.5rem;
  }

  .updated {
    font-size: 0.85rem;
    color: #888;
  }

  .status-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
  }

  .status-card {
    background: white;
    border-radius: 8px;
    padding: 1.25rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    text-align: center;
  }

  .status-card h3 {
    margin: 0 0 0.5rem;
    font-size: 0.85rem;
    color: #666;
    text-transform: uppercase;
    font-weight: 500;
  }

  .status-value {
    margin: 0;
    font-size: 1.75rem;
    font-weight: 600;
    color: #333;
  }

  .status-note {
    margin: 0.25rem 0 0;
    font-size: 0.8rem;
    color: #888;
  }

  /* Status colors */
  .ok .status-value {
    color: #2e7d32;
  }

  .degraded .status-value {
    color: #f57c00;
  }

  .stopping .status-value {
    color: #c62828;
  }

  .ready .status-value {
    color: #2e7d32;
  }

  .initializing .status-value {
    color: #f57c00;
  }

  .active .status-value {
    color: #1976d2;
  }

  .idle .status-value {
    color: #888;
  }

  .info-section {
    background: white;
    border-radius: 8px;
    padding: 1.25rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .info-section h2 {
    margin: 0 0 0.5rem;
    font-size: 1.1rem;
  }

  .info-section p {
    margin: 0;
    color: #666;
    font-size: 0.9rem;
    line-height: 1.5;
  }

  .error {
    color: #c33;
  }
</style>
