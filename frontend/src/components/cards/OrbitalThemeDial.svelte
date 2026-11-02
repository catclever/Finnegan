<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - OrbitalThemeDial Subcomponent
  Task: theme_carousel_mode_150
  Layer 3 Autonomous Feature Component (Single-Responsibility)
  Blooms an inward circular arc of theme buttons around .nav-left or .nav-right on hover.
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher, onDestroy } from 'svelte';
  import type { ThemeThread } from '../../types/card';
  import { 
    computeOrbitalButtonLayout, 
    getThemePalette, 
    findAvailableTheme, 
    recordThemeEmit 
  } from '../../lib/themeOrbit';

  export let themes: ThemeThread[] = [];
  export let activeThemeId: string | null = null;
  export let isRightSide: boolean = false;
  export let isNavHovered: boolean = false;
  export let onToggleTheme: ((themeId: string) => void) | undefined = undefined;

  const dispatch = createEventDispatcher();

  let hoveredThemeId: string | null = null;
  let isOpenState: boolean = false;
  let closeTimer: any = null;

  $: {
    if (isNavHovered) {
      if (closeTimer) {
        clearTimeout(closeTimer);
        closeTimer = null;
      }
      isOpenState = true;
    } else if (isOpenState) {
      if (!closeTimer) {
        closeTimer = setTimeout(() => {
          isOpenState = false;
          closeTimer = null;
        }, 350);
      }
    }
  }

  $: isOpen = isOpenState;

  $: positions = computeOrbitalButtonLayout(themes.length, isRightSide, 72);

  $: singleTheme = (themes && themes.length === 1) ? themes[0] : null;
  $: singleThemePal = singleTheme ? getThemePalette(singleTheme.id, 0) : null;
  $: isSingleThemeHovered = themes.length === 1 && isNavHovered && activeThemeId === null;

  interface SpeechBarrageItem {
    key: number;
    themeId: string;
    title: string;
    count: number;
    offsetX: number;
    color: string;
    glow: string;
    bgGradient: string;
    border: string;
    text: string;
  }

  let activeBarrage: SpeechBarrageItem | null = null;
  let barrageCounter = 0;
  let barrageTimer: any = null;
  let clearBarrageTimer: any = null;
  let previousThemesSig = '';

  function stopBarrageLoop() {
    if (barrageTimer) {
      clearTimeout(barrageTimer);
      barrageTimer = null;
    }
    if (clearBarrageTimer) {
      clearTimeout(clearBarrageTimer);
      clearBarrageTimer = null;
    }
  }

  function scheduleNextBarrage(isInitial: boolean = false, bypassCooldown: boolean = false) {
    if (barrageTimer) clearTimeout(barrageTimer);
    if (!themes || themes.length === 0) return;

    // Check interval: initial staggered delay, then repeat smoothly
    const baseDelay = isInitial
      ? (isRightSide ? 1400 + Math.random() * 400 : 500 + Math.random() * 300)
      : 3000 + Math.random() * 1500;

    barrageTimer = setTimeout(() => {
      if (!themes || themes.length === 0 || activeThemeId !== null) {
        scheduleNextBarrage(false);
        return;
      }

      // 30s Cooldown Check & Non-consecutive check (bypassed on card initial focus)
      const targetTheme = findAvailableTheme(themes, Date.now(), bypassCooldown);
      if (!targetTheme) {
        // All themes are currently on 30s cooldown; check again soon
        scheduleNextBarrage(false);
        return;
      }

      // Record emit timestamp so it enters 30s cooldown
      recordThemeEmit(targetTheme.id);

      const idx = themes.findIndex(t => t.id === targetTheme.id);
      const pal = getThemePalette(targetTheme.id, idx >= 0 ? idx : 0);
      const jitterX = Math.round((Math.random() - 0.5) * 12);

      barrageCounter += 1;
      activeBarrage = {
        key: barrageCounter,
        themeId: targetTheme.id,
        title: (targetTheme.title || '主题').trim(),
        count: targetTheme.cards?.length || 1,
        offsetX: jitterX,
        color: pal.primary,
        glow: pal.glow,
        bgGradient: pal.bgGradient,
        border: pal.border,
        text: pal.text
      };

      if (clearBarrageTimer) clearTimeout(clearBarrageTimer);
      clearBarrageTimer = setTimeout(() => {
        activeBarrage = null;
      }, 2600);

      scheduleNextBarrage(false);
    }, baseDelay);
  }

  $: currentThemesSig = themes ? themes.map(t => t.id).sort().join(',') : '';

  $: if (currentThemesSig !== previousThemesSig) {
    previousThemesSig = currentThemesSig;
    stopBarrageLoop();
    activeBarrage = null;
    if (themes && themes.length > 0 && activeThemeId === null) {
      scheduleNextBarrage(true, true);
    }
  }

  $: if (activeThemeId !== null) {
    activeBarrage = null;
    stopBarrageLoop();
  } else if (themes && themes.length > 0) {
    if (!barrageTimer && !activeBarrage) {
      scheduleNextBarrage(true);
    }
  } else {
    activeBarrage = null;
    stopBarrageLoop();
  }

  function handleButtonClick(themeId: string, e: MouseEvent) {
    e.stopPropagation();
    if (onToggleTheme) {
      onToggleTheme(themeId);
    }
    dispatch('toggleTheme', themeId);
  }

  function getButtonLabel(title: string): string {
    const clean = (title || '').trim();
    if (!clean) return '主题';
    return clean.slice(0, 3);
  }

  onDestroy(() => {
    if (closeTimer) clearTimeout(closeTimer);
    stopBarrageLoop();
  });
