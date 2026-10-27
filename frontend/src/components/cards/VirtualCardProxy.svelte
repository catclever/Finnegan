<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import { getCardComponent } from './cardRegistry';
  import { getCardDisplayTime } from '../../lib/time';
  import { createEventDispatcher } from 'svelte';

  export let cards: CosmicCard[] = [];
  export let isStackDeck: boolean = false;
  export let isOnTop: boolean = false;
  export let scrollExpansion: number = 0;
  export let globalTilt: { rx: number, ry: number } = { rx: 0, ry: 0 };
  export let isFocusedMode: boolean = false;

  const dispatch = createEventDispatcher();
  let activeIndex = 0;

  $: {
    // Reset activeIndex when cards array changes
    if (cards.length > 0 && activeIndex >= cards.length) {
      activeIndex = 0;
    }
  }

  function handleNext(e: MouseEvent) {
    e.stopPropagation();
    if (cards.length > 0) activeIndex = (activeIndex + 1) % cards.length;
  }

  function handlePrev(e: MouseEvent) {
    e.stopPropagation();
    if (cards.length > 0) activeIndex = (activeIndex - 1 + cards.length) % cards.length;
  }

  $: hasMultiple = cards.length > 1;

  function getTime(c: any) {
    return getCardDisplayTime(c);
  }

  function handleCardClick(e: MouseEvent, card: CosmicCard) {
    e.stopPropagation();
    dispatch('cardClick', card);
  }

  function handleCardDblClick(e: MouseEvent, card: CosmicCard) {
    e.stopPropagation();
    dispatch('cardDblClick', card);
  }
</script>

{#if cards.length > 0}
<div 
  class="virtual-proxy-wrapper" 
  class:is-stack={isStackDeck}
  class:is-on-top={isOnTop}
  style="
    transform: translateY({scrollExpansion * 300}px) 
               translateZ({isFocusedMode ? (isOnTop ? 125 : 116) : 125}px)
               {isFocusedMode ? '' : `rotateX(${-globalTilt.rx}deg) rotateY(${-globalTilt.ry}deg)`};
    z-index: {isFocusedMode ? (isOnTop ? 1005 : 995) : 1005};
  "
>
  {#each cards as card, i}
    {@const zIndex = cards.length - i}
    {@const offset = i * 8}
    {@const opacity = i === 0 ? 1 : Math.max(0, 1 - i * 0.25)}
    {@const scale = i === 0 ? 1 : Math.max(0.8, 1 - i * 0.05)}

    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div
      class="virtual-card"
      class:active={i === activeIndex}
      class:hidden={!isStackDeck && i !== activeIndex}
      style="z-index: {isStackDeck ? (i === activeIndex ? 10 : zIndex) : 1};
             transform: {isStackDeck && i !== activeIndex ? `translate(${offset}px, ${offset}px) scale(${scale})` : 'none'};
             opacity: {isStackDeck && i !== activeIndex ? opacity : (i === activeIndex ? 1 : 0)};"
      on:click={(e) => handleCardClick(e, card)}
      on:dblclick={(e) => handleCardDblClick(e, card)}
    >
      <div class="virtual-card-content">
        <svelte:component
          this={getCardComponent(card.type)}
          {card}
          isFocused={true}
          isHovered={false}
          thoughtTime={getTime(card)}
          babelTime={getTime(card)}
          displayTime={getTime(card)}
        />
      </div>

      {#if isStackDeck && hasMultiple && i === activeIndex}
        <div class="deck-controls" on:click|stopPropagation>
          <button on:click={handlePrev} title="上一张">
            <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none"><polyline points="15 18 9 12 15 6"></polyline></svg>
          </button>
          <span>{activeIndex + 1} / {cards.length}</span>
          <button on:click={handleNext} title="下一张">
            <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none"><polyline points="9 18 15 12 9 6"></polyline></svg>
          </button>
        </div>
      {/if}
    </div>
  {/each}
</div>
{/if}

<style>
  .virtual-proxy-wrapper {
    position: absolute;
    top: 40px;
    right: 40px;
    width: 320px;
    height: 320px;
    pointer-events: auto;
    perspective: 1000px;
    transform-style: preserve-3d;
    transition: box-shadow 0.28s ease;
  }

  .virtual-card {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(255, 255, 255, 0.96);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.9);
    border-radius: 20px;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.12), 0 0 0 1px rgba(255, 255, 255, 0.6) inset;
    overflow: hidden;
    transition: all 0.28s cubic-bezier(0.16, 1, 0.3, 1);
    display: flex;
    flex-direction: column;
    cursor: pointer;
    pointer-events: auto;
    color: #1e293b;
    --card-text-color: #1e293b;
    --card-text-secondary: rgba(60, 60, 60, 0.75);
  }

  .virtual-proxy-wrapper.is-on-top .virtual-card {
    box-shadow: 0 24px 60px rgba(0, 0, 0, 0.18), 0 0 0 1.5px rgba(245, 158, 11, 0.5) inset;
  }

  .virtual-card:hover {
    transform: translateY(-3px) scale(1.015) !important;
    box-shadow: 0 20px 48px rgba(0, 0, 0, 0.16), 0 0 0 1px rgba(255, 255, 255, 0.8) inset;
  }

  .virtual-card.hidden {
    display: none;
    opacity: 0;
    pointer-events: none;
  }

  .virtual-card-content {
    flex: 1;
    width: 420px;
    height: 420px;
    transform: scale(0.7619); /* 320 / 420 ≈ 0.7619 */
    transform-origin: top left;
    display: flex;
    flex-direction: column;
    padding: 26px 36px;
    box-sizing: border-box;
    pointer-events: auto;
    color: #1e293b;
    overflow-y: auto;
    overflow-x: hidden;
    scrollbar-width: none;
    -ms-overflow-style: none;
  }

  .virtual-card-content::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  :global(.virtual-card-content .gallery-image) {
    max-height: 180px;
    object-fit: contain;
    width: auto;
    margin: 0 auto;
  }

  .deck-controls {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 16px;
    background: rgba(30, 40, 50, 0.75);
    color: white;
    padding: 6px 16px;
    border-radius: 20px;
    font-size: 13px;
    font-weight: 500;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    z-index: 20;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
    pointer-events: auto;
  }

  .deck-controls button {
    background: none;
    border: none;
    color: white;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 4px;
    border-radius: 50%;
    transition: all 0.2s ease;
  }

  .deck-controls button:hover {
    color: #f59e0b;
    background: rgba(255, 255, 255, 0.15);
  }
</style>
