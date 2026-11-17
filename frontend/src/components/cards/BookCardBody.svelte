<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - BookCardBody Component
  Task: book_card_frontend_omnibar_and_body_172

  Ex Libris Bookplate & Immersive Marginalia Reader for Reading Note Cards (type: 'book'):
  - Scattered Mode (200x190): Warm cotton-paper Ex Libris bookplate with delicate inner
    border, subtle serif “ watermark, excerpt quote preview + personal marginalia preview,
    clickable foreground #tags, @ thread indicator, and 《书名》 + 作者 footer.
  - Focused Mode (420x420): Top header bar with book switcher dropdown (when multiple books
    exist in vault), linked author, and entry count badge; scrollable marginalia canvas
    rendering chronological notes for the selected book with Excerpt Quote Boxes,
    MarkdownContent commentary, CardImageGallery, and dblclick-to-edit per entry.
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import type { CosmicCard } from '../../types/card';
  import { getCardDisplayTime, formatPointTime, stageTime } from '../../lib/time';
  import { getCardTags, normalizeTag } from '../../lib/tagNetwork';
  import { toCosmicCard } from '../../lib/stageDecay';
  import {
    parseBookContentBlocks,
    extractBookHistory,
    findLinkedAuthor,
    getSameBookCards,
    cleanBookTitleDisplay,
    normalizeBookTitle,
  } from '../../lib/bookScroll';
  import MarkdownContent from './MarkdownContent.svelte';
  import CardImageGallery from './CardImageGallery.svelte';

  export let card: CosmicCard;
  export let allCards: CosmicCard[] = [];
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let thoughtTime: string = '';
  export let babelTime: string = '';
  export let displayTime: string = '';
  export let z: number | undefined = undefined;
  export let activeConstellationTags: Map<string, string> = new Map();
  export let activeConstellationTag: string | null = null;

  const dispatch = createEventDispatcher<{
    tagClick: { tag: string; cardId: string };
  }>();

  let vaultCards: CosmicCard[] = [];
  let selectedBookTitle: string = '';
  let lastCardId: string = '';
  let showBookSwitcher: boolean = false;

  declare const window: any;

  async function triggerEditor(entry: CosmicCard) {
    try {
      const fn = window?.go?.main?.App?.OpenCardInEditor;
      if (typeof fn === 'function') {
        await fn(entry.filePath || entry.id);
      }
    } catch (err) {
      console.warn('Failed to open book entry in editor:', err);
    }
  }

  async function refreshVaultCards() {
    try {
      const app = window?.go?.main?.App;
      if (app && typeof app.LoadLocalCards === 'function') {
        const loaded = await app.LoadLocalCards();
        if (Array.isArray(loaded)) {
          vaultCards = loaded.map((raw: any) =>
            toCosmicCard(raw, raw?.type || raw?.Type || 'thought', 'local')
          );
        }
      }
    } catch (err) {
      console.warn('Failed to load local vault cards for book scroll:', err);
    }
  }

  onMount(() => {
    refreshVaultCards();
  });

  $: if (isFocused) {
    refreshVaultCards();
  } else {
    showBookSwitcher = false;
  }

  $: cardBookTitle = cleanBookTitleDisplay(card?.bookTitle || '');

  $: if (card && (card.id !== lastCardId || !isFocused)) {
    lastCardId = card.id;
    selectedBookTitle = cardBookTitle;
  }

  // Merge live stage cards, vault cards, and current card (deduped by id, live takes priority)
  $: mergedBookPool = (() => {
    const map = new Map<string, CosmicCard>();
    if (Array.isArray(vaultCards)) {
      for (const vc of vaultCards) {
        if (vc?.id) map.set(vc.id, vc);
      }
    }
    if (Array.isArray(allCards)) {
      for (const ac of allCards) {
        if (ac?.id) map.set(ac.id, ac);
      }
    }
    if (card?.id) {
      map.set(card.id, card);
    }
    return Array.from(map.values());
  })();

  $: bookHistory = extractBookHistory(mergedBookPool);

  $: activeDisplayTitle = selectedBookTitle || cleanBookTitleDisplay(card?.title) || 'Reading Note';

  $: selectedBookAuthor = (() => {
    if (selectedBookTitle) {
      const linked = findLinkedAuthor(selectedBookTitle, bookHistory);
      if (linked) return linked;
    }
    return (card?.bookAuthor || '').trim();
  })();

  $: sameBookCards = (() => {
    if (selectedBookTitle) {
      const matched = getSameBookCards(selectedBookTitle, mergedBookPool);
      if (matched.length > 0) return matched;
    }
    return card ? [card] : [];
  })();

  $: blocks = parseBookContentBlocks(card?.content || '');
  $: manualKeywords = getCardTags(card);
  $: resolvedDisplayTime =
    displayTime ||
    thoughtTime ||
    babelTime ||
    formatPointTime(card?.createdAt || card?.spawnTime, $stageTime) ||
    getCardDisplayTime(card, $stageTime);

  $: currentZ = z !== undefined ? z : card?.z;
  $: hasActiveAura = Boolean(
    card?.auraTraces &&
      card.auraTraces.length > 0 &&
      card.auraTraces.some((t: any) => !t.duration || Date.now() - (t.hitTime || 0) < t.duration)
  );
  $: isForeground = currentZ !== undefined && currentZ <= 16;
  $: isTagVisible = isForeground || isHovered || isFocused || hasActiveAura;

  function getActiveTagColor(tag: string, tagsMap?: Map<string, string>): string | null {
    if (tagsMap && tagsMap.size > 0) {
      const clean = normalizeTag(tag);
      return tagsMap.get(clean) || tagsMap.get(tag) || null;
    }
    if (activeConstellationTag) {
      const clean = normalizeTag(tag);
      if (clean === normalizeTag(activeConstellationTag)) {
        return 'hsl(32, 85%, 52%)';
      }
    }
    return null;
  }

  function getTagPillStyle(tag: string, tagsMap?: Map<string, string>): string {
    const col = getActiveTagColor(tag, tagsMap);
    if (!col) return '';
    return `background: ${col} !important; color: #fff !important; box-shadow: 0 0 10px ${col} !important;`;
  }

  function handleTagClick(tag: string) {
    dispatch('tagClick', { tag, cardId: card?.id });
  }

  function formatAssetUrl(url: string): string {
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

  function resolveEntryImages(entry: CosmicCard): string[] {
    const raw = entry?.images || (entry as any)?.Images || [];
    return (Array.isArray(raw) ? raw : [raw])
      .map((item: any) => {
        if (typeof item === 'string') return item;
        if (item && typeof item === 'object') return item.url || item.data || item.path || '';
        return '';
      })
      .filter((u: string) => typeof u === 'string' && u.trim().length > 0)
      .map(formatAssetUrl);
  }

  function formatEntryTimestamp(entry: CosmicCard): string {
    return (
      formatPointTime(entry?.createdAt || entry?.spawnTime, $stageTime) ||
      getCardDisplayTime(entry, $stageTime) ||
      resolvedDisplayTime
    );
  }

  let threadCarouselIndex: number = 0;
  let threadCarouselTimer: ReturnType<typeof setInterval> | null = null;

  $: cardThreads = card?.threads && Array.isArray(card.threads) ? card.threads : [];
  $: if (threadCarouselIndex >= cardThreads.length) {
    threadCarouselIndex = 0;
  }

  function startThreadCarousel() {
    if (threadCarouselTimer) return;
    if (cardThreads.length > 1 && !isFocused) {
      threadCarouselTimer = setInterval(() => {
        threadCarouselIndex = (threadCarouselIndex + 1) % cardThreads.length;
      }, 2800);
    }
  }

  function stopThreadCarousel() {
    if (threadCarouselTimer) {
      clearInterval(threadCarouselTimer);
      threadCarouselTimer = null;
    }
  }

  $: if (cardThreads.length > 1 && !isFocused) {
    startThreadCarousel();
  } else {
    stopThreadCarousel();
  }

  onDestroy(() => {
    stopThreadCarousel();
  });

  function switchBook(bookTitle: string) {
    selectedBookTitle = bookTitle;
    showBookSwitcher = false;
  }
</script>

<div class="book-card-body" class:focused={isFocused} class:hovered={isHovered}>
  {#if !isFocused}
    <!-- Scattered Mode (200x190 · Ex Libris Bookplate) -->
    <div class="exlibris-plate">
      <div class="exlibris-inner-frame">
        <span class="quote-watermark" aria-hidden="true">“</span>

        <div class="scattered-content">
          {#if blocks.hasQuotes && blocks.quotes.length > 0}
            <blockquote class="scattered-quote">
              {blocks.quotes[0]}
            </blockquote>
            {#if blocks.comments.length > 0}
              <div class="scattered-comment">
                {blocks.comments[0]}
              </div>
            {/if}
          {:else}
            <div class="scattered-markdown">
              <MarkdownContent content={card?.content || ''} threads={card?.threads} isFocused={false} />
            </div>
          {/if}
        </div>

        {#if manualKeywords.length > 0}
          <div
            class="book-tags"
            class:tag-visible={isTagVisible}
            style="opacity: {isTagVisible ? 1 : 0}; pointer-events: {isTagVisible ? 'auto' : 'none'}; transition: opacity 0.25s ease;"
          >
            {#each manualKeywords.slice(0, 3) as tag}
              <!-- svelte-ignore a11y-click-events-have-key-events -->
              <!-- svelte-ignore a11y-no-static-element-interactions -->
              <span
                class="book-tag"
                class:active={Boolean(getActiveTagColor(tag, activeConstellationTags))}
                style={getTagPillStyle(tag, activeConstellationTags)}
                role="button"
                tabindex="0"
                on:click|stopPropagation={() => handleTagClick(tag)}
              >#{tag}</span>
            {/each}
          </div>
        {/if}

        <div class="exlibris-footer">
          <div class="book-citation" title="《{cardBookTitle || card?.title}》{card?.bookAuthor ? ` · ${card.bookAuthor}` : ''}">
            <span class="citation-title">《{cardBookTitle || card?.title || 'Reading Note'}》</span>
            {#if card?.bookAuthor}
              <span class="citation-author">{card.bookAuthor}</span>
            {/if}
          </div>
          {#if cardThreads.length > 0}
            <div class="book-thread-pill" title={`主题关联: ${cardThreads[threadCarouselIndex]?.title}`}>
              <span class="thread-symbol">@</span>
              <span class="thread-name">{cardThreads[threadCarouselIndex]?.title}</span>
              {#if cardThreads.length > 1}
                <span class="thread-count">{threadCarouselIndex + 1}/{cardThreads.length}</span>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>
  {:else}
    <!-- Focused Mode (420x420 · Immersive Marginalia & Same-Book Reader) -->
    <div class="book-focused-mode">
      <header class="focused-topbar">
        <div class="topbar-book-info">
          {#if bookHistory.length > 1}
            <div class="switcher-wrap">
              <button
                type="button"
                class="book-switcher-btn"
                on:click|stopPropagation={() => (showBookSwitcher = !showBookSwitcher)}
                title="切换藏书"
              >
                <span class="book-icon">📖</span>
                <span class="switcher-title">《{activeDisplayTitle}》</span>
                <span class="switcher-caret">{showBookSwitcher ? '▴' : '▾'}</span>
              </button>

              {#if showBookSwitcher}
                <div class="book-switcher-dropdown" on:wheel|stopPropagation>
                  {#each bookHistory as bk (bk.bookTitle)}
                    <!-- svelte-ignore a11y-click-events-have-key-events -->
                    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
                    <div
                      class="switcher-option"
                      class:current={normalizeBookTitle(bk.bookTitle) === normalizeBookTitle(selectedBookTitle)}
                      on:click|stopPropagation={() => switchBook(bk.bookTitle)}
                    >
                      <span class="opt-title">《{bk.bookTitle}》</span>
                      {#if bk.bookAuthor}
                        <span class="opt-author">{bk.bookAuthor}</span>
                      {/if}
                      {#if bk.count}
                        <span class="opt-count">{bk.count}</span>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {:else}
            <div class="static-book-heading">
              <span class="book-icon">📖</span>
              <span class="switcher-title">《{activeDisplayTitle}》</span>
            </div>
          {/if}

          {#if selectedBookAuthor}
            <span class="topbar-author">{selectedBookAuthor}</span>
          {/if}
        </div>

        <span class="entry-count-badge">共 {sameBookCards.length} 篇笔记</span>
      </header>

      <div class="marginalia-scroll-canvas" on:wheel|stopPropagation>
        {#each sameBookCards as entry, idx (entry.id)}
          {@const entryBlocks = parseBookContentBlocks(entry.content || '')}
          {@const entryImages = resolveEntryImages(entry)}
          {@const entryTags = getCardTags(entry)}
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <article
            class="marginalia-entry"
            on:dblclick|stopPropagation={() => triggerEditor(entry)}
            title="双击在外部编辑器中打开此篇笔记"
          >
            <div class="entry-meta-row">
              <span class="entry-time">{formatEntryTimestamp(entry)}</span>
              {#if entry.threads && entry.threads.length > 0}
                <div class="entry-threads">
                  {#each entry.threads as th}
                    <span class="entry-thread-chip">@{th.title}</span>
                  {/each}
                </div>
              {/if}
            </div>

            {#if entryBlocks.hasQuotes}
              <div class="entry-quotes">
                {#each entryBlocks.quotes as q}
                  <blockquote class="excerpt-quote-box">
                    <span class="quote-ornament" aria-hidden="true">“</span>
                    <p class="quote-text">{q}</p>
                  </blockquote>
                {/each}
              </div>
            {/if}

            {#if entryBlocks.comments.length > 0}
              <div class="entry-marginalia">
                {#each entryBlocks.comments as comment}
                  <div class="marginalia-paragraph">
                    <MarkdownContent content={comment} threads={entry.threads} isFocused={true} />
                  </div>
                {/each}
              </div>
            {:else if !entryBlocks.hasQuotes && entry.content}
              <div class="entry-marginalia">
                <MarkdownContent content={entry.content} threads={entry.threads} isFocused={true} />
              </div>
            {/if}

            {#if entryTags.length > 0}
              <div class="book-tags focused">
                {#each entryTags as tag}
                  <!-- svelte-ignore a11y-click-events-have-key-events -->
                  <!-- svelte-ignore a11y-no-static-element-interactions -->
                  <span
                    class="book-tag focused"
                    class:active={Boolean(getActiveTagColor(tag, activeConstellationTags))}
                    style={getTagPillStyle(tag, activeConstellationTags)}
                    role="button"
                    tabindex="0"
                    on:click|stopPropagation={() => handleTagClick(tag)}
                  >#{tag}</span>
                {/each}
              </div>
            {/if}

            {#if entryImages.length > 0}
              <div class="entry-gallery">
                <CardImageGallery images={entryImages} />
              </div>
            {/if}
          </article>

          {#if idx < sameBookCards.length - 1}
            <div class="bookplate-divider" aria-hidden="true">─── ❦ ───</div>
          {/if}
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .book-card-body {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
    z-index: 1;
    color: #3b2818;
    text-align: left;
    box-sizing: border-box;
  }

  /* ============================================================================
     SCATTERED MODE (200x190 · Ex Libris Bookplate)
     ============================================================================ */
  .exlibris-plate {
    width: 100%;
    height: 100%;
    background: linear-gradient(175deg, #faf7f0 0%, #f3ede0 100%);
    border-radius: 10px;
    padding: 5px;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .exlibris-inner-frame {
    position: relative;
    flex: 1;
    min-height: 0;
    border: 1px solid rgba(154, 118, 78, 0.28);
    border-radius: 7px;
    padding: 8px 9px 7px;
    display: flex;
    flex-direction: column;
    box-sizing: border-box;
    overflow: hidden;
  }

  .quote-watermark {
    position: absolute;
    top: -10px;
    right: 6px;
    font-family: "Songti SC", "Noto Serif SC", Georgia, serif;
    font-size: 54px;
    line-height: 1;
    color: rgba(140, 92, 46, 0.11);
    pointer-events: none;
    user-select: none;
    z-index: 0;
  }

  .scattered-content {
    position: relative;
    z-index: 1;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .scattered-quote {
    margin: 0;
    padding: 2px 0 2px 8px;
    border-left: 2.5px solid #9a6b3b;
    font-family: "Songti SC", "Noto Serif SC", Georgia, serif;
    font-size: 11.5px;
    line-height: 1.45;
    color: #3d2817;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .scattered-comment {
    font-size: 11px;
    line-height: 1.4;
    color: #6b533e;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .scattered-markdown {
    font-size: 11.5px;
    line-height: 1.45;
    color: #3d2817;
    overflow: hidden;
  }

  .book-tags {
    position: relative;
    z-index: 1;
    display: flex;
    flex-wrap: wrap;
    gap: 3px;
    margin-top: 4px;
    margin-bottom: 2px;
  }

  .book-tags.focused {
    opacity: 1;
    pointer-events: auto;
    margin-top: 8px;
    margin-bottom: 4px;
    gap: 6px;
  }

  .book-tag {
    font-size: 9.5px;
    font-weight: 500;
    color: #8c5222;
    background: rgba(154, 118, 78, 0.13);
    padding: 1px 5px;
    border-radius: 4px;
    user-select: none;
    white-space: nowrap;
    cursor: pointer;
    transition: background 0.2s ease, color 0.2s ease;
  }

  .book-tag:hover {
    background: rgba(154, 118, 78, 0.24);
    color: #5d3310;
  }

  .book-tag.focused {
    font-size: 11.5px;
    padding: 2px 8px;
    border-radius: 6px;
  }

  .book-tag.active {
    background: #8c5222 !important;
    color: #fff !important;
  }

  .exlibris-footer {
    position: relative;
    z-index: 1;
    margin-top: auto;
    padding-top: 5px;
    border-top: 1px dashed rgba(154, 118, 78, 0.26);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 4px;
    min-width: 0;
  }

  .book-citation {
    display: flex;
    align-items: baseline;
    gap: 4px;
    min-width: 0;
    flex: 1;
  }

  .citation-title {
    font-family: "Songti SC", "Noto Serif SC", Georgia, serif;
    font-size: 10.5px;
    font-weight: 700;
    color: #5c3a1e;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .citation-author {
    font-size: 9.5px;
    color: #8a6c4f;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex-shrink: 1;
  }

  .book-thread-pill {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    font-size: 9.5px;
    color: #7c4d25;
    background: rgba(154, 118, 78, 0.12);
    padding: 1px 5px;
    border-radius: 6px;
    max-width: 68px;
    flex-shrink: 0;
  }

  .thread-symbol {
    font-weight: 700;
  }

  .thread-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .thread-count {
    font-size: 8.5px;
    opacity: 0.75;
  }

  /* ============================================================================
     FOCUSED MODE (420x420 · Immersive Marginalia & Same-Book Reader)
     ============================================================================ */
  .book-focused-mode {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: linear-gradient(175deg, #faf7f0 0%, #f3ede0 100%);
    border: 1px solid rgba(154, 118, 78, 0.28);
    border-radius: 16px;
    padding: 16px 18px;
    box-sizing: border-box;
    overflow: hidden;
  }

  .focused-topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding-bottom: 10px;
    margin-bottom: 12px;
    border-bottom: 1.5px solid rgba(154, 118, 78, 0.25);
    flex-shrink: 0;
    position: relative;
    z-index: 10;
  }

  .topbar-book-info {
    display: flex;
    align-items: baseline;
    gap: 8px;
    min-width: 0;
    flex: 1;
  }

  .switcher-wrap {
    position: relative;
    min-width: 0;
  }

  .book-switcher-btn,
  .static-book-heading {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: transparent;
    border: none;
    padding: 2px 4px;
    margin: 0;
    border-radius: 6px;
    font-family: "Songti SC", "Noto Serif SC", Georgia, serif;
    font-size: 15px;
    font-weight: 700;
    color: #4a2e16;
    max-width: 100%;
  }

  .book-switcher-btn {
    cursor: pointer;
    transition: background 0.15s ease;
  }

  .book-switcher-btn:hover {
    background: rgba(154, 118, 78, 0.12);
  }

  .book-icon {
    font-size: 14px;
    flex-shrink: 0;
  }

  .switcher-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .switcher-caret {
    font-size: 11px;
    color: #8c6239;
    flex-shrink: 0;
  }

  .book-switcher-dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    min-width: 220px;
    max-width: 300px;
    max-height: 200px;
    overflow-y: auto;
    background: rgba(252, 249, 242, 0.98);
    border: 1px solid rgba(154, 118, 78, 0.32);
    border-radius: 10px;
    box-shadow: 0 12px 28px rgba(45, 28, 12, 0.18);
    padding: 4px;
    z-index: 50;
  }

  .switcher-option {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 9px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 12px;
    color: #3b2818;
  }

  .switcher-option:hover,
  .switcher-option.current {
    background: rgba(154, 118, 78, 0.14);
  }

  .opt-title {
    font-family: "Songti SC", "Noto Serif SC", Georgia, serif;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .opt-author {
    font-size: 11px;
    color: #8a6c4f;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
  }

  .opt-count {
    font-size: 10px;
    color: #8c5222;
    background: rgba(154, 118, 78, 0.12);
    padding: 1px 5px;
    border-radius: 8px;
    flex-shrink: 0;
  }

  .topbar-author {
    font-size: 12px;
    color: #7c5c3e;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .entry-count-badge {
    font-size: 11px;
    font-weight: 600;
    color: #7c4d25;
    background: rgba(154, 118, 78, 0.13);
    padding: 2px 8px;
    border-radius: 10px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .marginalia-scroll-canvas {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding-right: 4px;
    scrollbar-width: thin;
    scrollbar-color: rgba(154, 118, 78, 0.35) transparent;
  }

  .marginalia-scroll-canvas::-webkit-scrollbar {
    width: 5px;
  }

  .marginalia-scroll-canvas::-webkit-scrollbar-thumb {
    background: rgba(154, 118, 78, 0.35);
    border-radius: 8px;
  }

  .marginalia-entry {
    padding: 8px 6px;
    border-radius: 8px;
    transition: background 0.18s ease;
    cursor: text;
  }

  .marginalia-entry:hover {
    background: rgba(154, 118, 78, 0.05);
  }

  .entry-meta-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 8px;
  }

  .entry-time {
    font-size: 11px;
    font-weight: 600;
    color: #8a6c4f;
    letter-spacing: 0.02em;
  }

  .entry-threads {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .entry-thread-chip {
    font-size: 10.5px;
    color: #7c4d25;
    background: rgba(154, 118, 78, 0.12);
    padding: 1px 6px;
    border-radius: 6px;
  }

  .entry-quotes {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 10px;
  }

  .excerpt-quote-box {
    position: relative;
    margin: 0;
    padding: 10px 12px 10px 14px;
    background: rgba(255, 253, 248, 0.88);
    border: 1px solid rgba(154, 118, 78, 0.25);
    border-left: 3.5px solid #9a6b3b;
    border-radius: 4px 8px 8px 4px;
    box-shadow: 0 2px 6px rgba(62, 39, 35, 0.04);
  }

  .quote-ornament {
    position: absolute;
    top: -4px;
    right: 8px;
    font-family: "Songti SC", "Noto Serif SC", Georgia, serif;
    font-size: 28px;
    line-height: 1;
    color: rgba(154, 118, 78, 0.2);
    pointer-events: none;
    user-select: none;
  }

  .quote-text {
    margin: 0;
    font-family: "Songti SC", "Noto Serif SC", Georgia, serif;
    font-size: 13.5px;
    line-height: 1.65;
    color: #362313;
    white-space: pre-wrap;
  }

  .entry-marginalia {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 13.5px;
    line-height: 1.6;
    color: #2f2014;
  }

  .entry-gallery {
    margin-top: 10px;
  }

  .bookplate-divider {
    text-align: center;
    color: rgba(154, 118, 78, 0.45);
    font-size: 13px;
    margin: 10px 0;
    user-select: none;
  }
</style>
