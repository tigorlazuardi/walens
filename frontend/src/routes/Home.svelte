<script lang="ts">
  import { useRuntimeStatus } from '../lib/api/queries';

  const statusQuery = useRuntimeStatus();
</script>

<div class="home">
  <h1>Walens</h1>
  <p class="subtitle">Wallpaper Manager</p>

  {#if statusQuery.isLoading}
    <p>Loading...</p>
  {:else if statusQuery.isError}
    <p class="error">Failed to load status</p>
  {:else if statusQuery.data}
    <div class="status-cards">
      <div class="card">
        <h3>System</h3>
        <p class="status-value" class:ok={statusQuery.data.status === 'ok'} class:degraded={statusQuery.data.status === 'degraded'}>
          {statusQuery.data.status}
        </p>
      </div>
      <div class="card">
        <h3>Queue</h3>
        <p class="status-value">{statusQuery.data.queue_size} jobs</p>
      </div>
      <div class="card">
        <h3>Schedules</h3>
        <p class="status-value">{statusQuery.data.schedule_count} active</p>
      </div>
      <div class="card">
        <h3>Scheduler</h3>
        <p class="status-value">{statusQuery.data.scheduler_ready ? 'Ready' : 'Initializing'}</p>
      </div>
      <div class="card">
        <h3>Runner</h3>
        <p class="status-value">{statusQuery.data.runner_active ? 'Active' : 'Idle'}</p>
      </div>
    </div>
  {/if}
</div>

<style>
  .home {
    text-align: center;
    padding: 2rem 1rem;
  }

  h1 {
    margin: 0;
    font-size: 2rem;
    color: #333;
  }

  .subtitle {
    color: #666;
    margin: 0.5rem 0 2rem;
  }

  .status-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1rem;
    max-width: 800px;
    margin: 0 auto;
  }

  .card {
    background: white;
    border-radius: 8px;
    padding: 1rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .card h3 {
    margin: 0 0 0.5rem;
    font-size: 0.85rem;
    color: #666;
    text-transform: uppercase;
  }

  .status-value {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: #333;
  }

  .status-value.ok {
    color: #2e7d32;
  }

  .status-value.degraded {
    color: #f57c00;
  }

  .error {
    color: #c33;
  }
</style>
