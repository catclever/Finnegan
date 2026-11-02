<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - ThemeBottomBubble Component
  Task: theme_carousel_mode_150
  Resident frosted-glass bubble below focused card showing active theme title & index,
  emitting gentle rising colorful micro-bubbles matching the theme's vibrant palette.
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { ThemeThread } from '../../types/card';
  import { getThemePalette } from '../../lib/themeOrbit';

  export let theme: ThemeThread;
  export let themes: ThemeThread[] = [];
  export let currentIndex: number = 0;
  export let totalCards: number = 1;

  const dispatch = createEventDispatcher<{
    close: void;
    switchTheme: string;
  }>();

  $: palette = getThemePalette(theme.id);

  $: canSwitch = themes && themes.length > 1;
  $: currentThemeIdx = themes ? themes.findIndex(t => t.id === theme.id) : -1;
  $: currentThemeNum = currentThemeIdx !== -1 ? currentThemeIdx + 1 : 1;

  // Staggered micro-bubbles with gentle drift
  const microBubbles = [
    { size: 7, left: 18, delay: 0.2, duration: 2.8, drift: -8 },
    { size: 11, left: 32, delay: 1.1, duration: 3.4, drift: 6 },
    { size: 6, left: 50, delay: 0.6, duration: 2.5, drift: -4 },
    { size: 12, left: 68, delay: 1.8, duration: 3.6, drift: 8 },
    { size: 8, left: 82, delay: 0.9, duration: 3.1, drift: -6 },
    { size: 5, left: 42, delay: 2.2, duration: 2.4, drift: 5 },
  ];

  function handlePillClick(e: MouseEvent) {
    if (!canSwitch) return;
    const nextIdx = (currentThemeIdx + 1) % themes.length;
    const nextTheme = themes[nextIdx];
    if (nextTheme) {
      dispatch('switchTheme', nextTheme.id);
    }
  }

  function handleClose(e: MouseEvent) {
    e.stopPropagation();
    dispatch('close');
  }
</script>

<div 
  class="theme-bottom-bubble-container"
  style="
    --theme-primary: {palette.primary};
    --theme-border: {palette.border};
    --theme-glow: {palette.glow};
    --theme-text: {palette.text};
  "
