<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - TagDropdown Subcomponent
  Task: card_tag_input_and_autocomplete_092
  Renders the hashtag autocomplete suggestions dropdown menu in OmniBar.
  ======================================================================================
-->
<script lang="ts">
  export let matchedTags: string[] = [];
  export let highlightedIndex: number = 0;
  export let onSelect: (tag: string) => void = () => {};
  export let onHighlight: (index: number) => void = () => {};
</script>

{#if matchedTags.length > 0}
  <div class="dropdown-menu tag-dropdown" on:wheel|stopPropagation>
    <div class="dropdown-header">
      <span class="dropdown-title">已有标签与锚点 (Tags & Anchors)</span>
      <span class="dropdown-count">{matchedTags.length} 个候选</span>
    </div>
    <div class="tag-list">
      {#each matchedTags as tag, index}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div 
          class="dropdown-item tag-item" 
          class:active={index === highlightedIndex}
          on:mouseenter={() => onHighlight(index)}
          on:mousedown|preventDefault={() => onSelect(tag)}
        >
          <div class="tag-item-content">
            <span class="tag-hash">#</span>
            <span class="tag-name">{tag}</span>
          </div>
          <span class="tag-enter-cue">↵ 选择</span>
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
    border: 1px solid rgba(255, 255, 255, 0.8);
    border-radius: 18px;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.16);
    overflow: hidden;
    max-height: 260px;
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

  .tag-list {
    overflow-y: auto;
    flex: 1;
  }

  .dropdown-item {
    padding: 8px 16px;
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

  .tag-item-content {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .tag-hash {
    font-size: 12px;
    font-weight: 700;
    color: #0284c7;
    background: rgba(56, 189, 248, 0.15);
    padding: 1px 5px;
    border-radius: 4px;
  }

  .tag-name {
    font-size: 13px;
    font-weight: 500;
    color: #1e293b;
  }

  .tag-enter-cue {
    font-size: 10px;
    color: rgba(0, 0, 0, 0.35);
    opacity: 0;
    transition: opacity 0.12s ease;
  }

  .dropdown-item.active .tag-enter-cue {
    opacity: 1;
    color: #0284c7;
  }
</style>
