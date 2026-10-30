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
  import CosmicRayNetwork from './cards/CosmicRayNetwork.svelte';
  import type { BuguBeacon, BuguSession } from '../modules/bugu/types';
  import BskyButterflyBeacon from '../modules/atproto/BskyButterflyBeacon.svelte';
  import type { BskyBeacon } from '../modules/atproto/types';
  import { getMaxScrollExpansion, computeClusteredChildCoords, isClusteredChildCard } from '../lib/stageCoords';
  import VirtualCardProxy from './cards/VirtualCardProxy.svelte';

  // Props
  export let cards: CosmicCard[] = [];

  export let globalTilt: { rx: number; ry: number } = { rx: 0, ry: 0 };
  export let scrollExpansion: number = 0;
  export let focusedCardId: string | null = null;
  export let hoveredCardId: string | null = null;
  export let activeConstellationTags: Map<string, string> = new Map();
  export let activeConstellationTag: string | null = null;
  export let focusNavDirection: 'prev' | 'next' | null = null;
  export let isOmnibarFocused: boolean = false;
  export let isSearchActive: boolean = false;
  export let buguSession: BuguSession | null = null;
  export let buguBeacon: BuguBeacon | null = null;
  export let bskyBeacon: BskyBeacon | null = null;
  export let lastScrollDirection: 'down' | 'up' = 'down';
  export let isWindowFocused: boolean = true;
  export let isStageSleeping: boolean = false;
  export let totalAvailableCards: number = 0;

  $: void isOmnibarFocused;

  // Callbacks
  export let onToggleFocus: ((cardId: string) => void) | undefined = undefined;
  export let onHoverCard: ((cardId: string) => void) | undefined = undefined;
  export let onLeaveCard: (() => void) | undefined = undefined;
  export let onBgClick: ((e: MouseEvent) => void) | undefined = undefined;

  const dispatch = createEventDispatcher();

  let activeCueDirection: 'down' | 'up' = 'down';
  let screen1DwellTimer: any = null;
  let bottomDwellTimer: any = null;
  let prevScrollDir: 'down' | 'up' = 'down';

  // Track max scroll depth and boundaries
  $: maxScroll = getMaxScrollExpansion(cards, totalAvailableCards);
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

    // Step 5: Bottom Dwell Timer (Prompt to scroll up when reaching the bottom)
    if (isNearBottom) {
      if (!bottomDwellTimer && activeCueDirection !== 'up') {
        bottomDwellTimer = setTimeout(() => {
          if (isNearBottom) {
            activeCueDirection = 'up';
          }
          bottomDwellTimer = null;
        }, 1000);
      }
    } else {
      if (bottomDwellTimer) { clearTimeout(bottomDwellTimer); bottomDwellTimer = null; }
    }
  }


  // --- ATProto / Bluesky Butterfly Beacon Coordinates ---
  $: bskySourceCard = bskyBeacon?.sourceCardId ? cards.find(c => c.id === bskyBeacon.sourceCardId) : null;
  $: bskyBeaconX = bskySourceCard ? bskySourceCard.x : 0;
  $: bskyBeaconY = bskySourceCard ? bskySourceCard.y : Math.round(scrollExpansion * 300);
  $: isSearchStopped = bskyBeacon ? (bskyBeacon.status === 'completed' || bskyBeacon.status === 'fading') : false;

  // --- Mother-Child Hover Logic ---
  $: activeMotherCard = hoveredCardId ? cards.find(c => c.id === hoveredCardId && cards.some(child => child.echoOf === c.id)) : null;
  $: activeChildCard = hoveredCardId ? cards.find(c => c.id === hoveredCardId && c.echoOf) : null;
  $: hoveredMotherId = activeMotherCard ? activeMotherCard.id : (activeChildCard ? activeChildCard.echoOf : null);

  // Requirement 2: Scattered Mode Child Hover -> Virtual Mother Card
  $: offscreenMotherProxy = (() => {
    if (focusedCardId) return null;
    if (activeChildCard && activeChildCard.echoOf) {
      const mother = cards.find(c => c.id === activeChildCard.echoOf && !c.isFadingOut);
      if (mother) {
        // Check if mother is out of view (roughly beyond +/- 850px from scroll center)
        const isOffscreen = Math.abs(mother.y - scrollExpansion * 300) > 600;
        if (isOffscreen) {
          return mother;
        }
      }
    }
    return null;
  })();

  // Requirement 3: Focused Mode Mother -> Child Cards Stack Deck
  $: focusedMotherChildrenDeck = (() => {
    if (!focusedCardId) return [];
    const focusedCard = cards.find(c => c.id === focusedCardId);
    if (!focusedCard) return [];

    // Check if it's a mother card
    const children = cards.filter(c => c.echoOf === focusedCardId && !c.isFadingOut);
    if (children.length > 0) {
      return children;
    }
    return [];
  })();

  // Requirement 4: Focused Mode Child -> Virtual Mother Card
  $: focusedChildMotherProxy = (() => {
    if (!focusedCardId) return null;
    const focusedCard = cards.find(c => c.id === focusedCardId);
    if (focusedCard && focusedCard.echoOf) {
      return cards.find(c => c.id === focusedCard.echoOf && !c.isFadingOut) || null;
    }
    return null;
  })();

  $: proxyCardsToRender = focusedMotherChildrenDeck.length > 0
    ? focusedMotherChildrenDeck
    : (focusedChildMotherProxy ? [focusedChildMotherProxy] : (offscreenMotherProxy ? [offscreenMotherProxy] : []));

  $: isProxyStackDeck = focusedMotherChildrenDeck.length > 0;

  // Requirement 4: Scattered-State Deck Clustering
  // When a collected card has card.echoOf and is not temporary:
  // In scattered state (when neither it nor mother is focused), if mother is present in cards:
  // Physically cluster this child card directly under the mother card:
  // x = mother.x + index * 5, y = mother.y + index * 5, z = mother.z + 2.5, rot = mother.rotation + index * 1.2
  $: clusteredChildMap = (() => {
    const map = new Map<string, { x: number; y: number; z: number; rotation: number; motherId: string }>();
    if (focusedCardId !== null) return map;

    const motherChildrenMap = new Map<string, CosmicCard[]>();
    for (const c of cards) {
      if (isClusteredChildCard(c, cards)) {
        const mother = cards.find(m => m.id === c.echoOf && !m.isFadingOut);
        if (mother) {
          if (!motherChildrenMap.has(mother.id)) {
            motherChildrenMap.set(mother.id, []);
          }
          motherChildrenMap.get(mother.id)!.push(c);
        }
      }
    }

    for (const [motherId, children] of motherChildrenMap.entries()) {
      const mother = cards.find(m => m.id === motherId)!;
      children.forEach((child, idx) => {
        const childIndex = idx + 1;
        const coords = computeClusteredChildCoords(mother, childIndex);
        map.set(child.id, { ...coords, motherId });
      });
    }

    return map;
  })();

  let isVirtualCardOnTop: boolean = false;
  let lastFocusedCardId: string | null = null;
  $: if (focusedCardId !== lastFocusedCardId) {
    lastFocusedCardId = focusedCardId;
    isVirtualCardOnTop = false;
  }

  $: hasCardsBelow = cards.some(c => !c.isFadingOut && !isClusteredChildCard(c, cards) && (c.y - scrollExpansion * 300) > 280);
  $: hasCardsAbove = scrollExpansion > 0.15;

  // Requirement 3:
  // - Reached bottom: if lastScrollDirection was 'down', do not show ANY cue (unless bottom dwell timer flips it to 'up')
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

  function handleNavLeft(e: MouseEvent) {
    e.stopPropagation();
    dispatch('navigate', 'prev');
  }

  function handleNavRight(e: MouseEvent) {
    e.stopPropagation();
    dispatch('navigate', 'next');
  }

  onDestroy(() => {
    if (screen1DwellTimer) clearTimeout(screen1DwellTimer);
    if (bottomDwellTimer) clearTimeout(bottomDwellTimer);
  });
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div 
  class="cards-wrapper" 
  class:elevated={focusedCardId !== null} 
  class:sleeping={isStageSleeping || !isWindowFocused}
  on:click={handleWrapperClick}
