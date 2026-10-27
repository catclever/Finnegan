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
  export let onSelect: (candidate: CosmicCard) => void = () => {};
  export let onHighlight: (index: number) => void = () => {};
  export let onLoadMore: () => void = () => {};

  let menuEl: HTMLDivElement | null = null;

  // Normalize candidateItems or fallback to plain cards
  let items: CandidateCardItem[] = [];
  $: items = ((candidateItems && candidateItems.length > 0)
    ? candidateItems
    : (paramCardCandidates || []).map(c => ({ card: c, tier: 'current_screen' as const }))) as CandidateCardItem[];

  // Scroll active item smoothly into view whenever index updates
  $: if (menuEl && highlightedCardIndex >= 0 && items.length > 0) {
    tick().then(() => {
      if (!menuEl) return;
      const activeEl = menuEl.querySelector('.dropdown-item.active') as HTMLElement;
      if (activeEl) {
        activeEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      }
    });
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

{#if items.length > 0}
  <div 
    bind:this={menuEl}
    class="dropdown-menu card-dropdown" 
    on:scroll={handleScroll}
  >
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
        on:mouseenter={() => onHighlight(index)}
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
    scroll-behavior: smooth;
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
