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
  import { computeOrbitalButtonLayout } from '../../lib/themeOrbit';

  export let themes: ThemeThread[] = [];
  export let activeThemeId: string | null = null;
  export let isRightSide: boolean = false;
  export let isNavHovered: boolean = false;
  export let onToggleTheme: ((themeId: string) => void) | undefined = undefined;

  const dispatch = createEventDispatcher();

  let isDialHovered: boolean = false;
  let hoveredThemeId: string | null = null;
  let isGraceOpen: boolean = false;
  let closeTimer: any = null;

  $: {
    if (isNavHovered || isDialHovered) {
      if (closeTimer) {
        clearTimeout(closeTimer);
        closeTimer = null;
      }
      isGraceOpen = true;
    } else if (isGraceOpen) {
      if (!closeTimer) {
        closeTimer = setTimeout(() => {
          isGraceOpen = false;
          closeTimer = null;
        }, 280);
      }
    }
  }

  $: isOpen = isGraceOpen;

  $: positions = computeOrbitalButtonLayout(themes.length, isRightSide, 72);

  function handleDialMouseEnter() {
    isDialHovered = true;
    if (closeTimer) clearTimeout(closeTimer);
  }

  function handleDialMouseLeave() {
    isDialHovered = false;
    hoveredThemeId = null;
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
  });
</script>

{#if themes && themes.length > 0}
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div 
    class="orbital-theme-dial" 
    class:right-side={isRightSide}
    class:open={isOpen}
    on:mouseenter={handleDialMouseEnter}
    on:mouseleave={handleDialMouseLeave}
  >
    {#each themes as theme, index (theme.id)}
      {@const pos = positions[index] || { x: 0, y: 0 }}
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
          transition-delay: {isOpen ? `${index * 35}ms` : '0ms'};
        "
      >
        <button
          type="button"
          class="orbital-theme-btn"
          class:active={isActive}
          on:mouseenter={() => hoveredThemeId = theme.id}
          on:mouseleave={() => hoveredThemeId = null}
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
  }

  .orbital-theme-btn {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.65);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
    border: 1px solid rgba(255, 255, 255, 0.85);
    color: rgba(30, 41, 59, 0.88);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: -0.2px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1), inset 0 1px 0 rgba(255, 255, 255, 0.9);
    transition: all 0.22s cubic-bezier(0.16, 1, 0.3, 1);
    outline: none;
    user-select: none;
    -webkit-user-select: none;
    padding: 0;
  }

  .orbital-theme-btn:hover {
    background: rgba(255, 255, 255, 0.95);
    color: #0f172a;
    transform: scale(1.12);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.14), 0 0 12px rgba(56, 189, 248, 0.35);
  }

  /* Active Theme Button: Vibrant Aurora Gradient Glow */
  .orbital-theme-btn.active {
    background: linear-gradient(135deg, rgba(56, 189, 248, 0.92), rgba(168, 85, 247, 0.92));
    color: #ffffff;
    border-color: rgba(255, 255, 255, 0.95);
    box-shadow: 0 0 18px rgba(56, 189, 248, 0.7), 0 0 32px rgba(168, 85, 247, 0.45), 0 4px 14px rgba(0, 0, 0, 0.15);
  }

  .orbital-theme-btn.active:hover {
    transform: scale(1.15);
    box-shadow: 0 0 24px rgba(56, 189, 248, 0.85), 0 0 40px rgba(168, 85, 247, 0.6);
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