>
  <!-- Rising Micro-bubbles Emitter -->
  <div class="micro-bubbles-emitter" aria-hidden="true">
    {#each microBubbles as b}
      <span 
        class="micro-bubble"
        style="
          width: {b.size}px;
          height: {b.size}px;
          left: {b.left}%;
          animation-delay: {b.delay}s;
          animation-duration: {b.duration}s;
          --drift: {b.drift}px;
        "
      ></span>
    {/each}
  </div>

  <!-- Resident Pill Bubble -->
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div 
    class="theme-pill-bubble" 
    class:can-switch={canSwitch}
    role="status" 
    aria-label="主题模式: {theme.title}"
    on:click={handlePillClick}
    title={canSwitch ? `点击切换当前卡片的主题 (${currentThemeNum}/${themes.length})` : undefined}
  >
    <span class="pill-symbol">🪐</span>
    <span class="pill-title">{theme.title}</span>
    {#if canSwitch}
      <span class="pill-switch-badge" title={`点击切换到下一个主题 (${currentThemeNum}/${themes.length})`}>
        <svg viewBox="0 0 16 16" width="11" height="11" fill="currentColor">
          <path d="M1 11.5a.5.5 0 0 0 .5.5h11.793l-3.147 3.146a.5.5 0 0 0 .708.708l4-4a.5.5 0 0 0 0-.708l-4-4a.5.5 0 0 0-.708.708L13.293 11H1.5a.5.5 0 0 0-.5.5zm14-7a.5.5 0 0 1-.5.5H2.707l3.147 3.146a.5.5 0 1 1-.708.708l-4-4a.5.5 0 0 1 0-.708l4-4a.5.5 0 1 1 .708.708L2.707 4H14.5a.5.5 0 0 1 .5.5z"/>
        </svg>
        <span class="switch-index">{currentThemeNum}/{themes.length}</span>
      </span>
    {/if}
    <span class="pill-sep">·</span>
    <span class="pill-count">{currentIndex + 1} / {totalCards}</span>
    <button 
      type="button" 
      class="pill-close-btn" 
      title="退出主题模式 (Esc)"
      on:click={handleClose}
    >✕</button>
  </div>
</div>

<style>
  .theme-bottom-bubble-container {
    position: absolute;
    left: 50%;
    top: calc(50% + 242px);
    transform: translateX(-50%);
    z-index: 45;
    display: flex;
    flex-direction: column;
    align-items: center;
    pointer-events: auto;
    user-select: none;
    animation: bubble-appear 0.35s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }

  @keyframes bubble-appear {
    from {
      opacity: 0;
      transform: translate(-50%, 8px) scale(0.94);
    }
    to {
      opacity: 1;
      transform: translate(-50%, 0) scale(1);
    }
  }

  .theme-pill-bubble {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 14px 6px 12px;
    border-radius: 9999px;
    background: rgba(255, 255, 255, 0.85);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border: 1.5px solid var(--theme-border);
    box-shadow: 
      0 4px 20px var(--theme-glow),
      0 2px 8px rgba(0, 0, 0, 0.06);
    color: var(--theme-text);
    font-size: 12px;
    font-weight: 600;
    letter-spacing: -0.01em;
    position: relative;
    z-index: 2;
    transition: all 0.22s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .theme-pill-bubble:hover {
    background: rgba(255, 255, 255, 0.95);
    box-shadow: 
      0 6px 26px var(--theme-glow),
      0 2px 10px rgba(0, 0, 0, 0.08);
  }

  .theme-pill-bubble.can-switch {
    cursor: pointer;
  }

  .theme-pill-bubble.can-switch:hover {
    border-color: var(--theme-primary);
    transform: scale(1.03);
  }

  .pill-symbol {
    font-size: 13px;
    line-height: 1;
    filter: drop-shadow(0 1px 2px var(--theme-glow));
  }

  .pill-title {
    max-width: 170px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .pill-switch-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 1.5px 6px;
    border-radius: 9999px;
    background: var(--theme-glow, rgba(56, 189, 248, 0.15));
    border: 1px solid var(--theme-border, rgba(56, 189, 248, 0.3));
    color: var(--theme-primary, #0284c7);
    font-size: 10px;
    font-weight: 600;
    line-height: 1;
    margin-left: 1px;
    user-select: none;
    transition: all 0.2s ease;
  }

  .theme-pill-bubble:hover .pill-switch-badge {
    background: var(--theme-primary, #0284c7);
    color: #ffffff;
    border-color: transparent;
  }

  .switch-index {
    font-variant-numeric: tabular-nums;
  }

  .pill-sep {
    opacity: 0.5;
    font-size: 11px;
  }

  .pill-count {
    font-variant-numeric: tabular-nums;
    font-weight: 550;
    opacity: 0.85;
  }

  .pill-close-btn {
    border: none;
    background: rgba(0, 0, 0, 0.04);
    padding: 0;
    margin-left: 3px;
    font-size: 11px;
    color: var(--theme-text);
    opacity: 0.55;
    cursor: pointer;
    line-height: 1;
    border-radius: 50%;
    width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.18s ease;
  }

  .pill-close-btn:hover {
    opacity: 1;
    background: rgba(239, 68, 68, 0.16);
    color: #ef4444;
    transform: scale(1.15);
  }

  /* Micro Bubbles */
  .micro-bubbles-emitter {
    position: absolute;
    inset: -10px -20px 0 -20px;
    pointer-events: none;
    overflow: visible;
    z-index: 1;
  }

  .micro-bubble {
    position: absolute;
    bottom: 8px;
    border-radius: 50%;
    background: radial-gradient(circle at 35% 35%, rgba(255, 255, 255, 0.9), var(--theme-primary) 70%);
    box-shadow: 0 0 6px var(--theme-glow);
    opacity: 0;
    pointer-events: none;
    will-change: transform, opacity;
    animation-name: bubble-float-up;
    animation-timing-function: cubic-bezier(0.25, 0.1, 0.25, 1);
    animation-iteration-count: infinite;
  }

  @keyframes bubble-float-up {
    0% {
      opacity: 0;
      transform: translate3d(0, 0, 0) scale(0.5);
    }
    20% {
      opacity: 0.8;
    }
    70% {
      opacity: 0.6;
    }
    100% {
      opacity: 0;
      transform: translate3d(var(--drift, 0px), -64px, 0) scale(1.1);
    }
  }
</style>
