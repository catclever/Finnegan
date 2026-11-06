<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - UICard Component
  Physical shell, 3D spring coordinate container, and card body delegator.
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { spring } from 'svelte/motion';
  import { getCardDisplayTime, stageTime } from './lib/time';
  import { getCardComponent } from './components/cards/cardRegistry';
  import CardAuraGlow from './components/cards/CardAuraGlow.svelte';
  import CardCollectControl from './components/cards/CardCollectControl.svelte';
  import { OpenCardInEditor } from '../wailsjs/go/main/App.js';
  import { cardHasTag, cardHasKeyword } from './lib/tagNetwork';
  import { VIP_INK_BEAM_DURATION_MS } from './lib/stageDecay';
  import { computeScatteredSpringTarget } from './lib/stageCoords';

  const dispatch = createEventDispatcher();

  export let card: any;
  export let allCards: any[] = [];

  export let globalTilt: { rx: number, ry: number } = { rx: 0, ry: 0 };
  export let scrollExpansion: number = 0;
  export let isFocused: boolean = false;
  export let hasAnyFocused: boolean = false;
  export let isHovered: boolean = false;
  export let onToggleFocus: () => void;
  export let onFocusedClick: (() => void) | undefined = undefined;
  export let onHover: () => void;
  export let onLeave: () => void;
  export let focusNavDirection: 'prev' | 'next' | null = null;
  export let hoveredMotherId: string | null = null;
  export let activeConstellationTags: Map<string, string> = new Map();
  export let activeConstellationTag: string | null = null;
  export let isSearchStopped: boolean = false;
  export let clusteredOverride: { x: number; y: number; z: number; rotation: number; motherId: string } | null = null;
  export let themePeekRole: 'left' | 'right' | null = null;
  export let isThemeActive: boolean = false;
  export let isBuguScanning: boolean = false;
  export let isBskyScanning: boolean = false;
  $: void isThemeActive;

  let lastNavDir: 'prev' | 'next' | null = null;
  $: if (!isFocused) lastNavDir = null;

  let isVisible = true;
  $: isVisible = isFocused || themePeekRole !== null || Math.abs(card.y - scrollExpansion * 300) <= 850;

  let animSpring = spring({
    x: card.x,
    y: card.y,
    rotation: card.rotation,
    opacity: 1,
    scale: 1,
    blur: 0,
    saturation: 1,
    textGray: 38,
    textOpacity: 1,
    rotX: 0,
    rotY: 0,
    z3d: 0
  }, { stiffness: 0.06, damping: 0.35 });

  // Vertical pixel offset matching parent .scene translateY(-scrollExpansion * 300px)
  $: scrollOffsetY = scrollExpansion * 300;

  // --- Aura Search Flowing Light & Decay Logic ---
  $: isConstellationDimmed = (() => {
    if (isFocused) return false;
    if (activeConstellationTags && activeConstellationTags.size > 0) {
      const activeTags = Array.from(activeConstellationTags.keys());
      return !activeTags.some(tag => cardHasTag(card, tag) || cardHasKeyword(card, tag));
    }
    if (activeConstellationTag) {
      return !cardHasTag(card, activeConstellationTag) && !cardHasKeyword(card, activeConstellationTag);
    }
    return false;
  })();

  $: auraTraces = card.auraTraces || [];
  $: activeAuraTraces = auraTraces.filter((t: any) => !t.duration || Date.now() - (t.hitTime || 0) < t.duration);
  $: hasAura = activeAuraTraces.length > 0;

  // Find latest hit time and most recent aura color
  $: latestHitTime = hasAura 
    ? Math.max(...activeAuraTraces.map((t: any) => t.hitTime || 0)) 
    : 0;

  $: latestColor = hasAura 
    ? (activeAuraTraces[activeAuraTraces.length - 1]?.color || 'hsl(180, 95%, 62%)') 
    : '';

  // Safely convert hsl string to hsla with given alpha for valid WebKit box-shadows
  function toHsla(hslStr: string, alpha: number): string {
    if (!hslStr) return 'transparent';
    const m = hslStr.match(/hsl\(\s*([\d.]+)[,\s]+([\d.]+)%?[,\s]+([\d.]+)%?\s*\)/i);
    if (m) {
      return `hsla(${m[1]}, ${m[2]}%, ${m[3]}%, ${alpha})`;
    }
    return hslStr;
  }

  // Elapsed time in ms since the most recent search hit
  $: elapsed = hasAura ? Math.max(0, Date.now() - latestHitTime) : Infinity;

  // Temporal decay stages:
  // 1: < 3 min (180,000 ms)   - Bright flowing neon glow
  // 2: 3 ~ 10 min (600,000 ms)  - Soft steady border glow
  // 3: 10 ~ 15 min (900,000 ms) - Fine crisp 1.5px colored border
  // > 15 min: Extinguished (hasAura = false, auraStage = 0)
  $: auraStage = !hasAura ? 0 : (elapsed < 3 * 60 * 1000 ? 1 : (elapsed < 10 * 60 * 1000 ? 2 : 3));

  // Dynamic conic-gradient for spinning border
  $: conicGradient = (() => {
    if (!hasAura) return '';
    const colors = activeAuraTraces.map((t: any) => t.color).filter(Boolean);
    if (colors.length === 0) return '';
    if (colors.length === 1) {
      const c = colors[0];
      return `conic-gradient(from 0deg, transparent 0deg, ${c} 70deg, transparent 140deg, transparent 180deg, ${c} 250deg, transparent 320deg, transparent 360deg)`;
    }
    const stops = colors.map((c: string, i: number) => {
      const deg = Math.round((i / colors.length) * 360);
      return `${c} ${deg}deg`;
    });
    return `conic-gradient(from 0deg, ${stops.join(', ')}, ${colors[0]} 360deg)`;
  })();

  $: isBskyEcho = card.origin === 'atproto' && Boolean(card.echoOf);
  $: isBuguEcho = card.origin === 'bugu' && Boolean(card.echoOf);
  $: isTemporary = card.isTemporary === false ? false : Boolean(card.isTemporary || card.origin === 'bugu' || card.origin === 'atproto' || card.type === 'radar');

  let collectControlRef: CardCollectControl | null = null;
  $: isRecentVIP = Boolean(
    card.isVIP &&
    !hasAura &&
    !isBuguEcho &&
    !isBskyEcho &&
    !isBuguScanning &&
    !isBskyScanning &&
    ($stageTime - (card.spawnTime || (card.createdAt ? new Date(card.createdAt).getTime() : 0)) < VIP_INK_BEAM_DURATION_MS)
  );

  $: cardHash = card?.id ? card.id.split('').reduce((a: number, b: string) => a + b.charCodeAt(0), 0) : 0;

  // Dynamic box-shadow accounting for focus, temporary state & aura decay stages
  $: dynamicBoxShadow = (() => {
    if (isFocused) {
      if (hasAura) {
        return `0 0 0 1.5px ${toHsla(latestColor, 0.95)}, 0 0 28px ${toHsla(latestColor, 0.7)}, 0 40px 100px rgba(0, 0, 0, 0.22)`;
      }
      if (isBskyEcho) {
        return '0 0 0 1px rgba(56, 189, 248, 0.9), 0 0 8px rgba(56, 189, 248, 0.55), 0 0 20px rgba(56, 189, 248, 0.35), 0 0 45px rgba(14, 165, 233, 0.18), 0 0 75px rgba(14, 165, 233, 0.06), 0 40px 100px rgba(0, 0, 0, 0.22), inset 0 0 16px rgba(224, 242, 254, 0.45)';
      }
      return isBuguEcho
        ? '0 0 0 1px rgba(245, 158, 11, 0.9), 0 0 8px rgba(245, 158, 11, 0.55), 0 0 20px rgba(245, 158, 11, 0.35), 0 0 45px rgba(245, 158, 11, 0.18), 0 0 75px rgba(245, 158, 11, 0.06), 0 40px 100px rgba(0, 0, 0, 0.22), inset 0 0 16px rgba(254, 243, 199, 0.4)'
        : '0 40px 100px rgba(0, 0, 0, 0.22), 0 0 0 1px rgba(255, 255, 255, 0.9)';
    }
    if (!hasAura) {
      if (isRecentVIP) {
        return '';
      }
      return '0 8px 24px rgba(0, 0, 0, 0.04)';
    }
    if (auraStage === 1) {
      return `0 0 0 1.5px ${toHsla(latestColor, 0.95)}, 0 0 16px ${toHsla(latestColor, 0.85)}, 0 0 32px ${toHsla(latestColor, 0.4)}, 0 8px 24px rgba(0, 0, 0, 0.04)`;
    } else if (auraStage === 2) {
      return `0 0 0 1.2px ${toHsla(latestColor, 0.8)}, 0 0 12px ${toHsla(latestColor, 0.45)}, 0 8px 24px rgba(0, 0, 0, 0.04)`;
    } else {
      return `0 0 0 1px ${toHsla(latestColor, 0.65)}, 0 8px 24px rgba(0, 0, 0, 0.04)`;
    }
  })();

  $: {
    if (card.isFadingOut) {
      animSpring.set({
        x: card.x,
        y: card.y + 60,
        rotation: card.rotation + 10,
        opacity: 0,
        scale: 0.35,
        blur: 16,
        saturation: 0,
        textGray: 140,
        textOpacity: 0,
        rotX: 0,
        rotY: 0,
        z3d: 0
      });
    } else if (isFocused) {
      if (focusNavDirection && focusNavDirection !== lastNavDir) {
        lastNavDir = focusNavDirection;
        const offset = focusNavDirection === 'next' ? window.innerWidth : -window.innerWidth;
        animSpring.set({
          x: offset,
          y: scrollOffsetY,
          rotation: 0,
          opacity: 0,
          scale: 0.8,
          blur: 0,
          saturation: 1,
          textGray: 32,
          textOpacity: 1,
          rotX: 0,
          rotY: 0,
          z3d: 160
        }, { hard: true });
      }
      animSpring.set({
        x: 0,
        y: scrollOffsetY,
        rotation: 0,
        opacity: 1,
        scale: 1,
        blur: 0,
        saturation: 1,
        textGray: 32,
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 160
      });
    } else if (themePeekRole === 'left') {
      animSpring.set({
        x: -360,
        y: scrollOffsetY,
        rotation: 0,
        opacity: 0.95,
        scale: 0.82,
        blur: 0,
        saturation: 1,
        textGray: 38,
        textOpacity: 0.95,
        rotX: 0,
        rotY: 14,
        z3d: 125
      });
    } else if (themePeekRole === 'right') {
      animSpring.set({
        x: 360,
        y: scrollOffsetY,
        rotation: 0,
        opacity: 0.95,
        scale: 0.82,
        blur: 0,
        saturation: 1,
        textGray: 38,
        textOpacity: 0.95,
        rotX: 0,
        rotY: -14,
        z3d: 125
      });
    } else if (isHovered) {
      const pullX = Math.max(-440, Math.min(440, card.x));
      const pullY = Math.max(scrollOffsetY - 260, Math.min(scrollOffsetY + 260, card.y));
      animSpring.set({
        x: pullX,
        y: pullY,
        rotation: 0,
        opacity: 1,
        scale: 1.15,
        blur: 0,
        saturation: 1,
        textGray: 32,
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 140
      });

    } else if (!hasAnyFocused && hoveredMotherId !== null && card.echoOf === hoveredMotherId) {
      // Mother card hovered: straighten up child cards (rotation: 0)
      animSpring.set(
        computeScatteredSpringTarget(
          card.x,
          card.y,
          0,
          card.z || 0,
          scrollOffsetY,
          Boolean(card.isFresh),
          isConstellationDimmed,
          hasAnyFocused,
          globalTilt
        )
      );
    } else {
      const isClusteredScattered = !hasAnyFocused && !isFocused && clusteredOverride !== null;
      const targetX = isClusteredScattered ? clusteredOverride.x : card.x;
      const targetY = isClusteredScattered ? clusteredOverride.y : card.y;
      const targetRot = isClusteredScattered ? clusteredOverride.rotation : card.rotation;
      const targetZ = isClusteredScattered ? clusteredOverride.z : (card.z || 0);

      animSpring.set(
        computeScatteredSpringTarget(
          targetX,
          targetY,
          targetRot,
          targetZ,
          scrollOffsetY,
          Boolean(card.isFresh),
          isConstellationDimmed,
          hasAnyFocused,
          globalTilt
        )
      );
    }
  }

  $: displayTime = getCardDisplayTime(card, $stageTime);
  $: babelTime = displayTime;
  $: thoughtTime = displayTime;

  let focusedAt = 0;
  $: if (isFocused) {
    if (focusedAt === 0) focusedAt = Date.now();
  } else {
    focusedAt = 0;
  }

  async function triggerOpenEditor() {
    try {
      await OpenCardInEditor(card.filePath || card.id);
    } catch (err) {
      console.error('Failed to open card in editor:', err);
    }
  }

  async function handleDoubleClick(e: MouseEvent) {
    e.stopPropagation();
    if (isTemporary) {
      collectControlRef?.showCollectPrompt();
      return;
    }
    if (!isFocused) return;
    // Gate: avoid accidental double-clicks while the card is transitioning into focus
    if (Date.now() - focusedAt < 250) return;
    await triggerOpenEditor();
  }

  function handleClick(e: MouseEvent) {
    e.stopPropagation();
    // Option (⌥) + Click: Quick-trigger external editor directly in scattered state
    if (e.altKey) {
      e.preventDefault();
      if (isTemporary) {
        collectControlRef?.showCollectPrompt();
        return;
      }
      triggerOpenEditor();
      return;
    }
    if (!isFocused) {
      onToggleFocus();
    } else {
      if (onFocusedClick) onFocusedClick();
    }
  }
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div 
  class="card"
  class:inert={hasAnyFocused && !isFocused && !themePeekRole}
  class:constellation-dimmed={isConstellationDimmed && !themePeekRole}
  class:bugu-echo={isBuguEcho}
  class:bsky-echo={isBskyEcho}
  class:bugu-scanning={isBuguScanning}
  class:bsky-scanning={isBskyScanning}
  class:temporary={isTemporary}
  class:clustered-child={clusteredOverride !== null}
  class:focused={isFocused}
  class:hovered={isHovered && !isFocused}
  class:inspiration-vip={isRecentVIP && !isFocused}
  class:has-aura={hasAura}
  class:aura-stage-1={auraStage === 1}
  class:aura-stage-2={auraStage === 2}
  class:aura-stage-3={auraStage === 3}
  class:theme-peek={themePeekRole !== null}
  class:diary-parchment={card.type === 'diary'}
  title={isTemporary ? '需先收集卡片才能编辑' : (isFocused ? '双击进行编辑' : '⌥ + 单击在外部编辑')}
  on:click={handleClick}
  on:dblclick={handleDoubleClick}
  on:wheel={(e) => { if (isFocused) e.stopPropagation(); }}
  on:mouseenter={onHover}
  on:mouseleave={onLeave}
  style="
    transform: translate(calc(-50% + {$animSpring.x}px), calc(-50% + {$animSpring.y}px)) 
               translateZ({$animSpring.z3d}px)
               rotate({isFocused ? 0 : $animSpring.rotation}deg) 
               rotateX({isFocused ? 0 : $animSpring.rotX}deg) 
               rotateY({isFocused ? 0 : $animSpring.rotY}deg) 
               scale({$animSpring.scale});
    opacity: {$animSpring.opacity};
    filter: blur({$animSpring.blur}px) saturate({$animSpring.saturation});
    z-index: {isFocused ? 1000 : (themePeekRole ? 850 : (isHovered && !hasAnyFocused ? 999 : (hasAura ? 300 : Math.round(50 - (card.z || 0) * 0.3))))};
    {dynamicBoxShadow ? `box-shadow: ${dynamicBoxShadow};` : ''}
    {hasAura ? `border: 1.5px solid ${toHsla(latestColor, auraStage === 1 ? 0.95 : (auraStage === 2 ? 0.8 : 0.65))} !important;` : ''}
    --card-text-color: rgba({Math.round($animSpring.textGray || 38)}, {Math.round($animSpring.textGray || 38)}, {Math.round($animSpring.textGray || 38)}, {($animSpring.textOpacity ?? 0.95).toFixed(2)});
    --card-text-secondary: rgba({Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {($animSpring.textOpacity ?? 0.95).toFixed(2)});
    --temp-rim-delay: -{(cardHash % 55) / 10}s;
    --vip-breathe-delay: -{(cardHash % 80) / 10}s;
    {!isVisible ? 'visibility: hidden; pointer-events: none;' : ''}
  "
>
  <CardAuraGlow {card} {hasAura} {auraStage} {conicGradient} {isBuguEcho} {isBskyEcho} isStopped={isSearchStopped} {isFocused} isRecentVIP={isRecentVIP && !isFocused} {isTemporary} {isBuguScanning} {isBskyScanning} />

  {#if isTemporary}
    <CardCollectControl
      bind:this={collectControlRef}
      bind:card
      {isFocused}
      {isHovered}
      {hasAnyFocused}
      on:cardPromoted
    />
  {/if}

  <svelte:component
    this={getCardComponent(card.type)}
    {card}
    {allCards}
    z={card.z}
    {isFocused}
    {isHovered}
    {thoughtTime}
    {babelTime}
    {displayTime}
    {activeConstellationTags}
    {activeConstellationTag}
    on:tagClick={(e) => dispatch('tagClick', e.detail)}
  />
</div>

<style>
  .card {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 200px;
    height: 190px;
    padding: 12px 18px;
    text-align: left;
    pointer-events: auto;
    background: rgba(255, 255, 255, 0.76);
    border-radius: 16px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.04);
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
    cursor: pointer;
    transform-origin: center center;
    user-select: none;
    backdrop-filter: blur(22px);
    -webkit-backdrop-filter: blur(22px);
    border: 1px solid rgba(255, 255, 255, 0.65);
    overflow: hidden;
  }

  .card.inert {
    pointer-events: none !important;
  }

  .card.constellation-dimmed {
    opacity: 0.35 !important;
    transition: opacity 0.3s ease;
  }

  /* Ephemeral / Temporary Card Aesthetics: Translucent Mist-Blue Base + Dynamic Breathing Rim */
  .card.temporary {
    background: linear-gradient(135deg, rgba(226, 236, 248, 0.58) 0%, rgba(210, 222, 238, 0.42) 100%);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border: 1px solid transparent !important;
    box-shadow: inset 0 0 20px rgba(255, 255, 255, 0.7), inset 0 0 38px rgba(210, 226, 245, 0.45);
  }

  .card.temporary::before {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    pointer-events: none;
    z-index: 0;
    background: radial-gradient(
      ellipse at center,
      transparent 42%,
      rgba(248, 250, 252, 0.32) 76%,
      rgba(224, 234, 246, 0.70) 100%
    );
    animation: temporary-rim-breathe 5.5s ease-in-out infinite;
    animation-delay: var(--temp-rim-delay, 0s);
  }

  @keyframes temporary-rim-breathe {
    0%, 100% {
      opacity: 0.65;
    }
    50% {
      opacity: 0.95;
    }
  }

  .card.inspiration-vip {
    border: 1px solid transparent !important;
    animation: ink-vip-breathe 8s ease-in-out infinite;
    animation-delay: var(--vip-breathe-delay, 0s);
  }

  @keyframes ink-vip-breathe {
    0%, 100% {
      box-shadow: 0 8px 24px rgba(71, 85, 105, 0.06), 0 2px 6px rgba(100, 116, 139, 0.04);
    }
    50% {
      box-shadow: 0 12px 36px rgba(71, 85, 105, 0.12), 0 0 18px rgba(100, 116, 139, 0.08);
    }
  }

  .card.bugu-echo:not(.focused),
  .card.bsky-echo:not(.focused),
  .card.bugu-scanning:not(.focused),
  .card.bsky-scanning:not(.focused) {
    border: 1px solid transparent !important;
  }

  .card.bugu-echo.focused {
    border: 1px solid rgba(245, 158, 11, 0.85) !important;
    box-shadow: 
      0 0 0 1px rgba(245, 158, 11, 0.9),
      0 0 8px rgba(245, 158, 11, 0.55),
      0 0 20px rgba(245, 158, 11, 0.35),
      0 0 45px rgba(245, 158, 11, 0.18),
      0 0 75px rgba(245, 158, 11, 0.06),
      0 40px 100px rgba(0, 0, 0, 0.22),
      inset 0 0 16px rgba(254, 243, 199, 0.4) !important;
  }

  .card.bsky-echo.focused {
    border: 1px solid rgba(56, 189, 248, 0.85) !important;
    box-shadow: 
      0 0 0 1px rgba(56, 189, 248, 0.9),
      0 0 8px rgba(56, 189, 248, 0.55),
      0 0 20px rgba(56, 189, 248, 0.35),
      0 0 45px rgba(14, 165, 233, 0.18),
      0 0 75px rgba(14, 165, 233, 0.06),
      0 40px 100px rgba(0, 0, 0, 0.22),
      inset 0 0 16px rgba(224, 242, 254, 0.45) !important;
  }

  .card.has-aura.temporary::before,
  .card.bugu-echo.temporary::before,
  .card.bsky-echo.temporary::before,
  .card.bugu-scanning.temporary::before,
  .card.bsky-scanning.temporary::before {
    display: none;
  }

  .card.hovered::after {
    content: '';
    position: absolute;
    top: -250px;
    left: -250px;
    right: -250px;
    bottom: -250px;
    z-index: -1;
  }

  .card.focused {
    width: 420px;
    height: 420px;
    max-width: 88vw;
    max-height: 88vh;
    padding: 26px 36px;
    border-radius: 28px;
    background: rgba(255, 255, 255, 0.96);
    box-shadow: 0 40px 100px rgba(0, 0, 0, 0.22), 0 0 0 1px rgba(255, 255, 255, 0.9);
    cursor: default;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    scrollbar-width: none;
    -ms-overflow-style: none;
  }

  .card.temporary.focused {
    background: linear-gradient(145deg, rgba(226, 236, 248, 0.52) 0%, rgba(210, 222, 238, 0.40) 100%);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    border: 1px solid transparent !important;
  }

  .card.focused::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  .card.clustered-child {
    pointer-events: none !important;
  }

  .card.theme-peek {
    cursor: pointer;
    pointer-events: auto !important;
    transition: box-shadow 0.2s ease, filter 0.2s ease;
  }

  .card.theme-peek:hover {
    filter: brightness(1.05) saturate(1.1);
    box-shadow: 
      0 0 0 2px rgba(56, 189, 248, 0.9),
      0 0 32px rgba(56, 189, 248, 0.6),
      0 0 60px rgba(168, 85, 247, 0.35),
      0 20px 60px rgba(0, 0, 0, 0.2) !important;
  }

  .card.diary-parchment {
    background: linear-gradient(165deg, #fbf3df 0%, #f3e4c2 52%, #ead5a7 100%) !important;
    border-top: 4px solid #8b5a2b !important;
    border-bottom: 4px solid #8b5a2b !important;
    border-left: 1px solid rgba(139, 90, 43, 0.4) !important;
    border-right: 1px solid rgba(139, 90, 43, 0.4) !important;
    box-shadow: inset 0 2px 8px rgba(255, 255, 255, 0.6), 0 6px 16px rgba(139, 90, 43, 0.2) !important;
  }

  .card.diary-parchment.focused {
    box-shadow: 0 40px 100px rgba(62, 39, 35, 0.3), inset 0 2px 8px rgba(255, 255, 255, 0.6) !important;
    border-top: 8px solid #8b5a2b !important;
    border-bottom: 8px solid #8b5a2b !important;
  }
</style>
