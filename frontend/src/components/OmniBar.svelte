<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - OmniBar Component
  Task: frontend_components_split_012
  
  Encapsulates:
  - Floating Toast Feedback Banner
  - Top Action Banner / 条幅提示 with dynamic keyboard hint badges
  - Omnibar input box with command chip, locked card chip, and auto-sizing textarea
  - Command selection dropdown with keyboard navigation
  - Parameter card candidate dropdown with real-time search
  - Keyboard shortcut handling and window escape delegation
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher, onDestroy } from 'svelte';
  import type { CosmicCard, SearchSession, CommandDef } from '../types/card';

  // Props
  export let availableCommands: CommandDef[] = [];
  export let searchSession: SearchSession;
  export let focusedCardId: string | null = null;
  export let cards: CosmicCard[] = [];
  export let localVaultPool: CosmicCard[] = [];
  export let capturePending: boolean = false;
  export let isOmnibarFocused: boolean = false;

  // Callbacks
  export let onCaptureThought: ((text: string) => Promise<void> | void) | undefined = undefined;
  export let onFreeSearch: ((text: string) => void) | undefined = undefined;
  export let onRevertSearch: (() => void) | undefined = undefined;
  export let onCommitSearch: (() => void) | undefined = undefined;
  export let onExecuteCommand: ((cmd: CommandDef, card: CosmicCard) => Promise<void> | void) | undefined = undefined;
  export let onSearchLocalCards: ((query: string) => Promise<any[]>) | undefined = undefined;

  const dispatch = createEventDispatcher();

  // Internal state
  let omniText = "";
  let omnibarInputEl: HTMLTextAreaElement | null = null;

  // Command selection state
  let commandSelectMode: boolean = false;
  let highlightedCmdIndex: number = 0;

  // Command parameter mode state
  let commandParamMode: boolean = false;
  let selectedCommand: CommandDef | null = null;
  let lockedCard: CosmicCard | null = null;
  let paramCardCandidates: CosmicCard[] = [];
  let highlightedCardIndex: number = 0;
  let paramSearchDebounceTimer: any = null;

  // Toast feedback state
  let toastMessage: string | null = null;
  let toastTimer: any = null;

  export function showToast(msg: string, durationMs: number = 3200) {
    toastMessage = msg;
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
      toastMessage = null;
    }, durationMs);
  }

  // Reactive expressions
  $: showTopBanner = (omniText.trim().length > 0) || commandParamMode || commandSelectMode || searchSession.isActive;

  $: omnibarPlaceholder = commandParamMode
    ? (selectedCommand?.params?.[0]?.description || "吾道不孤")
    : "Capture a thought...";

  $: cmdQuery = omniText.startsWith('/') ? omniText.slice(1).trim().toLowerCase() : '';

  $: matchedCommands = commandSelectMode 
    ? availableCommands.filter(cmd => {
        const trig = (cmd.trigger || '').replace(/^\//, '').toLowerCase();
        const name = (cmd.name || '').toLowerCase();
        const desc = (cmd.description || '').toLowerCase();
        return !cmdQuery || trig.includes(cmdQuery) || name.includes(cmdQuery) || desc.includes(cmdQuery);
      })
    : [];

  function getCardDisplayTitle(c: CosmicCard | null | undefined): string {
    if (!c) return '';
    if (c.title && c.title !== 'Thought' && c.title !== 'Card' && c.title !== 'Untitled Card' && c.title !== 'Babel Echo') {
      return c.title;
    }
    const raw = c.content || c.translatedText || c.originalText || '';
    const clean = raw.split('\n')[0].replace(/^[#\s\-*]+/, '').trim();
    if (clean) {
      return clean.length > 20 ? clean.slice(0, 20) + '...' : clean;
    }
    return c.title || (c.type === 'babel' ? 'Babel Echo' : 'Thought');
  }

  function selectCommand(cmd: CommandDef) {
    selectedCommand = cmd;
    commandSelectMode = false;
    commandParamMode = true;
    omniText = "";
    lockedCard = null;
    highlightedCardIndex = 0;
    searchParamCards("");
  }

  function lockCard(candidate: CosmicCard) {
    lockedCard = candidate;
    paramCardCandidates = [];
    omniText = "";
    if (omnibarInputEl) {
      omnibarInputEl.style.height = 'auto';
      omnibarInputEl.focus();
    }
    setTimeout(() => {
      omnibarInputEl?.focus();
    }, 10);
  }

  function unlockCard() {
    lockedCard = null;
    omniText = "";
    if (omnibarInputEl) {
      omnibarInputEl.style.height = 'auto';
      omnibarInputEl.focus();
    }
    setTimeout(() => {
      omnibarInputEl?.focus();
    }, 10);
    searchParamCards("");
  }

  function exitCommandMode() {
    commandParamMode = false;
    commandSelectMode = false;
    selectedCommand = null;
    lockedCard = null;
    paramCardCandidates = [];
    omniText = "";
    if (omnibarInputEl) {
      omnibarInputEl.style.height = 'auto';
    }
  }

  export function closeCommandSelect(): void {
    if (commandSelectMode) {
      commandSelectMode = false;
      omniText = "";
    }
  }

  export function handleEscape(): boolean {
    if (commandParamMode) {
      if (lockedCard) {
        unlockCard();
      } else {
        exitCommandMode();
      }
      return true;
    }
    if (commandSelectMode) {
      commandSelectMode = false;
      omniText = "";
      return true;
    }
    if (searchSession.isActive) {
      if (onRevertSearch) onRevertSearch();
      dispatch('revertSearch');
      omniText = "";
      if (omnibarInputEl) {
        omnibarInputEl.style.height = 'auto';
      }
      return true;
    }
    return false;
  }

  export function handleGlobalEnter(): boolean {
    if (commandParamMode && lockedCard) {
      executeCurrentCommand();
      return true;
    }
    return false;
  }

  export function blurInput(): void {
    if (omnibarInputEl) {
      omnibarInputEl.blur();
    }
    isOmnibarFocused = false;
  }

  export function clearText(): void {
    omniText = "";
    if (omnibarInputEl) {
      omnibarInputEl.style.height = 'auto';
    }
  }

  function handleRevertSearch() {
    if (onRevertSearch) onRevertSearch();
    dispatch('revertSearch');
    omniText = "";
    if (omnibarInputEl) {
      omnibarInputEl.style.height = 'auto';
    }
  }

  function handleSolidifySearch() {
    if (onCommitSearch) onCommitSearch();
    dispatch('commitSearch');
    omniText = "";
    if (omnibarInputEl) {
      omnibarInputEl.style.height = 'auto';
    }
  }

  async function executeCurrentCommand() {
    if (!selectedCommand || !lockedCard) return;
    const cmd = selectedCommand;
    const card = lockedCard;
    
    showToast(`${cmd.name || cmd.trigger} 已启动...`);
    
    try {
      if (onExecuteCommand) {
        await onExecuteCommand(cmd, card);
      }
      dispatch('executeCommand', { cmd, card });
    } catch (err: any) {
      console.error("ExecuteModuleCommand error:", err);
      showToast(`执行失败: ${err?.message || err}`);
    } finally {
      commandParamMode = false;
      selectedCommand = null;
      lockedCard = null;
      omniText = "";
      if (omnibarInputEl) {
        omnibarInputEl.style.height = 'auto';
      }
    }
  }

  function searchParamCards(text: string) {
    if (paramSearchDebounceTimer) clearTimeout(paramSearchDebounceTimer);
    const q = text.trim();
    paramSearchDebounceTimer = setTimeout(async () => {
      try {
        if (!q) {
          // If empty query, show top cards from stage & vault as immediate options (deduplicated)
          const rawPool = [...cards.filter(c => !c.isFadingOut), ...localVaultPool];
          const seenIds = new Set<string>();
          const seenContent = new Set<string>();
          const uniqueCandidates: CosmicCard[] = [];

          for (const c of rawPool) {
            const normContent = (c.content || '').trim();
            if (c.id && seenIds.has(c.id)) continue;
            if (normContent && seenContent.has(normContent)) continue;
            if (c.id) seenIds.add(c.id);
            if (normContent) seenContent.add(normContent);
            uniqueCandidates.push(c);
            if (uniqueCandidates.length >= 6) break;
          }

          paramCardCandidates = uniqueCandidates;
          highlightedCardIndex = 0;
          return;
        }

        const results = onSearchLocalCards ? await onSearchLocalCards(q) : [];
        if (results && results.length > 0) {
          const seenIds = new Set<string>();
          const seenContent = new Set<string>();
          const uniqueResults: CosmicCard[] = [];

          for (const r of results) {
            const normContent = (r.content || '').trim();
            if (r.id && seenIds.has(r.id)) continue;
            if (normContent && seenContent.has(normContent)) continue;
            if (r.id) seenIds.add(r.id);
            if (normContent) seenContent.add(normContent);

            uniqueResults.push({
              id: r.id,
              type: r.type || 'thought',
              title: r.title || 'Card',
              content: r.content || '',
              originalText: r.originalText || '',
              translatedText: r.translatedText || '',
              sourceUrl: r.sourceUrl || '',
              x: 0, y: 0, rotation: 0, z: 50,
              spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false,
              origin: r.type === 'babel' ? 'bugu' : 'local'
            });
            if (uniqueResults.length >= 8) break;
          }
          paramCardCandidates = uniqueResults;
        } else {
          paramCardCandidates = [];
        }
        highlightedCardIndex = 0;
      } catch (err) {
        console.error("searchParamCards error:", err);
        paramCardCandidates = [];
      }
    }, 250);
  }

  async function captureCurrentThought() {
    const text = omniText.trim();
    if (!text || capturePending) return;
    try {
      if (onCaptureThought) {
        await onCaptureThought(text);
      }
      dispatch('captureThought', text);
      omniText = "";
      if (searchSession.isActive) {
        if (onCommitSearch) onCommitSearch();
        dispatch('commitSearch');
      }
      if (omnibarInputEl) {
        omnibarInputEl.style.height = 'auto';
      }
    } catch (err: any) {
      console.error(err);
      const oldText = text;
      omniText = "Error: " + err;
      setTimeout(() => { if (omniText.startsWith("Error:")) omniText = oldText; }, 2000);
    }
  }

  async function handleOmnibarKeydown(e: KeyboardEvent) {
    // Mode 1: Command Selection Dropdown Active
    if (commandSelectMode) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (matchedCommands.length > 0) {
          highlightedCmdIndex = (highlightedCmdIndex + 1) % matchedCommands.length;
        }
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (matchedCommands.length > 0) {
          highlightedCmdIndex = (highlightedCmdIndex - 1 + matchedCommands.length) % matchedCommands.length;
        }
        return;
      }
      if (e.key === ' ' || e.key === 'Spacebar') {
        const clean = cmdQuery;
        const matched = matchedCommands.find(cmd => (cmd.trigger || '').replace(/^\//, '').toLowerCase() === clean);
        if (matched) {
          e.preventDefault();
          selectCommand(matched);
          return;
        } else if (matchedCommands.length > 0 && highlightedCmdIndex >= 0 && highlightedCmdIndex < matchedCommands.length) {
          const highlighted = matchedCommands[highlightedCmdIndex];
          if ((highlighted.trigger || '').replace(/^\//, '').toLowerCase() === clean) {
            e.preventDefault();
            selectCommand(highlighted);
            return;
          }
        }
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault();
        if (matchedCommands.length > 0 && highlightedCmdIndex >= 0 && highlightedCmdIndex < matchedCommands.length) {
          selectCommand(matchedCommands[highlightedCmdIndex]);
        }
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        commandSelectMode = false;
        omniText = "";
        return;
      }
    }

    // Space trigger: Typing a valid command name (e.g. "/bugu" or "bugu") + Space enters param mode immediately
    if ((e.key === ' ' || e.key === 'Spacebar') && !commandParamMode) {
      const clean = omniText.trim().replace(/^\//, '').toLowerCase();
      const exactCmd = availableCommands.find(cmd => (cmd.trigger || '').replace(/^\//, '').toLowerCase() === clean);
      if (exactCmd) {
        e.preventDefault();
        selectCommand(exactCmd);
        return;
      }
    }

    // Mode 2: Command Parameter Mode Active (Two-Stage Enter)
    if (commandParamMode) {
      if (lockedCard) {
        // Card is locked: Enter 2 submits command; Esc/Backspace unlocks
        if (e.key === 'Enter') {
          e.preventDefault();
          await executeCurrentCommand();
          return;
        }
        if (e.key === 'Escape' || e.key === 'Backspace') {
          e.preventDefault();
          unlockCard();
          return;
        }
      } else {
        // Param candidate search dropdown active
        if (e.key === 'ArrowDown') {
          e.preventDefault();
          if (paramCardCandidates.length > 0) {
            highlightedCardIndex = (highlightedCardIndex + 1) % paramCardCandidates.length;
          }
          return;
        }
        if (e.key === 'ArrowUp') {
          e.preventDefault();
          if (paramCardCandidates.length > 0) {
            highlightedCardIndex = (highlightedCardIndex - 1 + paramCardCandidates.length) % paramCardCandidates.length;
          }
          return;
        }
        if (e.key === 'Enter') {
          e.preventDefault();
          if (paramCardCandidates.length > 0 && highlightedCardIndex >= 0 && highlightedCardIndex < paramCardCandidates.length) {
            // Enter 1: Lock card candidate
            lockCard(paramCardCandidates[highlightedCardIndex]);
          }
          return;
        }
        if (e.key === 'Escape') {
          e.preventDefault();
          exitCommandMode();
          return;
        }
        if (e.key === 'Backspace' && omniText === '') {
          e.preventDefault();
          exitCommandMode();
          return;
        }
      }
    }

    // Mode 3: Normal Mode Thought Capture vs Search Solidify
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      // Capturing a thought ONLY triggers on (e.metaKey || e.ctrlKey) && e.key === 'Enter'
      e.preventDefault();
      await captureCurrentThought();
      return;
    }

    if (e.key === 'Enter') {
      if (searchSession.isActive) {
        // Enter (固化) commits current search view
        e.preventDefault();
        if (onCommitSearch) onCommitSearch();
        dispatch('commitSearch');
        omniText = "";
        if (omnibarInputEl) {
          omnibarInputEl.style.height = 'auto';
        }
        return;
      }
      // Plain text multiline editing: allow standard newline insertion
    }

    if (e.key === 'Escape') {
      if (searchSession.isActive) {
        // Esc (一闪念) reverts stage
        e.preventDefault();
        if (onRevertSearch) onRevertSearch();
        dispatch('revertSearch');
        omniText = "";
        if (omnibarInputEl) {
          omnibarInputEl.style.height = 'auto';
        }
        return;
      }
    }
  }

  function handleInput(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    target.style.height = 'auto';
    target.style.height = target.scrollHeight + 'px';

    // Auto-detect command trigger followed by a space (e.g. "/bugu " or "bugu ")
    if (!commandParamMode) {
      const spaceIdx = omniText.indexOf(' ');
      if (spaceIdx > 0) {
        const cmdPart = omniText.slice(0, spaceIdx).trim().replace(/^\//, '').toLowerCase();
        const found = availableCommands.find(c => (c.trigger || '').replace(/^\//, '').toLowerCase() === cmdPart);
        if (found) {
          const restText = omniText.slice(spaceIdx + 1);
          selectCommand(found);
          if (restText) {
            omniText = restText;
            searchParamCards(restText);
          }
          return;
        }
      }
    }

    if (commandParamMode) {
      if (!lockedCard) {
        searchParamCards(omniText);
      }
    } else if (omniText.startsWith('/')) {
      commandSelectMode = true;
      highlightedCmdIndex = 0;
    } else {
      commandSelectMode = false;
      if (omniText.trim().length >= 2) {
        if (onFreeSearch) onFreeSearch(omniText);
        dispatch('freeSearch', omniText);
      } else {
        if (searchSession.isActive) {
          if (onRevertSearch) onRevertSearch();
          dispatch('revertSearch');
        }
      }
    }
  }

  onDestroy(() => {
    if (toastTimer) clearTimeout(toastTimer);
    if (paramSearchDebounceTimer) clearTimeout(paramSearchDebounceTimer);
  });
</script>

<!-- Omnibar Container with Universal Two-Stage Assembly & Command Badges -->
<div class="omnibar-container" class:faded={focusedCardId !== null}>
  <!-- Floating Toast Feedback Banner -->
  {#if toastMessage}
    <div class="toast-feedback">
      {toastMessage}
    </div>
  {/if}

  <!-- Top Action Banner / 条幅提示 -->
  {#if showTopBanner}
    <div class="omnibar-top-banner">
      {#if commandParamMode}
        {#if lockedCard}
          <span class="banner-badge ready-badge">🚀 准备就绪</span>
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <button type="button" class="banner-clickable-action" on:click={executeCurrentCommand} title="点击或按 Enter 执行命令">
            <span class="banner-key">↵</span>
            <span class="banner-action">执行命令</span>
          </button>
          <span class="banner-sep">·</span>
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <button type="button" class="banner-clickable-action" on:click={unlockCard} title="点击或按 Esc 重新选择">
            <span class="banner-key">Esc</span>
            <span class="banner-action">重新选择</span>
          </button>
        {:else}
          <span class="banner-badge param-badge">📄 观点参数</span>
          <span class="banner-key">↑ ↓</span>
          <span class="banner-action">移动</span>
          <span class="banner-sep">·</span>
          <span class="banner-key">↵</span>
          <span class="banner-action">锁定卡片</span>
          <span class="banner-sep">·</span>
          <span class="banner-key">Esc</span>
          <span class="banner-action">退出</span>
        {/if}
      {:else if commandSelectMode}
        <span class="banner-badge cmd-badge">🔭 模块命令</span>
        <span class="banner-key">↑ ↓</span>
        <span class="banner-action">选择</span>
        <span class="banner-sep">·</span>
        <span class="banner-key">↵</span>
        <span class="banner-action">确认</span>
        <span class="banner-sep">·</span>
        <span class="banner-key">Esc</span>
        <span class="banner-action">取消</span>
      {:else if searchSession.isActive}
        <span class="banner-badge search-badge" style="--glow-color: {searchSession.color || 'hsl(180, 95%, 62%)'}">
          <span class="banner-dot"></span> 探针共鸣中
        </span>
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <button type="button" class="banner-clickable-action" on:click={handleSolidifySearch} title="按 Enter 固化回声">
          <span class="banner-key">↵</span>
          <span class="banner-action">固化回声</span>
        </button>
        <span class="banner-sep">·</span>
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <button type="button" class="banner-clickable-action" on:click={handleRevertSearch} title="按 Esc 一闪念还原">
          <span class="banner-key">Esc</span>
          <span class="banner-action">一闪念还原</span>
        </button>
      {:else}
        <span class="banner-key">⌘ ↵</span>
        <span class="banner-action">发送思绪</span>
        <span class="banner-sep">·</span>
        <span class="banner-key">↵</span>
        <span class="banner-action">换行</span>
        <span class="banner-sep">·</span>
        <span class="banner-key">/</span>
        <span class="banner-action">命令</span>
      {/if}
    </div>
  {/if}

  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div 
    class="omnibar-box" 
    class:command-mode={commandParamMode}
    on:click={() => omnibarInputEl?.focus()}
  >
    <!-- Command Chip Badge: [🔭 /cmd] -->
    {#if commandParamMode && selectedCommand}
      <div class="omnibar-chip command-chip">
        <span class="chip-icon">🔭</span>
        <span class="chip-label">/{selectedCommand.trigger.replace(/^\//, '')}</span>
        <button class="chip-close" on:click={exitCommandMode} title="退出命令模式">✕</button>
      </div>
    {/if}

    <!-- Locked Card Parameter Chip Badge: [📄 <Card Title> ✕] -->
    {#if commandParamMode && lockedCard}
      <div class="omnibar-chip card-chip">
        <span class="chip-icon">📄</span>
        <span class="chip-label">{getCardDisplayTitle(lockedCard)}</span>
        <button class="chip-close" on:click={unlockCard} title="按 Esc 重新选择">✕</button>
      </div>
    {/if}

    <!-- Omnibar Input: Always mounted to preserve DOM focus and keydown events -->
    <textarea 
      bind:this={omnibarInputEl}
      class="omnibar-input" 
      class:locked-mode={commandParamMode && lockedCard !== null}
      placeholder={commandParamMode && lockedCard ? "" : omnibarPlaceholder} 
      bind:value={omniText}
      on:keydown={handleOmnibarKeydown}
      on:input={handleInput}
      on:focus={() => isOmnibarFocused = true}
      on:blur={() => isOmnibarFocused = false}
      disabled={capturePending}
      rows="1"
    ></textarea>
  </div>

  <!-- Dropdown 1: Command Selection Suggestions -->
  {#if commandSelectMode && matchedCommands.length > 0}
    <div class="dropdown-menu command-dropdown">
      {#each matchedCommands as cmd, index}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div 
          class="dropdown-item" 
          class:active={index === highlightedCmdIndex}
          on:mouseenter={() => highlightedCmdIndex = index}
          on:mousedown|preventDefault={() => selectCommand(cmd)}
        >
          <div class="cmd-item-header">
            <span class="cmd-icon">🔭</span>
            <span class="cmd-trigger">/{cmd.trigger.replace(/^\//, '')}</span>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Dropdown 2: Parameter Card Candidates -->
  {#if commandParamMode && !lockedCard && paramCardCandidates.length > 0}
    <div class="dropdown-menu card-dropdown">
      {#each paramCardCandidates as candidate, index}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div 
          class="dropdown-item card-item" 
          class:active={index === highlightedCardIndex}
          on:mouseenter={() => highlightedCardIndex = index}
          on:mousedown|preventDefault={() => lockCard(candidate)}
        >
          <div class="card-item-title">
            <span class="candidate-icon">📄</span>
            {getCardDisplayTitle(candidate)}
          </div>
          <div class="card-item-snippet">
            {candidate.content || candidate.translatedText || candidate.originalText || ''}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .omnibar-container {
    position: absolute;
    z-index: 50; /* Above normal background cards */
    display: flex;
    flex-direction: column;
    align-items: center;
    transition: opacity 0.5s ease, filter 0.5s ease;
    width: 580px;
    max-width: 90vw;
  }

  .omnibar-container.faded {
    opacity: 0;
    filter: blur(10px);
    pointer-events: none;
  }

  .omnibar-box {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 22px;
    border-radius: 30px;
    border: 2px solid rgba(255, 255, 255, 0.9);
    background: rgba(255, 255, 255, 0.88);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05);
    transition: background 0.3s ease, border 0.3s ease, box-shadow 0.3s ease, transform 0.2s ease;
    box-sizing: border-box;
    min-height: 60px;
  }

  .omnibar-box:focus-within {
    background: rgba(255, 255, 255, 0.96);
    border: 2px solid rgba(255, 255, 255, 1);
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2), 0 0 0 1px rgba(0, 0, 0, 0.05);
    transform: scale(1.015);
  }

  .omnibar-input {
    flex: 1;
    border: none;
    background: transparent;
    font-size: 1.15rem;
    color: #333;
    outline: none;
    text-align: left;
    resize: none;
    min-height: 28px;
    max-height: 200px;
    overflow-y: auto;
    line-height: 1.5;
    box-sizing: border-box;
    font-family: inherit;
    padding: 4px 6px;
  }

  .omnibar-input::placeholder {
    color: rgba(51, 51, 51, 0.55);
  }

  /* Hide scrollbar for omnibar input */
  .omnibar-input::-webkit-scrollbar {
    display: none;
  }
  .omnibar-input {
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  /* Omnibar Chips */
  .omnibar-chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 5px 12px;
    border-radius: 16px;
    font-size: 0.9rem;
    font-weight: 550;
    user-select: none;
    white-space: nowrap;
    animation: chip-appear 0.2s cubic-bezier(0.2, 0.9, 0.3, 1);
  }

  @keyframes chip-appear {
    from { transform: scale(0.85); opacity: 0; }
    to { transform: scale(1); opacity: 1; }
  }

  .command-chip {
    background: rgba(59, 130, 246, 0.15);
    border: 1px solid rgba(59, 130, 246, 0.35);
    color: #1d4ed8;
  }

  .card-chip {
    background: rgba(16, 185, 129, 0.15);
    border: 1px solid rgba(16, 185, 129, 0.35);
    color: #065f46;
    max-width: 250px;
  }

  .chip-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .chip-close {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 0.8rem;
    color: inherit;
    opacity: 0.6;
    padding: 0 2px;
    margin-left: 2px;
    border-radius: 50%;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
    transition: opacity 0.15s;
  }

  .chip-close:hover {
    opacity: 1;
  }

  /* Dropdown Menus */
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

  .dropdown-item.active,
  .dropdown-item:hover {
    background: rgba(59, 130, 246, 0.1);
  }

  .cmd-item-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .cmd-trigger {
    font-weight: 700;
    color: #2563eb;
    font-size: 0.95rem;
  }

  .card-item-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-weight: 600;
    color: #111827;
    font-size: 0.92rem;
  }

  .card-item-snippet {
    font-size: 0.8rem;
    color: rgba(75, 85, 99, 0.75);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.35;
  }

  .toast-feedback {
    margin-bottom: 10px;
    padding: 8px 18px;
    background: rgba(17, 24, 39, 0.88);
    color: #fff;
    font-size: 0.88rem;
    font-weight: 500;
    border-radius: 20px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    pointer-events: none;
    animation: toast-in 0.25s cubic-bezier(0.2, 0.9, 0.3, 1);
  }

  @keyframes toast-in {
    from { opacity: 0; transform: translateY(6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  /* Top Action Banner / 条幅提示 */
  .omnibar-top-banner {
    align-self: flex-start;
    margin-left: 22px;
    margin-bottom: 6px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 0;
    background: transparent;
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
    border: none;
    border-radius: 0;
    box-shadow: none;
    font-size: 0.72rem;
    color: rgba(50, 50, 50, 0.85);
    user-select: none;
    animation: banner-pop 0.18s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes banner-pop {
    from {
      opacity: 0;
      transform: translateY(3px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .banner-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 1px 6px;
    border-radius: 4px;
    font-weight: 600;
    font-size: 0.68rem;
    letter-spacing: 0.02em;
  }

  .banner-badge.cmd-badge {
    background: rgba(59, 130, 246, 0.12);
    color: #2563eb;
  }

  .banner-badge.param-badge {
    background: rgba(16, 185, 129, 0.12);
    color: #059669;
  }

  .banner-badge.ready-badge {
    background: rgba(245, 158, 11, 0.12);
    color: #d97706;
  }

  .banner-badge.search-badge {
    background: rgba(147, 51, 234, 0.1);
    color: #7e22ce;
  }

  .banner-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--glow-color, #7e22ce);
    box-shadow: 0 0 5px var(--glow-color, #7e22ce);
    display: inline-block;
  }

  .banner-key {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 1px 5px;
    background: rgba(255, 255, 255, 0.65);
    border: 1px solid rgba(255, 255, 255, 0.8);
    border-radius: 4px;
    font-size: 0.66rem;
    font-weight: 600;
    color: #374151;
    line-height: 1.25;
    font-family: inherit;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  }

  .banner-action {
    color: rgba(55, 65, 81, 0.85);
    font-weight: 500;
    font-size: 0.7rem;
    text-shadow: 0 1px 2px rgba(255, 255, 255, 0.6);
  }

  .banner-sep {
    color: rgba(0, 0, 0, 0.2);
    font-weight: 600;
    font-size: 0.65rem;
    margin: 0 1px;
  }

  .banner-clickable-action {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: transparent;
    border: none;
    padding: 0;
    margin: 0;
    cursor: pointer;
    font-family: inherit;
    border-radius: 4px;
    transition: transform 0.15s ease, opacity 0.15s ease;
  }

  .banner-clickable-action:hover {
    opacity: 0.8;
    transform: translateY(-1px);
  }

  .banner-clickable-action:active {
    transform: translateY(0);
  }

  .omnibar-input.locked-mode {
    width: 2px !important;
    min-width: 2px !important;
    max-width: 2px !important;
    padding: 0 !important;
    margin: 0 !important;
    opacity: 0;
    caret-color: transparent;
    resize: none;
    overflow: hidden;
  }
</style>
