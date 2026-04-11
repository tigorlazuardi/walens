<script lang="ts">
  import { useSubscriptions, useDevices, useSources, useCreateSubscription, useUpdateSubscription, useDeleteSubscription } from '../lib/api/queries';
  import type { CreateSubscriptionRequest, UpdateSubscriptionRequest } from '../lib/api/types';

  const subscriptionsQuery = useSubscriptions(() => ({}));
  const devicesQuery = useDevices(() => ({}));
  const sourcesQuery = useSources(() => ({}));

  let showCreate = $state(false);
  let editingSub = $state<string | null>(null);

  // Form state
  let formDeviceId = $state('');
  let formSourceId = $state('');
  let formEnabled = $state(true);

  const createMutation = useCreateSubscription();
  const updateMutation = useUpdateSubscription();
  const deleteMutation = useDeleteSubscription();

  function resetForm() {
    formDeviceId = devicesQuery.data?.items[0]?.id || '';
    formSourceId = sourcesQuery.data?.items[0]?.id || '';
    formEnabled = true;
  }

  function startEdit(sub: any) {
    editingSub = sub.id;
    formDeviceId = sub.device_id;
    formSourceId = sub.source_id;
    formEnabled = sub.is_enabled;
  }

  function cancelEdit() {
    editingSub = null;
    resetForm();
  }

  async function handleCreate(e: Event) {
    e.preventDefault();
    const body: CreateSubscriptionRequest = {
      device_id: formDeviceId,
      source_id: formSourceId,
      is_enabled: formEnabled,
    };
    await createMutation.mutateAsync(body);
    showCreate = false;
    resetForm();
  }

  async function handleUpdate(e: Event) {
    e.preventDefault();
    if (!editingSub) return;
    const body: UpdateSubscriptionRequest = {
      id: editingSub,
      device_id: formDeviceId,
      source_id: formSourceId,
      is_enabled: formEnabled,
    };
    await updateMutation.mutateAsync(body);
    editingSub = null;
    resetForm();
  }

  async function handleDelete(id: string) {
    if (confirm('Delete this subscription?')) {
      await deleteMutation.mutateAsync(id);
    }
  }

  function getDeviceName(deviceId: string): string {
    const device = devicesQuery.data?.items.find(d => d.id === deviceId);
    return device?.name || deviceId;
  }

  function getSourceName(sourceId: string): string {
    const source = sourcesQuery.data?.items.find(s => s.id === sourceId);
    return source?.name || sourceId;
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Subscriptions</h1>
    <button class="primary-btn" onclick={() => { showCreate = true; resetForm(); }}>+ Add Subscription</button>
  </div>

  {#if subscriptionsQuery.isLoading}
    <p>Loading...</p>
  {:else if subscriptionsQuery.isError}
    <p class="error">Failed to load subscriptions</p>
  {:else if subscriptionsQuery.data}
    <div class="sub-list">
      {#each subscriptionsQuery.data.items as sub}
        <div class="sub-card">
          {#if editingSub === sub.id}
            <form onsubmit={handleUpdate} class="edit-form">
              <div class="field">
                <label for="edit-device">Device</label>
                <select id="edit-device" bind:value={formDeviceId}>
                  {#if devicesQuery.data}
                    {#each devicesQuery.data.items as device}
                      <option value={device.id}>{device.name}</option>
                    {/each}
                  {/if}
                </select>
              </div>
              <div class="field">
                <label for="edit-source">Source</label>
                <select id="edit-source" bind:value={formSourceId}>
                  {#if sourcesQuery.data}
                    {#each sourcesQuery.data.items as source}
                      <option value={source.id}>{source.name}</option>
                    {/each}
                  {/if}
                </select>
              </div>
              <div class="field">
                <label class="checkbox-label">
                  <input type="checkbox" bind:checked={formEnabled} />
                  Enabled
                </label>
              </div>
              <div class="form-actions">
                <button type="button" class="secondary-btn" onclick={cancelEdit}>Cancel</button>
                <button type="submit" class="primary-btn">Save</button>
              </div>
            </form>
          {:else}
            <div class="sub-header">
              <div class="sub-names">
                <span class="device-name">{getDeviceName(sub.device_id)}</span>
                <span class="arrow">←</span>
                <span class="source-name">{getSourceName(sub.source_id)}</span>
              </div>
            </div>
            <div class="sub-info">
              <p class="status">
                {#if sub.is_enabled}
                  <span class="badge green">Enabled</span>
                {:else}
                  <span class="badge gray">Disabled</span>
                {/if}
              </p>
            </div>
            <div class="sub-actions">
              <button class="secondary-btn" onclick={() => startEdit(sub)}>Edit</button>
              <button class="danger-btn" onclick={() => handleDelete(sub.id)}>Delete</button>
            </div>
          {/if}
        </div>
      {:else}
        <p class="empty">No subscriptions configured. Add one to link devices to sources.</p>
      {/each}
    </div>
  {/if}

  {#if showCreate}
    <div class="modal-overlay" role="dialog" aria-modal="true" onclick={() => { showCreate = false; }} onkeydown={(e) => e.key === 'Escape' && (showCreate = false)} tabindex="-1">
      <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="document">
        <h2>Add Subscription</h2>
        <form onsubmit={handleCreate}>
          <div class="field">
            <label for="device">Device</label>
            <select id="device" bind:value={formDeviceId}>
              {#if devicesQuery.data}
                {#each devicesQuery.data.items as device}
                  <option value={device.id}>{device.name}</option>
                {/each}
              {:else}
                <option value="">Loading devices...</option>
              {/if}
            </select>
          </div>
          <div class="field">
            <label for="source">Source</label>
            <select id="source" bind:value={formSourceId}>
              {#if sourcesQuery.data}
                {#each sourcesQuery.data.items as source}
                  <option value={source.id}>{source.name}</option>
                {/each}
              {:else}
                <option value="">Loading sources...</option>
              {/if}
            </select>
          </div>
          <div class="field">
            <label class="checkbox-label">
              <input type="checkbox" bind:checked={formEnabled} />
              Enabled
            </label>
          </div>
          <div class="modal-actions">
            <button type="button" class="secondary-btn" onclick={() => { showCreate = false; }}>Cancel</button>
            <button type="submit" class="primary-btn">Create</button>
          </div>
        </form>
      </div>
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

  .sub-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .sub-card {
    background: white;
    border-radius: 8px;
    padding: 1rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .sub-header {
    margin-bottom: 0.5rem;
  }

  .sub-names {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 1rem;
  }

  .device-name {
    font-weight: 600;
    color: #333;
  }

  .arrow {
    color: #888;
  }

  .source-name {
    color: #1976d2;
  }

  .sub-info p {
    margin: 0.25rem 0;
    font-size: 0.9rem;
    color: #666;
  }

  .status {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }

  .sub-actions {
    margin-top: 1rem;
    display: flex;
    gap: 0.5rem;
  }

  .edit-form {
    padding: 0.5rem 0;
  }

  .form-actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
    margin-top: 1rem;
  }

  .badge {
    font-size: 0.7rem;
    padding: 0.125rem 0.375rem;
    border-radius: 3px;
    font-weight: 500;
  }

  .badge.green {
    background: #e8f5e9;
    color: #2e7d32;
  }

  .badge.gray {
    background: #f5f5f5;
    color: #666;
  }

  .empty {
    color: #888;
    font-style: italic;
  }

  .error {
    color: #c33;
  }

  /* Buttons */
  .primary-btn {
    background: #007bff;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .primary-btn:hover {
    background: #0056b3;
  }

  .secondary-btn {
    background: #f5f5f5;
    color: #333;
    border: 1px solid #ddd;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .secondary-btn:hover {
    background: #e8e8e8;
  }

  .danger-btn {
    background: #ffebee;
    color: #c62828;
    border: 1px solid #ffcdd2;
    padding: 0.375rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .danger-btn:hover {
    background: #ffcdd2;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    padding: 1rem;
  }

  .modal {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    width: 100%;
    max-width: 400px;
    max-height: 90vh;
    overflow-y: auto;
  }

  .modal h2 {
    margin: 0 0 1rem;
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

  .field select {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box;
  }

  .checkbox-label {
    display: flex !important;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }

  .checkbox-label input {
    width: auto;
  }

  .modal-actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
    margin-top: 1.5rem;
  }
</style>
