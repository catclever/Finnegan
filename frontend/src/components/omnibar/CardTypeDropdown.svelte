<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CardTypeDropdown Subcomponent
  Task: card_type_dropdown_and_module_driven_registry_151
  Renders the card type selection dropdown menu in OmniBar when user types ':' prefix.
  ======================================================================================
-->
<script lang="ts">
  import type { CardTypeOption } from '../../lib/cardType';

  export let matchedTypes: CardTypeOption[] = [];
  export let highlightedIndex: number = 0;
  export let onSelect: (typeOption: CardTypeOption) => void = () => {};
  export let onHighlight: (index: number) => void = () => {};
</script>

{#if matchedTypes.length > 0}
  <div class="dropdown-menu card-type-dropdown" on:wheel|stopPropagation>
    <div class="dropdown-header">
      <span class="dropdown-title">卡片类型 (Card Types)</span>
      <span class="dropdown-count">{matchedTypes.length} 个可用形态</span>
    </div>
    <div class="type-list">
      {#each matchedTypes as typeOpt, index}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div 
          class="dropdown-item type-item" 
          class:active={index === highlightedIndex}
          on:mouseenter={() => onHighlight(index)}
          on:mousedown|preventDefault={() => onSelect(typeOpt)}
        >
          <div class="type-item-content">
            <span class="type-icon">{typeOpt.icon}</span>
            <div class="type-meta">
              <span class="type-prefix">{typeOpt.prefix}</span>
              <span class="type-label">{typeOpt.label}</span>
              <span class="type-desc">{typeOpt.description}</span>
            </div>
          </div>
          <span class="type-enter-cue">↵ 选择</span>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .dropdown-menu {
    width: 100%;
    margin-top: 8px;
    background: rgba(255, 255, 255, 0.96);
    backdrop-filter: blur(28px);
    -webkit-backdrop-filter: blur(28px);
    border: 1px solid rgba(255, 255, 255, 0.85);
    border-radius: 20px;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.16);
    overflow: hidden;
    max-height: 280px;
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

  .dropdown-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 16px 6px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
    background: rgba(0, 0, 0, 0.02);
  }

  .dropdown-title {
    font-size: 11px;
    font-weight: 600;
    color: rgba(0, 0, 0, 0.45);
    letter-spacing: 0.02em;
  }

  .dropdown-count {
    font-size: 10px;
    color: rgba(0, 0, 0, 0.35);
  }

  .type-list {
    overflow-y: auto;
    flex: 1;
  }

  .dropdown-item {
    padding: 9px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    cursor: pointer;
    transition: background 0.12s ease;
    border-bottom: 1px solid rgba(0, 0, 0, 0.03);
  }

  .dropdown-item:last-child {
    border-bottom: none;
  }

  .dropdown-item.active {
    background: rgba(56, 189, 248, 0.12);
  }

  .type-item-content {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }

  .type-icon {
    font-size: 1.15rem;
    line-height: 1;
    flex-shrink: 0;
  }

  .type-meta {
    display: flex;
    align-items: baseline;
    gap: 8px;
    min-width: 0;
  }

  .type-prefix {
    font-size: 13px;
    font-weight: 700;
    color: #0284c7;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    background: rgba(56, 189, 248, 0.15);
    padding: 1px 6px;
    border-radius: 5px;
    flex-shrink: 0;
  }

  .type-label {
    font-size: 13px;
    font-weight: 600;
    color: #1e293b;
    flex-shrink: 0;
  }

  .type-desc {
    font-size: 11px;
    color: #64748b;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .type-enter-cue {
    font-size: 10px;
    color: rgba(0, 0, 0, 0.35);
    opacity: 0;
    transition: opacity 0.12s ease;
    flex-shrink: 0;
  }

  .dropdown-item.active .type-enter-cue {
    opacity: 1;
    color: #0284c7;
  }
</style>
