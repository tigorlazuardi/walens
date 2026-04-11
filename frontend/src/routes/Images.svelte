<script lang="ts">
  import { useImages, useSetImageFavorite, useBlacklistImage, useDeleteImage } from '../lib/api/queries';
  import { getRuntimeConfig } from '../lib/runtime';
  import { joinBasePath } from '../lib/path';
  import type { Image } from '../lib/api/types';
  import { Button, Card, Checkbox, Input } from '../lib/components/ui';

  const imagesQuery = useImages(() => ({}));
  const config = getRuntimeConfig();

  let favoritesOnly = $state(false);
  let searchQuery = $state('');

  const favoriteMutation = useSetImageFavorite();
  const blacklistMutation = useBlacklistImage();
  const deleteMutation = useDeleteImage();

  async function handleFavorite(image: Image) {
    await favoriteMutation.mutateAsync({ id: image.id, favorite: !image.is_favorite });
  }

  async function handleBlacklist(image: Image) {
    if (confirm('Blacklist this image? It will be hidden from the gallery.')) {
      await blacklistMutation.mutateAsync({ id: image.id });
    }
  }

  async function handleDelete(image: Image) {
    if (confirm('Delete this image? This cannot be undone.')) {
      await deleteMutation.mutateAsync({ id: image.id });
    }
  }

  function getThumbnailUrl(imageId: string): string {
    return joinBasePath(config.apiBase, 'v1/images/thumbnail', imageId);
  }

  function getImageUrl(imageId: string): string {
    return joinBasePath(config.apiBase, 'v1/images/image', imageId);
  }

  let filteredImages = $derived((imagesQuery.data?.items ?? []).filter((img) => {
    if (favoritesOnly && !img.is_favorite) return false;
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      return (
        img.artist?.toLowerCase().includes(q) ||
        img.uploader?.toLowerCase().includes(q) ||
        img.source_type?.toLowerCase().includes(q)
      );
    }
    return true;
  }));
</script>

<div class="space-y-4">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <h1 class="text-2xl font-semibold tracking-tight">Images</h1>
    <div class="flex flex-wrap items-center gap-3">
      <label for="favorites-only" class="flex items-center gap-2 text-sm text-slate-600">
        <Checkbox id="favorites-only" bind:checked={favoritesOnly} />
        Favorites only
      </label>
      <Input class="w-56" placeholder="Search..." bind:value={searchQuery} />
    </div>
  </div>

  {#if imagesQuery.isLoading}
    <p class="text-slate-500">Loading...</p>
  {:else if imagesQuery.isError}
    <p class="text-rose-600">Failed to load images</p>
  {:else if imagesQuery.data}
    <div class="columns-2 gap-3 sm:columns-3 lg:columns-4 xl:columns-5">
      {#each filteredImages as image}
        <Card class={image.is_favorite ? 'mb-3 break-inside-avoid overflow-hidden ring-2 ring-amber-400' : 'mb-3 break-inside-avoid overflow-hidden'}>
          <div class="group relative bg-slate-100">
            <a href={getImageUrl(image.id)} target="_blank" rel="noopener noreferrer">
              <img src={getThumbnailUrl(image.id)} alt="" loading="lazy" class="block h-auto w-full" />
            </a>
            <div class="absolute right-2 top-2 flex gap-2 opacity-0 transition-opacity group-hover:opacity-100">
              <Button size="icon" variant="outline" class="h-8 w-8 bg-black/60 text-white border-white/20 hover:bg-black/80" onclick={() => handleFavorite(image)} title={image.is_favorite ? 'Unfavorite' : 'Favorite'}>
                {image.is_favorite ? '★' : '☆'}
              </Button>
              <Button size="icon" variant="destructive" class="h-8 w-8" onclick={() => handleBlacklist(image)} title="Blacklist">✕</Button>
              <Button size="icon" variant="destructive" class="h-8 w-8" onclick={() => handleDelete(image)} title="Delete">🗑</Button>
            </div>
          </div>
          <div class="flex items-center justify-between gap-2 p-3 text-xs text-slate-600">
            {#if image.width && image.height}
              <span class="font-mono">{image.width}×{image.height}</span>
            {/if}
            {#if image.artist}
              <span class="truncate">{image.artist}</span>
            {/if}
          </div>
        </Card>
      {:else}
        <p class="py-10 text-center text-sm italic text-slate-500">No images found.</p>
      {/each}
    </div>
  {/if}
</div>
