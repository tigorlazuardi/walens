<script lang="ts">
  import { useSubscriptions, useDevices, useSources, useCreateSubscription, useUpdateSubscription, useDeleteSubscription } from '../lib/api/queries';
  import type { CreateSubscriptionRequest, UpdateSubscriptionRequest } from '../lib/api/types';
  import { Badge, Button, Card, Checkbox, Select } from '../lib/components/ui';

  const subscriptionsQuery = useSubscriptions(() => ({}));
  const devicesQuery = useDevices(() => ({}));
  const sourcesQuery = useSources(() => ({}));
  const createMutation = useCreateSubscription();
  const updateMutation = useUpdateSubscription();
  const deleteMutation = useDeleteSubscription();

  let showCreate = $state(false);
  let editingSub = $state<string | null>(null);
  let formDeviceId = $state('');
  let formSourceId = $state('');
  let formEnabled = $state(true);

  function resetForm() {
    formDeviceId = (devicesQuery.data?.items ?? [])[0]?.id || '';
    formSourceId = (sourcesQuery.data?.items ?? [])[0]?.id || '';
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
    const body: CreateSubscriptionRequest = { device_id: formDeviceId, source_id: formSourceId, is_enabled: formEnabled };
    await createMutation.mutateAsync(body);
    showCreate = false;
    resetForm();
  }

  async function handleUpdate(e: Event) {
    e.preventDefault();
    if (!editingSub) return;
    const body: UpdateSubscriptionRequest = { id: editingSub, device_id: formDeviceId, source_id: formSourceId, is_enabled: formEnabled };
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
    const device = (devicesQuery.data?.items ?? []).find((d) => d.id === deviceId);
    return device?.name || deviceId;
  }

  function getSourceName(sourceId: string): string {
    const source = (sourcesQuery.data?.items ?? []).find((s) => s.id === sourceId);
    return source?.name || sourceId;
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <h1 class="text-2xl font-semibold tracking-tight">Subscriptions</h1>
    <Button onclick={() => { showCreate = true; resetForm(); }}>+ Add Subscription</Button>
  </div>

  {#if subscriptionsQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if subscriptionsQuery.isError}
    <p class="text-rose-600">Failed to load subscriptions</p>
  {:else if subscriptionsQuery.data}
    <div class="grid gap-4 lg:grid-cols-2">
      {#each subscriptionsQuery.data.items as sub}
        <Card class="p-4">
          {#if editingSub === sub.id}
            <form class="space-y-4" onsubmit={handleUpdate}>
              <div class="space-y-2"><label class="text-sm font-medium">Device</label><Select bind:value={formDeviceId}>{#if devicesQuery.data}{#each devicesQuery.data.items as device}<option value={device.id}>{device.name}</option>{/each}{/if}</Select></div>
              <div class="space-y-2"><label class="text-sm font-medium">Source</label><Select bind:value={formSourceId}>{#if sourcesQuery.data}{#each sourcesQuery.data.items as source}<option value={source.id}>{source.name}</option>{/each}{/if}</Select></div>
              <label class="flex items-center gap-2 text-sm text-slate-700"><Checkbox bind:checked={formEnabled} />Enabled</label>
              <div class="flex justify-end gap-2"><Button type="button" variant="outline" onclick={cancelEdit}>Cancel</Button><Button type="submit">Save</Button></div>
            </form>
          {:else}
            <div class="space-y-3">
              <div class="flex items-center justify-between gap-3">
                <p class="font-medium">{getDeviceName(sub.device_id)} <span class="text-slate-400">←</span> {getSourceName(sub.source_id)}</p>
                <Badge variant={sub.is_enabled ? 'success' : 'secondary'}>{sub.is_enabled ? 'Enabled' : 'Disabled'}</Badge>
              </div>
              <div class="flex justify-end gap-2"><Button variant="outline" size="sm" onclick={() => startEdit(sub)}>Edit</Button><Button variant="destructive" size="sm" onclick={() => handleDelete(sub.id)}>Delete</Button></div>
            </div>
          {/if}
        </Card>
      {/each}
    </div>
  {/if}

  {#if showCreate}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="dialog" aria-modal="true" onclick={() => { showCreate = false; }}>
      <Card class="w-full max-w-lg p-5" onclick={(e) => e.stopPropagation()}>
        <h2 class="text-lg font-semibold">Add Subscription</h2>
        <form class="mt-4 space-y-4" onsubmit={handleCreate}>
          <div class="space-y-2"><label class="text-sm font-medium">Device</label><Select bind:value={formDeviceId}>{#if devicesQuery.data}{#each devicesQuery.data.items as device}<option value={device.id}>{device.name}</option>{/each}{:else}<option value="">Loading devices...</option>{/if}</Select></div>
          <div class="space-y-2"><label class="text-sm font-medium">Source</label><Select bind:value={formSourceId}>{#if sourcesQuery.data}{#each sourcesQuery.data.items as source}<option value={source.id}>{source.name}</option>{/each}{:else}<option value="">Loading sources...</option>{/if}</Select></div>
          <label class="flex items-center gap-2 text-sm text-slate-700"><Checkbox bind:checked={formEnabled} />Enabled</label>
          <div class="flex justify-end gap-2"><Button type="button" variant="outline" onclick={() => { showCreate = false; }}>Cancel</Button><Button type="submit">Create</Button></div>
        </form>
      </Card>
    </div>
  {/if}
</div>
