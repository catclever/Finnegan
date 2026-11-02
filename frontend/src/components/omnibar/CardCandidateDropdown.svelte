<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CardCandidateDropdown Subcomponent
  Task: global_candidate_cards_hierarchy_and_viewport_staging_087
  Renders the tiered candidate search dropdown menu with section cues and auto-scroll.
  ======================================================================================
-->
<script lang="ts">
  import { tick } from 'svelte';
  import type { CosmicCard, CandidateCardItem } from '../../types/card';
  import { getCardParamDisplayTitle } from '../../lib/cardTitle';

  export let candidateItems: CandidateCardItem[] = [];
  export let paramCardCandidates: CosmicCard[] = []; // Backward compatibility
  export let highlightedCardIndex: number = 0;
  export let omniText: string = "";
  export let hasMoreVault: boolean = false;
  export let directSearchAction: { label: string; text: string } | null = null;
  export let onDirectSearch: ((text: string) => void) | undefined = undefined;
  export let onSelect: (candidate: CosmicCard) => void = () => {};
  export let onHighlight: (index: number) => void = () => {};
  export let onLoadMore: () => void = () => {};

  // Stage 2 Source / Self Picker Props
  export let pickerMode: boolean = false;
  export let pickerOptions: Array<{ id: string; title: string; kind: 'root' | 'self'; label: string; hint: string; checked: boolean }> = [];
  export let onToggleOption: ((index: number) => void) | undefined = undefined;
  export let onConfirmPicker: (() => void) | undefined = undefined;

  let menuEl: HTMLDivElement | null = null;

  // Normalize candidateItems or fallback to plain cards
  let items: CandidateCardItem[] = [];
  $: items = ((candidateItems && candidateItems.length > 0)
    ? candidateItems
    : (paramCardCandidates || []).map(c => ({ card: c, tier: 'current_screen' as const }))) as CandidateCardItem[];

  // Scroll active item into view whenever index updates
  $: if (menuEl && (highlightedCardIndex === -1 || (highlightedCardIndex >= 0 && (items.length > 0 || pickerOptions.length > 0)))) {
    tick().then(() => {
      if (!menuEl) return;
      const activeEl = menuEl.querySelector('.dropdown-item.active') as HTMLElement;
      if (activeEl) {
        activeEl.scrollIntoView({ block: 'nearest', behavior: 'auto' });
      }
    });
  }

  function handleItemMouseMove(e: MouseEvent, index: number) {
    // Only update highlight when user actually moves the mouse physically,
    // preventing a stationary mouse from stealing highlight during keyboard scroll navigation.
    if (e.movementX !== 0 || e.movementY !== 0) {
      if (highlightedCardIndex !== index) {
        onHighlight(index);
      }
    }
  }

  function handleScroll(e: Event) {
    const el = e.target as HTMLElement;
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 48) {
      onLoadMore();
    }
  }

  function isFirstInTier(idx: number): boolean {
    if (idx === 0) return true;
    return items[idx].tier !== items[idx - 1].tier;
  }

  function getTierLabel(tier: string): string {
    const countInTier = items.filter(it => it.tier === tier).length;
    if (tier === 'current_screen') {
      return `🌌 当前屏幕 (${countInTier})`;
    } else if (tier === 'other_screens') {
      return `🪐 其他屏幕 (${countInTier})`;
    } else {
      return `📜 历史归档 (${countInTier}${hasMoreVault ? '+' : ''})`;
    }
  }
</script>

