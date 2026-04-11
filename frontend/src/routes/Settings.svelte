<script lang="ts">
  import { useConfig, useUpdateConfig } from '../lib/api/queries';
  import type { UpdateConfigRequest } from '../lib/api/types';

  const configQuery = useConfig();
  const updateMutation = useUpdateConfig();

  let dataDir = $state('');
  let logLevel = $state('');
  let saved = $state(false);

  // Load config when available
  $effect(() => {
    if (configQuery.data) {
      dataDir = configQuery.data.data_dir;
      logLevel = configQuery.data.log_level;
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
    setTimeout(() => { saved = false; }, 3000);
  }
</script>

<div class="page">
  <h1>Settings</h1>

  {#if configQuery.isLoading}
    <p>Loading...</p>
  {:else if configQuery.isError}
    <p class="error">Failed to load settings</p>
  {:else if configQuery.data}
    <div class="settings-card">
      <h2>Storage & Logging</h2>
      <p class="description">Configure application storage and logging.</p>

      <form onsubmit={handleSave}>
        <div class="field">
          <label for="dataDir">Data Directory</label>
          <input
            type="text"
            id="dataDir"
            bind:value={dataDir}
            placeholder="./data"
            required
          />
          <span class="hint">Directory where images and thumbnails are stored.</span>
        </div>

        <div class="field">
          <label for="logLevel">Log Level</label>
          <select id="logLevel" bind:value={logLevel}>
            <option value="debug">Debug</option>
            <option value="info">Info</option>
            <option value="warn">Warn</option>
            <option value="error">Error</option>
          </select>
          <span class="hint">Controls verbosity of application logging.</span>
        </div>

        <div class="actions">
          <button type="submit" class="primary-btn" disabled={updateMutation.isPending}>
            {updateMutation.isPending ? 'Saving...' : 'Save Settings'}
          </button>
          {#if saved}
            <span class="success">Settings saved!</span>
          {/if}
        </div>
      </form>
    </div>
  {/if}
</div>

<style>
  .page {
    padding: 1rem;
    max-width: 600px;
  }

  h1 {
    margin: 0 0 1.5rem;
    font-size: 1.5rem;
  }

  .settings-card {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .settings-card h2 {
    margin: 0 0 0.5rem;
    font-size: 1.1rem;
  }

  .description {
    color: #666;
    margin: 0 0 1.5rem;
    font-size: 0.9rem;
  }

  .field {
    margin-bottom: 1rem;
  }

  .field label {
    display: block;
    font-size: 0.875rem;
    color: #555;
    margin-bottom: 0.25rem;
  }

  .field input[type="text"],
  .field select {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box;
  }

  .field input:focus,
  .field select:focus {
    outline: none;
    border-color: #007bff;
  }

  .hint {
    display: block;
    font-size: 0.75rem;
    color: #888;
    margin-top: 0.25rem;
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-top: 1.5rem;
  }

  .primary-btn {
    background: #007bff;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .primary-btn:hover:not(:disabled) {
    background: #0056b3;
  }

  .primary-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .success {
    color: #2e7d32;
    font-size: 0.9rem;
  }

  .error {
    color: #c33;
  }
</style>
