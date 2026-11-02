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
  import WindowTitleBar from './components/WindowTitleBar.svelte';
  import { CaptureThought, CaptureThoughtWithImages, CaptureChatLog, CaptureChatLogWithImages, CaptureNote, GetTempCards, LoadLocalCards, SearchLocalCards, GetRegisteredCommands, GetModuleManifests, ExecuteModuleCommand, EvictTempCard, ReloadCard } from '../wailsjs/go/main/App.js';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import type { CosmicCard, SearchSession, CommandDef, ModuleManifest } from './types/card';
  import { hydrateModuleManifests, getModuleBeaconType } from './lib/promotionRegistry';
  import { loadExternalPluginCardComponents, syncModularCardComponents } from './components/cards/cardRegistry';
  import type { BuguSession, BuguBeacon } from './modules/bugu/types';
  import type { BskyBeacon } from './modules/atproto/types';
  import { generateAuraColor, transformSearchResults, calculateSearchRepulsion, commitUnallocatedSearchResults } from './lib/auraSearch';
  import { BASE_CAPACITY, MACRO_TTL_MS, BASE_STAGE_DECAY_MS, VIP_INK_BEAM_DURATION_MS, stageCandidate, replenishStageCandidates, toCosmicCard, ensureCardStagedInViewport } from './lib/stageDecay';
  import { computeNextScrollTarget } from './lib/stageCoords';
  import { isChatLogFormat } from './lib/chatlog';
  import { createBuguSessionController } from './modules/bugu/buguSession';
  import { createBabelSessionController } from './modules/babel/babelSession';
  import { createBskySessionController } from './modules/atproto/bskySession';
  import { createStageActivityController } from './lib/stageActivity';
  import {
    createOptimisticChatLogCard,
    reconcileChatLogCard,
    rollbackOptimisticCard,
    createNoteCard,
    mergeCapturedThought,
    promoteCardInPools,
    updateCardInPools,
    removeCardFromPools,
  } from './lib/cardCapture';
  import { toggleTagConstellation } from './lib/tagNetwork';
  import { evaluateCardsOnWindowFocus } from './lib/gravityFallback';
  import { refreshStageTime } from './lib/time';
  import { extractCardThemes } from './lib/themeOrbit';

  // --- Stage Pools & Selection State ---
  let memoryPool: CosmicCard[] = [], localVaultPool: CosmicCard[] = [], cards: CosmicCard[] = [];
  let latestManualCardId: string | null = null, focusedCardId: string | null = null, hoveredCardId: string | null = null;
  let activeThemeId: string | null = null;
  let activeConstellationTags: Map<string, string> = new Map();
  $: activeConstellationTag = activeConstellationTags.size > 0 ? Array.from(activeConstellationTags.keys())[0] : null;

  // --- Theme Thread & Carousel Orbit State ---
  $: allKnownCards = (() => {
    const map = new Map<string, CosmicCard>();
    cards.forEach(c => map.set(c.id, c));
    memoryPool.forEach(c => map.set(c.id, c));
    localVaultPool.forEach(c => map.set(c.id, c));
    return Array.from(map.values());
  })();

  $: focusedCard = focusedCardId ? (allKnownCards.find(c => c.id === focusedCardId) || null) : null;
  $: cardThemes = focusedCard ? extractCardThemes(focusedCard, allKnownCards) : [];
  $: activeTheme = (activeThemeId && cardThemes.length > 0) ? (cardThemes.find(t => t.id === activeThemeId) || null) : null;
  $: themeCards = activeTheme ? activeTheme.cards : [];

  $: leftPeekCardId = (() => {
    if (!activeThemeId || themeCards.length < 2 || !focusedCardId) return null;
    const idx = themeCards.findIndex(c => c.id === focusedCardId);
    if (idx === -1) return null;
    if (themeCards.length === 2) {
      return themeCards[(idx + 1) % 2].id;
    }
    const prevIdx = (idx - 1 + themeCards.length) % themeCards.length;
    return themeCards[prevIdx].id;
  })();

  $: rightPeekCardId = (() => {
    if (!activeThemeId || themeCards.length < 3 || !focusedCardId) return null;
    const idx = themeCards.findIndex(c => c.id === focusedCardId);
    if (idx === -1) return null;
    const nextIdx = (idx + 1) % themeCards.length;
    return themeCards[nextIdx].id;
  })();
  let isOmnibarFocused = false, capturePending = false;
  let isWindowFocused = true, isStageSleeping = false;
  let focusNavDirection: 'prev' | 'next' | null = null;
  let omniBarRef: OmniBar, availableCommands: CommandDef[] = [];
  let availableModuleManifests: ModuleManifest[] = [];
  let searchSession: SearchSession = { isActive: false, isConfirmed: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
  let searchDebounceTimer: any = null;
  let activeBuguSession: BuguSession | null = null;
  let activeBuguBeacon: BuguBeacon | null = null;
  let activeBskyBeacon: BskyBeacon | null = null;

  // --- Gravitational Tide (Multi-Screen Y-Axis Expansion) ---
  let scrollTarget = 0;
  let scrollExpansion = spring(0, { stiffness: 0.16, damping: 0.72 });
  let isScrolling = false;
  let scrollIdleTimer: any = null;
  let lastScrollDirection: 'down' | 'up' = 'down';
  $: targetCapacity = Math.round(BASE_CAPACITY * (1 + $scrollExpansion * 0.6));

  // --- Encapsulated Controllers ---
  const bskyController = createBskySessionController({
    onBeaconChange: (beacon) => { activeBskyBeacon = beacon; },
  });

  const babelController = createBabelSessionController({
    getCards: () => cards,
    getMemoryPool: () => memoryPool,
    getScrollExpansion: () => $scrollExpansion,
    onUpdateStage: (newCards, newMem) => {
      cards = newCards;
      if (newMem) memoryPool = newMem;
    },
    toCosmicCard,
    stageCandidate,
    onCardStaged: (c) => {
      if (c?.origin === 'atproto' || activeBskyBeacon) {
        bskyController.incrementButterfly();
      }
    },
    onQueueEmpty: () => {
      buguController.finishRadar();
      bskyController.finishBeacon();
    },
  });

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

  async function handleCaptureCurio(text: string, images?: string[]) {
    capturePending = true;
    try {
      const app = (window as any).go?.main?.App;
      if (app && typeof app.CaptureCurio === 'function') {
        const payload = (images || []).map(img => (typeof img === 'string' ? { data: img } : img));
        await app.CaptureCurio(text, payload);
      } else if (images && images.length > 0) {
        const payload = images.map(img => (typeof img === 'string' ? { data: img } : img));
        await CaptureThoughtWithImages(`:curio ${text}`, payload as any);
      } else {
        await CaptureThought(`:curio ${text}`);
      }
      if (searchSession.isActive) finishCaptureSearch();
    } catch (err) {
      console.error("handleCaptureCurio error:", err);
      throw err;
    } finally {
      capturePending = false;
    }
  }

  async function handleExecuteCommand(cmd: CommandDef, card?: CosmicCard | null, textQuery?: string) {
    const paramSpec = cmd.params?.[0];
    const beaconType = getModuleBeaconType(cmd.id) || getModuleBeaconType(cmd.namespace);
    const shouldAutoStage = Boolean(
      card && (
        paramSpec?.autoStageToViewport ||
        beaconType === 'radar' ||
        beaconType === 'butterfly' ||
        cmd.id.startsWith('bugu:') ||
        cmd.id === 'atproto:echo'
      )
    );

    if (shouldAutoStage && card) {
      const res = ensureCardStagedInViewport(card, $scrollExpansion, cards, memoryPool, { isVIP: true });
      cards = res.updatedCards.map(c => c.id === card.id ? { ...c, elevatedAt: Date.now() } : c);
      if (res.updatedMemoryPool) memoryPool = res.updatedMemoryPool;
    }

    if (beaconType === 'radar' || cmd.id.startsWith('bugu:')) {
      if (card) {
        buguController.startRadar(card.id, cards, memoryPool, localVaultPool);
      }
    } else if (beaconType === 'butterfly' || cmd.id === 'atproto:echo') {
      bskyController.startEcho(card?.id || null);
    }
    await ExecuteModuleCommand(cmd.id, {
      cardId: card?.id,
      card: card || undefined,
      content: textQuery || card?.content || card?.title,
      id: card?.id,
      text: textQuery || card?.content || card?.title,
    });
  }

  // --- Tag Association & Resurgence Recall Engine (Multi-tag Union Support) ---
  function handleTagClick(detail: { tag: string; cardId: string }) {
    const rawTag = detail?.tag;
    if (!rawTag) return;

    const res = toggleTagConstellation({
      rawTag,
      activeConstellationTags,
      cards,
      memoryPool,
      localVaultPool,
      scrollExpansion: $scrollExpansion,
      sourceCardId: activeBuguBeacon?.sourceCardId,
    });

    activeConstellationTags = res.activeConstellationTags;
    cards = res.cards;
    memoryPool = res.memoryPool;
    if (res.shouldClearFocus && focusedCardId !== null) {
      focusedCardId = null;
    }
    if (res.didActivate) {
      stageActivity.resetActivityTimer();
    }
  }

  function handleCardPromoted(detail: { originalId: string; card: CosmicCard }) {
    const res = promoteCardInPools(detail, { cards, memoryPool, localVaultPool, focusedCardId });
    cards = res.cards;
    memoryPool = res.memoryPool;
    localVaultPool = res.localVaultPool;
    focusedCardId = res.focusedCardId;
  }

  // --- Tilt, Focus, & Keydown Delegators ---
  let globalTilt = spring({ rx: 0, ry: 0 }, { stiffness: 0.04, damping: 0.25 });

  function handleMouseMove(e: MouseEvent) {
    stageActivity.resetActivityTimer();
    const mX = (e.clientX / window.innerWidth) * 2 - 1, mY = (e.clientY / window.innerHeight) * 2 - 1;
    globalTilt.set(focusedCardId === null ? { rx: mY * 4.5, ry: -mX * 4.5 } : { rx: 0, ry: 0 });
  }

  function toggleFocus(id: string) {
    if (focusedCardId === id) {
      focusedCardId = null;
      activeThemeId = null;
      focusNavDirection = null;
      return;
    }
    if (activeThemeId && (id === leftPeekCardId || id === rightPeekCardId)) {
      focusNavDirection = id === leftPeekCardId ? 'prev' : 'next';
    } else {
      focusNavDirection = null;
      activeThemeId = null;
    }
    focusedCardId = id;
    if (focusedCardId !== null) {
      globalTilt.set({ rx: 0, ry: 0 }, { hard: true });
      omniBarRef?.blurInput();
      omniBarRef?.closeCommandSelect();
      if (searchSession.isActive && !searchSession.isConfirmed) { confirmSearchView(); omniBarRef?.clearText(); }
    }
  }

  function handleToggleTheme(detail: any) {
    const themeId = typeof detail === 'string' ? detail : detail?.themeId;
    if (!themeId) return;

    if (activeThemeId === themeId) {
      activeThemeId = null;
      return;
    }

    const targetTheme = cardThemes.find(t => t.id === themeId);
    if (targetTheme && targetTheme.cards.length > 0) {
      let toAdd: CosmicCard[] = [];
      for (const tc of targetTheme.cards) {
        if (!cards.some(c => c.id === tc.id) && !toAdd.some(c => c.id === tc.id)) {
          toAdd.push({
            ...tc,
            spawnTime: tc.spawnTime || Date.now(),
            decayDuration: tc.decayDuration || 2 * 3600 * 1000,
            isFadingOut: false
          });
        }
      }
      if (toAdd.length > 0) {
        cards = [...cards, ...toAdd];
      }

      const isCurrentInTheme = targetTheme.cards.some(c => c.id === focusedCardId);
      if (!isCurrentInTheme) {
        focusedCardId = targetTheme.cards[0].id;
      }
    }
    activeThemeId = themeId;
  }

  function navigateFocus(direction: 'prev' | 'next') {
    if (focusedCardId === null) return;

    // Theme carousel mode: cycle strictly within theme cards
    if (activeThemeId && themeCards.length > 1) {
      const currentIndex = themeCards.findIndex(c => c.id === focusedCardId);
      if (currentIndex !== -1) {
        let targetIndex = direction === 'next' ? currentIndex + 1 : currentIndex - 1;
        if (targetIndex < 0) targetIndex = themeCards.length - 1;
        if (targetIndex >= themeCards.length) targetIndex = 0;

        const targetCard = themeCards[targetIndex];
        if (!cards.some(c => c.id === targetCard.id)) {
          cards = [...cards, {
            ...targetCard,
            spawnTime: targetCard.spawnTime || Date.now(),
            decayDuration: targetCard.decayDuration || 2 * 3600 * 1000,
            isFadingOut: false
          }];
        }
        focusNavDirection = direction;
        focusedCardId = targetCard.id;
        return;
      }
    }

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
    if (target && !target.closest('.card') && !target.closest('.nav-button') && !target.closest('.thought-tag') && !target.closest('.babel-tag') && !target.closest('.note-tag') && !target.closest('.chatlog-tag') && !target.closest('.orbital-theme-btn')) {
      if (focusedCardId !== null) {
        focusedCardId = null;
        activeThemeId = null;
      }
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
      if (focusedCardId !== null) {
        focusedCardId = null;
        activeThemeId = null;
        return;
      }
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
    const res = updateCardInPools(updatedCard, { cards, memoryPool, localVaultPool });
    cards = res.cards;
    memoryPool = res.memoryPool;
    localVaultPool = res.localVaultPool;
  }

  onMount(() => {
    GetRegisteredCommands().then(cmds => { if (Array.isArray(cmds)) availableCommands = cmds; }).catch(console.warn);
    if (typeof GetModuleManifests === 'function') {
      GetModuleManifests().then((manifests: any) => {
        if (Array.isArray(manifests)) {
          availableModuleManifests = manifests;
          hydrateModuleManifests(manifests);
          syncModularCardComponents(manifests);
          loadExternalPluginCardComponents(manifests).catch(console.warn);
        }
      }).catch(console.warn);
    }

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

    function handleBabelCardSpawned(c: any) {
      babelController.enqueue(c, c?.origin);
      if (c?.origin === 'atproto' || activeBskyBeacon) {
        bskyController.incrementButterfly();
      }
    }

    function handleBabelSessionCompleted() {
      babelController.finishSession();
      buguController.finishRadar();
      bskyController.finishBeacon();
    }

    EventsOn("babel:card_spawned", handleBabelCardSpawned);
    EventsOn("bugu:card_spawned", handleBabelCardSpawned);

    EventsOn("babel:session_completed", handleBabelSessionCompleted);
    EventsOn("bugu:radar_completed", handleBabelSessionCompleted);

    EventsOn("card:updated", (c: any) => {
      applyCardUpdate(c);
    });

    EventsOn("card:deleted", (data: any) => {
      const res = removeCardFromPools(data, { cards, memoryPool, localVaultPool, focusedCardId });
      cards = res.cards;
      memoryPool = res.memoryPool;
      localVaultPool = res.localVaultPool;
      focusedCardId = res.focusedCardId;
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
      const res = evaluateCardsOnWindowFocus(
        cards,
        now,
        focusedCardId,
        activeBuguBeacon?.sourceCardId || null,
        latestManualCardId
      );
      cards = res.updatedCards;
      latestManualCardId = res.updatedLatestManualCardId;

      if (res.fadingIds.length > 0) {
        setTimeout(() => {
          cards = cards.filter(c => !res.fadingIds.includes(c.id));
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
        if (c.isVIP) {
          const birthTime = c.spawnTime || (c.createdAt ? new Date(c.createdAt).getTime() : 0);
          if (birthTime && (now - birthTime >= VIP_INK_BEAM_DURATION_MS)) {
            c.isVIP = false;
            if (latestManualCardId === c.id) {
              latestManualCardId = null;
            }
          }
        }
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
      bskyController.destroy();
      babelController.destroy();
      buguController.destroy();
      stageActivity.destroy();
    };
  });
</script>

<svelte:window on:keydown={handleWindowKeydown} on:mousemove={handleMouseMove} on:wheel={handleWheel} on:click={handleWindowClick} />

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="container" on:click={handleWindowClick}>
  <WindowTitleBar />
  <CosmicStage
    {cards}
    {activeThemeId}
    {cardThemes}
    {leftPeekCardId}
    {rightPeekCardId}
    totalAvailableCards={cards.length + memoryPool.length + localVaultPool.length}
    buguBeacon={activeBuguBeacon} buguSession={activeBuguSession}
    bskyBeacon={activeBskyBeacon}
    {focusNavDirection}
    globalTilt={$globalTilt} scrollExpansion={$scrollExpansion} {focusedCardId} {hoveredCardId} {isOmnibarFocused}
    isSearchActive={searchSession.isActive} {isStageSleeping} {isWindowFocused}
    {lastScrollDirection}
    {activeConstellationTags}
    {activeConstellationTag}
    onToggleFocus={toggleFocus} onHoverCard={(id) => { hoveredCardId = id; }}
    onLeaveCard={() => hoveredCardId = null} onBgClick={handleWindowClick}
    on:navigate={(e) => navigateFocus(e.detail)}
    on:toggleTheme={(e) => handleToggleTheme(e.detail)}
    on:tagClick={(e) => handleTagClick(e.detail)}
    on:cardPromoted={(e) => handleCardPromoted(e.detail)}
  />

  <OmniBar
    bind:this={omniBarRef} {availableCommands} {searchSession} {focusedCardId} {cards} {localVaultPool} {capturePending}
    {isScrolling} scrollExpansion={$scrollExpansion}
    {availableModuleManifests}
    bind:isOmnibarFocused onCaptureThought={handleCaptureThought} onCaptureChatLog={handleCaptureChatLog} onCaptureNote={handleCaptureNote} onCaptureCurio={handleCaptureCurio} onFreeSearch={handleFreeSearch}
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
