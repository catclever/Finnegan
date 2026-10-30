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
  import { OpenCardInEditor, PromoteTempCard } from '../wailsjs/go/main/App.js';
  import { cardHasTag, cardHasKeyword } from './lib/tagNetwork';
  import { getCardPromotionStrategy } from './lib/promotionRegistry';
  import { VIP_INK_BEAM_DURATION_MS } from './lib/stageDecay';

  const dispatch = createEventDispatcher();

  export let card: any;

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

  let lastNavDir: 'prev' | 'next' | null = null;
  $: if (!isFocused) lastNavDir = null;

  let isVisible = true;
  $: isVisible = isFocused || Math.abs(card.y - scrollExpansion * 300) <= 850;

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
  $: promotionStrategy = getCardPromotionStrategy(card);

  let isCommentStripOpen = false;
  let commentText = '';
  let commentInputEl: HTMLInputElement | null = null;
  let commentStripEl: HTMLDivElement | null = null;
  $: isRecentVIP = Boolean(
    card.isVIP &&
    !hasAura &&
    !isBuguEcho &&
    !isBskyEcho &&
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
      const viewY = card.y - scrollOffsetY;
      const dist = Math.sqrt(card.x * card.x + viewY * viewY);
      const edgeFade = Math.max(0, Math.min(1, 1 - (dist - 440) / 240));
      const edgeBlur = Math.max(0, Math.min(5, (dist - 400) * 0.025));

      const zNorm = Math.max(0, Math.min(1, (card.z || 0) / 100));
      const uniformScale = Number((0.55 + 0.70 * Math.pow(1 - zNorm, 1.2)).toFixed(3));
      const z3dVal = (100 - (card.z || 0)) * 2.4 - 120;

      const depthBlur = zNorm > 0.60 ? (zNorm - 0.60) * 6 : 0;
      const baseBlur = Math.min(4.5, depthBlur + edgeBlur);

      const depthOpacity = 0.96 - (zNorm * 0.32);
      const baseOpacity = depthOpacity * (card.isFresh ? Math.max(0.85, edgeFade) : edgeFade);
      const baseSat = Math.max(0.42, 1.0 - (zNorm * 0.5));

      const targetGray = Math.round(34 + zNorm * 111);
      const targetTextOp = Number((0.98 - zNorm * 0.16).toFixed(2));

      const finalOpacity = isConstellationDimmed ? 0.35 : (hasAnyFocused ? baseOpacity * 0.35 : baseOpacity);

      animSpring.set({
        x: card.x,
        y: card.y,
        rotation: 0,
        opacity: finalOpacity,
        scale: uniformScale,
        blur: hasAnyFocused ? baseBlur + 10 : baseBlur,
        saturation: hasAnyFocused ? baseSat * 0.5 : baseSat,
        textGray: targetGray,
        textOpacity: targetTextOp,
        rotX: globalTilt.rx * 0.08,
        rotY: globalTilt.ry * 0.08,
        z3d: z3dVal
      });
    } else {
      const isClusteredScattered = !hasAnyFocused && !isFocused && clusteredOverride !== null;
      const targetX = isClusteredScattered ? clusteredOverride.x : card.x;
      const targetY = isClusteredScattered ? clusteredOverride.y : card.y;
      const targetRot = isClusteredScattered ? clusteredOverride.rotation : card.rotation;
      const targetZ = isClusteredScattered ? clusteredOverride.z : (card.z || 0);

      const viewY = targetY - scrollOffsetY;
      const dist = Math.sqrt(targetX * targetX + viewY * viewY);
      const edgeFade = Math.max(0, Math.min(1, 1 - (dist - 440) / 240));
      const edgeBlur = Math.max(0, Math.min(5, (dist - 400) * 0.025));

      const zNorm = Math.max(0, Math.min(1, targetZ / 100));
      const uniformScale = Number((0.55 + 0.70 * Math.pow(1 - zNorm, 1.2)).toFixed(3));
      const z3dVal = (100 - targetZ) * 2.4 - 120;

      const depthBlur = zNorm > 0.60 ? (zNorm - 0.60) * 6 : 0;
      const baseBlur = Math.min(4.5, depthBlur + edgeBlur);

      const depthOpacity = 0.96 - (zNorm * 0.32);
      const baseOpacity = depthOpacity * (card.isFresh ? Math.max(0.85, edgeFade) : edgeFade);
      const baseSat = Math.max(0.42, 1.0 - (zNorm * 0.5));

      const targetGray = Math.round(34 + zNorm * 111);
      const targetTextOp = Number((0.98 - zNorm * 0.16).toFixed(2));

      const finalOpacity = isConstellationDimmed ? 0.35 : (hasAnyFocused ? baseOpacity * 0.35 : baseOpacity);

      animSpring.set({
        x: targetX,
        y: targetY,
        rotation: targetRot,
        opacity: finalOpacity,
        scale: uniformScale,
        blur: hasAnyFocused ? baseBlur + 10 : baseBlur,
        saturation: hasAnyFocused ? baseSat * 0.5 : baseSat,
        textGray: targetGray,
        textOpacity: targetTextOp,
        rotX: globalTilt.rx * 0.08,
        rotY: globalTilt.ry * 0.08,
        z3d: z3dVal
      });
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

  let isPromoting = false;
  let collectPromptMessage: string | null = null;
  let collectPromptTimer: any = null;

  function showCollectPrompt() {
    collectPromptMessage = '需先收集';
    if (collectPromptTimer) clearTimeout(collectPromptTimer);
    collectPromptTimer = setTimeout(() => {
      collectPromptMessage = null;
    }, 2400);
  }

  function openCommentStrip() {
    isCommentStripOpen = true;
    setTimeout(() => {
      commentInputEl?.focus();
    }, 50);
  }

  function closeCommentStrip() {
    isCommentStripOpen = false;
    commentText = '';
  }

  function handleCollectClick(e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    if (isPromoting) return;

    if (promotionStrategy.hasCommentPrompt) {
      if (isCommentStripOpen) {
        closeCommentStrip();
      } else {
        openCommentStrip();
      }
    } else {
      executePromote('', card.echoOf || '');
    }
  }

  function handleCommentKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      e.stopPropagation();
      executePromote(commentText.trim(), card.echoOf || '');
    } else if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      closeCommentStrip();
    }
  }

  function handleWindowClick(e: MouseEvent) {
    if (!isCommentStripOpen) return;
    if (commentStripEl && !commentStripEl.contains(e.target as Node)) {
      closeCommentStrip();
    }
  }

  async function executePromote(comment: string = '', targetEchoOf?: string) {
    if (isPromoting) return;
    isPromoting = true;
    const oldId = card.id;
    try {
      let promoted: any = null;
      const finalEchoOf = targetEchoOf !== undefined ? targetEchoOf : (card.echoOf || '');
      if (typeof PromoteTempCard === 'function') {
        try {
          promoted = await PromoteTempCard(card.id, comment, finalEchoOf);
        } catch (ipcErr) {
          console.warn('PromoteTempCard IPC error:', ipcErr);
        }
      }

      // 原地转正 (In-place promotion)
      card.isTemporary = false;
      if (promoted) {
        if (promoted.id) card.id = promoted.id;
        if (promoted.filePath) card.filePath = promoted.filePath;
        if (promoted.type && promoted.type !== 'local') {
          card.type = promoted.type;
        } else if (card.type === 'radar') {
          card.type = 'thought';
        }
        if (promoted.origin) card.origin = promoted.origin;
        if (promoted.createdAt) card.createdAt = promoted.createdAt;
        if (promoted.echoOf !== undefined) {
          card.echoOf = promoted.echoOf;
        } else if (targetEchoOf === 'NONE') {
          delete card.echoOf;
        }
        if (promoted.userComment !== undefined) {
          card.userComment = promoted.userComment;
        } else if (comment) {
          card.userComment = comment;
        }
      } else {
        if (card.type === 'radar') card.type = 'thought';
        if (targetEchoOf === 'NONE') {
          delete card.echoOf;
        } else if (targetEchoOf) {
          card.echoOf = targetEchoOf;
        }
        if (comment) card.userComment = comment;
      }
      delete card.macroExpiresAt;
      card = { ...card };

      closeCommentStrip();
      dispatch('cardPromoted', { originalId: oldId, card });
    } catch (err) {
      console.error('Failed to promote temp card:', err);
      card.isTemporary = false;
      if (card.type === 'radar') card.type = 'thought';
      if (targetEchoOf === 'NONE') {
        delete card.echoOf;
      }
      if (comment) card.userComment = comment;
      delete card.macroExpiresAt;
      card = { ...card };
      closeCommentStrip();
      dispatch('cardPromoted', { originalId: oldId, card });
    } finally {
      isPromoting = false;
    }
  }

  // Alias handleCollect to handleCollectClick
  const handleCollect = handleCollectClick;

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
      showCollectPrompt();
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
        showCollectPrompt();
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

