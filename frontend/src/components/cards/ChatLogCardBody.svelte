<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - ChatLogCardBody Component
  Task: chat_log_card_frontend_032

  Renders the dialogue body of chat log cards in:
  - Scattering state (200x190) with screenplay dialogue snippet preview & gradient fade
  - Focused state (420x420) with full screenplay typography, metadata dateline, and natural scroll stream
  - Pure screenplay typography: character name cues, indented dialogue, elegant spacing
  - Strictly without cheesy badges or special copy buttons
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { CosmicCard } from '../../types/card';
  import { parseDialogue } from '../../lib/chatlog';
  import { formatIntervalTime, formatPointTime, stageTime } from '../../lib/time';
  import { getCardTags, normalizeTag } from '../../lib/tagNetwork';
  import CardImageGallery from './CardImageGallery.svelte';

  export let card: CosmicCard;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let thoughtTime: string = '';
  export let babelTime: string = '';
  export let displayTime: string = '';
  export let isEnriching: boolean = false;
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

  export function formatAssetUrl(url: string): string {
    if (!url) return '';
    if (url.startsWith('data:') || url.startsWith('http://') || url.startsWith('https://')) {
      return url;
    }
    if (url.startsWith('/api/assets/')) {
      return url;
    }
    const cleanName = url.startsWith('/') ? url.slice(1) : url;
    return `/api/assets/${cleanName}`;
  }

  function handleImageError(e: Event) {
    const el = e.currentTarget as HTMLElement | null;
    if (el) el.style.display = 'none';
  }

  $: enriching = isEnriching || !!card?.isEnriching;
  $: turns = parseDialogue(card?.content || '');
  $: title = card?.title && card.title !== 'Thought' && card.title !== 'Card' && card.title !== 'Babel Echo' ? card.title : '';
  $: participants = Array.isArray(card?.participants) ? card.participants : [];
  $: timeStr = displayTime || formatIntervalTime(card?.startedAt, card?.endedAt, $stageTime) || thoughtTime || babelTime;
  $: scatterTime = displayTime || formatPointTime(card?.startedAt || card?.createdAt || card?.spawnTime, $stageTime) || thoughtTime || babelTime;

  $: rawImages = card?.images || (card as any)?.Images || [];
  $: images = (Array.isArray(rawImages) ? rawImages : [rawImages])
    .map(item => {
      if (typeof item === 'string') return item;
      if (item && typeof item === 'object') {
        return item.url || item.data || item.path || '';
      }
      return '';
    })
    .filter((url: string) => typeof url === 'string' && url.trim().length > 0)
    .map(formatAssetUrl);

  $: hasImages = images.length > 0;
</script>