{#if pickerMode && pickerOptions.length > 0}
  <!-- Two-Stage Source / Self Multi-Select Picker -->
  <div 
    bind:this={menuEl}
    class="dropdown-menu card-dropdown picker-mode"
  >
    <div class="picker-header">
      <span class="picker-header-title">选择关联归组</span>
      <span class="picker-header-tip">空格 勾选 · 回车 确认</span>
    </div>

    {#each pickerOptions as opt, index}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div 
        class="dropdown-item picker-item"
        class:active={index === highlightedCardIndex}
        class:checked={opt.checked}
        on:mousemove={() => onHighlight(index)}
        on:mousedown|preventDefault={() => onToggleOption ? onToggleOption(index) : null}
      >
        <div class="checkbox-indicator" class:checked={opt.checked}>
          {#if opt.checked}
            ✓
          {/if}
        </div>
        <div class="picker-item-content">
          <span class="picker-icon">{opt.kind === 'root' ? '🪐' : '📄'}</span>
          <span class="picker-label">{opt.label}</span>
        </div>
        <span class="picker-hint-badge {opt.kind}">
          {opt.hint}
        </span>
      </div>
    {/each}

    <div class="picker-footer-bar">
      <span class="key-hint">↑↓ 移动 · 空格 选择 · 回车 确认 · Esc 返回</span>
      <button 
        type="button" 
        class="picker-confirm-btn"
        on:mousedown|preventDefault={() => onConfirmPicker ? onConfirmPicker() : null}
      >确认 (↵)</button>
    </div>
  </div>
{:else if items.length > 0 || (directSearchAction && directSearchAction.text.trim())}
  <div 
    bind:this={menuEl}
    class="dropdown-menu card-dropdown" 
    on:scroll={handleScroll}
  >
    {#if directSearchAction && directSearchAction.text.trim()}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div 
        class="dropdown-item direct-search-item" 
        class:active={highlightedCardIndex === -1}
        on:mousemove={() => onHighlight(-1)}
        on:mousedown|preventDefault={() => onDirectSearch ? onDirectSearch(directSearchAction.text.trim()) : null}
      >
        <div class="card-item-title">
          <span class="candidate-icon">🦋</span>
          <span class="title-text">搜索 “<strong>{directSearchAction.text.trim()}</strong>” · 直接检索 Bluesky</span>
        </div>
        <span class="direct-search-badge">直接搜索</span>
      </div>
    {/if}

    {#each items as item, index}
      <!-- Section Divider Header (Non-selectable cue) -->
      {#if isFirstInTier(index)}
        <div class="section-divider {item.tier}">
          <span class="divider-text">{getTierLabel(item.tier)}</span>
        </div>
      {/if}

      <!-- Interactive Card Item -->
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div 
        class="dropdown-item card-item" 
        class:active={index === highlightedCardIndex}
        on:mousemove={(e) => handleItemMouseMove(e, index)}
        on:mousedown|preventDefault={() => onSelect(item.card)}
      >
        <div class="card-item-title">
          <span class="candidate-icon">📄</span>
          <span class="title-text">{getCardParamDisplayTitle(item.card, omniText)}</span>
        </div>

        <!-- Tier Source Badge -->
        <span class="tier-chip {item.tier}">
          {#if item.tier === 'current_screen'}
            当前屏
          {:else if item.tier === 'other_screens'}
            第{item.screenIndex || 2}屏
          {:else}
            归档
          {/if}
        </span>
      </div>
    {/each}

    {#if hasMoreVault}
      <div class="load-more-hint">
        <span>↓ 继续下翻加载更多历史卡片...</span>
      </div>
    {/if}
  </div>
{/if}

<style>
  .dropdown-menu {
    width: 100%;
    margin-top: 8px;
    background: rgba(255, 255, 255, 0.96);
    backdrop-filter: blur(28px);
    -webkit-backdrop-filter: blur(28px);
    border: 1px solid rgba(255, 255, 255, 0.8);
    border-radius: 20px;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.16);
    max-height: 320px;
    overflow-y: auto;
    overscroll-behavior: contain;
    z-index: 60;
    display: flex;
    flex-direction: column;
    animation: dropdown-slide 0.15s ease-out;
    box-sizing: border-box;
  }

  @keyframes dropdown-slide {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  /* Section Divider Cue */
  .section-divider {
    padding: 8px 16px 4px 16px;
    display: flex;
    align-items: center;
    background: rgba(248, 250, 252, 0.75);
    border-top: 1px solid rgba(0, 0, 0, 0.05);
    border-bottom: 1px solid rgba(0, 0, 0, 0.03);
    user-select: none;
    pointer-events: none;
  }

  .section-divider:first-child {
    border-top: none;
    padding-top: 6px;
  }

  .divider-text {
    font-size: 0.74rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: #64748b;
  }

  .section-divider.current .divider-text {
    color: #2563eb;
  }

  .section-divider.other .divider-text {
    color: #7c3aed;
  }

  .section-divider.vault .divider-text {
    color: #64748b;
  }

  /* Two-Stage Picker Styles */
  .picker-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 16px 8px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.06);
    background: rgba(248, 250, 252, 0.85);
    user-select: none;
  }

  .picker-header-title {
    font-size: 0.8rem;
    font-weight: 700;
    color: #1e293b;
    letter-spacing: -0.01em;
  }

  .picker-header-tip {
    font-size: 0.72rem;
    color: #64748b;
  }

  .picker-item {
    cursor: pointer;
    user-select: none;
    transition: background 0.15s ease;
  }

  .picker-item.checked {
    background: rgba(14, 165, 233, 0.05);
  }

  .picker-item.active {
    background: rgba(14, 165, 233, 0.12) !important;
  }

  .checkbox-indicator {
    width: 17px;
    height: 17px;
    border-radius: 5px;
    border: 1.5px solid #cbd5e1;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 800;
    color: #ffffff;
    margin-right: 12px;
    transition: all 0.18s cubic-bezier(0.16, 1, 0.3, 1);
    flex-shrink: 0;
  }

  .checkbox-indicator.checked {
    background: #0ea5e9;
    border-color: #0ea5e9;
    box-shadow: 0 1px 4px rgba(14, 165, 233, 0.3);
  }

  .picker-item-content {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
  }

  .picker-icon {
    font-size: 13px;
    line-height: 1;
    flex-shrink: 0;
  }

  .picker-label {
    font-size: 0.86rem;
    font-weight: 600;
    color: #1e293b;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .picker-hint-badge {
    font-size: 0.72rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 9999px;
    flex-shrink: 0;
    letter-spacing: -0.01em;
  }

  .picker-hint-badge.root {
    background: rgba(14, 165, 233, 0.12);
    color: #0284c7;
    border: 1px solid rgba(14, 165, 233, 0.25);
  }

  .picker-hint-badge.self {
    background: rgba(139, 92, 246, 0.12);
    color: #7c3aed;
    border: 1px solid rgba(139, 92, 246, 0.25);
  }

  .picker-footer-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 14px;
    background: rgba(248, 250, 252, 0.9);
    border-top: 1px solid rgba(0, 0, 0, 0.05);
    font-size: 0.72rem;
    color: #64748b;
    margin-top: auto;
  }

  .key-hint {
    font-size: 0.71rem;
    color: #64748b;
  }

  .picker-confirm-btn {
    padding: 4px 12px;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 6px;
    background: #0ea5e9;
    color: #ffffff;
    border: none;
    cursor: pointer;
    box-shadow: 0 1px 4px rgba(14, 165, 233, 0.35);
    transition: all 0.15s ease;
  }

  .picker-confirm-btn:hover {
    background: #0284c7;
  }

  /* Interactive Dropdown Item */
  .dropdown-item {
    padding: 10px 16px;
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    cursor: pointer;
    transition: background 0.12s ease;
    border-bottom: 1px solid rgba(0, 0, 0, 0.03);
  }

  .dropdown-item:last-of-type {
    border-bottom: none;
  }

  .dropdown-item.active {
    background: rgba(59, 130, 246, 0.12);
  }

  /* Direct Search Action Item (Bluesky / Atproto) */
  .dropdown-item.direct-search-item {
    background: linear-gradient(90deg, rgba(2, 132, 199, 0.05) 0%, rgba(56, 189, 248, 0.03) 100%);
    border-bottom: 1px solid rgba(2, 132, 199, 0.12);
  }

  .dropdown-item.direct-search-item.active {
    background: linear-gradient(90deg, rgba(2, 132, 199, 0.15) 0%, rgba(56, 189, 248, 0.10) 100%);
  }

  .direct-search-item .candidate-icon {
    font-size: 1.05rem;
    filter: drop-shadow(0 0 4px rgba(56, 189, 248, 0.6));
  }

  .direct-search-item .title-text {
    color: #0369a1;
  }

  .direct-search-item .title-text strong {
    color: #0284c7;
    font-weight: 700;
  }

  .direct-search-badge {
    flex-shrink: 0;
    font-size: 0.68rem;
    font-weight: 600;
    padding: 2px 7px;
    border-radius: 9999px;
    letter-spacing: 0.02em;
    user-select: none;
    background: rgba(2, 132, 199, 0.12);
    color: #0284c7;
    border: 1px solid rgba(2, 132, 199, 0.25);
  }

  .card-item-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    color: #111827;
    font-size: 0.90rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
    flex: 1;
  }

  .title-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .candidate-icon {
    flex-shrink: 0;
    font-size: 0.95rem;
  }

  /* Tier Chip Badge */
  .tier-chip {
    flex-shrink: 0;
    font-size: 0.68rem;
    font-weight: 600;
    padding: 2px 7px;
    border-radius: 9999px;
    letter-spacing: 0.02em;
    user-select: none;
  }

  .tier-chip.current {
    background: rgba(37, 99, 235, 0.1);
    color: #2563eb;
    border: 1px solid rgba(37, 99, 235, 0.2);
  }

  .tier-chip.other {
    background: rgba(124, 58, 237, 0.1);
    color: #7c3aed;
    border: 1px solid rgba(124, 58, 237, 0.2);
  }

  .tier-chip.vault {
    background: rgba(100, 116, 139, 0.1);
    color: #64748b;
    border: 1px solid rgba(100, 116, 139, 0.2);
  }

  .load-more-hint {
    padding: 8px 16px;
    text-align: center;
    font-size: 0.76rem;
    color: #94a3b8;
    background: rgba(248, 250, 252, 0.5);
    user-select: none;
  }
</style>
