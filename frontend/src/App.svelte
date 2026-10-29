<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - Main App Component
  ======================================================================================
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { spring } from 'svelte/motion';
  import OmniBar from './components/OmniBar.svelte';
  import CosmicStage from './components/CosmicStage.svelte';
  import { CaptureThought, CaptureThoughtWithImages, CaptureChatLog, CaptureChatLogWithImages, CaptureNote, GetTempCards, LoadLocalCards, SearchLocalCards, GetRegisteredCommands, ExecuteModuleCommand, EvictTempCard, ReloadCard } from '../wailsjs/go/main/App.js';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import type { CosmicCard, SearchSession, CommandDef } from './types/card';
  import type { BuguSession, BuguBeacon } from './modules/bugu/types';
  import { generateAuraColor, transformSearchResults, calculateSearchRepulsion, commitUnallocatedSearchResults } from './lib/auraSearch';
  import { BASE_CAPACITY, MACRO_TTL_MS, BASE_STAGE_DECAY_MS, stageCandidate, replenishStageCandidates, isProtectedCard, toCosmicCard, ensureCardStagedInViewport } from './lib/stageDecay';
  import { computeNextScrollTarget } from './lib/stageCoords';
  import { isChatLogFormat } from './lib/chatlog';
  import { createBuguSessionController } from './modules/bugu/buguSession';
  import { createStageActivityController } from './lib/stageActivity';
  import { createOptimisticChatLogCard, reconcileChatLogCard, rollbackOptimisticCard, createNoteCard, mergeCapturedThought } from './lib/cardCapture';
  import { findUnstagedMatchingCards, cardHasTag, cardHasKeyword, calculateIntrinsicTargetZ, getDistinctTagColor, normalizeTag } from './lib/tagNetwork';
  import { applyGravitationalFallback } from './lib/gravityFallback';
  import { refreshStageTime } from './lib/time';
  import type { AuraTrace } from './types/card';

  // --- Stage Pools & Selection State ---
  let memoryPool: CosmicCard[] = [], localVaultPool: CosmicCard[] = [], cards: CosmicCard[] = [];
  let latestManualCardId: string | null = null, focusedCardId: string | null = null, hoveredCardId: string | null = null;
  let activeConstellationTags: Map<string, string> = new Map();
  $: activeConstellationTag = activeConstellationTags.size > 0 ? Array.from(activeConstellationTags.keys())[0] : null;
  let isOmnibarFocused = false, capturePending = false;
  let isWindowFocused = true, isStageSleeping = false;
  let focusNavDirection: 'prev' | 'next' | null = null;
  let omniBarRef: OmniBar, availableCommands: CommandDef[] = [];
  let searchSession: SearchSession = { isActive: false, isConfirmed: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
  let searchDebounceTimer: any = null;
  let activeBuguSession: BuguSession | null = null;
  let activeBuguBeacon: BuguBeacon | null = null;

  // --- Gravitational Tide (Multi-Screen Y-Axis Expansion) ---
  let scrollTarget = 0;
  let scrollExpansion = spring(0, { stiffness: 0.16, damping: 0.72 });
  let isScrolling = false;
  let scrollIdleTimer: any = null;
  let lastScrollDirection: 'down' | 'up' = 'down';
  $: targetCapacity = Math.round(BASE_CAPACITY * (1 + $scrollExpansion * 0.6));

  // --- Encapsulated Controllers ---
  const buguController = createBuguSessionController({
    getCards: () => cards,
    getMemoryPool: () => memoryPool,
    getScrollExpansion: () => $scrollExpansion,
    onBeaconChange: (beacon) => { activeBuguBeacon = beacon; },
    onSessionChange: (session) => { activeBuguSession = session; },
    onUpdateStage: (newCards, newMem) => {
      cards = newCards;
      if (newMem) memoryPool = newMem;
    },
    toCosmicCard,
    stageCandidate,
  });

  const stageActivity = createStageActivityController({
    onFocusChange: (focused) => { isWindowFocused = focused; },
    onSleepChange: (sleeping) => { isStageSleeping = sleeping; },
    canSleep: () => focusedCardId === null && !searchSession.isActive && !isOmnibarFocused,
  });

  function handleWheel(e: WheelEvent) {
    if (focusedCardId !== null || isOmnibarFocused) return;
    stageActivity.resetActivityTimer();
    if (Math.abs(e.deltaY) > 0.5) {
      lastScrollDirection = e.deltaY > 0 ? 'down' : 'up';
    }
    isScrolling = true;
    if (scrollIdleTimer) clearTimeout(scrollIdleTimer);
    scrollIdleTimer = setTimeout(() => {
      isScrolling = false;
    }, 450);

    const currentExp = $scrollExpansion;
    let baseTarget = scrollTarget;
    if (e.deltaY < 0 && baseTarget > currentExp) {
      baseTarget = currentExp;
    } else if (e.deltaY > 0 && baseTarget < currentExp) {
      baseTarget = currentExp;
    }

    const totalAvailable = cards.length + memoryPool.length + localVaultPool.length;
    scrollTarget = computeNextScrollTarget(baseTarget, e.deltaY, e.deltaMode, cards, totalAvailable);
    scrollExpansion.set(scrollTarget);
    if (e.deltaY > 0) replenishStage();
  }

  function replenishStage() {
    const res = replenishStageCandidates({ targetCapacity, cards, memoryPool, localVaultPool, searchSession, scrollExpansion: $scrollExpansion });
    if (res.updatedMemoryPool) memoryPool = res.updatedMemoryPool;
    if (res.remainingRankedResults !== undefined) searchSession.rankedResults = res.remainingRankedResults;
    if (res.updatedCards) {
      cards = res.updatedCards;
    } else if (res.newCardsToStage.length > 0) {
      cards = [...cards, ...res.newCardsToStage];
    }
  }

  // --- Aura Search Delegators ---
  async function applySearchRepulsion(rawResults: any[]) {
    if (!rawResults?.length) return;
    const now = Date.now();
    if (!searchSession.isActive) {
      searchSession.stashedCards = cards.map(c => ({ ...c }));
      searchSession.isActive = true;
      searchSession.isConfirmed = false;
      searchSession.searchId = 'search_' + now;
      searchSession.color = generateAuraColor(cards, memoryPool);
    }
    const searchId = searchSession.searchId;
    const color = searchSession.color;
    const baseCards = searchSession.stashedCards.map(c => ({ ...c }));
    const sCards = transformSearchResults(rawResults, [baseCards, memoryPool, localVaultPool], searchId, color, now, MACRO_TTL_MS, BASE_STAGE_DECAY_MS);
    const repulsion = calculateSearchRepulsion(baseCards, sCards, targetCapacity);
    searchSession.rankedResults = repulsion.rankedResults;
    cards = repulsion.stagedCards;
  }

  function revertSearchView() {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    if (!searchSession.isActive) return;
    cards = searchSession.stashedCards.map(c => ({ ...c }));
    searchSession = { isActive: false, isConfirmed: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
  }

  function confirmSearchView() {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    if (!searchSession.isActive) return;
    memoryPool = commitUnallocatedSearchResults(searchSession.rankedResults, cards, memoryPool, MACRO_TTL_MS);
    searchSession.isConfirmed = true;
  }

  function finishCaptureSearch() {
    if (!searchSession.isActive) return;
    confirmSearchView();
    searchSession = { isActive: false, isConfirmed: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
  }

  function restoreConfirmedSearch() {
    if (!searchSession.isActive || !searchSession.isConfirmed) return;
    revertSearchView();
    omniBarRef?.clearText();
  }

  function handleFreeSearch(text: string) {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    const query = text.trim();
    if (query.length < 1) {
      if (searchSession.isActive && !searchSession.isConfirmed) {
        revertSearchView();
      }
      return;
    }
    if (searchSession.isActive && searchSession.isConfirmed) {
      cards = searchSession.stashedCards.map(c => ({ ...c }));
      searchSession = { isActive: false, isConfirmed: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
    }
    searchDebounceTimer = setTimeout(async () => {
      try {
        if (!searchSession.isActive) {
          const now = Date.now();
          searchSession.stashedCards = cards.map(c => ({ ...c }));
          searchSession.isActive = true;
          searchSession.isConfirmed = false;
          searchSession.searchId = 'search_' + now;
          searchSession.color = generateAuraColor(cards, memoryPool);
        }
        const results = await SearchLocalCards(query);
        if (results?.length) {
          applySearchRepulsion(results);
        } else if (searchSession.isActive) {
          searchSession.rankedResults = [];
        }
      } catch (err) { console.error("SearchLocalCards error:", err); }
    }, 250);
  }

  async function handleCaptureThought(text: string, images?: string[]) {
    capturePending = true;
    try {
      if (isChatLogFormat(text)) {
        await handleCaptureChatLog(text, images);
      } else if (images && images.length > 0) {
        const payload = images.map(img => (typeof img === 'string' ? { data: img } : img));
        await CaptureThoughtWithImages(text, payload as any);
      } else {
        await CaptureThought(text);
      }
      if (searchSession.isActive) finishCaptureSearch();
    } finally { capturePending = false; }
  }

  async function handleCaptureChatLog(text: string, images?: string[]) {
    capturePending = true;
    const opt = createOptimisticChatLogCard({
      text,
      images,
      cards,
      memoryPool,
      localVaultPool,
      scrollExpansion: $scrollExpansion,
      sourceCardId: activeBuguBeacon?.sourceCardId,
      latestManualCardId,
    });
    const tempId = opt.tempId;
    cards = opt.cards;
    memoryPool = opt.memoryPool;
    localVaultPool = opt.localVaultPool;
    latestManualCardId = opt.latestManualCardId;

    try {
      let res: any;
      if (images && images.length > 0) {
        const payload = images.map(img => (typeof img === 'string' ? { data: img } : img));
        res = await CaptureChatLogWithImages(text, payload as any);
      } else {
        res = await CaptureChatLog(text);
      }

      const reconciled = reconcileChatLogCard({
        tempId,
        res,
        originalImages: images,
        cards,
        localVaultPool,
        latestManualCardId,
      });
      cards = reconciled.cards;
      localVaultPool = reconciled.localVaultPool;
      latestManualCardId = reconciled.latestManualCardId;

      if (searchSession.isActive) finishCaptureSearch();
    } catch (err) {
      console.warn("handleCaptureChatLog failed, falling back to CaptureThought:", err);
      try {
        let fallbackDocId = '';
        if (images && images.length > 0) {
          const payload = images.map(img => (typeof img === 'string' ? { data: img } : img));
          const fallbackRes = await CaptureThoughtWithImages(text, payload as any);
          fallbackDocId = (fallbackRes as any)?.DocID || (fallbackRes as any)?.docId || '';
        } else {
          const fallbackRes = await CaptureThought(text);
          fallbackDocId = typeof fallbackRes === 'string' ? fallbackRes : ((fallbackRes as any)?.DocID || (fallbackRes as any)?.docId || '');
        }

        const updateToThought = (c: CosmicCard): CosmicCard => {
          if (c.id === tempId) {
            return {
              ...c,
              id: fallbackDocId || c.id,
              type: 'thought',
              title: 'Thought',
              content: text,
              isEnriching: false,
            };
          }
          return c;
        };
        cards = cards.map(updateToThought);
        localVaultPool = localVaultPool.map(updateToThought);
        if (latestManualCardId === tempId && fallbackDocId) {
          latestManualCardId = fallbackDocId;
        }
        if (searchSession.isActive) finishCaptureSearch();
      } catch (fallbackErr) {
        console.error("Fallback CaptureThought also failed:", fallbackErr);
        const rollback = rollbackOptimisticCard({
          tempId,
          cards,
          localVaultPool,
          latestManualCardId,
        });
        cards = rollback.cards;
        localVaultPool = rollback.localVaultPool;
        latestManualCardId = rollback.latestManualCardId;
        throw fallbackErr;
      }
    } finally {
      capturePending = false;
    }
  }

  async function handleCaptureNote(text: string) {
    capturePending = true;
    try {
      const docId = await CaptureNote(text);
      const note = createNoteCard({
        docId,
        text,
        cards,
        memoryPool,
        localVaultPool,
        scrollExpansion: $scrollExpansion,
        sourceCardId: activeBuguBeacon?.sourceCardId,
        latestManualCardId,
      });
      cards = note.cards;
      memoryPool = note.memoryPool;
      localVaultPool = note.localVaultPool;
      if (note.latestManualCardId) latestManualCardId = note.latestManualCardId;

      if (searchSession.isActive) finishCaptureSearch();
    } catch (err) {
      console.error("handleCaptureNote error:", err);
      throw err;
    } finally {
      capturePending = false;
    }
  }

  async function handleExecuteCommand(cmd: CommandDef, card: CosmicCard) {
    const paramSpec = cmd.params?.[0];
    const shouldAutoStage = Boolean(paramSpec?.autoStageToViewport || cmd.id.startsWith('bugu:') || cmd.id === 'atproto:echo');

    if (shouldAutoStage) {
      const res = ensureCardStagedInViewport(card, $scrollExpansion, cards, memoryPool, { isVIP: true });
      cards = res.updatedCards.map(c => c.id === card.id ? { ...c, elevatedAt: Date.now() } : c);
      if (res.updatedMemoryPool) memoryPool = res.updatedMemoryPool;
    }

    if (cmd.id.startsWith('bugu:') || cmd.id === 'atproto:echo') {
      buguController.startRadar(card.id, cards, memoryPool, localVaultPool);
    }
    await ExecuteModuleCommand(cmd.id, { cardId: card.id, card, content: card.content || card.title, id: card.id, text: card.content || card.title });
  }

  // --- Tag Association & Resurgence Recall Engine (Multi-tag Union Support) ---
  function handleTagClick(detail: { tag: string; cardId: string }) {
    const rawTag = detail?.tag;
    if (!rawTag) return;
    const tag = normalizeTag(rawTag);
    if (!tag) return;

    if (activeConstellationTags.has(tag)) {
      // Toggle OFF: delete from map and remove its tag_resurgence_ traces
      activeConstellationTags.delete(tag);
      activeConstellationTags = new Map(activeConstellationTags);

      const targetPrefix = 'tag_resurgence_' + tag + '_';
      cards = cards.map(c => ({
        ...c,
        auraTraces: (c.auraTraces || []).filter(t => t.searchId !== 'tag_resurgence_' + tag + '_' + c.id && !t.searchId.startsWith(targetPrefix)),
      }));
      return;
    }

    // Exit focused mode so the user can see the constellation on the cosmic stage
    if (focusedCardId !== null) {
      focusedCardId = null;
    }

    activateConstellationTag(tag);
  }

  function activateConstellationTag(tag: string) {
    // Pick a distinct vibrant color using color wheel max angular gap / palette
    const tagColor = getDistinctTagColor(activeConstellationTags.values());
    activeConstellationTags.set(tag, tagColor);
    activeConstellationTags = new Map(activeConstellationTags);

    stageActivity.resetActivityTimer();
    const now = Date.now();

    // Elevate cards already on stage that possess this tag or keyword:
    // 1. Z: 10 foreground depth (<= 16 visibility zone)
    // 2. Neon Trace: { searchId: 'tag_resurgence_' + tag + '_' + c.id, color: tagColor, hitTime: now, duration: 120000 }
    // 3. elevatedAt: now
    let currentCards = cards.map(c => {
      if (cardHasTag(c, tag) || cardHasKeyword(c, tag)) {
        const searchId = 'tag_resurgence_' + tag + '_' + c.id;
        const auraTrace: AuraTrace = {
          searchId,
          color: tagColor,
          hitTime: now,
          duration: 120000,
        };
        const existingTraces = (c.auraTraces || []).filter(t => t.searchId !== searchId);
        return {
          ...c,
          z: 10,
          auraTraces: [...existingTraces, auraTrace],
          decayDuration: Math.max(c.decayDuration || 0, 120000 + 3600000),
          isFadingOut: false,
          elevatedAt: now,
        };
      }
      return c;
    });

    // Query unstaged matching cards from memoryPool and localVaultPool for this tag
    const unstaged = findUnstagedMatchingCards(tag, currentCards, memoryPool, localVaultPool);
    if (unstaged.length === 0) {
      cards = currentCards;
      return;
    }

    let nextMemoryPool = [...memoryPool];
    const viewportCenterY = Math.round($scrollExpansion * 300);

    for (const candidate of unstaged) {
      // Stage candidate near viewport center
      const res = stageCandidate(
        candidate,
        false,
        $scrollExpansion,
        currentCards,
        nextMemoryPool,
        activeBuguBeacon?.sourceCardId || null,
        viewportCenterY
      );

      currentCards = res.updatedActiveCards;
      nextMemoryPool = res.updatedMemoryPool;

      // Resurgence Lifecycle:
      // Fixed at foreground Z: 10 during the 2-minute reading window.
      // Candidate receives trace for this newly activated tag, AND for any other active tags it matches.
      const tracesToAdd: AuraTrace[] = [];
      for (const [activeTag, color] of activeConstellationTags.entries()) {
        if (cardHasTag(candidate, activeTag) || cardHasKeyword(candidate, activeTag)) {
          tracesToAdd.push({
            searchId: 'tag_resurgence_' + activeTag + '_' + candidate.id,
            color,
            hitTime: now,
            duration: 120000,
          });
        }
      }

      currentCards = currentCards.map(c => {
        if (c.id === candidate.id) {
          const idsToAdd = new Set(tracesToAdd.map(t => t.searchId));
          const existing = (c.auraTraces || []).filter(t => !idsToAdd.has(t.searchId));
          return {
            ...c,
            z: 10,
            auraTraces: [...existing, ...tracesToAdd],
            decayDuration: Math.max(c.decayDuration || 0, 120000 + 3600000),
            isFadingOut: false,
            elevatedAt: now,
          };
        }
        return c;
      });

      // Ensure candidate is recorded in memoryPool with 24h macro TTL if permanent vault card
      if (!nextMemoryPool.some(m => m.id === candidate.id)) {
        nextMemoryPool = [
          {
            ...candidate,
            macroExpiresAt: now + MACRO_TTL_MS,
          },
          ...nextMemoryPool,
        ];
      }
    }

    cards = currentCards;
    memoryPool = nextMemoryPool;
  }

  function handleCardPromoted(detail: { originalId: string; card: CosmicCard }) {
    if (!detail) return;
    const { originalId, card: promoted } = detail;
    cards = cards.map(c => {
      if (c.id === originalId) {
        return {
          ...c,
          ...promoted,
          origin: 'local',
          macroExpiresAt: undefined,
        };
      }
      return c;
    });
    memoryPool = memoryPool.filter(c => c.id !== originalId && c.id !== promoted.id);
    if (focusedCardId === originalId) {
      focusedCardId = promoted.id;
    }
    if (!localVaultPool.some(v => v.id === promoted.id)) {
      localVaultPool = [promoted, ...localVaultPool];
    }
  }

  // --- Tilt, Focus, & Keydown Delegators ---
  let globalTilt = spring({ rx: 0, ry: 0 }, { stiffness: 0.04, damping: 0.25 });

  function handleMouseMove(e: MouseEvent) {
    stageActivity.resetActivityTimer();
    const mX = (e.clientX / window.innerWidth) * 2 - 1, mY = (e.clientY / window.innerHeight) * 2 - 1;
    globalTilt.set(focusedCardId === null ? { rx: mY * 4.5, ry: -mX * 4.5 } : { rx: 0, ry: 0 });
  }

  function toggleFocus(id: string) {
    focusedCardId = focusedCardId === id ? null : id;
    focusNavDirection = null;
    if (focusedCardId !== null) {
      globalTilt.set({ rx: 0, ry: 0 }, { hard: true });
      omniBarRef?.blurInput();
      omniBarRef?.closeCommandSelect();
      if (searchSession.isActive && !searchSession.isConfirmed) { confirmSearchView(); omniBarRef?.clearText(); }
    }
  }

  function navigateFocus(direction: 'prev' | 'next') {
    if (focusedCardId === null) return;

    const allCardsMap = new Map<string, CosmicCard>();
    cards.forEach(c => allCardsMap.set(c.id, c));
    memoryPool.forEach(c => allCardsMap.set(c.id, c));
    localVaultPool.forEach(c => allCardsMap.set(c.id, c));

    const orderedCards = Array.from(allCardsMap.values());
    if (orderedCards.length <= 1) return;

    const currentIndex = orderedCards.findIndex(c => c.id === focusedCardId);
    if (currentIndex === -1) return;

    let targetIndex = direction === 'next' ? currentIndex + 1 : currentIndex - 1;
    if (targetIndex < 0) targetIndex = orderedCards.length - 1;
    if (targetIndex >= orderedCards.length) targetIndex = 0;

    const targetCard = orderedCards[targetIndex];

    if (!cards.find(c => c.id === targetCard.id)) {
      cards = [...cards, {
        ...targetCard,
        spawnTime: targetCard.spawnTime || Date.now(),
        decayDuration: targetCard.decayDuration || 2 * 3600 * 1000,
        isFadingOut: false
      }];
    }

    focusNavDirection = direction;
    focusedCardId = targetCard.id;

    const oldCard = cards.find(c => c.id === orderedCards[currentIndex].id);
    if (oldCard && !oldCard.isVIP && oldCard.decayDuration === 0) {
      setTimeout(() => {
        if (focusedCardId !== oldCard.id) {
          cards = cards.filter(c => c.id !== oldCard.id);
        }
      }, 500);
    }
  }

  function handleWindowClick(e: MouseEvent) {
    const target = e.target as Element | null;
    if (target?.closest('.omnibar-container')) return;
    omniBarRef?.blurInput();
    omniBarRef?.closeCommandSelect();
    if (searchSession.isActive && !searchSession.isConfirmed) { revertSearchView(); omniBarRef?.clearText(); }
    if (target && !target.closest('.card') && !target.closest('.nav-button') && !target.closest('.thought-tag') && !target.closest('.babel-tag') && !target.closest('.note-tag') && !target.closest('.chatlog-tag')) {
      if (focusedCardId !== null) focusedCardId = null;
      if (activeConstellationTags.size > 0) {
        activeConstellationTags = new Map();
        cards = cards.map(c => ({
          ...c,
          auraTraces: (c.auraTraces || []).filter(t => !t.searchId.startsWith('tag_resurgence_')),
        }));
      }
    }
  }

  function handleWindowKeydown(e: KeyboardEvent) {
    stageActivity.resetActivityTimer();
    const isMetaOrCtrl = e.metaKey || e.ctrlKey;
    const key = e.key.toLowerCase();

    // 1. Force refresh / Hard reload (Strictly requires Shift: Cmd+Shift+R, Ctrl+Shift+R, Shift+F5)
    if ((isMetaOrCtrl && e.shiftKey && key === 'r') || (e.shiftKey && e.key === 'F5')) {
      e.preventDefault();
      window.location.reload();
      return;
    }

    if (e.key === 'Escape') {
      if (activeConstellationTags.size > 0) {
        activeConstellationTags = new Map();
        cards = cards.map(c => ({
          ...c,
          auraTraces: (c.auraTraces || []).filter(t => !t.searchId.startsWith('tag_resurgence_')),
        }));
        return;
      }
      if (focusedCardId !== null) { focusedCardId = null; return; }
      if (searchSession.isActive && searchSession.isConfirmed) {
        restoreConfirmedSearch();
        return;
      }
      if (omniBarRef?.handleEscape()) return;
    }
    if (focusedCardId !== null) {
      if (e.key === 'ArrowLeft') { navigateFocus('prev'); return; }
      if (e.key === 'ArrowRight') { navigateFocus('next'); return; }
    }
    if (e.key === 'Enter') omniBarRef?.handleGlobalEnter();
  }

  function applyCardUpdate(updatedCard: any) {
    if (!updatedCard) return;
    const updated = toCosmicCard(updatedCard, updatedCard.type || 'thought', 'local');

    cards = cards.map(existing => {
      const isMatch = existing.id === updated.id || 
        (Boolean(existing.filePath) && Boolean(updated.filePath) && existing.filePath === updated.filePath) ||
        (Boolean(existing.filePath) && existing.filePath.endsWith(updated.id)) ||
        (Boolean(updated.filePath) && updated.filePath.endsWith(existing.id));
      if (isMatch) {
        return {
          ...existing,
          title: updated.title,
          content: updated.content,
          originalText: updated.originalText,
          translatedText: updated.translatedText,
          images: updated.images,
          keywords: updated.keywords || existing.keywords,
          type: updated.type || existing.type,
          filePath: updated.filePath || existing.filePath,
        };
      }
      return existing;
    });

    localVaultPool = localVaultPool.map(existing => {
      const isMatch = existing.id === updated.id || 
        (Boolean(existing.filePath) && Boolean(updated.filePath) && existing.filePath === updated.filePath);
      if (isMatch) {
        return {
          ...existing,
          title: updated.title,
          content: updated.content,
          images: updated.images,
          keywords: updated.keywords || existing.keywords,
          type: updated.type || existing.type,
          filePath: updated.filePath || existing.filePath,
        };
      }
      return existing;
    });

    memoryPool = memoryPool.map(existing => {
      const isMatch = existing.id === updated.id || 
        (Boolean(existing.filePath) && Boolean(updated.filePath) && existing.filePath === updated.filePath);
      if (isMatch) {
        return {
          ...existing,
          title: updated.title,
          content: updated.content,
          images: updated.images,
          keywords: updated.keywords || existing.keywords,
          type: updated.type || existing.type,
          filePath: updated.filePath || existing.filePath,
        };
      }
      return existing;
    });
  }

  onMount(() => {
    GetRegisteredCommands().then(cmds => { if (Array.isArray(cmds)) availableCommands = cmds; }).catch(console.warn);

    Promise.all([LoadLocalCards(), GetTempCards()]).then(([localCards, tempCards]) => {
      if (tempCards?.length) memoryPool = tempCards.map(c => ({ ...toCosmicCard(c, c.type || 'babel', (c as any).origin || 'bugu', MACRO_TTL_MS), isFresh: false, spawnTime: 0, decayDuration: 0, z: 50 }));
      if (localCards?.length) localVaultPool = localCards.map(c => ({ ...toCosmicCard(c, c.type || 'thought', 'local'), isFresh: false, spawnTime: 0, decayDuration: 0, z: 50 }));
      replenishStage();
    }).catch(err => console.error("loadCards error:", err));

    EventsOn("thought:captured", (data: any) => {
      const merged = mergeCapturedThought({
        data,
        cards,
        memoryPool,
        localVaultPool,
        scrollExpansion: $scrollExpansion,
        sourceCardId: activeBuguBeacon?.sourceCardId,
        latestManualCardId,
      });
      cards = merged.cards;
      memoryPool = merged.memoryPool;
      localVaultPool = merged.localVaultPool;
      latestManualCardId = merged.latestManualCardId;
    });

    EventsOn("chatlog:card_spawned", (c: any) => {
      const card = toCosmicCard(c, 'chat_log', 'bugu', MACRO_TTL_MS);
      memoryPool = [card, ...memoryPool.filter(item => item.id !== card.id)];
      replenishStage();
    });

    EventsOn("bugu:card_spawned", (c: any) => {
      buguController.enqueueBabelCard(c);
    });

    EventsOn("bugu:radar_completed", () => {
      buguController.finishRadar();
    });

    EventsOn("card:updated", (c: any) => {
      applyCardUpdate(c);
    });

    EventsOn("card:deleted", (data: any) => {
      const id = data?.id;
      const filePath = data?.filePath;
      if (id || filePath) {
        cards = cards.filter(c => c.id !== id && (!filePath || c.filePath !== filePath));
        localVaultPool = localVaultPool.filter(c => c.id !== id && (!filePath || c.filePath !== filePath));
        memoryPool = memoryPool.filter(c => c.id !== id && (!filePath || c.filePath !== filePath));
        if (focusedCardId === id) focusedCardId = null;
      }
    });

    const handleWindowFocus = () => {
      // Focus-driven relative time refresh (刚刚 / 今天 / 昨天 / N天前)
      refreshStageTime();

      if (focusedCardId) {
        const focusedCard = cards.find(c => c.id === focusedCardId);
        const target = focusedCard?.filePath || focusedCardId;
        ReloadCard(target).then(refreshed => {
          if (refreshed) applyCardUpdate(refreshed);
        }).catch(() => {});
      }

      const now = Date.now();
      const fadingIds: string[] = [];
      cards = cards.map(c => {
        if (c.id === focusedCardId) return c;
        if (c.isFadingOut) return c;

        const isTemporary = Boolean(c.isTemporary || c.origin === 'bugu' || c.type === 'radar');
        if (isTemporary) {
          if (isProtectedCard(c, cards, activeBuguBeacon?.sourceCardId)) return c;
          if (c.decayDuration && (now - c.spawnTime >= c.decayDuration)) {
            if (c.z < 65) {
              return {
                ...c,
                z: c.z + 20 + Math.random() * 5,
                spawnTime: now
              };
            } else {
              fadingIds.push(c.id);
              return {
                ...c,
                isFadingOut: true
              };
            }
          }
          return c;
        }

        // Active Bugu mother card is protected during active radar session
        if (activeBuguBeacon?.sourceCardId && c.id === activeBuguBeacon.sourceCardId) {
          return c;
        }

        // Universal Gravitational Fallback Engine for permanent/normal cards:
        // Evaluates cards whose 10-minute elevation protection has elapsed
        return applyGravitationalFallback(c, now);
      });

      if (fadingIds.length > 0) {
        setTimeout(() => {
          cards = cards.filter(c => !fadingIds.includes(c.id));
        }, 2500);
      }
    };
    window.addEventListener('focus', handleWindowFocus);

    const handleVisibilityChange = () => {
      if (typeof document !== 'undefined' && document.visibilityState === 'visible') {
        handleWindowFocus();
      }
    };
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', handleVisibilityChange);
    }

    const metabolicInterval = setInterval(() => {
      refreshStageTime();
      if (searchSession.isActive && !searchSession.isConfirmed) return;
      const now = Date.now();
      const expiredCards = memoryPool.filter(c => c.macroExpiresAt && c.macroExpiresAt <= now);
      if (expiredCards.length > 0) {
        memoryPool = memoryPool.filter(c => !c.macroExpiresAt || c.macroExpiresAt > now);
        for (const ec of expiredCards) {
          if (ec.origin === 'bugu') {
            EvictTempCard(ec.id).catch(() => {});
          }
        }
      }

      cards.forEach(c => {
        if (c.id === focusedCardId) return;
        if (c.isFresh && now - c.spawnTime > 8 * 60 * 1000) c.isFresh = false;
      });
      replenishStage();
    }, 45000);

    return () => {
      window.removeEventListener('focus', handleWindowFocus);
      if (typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', handleVisibilityChange);
      }
      clearInterval(metabolicInterval);
      if (scrollIdleTimer) clearTimeout(scrollIdleTimer);
      if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
      buguController.destroy();
      stageActivity.destroy();
    };
  });
</script>

<svelte:window on:keydown={handleWindowKeydown} on:mousemove={handleMouseMove} on:wheel={handleWheel} on:click={handleWindowClick} />

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="container" on:click={handleWindowClick}>
  <CosmicStage
    {cards}
    totalAvailableCards={cards.length + memoryPool.length + localVaultPool.length}
    buguBeacon={activeBuguBeacon} buguSession={activeBuguSession}
    {focusNavDirection}
    globalTilt={$globalTilt} scrollExpansion={$scrollExpansion} {focusedCardId} {hoveredCardId} {isOmnibarFocused}
    isSearchActive={searchSession.isActive} {isStageSleeping} {isWindowFocused}
    {lastScrollDirection}
    {activeConstellationTags}
    {activeConstellationTag}
    onToggleFocus={toggleFocus} onHoverCard={(id) => { hoveredCardId = id; }}
    onLeaveCard={() => hoveredCardId = null} onBgClick={handleWindowClick}
    on:navigate={(e) => navigateFocus(e.detail)}
    on:tagClick={(e) => handleTagClick(e.detail)}
    on:cardPromoted={(e) => handleCardPromoted(e.detail)}
  />

  <OmniBar
    bind:this={omniBarRef} {availableCommands} {searchSession} {focusedCardId} {cards} {localVaultPool} {capturePending}
    {isScrolling} scrollExpansion={$scrollExpansion}
    bind:isOmnibarFocused onCaptureThought={handleCaptureThought} onCaptureChatLog={handleCaptureChatLog} onCaptureNote={handleCaptureNote} onFreeSearch={handleFreeSearch}
    onRevertSearch={revertSearchView} onCommitSearch={confirmSearchView} onRestoreSearch={restoreConfirmedSearch}
    onExecuteCommand={handleExecuteCommand} onSearchLocalCards={SearchLocalCards}
  />
</div>

<style>
  :global(body), :global(html), :global(#app) {
    background: transparent !important; background-color: transparent !important;
    margin: 0; padding: 0; width: 100vw; height: 100vh; overflow: hidden;
  }
  .container {
    width: 100vw; height: 100vh; position: relative;
    display: flex; justify-content: center; align-items: center; background: transparent;
  }
</style>
