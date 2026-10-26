<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CardCandidateDropdown Subcomponent
  Task: modularize_backend_and_frontend_architecture_074
  Renders the parameter card candidate search dropdown menu.
  ======================================================================================
-->
<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import { getCardParamDisplayTitle } from '../../lib/cardTitle';

  export let paramCardCandidates: CosmicCard[] = [];
  export let highlightedCardIndex: number = 0;
  export let omniText: string = "";
  export let onSelect: (candidate: CosmicCard) => void = () => {};
  export let onHighlight: (index: number) => void = () => {};
</script>

{#if paramCardCandidates.length > 0}
  <div class="dropdown-menu card-dropdown" on:wheel|stopPropagation>
    {#each paramCardCandidates as candidate, index}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div 
        class="dropdown-item card-item" 
        class:active={index === highlightedCardIndex}
        on:mouseenter={() => onHighlight(index)}
        on:mousedown|preventDefault={() => onSelect(candidate)}
      >
        <div class="card-item-title">
          <span class="candidate-icon">📄</span>
          {getCardParamDisplayTitle(candidate, omniText)}
        </div>
      </div>
    {/each}
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
    overflow: hidden;
    max-height: 280px;
    overflow-y: auto;
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

  .dropdown-item {
    padding: 10px 16px;
    display: flex;
    flex-direction: column;
    gap: 3px;
    cursor: pointer;
    transition: background 0.15s ease;
    border-bottom: 1px solid rgba(0, 0, 0, 0.04);
  }

  .dropdown-item:last-child {
    border-bottom: none;
  }

  .dropdown-item.active {
    background: rgba(59, 130, 246, 0.1);
  }

  .dropdown-item.card-item {
    flex-direction: row;
    align-items: center;
    padding: 9px 16px;
  }

  .card-item-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-weight: 600;
    color: #111827;
    font-size: 0.92rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
    width: 100%;
  }

  .candidate-icon {
    flex-shrink: 0;
  }
</style>
