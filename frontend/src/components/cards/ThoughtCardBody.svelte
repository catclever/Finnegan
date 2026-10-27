<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - ThoughtCardBody Subcomponent
  Task: thought_images_scattering_029
  
  Renders the body of thought cards in Scattering state (200x190) and focused state.
  Features:
  - Formats asset URLs via /api/assets/<name> if not data: or http(s)
  - Album Glow backdrop with 32px Gaussian blur behind contained images
  - 82px height container with object-fit: contain (zero cropping)
  - Auto-cycling carousel every 3500ms with smooth crossfade when images > 1
  - Carousel pauses when isHovered is true
  - Tiny indicator dots at bottom of the 82px stage for multi-image cards
  ======================================================================================
-->
<script lang="ts">
  import { onDestroy } from 'svelte';
  import { getCardDisplayTime } from '../../lib/time';
  import CardImageGallery from './CardImageGallery.svelte';
  import MarkdownContent from './MarkdownContent.svelte';

  export let card: any;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let thoughtTime: string = '';
  export let babelTime: string = '';
  export let displayTime: string = '';

  $: resolvedDisplayTime = displayTime || thoughtTime || babelTime || getCardDisplayTime(card);

  // Extract only manual keywords that have '*' provenance marker (rendered without '*')
  $: manualKeywords = (() => {
    if (!card) return [];
    const list: string[] = [];
    const seen = new Set<string>();

    const kwSource = card.keywords || (card as any).Keywords;
    if (kwSource && typeof kwSource === 'object') {
      for (const values of Object.values(kwSource)) {
        if (Array.isArray(values)) {
          for (const item of values) {
            if (typeof item === 'string' && item.endsWith('*')) {
              const clean = item.replace(/\*+$/, '').trim();
              if (clean && !seen.has(clean)) {
                seen.add(clean);
                list.push(clean);
              }
            }
          }
        }
      }
    }
    return list;
  })();

  export function formatAssetUrl(url: string): string {
    if (!url) return '';
    if (url.startsWith('data:') || url.startsWith('http://') || url.startsWith('https://')) {
      return url;
    }
    if (url.startsWith('/api/assets/')) {
      return url;
    }
    const cleanName = url.startsWith('/') ? url.slice(1) : url;
    return `/api/assets/${cleanName}`;
  }

  $: rawImages = card?.images || (card as any)?.Images || [];
  $: images = (Array.isArray(rawImages) ? rawImages : [rawImages])
    .map(item => {
      if (typeof item === 'string') return item;
      if (item && typeof item === 'object') {
        return item.url || item.data || item.path || '';
      }
      return '';
    })
    .filter((url: string) => typeof url === 'string' && url.trim().length > 0)
    .map(formatAssetUrl);

  $: hasImages = images.length > 0;

  let currentIndex: number = 0;
  let carouselTimer: ReturnType<typeof setInterval> | null = null;

  function startCarousel() {
    if (carouselTimer) return;
    if (images.length > 1 && !isHovered && !isFocused) {
      carouselTimer = setInterval(() => {
        currentIndex = (currentIndex + 1) % images.length;
      }, 3500);
    }
  }

  function stopCarousel() {
    if (carouselTimer) {
      clearInterval(carouselTimer);
      carouselTimer = null;
    }
  }

  $: if (images.length > 1 && !isHovered && !isFocused) {
    startCarousel();
  } else {
    stopCarousel();
  }

  $: if (currentIndex >= images.length) {
    currentIndex = 0;
  }

  onDestroy(() => {
    stopCarousel();
  });

  function handleImageError(e: Event) {
    const el = e.currentTarget as HTMLElement | null;
    if (el) el.style.display = 'none';
  }
</script>

