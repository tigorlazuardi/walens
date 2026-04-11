<script lang="ts">
  import { useSources, useSourceTypes, useCreateSource, useUpdateSource, useDeleteSource } from '../lib/api/queries';
  import type { CreateSourceRequest, UpdateSourceRequest, Source } from '../lib/api/types';

  const sourcesQuery = useSources(() => ({}));
  const typesQuery = useSourceTypes();

  let showCreate = $state(false);
  let editingSource = $state<string | null>(null);

  // Form state
  let formName = $state('');
  let formType = $state('');
  let formParams = $state('{}');
  let formLookupCount = $state(100);
  let formEnabled = $state(true);

  // Pragmatic param fields for booru sources
  let formTags = $state('');
  let formRating = $state('');

  const createMutation = useCreateSource();
  const updateMutation = useUpdateSource();
  const deleteMutation = useDeleteSource();

  function resetForm() {
    formName = '';
    formType = typesQuery.data?.items[0]?.type_name || '';
    formParams = '{}';
    formLookupCount = 100;
    formEnabled = true;
    formTags = '';
    formRating = '';
  }

  function populateForm(source: Source) {
    formName = source.name;
    formType = source.source_type;
    formLookupCount = source.lookup_count;
    formEnabled = source.is_enabled;
    
    // Parse existing params
    if (source.params) {
      const p = source.params as Record<string, unknown>;
      if (source.source_type === 'booru') {
        formTags = (p.tags as string || '');
        formRating = (p.rating as string || '');
        formParams = JSON.stringify({ tags: formTags, rating: formRating }, null, 2);
      } else if (source.source_type === 'reddit') {
        formParams = JSON.stringify(p, null, 2);
      } else {
        formParams = JSON.stringify(source.params, null, 2);
      }
    } else {
      formParams = '{}';
    }
  }

  function startEdit(source: Source) {
    editingSource = source.id;
    populateForm(source);
  }

  function cancelEdit() {
    editingSource = null;
    resetForm();
  }

  function updateParamsFromFields() {
    if (formType === 'booru') {
      formParams = JSON.stringify({
        tags: formTags,
        rating: formRating,
      }, null, 2);
    }
  }

  $effect(() => {
    if (formType === 'booru' && !formTags && !formRating) {
      // Try to parse existing params when switching to booru
      try {
        const p = JSON.parse(formParams);
        if (p.tags) formTags = p.tags;
        if (p.rating) formRating = p.rating;
      } catch {}
    }
  });

  async function handleCreate(e: Event) {
    e.preventDefault();
    let params: Record<string, unknown> = {};
    
    if (formType === 'booru') {
      params = {
        tags: formTags,
        rating: formRating,
      };
    } else {
      try {
        params = JSON.parse(formParams);
      } catch {
        alert('Invalid JSON in params');
        return;
      }
    }

    const body: CreateSourceRequest = {
      name: formName,
      source_type: formType,
      params,
      lookup_count: formLookupCount,
      is_enabled: formEnabled,
    };
    await createMutation.mutateAsync(body);
    showCreate = false;
    resetForm();
  }

  async function handleUpdate(e: Event) {
    e.preventDefault();
    if (!editingSource) return;
    
    let params: Record<string, unknown> = {};
    
    if (formType === 'booru') {
      params = {
        tags: formTags,
        rating: formRating,
      };
    } else {
      try {
        params = JSON.parse(formParams);
      } catch {
        alert('Invalid JSON in params');
        return;
      }
    }

    const body: UpdateSourceRequest = {
      id: editingSource,
      name: formName,
      source_type: formType,
      params,
      lookup_count: formLookupCount,
      is_enabled: formEnabled,
    };
    await updateMutation.mutateAsync(body);
    editingSource = null;
    resetForm();
  }

  async function handleDelete(id: string) {
    if (confirm('Delete this source? Images from this source will not be deleted.')) {
      await deleteMutation.mutateAsync(id);
    }
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Sources</h1>
    <button class="primary-btn" onclick={() => { showCreate = true; resetForm(); }}>+ Add Source</button>
  </div>

  {#if sourcesQuery.isLoading}
    <p>Loading...</p>
  {:else if sourcesQuery.isError}
    <p class="error">Failed to load sources</p>
  {:else if sourcesQuery.data}
    <div class="source-list">
      {#each sourcesQuery.data.items as source}
        <div class="source-card">
          {#if editingSource === source.id}
            <form onsubmit={handleUpdate}>
              <div class="field">
                <label for="edit-name">Name</label>
                <input type="text" id="edit-name" bind:value={formName} required />
              </div>
              <div class="field">
                <label for="edit-type">Source Type</label>
                <select id="edit-type" bind:value={formType}>
                  {#if typesQuery.data}
                    {#each typesQuery.data.items as t}
                      <option value={t.type_name}>{t.display_name}</option>
                    {/each}
                  {/if}
                </select>
              </div>
              
              {#if formType === 'booru'}
                <div class="field">
                  <label for="edit-tags">Tags</label>
                  <input type="text" id="edit-tags" bind:value={formTags} placeholder="tag1 tag2 tag3" />
                </div>
                <div class="field">
                  <label for="edit-rating">Rating (optional)</label>
                  <input type="text" id="edit-rating" bind:value={formRating} placeholder="safe, questionable, explicit" />
                </div>
              {:else}
                <div class="field">
                  <label for="edit-params">Params (JSON)</label>
                  <textarea id="edit-params" bind:value={formParams} rows="4" placeholder="JSON object"></textarea>
                </div>
              {/if}
              
              <div class="field">
                <label for="edit-lookup">Lookup Count</label>
                <input type="number" id="edit-lookup" bind:value={formLookupCount} min="1" max="10000" />
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
            <div class="source-header">
              <h3>{source.name}</h3>
              <span class="type-badge">{source.source_type}</span>
            </div>
            <div class="source-info">
              <p>Lookup count: {source.lookup_count}</p>
              {#if source.params && Object.keys(source.params).length > 0}
                <p class="params-preview">{JSON.stringify(source.params).substring(0, 60)}{JSON.stringify(source.params).length > 60 ? '...' : ''}</p>
              {/if}
              <p class="status">
                {#if source.is_enabled}
                  <span class="badge green">Enabled</span>
                {:else}
                  <span class="badge gray">Disabled</span>
                {/if}
              </p>
            </div>
            <div class="source-actions">
              <button class="secondary-btn" onclick={() => startEdit(source)}>Edit</button>
              <button class="danger-btn" onclick={() => handleDelete(source.id)}>Delete</button>
            </div>
          {/if}
        </div>
      {:else}
        <p class="empty">No sources configured. Add one to get started.</p>
      {/each}
    </div>
  {/if}

  {#if showCreate}
    <div class="modal-overlay" role="dialog" aria-modal="true" onclick={() => { showCreate = false; }} onkeydown={(e) => e.key === 'Escape' && (showCreate = false)} tabindex="-1">
      <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="document">
        <h2>Add Source</h2>
        <form onsubmit={handleCreate}>
          <div class="field">
            <label for="name">Name</label>
            <input type="text" id="name" bind:value={formName} required />
          </div>
          <div class="field">
            <label for="type">Source Type</label>
            <select id="type" bind:value={formType}>
              {#if typesQuery.data}
                {#each typesQuery.data.items as t}
                  <option value={t.type_name}>{t.display_name}</option>
                {/each}
              {/if}
            </select>
          </div>
          
          {#if formType === 'booru'}
            <div class="field">
              <label for="tags">Tags</label>
              <input type="text" id="tags" bind:value={formTags} placeholder="tag1 tag2 tag3" />
            </div>
            <div class="field">
              <label for="rating">Rating (optional)</label>
              <input type="text" id="rating" bind:value={formRating} placeholder="safe, questionable, explicit" />
            </div>
          {:else}
            <div class="field">
              <label for="params">Params (JSON)</label>
              <textarea id="params" bind:value={formParams} rows="4" placeholder="JSON object"></textarea>
            </div>
          {/if}
          
          <div class="field">
            <label for="lookup">Lookup Count</label>
            <input type="number" id="lookup" bind:value={formLookupCount} min="1" max="10000" />
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

  .source-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .source-card {
    background: white;
    border-radius: 8px;
    padding: 1rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .source-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .source-header h3 {
    margin: 0;
    font-size: 1rem;
  }

  .type-badge {
    font-size: 0.7rem;
    padding: 0.125rem 0.5rem;
    background: #e3f2fd;
    color: #1976d2;
    border-radius: 3px;
    font-weight: 500;
  }

  .source-info p {
    margin: 0.25rem 0;
    font-size: 0.9rem;
    color: #666;
  }

  .params-preview {
    font-family: monospace;
    font-size: 0.8rem !important;
    color: #888 !important;
    word-break: break-all;
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

  .badge.gray {
    background: #f5f5f5;
    color: #666;
  }

  .source-actions {
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
  .field input[type="number"],
  .field select,
  .field textarea {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
    box-sizing: border-box;
    font-family: inherit;
  }

  .field textarea {
    resize: vertical;
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