>
  {#if focusedCardId !== null}
    <div class="nav-button nav-left" on:click={handleNavLeft}>
      <svg viewBox="0 0 24 24" width="32" height="32" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>
    </div>
    <div class="nav-button nav-right" on:click={handleNavRight}>
      <svg viewBox="0 0 24 24" width="32" height="32" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
    </div>
  {/if}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div 
    class="scene" 
    style="
      transform: rotateX({focusedCardId !== null ? 0 : globalTilt.rx}deg) rotateY({focusedCardId !== null ? 0 : globalTilt.ry}deg) translateY({-scrollExpansion * 300}px); 
      transform-style: preserve-3d;
    "
  >
    <!-- Focus Backdrop -->
    {#if focusedCardId !== null}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div
        class="focus-backdrop"
        style="transform: translateZ(112px) translateY({scrollExpansion * 300}px);"
        on:click|stopPropagation={handleWrapperClick}
      ></div>
    {/if}

    <!-- Search Resonance Backdrop: isolates foreground search cards from pushed-back background cards -->
    {#if isSearchActive && focusedCardId === null}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div
        class="search-backdrop"
        style="transform: translateZ(5px) translateY({scrollExpansion * 300}px);"
        on:click|stopPropagation={handleWrapperClick}
      ></div>
    {/if}

    <CosmicRayNetwork 
      {cards} 
      beacon={buguBeacon} 
      session={buguSession} 
      {scrollExpansion} 
      {hoveredMotherId} 
      {hoveredCardId} 
      {focusedCardId} 
      {activeConstellationTag} 
    />

    {#if bskyBeacon}
      <BskyButterflyBeacon
        x={bskyBeaconX}
        y={bskyBeaconY}
        status={bskyBeacon.status}
        count={bskyBeacon.butterflyCount}
      />
    {/if}

    {#each cards as card (card.id)}
      <UICard 
        {card}

        {globalTilt}
        {scrollExpansion}
        {activeConstellationTags}
        {activeConstellationTag}
        isSearchStopped={isSearchStopped}
        hasAnyFocused={focusedCardId !== null}
        focusNavDirection={focusedCardId === card.id ? focusNavDirection : null}
        hoveredMotherId={hoveredMotherId}
        isFocused={focusedCardId === card.id}
        isHovered={hoveredCardId === card.id}
        clusteredOverride={clusteredChildMap.get(card.id) || null}
        onToggleFocus={() => handleToggleFocus(card.id)}
        onFocusedClick={() => { isVirtualCardOnTop = false; }}
        onHover={() => handleHover(card.id)}
        onLeave={handleLeave}
        on:tagClick={(e) => dispatch('tagClick', e.detail)}
        on:cardPromoted={(e) => dispatch('cardPromoted', e.detail)}
      />
    {/each}

    {#if proxyCardsToRender.length > 0}
      <VirtualCardProxy
        cards={proxyCardsToRender}
        isStackDeck={isProxyStackDeck}
        isOnTop={isVirtualCardOnTop}
        scrollExpansion={scrollExpansion}
        globalTilt={globalTilt}
        isFocusedMode={focusedCardId !== null}
        on:cardClick={(e) => {
          if (focusedCardId !== null) {
            isVirtualCardOnTop = true;
          } else {
            handleToggleFocus(e.detail.id);
          }
        }}
        on:cardDblClick={(e) => {
          handleToggleFocus(e.detail.id);
        }}
      />
    {/if}
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
  .cards-wrapper.sleeping :global(*) {
    animation-play-state: paused !important;
  }

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
  }

  .cards-wrapper.elevated {
    z-index: 100; /* Elevated above omnibar-container (z-index 50) */
  }

  .scene {
    pointer-events: none;
    width: 100%;
    height: 100%;
    position: relative;
    transform-style: preserve-3d;
    will-change: transform;
  }

  .focus-backdrop {
    position: absolute;
    top: -100vh;
    left: -100vw;
    width: 300vw;
    height: 300vh;
    background: rgba(248, 250, 252, 0.72);
    z-index: 500;
    pointer-events: auto;
    animation: focus-fade-in 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .search-backdrop {
    position: absolute;
    top: -100vh;
    left: -100vw;
    width: 300vw;
    height: 300vh;
    background: rgba(242, 245, 249, 0.38);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    z-index: 100;
    pointer-events: auto;
    animation: focus-fade-in 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes focus-fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  /* Navigation Buttons */
  .nav-button {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.35);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    color: rgba(30, 40, 50, 0.7);
    display: flex;
    justify-content: center;
    align-items: center;
    cursor: pointer;
    z-index: 600;
    pointer-events: auto;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
    transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .nav-button:hover {
    background: rgba(255, 255, 255, 0.85);
    color: rgba(20, 30, 40, 0.9);
    transform: translateY(-50%) scale(1.1);
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.12);
  }

  .nav-left {
    left: 40px;
  }

  .nav-right {
    right: 40px;
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
