<script lang="ts">
  import { useSchedules, useSources, useCreateSchedule, useUpdateSchedule, useDeleteSchedule } from '../lib/api/queries';
  import type { CreateScheduleRequest, UpdateScheduleRequest } from '../lib/api/types';
  import { Badge, Button, Card, Checkbox, Input, Select } from '../lib/components/ui';

  const schedulesQuery = useSchedules(() => ({}));
  const sourcesQuery = useSources(() => ({}));
  const createMutation = useCreateSchedule();
  const updateMutation = useUpdateSchedule();
  const deleteMutation = useDeleteSchedule();

  let showCreate = $state(false);
  let editingSchedule = $state<string | null>(null);
  let formSourceId = $state('');
  let formCron = $state('');
  let formEnabled = $state(true);

  function resetForm() {
    formSourceId = (sourcesQuery.data?.items ?? [])[0]?.id || '';
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
    const body: CreateScheduleRequest = { source_id: formSourceId, cron_expr: formCron, is_enabled: formEnabled };
    await createMutation.mutateAsync(body);
    showCreate = false;
    resetForm();
  }

  async function handleUpdate(e: Event) {
    e.preventDefault();
    if (!editingSchedule) return;
    const body: UpdateScheduleRequest = { id: editingSchedule, source_id: formSourceId, cron_expr: formCron, is_enabled: formEnabled };
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
    const source = (sourcesQuery.data?.items ?? []).find((s) => s.id === sourceId);
    return source?.name || sourceId;
  }

  function handleCreateDialogKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      showCreate = false;
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <h1 class="text-2xl font-semibold tracking-tight">Schedules</h1>
    <Button onclick={() => { showCreate = true; resetForm(); }}>+ Add Schedule</Button>
  </div>

  {#if schedulesQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if schedulesQuery.isError}
    <p class="text-rose-600">Failed to load schedules</p>
  {:else if schedulesQuery.data}
    <div class="grid gap-4 lg:grid-cols-2">
      {#each schedulesQuery.data.items as schedule}
        <Card class="p-4">
          {#if editingSchedule === schedule.id}
            <form class="space-y-4" onsubmit={handleUpdate}>
              <div class="space-y-2"><label for={`schedule-${schedule.id}-source`} class="text-sm font-medium">Source</label><Select id={`schedule-${schedule.id}-source`} bind:value={formSourceId}>{#if sourcesQuery.data}{#each sourcesQuery.data.items as source}<option value={source.id}>{source.name}</option>{/each}{/if}</Select></div>
              <div class="space-y-2"><label for={`schedule-${schedule.id}-cron`} class="text-sm font-medium">Cron Expression</label><Input id={`schedule-${schedule.id}-cron`} bind:value={formCron} required placeholder="* * * * *" /></div>
              <div class="flex items-center gap-2 text-sm text-slate-700"><Checkbox id={`schedule-${schedule.id}-enabled`} bind:checked={formEnabled} /><label for={`schedule-${schedule.id}-enabled`}>Enabled</label></div>
              <div class="flex justify-end gap-2"><Button type="button" variant="outline" onclick={cancelEdit}>Cancel</Button><Button type="submit">Save</Button></div>
            </form>
          {:else}
            <div class="space-y-3">
              <div class="flex items-start justify-between gap-3">
                <h3 class="font-semibold">{getSourceName(schedule.source_id)}</h3>
                <Badge variant={schedule.is_enabled ? 'success' : 'secondary'}>{schedule.is_enabled ? 'Enabled' : 'Disabled'}</Badge>
              </div>
              <p class="font-mono text-xs text-slate-500">{schedule.cron_expr}</p>
              <div class="flex justify-end gap-2"><Button variant="outline" size="sm" onclick={() => startEdit(schedule)}>Edit</Button><Button variant="destructive" size="sm" onclick={() => handleDelete(schedule.id)}>Delete</Button></div>
            </div>
          {/if}
        </Card>
      {/each}
    </div>
  {/if}

  {#if showCreate}
    <div class="fixed inset-0 z-50">
      <button type="button" class="absolute inset-0 bg-black/50" aria-label="Close dialog" onclick={() => { showCreate = false; }}></button>
      <div class="relative z-10 flex min-h-full items-center justify-center p-4" role="dialog" tabindex="-1" aria-modal="true" aria-labelledby="schedule-create-title" onkeydown={handleCreateDialogKeydown}>
      <Card class="w-full max-w-lg p-5">
        <h2 id="schedule-create-title" class="text-lg font-semibold">Add Schedule</h2>
        <form class="mt-4 space-y-4" onsubmit={handleCreate}>
          <div class="space-y-2"><label for="schedule-create-source" class="text-sm font-medium">Source</label><Select id="schedule-create-source" bind:value={formSourceId}>{#if sourcesQuery.data}{#each sourcesQuery.data.items as source}<option value={source.id}>{source.name}</option>{/each}{:else}<option value="">Loading sources...</option>{/if}</Select></div>
          <div class="space-y-2"><label for="schedule-create-cron" class="text-sm font-medium">Cron Expression</label><Input id="schedule-create-cron" bind:value={formCron} required placeholder="* * * * *" /><span class="text-xs text-slate-500">Format: minute hour day month weekday</span></div>
          <div class="flex items-center gap-2 text-sm text-slate-700"><Checkbox id="schedule-create-enabled" bind:checked={formEnabled} /><label for="schedule-create-enabled">Enabled</label></div>
          <div class="flex justify-end gap-2"><Button type="button" variant="outline" onclick={() => { showCreate = false; }}>Cancel</Button><Button type="submit">Create</Button></div>
        </form>
      </Card>
      </div>
    </div>
  {/if}
</div>
