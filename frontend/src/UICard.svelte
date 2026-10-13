<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - UICard Component Draft
  Task: omnibar_command_flow_009 & aura_search_stage_010
  
  [Bug Fix & Feature Documentation]
  1. Specific Problem Fixed / Requirements:
     - Implemented dynamic Flowing Light Aura Border for search-hit cards.
     - Supported single-color rotating neon beam and multi-color conic-gradient spinning halo.
     - Implemented 3-stage temporal decay:
       * < 3 min: Bright flowing glow with dual-layer neon box-shadow.
       * 3 ~ 30 min: Soft steady border glow with gentle rotation.
       * > 30 min: Fine crisp 1.5px colored border, subtle and elegant until memory eviction.
     - Handled search re-hits: refreshed hit time triggers immediate return to Stage 1.
     - Fixed CSS color syntax: properly converted HSL to HSLA for box-shadow alpha channels,
       preventing invalid syntax like 'hsl(...)44' in WebKit.
     - Cleaned up unused imports and variables flagged in linter.
  2. Resolution & Logic:
     - Leveraged CSS conic-gradient within an absolute spinning pseudo/inner container clipped
       by the card's rounded border radius.
     - Created an inner translucent frosted-glass mask to let only the outer 1.8px perimeter glow.
     - Computed reactive decay stage based on (Date.now() - latestHitTime).
  3. Caveats & Future Considerations:
     - Preserved card hit-testing and 3D transform performance by keeping pointer-events: none
       on aura decorative layers.
  ======================================================================================
