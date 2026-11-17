<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - BookMetaBar Subcomponent
  Task: book_card_frontend_omnibar_and_body_172

  Linked Book Metadata Bar rendered right above WritingPaper when `:book ` mode is active:
  - Exactly two inputs: 📖 书名 (bookTitle) and ✍️ 作者 (bookAuthor)
  - Remembers last used (bookTitle, bookAuthor) in localStorage ('finnegan_book_history_v1')
  - Sleek historical books dropdown on focus/click/input with keyboard navigation
  - Auto-fills linked bookAuthor on selection or exact known bookTitle match
  ======================================================================================
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import type { CosmicCard } from '../../types/card';
  import {
    extractBookHistory,
    findLinkedAuthor,
    cleanBookTitleDisplay,
    normalizeBookTitle,
    type BookMeta,
  } from '../../lib/bookScroll';

  const STORAGE_KEY = 'finnegan_book_history_v1';

  export let bookTitle: string = '';
  export let bookAuthor: string = '';
  export let cards: CosmicCard[] = [];
  export let localVaultPool: CosmicCard[] = [];
  export let bookHistory: BookMeta[] | undefined = undefined;

  let cachedHistory: BookMeta[] = [];
  let showDropdown: boolean = false;
  let isTitleFilterActive: boolean = false;
  let highlightedIdx: number = 0;
  let blurTimer: ReturnType<typeof setTimeout> | null = null;

  function loadCachedHistory(): BookMeta[] {
    try {
      if (typeof localStorage === 'undefined') return [];
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return [];
      const parsed = JSON.parse(raw);
      if (!Array.isArray(parsed)) return [];
      return parsed
        .filter((item) => item && typeof item.bookTitle === 'string' && item.bookTitle.trim())
        .map((item) => ({
          bookTitle: cleanBookTitleDisplay(item.bookTitle),
          bookAuthor: typeof item.bookAuthor === 'string' ? item.bookAuthor.trim() : '',
          lastUsedAt: typeof item.lastUsedAt === 'number' ? item.lastUsedAt : 0,
          count: typeof item.count === 'number' ? item.count : 0,
        }));
    } catch {
      return [];
    }
  }

  export function saveToHistory(title: string, author: string): void {
    const cleanTitle = cleanBookTitleDisplay(title);
    if (!cleanTitle) return;
    const cleanAuthor = (author || '').trim();
    const now = Date.now();
    const norm = normalizeBookTitle(cleanTitle);

    const current = loadCachedHistory();
    const existing = current.find((item) => normalizeBookTitle(item.bookTitle) === norm);
    const nextList: BookMeta[] = [
      {
        bookTitle: cleanTitle,
        bookAuthor: cleanAuthor || existing?.bookAuthor || '',
        lastUsedAt: now,
        count: (existing?.count || 0) + 1,
      },
      ...current.filter((item) => normalizeBookTitle(item.bookTitle) !== norm),
    ].slice(0, 50);

    cachedHistory = nextList;
    try {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(nextList));
      }
    } catch {
      // Ignore storage quota / privacy errors
    }
  }

  onMount(() => {
    cachedHistory = loadCachedHistory();
    const initialHistory =
      bookHistory ?? extractBookHistory([...(cards || []), ...(localVaultPool || [])], cachedHistory);

    if (!bookTitle.trim() && initialHistory.length > 0) {
      bookTitle = initialHistory[0].bookTitle;
      if (!bookAuthor.trim() && initialHistory[0].bookAuthor) {
        bookAuthor = initialHistory[0].bookAuthor;
      }
    }
  });

  $: mergedHistory =
    bookHistory ?? extractBookHistory([...(cards || []), ...(localVaultPool || [])], cachedHistory);

  $: filteredHistory = (() => {
    const q = isTitleFilterActive ? normalizeBookTitle(bookTitle) : '';
    if (!q) return mergedHistory;
    return mergedHistory.filter(
      (item) =>
        normalizeBookTitle(item.bookTitle).includes(q) ||
        (item.bookAuthor || '').toLowerCase().includes(q)
    );
  })();

  $: if (highlightedIdx >= filteredHistory.length) {
    highlightedIdx = Math.max(0, filteredHistory.length - 1);
  }

  function handleTitleFocus() {
    if (blurTimer) {
      clearTimeout(blurTimer);
      blurTimer = null;
    }
    isTitleFilterActive = false;
    showDropdown = true;
    highlightedIdx = 0;
  }

  function handleTitleClick() {
    if (blurTimer) {
      clearTimeout(blurTimer);
      blurTimer = null;
    }
    isTitleFilterActive = false;
    showDropdown = true;
  }

  function handleTitleBlur() {
    if (blurTimer) clearTimeout(blurTimer);
    blurTimer = setTimeout(() => {
      showDropdown = false;
    }, 150);
  }

  function handleTitleInput() {
    isTitleFilterActive = true;
    showDropdown = true;
    highlightedIdx = 0;

    if (!bookAuthor.trim()) {
      const linked = findLinkedAuthor(bookTitle, mergedHistory);
      if (linked) {
        bookAuthor = linked;
      }
    }
  }

  function selectBook(item: BookMeta) {
    if (blurTimer) {
      clearTimeout(blurTimer);
      blurTimer = null;
    }
    bookTitle = item.bookTitle;
    bookAuthor = item.bookAuthor || '';
    showDropdown = false;
    isTitleFilterActive = false;
  }

  function handleTitleKeydown(e: KeyboardEvent) {
    if (!showDropdown || filteredHistory.length === 0) {
      if (e.key === 'ArrowDown' && mergedHistory.length > 0) {
        e.preventDefault();
        isTitleFilterActive = false;
        showDropdown = true;
        highlightedIdx = 0;
      }
      return;
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      highlightedIdx = (highlightedIdx + 1) % filteredHistory.length;
      return;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      highlightedIdx = (highlightedIdx - 1 + filteredHistory.length) % filteredHistory.length;
      return;
    }
    if (e.key === 'Enter') {
      if (highlightedIdx >= 0 && highlightedIdx < filteredHistory.length) {
        e.preventDefault();
        e.stopPropagation();
        selectBook(filteredHistory[highlightedIdx]);
      }
      return;
    }
    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      showDropdown = false;
    }
  }

  function clearTitle() {
    bookTitle = '';
    bookAuthor = '';
    isTitleFilterActive = true;
    showDropdown = true;
    highlightedIdx = 0;
  }
