<script lang="ts">
  import { useImages, useSetImageFavorite, useBlacklistImage, useDeleteImage } from '../lib/api/queries';
  import { getApiUrl } from '../lib/runtime';
  import type { Image } from '../lib/api/types';

  const imagesQuery = useImages(() => ({}));

  let favoritesOnly = $state(false);
  let searchQuery = $state('');

  const favoriteMutation = useSetImageFavorite();
  const blacklistMutation = useBlacklistImage();
  const deleteMutation = useDeleteImage();

  async function handleFavorite(image: Image) {
    await favoriteMutation.mutateAsync({
      id: image.id,
      favorite: !image.is_favorite,
    });
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
    const apiBase = getApiUrl();
    return `${apiBase}/v1/images/thumbnail/${imageId}`;
  }

  function getImageUrl(imageId: string): string {
    const apiBase = getApiUrl();
    return `${apiBase}/v1/images/image/${imageId}`;
  }

  // Filter images client-side for simple search/favorites
  let filteredImages = $derived(imagesQuery.data?.items.filter(img => {
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
  }) || []);
</script>

<div class="page">
  <div class="page-header">
    <h1>Images</h1>
    <div class="header-actions">
      <label class="checkbox-label">
        <input type="checkbox" bind:checked={favoritesOnly} />
        Favorites only
      </label>
      <input type="text" placeholder="Search..." bind:value={searchQuery} class="search-input" />
    </div>
  </div>

  {#if imagesQuery.isLoading}
    <p>Loading...</p>
  {:else if imagesQuery.isError}
    <p class="error">Failed to load images</p>
  {:else if imagesQuery.data}
    <div class="masonry">
      {#each filteredImages as image}
        <div class="image-card" class:favorite={image.is_favorite}>
          <div class="image-wrapper">
            <a href={getImageUrl(image.id)} target="_blank" rel="noopener noreferrer">
              <img src={getThumbnailUrl(image.id)} alt="" loading="lazy" />
            </a>
            <div class="overlay">
              <button class="icon-btn" onclick={() => handleFavorite(image)} title={image.is_favorite ? 'Unfavorite' : 'Favorite'}>
                {image.is_favorite ? '★' : '☆'}
              </button>
              <button class="icon-btn danger" onclick={() => handleBlacklist(image)} title="Blacklist">✕</button>
              <button class="icon-btn danger" onclick={() => handleDelete(image)} title="Delete">🗑</button>
            </div>
          </div>
          <div class="image-info">
            {#if image.width && image.height}
              <span class="dimensions">{image.width}×{image.height}</span>
            {/if}
            {#if image.artist}
              <span class="artist">{image.artist}</span>
            {/if}
          </div>
        </div>
      {:else}
        <p class="empty">No images found.</p>
      {/each}
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
    margin-bottom: 1rem;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  h1 {
    margin: 0;
    font-size: 1.5rem;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    cursor: pointer;
  }

  .search-input {
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 0.9rem;
    width: 200px;
  }

  .masonry {
    columns: 2;
    column-gap: 0.75rem;
  }

  @media (min-width: 640px) {
    .masonry {
      columns: 3;
    }
  }

  @media (min-width: 1024px) {
    .masonry {
      columns: 4;
    }
  }

  @media (min-width: 1280px) {
    .masonry {
      columns: 5;
    }
  }

  .image-card {
    break-inside: avoid;
    margin-bottom: 0.75rem;
    background: white;
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .image-card.favorite {
    box-shadow: 0 0 0 2px #f57c00;
  }

  .image-wrapper {
    position: relative;
    background: #f5f5f5;
  }

  .image-wrapper a {
    display: block;
  }

  .image-wrapper img {
    width: 100%;
    height: auto;
    display: block;
  }

  .placeholder {
    padding: 2rem;
    text-align: center;
    color: #888;
    font-size: 0.85rem;
  }

  .overlay {
    position: absolute;
    top: 0.25rem;
    right: 0.25rem;
    display: flex;
    gap: 0.25rem;
    opacity: 0;
    transition: opacity 0.2s;
  }

  .image-card:hover .overlay {
    opacity: 1;
  }

  .icon-btn {
    width: 28px;
    height: 28px;
    border-radius: 4px;
    border: none;
    background: rgba(0, 0, 0, 0.6);
    color: white;
    cursor: pointer;
    font-size: 1rem;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon-btn:hover {
    background: rgba(0, 0, 0, 0.8);
  }

  .icon-btn.danger:hover {
    background: #c62828;
  }

  .image-info {
    padding: 0.5rem;
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: #666;
  }

  .dimensions {
    font-family: monospace;
  }

  .artist {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 60%;
  }

  .empty {
    color: #888;
    font-style: italic;
    text-align: center;
    padding: 3rem;
    column-span: all;
  }

  .error {
    color: #c33;
  }
</style>
