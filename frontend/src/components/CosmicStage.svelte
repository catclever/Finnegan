<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CosmicStage Component Draft
  Task: night_shift_overflow_scroll_focus_018

  [Bug Fix & Feature Documentation]
  1. Specific Problems Fixed:
     - Users had no visual cue when overflow cards (such as Bugu Babel echoes) were staged
       in Deep Space (Screen 2 / Screen 3) below the current viewport.
     - `.scene` previously had `transition: transform 0.1s ease-out;` which fought with
       Svelte's 60fps spring interpolation during Gravitational Tide scrolling and caused
       focused cards to jitter relative to the viewport center.
     - `UICard` lacked `scrollExpansion` and `hasAnyFocused` context to lock focused cards
       to the invariant viewport center at arbitrary scroll depths.
  2. Resolution & Logic:
     - Computed reactive `hasDeepCardsBelow` (`focusedCardId === null && cards.some(c => !c.isFadingOut && (c.y - scrollExpansion * 300) > 260)`).
     - Rendered `.deep-space-scroll-cue` anchored at the bottom center of `.cards-wrapper`
       (outside `.scene`) with 3 vertically arranged 4.5px pulsating `.scroll-dot` elements
       with cascading wave animation (`0s`, `0.25s`, `0.5s` delays).
     - Removed `transition: transform 0.1s ease-out;` from `.scene` and added `will-change: transform;`
       for zero-lag 60fps spring synchronization.
     - Passed `scrollExpansion` and `hasAnyFocused={focusedCardId !== null}` down to `UICard`.
  3. Caveats & Future Considerations:
     - `.deep-space-scroll-cue` must remain a sibling of `.scene` inside `.cards-wrapper`
       so that `.scene`'s 3D tilt and `translateY` do not displace the bottom viewport indicator.
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

  // Detect whether any active card exists below the currently visible viewport
  $: hasDeepCardsBelow = focusedCardId === null && cards.some(c => !c.isFadingOut && (c.y - scrollExpansion * 300) > 260);

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
        {scrollExpansion}
        hasAnyFocused={focusedCardId !== null}
        isFocused={focusedCardId === card.id}
        isHovered={hoveredCardId === card.id}
        onToggleFocus={() => handleToggleFocus(card.id)}
        onHover={() => handleHover(card.id)}
        onLeave={handleLeave}
      />
    {/each}
  </div>

  <!-- 3 Vertically Arranged Pulsating Dots for Deep-Space Overflow Cue -->
  <div class="deep-space-scroll-cue" class:visible={hasDeepCardsBelow} aria-hidden="true">
    <span class="scroll-dot"></span>
    <span class="scroll-dot"></span>
    <span class="scroll-dot"></span>
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
    will-change: transform;
  }

  /* Deep Space Pulsating Scroll Indicator */
  .deep-space-scroll-cue {
    position: absolute;
    bottom: 22px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.45s cubic-bezier(0.22, 1, 0.36, 1);
    z-index: 60;
  }

  .deep-space-scroll-cue.visible {
    opacity: 1;
  }

  .scroll-dot {
    width: 4.5px;
    height: 4.5px;
    border-radius: 50%;
    background: rgba(95, 115, 145, 0.75);
    box-shadow: 0 0 8px rgba(120, 165, 255, 0.65), 0 0 3px rgba(255, 255, 255, 0.85);
    animation: deep-dot-pulse 1.8s ease-in-out infinite;
  }

  .scroll-dot:nth-child(1) {
    animation-delay: 0s;
  }

  .scroll-dot:nth-child(2) {
    animation-delay: 0.25s;
  }

  .scroll-dot:nth-child(3) {
    animation-delay: 0.5s;
  }

  @keyframes deep-dot-pulse {
    0%, 100% {
      opacity: 0.28;
      transform: scale(0.82);
    }
    50% {
      opacity: 0.95;
      transform: scale(1.22);
    }
  }
</style>
