<script lang="ts">
  import { useSchedules, useSources, useCreateSchedule, useUpdateSchedule, useDeleteSchedule } from '../lib/api/queries';
  import type { CreateScheduleRequest, UpdateScheduleRequest } from '../lib/api/types';

  const schedulesQuery = useSchedules(() => ({}));
  const sourcesQuery = useSources(() => ({}));

  let showCreate = $state(false);
  let editingSchedule = $state<string | null>(null);

  // Form state
  let formSourceId = $state('');
  let formCron = $state('');
  let formEnabled = $state(true);

  const createMutation = useCreateSchedule();
  const updateMutation = useUpdateSchedule();
  const deleteMutation = useDeleteSchedule();

  function resetForm() {
    formSourceId = sourcesQuery.data?.items[0]?.id || '';
    formCron = '';
    formEnabled = true;
  }

  function startEdit(schedule: any) {
    editingSchedule = schedule.id;
    formSourceId = schedule.source_id;
    formCron = schedule.cron_expr;
    formEnabled = schedule.is_enabled;
  }

  function cancelEdit() {
    editingSchedule = null;
    resetForm();
  }

  async function handleCreate(e: Event) {
    e.preventDefault();
    const body: CreateScheduleRequest = {
      source_id: formSourceId,
      cron_expr: formCron,
      is_enabled: formEnabled,
    };
    await createMutation.mutateAsync(body);
    showCreate = false;
    resetForm();
  }

  async function handleUpdate(e: Event) {
    e.preventDefault();
    if (!editingSchedule) return;
    const body: UpdateScheduleRequest = {
      id: editingSchedule,
      source_id: formSourceId,
      cron_expr: formCron,
      is_enabled: formEnabled,
    };
    await updateMutation.mutateAsync(body);
    editingSchedule = null;
    resetForm();
  }

  async function handleDelete(id: string) {
    if (confirm('Delete this schedule?')) {
      await deleteMutation.mutateAsync(id);
    }
  }

  function getSourceName(sourceId: string): string {
    const source = sourcesQuery.data?.items.find(s => s.id === sourceId);
    return source?.name || sourceId;
  }

  function formatCron(expr: string): string {
    // Simple display - could be enhanced with a cron parser
    return expr;
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Schedules</h1>
    <button class="primary-btn" onclick={() => { showCreate = true; resetForm(); }}>+ Add Schedule</button>
  </div>

  {#if schedulesQuery.isLoading}
    <p>Loading...</p>
  {:else if schedulesQuery.isError}
    <p class="error">Failed to load schedules</p>
  {:else if schedulesQuery.data}
    <div class="schedule-list">
      {#each schedulesQuery.data.items as schedule}
        <div class="schedule-card">
          {#if editingSchedule === schedule.id}
            <form onsubmit={handleUpdate} class="edit-form">
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
                <label for="edit-cron">Cron Expression</label>
                <input type="text" id="edit-cron" bind:value={formCron} required placeholder="* * * * *" />
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
            <div class="schedule-header">
              <h3>{getSourceName(schedule.source_id)}</h3>
              <span class="cron-badge">{formatCron(schedule.cron_expr)}</span>
            </div>
            <div class="schedule-info">
              <p class="status">
                {#if schedule.is_enabled}
                  <span class="badge green">Enabled</span>
                {:else}
                  <span class="badge gray">Disabled</span>
                {/if}
                {#if schedule.next_run_at}
                  <span class="next-run">Next: {new Date(schedule.next_run_at).toLocaleString()}</span>
                {/if}
              </p>
            </div>
            <div class="schedule-actions">
              <button class="secondary-btn" onclick={() => startEdit(schedule)}>Edit</button>
              <button class="danger-btn" onclick={() => handleDelete(schedule.id)}>Delete</button>
            </div>
          {/if}
        </div>
      {:else}
        <p class="empty">No schedules configured. Add one to automate source syncs.</p>
      {/each}
    </div>
  {/if}

  {#if showCreate}
    <div class="modal-overlay" role="dialog" aria-modal="true" onclick={() => { showCreate = false; }} onkeydown={(e) => e.key === 'Escape' && (showCreate = false)} tabindex="-1">
      <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="document">
        <h2>Add Schedule</h2>
        <form onsubmit={handleCreate}>
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
            <label for="cron">Cron Expression</label>
            <input type="text" id="cron" bind:value={formCron} required placeholder="* * * * *" />
            <span class="hint">Format: minute hour day month weekday</span>
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

  .schedule-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .schedule-card {
    background: white;
    border-radius: 8px;
    padding: 1rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .schedule-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .schedule-header h3 {
    margin: 0;
    font-size: 1rem;
  }

  .cron-badge {
    font-size: 0.75rem;
    font-family: monospace;
    background: #f5f5f5;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    color: #333;
  }

  .schedule-info p {
    margin: 0.25rem 0;
    font-size: 0.9rem;
    color: #666;
  }

  .status {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    margin-top: 0.5rem;
  }

  .next-run {
    font-size: 0.8rem;
    color: #888;
  }

  .schedule-actions {
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

  .field input[type="text"],
  .field select {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box;
  }

  .hint {
    display: block;
    font-size: 0.75rem;
    color: #888;
    margin-top: 0.25rem;
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
