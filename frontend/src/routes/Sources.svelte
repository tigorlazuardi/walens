<script lang="ts">
  import { useSources, useSourceTypes, useCreateSource, useUpdateSource, useDeleteSource } from '../lib/api/queries';
  import type { CreateSourceRequest, UpdateSourceRequest, Source } from '../lib/api/types';
  import { Badge, Button, Card, Checkbox, Input, Select, Textarea } from '../lib/components/ui';

  const sourcesQuery = useSources(() => ({}));
  const typesQuery = useSourceTypes();
  const createMutation = useCreateSource();
  const updateMutation = useUpdateSource();
  const deleteMutation = useDeleteSource();

  let showCreate = $state(false);
  let editingSource = $state<string | null>(null);

  let formName = $state('');
  let formType = $state('');
  let formParams = $state('{}');
  let formLookupCount = $state(100);
  let formEnabled = $state(true);
  let formTags = $state('');
  let formRating = $state('');

  function resetForm() {
    formName = '';
    formType = (typesQuery.data?.items ?? [])[0]?.type_name || '';
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
    if (source.params) {
      const p = source.params as unknown as Record<string, unknown>;
      if (source.source_type === 'booru') {
        formTags = (p.tags as string || '');
        formRating = (p.rating as string || '');
        formParams = JSON.stringify({ tags: formTags, rating: formRating }, null, 2);
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

  async function handleCreate(e: Event) {
    e.preventDefault();
    const params = formType === 'booru'
      ? { tags: formTags, rating: formRating }
      : (() => { try { return JSON.parse(formParams); } catch { alert('Invalid JSON in params'); return null; } })();
    if (!params) return;
    const body: CreateSourceRequest = { name: formName, source_type: formType, params, lookup_count: formLookupCount, is_enabled: formEnabled };
    await createMutation.mutateAsync(body);
    showCreate = false;
    resetForm();
  }

  async function handleUpdate(e: Event) {
    e.preventDefault();
    if (!editingSource) return;
    const params = formType === 'booru'
      ? { tags: formTags, rating: formRating }
      : (() => { try { return JSON.parse(formParams); } catch { alert('Invalid JSON in params'); return null; } })();
    if (!params) return;
    const body: UpdateSourceRequest = { id: editingSource, name: formName, source_type: formType, params, lookup_count: formLookupCount, is_enabled: formEnabled };
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

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <h1 class="text-2xl font-semibold tracking-tight">Sources</h1>
    <Button onclick={() => { showCreate = true; resetForm(); }}>+ Add Source</Button>
  </div>

  {#if sourcesQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if sourcesQuery.isError}
    <p class="text-rose-600">Failed to load sources</p>
  {:else if sourcesQuery.data}
    <div class="grid gap-4 lg:grid-cols-2">
      {#each sourcesQuery.data.items ?? [] as source}
        <Card class="p-4">
          {#if editingSource === source.id}
            <form class="space-y-4" onsubmit={handleUpdate}>
              <div class="space-y-2"><label class="text-sm font-medium">Name</label><Input bind:value={formName} required /></div>
              <div class="space-y-2"><label class="text-sm font-medium">Source Type</label><Select bind:value={formType}>{#if typesQuery.data}{#each typesQuery.data.items ?? [] as t}<option value={t.type_name}>{t.display_name}</option>{/each}{/if}</Select></div>
              {#if formType === 'booru'}
                <div class="space-y-2"><label class="text-sm font-medium">Tags</label><Input bind:value={formTags} placeholder="tag1 tag2 tag3" /></div>
                <div class="space-y-2"><label class="text-sm font-medium">Rating (optional)</label><Input bind:value={formRating} placeholder="safe, questionable, explicit" /></div>
              {:else}
                <div class="space-y-2"><label class="text-sm font-medium">Params (JSON)</label><Textarea bind:value={formParams} rows="6" placeholder="JSON object" /></div>
              {/if}
              <div class="space-y-2"><label class="text-sm font-medium">Lookup Count</label><Input type="number" bind:value={formLookupCount} min="1" max="10000" /></div>
              <label class="flex items-center gap-2 text-sm text-slate-700"><Checkbox bind:checked={formEnabled} />Enabled</label>
              <div class="flex justify-end gap-2"><Button type="button" variant="outline" onclick={cancelEdit}>Cancel</Button><Button type="submit">Save</Button></div>
            </form>
          {:else}
            <div class="space-y-3">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <h3 class="font-semibold">{source.name}</h3>
                  <p class="text-xs text-slate-500">{source.source_type}</p>
                </div>
                <Badge variant={source.is_enabled ? 'success' : 'secondary'}>{source.is_enabled ? 'Enabled' : 'Disabled'}</Badge>
              </div>
              <p class="text-sm text-slate-600">Lookup count: {source.lookup_count}</p>
              {#if source.params && Object.keys(source.params).length > 0}
                <p class="break-all font-mono text-xs text-slate-500">{JSON.stringify(source.params)}</p>
              {/if}
              <div class="flex justify-end gap-2">
                <Button variant="outline" size="sm" onclick={() => startEdit(source)}>Edit</Button>
                <Button variant="destructive" size="sm" onclick={() => handleDelete(source.id)}>Delete</Button>
              </div>
            </div>
          {/if}
        </Card>
      {/each}
    </div>
  {/if}

  {#if showCreate}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="dialog" aria-modal="true" onclick={() => { showCreate = false; }}>
      <Card class="w-full max-w-lg p-5" onclick={(e) => e.stopPropagation()}>
        <h2 class="text-lg font-semibold">Add Source</h2>
        <form class="mt-4 space-y-4" onsubmit={handleCreate}>
          <div class="space-y-2"><label class="text-sm font-medium">Name</label><Input bind:value={formName} required /></div>
          <div class="space-y-2"><label class="text-sm font-medium">Source Type</label><Select bind:value={formType}>{#if typesQuery.data}{#each typesQuery.data.items ?? [] as t}<option value={t.type_name}>{t.display_name}</option>{/each}{/if}</Select></div>
          {#if formType === 'booru'}
            <div class="space-y-2"><label class="text-sm font-medium">Tags</label><Input bind:value={formTags} placeholder="tag1 tag2 tag3" /></div>
            <div class="space-y-2"><label class="text-sm font-medium">Rating (optional)</label><Input bind:value={formRating} placeholder="safe, questionable, explicit" /></div>
          {:else}
            <div class="space-y-2"><label class="text-sm font-medium">Params (JSON)</label><Textarea bind:value={formParams} rows="6" placeholder="JSON object" /></div>
          {/if}
          <div class="space-y-2"><label class="text-sm font-medium">Lookup Count</label><Input type="number" bind:value={formLookupCount} min="1" max="10000" /></div>
          <label class="flex items-center gap-2 text-sm text-slate-700"><Checkbox bind:checked={formEnabled} />Enabled</label>
          <div class="flex justify-end gap-2"><Button type="button" variant="outline" onclick={() => { showCreate = false; }}>Cancel</Button><Button type="submit">Create</Button></div>
        </form>
      </Card>
    </div>
  {/if}
</div>
