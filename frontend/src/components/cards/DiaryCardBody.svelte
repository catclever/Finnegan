<script lang="ts">
  import { onMount } from 'svelte';
  import { getDiaryDateKey, getSameDayDiaryCards, getAvailableDiaryDateMap, computeCelestialWeatherGlyph } from '../../lib/diaryScroll';
  import type { DiaryCard } from '../../lib/diaryScroll';
  import DiaryDatePicker from './DiaryDatePicker.svelte';
  import MarkdownContent from './MarkdownContent.svelte';

  export let card: DiaryCard;
  export let isFocused: boolean = false;

  // Assume allCards is accessible through window.go.main.App.GetCards or similar
  // For the sake of this component, we simulate having allCards. In a real scenario,
  // we would fetch from a store or window API.
  let allCards: DiaryCard[] = [];

  let availableDateMap = new Map<string, number>();
  let selectedDateKey = '';
  let sameDayCards: DiaryCard[] = [];

  // To resolve editor call
  declare const window: any;
  const OpenCardInEditor = window?.go?.main?.App?.OpenCardInEditor || (async () => {});

  $: if (card) {
    selectedDateKey = getDiaryDateKey(card);
  }

  // A mock to load all cards if needed, this can be adapted to actual data source.
  onMount(async () => {
    try {
      if (window?.go?.main?.App?.GetCards) {
        const res = await window.go.main.App.GetCards();
        if (res && res.cards) {
          allCards = res.cards;
        }
      } else {
        // Fallback for isolated UI tests, include the current card at least
        allCards = [card];
      }
      availableDateMap = getAvailableDiaryDateMap(allCards);
      updateSameDayCards();
    } catch (e) {
      console.warn("Failed to load all cards", e);
      allCards = [card];
      availableDateMap = getAvailableDiaryDateMap(allCards);
      updateSameDayCards();
    }
  });

  function updateSameDayCards() {
    if (selectedDateKey) {
      sameDayCards = getSameDayDiaryCards(selectedDateKey, allCards);
    }
  }

  function handleSelectDate(e: CustomEvent<{dateKey: string}>) {
    selectedDateKey = e.detail.dateKey;
    updateSameDayCards();
  }

  function triggerEditor(entry: DiaryCard) {
    OpenCardInEditor(entry.filePath || entry.id);
  }

  // Scattered Mode format (Date and Month like 11 / 06)
  $: scatteredDateStr = (() => {
    const d = card.createdAt ? new Date(card.createdAt) : new Date();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${month} / ${day}`;
  })();

</script>

{#if !isFocused}
  <!-- Scattered Mode: Mini Parchment Scroll -->
  <div class="diary-scattered-mode">
    <div class="diary-header">
      <span class="date-stamp">{scatteredDateStr}</span>
      <div class="weather-icon-wrapper" title="{new Date(card.createdAt || Date.now()).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})} · {card.weather?.condition || '未知'}">
        {@html computeCelestialWeatherGlyph(card.createdAt, card.weather?.condition)}
      </div>
    </div>

    <div class="diary-body">
      <div class="markdown-preview">
        {#if card.content}
          <MarkdownContent content={card.content} compact={true} />
        {/if}
      </div>
      {#if card.images && card.images.length > 0}
        <div class="thumbnail">
          <img src={card.images[0]} alt="thumbnail" />
        </div>
      {/if}
    </div>

    <div class="diary-footer">
      {#if card.location?.name}
        <span class="location-badge">📍 {card.location.name}</span>
      {/if}
      <div class="tags">
        {#if card.tags}
          {#each card.tags.slice(0, 2) as tag}
            <span class="tag">#{tag}</span>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{:else}
  <!-- Focused Mode: Combined Parchment Scroll -->
  <div class="diary-focused-mode">
    <div class="focused-topbar">
      <DiaryDatePicker
        currentDateKey={selectedDateKey}
        availableDateMap={availableDateMap}
        on:selectDate={handleSelectDate}
      />
      <span class="summary-count">当日共 {sameDayCards.length} 篇手记</span>
    </div>

    <div class="main-scroll-canvas">
      {#if sameDayCards.length === 0}
        <div class="empty-parchment-state">
          「此日尚无墨迹留存」
        </div>
      {:else}
        {#each sameDayCards as entry, index}
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <div class="diary-chapter" on:dblclick={() => triggerEditor(entry)}>
            <div class="chapter-header">
              <span class="chapter-time">{new Date(entry.createdAt || Date.now()).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
              <div class="weather-icon-wrapper">
                {@html computeCelestialWeatherGlyph(entry.createdAt, entry.weather?.condition)}
              </div>
            </div>

            {#if entry.location?.name}
              <div class="chapter-location">📍 {entry.location.name}</div>
            {/if}

            <div class="chapter-content">
              <MarkdownContent content={entry.content || ''} />
            </div>

            {#if entry.images && entry.images.length > 0}
              <div class="chapter-images">
                {#each entry.images as img}
                  <img src={img} alt="entry image" />
                {/each}
              </div>
            {/if}
          </div>

          {#if index < sameDayCards.length - 1}
            <div class="ink-fleuron-divider">─── ❦ ───</div>
          {/if}
        {/each}
      {/if}
    </div>
  </div>
{/if}

<style>
  /* Scattered Mode Styles */
  .diary-scattered-mode {
    display: flex;
    flex-direction: column;
    height: 100%;
    font-family: serif;
    color: #3e2723;
  }

  .diary-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px dashed rgba(139, 90, 43, 0.4);
    padding-bottom: 6px;
    margin-bottom: 8px;
  }

  .date-stamp {
    font-size: 16px;
    font-weight: bold;
    letter-spacing: 1px;
    color: #5d4037;
  }

  .weather-icon-wrapper {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
  }

  .diary-body {
    flex: 1;
    overflow: hidden;
    position: relative;
    font-size: 12px;
    line-height: 1.5;
  }

  .markdown-preview {
    display: -webkit-box;
    -webkit-line-clamp: 4;
    -webkit-box-orient: vertical;
    overflow: hidden;
    opacity: 0.85;
  }

  .thumbnail {
    position: absolute;
    bottom: 0;
    right: 0;
    width: 48px;
    height: 48px;
    border-radius: 4px;
    overflow: hidden;
    border: 1px solid rgba(139, 90, 43, 0.3);
    box-shadow: 2px 2px 4px rgba(0,0,0,0.1);
  }

  .thumbnail img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .diary-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 8px;
    font-size: 10px;
    color: #5d4037;
  }

  .location-badge {
    background: rgba(139, 90, 43, 0.1);
    padding: 2px 6px;
    border-radius: 8px;
  }

  .tags {
    display: flex;
    gap: 4px;
  }

  .tag {
    opacity: 0.7;
  }

  /* Focused Mode Styles */
  .diary-focused-mode {
    display: flex;
    flex-direction: column;
    height: 100%;
    font-family: serif;
    color: #3e2723;
  }

  .focused-topbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    padding-bottom: 12px;
    border-bottom: 2px solid rgba(139, 90, 43, 0.3);
  }

  .summary-count {
    font-size: 14px;
    font-style: italic;
    color: #5d4037;
  }

  .main-scroll-canvas {
    flex: 1;
    overflow-y: auto;
    padding-right: 12px;
    scrollbar-width: thin;
    scrollbar-color: rgba(139, 90, 43, 0.4) transparent;
  }

  .main-scroll-canvas::-webkit-scrollbar {
    width: 6px;
  }

  .main-scroll-canvas::-webkit-scrollbar-thumb {
    background-color: rgba(139, 90, 43, 0.4);
    border-radius: 10px;
  }

  .empty-parchment-state {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100%;
    font-size: 18px;
    color: rgba(139, 90, 43, 0.6);
    font-style: italic;
    letter-spacing: 2px;
  }

  .diary-chapter {
    padding: 16px 0;
    transition: background 0.2s;
    border-radius: 8px;
  }

  .diary-chapter:hover {
    background: rgba(139, 90, 43, 0.05);
  }

  .chapter-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
  }

  .chapter-time {
    font-size: 16px;
    font-weight: bold;
    color: #5d4037;
  }

  .chapter-location {
    font-size: 12px;
    color: #8b5a2b;
    margin-bottom: 12px;
    display: inline-block;
    background: rgba(139, 90, 43, 0.1);
    padding: 2px 8px;
    border-radius: 12px;
  }

  .chapter-content {
    font-size: 14px;
    line-height: 1.6;
    margin-bottom: 16px;
  }

  .chapter-images {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding-bottom: 8px;
  }

  .chapter-images img {
    height: 120px;
    border-radius: 6px;
    border: 1px solid rgba(139, 90, 43, 0.2);
    box-shadow: 2px 2px 6px rgba(0,0,0,0.1);
  }

  .ink-fleuron-divider {
    text-align: center;
    color: rgba(139, 90, 43, 0.5);
    font-size: 18px;
    margin: 20px 0;
    user-select: none;
  }
</style>
