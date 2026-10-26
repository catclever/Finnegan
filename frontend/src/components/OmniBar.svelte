<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - OmniBar Component
  Task: modularize_backend_and_frontend_architecture_074
  
  Assembles:
  - Floating Toast Feedback Banner
  - Top Action Banner (TopActionBanner.svelte)
  - Omnibar input box with command chip, locked card chip, and auto-sizing textarea
  - Attachment tray (AttachmentTray.svelte)
  - Command selection dropdown (CommandDropdown.svelte)
  - Parameter card candidate dropdown (CardCandidateDropdown.svelte)
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher, onDestroy } from 'svelte';
  import type { CosmicCard, SearchSession, CommandDef } from '../types/card';
  import AttachmentTray from './omnibar/AttachmentTray.svelte';
  import TopActionBanner from './omnibar/TopActionBanner.svelte';
  import CommandDropdown from './omnibar/CommandDropdown.svelte';
  import CardCandidateDropdown from './omnibar/CardCandidateDropdown.svelte';
  import { isChatLogFormat, extractHashTitle } from '../lib/chatlog';
  import { parseDate } from '../lib/time';
  import { getCardDisplayTitle } from '../lib/cardTitle';
  import { CaptureChatLog, CaptureChatLogWithImages } from '../../wailsjs/go/main/App.js';

  // Props
  export let availableCommands: CommandDef[] = [];
  export let searchSession: SearchSession;
  export let focusedCardId: string | null = null;
  export let cards: CosmicCard[] = [];
  export let localVaultPool: CosmicCard[] = [];
  export let capturePending: boolean = false;
  export let isOmnibarFocused: boolean = false;
  export let isScrolling: boolean = false;

  // Callbacks
  export let onCaptureThought: ((text: string, images?: string[]) => Promise<void> | void) | undefined = undefined;
  export let onCaptureChatLog: ((text: string, images?: string[]) => Promise<void> | void) | undefined = undefined;
  export let onCaptureNote: ((text: string) => Promise<void> | void) | undefined = undefined;
  export let onFreeSearch: ((text: string) => void) | undefined = undefined;
  export let onRevertSearch: (() => void) | undefined = undefined;
  export let onCommitSearch: (() => void) | undefined = undefined;
  export let onRestoreSearch: (() => void) | undefined = undefined;
  export let onExecuteCommand: ((cmd: CommandDef, card: CosmicCard) => Promise<void> | void) | undefined = undefined;
  export let onSearchLocalCards: ((query: string) => Promise<any[]>) | undefined = undefined;

  const dispatch = createEventDispatcher();

  // Internal state
  let omniText = "";
  let omnibarInputEl: HTMLTextAreaElement | null = null;
  let attachedImages: string[] = [];
  let deleteHighlightIndex: number = -1;
  let isDraggingOver: boolean = false;
  const MAX_IMAGES = 4;

  function addImageFiles(files: FileList | File[]) {
    const fileList = Array.from(files).filter(f => f.type.startsWith('image/'));
    if (fileList.length === 0) return;

    const availableSlots = MAX_IMAGES - attachedImages.length;
    if (availableSlots <= 0) {
      showToast(`最多添加 ${MAX_IMAGES} 张图片`);
      return;
    }

    const filesToLoad = fileList.slice(0, availableSlots);
    for (const file of filesToLoad) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const dataUrl = e.target?.result;
        if (typeof dataUrl === 'string') {
          if (attachedImages.length < MAX_IMAGES) {
            attachedImages = [...attachedImages, dataUrl];
            deleteHighlightIndex = -1;
          }
        }
      };
      reader.readAsDataURL(file);
    }
  }

  function removeImage(index: number) {
    if (index >= 0 && index < attachedImages.length) {
      attachedImages = attachedImages.filter((_, i) => i !== index);
      deleteHighlightIndex = -1;
    }
  }

  function handlePaste(e: ClipboardEvent) {
    if (e.clipboardData?.files && e.clipboardData.files.length > 0) {
      const imgFiles = Array.from(e.clipboardData.files).filter(f => f.type.startsWith('image/'));
      if (imgFiles.length > 0) {
        e.preventDefault();
        addImageFiles(imgFiles);
        return;
      }
    }
    if (e.clipboardData?.items) {
      const imgFiles: File[] = [];
      for (let i = 0; i < e.clipboardData.items.length; i++) {
        const item = e.clipboardData.items[i];
        if (item.type.startsWith('image/')) {
          const f = item.getAsFile();
          if (f) imgFiles.push(f);
        }
      }
      if (imgFiles.length > 0) {
        e.preventDefault();
        addImageFiles(imgFiles);
        return;
      }
    }
  }

  function handleDragOver(e: DragEvent) {
    if (e.dataTransfer && Array.from(e.dataTransfer.types).includes('Files')) {
      e.preventDefault();
      isDraggingOver = true;
    }
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    isDraggingOver = false;
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDraggingOver = false;
    if (e.dataTransfer?.files) {
      addImageFiles(e.dataTransfer.files);
    }
  }

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
  $: hashTitleInfo = extractHashTitle(omniText);
  $: explicitUserTitle = hashTitleInfo.title;
  $: isDialogueFormat = isChatLogFormat(omniText);
  $: isNoteFormat = /^\/note(\s+[\s\S]*)?$/i.test(omniText.trim());
  $: noteContent = omniText.trim().replace(/^\/note\s*/i, '');
  $: noteHashInfo = extractHashTitle(noteContent);
  $: explicitNoteTitle = noteHashInfo.title;
  $: showTopBanner = (omniText.trim().length > 0) || (attachedImages.length > 0) || commandParamMode || commandSelectMode || (searchSession.isActive && !searchSession.isConfirmed) || isNoteFormat;

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

  function selectCommand(cmd: CommandDef) {
    const trigger = (cmd.trigger || '').replace(/^\//, '').toLowerCase();
    if (trigger === 'note' || cmd.id === 'core:note') {
      commandSelectMode = false;
      commandParamMode = false;
      omniText = `/${trigger} `;
      if (omnibarInputEl) {
        omnibarInputEl.style.height = 'auto';
        omnibarInputEl.focus();
      }
      return;
    }
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
    if (deleteHighlightIndex !== -1) {
      deleteHighlightIndex = -1;
      return true;
    }
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
    attachedImages = [];
    deleteHighlightIndex = -1;
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

  function handleRestoreSearch() {
    if (onRestoreSearch) onRestoreSearch();
    omniText = "";
    if (omnibarInputEl) omnibarInputEl.style.height = 'auto';
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
    const requireKeywords = Boolean(selectedCommand?.params?.[0]?.requireKeywords || (selectedCommand?.trigger || '').replace(/^\//, '').toLowerCase() === 'bugu');
    paramSearchDebounceTimer = setTimeout(async () => {
      try {
        if (!q) {
          let rawPool = [...cards.filter(c => !c.isFadingOut), ...localVaultPool];

          if (requireKeywords) {
            rawPool = rawPool.filter(c => c.type !== 'note' && Boolean(c.keywords && Object.values(c.keywords).some(arr => Array.isArray(arr) && arr.length > 0)));
          }

          rawPool.sort((a, b) => {
            const timeA = parseDate(a.createdAt)?.getTime() ?? (typeof a.spawnTime === 'number' ? a.spawnTime : 0);
            const timeB = parseDate(b.createdAt)?.getTime() ?? (typeof b.spawnTime === 'number' ? b.spawnTime : 0);
            return timeB - timeA;
          });

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
          const filteredResults = requireKeywords
            ? results.filter((r: any) => r.type !== 'note' && Boolean(r.keywords && Object.values(r.keywords).some((arr: any) => Array.isArray(arr) && arr.length > 0)))
            : results;

          const seenIds = new Set<string>();
          const seenContent = new Set<string>();
          const uniqueResults: CosmicCard[] = [];

          for (const r of filteredResults) {
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
              createdAt: r.createdAt || r.CreatedAt,
              x: 0, y: 0, rotation: 0, z: 50,
              keywords: r.keywords || r.Keywords,
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
    const imgs = [...attachedImages];
    if ((!text && imgs.length === 0) || capturePending) return;

    omniText = "";
    attachedImages = [];
    deleteHighlightIndex = -1;
    if (omnibarInputEl) {
      omnibarInputEl.style.height = 'auto';
    }

    try {
      if (isDialogueFormat) {
        if (imgs.length > 0) {
          if (onCaptureChatLog) {
            await onCaptureChatLog(text, imgs);
          } else {
            const payload = imgs.map(img => ({ data: img }));
            await CaptureChatLogWithImages(text, payload as any);
          }
          dispatch('captureChatLog', { text, images: imgs });
        } else {
          if (onCaptureChatLog) {
            await onCaptureChatLog(text, []);
          } else {
            await CaptureChatLog(text);
          }
          dispatch('captureChatLog', { text, images: [] });
        }
      } else {
        if (onCaptureThought) {
          await onCaptureThought(text, imgs);
        }
        dispatch('captureThought', { text, images: imgs });
      }
      if (searchSession.isActive) {
        if (onCommitSearch) onCommitSearch();
        dispatch('commitSearch');
      }
    } catch (err: any) {
      console.error(err);
      const oldText = text;
      omniText = "Error: " + err;
      setTimeout(() => { if (omniText.startsWith("Error:")) omniText = oldText; }, 2000);
    }
  }

  async function captureCurrentNote(text: string) {
    const trimmed = text.trim();
    if (!trimmed || capturePending) return;

    omniText = "";
    commandSelectMode = false;
    deleteHighlightIndex = -1;
    if (omnibarInputEl) {
      omnibarInputEl.style.height = 'auto';
    }

    try {
      if (onCaptureNote) {
        await onCaptureNote(trimmed);
      }
      showToast("已保存备忘");
    } catch (err: any) {
      console.error("CaptureNote error:", err);
      showToast(`保存失败: ${err?.message || err}`);
    }
  }

  async function handleOmnibarKeydown(e: KeyboardEvent) {
    if (e.key !== 'Backspace' && deleteHighlightIndex !== -1) {
      deleteHighlightIndex = -1;
    }

    // Note Capture Mode: /note <content>
    if (isNoteFormat) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        e.preventDefault();
        if (noteContent.trim()) {
          await captureCurrentNote(noteContent);
        }
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        omniText = "";
        commandSelectMode = false;
        commandParamMode = false;
        if (omnibarInputEl) omnibarInputEl.style.height = 'auto';
        return;
      }
    }

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
      if (clean === 'note') {
        commandSelectMode = false;
        commandParamMode = false;
        return;
      }
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

    // Backspace Deletion for Attached Images when Input is Empty
    if (e.key === 'Backspace' && omniText === '' && attachedImages.length > 0 && !commandParamMode) {
      e.preventDefault();
      if (deleteHighlightIndex >= 0 && deleteHighlightIndex < attachedImages.length) {
        removeImage(deleteHighlightIndex);
      } else {
        deleteHighlightIndex = attachedImages.length - 1;
      }
      return;
    }

    // Mode 3: Normal Mode Thought Capture vs Search Solidify
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault();
      if (isNoteFormat) {
        if (noteContent) {
          await captureCurrentNote(noteContent);
        }
        return;
      }
      await captureCurrentThought();
      return;
    }

    if (e.key === 'Enter' && e.shiftKey && searchSession.isActive && !searchSession.isConfirmed) {
      e.preventDefault();
      handleSolidifySearch();
      return;
    }

    if (e.key === 'Escape') {
      if (searchSession.isActive) {
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

    if (!commandParamMode) {
      const spaceIdx = omniText.indexOf(' ');
      if (spaceIdx > 0) {
        const cmdPart = omniText.slice(0, spaceIdx).trim().replace(/^\//, '').toLowerCase();
        if (cmdPart === 'note') {
          commandSelectMode = false;
          commandParamMode = false;
          return;
        }
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
      if (isNoteFormat) {
        commandSelectMode = false;
      } else {
        commandSelectMode = true;
        highlightedCmdIndex = 0;
      }
    } else {
      commandSelectMode = false;
      if (omniText.trim().length >= 1) {
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

  function handleInputWheel(e: WheelEvent) {
    e.stopPropagation();
    if (!omnibarInputEl) return;
    const { scrollHeight, clientHeight, scrollTop } = omnibarInputEl;
    const isScrollable = scrollHeight > clientHeight + 1;
    if (!isScrollable) {
      e.preventDefault();
      return;
    }
    const isAtTop = scrollTop <= 0 && e.deltaY < 0;
    const isAtBottom = scrollTop + clientHeight >= scrollHeight - 1 && e.deltaY > 0;
    if (isAtTop || isAtBottom) {
      e.preventDefault();
    }
  }

  onDestroy(() => {
    if (toastTimer) clearTimeout(toastTimer);
    if (paramSearchDebounceTimer) clearTimeout(paramSearchDebounceTimer);
  });
</script>

<!-- Omnibar Container with Universal Two-Stage Assembly & Command Badges -->
<div class="omnibar-container" class:faded={focusedCardId !== null || isScrolling} class:hidden={focusedCardId !== null}>
  <!-- Floating Toast Feedback Banner -->
  {#if toastMessage}
    <div class="toast-feedback">
      {toastMessage}
    </div>
  {/if}

  <!-- Top Action Banner -->
  <TopActionBanner
    {showTopBanner}
    {commandParamMode}
    {lockedCard}
    {commandSelectMode}
    {searchSession}
    {isNoteFormat}
    {explicitNoteTitle}
    {isDialogueFormat}
    {explicitUserTitle}
    onExecuteCommand={executeCurrentCommand}
    onUnlockCard={unlockCard}
    onSolidifySearch={handleSolidifySearch}
    onRevertSearch={handleRevertSearch}
    onRestoreSearch={handleRestoreSearch}
  />

  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div 
    class="omnibar-box" 
    class:command-mode={commandParamMode}
    class:drag-over={isDraggingOver}
    on:click={() => omnibarInputEl?.focus()}
    on:paste={handlePaste}
    on:dragover={handleDragOver}
    on:dragleave={handleDragLeave}
    on:drop={handleDrop}
    on:wheel|stopPropagation={(e) => {
      if (e.target !== omnibarInputEl) {
        e.preventDefault();
      }
    }}
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
      on:paste={handlePaste}
      on:focus={() => isOmnibarFocused = true}
      on:blur={() => isOmnibarFocused = false}
      on:wheel={handleInputWheel}
      disabled={capturePending}
      rows="1"
    ></textarea>

    <!-- AttachmentTray below textarea inside .omnibar-box -->
    <AttachmentTray 
      images={attachedImages} 
      deleteHighlightIndex={deleteHighlightIndex} 
      on:remove={(e) => removeImage(e.detail)} 
      onRemove={removeImage}
    />
  </div>

  <!-- Dropdown 1: Command Selection Suggestions -->
  {#if commandSelectMode && matchedCommands.length > 0}
    <CommandDropdown
      {matchedCommands}
      {highlightedCmdIndex}
      onHighlight={(idx) => highlightedCmdIndex = idx}
      onSelect={selectCommand}
    />
  {/if}

  <!-- Dropdown 2: Parameter Card Candidates -->
  {#if commandParamMode && !lockedCard && paramCardCandidates.length > 0}
    <CardCandidateDropdown
      {paramCardCandidates}
      {highlightedCardIndex}
      {omniText}
      onHighlight={(idx) => highlightedCardIndex = idx}
      onSelect={lockCard}
    />
  {/if}
</div>

<style>
  .omnibar-container {
    position: absolute;
    z-index: 50;
    display: flex;
    flex-direction: column;
    align-items: center;
    transition: opacity 0.35s cubic-bezier(0.16, 1, 0.3, 1), filter 0.35s cubic-bezier(0.16, 1, 0.3, 1), transform 0.35s cubic-bezier(0.16, 1, 0.3, 1);
    width: 580px;
    max-width: 90vw;
  }

  .omnibar-container.faded {
    opacity: 0;
    filter: blur(10px);
    pointer-events: none;
    transform: scale(0.96);
  }

  .omnibar-container.hidden {
    opacity: 0 !important;
    pointer-events: none !important;
    visibility: hidden;
  }

  .omnibar-box {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
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

  .omnibar-box.drag-over {
    border-color: #3b82f6;
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.4), 0 20px 60px rgba(0, 0, 0, 0.2);
    background: rgba(255, 255, 255, 0.98);
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
