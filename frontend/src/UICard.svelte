<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - UICard Component
  Physical shell, 3D spring coordinate container, and card body delegator.
  ======================================================================================
-->
<script lang="ts">
  import { spring } from 'svelte/motion';
  import { getCardDisplayTime } from './lib/time';
  import ThoughtCardBody from './components/cards/ThoughtCardBody.svelte';
  import ChatLogCardBody from './components/cards/ChatLogCardBody.svelte';
  import NoteCardBody from './components/cards/NoteCardBody.svelte';
  import BabelCardBody from './components/cards/BabelCardBody.svelte';
  import CardAuraGlow from './components/cards/CardAuraGlow.svelte';

  export let card: any;

  export let globalTilt: { rx: number, ry: number } = { rx: 0, ry: 0 };
  export let scrollExpansion: number = 0;
  export let isFocused: boolean = false;
  export let hasAnyFocused: boolean = false;
  export let isHovered: boolean = false;
  export let onToggleFocus: () => void;
  export let onHover: () => void;
  export let onLeave: () => void;
  export let focusNavDirection: 'prev' | 'next' | null = null;
  export let hoveredMotherId: string | null = null;

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
  $: auraTraces = card.auraTraces || [];
  $: hasAura = auraTraces.length > 0;

  // Find latest hit time and most recent aura color
  $: latestHitTime = hasAura 
    ? Math.max(...auraTraces.map((t: any) => t.hitTime || 0)) 
    : 0;

  $: latestColor = hasAura 
    ? (auraTraces[auraTraces.length - 1]?.color || 'hsl(180, 95%, 62%)') 
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
  // 1: < 3 min (180,000 ms)  - Bright flowing neon glow
  // 2: 3 ~ 30 min (1.8m ms)  - Soft steady border glow
  // 3: > 30 min (> 1.8m ms)  - Fine crisp 1.5px colored border
  $: auraStage = !hasAura ? 0 : (elapsed < 3 * 60 * 1000 ? 1 : (elapsed < 30 * 60 * 1000 ? 2 : 3));

  // Dynamic conic-gradient for spinning border
  $: conicGradient = (() => {
    if (!hasAura) return '';
    const colors = auraTraces.map((t: any) => t.color).filter(Boolean);
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

  $: isBuguEcho = card.type === 'babel' && Boolean(card.echoOf);
  $: isTemporary = card.origin === 'bugu' || card.type === 'babel';

  // Dynamic box-shadow accounting for focus, temporary state & aura decay stages
  $: dynamicBoxShadow = (() => {
    if (isFocused) {
      if (hasAura) {
        return `0 0 0 1.5px ${toHsla(latestColor, 0.95)}, 0 0 28px ${toHsla(latestColor, 0.7)}, 0 40px 100px rgba(0, 0, 0, 0.22)`;
      }
      return isBuguEcho
        ? '0 0 0 1px rgba(245, 158, 11, 0.9), 0 0 8px rgba(245, 158, 11, 0.55), 0 0 20px rgba(245, 158, 11, 0.35), 0 0 45px rgba(245, 158, 11, 0.18), 0 0 75px rgba(245, 158, 11, 0.06), 0 40px 100px rgba(0, 0, 0, 0.22), inset 0 0 16px rgba(254, 243, 199, 0.4)'
        : '0 40px 100px rgba(0, 0, 0, 0.22), 0 0 0 1px rgba(255, 255, 255, 0.9)';
    }
    if (!hasAura) {
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
        scale: 1.05,
        blur: 0,
        saturation: 1,
        textGray: 32,
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 120
      });

    } else if (!hasAnyFocused && hoveredMotherId !== null && card.echoOf === hoveredMotherId) {
      // Mother card hovered: straighten up child cards (rotation: 0)
      const viewY = card.y - scrollOffsetY;
      const dist = Math.sqrt(card.x * card.x + viewY * viewY);
      const edgeFade = Math.max(0, Math.min(1, 1 - (dist - 440) / 240));
      const edgeBlur = Math.max(0, Math.min(5, (dist - 400) * 0.025));

      const zNorm = Math.max(0, Math.min(1, (card.z || 0) / 100));
      const uniformScale = 1.08 - (zNorm * 0.44);
      const z3dVal = (100 - (card.z || 0)) * 1.5 - 50;

      const depthBlur = zNorm > 0.65 ? (zNorm - 0.65) * 6 : 0;
      const baseBlur = Math.min(4.5, depthBlur + edgeBlur);

      const depthOpacity = 0.96 - (zNorm * 0.32);
      const baseOpacity = depthOpacity * (card.isFresh ? Math.max(0.85, edgeFade) : edgeFade);
      const baseSat = Math.max(0.42, 1.0 - (zNorm * 0.5));

      const targetGray = Math.round(34 + zNorm * 111);
      const targetTextOp = Number((0.98 - zNorm * 0.16).toFixed(2));

      animSpring.set({
        x: card.x,
        y: card.y,
        rotation: 0,
        opacity: hasAnyFocused ? baseOpacity * 0.35 : baseOpacity,
        scale: uniformScale,
        blur: hasAnyFocused ? baseBlur + 10 : baseBlur,
        saturation: hasAnyFocused ? baseSat * 0.5 : baseSat,
        textGray: targetGray,
        textOpacity: targetTextOp,
        rotX: globalTilt.rx * 0.25,
        rotY: globalTilt.ry * 0.25,
        z3d: z3dVal
      });
    } else {
      const viewY = card.y - scrollOffsetY;
      const dist = Math.sqrt(card.x * card.x + viewY * viewY);
      const edgeFade = Math.max(0, Math.min(1, 1 - (dist - 440) / 240));
      const edgeBlur = Math.max(0, Math.min(5, (dist - 400) * 0.025));

      const zNorm = Math.max(0, Math.min(1, (card.z || 0) / 100));
      const uniformScale = 1.08 - (zNorm * 0.44);
      const z3dVal = (100 - (card.z || 0)) * 1.5 - 50;

      const depthBlur = zNorm > 0.65 ? (zNorm - 0.65) * 6 : 0;
      const baseBlur = Math.min(4.5, depthBlur + edgeBlur);

      const depthOpacity = 0.96 - (zNorm * 0.32);
      const baseOpacity = depthOpacity * (card.isFresh ? Math.max(0.85, edgeFade) : edgeFade);
      const baseSat = Math.max(0.42, 1.0 - (zNorm * 0.5));

      const targetGray = Math.round(34 + zNorm * 111);
      const targetTextOp = Number((0.98 - zNorm * 0.16).toFixed(2));

      animSpring.set({
        x: card.x,
        y: card.y,
        rotation: card.rotation,
        opacity: hasAnyFocused ? baseOpacity * 0.35 : baseOpacity,
        scale: uniformScale,
        blur: hasAnyFocused ? baseBlur + 10 : baseBlur,
        saturation: hasAnyFocused ? baseSat * 0.5 : baseSat,
        textGray: targetGray,
        textOpacity: targetTextOp,
        rotX: globalTilt.rx * 0.25,
        rotY: globalTilt.ry * 0.25,
        z3d: z3dVal
      });
    }
  }

  $: babelTime = card.type === 'babel' ? getCardDisplayTime(card) : '';
  $: thoughtTime = card.type !== 'babel' ? getCardDisplayTime(card) : '';
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div 
  class="card"
  class:inert={hasAnyFocused && !isFocused}
  class:bugu-echo={isBuguEcho}
  class:temporary={isTemporary}
  class:focused={isFocused}
  class:hovered={isHovered && !isFocused}
  class:has-aura={hasAura}
  class:aura-stage-1={auraStage === 1}
  class:aura-stage-2={auraStage === 2}
  class:aura-stage-3={auraStage === 3}
  on:click={(e) => { e.stopPropagation(); onToggleFocus(); }}
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
    {!isVisible ? 'visibility: hidden; pointer-events: none;' : ''}
  "
>
  <CardAuraGlow {card} {hasAura} {auraStage} {conicGradient} {isBuguEcho} {isFocused} />

  {#if card.type === 'babel'}
    <BabelCardBody {card} {isFocused} {babelTime} />
  {:else if card.type === 'chat_log'}
    <ChatLogCardBody {card} {isFocused} {isHovered} {thoughtTime} />
  {:else if card.type === 'note'}
    <NoteCardBody {card} {isFocused} {isHovered} {thoughtTime} />
  {:else}
    <ThoughtCardBody {card} {isFocused} {isHovered} {thoughtTime} />
  {/if}
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
  }

  @keyframes temporary-rim-breathe {
    0%, 100% {
      opacity: 0.65;
    }
    50% {
      opacity: 0.95;
    }
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

  .card.has-aura.temporary::before {
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
</style>
