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
  import { createEventDispatcher, onDestroy } from 'svelte';
  import UICard from '../UICard.svelte';
  import type { CosmicCard } from '../types/card';
  import BuguRayNetwork from '../modules/bugu/BuguRayNetwork.svelte';
  import type { BuguBeacon, BuguSession } from '../modules/bugu/types';
  import { getMaxScrollExpansion } from '../lib/stageDecay';

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
  export let lastScrollDirection: 'down' | 'up' = 'down';

  // Callbacks
  export let onToggleFocus: ((cardId: string) => void) | undefined = undefined;
  export let onHoverCard: ((cardId: string) => void) | undefined = undefined;
  export let onLeaveCard: (() => void) | undefined = undefined;
  export let onBgClick: ((e: MouseEvent) => void) | undefined = undefined;

  const dispatch = createEventDispatcher();

  let activeCueDirection: 'down' | 'up' = 'down';
  let screen1DwellTimer: any = null;
  let prevScrollDir: 'down' | 'up' = 'down';

  // Track max scroll depth and boundaries
  $: maxScroll = getMaxScrollExpansion(cards);
  $: isNearBottom = scrollExpansion >= (maxScroll - 0.22);
  $: isNearTop = scrollExpansion <= 0.08;

  // Requirement 1 & 2:
  // - On initial load or staying on Screen 1 for a while, default to appearing at the bottom ('down')
  // - Outside Screen 1 (scrollExpansion > 0.08), follow lastScrollDirection
  $: {
    if (lastScrollDirection !== prevScrollDir) {
      prevScrollDir = lastScrollDirection;
      if (scrollExpansion <= 0.08 && lastScrollDirection === 'up') {
        activeCueDirection = 'up';
        if (screen1DwellTimer) clearTimeout(screen1DwellTimer);
        screen1DwellTimer = setTimeout(() => {
          if (scrollExpansion <= 0.08) {
            activeCueDirection = 'down';
          }
          screen1DwellTimer = null;
        }, 1200);
      }
    }

    if (scrollExpansion <= 0.08) {
      if (lastScrollDirection === 'up') {
        if (!screen1DwellTimer && activeCueDirection !== 'down') {
          screen1DwellTimer = setTimeout(() => {
            if (scrollExpansion <= 0.08) {
              activeCueDirection = 'down';
            }
            screen1DwellTimer = null;
          }, 1200);
        }
      } else {
        activeCueDirection = 'down';
        if (screen1DwellTimer) { clearTimeout(screen1DwellTimer); screen1DwellTimer = null; }
      }
    } else {
      if (screen1DwellTimer) { clearTimeout(screen1DwellTimer); screen1DwellTimer = null; }
      activeCueDirection = lastScrollDirection;
    }
  }

  $: hasCardsBelow = cards.some(c => !c.isFadingOut && (c.y - scrollExpansion * 300) > 280);
  $: hasCardsAbove = scrollExpansion > 0.15;

  // Requirement 3:
  // - Reached bottom: if lastScrollDirection was 'down', do not show ANY cue
  // - Reached top: if lastScrollDirection was 'up', do not show ANY cue (until dwell timer flips it to 'down')
  $: isCueVisible = (() => {
    if (focusedCardId !== null) return false;
    if (activeCueDirection === 'down') {
      if (isNearBottom && lastScrollDirection === 'down') return false;
      return hasCardsBelow;
    } else {
      if (isNearTop && lastScrollDirection === 'up') return false;
      return hasCardsAbove;
    }
  })();

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

  onDestroy(() => {
    if (screen1DwellTimer) clearTimeout(screen1DwellTimer);
  });
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
    <BuguRayNetwork {cards} beacon={buguBeacon} session={buguSession} {scrollExpansion} />

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

  <!-- 3 Vertically Arranged Pulsating Dots for Directional Overflow Cue -->
  <div 
    class="deep-space-scroll-cue direction-{activeCueDirection}" 
    class:visible={isCueVisible} 
    aria-hidden="true"
  >
    <span class="scroll-dot dot-1"></span>
    <span class="scroll-dot dot-2"></span>
    <span class="scroll-dot dot-3"></span>
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
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 7px;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.45s cubic-bezier(0.22, 1, 0.36, 1), top 0.3s ease, bottom 0.3s ease;
    z-index: 60;
  }

  .deep-space-scroll-cue.direction-down {
    bottom: 24px;
    top: auto;
  }

  .deep-space-scroll-cue.direction-up {
    top: 24px;
    bottom: auto;
  }

  .deep-space-scroll-cue.visible {
    opacity: 1;
  }

  .scroll-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    transition: background 0.3s ease, box-shadow 0.3s ease;
  }

  /* Requirement 4: Downward cue is 深灰蓝色 (Deep Slate Blue) */
  .direction-down .scroll-dot {
    background: #4a5c78;
    box-shadow: 0 0 8px rgba(74, 92, 120, 0.6), 0 1px 3px rgba(0, 0, 0, 0.3);
    animation: deep-dot-pulse-down 1.8s ease-in-out infinite;
  }

  .direction-down .dot-1 {
    animation-delay: 0s;
  }

  .direction-down .dot-2 {
    animation-delay: 0.25s;
  }

  .direction-down .dot-3 {
    animation-delay: 0.5s;
  }

  @keyframes deep-dot-pulse-down {
    0%, 100% {
      opacity: 0.35;
      transform: scale(0.85);
      background: #3b4b63;
      box-shadow: 0 0 5px rgba(59, 75, 99, 0.4), 0 1px 2px rgba(0, 0, 0, 0.25);
    }
    50% {
      opacity: 1;
      transform: scale(1.25);
      background: #6f8cb0;
      box-shadow: 0 0 14px rgba(111, 140, 176, 0.85), 0 0 4px rgba(145, 175, 210, 0.6), 0 2px 5px rgba(0, 0, 0, 0.35);
    }
  }

  /* Requirement 4: Upward cue is 蓝白色 (Ice Blue-White) */
  .direction-up .scroll-dot {
    background: #edf5ff;
    box-shadow: 0 0 10px rgba(130, 185, 255, 0.75), 0 0 3px rgba(255, 255, 255, 0.9);
    animation: deep-dot-pulse-up 1.8s ease-in-out infinite;
  }

  .direction-up .dot-3 {
    animation-delay: 0s;
  }

  .direction-up .dot-2 {
    animation-delay: 0.25s;
  }

  .direction-up .dot-1 {
    animation-delay: 0.5s;
  }

  @keyframes deep-dot-pulse-up {
    0%, 100% {
      opacity: 0.38;
      transform: scale(0.85);
      background: #d0e2ff;
      box-shadow: 0 0 6px rgba(110, 170, 245, 0.5), 0 0 2px rgba(255, 255, 255, 0.6);
    }
    50% {
      opacity: 1;
      transform: scale(1.3);
      background: #ffffff;
      box-shadow: 0 0 16px rgba(150, 205, 255, 0.95), 0 0 5px rgba(255, 255, 255, 1), 0 2px 6px rgba(0, 0, 0, 0.25);
    }
  }
</style>
