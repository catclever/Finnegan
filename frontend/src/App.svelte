<!-- FINNEGAN COSMIC STAGE - Main App Component Draft (Task: frontend_components_split_012) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { spring } from 'svelte/motion';
  import OmniBar from './components/OmniBar.svelte';
  import CosmicStage from './components/CosmicStage.svelte';
  import { 
    CaptureThought, GetTempCards, LoadLocalCards, SearchLocalCards, 
    GetRegisteredCommands, ExecuteModuleCommand 
  } from '../wailsjs/go/main/App.js';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import type { CosmicCard, SearchSession, CommandDef } from './types/card';
  import type { BuguSession } from './modules/bugu/types';
  import { 
    generateAuraColor, transformSearchResults, calculateSearchRepulsion, commitUnallocatedSearchResults 
  } from './lib/auraSearch';
  import { 
    BASE_CAPACITY, MACRO_TTL_MS, BASE_STAGE_DECAY_MS, stageCandidate, replenishStageCandidates 
  } from './lib/stageDecay';

  // --- Stage Pools & Selection State ---
  let memoryPool: CosmicCard[] = [], localVaultPool: CosmicCard[] = [], cards: CosmicCard[] = [];
  let latestManualCardId: string | null = null, focusedCardId: string | null = null, hoveredCardId: string | null = null;
  let isOmnibarFocused = false, capturePending = false, time = 0;
  let omniBarRef: OmniBar, availableCommands: CommandDef[] = [];
  let searchSession: SearchSession = { isActive: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
  let searchDebounceTimer: any = null, tideTimer: any = null;
  let activeBuguSession: BuguSession | null = null, buguFadeTimer: any = null;

  // --- Gravitational Tide (Y-Axis Expansion) ---
  let scrollExpansion = spring(0, { stiffness: 0.05, damping: 0.3 });
  $: targetCapacity = Math.round(BASE_CAPACITY * (1 + $scrollExpansion * 0.6));

  function resetTideTimer() {
    if (tideTimer) clearTimeout(tideTimer);
    tideTimer = setTimeout(() => {
      scrollExpansion.set(0);
      setTimeout(() => {
        cards.forEach(c => {
          if (c.y > 400 && !c.isVIP && !c.isFadingOut) {
            c.isFadingOut = true;
            setTimeout(() => { cards = cards.filter(item => item.id !== c.id); }, 2500);
          }
        });
      }, 1000);
    }, 45000);
  }

  function handleWheel(e: WheelEvent) {
    if (focusedCardId !== null) return;
    const delta = e.deltaY * 0.0012;
    scrollExpansion.set(delta > 0 ? Math.min(1.0, $scrollExpansion + delta) : Math.max(0, $scrollExpansion + delta));
    resetTideTimer();
    if (delta > 0) replenishStage();
  }

  function replenishStage() {
    const res = replenishStageCandidates({
      targetCapacity, cards, memoryPool, localVaultPool, searchSession, scrollExpansion: $scrollExpansion
    });
    if (res.updatedMemoryPool) memoryPool = res.updatedMemoryPool;
    if (res.remainingRankedResults !== undefined) searchSession.rankedResults = res.remainingRankedResults;
    if (res.newCardsToStage.length > 0) cards = [...cards, ...res.newCardsToStage];
  }

  // --- Aura Search Delegators ---
  async function applySearchRepulsion(rawResults: any[]) {
    if (!rawResults?.length) return;
    const now = Date.now();
    if (!searchSession.isActive) {
      searchSession.stashedCards = cards.map(c => ({ ...c }));
      searchSession.isActive = true;
    }
    const color = generateAuraColor(cards, memoryPool), searchId = 'search_' + now;
    searchSession.searchId = searchId;
    searchSession.color = color;
    const sCards = transformSearchResults(rawResults, [cards, memoryPool, localVaultPool], searchId, color, now, MACRO_TTL_MS, BASE_STAGE_DECAY_MS);
    const repulsion = calculateSearchRepulsion(cards, sCards, targetCapacity);
    searchSession.rankedResults = repulsion.rankedResults;
    cards = repulsion.stagedCards;
  }

  function revertSearchView() {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    if (!searchSession.isActive) return;
    cards = searchSession.stashedCards.map(c => ({ ...c }));
    searchSession = { isActive: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
  }

  function commitSearchView() {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    if (!searchSession.isActive) return;
    memoryPool = commitUnallocatedSearchResults(searchSession.rankedResults, cards, memoryPool, MACRO_TTL_MS);
    searchSession = { isActive: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
  }

  function handleFreeSearch(text: string) {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    const query = text.trim();
    if (query.length < 2) return;
    searchDebounceTimer = setTimeout(async () => {
      try {
        const results = await SearchLocalCards(query);
        if (results?.length) applySearchRepulsion(results);
      } catch (err) { console.error("SearchLocalCards error:", err); }
    }, 250);
  }

  async function handleCaptureThought(text: string) {
    capturePending = true;
    try {
      await CaptureThought(text);
      if (searchSession.isActive) commitSearchView();
    } finally { capturePending = false; }
  }

  function startBuguSession(sourceCardId: string) {
    if (buguFadeTimer) clearTimeout(buguFadeTimer);
    activeBuguSession = {
      id: 'session_' + Date.now(),
      sourceCardId,
      targetCardIds: [],
      status: 'scanning',
      startedAt: Date.now()
    };
    buguFadeTimer = setTimeout(() => {
      if (activeBuguSession) activeBuguSession.status = 'fading';
      buguFadeTimer = setTimeout(() => {
        activeBuguSession = null;
      }, 2500);
    }, 25000);
  }

  async function handleExecuteCommand(cmd: CommandDef, card: CosmicCard) {
    if (cmd.id.startsWith('bugu:')) {
      startBuguSession(card.id);
    }
    await ExecuteModuleCommand(cmd.id, {
      cardId: card.id, card, content: card.content || card.title, id: card.id, text: card.content || card.title
    });
  }

  // --- Tilt, Focus, & Keydown Delegators ---
  let globalTilt = spring({ rx: 0, ry: 0 }, { stiffness: 0.04, damping: 0.25 });

  function handleMouseMove(e: MouseEvent) {
    resetTideTimer();
    const mX = (e.clientX / window.innerWidth) * 2 - 1, mY = (e.clientY / window.innerHeight) * 2 - 1;
    globalTilt.set(focusedCardId === null ? { rx: mY * 15, ry: -mX * 15 } : { rx: 0, ry: 0 });
  }

  function toggleFocus(id: string) {
    resetTideTimer();
    focusedCardId = focusedCardId === id ? null : id;
    if (focusedCardId !== null) {
      globalTilt.set({ rx: 0, ry: 0 }, { hard: true });
      omniBarRef?.blurInput();
      omniBarRef?.closeCommandSelect();
      if (searchSession.isActive) {
        revertSearchView();
        omniBarRef?.clearText();
      }
    }
  }

  function handleWindowClick(e: MouseEvent) {
    resetTideTimer();
    const target = e.target as Element | null;
    if (target && target.closest('.omnibar-container')) {
      return;
    }
    // Clicked outside Omnibar: blur input, close select, and revert uncommitted search
    omniBarRef?.blurInput();
    omniBarRef?.closeCommandSelect();
    if (searchSession.isActive) {
      revertSearchView();
      omniBarRef?.clearText();
    }
    if (target && !target.closest('.card')) {
      focusedCardId = null;
    }
  }

  function handleWindowKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (focusedCardId !== null) { focusedCardId = null; return; }
      if (omniBarRef?.handleEscape()) return;
    }
    if (e.key === 'Enter') {
      if (omniBarRef?.handleGlobalEnter()) return;
    }
  }

  const toCosmic = (c: any, type: string, origin: 'manual' | 'bugu' | 'local', ttl?: number): CosmicCard => {
    const rawContent = c.content || c.Content || c.translatedText || c.TranslatedText || c.translated_text || c.originalText || '';
    let title = c.title || c.Title;
    if (!title || title === 'Thought') {
      const firstLine = rawContent.split('\n')[0].replace(/^[#\s\-*]+/, '').trim();
      title = firstLine ? (firstLine.length > 20 ? firstLine.slice(0, 20) + '...' : firstLine) : (type === 'babel' ? 'Babel Echo' : 'Thought');
    }
    return {
      id: c.id || c.ID || (type + '_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7)),
      type: c.type || c.Type || type,
      title,
      content: rawContent,
      originalText: c.originalText || c.OriginalText || c.original_text || '',
      translatedText: c.translatedText || c.TranslatedText || c.translated_text || '',
      sourceUrl: c.sourceUrl || c.SourceURL || c.source_url || '',
      createdAt: c.createdAt || c.CreatedAt || c.created_at || new Date().toISOString(),
      echoOf: c.echoOf || c.EchoOf || c.echo_of || '',
      x: 0, y: 0, rotation: 0, z: 50,
      spawnTime: Date.now(), decayDuration: BASE_STAGE_DECAY_MS, isFresh: true, isVIP: false, origin,
      macroExpiresAt: ttl ? Date.now() + ttl : undefined
    };
  };

  // --- Metabolic Loop & Lifecycle ---
  onMount(() => {
    GetRegisteredCommands().then(cmds => { if (Array.isArray(cmds)) availableCommands = cmds; }).catch(console.warn);

    Promise.all([LoadLocalCards(), GetTempCards()]).then(([localCards, tempCards]) => {
      if (tempCards?.length) memoryPool = tempCards.map(c => ({ ...toCosmic(c, 'babel', 'bugu', MACRO_TTL_MS), isFresh: false, spawnTime: 0, decayDuration: 0, z: 50 }));
      if (localCards?.length) localVaultPool = localCards.map(c => ({ ...toCosmic(c, 'thought', 'local'), isFresh: false, spawnTime: 0, decayDuration: 0, z: 50 }));
      replenishStage();
    }).catch(err => console.error("loadCards error:", err));

    EventsOn("thought:captured", (data: any) => {
      const cardId = data.DocID || ('thought_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7));
      startBuguSession(cardId);
      if (latestManualCardId) {
        const prevVIP = cards.find(c => c.id === latestManualCardId);
        if (prevVIP) { prevVIP.isVIP = false; prevVIP.decayDuration = BASE_STAGE_DECAY_MS; }
      }
      latestManualCardId = cardId;
      const baseCard = toCosmic({ id: cardId, content: data.Text }, 'thought', 'manual');
      const stagedCard = stageCandidate(baseCard, true, $scrollExpansion, cards, memoryPool);
      const newCard = { ...stagedCard, isVIP: true, decayDuration: 12 * 3600 * 1000, z: 12 };
      localVaultPool = [newCard, ...localVaultPool.filter(c => c.id !== newCard.id)];
      if (cards.filter(c => !c.isFadingOut).length >= targetCapacity) {
        const evictable = cards.filter(c => !c.isVIP && !c.isFadingOut).sort((a, b) => (a.spawnTime + a.decayDuration) - (b.spawnTime + b.decayDuration));
        if (evictable.length > 0) {
          evictable[0].isFadingOut = true;
          setTimeout(() => { cards = cards.filter(c => c.id !== evictable[0].id); }, 2500);
        }
      }
      cards = [...cards.filter(c => c.id !== newCard.id), newCard];
    });

    EventsOn("bugu:card_spawned", (c: any) => {
      const newCard = toCosmic(c, 'babel', 'bugu', MACRO_TTL_MS);
      memoryPool = [newCard, ...memoryPool.filter(m => m.id !== newCard.id)];
      if (cards.filter(card => !card.isFadingOut).length >= targetCapacity) {
        const evictable = cards.filter(card => !card.isVIP && !card.isFadingOut).sort((a, b) => (a.spawnTime + a.decayDuration) - (b.spawnTime + b.decayDuration));
        if (evictable.length > 0) {
          evictable[0].isFadingOut = true;
          setTimeout(() => { cards = cards.filter(card => card.id !== evictable[0].id); }, 2500);
        }
      }
      const staged = stageCandidate(newCard, true, $scrollExpansion, cards, memoryPool);
      cards = [...cards.filter(card => card.id !== staged.id), staged];

      if (activeBuguSession) {
        const echoOf = c.echoOf || c.echo_of || c.EchoOf;
        if (!echoOf || echoOf === activeBuguSession.sourceCardId) {
          activeBuguSession.targetCardIds = [...activeBuguSession.targetCardIds.filter(id => id !== staged.id), staged.id];
          activeBuguSession.status = 'linked';
        }
      }
    });

    const metabolicInterval = setInterval(() => {
      const now = Date.now();
      memoryPool = memoryPool.filter(c => !c.macroExpiresAt || c.macroExpiresAt > now);
      cards.forEach(c => {
        if (c.isFresh && now - c.spawnTime > 8 * 60 * 1000) c.isFresh = false;
        if (!c.isVIP && !c.isFadingOut && (now - c.spawnTime > c.decayDuration)) {
          c.isFadingOut = true;
          setTimeout(() => { cards = cards.filter(card => card.id !== c.id); }, 2500);
        }
      });
      replenishStage();
    }, 2000);

    let frame: number;
    const loop = () => { time = performance.now() * 0.001; frame = requestAnimationFrame(loop); };
    loop();

    return () => {
      cancelAnimationFrame(frame);
      clearInterval(metabolicInterval);
      if (tideTimer) clearTimeout(tideTimer);
      if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    };
  });
</script>

<svelte:window on:keydown={handleWindowKeydown} on:mousemove={handleMouseMove} on:wheel={handleWheel} on:click={handleWindowClick} />

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="container" on:click={handleWindowClick}>
  <OmniBar
    bind:this={omniBarRef}
    {availableCommands}
    {searchSession}
    {focusedCardId}
    {cards}
    {localVaultPool}
    {capturePending}
    bind:isOmnibarFocused
    onCaptureThought={handleCaptureThought}
    onFreeSearch={handleFreeSearch}
    onRevertSearch={revertSearchView}
    onCommitSearch={commitSearchView}
    onExecuteCommand={handleExecuteCommand}
    onSearchLocalCards={SearchLocalCards}
  />

  <CosmicStage
    {cards}
    {time}
    buguSession={activeBuguSession}
    globalTilt={$globalTilt}
    scrollExpansion={$scrollExpansion}
    {focusedCardId}
    {hoveredCardId}
    {isOmnibarFocused}
    onToggleFocus={toggleFocus}
    onHoverCard={(id) => { hoveredCardId = id; resetTideTimer(); }}
    onLeaveCard={() => hoveredCardId = null}
    onBgClick={handleWindowClick}
  />
</div>

<style>
  :global(body), :global(html), :global(#app) {
    background: transparent !important;
    background-color: transparent !important;
    margin: 0; padding: 0;
    width: 100vw; height: 100vh;
    overflow: hidden;
  }
  .container {
    width: 100vw; height: 100vh;
    position: relative;
    display: flex; justify-content: center; align-items: center;
    background: transparent;
  }
</style>