-->
<script lang="ts">
  import { spring } from 'svelte/motion';
  import { BrowserOpenURL } from '../wailsjs/runtime/runtime.js';

  export let card: any;
  export let time: number = 0;
  export let globalTilt: { rx: number, ry: number } = { rx: 0, ry: 0 };
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

  // Dynamic box-shadow accounting for focus & aura decay stages
  $: dynamicBoxShadow = (() => {
    if (isFocused) {
      return hasAura 
        ? `0 40px 100px rgba(0, 0, 0, 0.22), 0 0 24px ${toHsla(latestColor, 0.7)}, 0 0 0 1px rgba(255, 255, 255, 0.9)`
        : '0 40px 100px rgba(0, 0, 0, 0.22), 0 0 0 1px rgba(255, 255, 255, 0.9)';
    }
    if (!hasAura) {
      return '0 8px 24px rgba(0, 0, 0, 0.04)';
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
        y: 0,
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
      // Gently keep within comfortable viewport margin on hover without shifting toward Omnibar
      const pullX = Math.max(-440, Math.min(440, card.x));
      const pullY = Math.max(-260, Math.min(260, card.y));
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
      const dist = Math.sqrt(card.x * card.x + card.y * card.y);
      const edgeFade = Math.max(0, Math.min(1, 1 - (dist - 440) / 240));
      const edgeBlur = Math.max(0, Math.min(5, (dist - 400) * 0.025));
      
      const anotherFocused = !isFocused && document.querySelector('.card.focused');
      
      if (anotherFocused) {
        animSpring.set({
          x: card.x * 1.4,
          y: card.y * 1.4,
          rotation: card.rotation,
          opacity: 0.08,
          scale: 0.7,
          blur: 10,
          saturation: 0.1,
          textGray: 150,
          textOpacity: 0.15,
          rotX: 0,
          rotY: 0,
          z3d: 0
        });
      } else {
        // At the exact same depth/distance, card size is strictly uniform
        const uniformScale = Math.max(0.55, 1 - (card.z * 0.0045));
        
        // Card container opacity and blur (capped at max 5px soft blur)
        const depthBlur = card.z > 70 ? (card.z - 70) * 0.04 : 0;
        const baseBlur = Math.min(5, card.isFresh ? edgeBlur * 0.5 : (depthBlur + edgeBlur));
        const baseOpacity = (card.isFresh ? 1.0 : (0.95 - (card.z * 0.005))) * (card.isFresh ? Math.max(0.7, edgeFade) : edgeFade);
        const baseSat = card.isFresh ? 1.0 : Math.max(0.4, 1 - (card.z * 0.006));
        // Pure continuous linear formula based strictly on depth distance Z (no discrete stages/gears):
        // Gray(Z) = 38 + (Z / 100) * (135 - 38) = 38 + 0.97 * Z
        const zNorm = Math.max(0, Math.min(1, (card.z || 0) / 100));
        const targetGray = 38 + zNorm * (135 - 38);
        const targetTextOp = 0.95;
        
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
          z3d: (100 - card.z) * 0.4 + pulseZ
        });
      }
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
  class:focused={isFocused}
  class:hovered={isHovered && !isFocused}
  class:has-aura={hasAura}
  class:aura-stage-1={auraStage === 1}
  class:aura-stage-2={auraStage === 2}
  class:aura-stage-3={auraStage === 3}
  on:click={(e) => { e.stopPropagation(); onToggleFocus(); }}
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
    {auraStage === 3 ? `border: 1.5px solid ${toHsla(latestColor, 0.75)};` : ''}
    --card-text-color: rgba({Math.round($animSpring.textGray || 38)}, {Math.round($animSpring.textGray || 38)}, {Math.round($animSpring.textGray || 38)}, {($animSpring.textOpacity ?? 0.95).toFixed(2)});
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
      {:else}
        <span></span>
      {/if}
      {#if cardTime}
        <span class="card-time">{cardTime}</span>
      {/if}
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
  {:else}
    {#if cardTime && isFocused}
      <div class="thought-header">
        <span class="card-time">{cardTime}</span>
      </div>
    {/if}
    <div class="thought-content">
      <p>{card.content}</p>
    </div>
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
    background: rgba(255, 255, 255, 0.7);
    border-radius: 16px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.04);
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
    cursor: pointer;
    transform-origin: center center;
    user-select: none;
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.5);
    overflow: hidden;
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
  .thought-content {
    position: relative;
    z-index: 1;
  }

  .babel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
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

  .card-time {
    font-size: 0.7rem;
    color: rgba(0, 0, 0, 0.42);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.02em;
    user-select: none;
    white-space: nowrap;
  }

  .thought-header {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 6px;
    width: 100%;
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
    overflow-y: auto;
    max-height: 400px;
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
    overflow-y: auto;
  }

  .card.focused .thought-content {
    justify-content: center;
    height: 100%;
    overflow-y: auto;
    text-align: left;
  }

  .card.focused .thought-content p {
    font-size: 1.15rem;
    line-height: 1.75;
    display: block;
    -webkit-line-clamp: unset;
    overflow: visible;
    text-align: left;
    color: rgba(32, 32, 32, 0.98);
  }

  .card.focused .babel-content {
    justify-content: flex-start;
    height: 100%;
    overflow-y: auto;
    text-align: left;
  }

  .card.focused .twinkle-container,
  .card.focused .single-container {
    height: auto;
    max-height: none;
    overflow: visible;
  }

  .card.focused .original,
  .card.focused .translated,
  .card.focused .pure-original {
    -webkit-line-clamp: unset !important;
    display: block !important;
    overflow: visible !important;
    text-overflow: unset !important;
    max-height: none !important;
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

  .thought-content {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: center; /* Vertical center */
    height: 100%;
    width: 100%;
    overflow: hidden;
    text-align: left;
  }

  .thought-content p {
    margin: 0;
    font-size: 0.825rem;
    line-height: 1.55;
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
    font-weight: 450;
    letter-spacing: 0.01em;
    display: -webkit-box;
    -webkit-line-clamp: 6;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    word-break: break-word;
    width: 100%;
    text-align: left;
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
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
    opacity: 0.75;
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
  
  /* Global sleek scrollbar for cards and other elements */
  ::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }
  ::-webkit-scrollbar-track {
    background: transparent;
    margin: 8px 0; 
  }
  ::-webkit-scrollbar-thumb {
    background: rgba(0, 0, 0, 0.15);
    border-radius: 10px;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: rgba(0, 0, 0, 0.3);
  }
</style>
