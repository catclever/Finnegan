<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - BabelCardBody Component
  Task: modularize_backend_and_frontend_architecture_074
  Renders the dedicated body for babel cards (card.type === 'babel').
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { getCardDisplayTime, stageTime } from '../../lib/time';
  import { getCardTags, normalizeTag } from '../../lib/tagNetwork';

  export let card: any;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let babelTime: string = '';
  export let thoughtTime: string = '';
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

  function getHostname(url: string | undefined): string {
    if (!url) return '';
    try {
      return new URL(url).hostname || 'Source';
    } catch {
      return 'Source';
    }
  }

  $: hostname = getHostname(card.sourceUrl);

  $: hasTranslation = Boolean(
    card.translatedText &&
    card.translatedText.trim() !== '' &&
    card.originalText &&
    card.translatedText.trim() !== card.originalText.trim()
  );

  function handleOpenSource(e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    if (card.sourceUrl) {
      BrowserOpenURL(card.sourceUrl);
    }
  }

  function handleAvatarError(e: Event) {
    const el = e.currentTarget as HTMLElement | null;
    if (el) el.style.display = 'none';
  }

  $: resolvedDisplayTime = displayTime || babelTime || thoughtTime || getCardDisplayTime(card, $stageTime);

  $: isBluesky = card.origin === 'atproto' || (typeof card.sourceUrl === 'string' && card.sourceUrl.includes('bsky.app'));

  $: staggerDelay = -(((card?.id ? card.id.split('').reduce((a: number, b: string) => a + b.charCodeAt(0), 0) : 0) % 80) / 10);

  function handleFetchComments(e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    if ((window as any).runtime?.EventsEmit) {
      (window as any).runtime.EventsEmit('atproto:fetch_comments', { uri: card.sourceUrl || card.id, parentCardId: card.id });
    }
  }
</script>

