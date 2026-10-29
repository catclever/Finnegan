<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - NoteCardBody Component
  Task: note_card_rendering_and_search_focus_fix_072

  Renders dedicated body for note cards (type: 'note'):
  - Scattering Mode (200x190): Clean layout with note badge 📝, title if present,
    snippet of content with clean/monospace styling, footer time.
  - Focused Mode (420x420): Clean header with time + prominent title (h3),
    body with pre-wrap and monospace font for commands/scripts/memos,
    natural smooth scrolling with hidden scrollbar.
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { CosmicCard } from '../../types/card';
  import { getCardDisplayTime, formatPointTime, stageTime } from '../../lib/time';
  import { getCardTags, normalizeTag } from '../../lib/tagNetwork';
  import MarkdownContent from './MarkdownContent.svelte';

  export let card: CosmicCard;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let thoughtTime: string = '';
  export let babelTime: string = '';
  export let displayTime: string = '';
  export let z: number | undefined = undefined;
  export let activeConstellationTags: Map<string, string> = new Map();
  export let activeConstellationTag: string | null = null;

  function getActiveTagColor(tag: string, tagsMap?: Map<string, string>): string | null {
    if (tagsMap && tagsMap.size > 0) {
      const clean = normalizeTag(tag);
      return tagsMap.get(clean) || tagsMap.get(tag) || null;
    }
    if (activeConstellationTag) {
      const clean = normalizeTag(tag);
      if (clean === normalizeTag(activeConstellationTag)) {
        return 'hsl(199, 90%, 60%)';
      }
    }
    return null;
  }

  function getTagPillStyle(tag: string, tagsMap?: Map<string, string>): string {
    const col = getActiveTagColor(tag, tagsMap);
    if (!col) return '';
    return `background: ${col} !important; color: #fff !important; box-shadow: 0 0 10px ${col} !important;`;
  }

  $: manualKeywords = getCardTags(card);

  const dispatch = createEventDispatcher<{
    tagClick: { tag: string; cardId: string };
  }>();

  $: currentZ = z !== undefined ? z : card?.z;
  $: hasActiveAura = Boolean(
    card?.auraTraces &&
    card.auraTraces.length > 0 &&
    card.auraTraces.some((t: any) => !t.duration || Date.now() - (t.hitTime || 0) < t.duration)
  );
  $: isForeground = currentZ !== undefined && currentZ <= 16;
  $: isTagVisible = isForeground || isHovered || isFocused || hasActiveAura;

  function handleTagClick(tag: string) {
    dispatch('tagClick', { tag, cardId: card?.id });
  }

  $: title = card?.title && card.title !== 'Thought' && card.title !== 'Card' && card.title !== 'Note' ? card.title : '';
  $: resolvedDisplayTime = displayTime || thoughtTime || babelTime || formatPointTime(card?.createdAt || card?.spawnTime, $stageTime) || getCardDisplayTime(card, $stageTime);
</script>

