<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - AttachmentTray Component
  Task: thought_images_omnibar_028
  
  Displays attached thought images as 52x52px rounded thumbnails with hover removal cross
  and active deletion highlight for Backspace keyboard removal.
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let images: string[] = [];
  export let deleteHighlightIndex: number = -1;
  export let highlightIndex: number = -1;
  export let onRemove: ((index: number) => void) | undefined = undefined;

  const dispatch = createEventDispatcher();

  $: activeHighlight = deleteHighlightIndex >= 0 ? deleteHighlightIndex : highlightIndex;

  function handleRemove(index: number) {
    if (onRemove) {
      onRemove(index);
    }
    dispatch('remove', index);
  }
</script>

{#if images && images.length > 0}
  <div class="attachment-tray" role="list" aria-label="Attached images">
    {#each images as img, i (i)}
      <div 
        class="thumbnail-wrapper" 
        class:delete-highlight={activeHighlight === i}
        class:active-delete={activeHighlight === i}
        class:highlighted={activeHighlight === i}
        role="listitem"
      >
        <img src={img} alt="Attachment {i + 1}" class="thumbnail" />
        <button 
          type="button" 
          class="remove-btn" 
          on:click|stopPropagation={() => handleRemove(i)} 
          title="Remove attachment" 
          aria-label="Remove attachment {i + 1}"
        >
          <span class="cross-icon">✕</span>
        </button>
      </div>
    {/each}
  </div>
{/if}

<style>
  .attachment-tray {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding-top: 6px;
    padding-bottom: 2px;
    box-sizing: border-box;
  }

  .thumbnail-wrapper {
    position: relative;
    width: 52px;
    height: 52px;
    border-radius: 10px;
    overflow: hidden;
    flex-shrink: 0;
    box-sizing: border-box;
    border: 1px solid rgba(0, 0, 0, 0.08);
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
    background: rgba(0, 0, 0, 0.04);
    transition: transform 0.15s ease, box-shadow 0.15s ease, outline 0.15s ease, filter 0.15s ease;
  }

  .thumbnail {
    width: 52px;
    height: 52px;
    object-fit: cover;
    border-radius: 10px;
    display: block;
    user-select: none;
    -webkit-user-drag: none;
  }

  .remove-btn {
    position: absolute;
    top: 3px;
    right: 3px;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.65);
    color: #ffffff;
    border: none;
    padding: 0;
    margin: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s ease, background 0.15s ease, transform 0.1s ease;
    font-size: 10px;
    line-height: 1;
    z-index: 2;
  }

  .thumbnail-wrapper:hover .remove-btn {
    opacity: 1;
  }

  .remove-btn:hover {
    background: rgba(220, 38, 38, 0.95);
    transform: scale(1.1);
  }

  .cross-icon {
    display: inline-block;
    font-size: 10px;
    font-weight: 700;
  }

  /* Active delete highlight */
  .thumbnail-wrapper.delete-highlight,
  .thumbnail-wrapper.active-delete,
  .thumbnail-wrapper.highlighted {
    outline: 2px solid #ef4444;
    box-shadow: 0 0 0 2px #ef4444, 0 4px 12px rgba(239, 68, 68, 0.35);
    filter: brightness(0.85);
    transform: scale(0.96);
  }
</style>
