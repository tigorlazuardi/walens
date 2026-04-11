<script lang="ts">
  import { useDevices, useCreateDevice, useUpdateDevice, useDeleteDevice } from '../lib/api/queries';
  import type { CreateDeviceRequest, UpdateDeviceRequest, Device } from '../lib/api/types';

  const devicesQuery = useDevices(() => ({}));

  let showCreate = $state(false);
  let editingDevice = $state<string | null>(null);

  // Form state
  let formName = $state('');
  let formSlug = $state('');
  let formWidth = $state(1920);
  let formHeight = $state(1080);
  let formMinWidth = $state(0);
  let formMaxWidth = $state(0);
  let formMinHeight = $state(0);
  let formMaxHeight = $state(0);
  let formMinFilesize = $state(0);
  let formMaxFilesize = $state(0);
  let formAspectTolerance = $state(0.1);
  let formAdultAllowed = $state(false);
  let formEnabled = $state(true);

  const createMutation = useCreateDevice();
  const updateMutation = useUpdateDevice();
  const deleteMutation = useDeleteDevice();

  function resetForm() {
    formName = '';
    formSlug = '';
    formWidth = 1920;
    formHeight = 1080;
    formMinWidth = 0;
    formMaxWidth = 0;
    formMinHeight = 0;
    formMaxHeight = 0;
    formMinFilesize = 0;
    formMaxFilesize = 0;
    formAspectTolerance = 0.1;
    formAdultAllowed = false;
    formEnabled = true;
  }

  function populateForm(device: Device) {
    formName = device.name;
    formSlug = device.slug;
    formWidth = device.screen_width;
    formHeight = device.screen_height;
    formMinWidth = device.min_image_width;
    formMaxWidth = device.max_image_width;
    formMinHeight = device.min_image_height;
    formMaxHeight = device.max_image_height;
    formMinFilesize = device.min_filesize;
    formMaxFilesize = device.max_filesize;
    formAspectTolerance = device.aspect_ratio_tolerance;
    formAdultAllowed = device.is_adult_allowed;
    formEnabled = device.is_enabled;
  }

  function startEdit(device: Device) {
    editingDevice = device.id;
    populateForm(device);
  }

  function cancelEdit() {
    editingDevice = null;
    resetForm();
  }

  async function handleCreate(e: Event) {
    e.preventDefault();
    const body: CreateDeviceRequest = {
      name: formName,
      slug: formSlug.toLowerCase().replace(/[^a-z0-9-]/g, '-'),
      screen_width: formWidth,
      screen_height: formHeight,
      min_image_width: formMinWidth || undefined,
      max_image_width: formMaxWidth || undefined,
      min_image_height: formMinHeight || undefined,
      max_image_height: formMaxHeight || undefined,
      min_filesize: formMinFilesize || undefined,
      max_filesize: formMaxFilesize || undefined,
      aspect_ratio_tolerance: formAspectTolerance || undefined,
      is_adult_allowed: formAdultAllowed,
      is_enabled: formEnabled,
    };
    await createMutation.mutateAsync(body);
    showCreate = false;
    resetForm();
  }

  async function handleUpdate(e: Event) {
    e.preventDefault();
    if (!editingDevice) return;
    const body: UpdateDeviceRequest = {
      id: editingDevice,
      name: formName,
      slug: formSlug.toLowerCase().replace(/[^a-z0-9-]/g, '-'),
      screen_width: formWidth,
      screen_height: formHeight,
      min_image_width: formMinWidth || undefined,
      max_image_width: formMaxWidth || undefined,
      min_image_height: formMinHeight || undefined,
      max_image_height: formMaxHeight || undefined,
      min_filesize: formMinFilesize || undefined,
      max_filesize: formMaxFilesize || undefined,
      aspect_ratio_tolerance: formAspectTolerance || undefined,
      is_adult_allowed: formAdultAllowed,
      is_enabled: formEnabled,
    };
    await updateMutation.mutateAsync(body);
    editingDevice = null;
    resetForm();
  }

  async function handleDelete(id: string) {
    if (confirm('Delete this device?')) {
      await deleteMutation.mutateAsync(id);
    }
  }

  function generateSlug(name: string) {
    formSlug = name.toLowerCase().replace(/[^a-z0-9-]/g, '-');
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Devices</h1>
    <button class="primary-btn" onclick={() => { showCreate = true; resetForm(); }}>+ Add Device</button>
  </div>

  {#if devicesQuery.isLoading}
    <p>Loading...</p>
  {:else if devicesQuery.isError}
    <p class="error">Failed to load devices</p>
  {:else if devicesQuery.data}
    <div class="device-grid">
      {#each devicesQuery.data.items as device}
        <div class="device-card">
          {#if editingDevice === device.id}
            <form onsubmit={handleUpdate}>
              <div class="field">
                <label for="edit-name">Name</label>
                <input type="text" id="edit-name" bind:value={formName} onblur={() => { if (!formSlug) generateSlug(formName); }} required />
              </div>
              <div class="field">
                <label for="edit-slug">Slug</label>
                <input type="text" id="edit-slug" bind:value={formSlug} pattern="[a-z0-9-]+" required />
              </div>
              <div class="field-row">
                <div class="field">
                  <label for="edit-width">Screen Width</label>
                  <input type="number" id="edit-width" bind:value={formWidth} min="1" required />
                </div>
                <div class="field">
                  <label for="edit-height">Screen Height</label>
                  <input type="number" id="edit-height" bind:value={formHeight} min="1" required />
                </div>
              </div>
              <details class="adv-section">
                <summary>Advanced</summary>
                <div class="field-row">
                  <div class="field">
                    <label for="edit-minw">Min Width</label>
                    <input type="number" id="edit-minw" bind:value={formMinWidth} min="0" />
                  </div>
                  <div class="field">
                    <label for="edit-maxw">Max Width</label>
                    <input type="number" id="edit-maxw" bind:value={formMaxWidth} min="0" />
                  </div>
                </div>
                <div class="field-row">
                  <div class="field">
                    <label for="edit-minh">Min Height</label>
                    <input type="number" id="edit-minh" bind:value={formMinHeight} min="0" />
                  </div>
                  <div class="field">
                    <label for="edit-maxh">Max Height</label>
                    <input type="number" id="edit-maxh" bind:value={formMaxHeight} min="0" />
                  </div>
                </div>
                <div class="field-row">
                  <div class="field">
                    <label for="edit-minsz">Min Filesize</label>
                    <input type="number" id="edit-minsz" bind:value={formMinFilesize} min="0" />
                  </div>
                  <div class="field">
                    <label for="edit-maxsz">Max Filesize</label>
                    <input type="number" id="edit-maxsz" bind:value={formMaxFilesize} min="0" />
                  </div>
                </div>
                <div class="field">
                  <label for="edit-art">Aspect Ratio Tolerance</label>
                  <input type="number" id="edit-art" bind:value={formAspectTolerance} min="0" step="0.01" />
                </div>
              </details>
              <div class="field">
                <label class="checkbox-label">
                  <input type="checkbox" bind:checked={formAdultAllowed} />
                  Allow Adult Content
                </label>
              </div>
              <div class="field">
                <label class="checkbox-label">
                  <input type="checkbox" bind:checked={formEnabled} />
                  Enabled
                </label>
              </div>
              <div class="modal-actions">
                <button type="button" class="secondary-btn" onclick={cancelEdit}>Cancel</button>
                <button type="submit" class="primary-btn">Save</button>
              </div>
            </form>
          {:else}
            <div class="device-header">
              <h3>{device.name}</h3>
              <span class="slug">{device.slug}</span>
            </div>
            <div class="device-info">
              <p>{device.screen_width} × {device.screen_height}</p>
              <p class="status">
                {#if device.is_enabled}
                  <span class="badge green">Enabled</span>
                {:else}
                  <span class="badge gray">Disabled</span>
                {/if}
                {#if device.is_adult_allowed}
                  <span class="badge yellow">Adult OK</span>
                {/if}
              </p>
            </div>
            <div class="device-actions">
              <button class="secondary-btn" onclick={() => startEdit(device)}>Edit</button>
              <button class="danger-btn" onclick={() => handleDelete(device.id)}>Delete</button>
            </div>
          {/if}
        </div>
      {:else}
        <p class="empty">No devices configured. Add one to get started.</p>
      {/each}
    </div>
  {/if}

  {#if showCreate}
    <div class="modal-overlay" role="dialog" aria-modal="true" onclick={() => { showCreate = false; }} onkeydown={(e) => e.key === 'Escape' && (showCreate = false)} tabindex="-1">
      <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="document">
        <h2>Add Device</h2>
        <form onsubmit={handleCreate}>
          <div class="field">
            <label for="name">Name</label>
            <input type="text" id="name" bind:value={formName} onblur={() => { if (!formSlug) generateSlug(formName); }} required />
          </div>
          <div class="field">
            <label for="slug">Slug</label>
            <input type="text" id="slug" bind:value={formSlug} pattern="[a-z0-9-]+" required />
          </div>
          <div class="field-row">
            <div class="field">
              <label for="width">Screen Width</label>
              <input type="number" id="width" bind:value={formWidth} min="1" required />
            </div>
            <div class="field">
              <label for="height">Screen Height</label>
              <input type="number" id="height" bind:value={formHeight} min="1" required />
            </div>
          </div>
          <details class="adv-section">
            <summary>Advanced</summary>
            <div class="field-row">
              <div class="field">
                <label for="minw">Min Width</label>
                <input type="number" id="minw" bind:value={formMinWidth} min="0" />
              </div>
              <div class="field">
                <label for="maxw">Max Width</label>
                <input type="number" id="maxw" bind:value={formMaxWidth} min="0" />
              </div>
            </div>
            <div class="field-row">
              <div class="field">
                <label for="minh">Min Height</label>
                <input type="number" id="minh" bind:value={formMinHeight} min="0" />
              </div>
              <div class="field">
                <label for="maxh">Max Height</label>
                <input type="number" id="maxh" bind:value={formMaxHeight} min="0" />
              </div>
            </div>
            <div class="field-row">
              <div class="field">
                <label for="minsz">Min Filesize</label>
                <input type="number" id="minsz" bind:value={formMinFilesize} min="0" />
              </div>
              <div class="field">
                <label for="maxsz">Max Filesize</label>
                <input type="number" id="maxsz" bind:value={formMaxFilesize} min="0" />
              </div>
            </div>
            <div class="field">
              <label for="art">Aspect Ratio Tolerance</label>
              <input type="number" id="art" bind:value={formAspectTolerance} min="0" step="0.01" />
            </div>
          </details>
          <div class="field">
            <label class="checkbox-label">
              <input type="checkbox" bind:checked={formAdultAllowed} />
              Allow Adult Content
            </label>
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

  .device-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1rem;
  }

  .device-card {
    background: white;
    border-radius: 8px;
    padding: 1rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .device-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 0.5rem;
  }

  .device-header h3 {
    margin: 0;
    font-size: 1rem;
  }

  .slug {
    font-size: 0.75rem;
    color: #888;
    font-family: monospace;
  }

  .device-info p {
    margin: 0.25rem 0;
    font-size: 0.9rem;
    color: #666;
  }

  .status {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.5rem;
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

  .badge.yellow {
    background: #fff8e1;
    color: #f57c00;
  }

  .badge.gray {
    background: #f5f5f5;
    color: #666;
  }

  .device-actions {
    margin-top: 1rem;
    display: flex;
    gap: 0.5rem;
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
    padding: 0.375rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85rem;
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
  .field input[type="number"] {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box;
  }

  .field-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
  }

  .adv-section {
    margin-bottom: 1rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    padding: 0.5rem;
  }

  .adv-section summary {
    cursor: pointer;
    font-size: 0.875rem;
    color: #666;
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