</script>

{#if themes && themes.length > 0}
  <!-- Autonomous Rising Speech Bubble (Continuous linear rise, pauses on hover) -->
  {#if activeBarrage}
    {#key activeBarrage.key}
      <button
        type="button"
        class="nav-speech-barrage"
        class:right-side={isRightSide}
        style="
          --barrage-primary: {activeBarrage.color};
          --barrage-glow: {activeBarrage.glow};
          --barrage-bg: {activeBarrage.bgGradient};
          --barrage-border: {activeBarrage.border};
          --barrage-text: {activeBarrage.text};
          --barrage-offset-x: {activeBarrage.offsetX}px;
        "
        on:click={(e) => handleButtonClick(activeBarrage.themeId, e)}
        title={`进入主题「${activeBarrage.title}」`}
      >
        <span class="speech-dot"></span>
        <span class="speech-text">{activeBarrage.title}</span>
      </button>
    {/key}
  {:else if isSingleThemeHovered && singleTheme && singleThemePal}
    <!-- Gentle Static Fallback when hovering single-theme nav button outside active barrage -->
    <button
      type="button"
      class="nav-speech-barrage static-hover"
      class:right-side={isRightSide}
      style="
        --barrage-primary: {singleThemePal.primary};
        --barrage-glow: {singleThemePal.glow};
        --barrage-bg: {singleThemePal.bgGradient};
        --barrage-border: {singleThemePal.border};
        --barrage-text: {singleThemePal.text};
        --barrage-offset-x: 0px;
      "
      on:click={(e) => handleButtonClick(singleTheme.id, e)}
      title={`进入主题「${singleTheme.title}」`}
    >
      <span class="speech-dot"></span>
      <span class="speech-text">{singleTheme.title}</span>
    </button>
  {/if}

  <!-- Orbital Satellite Dial: Shown when a card has multiple themes -->
  {#if themes.length > 1}
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div 
      class="orbital-theme-dial" 
      class:right-side={isRightSide}
      class:open={isOpen}
    >
      <!-- Unbroken 220px circular bridge covering nav button and all satellites to prevent mouseleave drops -->
      {#if isOpen}
        <div class="orbital-hit-bridge"></div>
      {/if}

      {#each themes as theme, index (theme.id)}
        {@const pos = positions[index] || { x: 0, y: 0 }}
        {@const palette = getThemePalette(theme.id, index)}
        {@const isActive = activeThemeId === theme.id}
        {@const isItemHovered = hoveredThemeId === theme.id}
        
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div 
          class="orbital-button-wrapper"
          style="
            transform: translate({isOpen ? pos.x : 0}px, {isOpen ? pos.y : 0}px) scale({isOpen ? 1 : 0.4});
            opacity: {isOpen ? 1 : 0};
            pointer-events: {isOpen ? 'auto' : 'none'};
            transition-delay: {isOpen ? `${index * 30}ms` : '0ms'};
          "
        >
          <button
            type="button"
            class="orbital-theme-btn"
            class:active={isActive}
            style="
              --btn-primary: {palette.primary};
              --btn-glow: {palette.glow};
              --btn-border: {palette.border};
              --btn-text: {palette.text};
              --btn-solid: {palette.bgSolid};
            "
            on:mouseenter={() => { hoveredThemeId = theme.id; }}
            on:mouseleave={() => { hoveredThemeId = null; }}
            on:click={(e) => handleButtonClick(theme.id, e)}
            title={`「${theme.title} · ${theme.cards.length}卡」`}
          >
            <span class="theme-label">{getButtonLabel(theme.title)}</span>
          </button>

          {#if isItemHovered}
            <div 
              class="orbital-tooltip"
              class:align-left={isRightSide}
              class:align-right={!isRightSide}
            >
              「{theme.title} · {theme.cards.length}卡」
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
{/if}

<style>
  .orbital-theme-dial {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    pointer-events: none;
    z-index: 1100;
  }

  /* Seamless invisible hover bridge connecting nav button directly to satellite buttons */
  .orbital-hit-bridge {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 220px;
    height: 220px;
    border-radius: 50%;
    background: transparent;
    pointer-events: auto;
    z-index: 1050;
  }

  /* Bottom-to-Top Floating Dialogue Bubble Barrage beside Left/Right Nav Button */
  .nav-speech-barrage {
    position: absolute;
    top: 50%;
    left: calc(100% + 12px);
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px 6px 10px;
    border-radius: 14px 14px 14px 3px;
    background: var(--barrage-bg, rgba(255, 255, 255, 0.9));
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border: 1.2px solid var(--barrage-border, rgba(255, 255, 255, 0.85));
    color: var(--barrage-text, #1e293b);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.08), 0 0 14px var(--barrage-glow, rgba(56, 189, 248, 0.25));
    white-space: nowrap;
    cursor: pointer;
    pointer-events: auto;
    z-index: 1080;
    appearance: none;
    font-family: inherit;
    line-height: 1.25;
    outline: none;
    will-change: transform, opacity;
    animation: speech-barrage-rise 2.5s linear forwards;
    transition: box-shadow 0.25s ease, border-color 0.25s ease;
  }

  .nav-speech-barrage:hover {
    animation-play-state: paused;
    border-color: var(--barrage-primary, #0284c7);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.14), 0 0 18px var(--barrage-glow, rgba(56, 189, 248, 0.4));
  }

  .nav-speech-barrage.static-hover {
    animation: none;
    opacity: 1;
    transform: translate3d(0, -10px, 0);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12), 0 0 16px var(--barrage-glow, rgba(56, 189, 248, 0.35));
    border-color: var(--barrage-primary, #0284c7);
  }

  .nav-speech-barrage.static-hover:hover {
    transform: translate3d(0, -10px, 0) scale(1.05);
    box-shadow: 0 8px 26px rgba(0, 0, 0, 0.16), 0 0 20px var(--barrage-glow, rgba(56, 189, 248, 0.5));
  }

  .nav-speech-barrage.right-side {
    left: auto;
    right: calc(100% + 12px);
    border-radius: 14px 14px 3px 14px;
  }

  .speech-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--barrage-primary, #0ea5e9);
    box-shadow: 0 0 6px var(--barrage-primary, #0ea5e9);
    flex-shrink: 0;
  }

  .speech-text {
    font-size: 11.5px;
    font-weight: 600;
    letter-spacing: 0.01em;
    max-width: 130px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    line-height: 1.25;
  }

  /* Steady, unbroken continuous upward displacement (strictly linear constant rate 88px / 2.5s) */
  @keyframes speech-barrage-rise {
    0% {
      opacity: 0;
      transform: translate3d(var(--barrage-offset-x, 0px), 24px, 0);
    }
    15% {
      opacity: 1;
      transform: translate3d(var(--barrage-offset-x, 0px), 11px, 0);
    }
    85% {
      opacity: 1;
      transform: translate3d(var(--barrage-offset-x, 0px), -51px, 0);
    }
    100% {
      opacity: 0;
      transform: translate3d(var(--barrage-offset-x, 0px), -64px, 0);
    }
  }

  .orbital-button-wrapper {
    position: absolute;
    top: 0;
    left: 0;
    margin-left: -22px;
    margin-top: -22px;
    width: 44px;
    height: 44px;
    transition: transform 0.32s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.25s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1120;
  }

  .orbital-theme-btn {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.75);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
    border: 1.5px solid var(--btn-border, rgba(255, 255, 255, 0.85));
    color: var(--btn-text, rgba(30, 41, 59, 0.88));
    font-size: 11px;
    font-weight: 600;
    letter-spacing: -0.2px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.08), 0 0 8px var(--btn-glow, rgba(56, 189, 248, 0.2));
    transition: all 0.22s cubic-bezier(0.16, 1, 0.3, 1);
    outline: none;
    user-select: none;
    -webkit-user-select: none;
    padding: 0;
    z-index: 1125;
  }

  .orbital-theme-btn:hover {
    background: rgba(255, 255, 255, 0.95);
    border-color: var(--btn-primary, #0284c7);
    color: var(--btn-text, #0f172a);
    transform: scale(1.12);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12), 0 0 16px var(--btn-glow, rgba(56, 189, 248, 0.4));
  }

  /* Active Theme Button: Colored Solid / Palette Gradient with Vibrant Glow */
  .orbital-theme-btn.active {
    background: var(--btn-solid, #0284c7);
    color: #ffffff;
    border-color: rgba(255, 255, 255, 0.95);
    box-shadow: 0 0 18px var(--btn-glow, rgba(56, 189, 248, 0.7)), 0 4px 14px rgba(0, 0, 0, 0.15);
  }

  .orbital-theme-btn.active:hover {
    transform: scale(1.15);
    box-shadow: 0 0 26px var(--btn-glow, rgba(56, 189, 248, 0.85)), 0 6px 18px rgba(0, 0, 0, 0.2);
  }

  .theme-label {
    max-width: 36px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: center;
    line-height: 1.1;
  }

  /* Tooltip Pill */
  .orbital-tooltip {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    background: rgba(15, 23, 42, 0.88);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    color: #f8fafc;
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 9999px;
    padding: 3px 10px;
    font-size: 11px;
    font-weight: 500;
    white-space: nowrap;
    pointer-events: none;
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.2);
    z-index: 1200;
  }

  .orbital-tooltip.align-right {
    left: calc(100% + 10px);
  }

  .orbital-tooltip.align-left {
    right: calc(100% + 10px);
  }
</style>