<div class="babel-card-body" class:focused={isFocused} class:hovered={isHovered}>
  <div class="babel-header">
    <div class="header-left">
      {#if card.sourceUrl}
        <a 
          href={card.sourceUrl} 
          target="_blank" 
          class="source-link" 
          on:click={handleOpenSource}
          title={card.sourceUrl}
        >
          {#if isBluesky}
            <span class="domain-icon">🦋</span> @{card.authorName || 'bsky.app'}
          {:else}
            <span class="domain-icon">🌐</span> {hostname}
          {/if}
        </a>
      {/if}
    </div>
  </div>

  {#if hasTranslation}
    <div class="babel-content twinkle-container" class:focused={isFocused}>
      <div class="twinkle-text lang-trans" style="animation-delay: {staggerDelay}s;">
        <p class="translated">{card.translatedText}</p>
      </div>
      <div class="twinkle-text lang-orig" style="animation-delay: {staggerDelay}s;">
        <p class="original">{card.originalText}</p>
      </div>
    </div>
  {:else}
    <div class="babel-content single-container" class:focused={isFocused}>
      <p class="pure-original">{card.originalText || card.content}</p>
    </div>
  {/if}

  {#if manualKeywords.length > 0}
    <div 
      class="babel-tags" 
      class:focused={isFocused}
      class:tag-visible={isTagVisible}
      style={!isFocused ? `opacity: ${isTagVisible ? 1 : 0}; pointer-events: ${isTagVisible ? 'auto' : 'none'}; transition: opacity 0.25s ease;` : ''}
    >
      {#each manualKeywords as tag}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <span 
          class="babel-tag" 
          class:focused={isFocused}
          class:active={Boolean(getActiveTagColor(tag, activeConstellationTags))}
          style={getTagPillStyle(tag, activeConstellationTags)}
          role="button" 
          tabindex="0"
          on:click|stopPropagation={() => handleTagClick(tag)}
        >#{tag}</span>
      {/each}
    </div>
  {/if}

  <div class="babel-footer">
    <div class="footer-left">
      {#if card.authorName || card.authorAvatar}
        <div class="author-pill" title={card.authorName || 'Author'}>
          {#if card.authorAvatar}
            <img src={card.authorAvatar} alt="" class="author-avatar" on:error={handleAvatarError} />
          {/if}
          {#if card.authorName}
            <span class="author-name">{card.authorName}</span>
          {/if}
        </div>
      {/if}
    </div>

    {#if isFocused && isBluesky}
      <div class="bsky-interactions">
        <button on:click={handleFetchComments} title="Reply">
          💬 {card.replyCount || card.commentsCount || ''}
        </button>
        {#if card.repostCount !== undefined}
          <button title="Repost">
            🔁 {card.repostCount}
          </button>
        {/if}
        {#if card.likeCount !== undefined}
          <button title="Like">
            ♥ {card.likeCount}
          </button>
        {/if}
      </div>
    {/if}

    {#if resolvedDisplayTime}
      <span class="card-time">{resolvedDisplayTime}</span>
    {/if}
  </div>
</div>

<style>
  .babel-card-body {
    position: relative;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    z-index: 1;
  }

  .babel-header,
  .babel-content,
  .babel-footer {
    position: relative;
    z-index: 1;
  }

  .babel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .babel-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
  }

  .babel-card-body.focused .babel-footer {
    position: relative;
    margin-top: auto;
    padding-top: 16px;
    bottom: auto;
    left: auto;
    right: auto;
  }

  .babel-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 4px;
    margin-bottom: 2px;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.25s ease;
  }

  .babel-tags.tag-visible {
    opacity: 1;
    pointer-events: auto;
  }

  .babel-tags.focused {
    opacity: 1;
    pointer-events: auto;
    margin-top: 8px;
    margin-bottom: 8px;
    gap: 6px;
  }

  .babel-tag {
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

  .babel-tag:hover {
    background: rgba(56, 189, 248, 0.24);
    color: #0369a1;
  }

  .babel-tag.focused {
    font-size: 12px;
    padding: 2px 8px;
    border-radius: 6px;
  }

  .babel-tag.active {
    background: #0284c7 !important;
    color: #ffffff !important;
    box-shadow: 0 0 10px rgba(2, 132, 199, 0.6) !important;
  }

  .header-left,
  .footer-left {
    display: flex;
    align-items: center;
    gap: 6px;
    overflow: hidden;
    flex: 1;
    min-width: 0;
  }

  .author-pill {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 0.72rem;
    color: rgba(0, 0, 0, 0.65);
    background: rgba(255, 255, 255, 0.55);
    padding: 2px 7px;
    border-radius: 12px;
    max-width: 120px;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    border: 1px solid rgba(255, 255, 255, 0.7);
  }

  .author-avatar {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    object-fit: cover;
    flex-shrink: 0;
  }

  .author-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .source-link {
    font-size: 0.75rem;
    color: rgba(0, 0, 0, 0.5);
    text-decoration: none;
    display: flex;
    align-items: center;
    gap: 5px;
    background: rgba(255, 255, 255, 0.6);
    padding: 3px 8px;
    border-radius: 20px;
    transition: all 0.2s;
  }

  .source-link:hover {
    background: rgba(255, 255, 255, 0.9);
    color: rgba(0, 0, 0, 0.8);
  }

  :global(.card.temporary) .author-pill {
    background: rgba(255, 255, 255, 0.6);
    border: 1px solid rgba(226, 232, 240, 0.8);
  }

  :global(.card.temporary) .source-link {
    background: rgba(255, 255, 255, 0.6);
    border: 1px solid rgba(226, 232, 240, 0.75);
  }

  .card-time {
    font-size: 0.7rem;
    color: rgba(0, 0, 0, 0.42);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.02em;
    user-select: none;
    white-space: nowrap;
  }

  .bsky-interactions {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-left: auto;
    margin-right: 8px;
  }

  .bsky-interactions button {
    background: rgba(255, 255, 255, 0.4);
    border: none;
    border-radius: 12px;
    padding: 4px 8px;
    font-size: 0.75rem;
    color: rgba(0, 0, 0, 0.7);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    transition: all 0.2s ease;
    font-family: inherit;
    backdrop-filter: blur(4px);
  }

  .bsky-interactions button:hover {
    background: rgba(255, 255, 255, 0.8);
    color: rgba(0, 0, 0, 0.9);
  }

  .single-container {
    position: relative;
    width: 100%;
    height: 105px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .pure-original {
    font-size: 0.825rem;
    line-height: 1.55;
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
    font-weight: 450;
    display: -webkit-box;
    -webkit-line-clamp: 5;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
    margin: 0;
  }

  /* Twinkling effect */
  .twinkle-container {
    position: relative;
    width: 100%;
    height: 95px;
  }

  .twinkle-text {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .lang-trans {
    animation: twinkle-a 8s infinite cubic-bezier(0.4, 0, 0.2, 1);
  }

  .lang-orig {
    animation: twinkle-b 8s infinite cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* Stop animation completely when focused */
  .twinkle-container.focused .lang-trans,
  .twinkle-container.focused .lang-orig {
    animation: none !important;
    position: relative;
    opacity: 1;
    filter: none;
    height: auto;
    margin-bottom: 10px;
    display: block;
  }

  .twinkle-container.focused {
    height: auto;
    overflow: visible;
    max-height: none;
  }

  @keyframes twinkle-a {
    0%, 35% { opacity: 1; filter: blur(0px); z-index: 2; }
    45%, 85% { opacity: 0; filter: blur(8px); z-index: 1; }
    100% { opacity: 1; filter: blur(0px); z-index: 2; }
  }

  @keyframes twinkle-b {
    0%, 35% { opacity: 0; filter: blur(8px); z-index: 1; }
    45%, 85% { opacity: 1; filter: blur(0px); z-index: 2; }
    100% { opacity: 0; filter: blur(8px); z-index: 1; }
  }

  .babel-card-body.focused {
    flex: 1;
    min-height: 0;
  }

  .babel-card-body.focused .babel-content::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  .babel-card-body.focused .babel-content {
    justify-content: flex-start;
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
    text-align: left;
  }

  .babel-card-body.focused .twinkle-container,
  .babel-card-body.focused .single-container {
    height: auto;
    min-height: 0;
    overflow-y: auto !important;
    overflow-x: hidden !important;
  }

  .babel-card-body.focused .original,
  .babel-card-body.focused .translated,
  .babel-card-body.focused .pure-original {
    -webkit-line-clamp: unset !important;
    display: block !important;
    overflow: visible !important;
    text-overflow: unset !important;
    max-height: none !important;
    padding-bottom: 18px;
  }

  .babel-card-body.focused .translated {
    font-size: 1.1rem;
    line-height: 1.75;
    color: rgba(30, 30, 30, 0.98);
    font-weight: 500;
  }

  .babel-card-body.focused .original {
    font-size: 0.95rem;
    line-height: 1.65;
    color: rgba(60, 60, 60, 0.85);
    margin-top: 16px;
    padding-top: 14px;
    border-top: 1px dashed rgba(0, 0, 0, 0.15);
  }

  .babel-card-body.focused .pure-original {
    font-size: 1.15rem;
    line-height: 1.75;
    color: rgba(30, 30, 30, 0.98);
  }

  p {
    margin: 0;
    font-size: 0.825rem;
    line-height: 1.5;
    white-space: pre-wrap;
    text-align: left;
  }

  .original {
    font-size: 0.775rem;
    color: var(--card-text-secondary, rgba(70, 70, 70, 0.9));
    font-style: italic;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
  }

  .translated {
    font-size: 0.825rem;
    font-weight: 500;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
  }
</style>