<div class="note-card-body" class:focused={isFocused} class:hovered={isHovered}>
  {#if isFocused}
    <!-- Focused Mode (420x420) -->
    <header class="note-focused-header">
      <div class="note-header-top">
        <span class="note-badge" title="Note">📝</span>
        {#if resolvedDisplayTime}
          <span class="card-time">{resolvedDisplayTime}</span>
        {/if}
      </div>
      {#if title}
        <h3 class="note-focused-title">{title}</h3>
      {/if}
      <div class="note-divider" aria-hidden="true"></div>
    </header>

    <div class="note-focused-body">
      <MarkdownContent content={card?.content || ''} isFocused={true} />
    </div>

    {#if manualKeywords.length > 0}
      <div class="note-tags focused">
        {#each manualKeywords as tag}
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <span 
            class="note-tag focused" 
            class:active={Boolean(getActiveTagColor(tag, activeConstellationTags))}
            style={getTagPillStyle(tag, activeConstellationTags)}
            role="button" 
            tabindex="0"
            on:click|stopPropagation={() => handleTagClick(tag)}
          >#{tag}</span>
        {/each}
      </div>
    {/if}
  {:else}
    <!-- Scattering Mode (200x190) -->
    <div class="note-preview">
      <div class="note-header-row">
        <span class="note-badge" title="Note">📝</span>
        {#if title}
          <span class="note-preview-title" title={title}>{title}</span>
        {/if}
      </div>
      <div class="note-snippet">
        <MarkdownContent content={card?.content || ''} isFocused={false} />
      </div>
      {#if manualKeywords.length > 0}
        <div 
          class="note-tags" 
          class:tag-visible={isTagVisible}
          style="opacity: {isTagVisible ? 1 : 0}; pointer-events: {isTagVisible ? 'auto' : 'none'}; transition: opacity 0.25s ease;"
        >
          {#each manualKeywords as tag}
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <span 
              class="note-tag" 
              class:active={Boolean(getActiveTagColor(tag, activeConstellationTags))}
              style={getTagPillStyle(tag, activeConstellationTags)}
              role="button" 
              tabindex="0"
              on:click|stopPropagation={() => handleTagClick(tag)}
            >#{tag}</span>
          {/each}
        </div>
      {/if}
      {#if resolvedDisplayTime}
        <div class="note-footer">
          <span class="card-time">{resolvedDisplayTime}</span>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .note-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 4px;
    margin-bottom: 2px;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.25s ease;
  }

  .note-tags.tag-visible {
    opacity: 1;
    pointer-events: auto;
  }

  .note-tags.focused {
    opacity: 1;
    pointer-events: auto;
    margin-top: 8px;
    margin-bottom: 8px;
    gap: 6px;
  }

  .note-tag {
    font-size: 10px;
    font-weight: 500;
    color: #0284c7;
    background: rgba(56, 189, 248, 0.12);
    padding: 1px 6px;
    border-radius: 4px;
    letter-spacing: 0.02em;
    user-select: none;
    white-space: nowrap;
    cursor: pointer;
    transition: background 0.2s ease, transform 0.15s ease, color 0.2s ease;
  }

  .note-tag:hover {
    background: rgba(56, 189, 248, 0.24);
    color: #0369a1;
  }

  .note-tag.focused {
    font-size: 12px;
    padding: 2px 8px;
    border-radius: 6px;
  }

  .note-tag.active {
    background: #0284c7 !important;
    color: #ffffff !important;
    box-shadow: 0 0 10px rgba(2, 132, 199, 0.6) !important;
  }
  .note-card-body {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    position: relative;
    z-index: 1;
    overflow: hidden;
    text-align: left;
    box-sizing: border-box;
  }

  .note-card-body.focused {
    flex: 1;
    min-height: 0;
    height: 100%;
    max-height: 100%;
    overflow: hidden;
  }

  /* --- Focused Mode (420x420) --- */
  .note-focused-header {
    flex-shrink: 0;
    margin-bottom: 2px;
  }

  .note-header-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 6px;
  }

  .note-badge {
    font-size: 13px;
    line-height: 1;
    flex-shrink: 0;
    user-select: none;
  }

  .note-focused-title {
    margin: 0 0 6px 0;
    font-size: 1.15rem;
    font-weight: 700;
    line-height: 1.35;
    letter-spacing: -0.015em;
    color: var(--card-text-color, rgba(22, 22, 22, 0.96));
    word-break: break-word;
  }

  .note-divider {
    width: 100%;
    height: 1px;
    background: linear-gradient(to right, rgba(0, 0, 0, 0.09), rgba(0, 0, 0, 0.02));
    margin-top: 8px;
    margin-bottom: 12px;
  }

  .note-focused-body {
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
    padding-right: 4px;
  }

  .note-focused-body::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }


  /* --- Scattering Mode (200x190) --- */
  .note-preview {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    text-align: left;
  }

  .note-header-row {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    margin-bottom: 6px;
    flex-shrink: 0;
    min-height: 20px;
  }

  .note-preview-title {
    font-size: 0.82rem;
    font-weight: 650;
    line-height: 1.3;
    color: var(--card-text-color, rgba(28, 28, 28, 0.95));
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    letter-spacing: -0.01em;
    flex: 1;
    min-width: 0;
  }

  .note-snippet {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    position: relative;
    mask-image: linear-gradient(to bottom, black 72%, transparent 100%);
    -webkit-mask-image: linear-gradient(to bottom, black 72%, transparent 100%);
  }

  .note-footer {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    margin-top: auto;
    padding-top: 4px;
    flex-shrink: 0;
  }

  .card-time {
    font-size: 11px;
    letter-spacing: 0.02em;
    color: var(--card-text-secondary, rgba(62, 62, 62, 0.7));
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    opacity: 0.85;
    flex-shrink: 0;
  }
</style>
