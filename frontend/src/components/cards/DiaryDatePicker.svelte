<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let currentDateKey: string;
  export let availableDateMap: Map<string, number> = new Map();

  const dispatch = createEventDispatcher();

  let showPopover = false;

  function parseDateKey(key: string): Date {
    const parts = key.split('-');
    if (parts.length === 3) {
      return new Date(parseInt(parts[0]), parseInt(parts[1]) - 1, parseInt(parts[2]));
    }
    return new Date();
  }

  function getPrevDay(key: string): string {
    const d = parseDateKey(key);
    d.setDate(d.getDate() - 1);
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  function getNextDay(key: string): string {
    const d = parseDateKey(key);
    d.setDate(d.getDate() + 1);
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  function getTodayKey(): string {
    const d = new Date();
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  $: todayKey = getTodayKey();
  $: isToday = currentDateKey === todayKey;

  // Format for display (e.g. 📜 2026.11.06 · 周五)
  $: displayTitle = (() => {
    const d = parseDateKey(currentDateKey);
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
    const weekday = weekdays[d.getDay()];
    return `📜 ${year}.${month}.${day} · ${weekday}`;
  })();

  function handlePrev() {
    const prev = getPrevDay(currentDateKey);
    dispatch('selectDate', { dateKey: prev });
  }

  function handleNext() {
    if (isToday) return; // Cannot go past today
    const next = getNextDay(currentDateKey);
    if (next > todayKey) return;
    dispatch('selectDate', { dateKey: next });
  }

  function togglePopover() {
    showPopover = !showPopover;
  }

  function selectDate(key: string) {
    if (key > todayKey) return;
    dispatch('selectDate', { dateKey: key });
    showPopover = false;
  }

  // Calendar generation for popover (simple implementation showing current month)
  $: calendarDays = (() => {
    const d = parseDateKey(currentDateKey);
    const year = d.getFullYear();
    const month = d.getMonth();

    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);

    const days = [];

    // Padding before first day
    const startPadding = firstDay.getDay();
    for (let i = 0; i < startPadding; i++) {
      days.push({ empty: true });
    }

    // Days in month
    for (let i = 1; i <= lastDay.getDate(); i++) {
      const dayKey = `${year}-${String(month + 1).padStart(2, '0')}-${String(i).padStart(2, '0')}`;
      days.push({
        empty: false,
        key: dayKey,
        num: i,
        hasEntries: availableDateMap.has(dayKey),
        isCurrent: dayKey === currentDateKey,
        isFuture: dayKey > todayKey
      });
    }
    return days;
  })();
</script>

<div class="diary-date-picker-container">
  <div class="date-controls">
    <button class="nav-btn" on:click={handlePrev}>◀</button>
    <button class="date-pill" on:click={togglePopover}>{displayTitle}</button>
    <button class="nav-btn" class:disabled={isToday} on:click={handleNext} disabled={isToday}>▶</button>
  </div>

  {#if showPopover}
    <div class="vintage-calendar-popover">
      <div class="calendar-header">
        <span class="month-title">{parseDateKey(currentDateKey).getFullYear()}年 {parseDateKey(currentDateKey).getMonth() + 1}月</span>
      </div>
      <div class="calendar-grid weekdays">
        <span>日</span><span>一</span><span>二</span><span>三</span><span>四</span><span>五</span><span>六</span>
      </div>
      <div class="calendar-grid">
        {#each calendarDays as day}
          {#if day.empty}
            <div class="day empty"></div>
          {:else}
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <div
              class="day"
              class:current={day.isCurrent}
              class:future={day.isFuture}
              class:has-entries={day.hasEntries}
              on:click={() => { if(!day.isFuture) selectDate(day.key || '') }}
            >
              {day.num}
              {#if day.hasEntries}
                <div class="wax-seal-indicator"></div>
              {/if}
            </div>
          {/if}
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .diary-date-picker-container {
    position: relative;
    display: inline-block;
    z-index: 100;
  }

  .date-controls {
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(243, 228, 194, 0.8);
    border: 1px solid rgba(139, 90, 43, 0.3);
    border-radius: 20px;
    padding: 4px 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  }

  .nav-btn {
    background: none;
    border: none;
    color: #5d4037;
    cursor: pointer;
    font-size: 14px;
    padding: 4px;
  }

  .nav-btn.disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .date-pill {
    background: none;
    border: none;
    font-family: serif;
    font-weight: 600;
    color: #3e2723;
    cursor: pointer;
    font-size: 14px;
    padding: 4px 8px;
  }

  .date-pill:hover {
    background: rgba(255, 255, 255, 0.3);
    border-radius: 12px;
  }

  .vintage-calendar-popover {
    position: absolute;
    top: 100%;
    left: 0;
    margin-top: 8px;
    width: 240px;
    background: linear-gradient(165deg, #fbf3df 0%, #f3e4c2 52%, #ead5a7 100%);
    border: 2px solid #8b5a2b;
    border-radius: 8px;
    padding: 12px;
    box-shadow: 4px 4px 12px rgba(0,0,0,0.2);
    font-family: serif;
    color: #3e2723;
  }

  .calendar-header {
    text-align: center;
    font-weight: bold;
    margin-bottom: 8px;
    border-bottom: 1px solid rgba(139, 90, 43, 0.3);
    padding-bottom: 4px;
  }

  .calendar-grid {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 4px;
    text-align: center;
  }

  .weekdays {
    font-size: 12px;
    font-weight: bold;
    margin-bottom: 4px;
    color: #5d4037;
  }

  .day {
    position: relative;
    padding: 6px 0;
    font-size: 13px;
    cursor: pointer;
    border-radius: 4px;
    transition: background 0.2s;
  }

  .day:not(.empty):not(.future):hover {
    background: rgba(139, 90, 43, 0.2);
  }

  .day.current {
    font-weight: bold;
    background: rgba(139, 90, 43, 0.3);
    border: 1px solid rgba(139, 90, 43, 0.6);
  }

  .day.future {
    color: rgba(62, 39, 35, 0.3);
    cursor: not-allowed;
  }

  .wax-seal-indicator {
    position: absolute;
    bottom: 2px;
    left: 50%;
    transform: translateX(-50%);
    width: 4px;
    height: 4px;
    background-color: #8b0000; /* dark red wax seal */
    border-radius: 50%;
  }
</style>
