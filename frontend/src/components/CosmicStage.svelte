<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CosmicStage Component
  Task: frontend_components_split_012
  
  Encapsulates:
  - 3D perspective viewport container (.cards-wrapper) with dynamic elevation
  - Dynamic transform scene (.scene) responding to global tilt & tidal expansion
  - UICard instance iteration and lifecycle bindings
  - Focus toggling, hovering, and background click handling
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import UICard from '../UICard.svelte';
  import type { CosmicCard } from '../types/card';
  import BuguRayNetwork from '../modules/bugu/BuguRayNetwork.svelte';
  import type { BuguBeacon, BuguSession } from '../modules/bugu/types';

  // Props
  export let cards: CosmicCard[] = [];
  export let time: number = 0;
  export let globalTilt: { rx: number; ry: number } = { rx: 0, ry: 0 };
  export let scrollExpansion: number = 0;
  export let focusedCardId: string | null = null;
  export let hoveredCardId: string | null = null;
  export let isOmnibarFocused: boolean = false;
  export let buguSession: BuguSession | null = null;
  export let buguBeacon: BuguBeacon | null = null;

  // Callbacks
  export let onToggleFocus: ((cardId: string) => void) | undefined = undefined;
  export let onHoverCard: ((cardId: string) => void) | undefined = undefined;
  export let onLeaveCard: (() => void) | undefined = undefined;
  export let onBgClick: ((e: MouseEvent) => void) | undefined = undefined;

  const dispatch = createEventDispatcher();

  function handleToggleFocus(id: string) {
    if (onToggleFocus) onToggleFocus(id);
    dispatch('toggleFocus', id);
  }

  function handleHover(id: string) {
    if (onHoverCard) onHoverCard(id);
    dispatch('hoverCard', id);
  }

  function handleLeave() {
    if (onLeaveCard) onLeaveCard();
    dispatch('leaveCard');
  }

  function handleWrapperClick(e: MouseEvent) {
    if (onBgClick) onBgClick(e);
    dispatch('bgClick', e);
  }
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div 
  class="cards-wrapper" 
  class:elevated={(hoveredCardId !== null && !isOmnibarFocused) || focusedCardId !== null} 
  on:click={handleWrapperClick}
>
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div 
    class="scene" 
    style="
      transform: rotateX({globalTilt.rx}deg) rotateY({globalTilt.ry}deg) translateY({-scrollExpansion * 300}px); 
      transform-style: preserve-3d;
    "
  >
    <BuguRayNetwork {cards} beacon={buguBeacon} session={buguSession} />

    {#each cards as card (card.id)}
      <UICard 
        {card}
        {time}
        {globalTilt}
        isFocused={focusedCardId === card.id}
        isHovered={hoveredCardId === card.id}
        onToggleFocus={() => handleToggleFocus(card.id)}
        onHover={() => handleHover(card.id)}
        onLeave={handleLeave}
      />
    {/each}
  </div>
</div>

<style>
  /* Cards wrapper & 3D Scene */
  .cards-wrapper {
    position: absolute;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    perspective: 1000px;
    pointer-events: none;
    z-index: 5;
    -webkit-mask-image: radial-gradient(ellipse at center, black 80%, rgba(0, 0, 0, 0.4) 93%, transparent 100%);
    mask-image: radial-gradient(ellipse at center, black 80%, rgba(0, 0, 0, 0.4) 93%, transparent 100%);
    transition: -webkit-mask-image 0.25s ease, mask-image 0.25s ease;
  }

  .cards-wrapper.elevated {
    z-index: 100; /* Elevated above omnibar-container (z-index 50) */
    -webkit-mask-image: none;
    mask-image: none;
  }

  .scene {
    pointer-events: none;
    width: 100%;
    height: 100%;
    position: relative;
    transform-style: preserve-3d;
    transition: transform 0.1s ease-out;
  }
</style>
