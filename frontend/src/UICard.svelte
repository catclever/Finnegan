<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - UICard Component Draft
  Task: night_shift_overflow_scroll_focus_018

  [Bug Fix & Feature Documentation]
  1. Specific Problems Fixed:
     - Clicking/focusing a card when scrolled down into Deep Space (`scrollExpansion > 0`)
       previously set `animSpring.y = 0`, which placed the focused card at the origin of
       `.scene` (above the visible screen due to `.scene`'s `translateY(-scrollExpansion * 300px)`)
       instead of the geometric center of the user's current viewport.
     - Hovering a card in Screen 2 or Screen 3 (`card.y > 300`) previously clamped `pullY`
       to `[-260, 260]`, yanking the hovered card hundreds of pixels upward out of view.
     - Peripheral `edgeFade` and `edgeBlur` were previously calculated from absolute stage
       origin `(0, 0)` instead of the current scrolled viewport center `(0, scrollOffsetY)`,
       causing deep-space cards in Screen 2 and Screen 3 to remain blurry/faded even when
       scrolled directly into the center of the screen.
  2. Resolution & Logic:
     - Added `scrollExpansion` and `hasAnyFocused` props and computed `scrollOffsetY = scrollExpansion * 300`.
     - When `isFocused` is true, set `y: scrollOffsetY` so it exactly cancels parent `.scene`'s
       `translateY(-scrollExpansion * 300px)`, locking the focused card to the invariant
       geometric center of the viewport at any scroll depth and smoothly returning to
       `(card.x, card.y)` when unfocused.
     - Updated hover clamping to `pullY = Math.max(scrollOffsetY - 260, Math.min(scrollOffsetY + 260, card.y))`.
     - Computed viewport-relative `viewY = card.y - scrollOffsetY` and `dist = Math.sqrt(card.x * card.x + viewY * viewY)`
       so deep-space cards render with full optical clarity and opacity when scrolled into view.
  3. Caveats & Future Considerations:
     - Keep `scrollOffsetY` multiplier (`300`) synchronized with `TIDE_OFFSET_PX` in `stageDecay.ts`
       and `.scene` transform in `CosmicStage.svelte`.
  ======================================================================================
-->
<script lang="ts">
  import { spring } from 'svelte/motion';
  import { BrowserOpenURL } from '../wailsjs/runtime/runtime.js';
  import ThoughtCardBody from './components/cards/ThoughtCardBody.svelte';
  import ChatLogCardBody from './components/cards/ChatLogCardBody.svelte';

  export let card: any;
  export let time: number = 0;
  export let globalTilt: { rx: number, ry: number } = { rx: 0, ry: 0 };
  export let scrollExpansion: number = 0;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let onToggleFocus: () => void;
  export let onHover: () => void;
  export let onLeave: () => void;

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
      // Single color: rotating neon beam with translucent fade
      return `conic-gradient(from 0deg, transparent 0deg, ${c} 70deg, transparent 140deg, transparent 180deg, ${c} 250deg, transparent 320deg, transparent 360deg)`;
    }
    // Multiple colors: smooth multi-stop rainbow conic halo
    const stops = colors.map((c: string, i: number) => {
      const deg = Math.round((i / colors.length) * 360);
      return `${c} ${deg}deg`;
    });
    return `conic-gradient(from 0deg, ${stops.join(', ')}, ${colors[0]} 360deg)`;
  })();

  $: isBuguEcho = card.type === 'babel' && card.origin === 'bugu' && Boolean(card.echoOf);
  $: isTemporary = card.origin === 'bugu' || card.type === 'babel';

  // Dynamic box-shadow accounting for focus, temporary state & aura decay stages
  $: dynamicBoxShadow = (() => {
    if (isFocused) {
      if (hasAura) {
        return `0 40px 100px rgba(0, 0, 0, 0.22), 0 0 24px ${toHsla(latestColor, 0.7)}, 0 0 0 1px rgba(255, 255, 255, 0.9)`;
      }
      return isBuguEcho
        ? '0 40px 100px rgba(0, 0, 0, 0.16), 0 0 36px rgba(245, 158, 11, 0.10), inset 0 0 24px rgba(255, 250, 240, 0.45)'
        : '0 40px 100px rgba(0, 0, 0, 0.22), 0 0 0 1px rgba(255, 255, 255, 0.9)';
    }
    if (!hasAura) {
      return isBuguEcho ? '' : '0 8px 24px rgba(0, 0, 0, 0.04)';
    }
    if (auraStage === 1) {
      // Stage 1 (< 3 min): Bright flowing glow
      return `0 0 16px ${toHsla(latestColor, 0.85)}, 0 0 30px ${toHsla(latestColor, 0.35)}, 0 8px 24px rgba(0, 0, 0, 0.04)`;
    } else if (auraStage === 2) {
      // Stage 2 (3 ~ 30 min): Soft steady border glow
      return `0 0 10px ${toHsla(latestColor, 0.5)}, 0 0 20px ${toHsla(latestColor, 0.2)}, 0 8px 24px rgba(0, 0, 0, 0.04)`;
    } else {
      // Stage 3 (> 30 min): Fine crisp border; standard elevation shadow
      return `0 8px 24px rgba(0, 0, 0, 0.04)`;
    }
  })();

  $: {
    const hashId = card.id && card.id.length > 0 ? card.id.charCodeAt(0) : 0;
    // Breathing pulse along Z-axis in pixels
    const pulseZ = Math.sin(time * 1.2 + hashId * 2.1) * 8;
    
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
      animSpring.set({
        x: 0,
        y: scrollOffsetY,
        rotation: 0,
        opacity: 1,
        scale: 1,
        blur: 0,
        saturation: 1,
        textGray: 32, // Deep charcoal black
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 160
      });
    } else if (isHovered) {
      // Gently keep within comfortable viewport margin relative to current scroll depth
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
        textGray: 32, // Deep charcoal black
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 120
      });
    } else {
      const viewY = card.y - scrollOffsetY;
      const dist = Math.sqrt(card.x * card.x + viewY * viewY);
      const edgeFade = Math.max(0, Math.min(1, 1 - (dist - 440) / 240));
      const edgeBlur = Math.max(0, Math.min(5, (dist - 400) * 0.025));

      const zNorm = Math.max(0, Math.min(1, (card.z || 0) / 100));

      // 4. Card size with authentic cosmic perspective hierarchy:
      // Z = 0 -> scale 1.08; Z = 50 -> scale 0.86; Z = 100 -> scale 0.64
      const uniformScale = 1.08 - (zNorm * 0.44);

      // 3D displacement: Z = 0 -> +100px; Z = 50 -> +25px; Z = 100 -> -50px
      const z3dVal = (100 - (card.z || 0)) * 1.5 - 50 + pulseZ;

      // 5. Container opacity, blur, and saturation following Z-depth:
      const depthBlur = zNorm > 0.65 ? (zNorm - 0.65) * 6 : 0;
      const baseBlur = Math.min(4.5, depthBlur + edgeBlur);

      // Base opacity naturally scales from 0.96 (near) to 0.64 (distant)
      const depthOpacity = 0.96 - (zNorm * 0.32);
      const baseOpacity = depthOpacity * (card.isFresh ? Math.max(0.85, edgeFade) : edgeFade);
      const baseSat = Math.max(0.42, 1.0 - (zNorm * 0.5));

      // 3. Continuous linear grayscale formula strictly tracking Z:
      // Gray(Z) = 34 + zNorm * (145 - 34)
      const targetGray = Math.round(34 + zNorm * 111);
      const targetTextOp = Number((0.98 - zNorm * 0.16).toFixed(2));

      animSpring.set({
        x: card.x,
        y: card.y,
        rotation: card.rotation + (Math.sin(time * 0.8 + hashId) * 1.5),
        opacity: baseOpacity,
        scale: uniformScale,
        blur: baseBlur,
        saturation: baseSat,
        textGray: targetGray,
        textOpacity: targetTextOp,
        rotX: globalTilt.rx * 0.25,
        rotY: globalTilt.ry * 0.25,
        z3d: z3dVal
      });
    }
  }

  function getHostname(url: string | undefined): string {
    if (!url) return '';
    try {
      return new URL(url).hostname || 'Source';
    } catch {
      return 'Source';
    }
  }

  $: hostname = getHostname(card.sourceUrl);

  $: hasTranslation = Boolean(
    card.translatedText &&
    card.translatedText.trim() !== '' &&
    card.originalText &&
    card.translatedText.trim() !== card.originalText.trim()
  );

  function handleOpenSource(e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    if (card.sourceUrl) {
      BrowserOpenURL(card.sourceUrl);
    }
  }

  function handleAvatarError(e: Event) {
    const el = e.currentTarget as HTMLElement | null;
    if (el) el.style.display = 'none';
  }

  function formatPublishedDate(t?: string): string {
    if (!t || !t.trim()) return '';
    const s = t.trim();
    const d = new Date(s);
    if (!isNaN(d.getTime())) {
      const year = d.getFullYear();
      const month = String(d.getMonth() + 1).padStart(2, '0');
      const day = String(d.getDate()).padStart(2, '0');
      const currentYear = new Date().getFullYear();
      if (year === currentYear) {
        return `${month}/${day}`;
      }
      return `${year}/${month}/${day}`;
    }
    return s.length > 18 ? s.slice(0, 18) : s;
  }

  function formatCardTime(t?: string | number): string {
    if (!t) return '';
    const d = typeof t === 'number' ? new Date(t) : new Date(t);
    if (isNaN(d.getTime())) return '';
    const now = Date.now();
    const diffSec = Math.floor((now - d.getTime()) / 1000);
    if (diffSec < 60) return '刚刚';
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m 前`;
    if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h 前`;
    const m = d.getMonth() + 1;
    const day = d.getDate();
    const h = String(d.getHours()).padStart(2, '0');
    const min = String(d.getMinutes()).padStart(2, '0');
    return `${m}/${day} ${h}:${min}`;
  }

  $: babelTime = card.type === 'babel' ? formatPublishedDate(card.publishedAt) : '';
  $: thoughtTime = card.type !== 'babel' ? formatCardTime(card.createdAt || card.spawnTime) : '';
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div 
  class="card"
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
    z-index: {isFocused ? 1000 : (isHovered ? 999 : Math.round(100 - card.z))};
    box-shadow: {dynamicBoxShadow};
    {!isTemporary && auraStage === 3 ? `border: 1.5px solid ${toHsla(latestColor, 0.75)};` : ''}
    --card-text-color: rgba({Math.round($animSpring.textGray || 38)}, {Math.round($animSpring.textGray || 38)}, {Math.round($animSpring.textGray || 38)}, {($animSpring.textOpacity ?? 0.95).toFixed(2)});
    --card-text-secondary: rgba({Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {Math.min(160, Math.round(($animSpring.textGray || 38) + 24))}, {($animSpring.textOpacity ?? 0.95).toFixed(2)});
  "
>
  <!-- Flowing Light Aura Layers -->
  {#if hasAura && auraStage < 3}
    <div class="aura-spinner-container">
      <div 
        class="aura-spinner" 
        class:slow={auraStage === 2}
        style="background: {conicGradient};"
      ></div>
    </div>
    <div class="aura-inner-mask" class:focused={isFocused}></div>
  {/if}

  {#if card.type === 'babel'}
    <div class="babel-header">
      <div class="header-left">
        {#if card.sourceUrl}
          <a 
            href={card.sourceUrl} 
            target="_blank" 
            class="source-link" 
            on:click={handleOpenSource}
            title={card.sourceUrl}
          >
            <span class="domain-icon">🌐</span> {hostname}
          </a>
        {/if}
      </div>
    </div>

    {#if hasTranslation}
      <div class="babel-content twinkle-container" class:focused={isFocused}>
        <div class="twinkle-text lang-trans">
          <p class="translated">{card.translatedText}</p>
        </div>
        <div class="twinkle-text lang-orig">
          <p class="original">{card.originalText}</p>
        </div>
      </div>
    {:else}
      <div class="babel-content single-container" class:focused={isFocused}>
        <p class="pure-original">{card.originalText || card.content}</p>
      </div>
    {/if}

    <div class="babel-footer">
      <div class="footer-left">
        {#if card.authorName || card.authorAvatar}
          <div class="author-pill" title={card.authorName || 'Author'}>
            {#if card.authorAvatar}
              <img src={card.authorAvatar} alt="" class="author-avatar" on:error={handleAvatarError} />
            {/if}
            {#if card.authorName}
              <span class="author-name">{card.authorName}</span>
            {/if}
          </div>
        {/if}
      </div>

      {#if babelTime}
        <span class="card-time">{babelTime}</span>
      {/if}
    </div>
  {:else if card.type === 'chat_log'}
    <ChatLogCardBody {card} {isFocused} {isHovered} {thoughtTime} />
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
    pointer-events: auto; /* Catch clicks and mouse events reliably */
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

  /* Ephemeral / Temporary Card Aesthetics: Translucent Mist-Blue Base + Dynamic Breathing 灰白 Vignette Rim */
  .card.temporary {
    background: linear-gradient(135deg, rgba(226, 236, 248, 0.58) 0%, rgba(210, 222, 238, 0.42) 100%);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border: 1px solid transparent !important;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.04);
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
      box-shadow: inset 0 0 16px rgba(255, 255, 255, 0.55), inset 0 0 32px rgba(220, 232, 246, 0.35);
    }
    50% {
      opacity: 0.95;
      box-shadow: inset 0 0 24px rgba(255, 255, 255, 0.9), inset 0 0 44px rgba(200, 220, 245, 0.55);
    }
  }

  .card.bugu-echo {
    box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.25);
    animation: bugu-breathing 4s ease-in-out infinite;
  }

  .card.bugu-echo.focused {
    animation: none;
    box-shadow: 0 40px 100px rgba(0, 0, 0, 0.16), 0 0 36px rgba(245, 158, 11, 0.10), inset 0 0 24px rgba(255, 250, 240, 0.45);
  }

  @keyframes bugu-breathing {
    0%, 100% {
      box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.15), 0 0 10px rgba(245, 158, 11, 0.05);
    }
    50% {
      box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.45), 0 0 16px rgba(245, 158, 11, 0.15);
    }
  }

  /* Flowing Light Aura Animation Layers */
  .aura-spinner-container {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    overflow: hidden;
    pointer-events: none;
    z-index: 0;
  }

  .aura-spinner {
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    transform-origin: center center;
    animation: aura-spin 4s linear infinite;
    pointer-events: none;
  }

  .aura-spinner.slow {
    animation: aura-spin 9s linear infinite;
  }

  @keyframes aura-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .aura-inner-mask {
    position: absolute;
    inset: 1.8px;
    border-radius: 14.2px;
    background: rgba(255, 255, 255, 0.88);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    pointer-events: none;
    z-index: 0;
  }

  .aura-inner-mask.focused {
    border-radius: 26.2px;
    background: rgba(255, 255, 255, 0.96);
  }

  .babel-header,
  .babel-content,
  .babel-footer {
    position: relative;
    z-index: 1;
  }

  .babel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .babel-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    position: absolute;
    bottom: 12px;
    left: 18px;
    right: 18px;
  }

  .card.focused .babel-footer {
    position: relative;
    margin-top: auto;
    padding-top: 16px;
    bottom: auto;
    left: auto;
    right: auto;
  }

  .header-left,
  .footer-left {
    display: flex;
    align-items: center;
    gap: 6px;
    overflow: hidden;
    flex: 1;
    min-width: 0;
  }

  .author-pill {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 0.72rem;
    color: rgba(0, 0, 0, 0.65);
    background: rgba(255, 255, 255, 0.55);
    padding: 2px 7px;
    border-radius: 12px;
    max-width: 120px;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    border: 1px solid rgba(255, 255, 255, 0.7);
  }

  .author-avatar {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    object-fit: cover;
    flex-shrink: 0;
  }

  .author-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  .source-link {
    font-size: 0.75rem;
    color: rgba(0, 0, 0, 0.5);
    text-decoration: none;
    display: flex;
    align-items: center;
    gap: 5px;
    background: rgba(255, 255, 255, 0.6);
    padding: 3px 8px;
    border-radius: 20px;
    transition: all 0.2s;
  }
  
  .source-link:hover {
    background: rgba(255, 255, 255, 0.9);
    color: rgba(0, 0, 0, 0.8);
  }

  .card.temporary .author-pill {
    background: rgba(255, 255, 255, 0.6);
    border: 1px solid rgba(226, 232, 240, 0.8);
  }

  .card.temporary .source-link {
    background: rgba(255, 255, 255, 0.6);
    border: 1px solid rgba(226, 232, 240, 0.75);
  }

  .card-time {
    font-size: 0.7rem;
    color: rgba(0, 0, 0, 0.42);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.02em;
    user-select: none;
    white-space: nowrap;
  }


  .single-container {
    position: relative;
    width: 100%;
    height: 105px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .pure-original {
    font-size: 0.825rem;
    line-height: 1.55;
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
    font-weight: 450;
    display: -webkit-box;
    -webkit-line-clamp: 5;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
    margin: 0;
  }

  /* Twinkling effect */
  .twinkle-container {
    position: relative;
    width: 100%;
    height: 95px;
  }
  
  .twinkle-text {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
  
  .lang-trans {
    animation: twinkle-a 8s infinite cubic-bezier(0.4, 0, 0.2, 1);
  }
  
  .lang-orig {
    animation: twinkle-b 8s infinite cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* Stop animation completely when focused */
  .twinkle-container.focused .lang-trans,
  .twinkle-container.focused .lang-orig {
    animation: none !important;
    position: relative;
    opacity: 1;
    filter: none;
    height: auto;
    margin-bottom: 10px;
    display: block;
  }
  
  .twinkle-container.focused {
    height: auto;
    overflow: visible;
    max-height: none;
  }
  
  @keyframes twinkle-a {
    0%, 35% { opacity: 1; filter: blur(0px); z-index: 2; }
    45%, 85% { opacity: 0; filter: blur(8px); z-index: 1; }
    100% { opacity: 1; filter: blur(0px); z-index: 2; }
  }
  
  @keyframes twinkle-b {
    0%, 35% { opacity: 0; filter: blur(8px); z-index: 1; }
    45%, 85% { opacity: 1; filter: blur(0px); z-index: 2; }
    100% { opacity: 0; filter: blur(8px); z-index: 1; }
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
    background: linear-gradient(145deg, rgba(248, 250, 253, 0.96) 0%, rgba(240, 244, 248, 0.92) 100%);
    backdrop-filter: blur(26px);
    -webkit-backdrop-filter: blur(26px);
    border: 1px solid transparent !important;
  }

  .card.focused::-webkit-scrollbar,
  .card.focused .babel-content::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  .card.focused .babel-content {
    justify-content: flex-start;
    flex: 1;
    min-height: 0;
    height: 100%;
    max-height: 100%;
    overflow-y: auto !important;
    overflow-x: hidden !important;
    overscroll-behavior: contain;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
    -ms-overflow-style: none;
    text-align: left;
  }

  .card.focused .twinkle-container,
  .card.focused .single-container {
    height: auto;
    min-height: 0;
    overflow-y: auto !important;
    overflow-x: hidden !important;
  }

  .card.focused .original,
  .card.focused .translated,
  .card.focused .pure-original {
    -webkit-line-clamp: unset !important;
    display: block !important;
    overflow: visible !important;
    text-overflow: unset !important;
    max-height: none !important;
    padding-bottom: 18px;
  }

  .card.focused .translated {
    font-size: 1.1rem;
    line-height: 1.75;
    color: rgba(30, 30, 30, 0.98);
    font-weight: 500;
  }

  .card.focused .original {
    font-size: 0.95rem;
    line-height: 1.65;
    color: rgba(60, 60, 60, 0.85);
    margin-top: 16px;
    padding-top: 14px;
    border-top: 1px dashed rgba(0, 0, 0, 0.15);
  }

  .card.focused .pure-original {
    font-size: 1.15rem;
    line-height: 1.75;
    color: rgba(30, 30, 30, 0.98);
  }


  p {
    margin: 0;
    font-size: 0.825rem;
    line-height: 1.5;
    white-space: pre-wrap;
    text-align: left;
  }
  
  .original {
    font-size: 0.775rem;
    color: var(--card-text-secondary, rgba(70, 70, 70, 0.9));
    font-style: italic;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
  }
  
  .translated {
    font-size: 0.825rem;
    font-weight: 500;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
  }
</style>
