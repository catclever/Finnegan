<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import BabelCardBody from './BabelCardBody.svelte';
  import ChatLogCardBody from './ChatLogCardBody.svelte';
  import NoteCardBody from './NoteCardBody.svelte';
  import ThoughtCardBody from './ThoughtCardBody.svelte';
  import { getCardDisplayTime } from '../../lib/time';
  import { createEventDispatcher } from 'svelte';

  export let cards: CosmicCard[] = [];
  export let isStackDeck: boolean = false;

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

  function handleCardClick(card: CosmicCard) {
    dispatch('cardClick', card);
  }
</script>

{#if cards.length > 0}
<div class="virtual-proxy-wrapper" class:is-stack={isStackDeck}>
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
      on:click={() => handleCardClick(card)}
    >
      <div class="virtual-card-content">
        {#if card.type === 'babel'}
          <BabelCardBody {card} isFocused={true} babelTime={getTime(card)} />
        {:else if card.type === 'chat_log'}
          <ChatLogCardBody {card} isFocused={true} thoughtTime={getTime(card)} />
        {:else if card.type === 'note'}
          <NoteCardBody {card} isFocused={true} thoughtTime={getTime(card)} />
        {:else}
          <ThoughtCardBody {card} isFocused={true} thoughtTime={getTime(card)} />
        {/if}
      </div>

      {#if isStackDeck && hasMultiple && i === activeIndex}
        <div class="deck-controls" on:click|stopPropagation>
          <button on:click={handlePrev}>
            <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none"><polyline points="15 18 9 12 15 6"></polyline></svg>
          </button>
          <span>{activeIndex + 1} / {cards.length}</span>
          <button on:click={handleNext}>
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
    z-index: 1000;
    perspective: 1000px;
  }

  .virtual-card {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border: 1px solid rgba(255, 255, 255, 0.9);
    border-radius: 20px;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.12), 0 0 0 1px rgba(255, 255, 255, 0.6) inset;
    overflow: hidden;
    transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
    display: flex;
    flex-direction: column;
    cursor: pointer;
  }

  .virtual-card:hover {
    transform: translateY(-4px) scale(1.02) !important;
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
    /* Mimic the focused card padding and styles */
    padding: 26px 36px;
    box-sizing: border-box;
    pointer-events: none;
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
