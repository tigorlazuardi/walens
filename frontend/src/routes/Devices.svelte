<script lang="ts">
  import { useDevices, useCreateDevice, useUpdateDevice, useDeleteDevice } from '../lib/api/queries';
  import type { CreateDeviceRequest, UpdateDeviceRequest, Device } from '../lib/api/types';
  import { Badge, Button, Card, Checkbox, Input } from '../lib/components/ui';

  const devicesQuery = useDevices(() => ({}));
  const createMutation = useCreateDevice();
  const updateMutation = useUpdateDevice();
  const deleteMutation = useDeleteDevice();

  let showCreate = $state(false);
  let editingDevice = $state<string | null>(null);

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

  function generateSlug(name: string) {
    formSlug = name.toLowerCase().replace(/[^a-z0-9-]/g, '-');
  }

  async function handleCreate(e: Event) {
    e.preventDefault();
    const body: CreateDeviceRequest = {
      name: formName,
      slug: formSlug.toLowerCase().replace(/[^a-z0-9-]/g, '-'),
      screen_width: formWidth,
      screen_height: formHeight,
      min_image_width: formMinWidth,
      max_image_width: formMaxWidth,
      min_image_height: formMinHeight,
      max_image_height: formMaxHeight,
      min_filesize: formMinFilesize,
      max_filesize: formMaxFilesize,
      aspect_ratio_tolerance: formAspectTolerance,
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
      min_image_width: formMinWidth,
      max_image_width: formMaxWidth,
      min_image_height: formMinHeight,
      max_image_height: formMaxHeight,
      min_filesize: formMinFilesize,
      max_filesize: formMaxFilesize,
      aspect_ratio_tolerance: formAspectTolerance,
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

  function handleCreateDialogKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      showCreate = false;
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <h1 class="text-2xl font-semibold tracking-tight">Devices</h1>
    <Button onclick={() => { showCreate = true; resetForm(); }}>+ Add Device</Button>
  </div>

  {#if devicesQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if devicesQuery.isError}
    <p class="text-rose-600">Failed to load devices</p>
  {:else if devicesQuery.data}
    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {#each devicesQuery.data.items as device}
        <Card class="p-4">
          {#if editingDevice === device.id}
            <form class="space-y-4" onsubmit={handleUpdate}>
              <div class="space-y-2">
                <label for={`device-${device.id}-name`} class="text-sm font-medium">Name</label>
                <Input id={`device-${device.id}-name`} bind:value={formName} onblur={() => { if (!formSlug) generateSlug(formName); }} required />
              </div>
              <div class="space-y-2">
                <label for={`device-${device.id}-slug`} class="text-sm font-medium">Slug</label>
                <Input id={`device-${device.id}-slug`} bind:value={formSlug} pattern="[a-z0-9-]+" required />
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div class="space-y-2"><label for={`device-${device.id}-width`} class="text-sm font-medium">Screen Width</label><Input id={`device-${device.id}-width`} type="number" bind:value={formWidth} min="1" required /></div>
                <div class="space-y-2"><label for={`device-${device.id}-height`} class="text-sm font-medium">Screen Height</label><Input id={`device-${device.id}-height`} type="number" bind:value={formHeight} min="1" required /></div>
              </div>
              <details class="rounded-lg border border-slate-200 p-3">
                <summary class="cursor-pointer text-sm font-medium text-slate-600">Advanced</summary>
                <div class="mt-3 space-y-3">
                  <div class="grid grid-cols-2 gap-3">
                    <Input id={`device-${device.id}-min-width`} type="number" bind:value={formMinWidth} min="0" placeholder="Min Width" />
                    <Input id={`device-${device.id}-max-width`} type="number" bind:value={formMaxWidth} min="0" placeholder="Max Width" />
                  </div>
                  <div class="grid grid-cols-2 gap-3">
                    <Input id={`device-${device.id}-min-height`} type="number" bind:value={formMinHeight} min="0" placeholder="Min Height" />
                    <Input id={`device-${device.id}-max-height`} type="number" bind:value={formMaxHeight} min="0" placeholder="Max Height" />
                  </div>
                  <div class="grid grid-cols-2 gap-3">
                    <Input id={`device-${device.id}-min-filesize`} type="number" bind:value={formMinFilesize} min="0" placeholder="Min Filesize" />
                    <Input id={`device-${device.id}-max-filesize`} type="number" bind:value={formMaxFilesize} min="0" placeholder="Max Filesize" />
                  </div>
                  <Input id={`device-${device.id}-aspect-tolerance`} type="number" bind:value={formAspectTolerance} min="0" step="0.01" placeholder="Aspect Ratio Tolerance" />
                </div>
              </details>
              <div class="flex items-center gap-2 text-sm text-slate-700"><Checkbox id={`device-${device.id}-adult-allowed`} bind:checked={formAdultAllowed} /><label for={`device-${device.id}-adult-allowed`}>Allow Adult Content</label></div>
              <div class="flex items-center gap-2 text-sm text-slate-700"><Checkbox id={`device-${device.id}-enabled`} bind:checked={formEnabled} /><label for={`device-${device.id}-enabled`}>Enabled</label></div>
              <div class="flex justify-end gap-2">
                <Button type="button" variant="outline" onclick={cancelEdit}>Cancel</Button>
                <Button type="submit">Save</Button>
              </div>
            </form>
          {:else}
            <div class="space-y-3">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <h3 class="font-semibold">{device.name}</h3>
                  <p class="font-mono text-xs text-slate-500">{device.slug}</p>
                </div>
                <div class="flex gap-2">
                  <Badge variant={device.is_enabled ? 'success' : 'secondary'}>{device.is_enabled ? 'Enabled' : 'Disabled'}</Badge>
                  {#if device.is_adult_allowed}
                    <Badge variant="warning">Adult OK</Badge>
                  {/if}
                </div>
              </div>
              <p class="text-sm text-slate-600">{device.screen_width} × {device.screen_height}</p>
              <div class="flex justify-end gap-2">
                <Button variant="outline" size="sm" onclick={() => startEdit(device)}>Edit</Button>
                <Button variant="destructive" size="sm" onclick={() => handleDelete(device.id)}>Delete</Button>
              </div>
            </div>
          {/if}
        </Card>
      {/each}
    </div>
  {/if}

  {#if showCreate}
    <div class="fixed inset-0 z-50">
      <button type="button" class="absolute inset-0 bg-black/50" aria-label="Close dialog" onclick={() => { showCreate = false; }}></button>
      <div class="relative z-10 flex min-h-full items-center justify-center p-4" role="dialog" tabindex="-1" aria-modal="true" aria-labelledby="device-create-title" onkeydown={handleCreateDialogKeydown}>
      <Card class="w-full max-w-lg p-5">
        <h2 id="device-create-title" class="text-lg font-semibold">Add Device</h2>
        <form class="mt-4 space-y-4" onsubmit={handleCreate}>
          <div class="space-y-2"><label for="device-create-name" class="text-sm font-medium">Name</label><Input id="device-create-name" bind:value={formName} onblur={() => { if (!formSlug) generateSlug(formName); }} required /></div>
          <div class="space-y-2"><label for="device-create-slug" class="text-sm font-medium">Slug</label><Input id="device-create-slug" bind:value={formSlug} pattern="[a-z0-9-]+" required /></div>
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-2"><label for="device-create-width" class="text-sm font-medium">Screen Width</label><Input id="device-create-width" type="number" bind:value={formWidth} min="1" required /></div>
            <div class="space-y-2"><label for="device-create-height" class="text-sm font-medium">Screen Height</label><Input id="device-create-height" type="number" bind:value={formHeight} min="1" required /></div>
          </div>
          <details class="rounded-lg border border-slate-200 p-3">
            <summary class="cursor-pointer text-sm font-medium text-slate-600">Advanced</summary>
            <div class="mt-3 space-y-3">
              <div class="grid grid-cols-2 gap-3"><Input id="device-create-min-width" type="number" bind:value={formMinWidth} min="0" placeholder="Min Width" /><Input id="device-create-max-width" type="number" bind:value={formMaxWidth} min="0" placeholder="Max Width" /></div>
              <div class="grid grid-cols-2 gap-3"><Input id="device-create-min-height" type="number" bind:value={formMinHeight} min="0" placeholder="Min Height" /><Input id="device-create-max-height" type="number" bind:value={formMaxHeight} min="0" placeholder="Max Height" /></div>
              <div class="grid grid-cols-2 gap-3"><Input id="device-create-min-filesize" type="number" bind:value={formMinFilesize} min="0" placeholder="Min Filesize" /><Input id="device-create-max-filesize" type="number" bind:value={formMaxFilesize} min="0" placeholder="Max Filesize" /></div>
              <Input id="device-create-aspect-tolerance" type="number" bind:value={formAspectTolerance} min="0" step="0.01" placeholder="Aspect Ratio Tolerance" />
            </div>
          </details>
          <div class="flex items-center gap-2 text-sm text-slate-700"><Checkbox id="device-create-adult-allowed" bind:checked={formAdultAllowed} /><label for="device-create-adult-allowed">Allow Adult Content</label></div>
          <div class="flex items-center gap-2 text-sm text-slate-700"><Checkbox id="device-create-enabled" bind:checked={formEnabled} /><label for="device-create-enabled">Enabled</label></div>
          <div class="flex justify-end gap-2">
            <Button type="button" variant="outline" onclick={() => { showCreate = false; }}>Cancel</Button>
            <Button type="submit">Create</Button>
          </div>
        </form>
      </Card>
      </div>
    </div>
  {/if}
</div>
