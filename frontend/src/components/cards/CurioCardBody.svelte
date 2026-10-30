<script lang="ts">
  import { createEventDispatcher, onDestroy } from 'svelte';
  import MarkdownContent from './MarkdownContent.svelte';
  import CardImageGallery from './CardImageGallery.svelte';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import type { CosmicCard } from '../../types/card';

  export let card: CosmicCard;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let displayTime: string = "";
  export let thoughtTime: string = "";
  export let babelTime: string = "";
  export let z: number | undefined = undefined;

  const dispatch = createEventDispatcher<{ tagClick: { tag: string; cardId: string } }>();

  let currentIndex = 0;
  let carouselTimer: any;
  $: images = card.images || [];
  $: hasImages = images.length > 0;
  $: resolvedDisplayTime = displayTime || thoughtTime || babelTime;

  function startCarousel() {
    if (!carouselTimer && images.length > 1) {
      carouselTimer = setInterval(() => {
        currentIndex = (currentIndex + 1) % images.length;
      }, 3000);
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

  onDestroy(() => stopCarousel());

  function handleTagClick(tag: string) {
    dispatch('tagClick', { tag, cardId: card.id });
  }

  function openPath(path?: string) {
    if (path) BrowserOpenURL(path);
  }

  const intentColors: Record<string, string> = {
    wish: '#10b981',
    someday: '#f59e0b',
    archive: '#64748b',
    acquired: 'transparent'
  };
</script>

<div class="curio-card-body" class:focused={isFocused}>
  {#if !isFocused}
    <div class="scattering-top-row">
      {#if card.platform}<span class="badge platform-badge">{card.platform}</span>{/if}
      {#if card.price}<span class="price">{card.price}</span>{/if}
      {#if card.intent}
        <span class="badge intent-badge" class:acquired={card.intent === 'acquired'} style="background-color: {card.intent !== 'acquired' ? intentColors[card.intent] : 'transparent'}">
          {card.intent}
        </span>
      {/if}
      {#if card.availability}
        <span class="badge availability-badge">{card.availability}</span>
      {/if}
    </div>

    {#if hasImages}
      <div class="image-stage" style="height: 92px;">
        <div class="album-glow-backdrop">
          {#each images as img, i}
            <img src={img} alt="" class="backdrop-img" class:active={currentIndex === i} />
          {/each}
        </div>
        <div class="stage-viewport" style="height: 92px;">
          {#each images as img, i}
            <img src={img} alt="" class="stage-img" class:active={currentIndex === i} style="object-fit: contain;" />
          {/each}
        </div>
        {#if images.length > 1}
          <div class="indicator-dots">
            {#each images as _, i}
              <button class="indicator-dot" class:active={currentIndex === i} on:click|stopPropagation={() => currentIndex = i} />
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <div class="scattering-bottom">
      {#if card.title}<div class="curio-title">{card.title}</div>{/if}
      {#if card.content}
        <div class="content-fade">
          <MarkdownContent content={card.content} isFocused={false} />
        </div>
      {/if}
      {#if card.tags && card.tags.length > 0}
        <div class="tags">
          {#each card.tags as tag}
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <span class="tag" on:click|stopPropagation={() => handleTagClick(tag)}>#{tag}</span>
          {/each}
        </div>
      {/if}
      {#if resolvedDisplayTime}<div class="time">{resolvedDisplayTime}</div>{/if}
    </div>

  {:else}
    <div class="focused-header">
      <div class="meta-row">
        {#if card.platform}<span class="badge platform-badge">{card.platform}</span>{/if}
        {#if card.availability}<span class="badge availability-badge">{card.availability}</span>{/if}
        {#if card.intent}
          <span class="badge intent-badge" class:acquired={card.intent === 'acquired'} style="background-color: {card.intent !== 'acquired' ? intentColors[card.intent] : 'transparent'}">
            {card.intent}
          </span>
        {/if}
      </div>
      {#if card.title}<h2 class="specimen-title">{card.title}</h2>{/if}
      {#if card.brand}<div class="brand">{card.brand}</div>{/if}
      {#if card.price}<div class="price large">{card.price}</div>{/if}
      {#if resolvedDisplayTime}<div class="time">{resolvedDisplayTime}</div>{/if}
    </div>

    {#if hasImages}
      <CardImageGallery {images} />
    {/if}

    <div class="archive-action-bar">
      {#if card.snapshotPath}
        <button on:click={() => openPath(card.snapshotPath)}>⚡ 打开本地网页快照</button>
      {/if}
      {#if card.screenshotPath}
        <button on:click={() => openPath(card.screenshotPath)}>🖼️ 查看长截图</button>
      {/if}
      {#if card.sourceUrl}
        <button on:click={() => openPath(card.sourceUrl)}>↗ 原始链接</button>
      {/if}
    </div>

    {#if card.content}
      <div class="markdown-notes">
        <MarkdownContent content={card.content} isFocused={true} />
      </div>
    {/if}

    {#if card.tags && card.tags.length > 0}
      <div class="tags focused">
        {#each card.tags as tag}
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <span class="tag" on:click|stopPropagation={() => handleTagClick(tag)}>#{tag}</span>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .curio-card-body {
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

  .curio-card-body.focused {
    flex: 1;
    min-height: 0;
    overflow-y: auto !important;
    overflow-x: hidden !important;
    scrollbar-width: none;
  }
  .curio-card-body.focused::-webkit-scrollbar { display: none; }

  .scattering-top-row, .meta-row {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-bottom: 6px;
    align-items: center;
  }

  .badge {
    font-size: 10px;
    padding: 2px 6px;
    border-radius: 4px;
    color: white;
    font-weight: 500;
  }
  .platform-badge { background: #3b82f6; }
  .availability-badge { background: #ef4444; }
  .intent-badge.acquired {
    border: 1px solid #10b981;
    color: #10b981;
  }

  .price {
    font-size: 11px;
    font-weight: 600;
    color: #059669;
  }
  .price.large { font-size: 16px; margin: 4px 0; }

  .image-stage {
    position: relative;
    width: 100%;
    border-radius: 10px;
    background: rgba(0,0,0,0.03);
    margin-bottom: 8px;
    overflow: hidden;
  }
  .album-glow-backdrop {
    position: absolute;
    inset: -12px;
    filter: blur(32px);
    opacity: 0.68;
    z-index: 1;
  }
  .backdrop-img, .stage-img {
    position: absolute;
    top: 0; left: 0;
    width: 100%; height: 100%;
    opacity: 0;
    transition: opacity 0.6s ease;
  }
  .backdrop-img { object-fit: cover; transform: scale(1.2); }
  .backdrop-img.active, .stage-img.active { opacity: 1; }
  .stage-viewport { position: relative; z-index: 2; }

  .indicator-dots {
    position: absolute;
    bottom: 5px; left: 50%;
    transform: translateX(-50%);
    display: flex; gap: 4px;
    z-index: 10;
  }
  .indicator-dot {
    width: 4px; height: 4px;
    border-radius: 50%;
    background: rgba(255,255,255,0.45);
    border: none; padding: 0;
  }
  .indicator-dot.active { background: #fff; transform: scale(1.35); }

  .curio-title {
    font-weight: 600;
    font-size: 12px;
    margin-bottom: 4px;
  }
  .specimen-title {
    font-size: 18px;
    font-weight: 700;
    margin: 4px 0;
  }
  .brand {
    font-size: 12px;
    color: #64748b;
  }

  .content-fade {
    mask-image: linear-gradient(to bottom, black 60%, transparent 100%);
    -webkit-mask-image: linear-gradient(to bottom, black 60%, transparent 100%);
    max-height: 40px;
    overflow: hidden;
  }

  .archive-action-bar {
    display: flex;
    gap: 8px;
    margin: 12px 0;
    flex-wrap: wrap;
  }
  .archive-action-bar button {
    background: rgba(255,255,255,0.8);
    border: 1px solid rgba(0,0,0,0.1);
    border-radius: 6px;
    padding: 4px 8px;
    font-size: 12px;
    cursor: pointer;
    backdrop-filter: blur(4px);
  }
  .archive-action-bar button:hover { background: #fff; }

  .tags {
    display: flex; flex-wrap: wrap; gap: 4px; margin-top: 4px;
  }
  .tag {
    font-size: 10px;
    color: #0284c7;
    background: rgba(56,189,248,0.12);
    padding: 1px 6px;
    border-radius: 4px;
    cursor: pointer;
  }
  .tag:hover { background: rgba(56,189,248,0.24); }

  .time {
    font-size: 10px;
    color: rgba(0,0,0,0.42);
    margin-top: auto;
    text-align: right;
  }
</style>