<div class="thought-card-body" class:focused={isFocused} class:has-images={hasImages}>
  {#if isFocused}
    {#if resolvedDisplayTime}
      <div class="thought-header">
        <span class="card-time">{resolvedDisplayTime}</span>
      </div>
    {/if}

    <!-- Focused mode: Invariant Focus natural scroll and horizontal swipe (Option A: Text first, Images below) -->
    {#if card && card.content}
      <div class="thought-content focused" class:has-images={hasImages}>
        <MarkdownContent content={card.content} isFocused={true} />
      </div>
    {/if}

    {#if manualKeywords.length > 0}
      <div class="thought-tags focused">
        {#each manualKeywords as tag}
          <span class="thought-tag focused">#{tag}</span>
        {/each}
      </div>
    {/if}

    {#if hasImages}
      <CardImageGallery {images} />
    {/if}
  {:else}
    <!-- Scattering mode (200x190): 82px stage with Album Glow backdrop and auto-cycling carousel -->
    {#if hasImages}
      <div 
        class="thought-image-stage image-stage image-container" 
        style="height: 82px;"
      >
        <!-- Album Glow Backdrop with 32px Gaussian blur -->
        <div 
          class="album-glow-backdrop" 
          style="filter: blur(32px); -webkit-filter: blur(32px);"
          aria-hidden="true"
        >
          {#each images as img, i (img + i)}
            <img
              src={img}
              alt=""
              class="backdrop-img"
              class:active={currentIndex === i}
              on:error={handleImageError}
            />
          {/each}
        </div>

        <!-- 82px height container with object-fit: contain (zero cropping) -->
        <div class="stage-viewport" style="height: 82px;">
          {#each images as img, i (img + i)}
            <img
              src={img}
              alt="Thought attachment {i + 1}"
              class="stage-img"
              class:active={currentIndex === i}
              style="object-fit: contain;"
              on:error={handleImageError}
            />
          {/each}
        </div>

        <!-- Tiny indicator dots at bottom of the 82px stage -->
        {#if images.length > 1}
          <div class="indicator-dots" role="tablist" aria-label="Image indicators">
            {#each images as _, i}
              <button
                type="button"
                class="indicator-dot dot"
                class:active={currentIndex === i}
                on:click|stopPropagation={() => { currentIndex = i; }}
                aria-label="Image {i + 1} of {images.length}"
              />
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    {#if card && card.content}
      <div class="thought-content" class:has-images={hasImages}>
        <MarkdownContent content={card.content} isFocused={false} />
      </div>
    {/if}

    {#if manualKeywords.length > 0}
      <div class="thought-tags">
        {#each manualKeywords as tag}
          <span class="thought-tag">#{tag}</span>
        {/each}
      </div>
    {/if}

    {#if resolvedDisplayTime}
      <div class="thought-footer">
        <span class="card-time">{resolvedDisplayTime}</span>
      </div>
    {/if}
  {/if}
</div>

<style>
  .thought-card-body {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    position: relative;
    z-index: 1;
    overflow: hidden;
    text-align: left;
    color: var(--card-text-color, #1e293b);
  }

  .thought-card-body.focused {
    flex: 1;
    min-height: 0;
    height: 100%;
    max-height: 100%;
    overflow-y: auto !important;
    overflow-x: hidden !important;
    overscroll-behavior: contain;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
    -ms-overflow-style: none;
  }

  .thought-card-body.focused::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  .thought-header {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 6px;
    width: 100%;
    flex-shrink: 0;
  }

  .thought-footer {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    width: 100%;
    margin-top: auto;
    padding-top: 4px;
    flex-shrink: 0;
    position: relative;
    z-index: 2;
  }

  .thought-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 4px;
    margin-bottom: 2px;
  }

  .thought-tags.focused {
    margin-top: 8px;
    margin-bottom: 8px;
    gap: 6px;
  }

  .thought-tag {
    font-size: 10px;
    font-weight: 500;
    color: #0284c7;
    background: rgba(56, 189, 248, 0.12);
    padding: 1px 6px;
    border-radius: 4px;
    letter-spacing: 0.02em;
    user-select: none;
    white-space: nowrap;
  }

  .thought-tag.focused {
    font-size: 12px;
    padding: 2px 8px;
    border-radius: 6px;
  }

  .card-time {
    font-size: 0.7rem;
    color: rgba(0, 0, 0, 0.42);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.02em;
    user-select: none;
    white-space: nowrap;
  }

  /* 82px height container for image stage */
  .thought-image-stage {
    position: relative;
    width: 100%;
    height: 82px;
    border-radius: 10px;
    overflow: hidden;
    background: rgba(0, 0, 0, 0.03);
    flex-shrink: 0;
    margin-bottom: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  /* Album Glow Backdrop with 32px Gaussian blur */
  .album-glow-backdrop {
    position: absolute;
    inset: -12px;
    filter: blur(32px);
    -webkit-filter: blur(32px);
    opacity: 0.68;
    pointer-events: none;
    z-index: 1;
    overflow: hidden;
    border-radius: 12px;
  }

  .backdrop-img {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    transform: scale(1.2);
    opacity: 0;
    transition: opacity 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .backdrop-img.active {
    opacity: 1;
  }

  /* Viewport and contained image */
  .stage-viewport {
    position: relative;
    width: 100%;
    height: 82px;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2;
  }

  .stage-img {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 82px;
    max-height: 82px;
    object-fit: contain;
    opacity: 0;
    transition: opacity 0.6s cubic-bezier(0.4, 0, 0.2, 1);
    pointer-events: none;
    user-select: none;
    -webkit-user-drag: none;
  }

  .stage-img.active {
    opacity: 1;
    pointer-events: auto;
  }

  /* Tiny indicator dots at bottom of the 82px stage */
  .indicator-dots {
    position: absolute;
    bottom: 5px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 4px;
    z-index: 10;
    pointer-events: auto;
  }

  .indicator-dot {
    width: 4px;
    height: 4px;
    border-radius: 50%;
    border: none;
    padding: 0;
    background: rgba(255, 255, 255, 0.45);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.35);
    cursor: pointer;
    transition: all 0.25s ease;
  }

  .indicator-dot.active {
    background: #ffffff;
    transform: scale(1.35);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.55);
  }

  /* Thought Content (Text) */
  .thought-content {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    width: 100%;
    text-align: left;
    position: relative;
    z-index: 1;
  }

  .thought-content:not(.has-images) {
    justify-content: center;
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .thought-content.has-images {
    justify-content: flex-start;
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }


  /* Focused State Styles */
  .thought-content.focused {
    display: block;
    justify-content: flex-start;
    flex: none;
    min-height: 0;
    height: auto;
    max-height: none;
    overflow: visible !important;
    text-align: left;
    color: var(--card-text-color, #1e293b);
  }

  .thought-content.focused.has-images {
    margin-bottom: 12px;
  }

</style>