</script>

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="book-meta-bar" on:wheel|stopPropagation>
  <div class="book-field title-field">
    <span class="field-label">📖 书名</span>
    <div class="input-wrap">
      <input
        type="text"
        class="meta-input"
        placeholder="输入或选择书名..."
        bind:value={bookTitle}
        on:focus={handleTitleFocus}
        on:click={handleTitleClick}
        on:blur={handleTitleBlur}
        on:input={handleTitleInput}
        on:keydown={handleTitleKeydown}
      />
      {#if bookTitle}
        <button
          type="button"
          class="clear-btn"
          title="清空书名切换书籍"
          on:mousedown|preventDefault
          on:click={clearTitle}
        >
          ✕
        </button>
      {/if}
    </div>

    {#if showDropdown && filteredHistory.length > 0}
      <div class="book-history-dropdown" on:wheel|stopPropagation>
        <div class="dropdown-caption">历史藏书 ({filteredHistory.length})</div>
        <ul class="history-list">
          {#each filteredHistory as item, idx (item.bookTitle)}
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
            <li
              class="history-item"
              class:active={idx === highlightedIdx}
              on:mouseenter={() => (highlightedIdx = idx)}
              on:mousedown|preventDefault={() => selectBook(item)}
            >
              <span class="history-title">《{item.bookTitle}》</span>
              {#if item.bookAuthor}
                <span class="history-author">— {item.bookAuthor}</span>
              {/if}
              {#if item.count && item.count > 0}
                <span class="history-count">{item.count} 篇</span>
              {/if}
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </div>

  <div class="meta-divider" aria-hidden="true"></div>

  <div class="book-field author-field">
    <span class="field-label">✍️ 作者</span>
    <input
      type="text"
      class="meta-input"
      placeholder="作者（选填）"
      bind:value={bookAuthor}
    />
  </div>
</div>

<style>
  .book-meta-bar {
    position: relative;
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 14px;
    margin-bottom: 8px;
    border-radius: 12px;
    background: linear-gradient(175deg, rgba(250, 247, 240, 0.96) 0%, rgba(243, 237, 224, 0.96) 100%);
    border: 1px solid rgba(154, 118, 78, 0.28);
    box-shadow:
      inset 0 1px 0 rgba(255, 255, 255, 0.85),
      0 8px 24px rgba(62, 39, 35, 0.12);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    box-sizing: border-box;
    z-index: 30;
    animation: meta-slide-in 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes meta-slide-in {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .book-field {
    position: relative;
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .title-field {
    flex: 1.45;
  }

  .author-field {
    flex: 1;
  }

  .field-label {
    font-size: 0.78rem;
    font-weight: 600;
    color: #7c4d25;
    white-space: nowrap;
    user-select: none;
    flex-shrink: 0;
  }

  .input-wrap {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    position: relative;
  }

  .meta-input {
    width: 100%;
    min-width: 0;
    border: none;
    border-bottom: 1px solid rgba(154, 118, 78, 0.25);
    background: rgba(255, 255, 255, 0.55);
    border-radius: 6px;
    padding: 4px 22px 4px 8px;
    font-size: 0.84rem;
    color: #3b2818;
    outline: none;
    transition: border-color 0.2s ease, background 0.2s ease;
    font-family: inherit;
  }

  .author-field .meta-input {
    padding-right: 8px;
  }

  .meta-input:focus {
    background: rgba(255, 255, 255, 0.9);
    border-bottom-color: #9a6b3b;
  }

  .meta-input::placeholder {
    color: rgba(124, 77, 37, 0.45);
  }

  .clear-btn {
    position: absolute;
    right: 4px;
    background: transparent;
    border: none;
    color: rgba(124, 77, 37, 0.5);
    cursor: pointer;
    font-size: 0.72rem;
    padding: 2px 4px;
    line-height: 1;
    border-radius: 50%;
    transition: color 0.15s ease, background 0.15s ease;
  }

  .clear-btn:hover {
    color: #5d3819;
    background: rgba(154, 118, 78, 0.12);
  }

  .meta-divider {
    width: 1px;
    height: 20px;
    background: rgba(154, 118, 78, 0.22);
    flex-shrink: 0;
  }

  .book-history-dropdown {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    right: -40px;
    max-height: 210px;
    overflow-y: auto;
    background: rgba(252, 249, 242, 0.98);
    border: 1px solid rgba(154, 118, 78, 0.3);
    border-radius: 10px;
    box-shadow: 0 12px 32px rgba(45, 28, 12, 0.18);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    padding: 4px;
    z-index: 60;
  }

  .dropdown-caption {
    padding: 4px 10px 3px;
    font-size: 0.68rem;
    font-weight: 600;
    color: rgba(124, 77, 37, 0.6);
    letter-spacing: 0.03em;
    user-select: none;
  }

  .history-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .history-item {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    border-radius: 7px;
    cursor: pointer;
    font-size: 0.82rem;
    color: #3b2818;
    transition: background 0.12s ease;
  }

  .history-item.active,
  .history-item:hover {
    background: rgba(154, 118, 78, 0.14);
  }

  .history-title {
    font-weight: 600;
    font-family: "Songti SC", "Noto Serif SC", Georgia, serif;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .history-author {
    font-size: 0.76rem;
    color: #7c5c3e;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
  }

  .history-count {
    font-size: 0.68rem;
    color: #8c6239;
    background: rgba(154, 118, 78, 0.12);
    padding: 1px 6px;
    border-radius: 10px;
    flex-shrink: 0;
  }
</style>