<div class="chat-log-card-body" class:focused={isFocused} class:hovered={isHovered} class:has-images={hasImages} class:enriching={enriching}>
  {#if isFocused}
    <!-- Focused mode: 420x420 Screenplay Layout -->
    <header class="screenplay-header">
      {#if title}
        <h2 class="screenplay-title">{title}</h2>
      {/if}
      <div class="screenplay-meta">
        {#if participants.length > 0}
          <span class="meta-participants">{participants.join(' · ')}</span>
        {/if}
        {#if timeStr}
          <span class="meta-time">{timeStr}</span>
        {/if}
      </div>
      <div class="screenplay-divider" aria-hidden="true"></div>
    </header>

    <div class="screenplay-stream">
      {#if hasImages}
        <CardImageGallery {images} />
      {/if}

      {#each turns as turn, i (turn.speaker + i)}
        <div class="screenplay-turn">
          {#if turn.speaker}
            <div class="screenplay-character">{turn.speaker}</div>
          {/if}
          <div class="screenplay-dialogue">
            {#each turn.lines as line}
              <p class="dialogue-line">{line}</p>
            {/each}
          </div>
        </div>
      {/each}
    </div>

    {#if manualKeywords.length > 0}
      <div class="chatlog-tags focused">
        {#each manualKeywords as tag}
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <span 
            class="chatlog-tag focused" 
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
    <!-- Scattering mode: 200x190 Screenplay Preview -->
    <div class="screenplay-preview">
      {#if title || hasImages}
        <div class="preview-header-row">
          {#if title}
            <div class="preview-title" title={title}>{title}</div>
          {/if}
          {#if hasImages}
            <div
              class="polaroid-stamp"
              class:stacked={images.length > 1}
              title="{images.length} attachment{images.length > 1 ? 's' : ''}"
            >
              <img
                src={images[0]}
                alt="Thumbnail"
                class="stamp-thumb"
                on:error={handleImageError}
              />
              {#if images.length > 1}
                <span class="stamp-count">{images.length}</span>
              {/if}
            </div>
          {/if}
        </div>
      {/if}
      <div class="preview-turns">
        {#each turns.slice(0, 3) as turn, i (turn.speaker + i)}
          <div class="preview-turn">
            {#if turn.speaker}
              <span class="preview-character">{turn.speaker}</span>
            {/if}
            {#each turn.lines.slice(0, 2) as line}
              <p class="preview-line">{line}</p>
            {/each}
          </div>
        {/each}
      </div>
      {#if manualKeywords.length > 0}
        <div 
          class="chatlog-tags" 
          class:tag-visible={isTagVisible}
          style="opacity: {isTagVisible ? 1 : 0}; pointer-events: {isTagVisible ? 'auto' : 'none'}; transition: opacity 0.25s ease;"
        >
          {#each manualKeywords as tag}
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <span 
              class="chatlog-tag" 
              class:active={Boolean(getActiveTagColor(tag, activeConstellationTags))}
              style={getTagPillStyle(tag, activeConstellationTags)}
              role="button" 
              tabindex="0"
              on:click|stopPropagation={() => handleTagClick(tag)}
            >#{tag}</span>
          {/each}
        </div>
      {/if}
      {#if scatterTime}
        <div class="preview-footer">
          <span class="card-time">{scatterTime}</span>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .chatlog-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 4px;
    margin-bottom: 2px;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.25s ease;
  }

  .chatlog-tags.tag-visible {
    opacity: 1;
    pointer-events: auto;
  }

  .chatlog-tags.focused {
    opacity: 1;
    pointer-events: auto;
    margin-top: 8px;
    margin-bottom: 8px;
    gap: 6px;
  }

  .chatlog-tag {
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

  .chatlog-tag:hover {
    background: rgba(56, 189, 248, 0.24);
    color: #0369a1;
  }

  .chatlog-tag.focused {
    font-size: 12px;
    padding: 2px 8px;
    border-radius: 6px;
  }

  .chatlog-tag.active {
    background: #0284c7 !important;
    color: #ffffff !important;
    box-shadow: 0 0 10px rgba(2, 132, 199, 0.6) !important;
  }
  .chat-log-card-body {
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

  /* Focused mode: natural scrollable stream */
  .chat-log-card-body.focused {
    flex: 1;
    min-height: 0;
    height: 100%;
    max-height: 100%;
    overflow: hidden;
  }

  /* --- Focused Screenplay Typography --- */
  .screenplay-header {
    flex-shrink: 0;
    margin-bottom: 2px;
  }

  .screenplay-title {
    margin: 0 0 6px 0;
    font-size: 1.15rem;
    font-weight: 700;
    line-height: 1.35;
    letter-spacing: -0.015em;
    color: var(--card-text-color, rgba(22, 22, 22, 0.96));
    word-break: break-word;
  }

  .screenplay-meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 0.74rem;
    color: rgba(0, 0, 0, 0.46);
    font-variant-numeric: tabular-nums;
  }

  .meta-participants {
    font-weight: 550;
    letter-spacing: 0.03em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .meta-time {
    flex-shrink: 0;
    white-space: nowrap;
    letter-spacing: 0.02em;
  }

  .screenplay-divider {
    width: 100%;
    height: 1px;
    background: linear-gradient(to right, rgba(0, 0, 0, 0.09), rgba(0, 0, 0, 0.02));
    margin-top: 10px;
    margin-bottom: 12px;
  }

  .screenplay-stream {
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

  .screenplay-stream::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  .screenplay-turn {
    margin-bottom: 14px;
  }

  .screenplay-turn:last-child {
    margin-bottom: 6px;
  }

  .screenplay-character {
    font-size: 0.76rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: rgba(0, 0, 0, 0.72);
    margin-bottom: 4px;
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  }

  .screenplay-dialogue {
    margin-left: 10px;
    border-left: 2px solid rgba(0, 0, 0, 0.07);
    padding-left: 10px;
  }

  .dialogue-line {
    margin: 0 0 6px 0;
    font-size: 0.91rem;
    line-height: 1.6;
    color: var(--card-text-color, rgba(32, 32, 32, 0.95));
    font-weight: 450;
    word-break: break-word;
  }

  .dialogue-line:last-child {
    margin-bottom: 0;
  }

  /* --- Scattering Screenplay Typography (200x190) --- */
  .screenplay-preview {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    text-align: left;
  }

  .preview-header-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    width: 100%;
    margin-bottom: 6px;
    padding: 2px 4px 2px 0;
    box-sizing: border-box;
    flex-shrink: 0;
    min-height: 24px;
  }

  .preview-title {
    font-size: 0.82rem;
    font-weight: 650;
    line-height: 1.3;
    color: var(--card-text-color, rgba(28, 28, 28, 0.95));
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-bottom: 0;
    letter-spacing: -0.01em;
    flex: 1;
    min-width: 0;
  }

  /* Polaroid Micro Stamp Badge (Option B) */
  .polaroid-stamp {
    position: relative;
    width: 22px;
    height: 22px;
    min-width: 22px;
    min-height: 22px;
    padding: 2px;
    box-sizing: border-box;
    background: #ffffff;
    border-radius: 4px;
    transform: rotate(-3deg);
    transform-origin: center center;
    margin-left: auto;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.18), 0 0.5px 1px rgba(0, 0, 0, 0.12);
    transition: transform 0.2s cubic-bezier(0.2, 0.8, 0.2, 1), box-shadow 0.2s cubic-bezier(0.2, 0.8, 0.2, 1);
    user-select: none;
    cursor: default;
  }

  /* Stacked shadow if images.length > 1 */
  .polaroid-stamp.stacked {
    box-shadow:
      1.5px 1.5px 0 0 #ffffff,
      1.5px 1.5px 0 0.5px rgba(0, 0, 0, 0.14),
      3px 3px 0 0 #f5f5f5,
      3px 3px 0 0.5px rgba(0, 0, 0, 0.1),
      0 2px 4px rgba(0, 0, 0, 0.16);
  }

  /* Slight elevation on hover */
  .polaroid-stamp:hover {
    transform: rotate(-1deg) translateY(-1.5px) scale(1.06);
    box-shadow: 0 3px 7px rgba(0, 0, 0, 0.22), 0 1px 2px rgba(0, 0, 0, 0.14);
  }

  .polaroid-stamp.stacked:hover {
    transform: rotate(-1deg) translateY(-1.5px) scale(1.06);
    box-shadow:
      2px 2px 0 0 #ffffff,
      2px 2px 0 0.5px rgba(0, 0, 0, 0.16),
      4px 4px 0 0 #f5f5f5,
      4px 4px 0 0.5px rgba(0, 0, 0, 0.12),
      0 4px 9px rgba(0, 0, 0, 0.22);
  }

  .chat-log-card-body.hovered .polaroid-stamp:not(:hover) {
    transform: rotate(-2deg) translateY(-0.8px);
    box-shadow: 0 2px 5px rgba(0, 0, 0, 0.2), 0 0.5px 1.5px rgba(0, 0, 0, 0.12);
  }

  .chat-log-card-body.hovered .polaroid-stamp.stacked:not(:hover) {
    transform: rotate(-2deg) translateY(-0.8px);
    box-shadow:
      1.5px 1.5px 0 0 #ffffff,
      1.5px 1.5px 0 0.5px rgba(0, 0, 0, 0.15),
      3.5px 3.5px 0 0 #f5f5f5,
      3.5px 3.5px 0 0.5px rgba(0, 0, 0, 0.11),
      0 3px 6px rgba(0, 0, 0, 0.18);
  }

  .stamp-thumb {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 2px;
    display: block;
    pointer-events: none;
    user-select: none;
    -webkit-user-drag: none;
  }

  .stamp-count {
    position: absolute;
    top: -4px;
    right: -4px;
    min-width: 12px;
    height: 12px;
    padding: 0 2.5px;
    background: rgba(20, 20, 20, 0.86);
    color: #ffffff;
    font-size: 7.5px;
    font-weight: 700;
    line-height: 12px;
    text-align: center;
    border-radius: 6px;
    box-sizing: border-box;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.35);
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    pointer-events: none;
    letter-spacing: -0.02em;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2;
  }

  .preview-turns {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
    overflow: hidden;
    mask-image: linear-gradient(to bottom, black 70%, transparent 100%);
    -webkit-mask-image: linear-gradient(to bottom, black 70%, transparent 100%);
  }

  .preview-turn {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .preview-character {
    font-size: 0.67rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: rgba(0, 0, 0, 0.55);
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    display: block;
    line-height: 1.2;
    transition: color 0.15s ease;
  }

  .chat-log-card-body.hovered .preview-character {
    color: rgba(0, 0, 0, 0.75);
  }

  .preview-line {
    font-size: 0.76rem;
    line-height: 1.38;
    color: var(--card-text-color, rgba(38, 38, 38, 0.92));
    margin: 0;
    padding-left: 6px;
    border-left: 1.5px solid rgba(0, 0, 0, 0.08);
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  /* --- Shimmer Animation when isEnriching --- */
  @keyframes shimmer-sweep {
    0% {
      background-position: -200% 0;
    }
    100% {
      background-position: 200% 0;
    }
  }

  @keyframes pulse-subtle {
    0%, 100% {
      opacity: 0.75;
    }
    50% {
      opacity: 1;
    }
  }

  .chat-log-card-body.enriching .preview-title,
  .chat-log-card-body.enriching .screenplay-title {
    background: linear-gradient(
      90deg,
      rgba(70, 70, 70, 0.6) 0%,
      rgba(15, 15, 15, 0.95) 35%,
      rgba(130, 130, 130, 0.3) 50%,
      rgba(15, 15, 15, 0.95) 65%,
      rgba(70, 70, 70, 0.6) 100%
    );
    background-size: 200% 100%;
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
    animation: shimmer-sweep 2.2s infinite ease-in-out, pulse-subtle 1.8s infinite ease-in-out;
  }

  .chat-log-card-body.enriching::after {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    pointer-events: none;
    background: linear-gradient(
      115deg,
      transparent 0%,
      transparent 35%,
      rgba(255, 255, 255, 0.35) 50%,
      transparent 65%,
      transparent 100%
    );
    background-size: 200% 100%;
    animation: shimmer-sweep 2.2s infinite linear;
    border-radius: 6px;
  }

  .chat-log-card-body.enriching .preview-turns,
  .chat-log-card-body.enriching .screenplay-stream {
    opacity: 0.85;
    transition: opacity 0.3s ease;
  }

  .preview-footer {
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