<svelte:window on:click={handleWindowClick} />

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div 
  class="card"
  class:inert={hasAnyFocused && !isFocused}
  class:constellation-dimmed={isConstellationDimmed}
  class:bugu-echo={isBuguEcho}
  class:bsky-echo={isBskyEcho}
  class:temporary={isTemporary}
  class:clustered-child={clusteredOverride !== null}
  class:focused={isFocused}
  class:hovered={isHovered && !isFocused}
  class:inspiration-vip={isRecentVIP && !isFocused}
  class:has-aura={hasAura}
  class:aura-stage-1={auraStage === 1}
  class:aura-stage-2={auraStage === 2}
  class:aura-stage-3={auraStage === 3}
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
    z-index: {isFocused ? 1000 : (isHovered && !hasAnyFocused ? 999 : (hasAura ? 300 : Math.round(50 - (card.z || 0) * 0.3)))};
    {dynamicBoxShadow ? `box-shadow: ${dynamicBoxShadow};` : ''}
    {hasAura ? `border: 1.5px solid ${toHsla(latestColor, auraStage === 1 ? 0.95 : (auraStage === 2 ? 0.8 : 0.65))} !important;` : ''}
    --card-text-color: rgba({Math.round($animSpring.textGray || 38)}, {Math.round($animSpring.textGray || 38)}, {Math.round($animSpring.textGray || 38)}, {($animSpring.textOpacity ?? 0.95).toFixed(2)});
    --card-text-secondary: rgba({Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {($animSpring.textOpacity ?? 0.95).toFixed(2)});
    --temp-rim-delay: -{(cardHash % 55) / 10}s;
    --vip-breathe-delay: -{(cardHash % 80) / 10}s;
    {!isVisible ? 'visibility: hidden; pointer-events: none;' : ''}
  "
>
  <CardAuraGlow {card} {hasAura} {auraStage} {conicGradient} {isBuguEcho} {isBskyEcho} isStopped={isSearchStopped} {isFocused} isRecentVIP={isRecentVIP && !isFocused} {isTemporary} />

  {#if collectPromptMessage}
    <div class="collect-prompt-toast">
      {collectPromptMessage}
    </div>
  {/if}

  {#if isTemporary && (isFocused || (isHovered && !hasAnyFocused))}
    <button
      type="button"
      class="collect-btn"
      class:focused={isFocused}
      class:scattered={!isFocused}
      class:has-comment-prompt={promotionStrategy.hasCommentPrompt}
      on:click={handleCollect}
      title={promotionStrategy.tooltip}
      aria-label={promotionStrategy.tooltip}
    >
      {#if promotionStrategy.icon === 'anchor_comment'}
        <svg
          class="collect-anchor-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <!-- Speech / Note Bubble with Anchor Pin -->
          <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" />
          <path d="M9 11h6" stroke-width="1.6" opacity="0.75" />
          <path d="M12 8v6" stroke-width="1.6" opacity="0.75" />
        </svg>
      {:else}
        <svg
          class="collect-book-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <!-- Open Book Base (Spreads & Central Spine) -->
          <path
            class="book-spine-cover"
            d="M12 19.5C8 17.5 4.5 17.5 2 18V5.5C4.5 5 8 5 12 7M12 19.5C16 17.5 19.5 17.5 22 18V5.5C19.5 5 16 5 12 7M12 19.5V7"
          />
          <!-- Book Page Subtle Text Texture -->
          <path
            class="book-lines"
            d="M6 10.5h3M6 13.5h3M15 10.5h3M15 13.5h3"
            stroke-width="1.2"
            opacity="0.38"
          />
          <!-- Adding Page Animation (往里加页动效) -->
          <g class="adding-page">
            <path
              d="M12 7C14 5 17.5 5 19.5 5.5V16.5C17.5 16 14 16 12 18V7Z"
              fill="currentColor"
              fill-opacity="0.32"
              stroke-width="1.6"
            />
            <path
              d="M14 9.5h3.5M14 12.5h2.5"
              stroke-width="1"
              opacity="0.65"
            />
          </g>
        </svg>
      {/if}
    </button>
  {/if}

  {#if isTemporary && isCommentStripOpen}
    <div
      bind:this={commentStripEl}
      class="comment-input-strip"
      class:focused-mode={isFocused}
      on:click|stopPropagation
    >
      <input
        bind:this={commentInputEl}
        type="text"
        class="comment-input"
        placeholder={promotionStrategy.placeholder || '留下附着批注（回车直接收录）...'}
        bind:value={commentText}
        on:keydown={handleCommentKeydown}
      />
      <button
        type="button"
        class="strip-btn standalone-collect-btn"
        on:click|stopPropagation={() => executePromote('', 'NONE')}
        title="作为独立卡片收录（不附着母卡）"
        aria-label="作为独立卡片收录（不附着母卡）"
      >
        <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none">
          <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
          <polyline points="15 3 21 3 21 9"></polyline>
          <line x1="10" y1="14" x2="21" y2="3"></line>
        </svg>
      </button>
      <button
        type="button"
        class="strip-btn commit-btn"
        on:click|stopPropagation={() => executePromote(commentText.trim(), card.echoOf || '')}
        title="确认附着收录 (Enter)"
        aria-label="确认附着收录"
      >
        <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2.2" fill="none">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
      </button>
    </div>
  {/if}

  <svelte:component
    this={getCardComponent(card.type)}
    {card}
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
  .card.bsky-echo:not(.focused) {
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
  .card.bsky-echo.temporary::before {
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

  /* Collect (转正) Button for Temporary Cards in Scattered & Focused States */
  .collect-btn {
    position: absolute;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    cursor: pointer;
    user-select: none;
    z-index: 100;
    transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .collect-btn.scattered {
    top: 10px;
    right: 10px;
    width: 26px;
    height: 26px;
    padding: 0;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.92);
    color: #0284c7;
    border: 1px solid rgba(56, 189, 248, 0.45);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  .collect-btn.scattered:hover {
    background: #0284c7;
    color: #ffffff;
    border-color: #0284c7;
    box-shadow: 0 4px 14px rgba(2, 132, 199, 0.38);
    transform: scale(1.08);
  }

  .collect-btn.focused {
    top: 18px;
    right: 18px;
    width: 32px;
    height: 32px;
    padding: 0;
    border-radius: 50%;
    background: rgba(2, 132, 199, 0.12);
    color: #0284c7;
    border: 1px solid rgba(2, 132, 199, 0.35);
    box-shadow: 0 2px 8px rgba(2, 132, 199, 0.12);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  .collect-btn.focused:hover {
    background: #0284c7;
    color: #ffffff;
    border-color: #0284c7;
    box-shadow: 0 4px 16px rgba(2, 132, 199, 0.42);
    transform: scale(1.08);
  }

  .collect-btn:active {
    transform: scale(0.94);
  }

  .collect-book-icon {
    display: block;
    overflow: visible;
    flex-shrink: 0;
  }

  .collect-btn.scattered .collect-book-icon {
    width: 15px;
    height: 15px;
  }

  .collect-btn.focused .collect-book-icon {
    width: 18px;
    height: 18px;
  }

  .book-spine-cover {
    transform-origin: 12px 19.5px;
    transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .collect-btn:hover .book-spine-cover {
    transform: scale(1.04);
  }

  /* 往里加页动效 (Page-Adding Settle Animation) */
  .adding-page {
    transform-origin: 12px 19.5px;
    transition: transform 0.35s cubic-bezier(0.34, 1.45, 0.7, 1), opacity 0.25s ease;
    transform: translate(3.5px, -6px) rotate(16deg) scale(0.75);
    opacity: 0;
  }

  .collect-btn:hover .adding-page {
    transform: translate(0, 0) rotate(0deg) scale(1);
    opacity: 1;
    animation: page-settle 0.45s cubic-bezier(0.34, 1.45, 0.7, 1);
  }

  @keyframes page-settle {
    0% {
      transform: translate(4.5px, -7.5px) rotate(22deg) scale(0.7);
      opacity: 0;
    }
    65% {
      transform: translate(-0.6px, 0.5px) rotate(-3deg) scale(1.03);
      opacity: 1;
    }
    100% {
      transform: translate(0, 0) rotate(0deg) scale(1);
      opacity: 1;
    }
  }

  .card.clustered-child {
    pointer-events: none !important;
  }

  .collect-anchor-icon {
    display: block;
    overflow: visible;
    flex-shrink: 0;
  }

  .collect-btn.scattered .collect-anchor-icon {
    width: 15px;
    height: 15px;
  }

  .collect-btn.focused .collect-anchor-icon {
    width: 18px;
    height: 18px;
  }

  /* Frosted Glass In-place Comment Strip (Option B) */
  .comment-input-strip {
    position: absolute;
    bottom: 10px;
    left: 10px;
    right: 10px;
    z-index: 1005;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 6px 4px 10px;
    background: rgba(255, 255, 255, 0.90);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.95);
    border-radius: 12px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.16), 0 0 0 1px rgba(0, 0, 0, 0.05);
    animation: strip-slide-up 0.2s cubic-bezier(0.16, 1, 0.3, 1);
    pointer-events: auto;
  }

  .comment-input-strip.focused-mode {
    bottom: 20px;
    left: 24px;
    right: 24px;
    padding: 6px 8px 6px 14px;
    border-radius: 14px;
  }

  @keyframes strip-slide-up {
    from {
      opacity: 0;
      transform: translateY(8px) scale(0.96);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  .comment-input {
    flex: 1;
    border: none;
    background: transparent;
    outline: none;
    font-size: 11px;
    color: #1e293b;
    padding: 4px 0;
    min-width: 0;
    font-family: inherit;
  }

  .comment-input-strip.focused-mode .comment-input {
    font-size: 13px;
  }

  .comment-input::placeholder {
    color: #94a3b8;
    font-size: 10.5px;
  }

  .comment-input-strip.focused-mode .comment-input::placeholder {
    font-size: 12px;
  }

  .strip-btn {
    width: 24px;
    height: 24px;
    border-radius: 7px;
    border: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    background: rgba(0, 0, 0, 0.04);
    color: #475569;
    transition: all 0.15s ease;
    padding: 0;
    flex-shrink: 0;
  }

  .strip-btn:hover {
    background: rgba(0, 0, 0, 0.09);
    color: #0f172a;
  }

  .strip-btn.standalone-collect-btn:hover {
    background: rgba(2, 132, 199, 0.15);
    color: #0284c7;
  }

  .strip-btn.commit-btn {
    background: #0284c7;
    color: #ffffff;
  }

  .strip-btn.commit-btn:hover {
    background: #0369a1;
  }

  /* Collect Prompt Floating Toast */
  .collect-prompt-toast {
    position: absolute;
    top: 12px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(15, 23, 42, 0.92);
    color: #f8fafc;
    font-size: 0.78rem;
    font-weight: 500;
    padding: 5px 12px;
    border-radius: 9999px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    z-index: 1002;
    pointer-events: none;
    white-space: nowrap;
    animation: toast-pop 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes toast-pop {
    from {
      opacity: 0;
      transform: translate(-50%, -6px) scale(0.96);
    }
    to {
      opacity: 1;
      transform: translate(-50%, 0) scale(1);
    }
  }
</style>
