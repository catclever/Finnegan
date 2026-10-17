<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - Main App Component Draft (Task: night_shift_overflow_scroll_focus_018)
  1. Problem Fixed: Protected Bugu mother cards from eviction/fade-out; enabled multi-screen
     overflow up to 3x capacity (30 cards) across Screens 1-3; smoothed Gravitational Tide
     wheel scrolling using an accumulated `scrollTarget` with impulse clamping.
  2. Resolution: Added `isProtectedCard` checks in `thought:captured`, `bugu:card_spawned`,
     `metabolicInterval`, and `resetTideTimer`; marked `sourceCardId` as VIP (12h) in
     `startBuguRadar`; drove `scrollExpansion` spring (`0.08/0.45`) via `computeNextScrollTarget`.
  3. Caveats: `resetTideTimer` only fades out `origin === 'local'` deep cards (`y > 400`)
     after 45s idle so Bugu overflow echoes in Screen 2/3 remain accessible.
  ======================================================================================
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { spring } from 'svelte/motion';
  import OmniBar from './components/OmniBar.svelte';
  import CosmicStage from './components/CosmicStage.svelte';
  import { CaptureThought, CaptureThoughtWithImages, CaptureChatLog, CaptureChatLogWithImages, GetTempCards, LoadLocalCards, SearchLocalCards, GetRegisteredCommands, ExecuteModuleCommand } from '../wailsjs/go/main/App.js';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import type { CosmicCard, SearchSession, CommandDef } from './types/card';
  import type { BuguSession, BuguBeacon } from './modules/bugu/types';
  import { generateAuraColor, transformSearchResults, calculateSearchRepulsion, commitUnallocatedSearchResults } from './lib/auraSearch';
  import { BASE_CAPACITY, MACRO_TTL_MS, BASE_STAGE_DECAY_MS, stageCandidate, replenishStageCandidates, isProtectedCard, computeNextScrollTarget, toCosmicCard } from './lib/stageDecay';
  import { isChatLogFormat } from './lib/chatlog';

  // --- Stage Pools & Selection State ---
  let memoryPool: CosmicCard[] = [], localVaultPool: CosmicCard[] = [], cards: CosmicCard[] = [];
  let latestManualCardId: string | null = null, focusedCardId: string | null = null, hoveredCardId: string | null = null;
  let isOmnibarFocused = false, capturePending = false;
  let isWindowFocused = true, isStageSleeping = false;
  let activityIdleTimer: any = null;
  let focusNavDirection: 'prev' | 'next' | null = null;
  let omniBarRef: OmniBar, availableCommands: CommandDef[] = [];
  let searchSession: SearchSession = { isActive: false, isConfirmed: false, searchId: '', stashedCards: [], rankedResults: [], color: '' };
  let searchDebounceTimer: any = null, tideTimer: any = null;
  let activeBuguSession: BuguSession | null = null;
  let activeBuguBeacon: BuguBeacon | null = null, buguBeaconTimer: any = null;
  let babelQueue: any[] = [], babelQueueTimer: any = null, radarBackendFinished = false;

  // --- Gravitational Tide (Multi-Screen Y-Axis Expansion) ---
  let scrollTarget = 0;
  let scrollExpansion = spring(0, { stiffness: 0.16, damping: 0.72 });
  let isScrolling = false;
  let scrollIdleTimer: any = null;
  let lastScrollDirection: 'down' | 'up' = 'down';
  $: targetCapacity = Math.round(BASE_CAPACITY * (1 + $scrollExpansion * 0.6));

  function resetTideTimer() {
    if (tideTimer) clearTimeout(tideTimer);
    tideTimer = setTimeout(() => {
      scrollTarget = 0;
      scrollExpansion.set(0);
      setTimeout(() => {
        cards.forEach(c => {
          if (c.y > 400 && c.origin === 'local' && !isProtectedCard(c, cards, activeBuguBeacon?.sourceCardId) && !c.isFadingOut) {
            c.isFadingOut = true;
            setTimeout(() => { cards = cards.filter(item => item.id !== c.id); }, 2500);
          }
        });
      }, 1000);
    }, 45000);
  }

  function handleWheel(e: WheelEvent) {
    resetActivityTimer();
    if (focusedCardId !== null) return;
    if (Math.abs(e.deltaY) > 0.5) {
      lastScrollDirection = e.deltaY > 0 ? 'down' : 'up';
    }
    isScrolling = true;
    if (scrollIdleTimer) clearTimeout(scrollIdleTimer);
    scrollIdleTimer = setTimeout(() => {
      isScrolling = false;
    }, 450);

    // Dynamic reversal anchoring:
    // If user reverses scroll direction, immediately anchor baseTarget to the live spring value ($scrollExpansion)
    // to eliminate latency and prevent having to scroll through accumulated overshoot buffer.
    const currentExp = $scrollExpansion;
    let baseTarget = scrollTarget;
    if (e.deltaY < 0 && baseTarget > currentExp) {
      baseTarget = currentExp;
    } else if (e.deltaY > 0 && baseTarget < currentExp) {
      baseTarget = currentExp;
    }

    scrollTarget = computeNextScrollTarget(baseTarget, e.deltaY, e.deltaMode, cards);
    scrollExpansion.set(scrollTarget);
    resetTideTimer();
    if (e.deltaY > 0) replenishStage();
  }

  function replenishStage() {
    const res = replenishStageCandidates({ targetCapacity, cards, memoryPool, localVaultPool, searchSession, scrollExpansion: $scrollExpansion });
    if (res.updatedMemoryPool) memoryPool = res.updatedMemoryPool;
    if (res.remainingRankedResults !== undefined) searchSession.rankedResults = res.remainingRankedResults;
    if (res.updatedCards) {
      cards = [...res.updatedCards.filter(c => !res.newCardsToStage.find(n => n.id === c.id)), ...res.newCardsToStage];
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
    if (query.length < 2) {
      if (searchSession.isActive && !searchSession.isConfirmed) {
        revertSearchView();
      }
      return;
    }
    // Typing a new query after a confirmed search starts a fresh reversible session.
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
    const tempId = 'optimistic_chatlog_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7);

    if (latestManualCardId) {
      const prevVIP = cards.find(c => c.id === latestManualCardId);
      if (prevVIP && !isProtectedCard({ ...prevVIP, isVIP: false }, cards, activeBuguBeacon?.sourceCardId)) {
        prevVIP.isVIP = false;
        prevVIP.decayDuration = BASE_STAGE_DECAY_MS;
      }
    }
    latestManualCardId = tempId;

    const baseCard = toCosmicCard({ 
      id: tempId, 
      title: '正在提炼主题...',
      type: 'chat_log',
      content: text, 
      images: images || [],
      isEnriching: true,
    }, 'chat_log', 'manual');
    baseCard.isEnriching = true;

    const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(baseCard, true, $scrollExpansion, cards, memoryPool, activeBuguBeacon?.sourceCardId);
    if (updatedMemoryPool) memoryPool = updatedMemoryPool;
    const newCard = { 
      ...stagedCard, 
      images: images || stagedCard.images, 
      isVIP: true, 
      decayDuration: 12 * 3600 * 1000, 
      z: 12, 
      isEnriching: true 
    };

    localVaultPool = [newCard, ...localVaultPool.filter(c => c.id !== newCard.id)];
    cards = [...updatedActiveCards.filter(c => c.id !== newCard.id), newCard];

    try {
      let res: any;
      if (images && images.length > 0) {
        const payload = images.map(img => (typeof img === 'string' ? { data: img } : img));
        res = await CaptureChatLogWithImages(text, payload as any);
      } else {
        res = await CaptureChatLog(text);
      }

      const realId = res?.docId || res?.DocID;
      const realTitle = res?.title || res?.Title;
      const realParticipants = res?.participants || res?.Participants;
      const realStartedAt = res?.startedAt || res?.StartedAt;
      const realEndedAt = res?.endedAt || res?.EndedAt;
      const realImages = res?.images || res?.Images || images;

      if (latestManualCardId === tempId && realId) {
        latestManualCardId = realId;
      }

      cards = cards.map(c => {
        if (c.id === tempId || (realId && c.id === realId)) {
          return {
            ...c,
            id: realId || c.id,
            title: realTitle || c.title,
            participants: realParticipants || c.participants,
            startedAt: realStartedAt || c.startedAt,
            endedAt: realEndedAt || c.endedAt,
            images: realImages || c.images,
            isEnriching: false,
          };
        }
        return c;
      });

      localVaultPool = localVaultPool.map(c => {
        if (c.id === tempId || (realId && c.id === realId)) {
          return {
            ...c,
            id: realId || c.id,
            title: realTitle || c.title,
            participants: realParticipants || c.participants,
            startedAt: realStartedAt || c.startedAt,
            endedAt: realEndedAt || c.endedAt,
            images: realImages || c.images,
            isEnriching: false,
          };
        }
        return c;
      });

      if (searchSession.isActive) finishCaptureSearch();
    } catch (err) {
      console.error("handleCaptureChatLog error:", err);
      cards = cards.filter(c => c.id !== tempId);
      localVaultPool = localVaultPool.filter(c => c.id !== tempId);
      if (latestManualCardId === tempId) {
        latestManualCardId = null;
      }
      throw err;
    } finally {
      capturePending = false;
    }
  }

  function finishBuguBeacon() {
    if (buguBeaconTimer) clearTimeout(buguBeaconTimer);
    if (!activeBuguBeacon || activeBuguBeacon.status === 'fading') return;
    const sid = activeBuguBeacon.sourceCardId;
    activeBuguBeacon = { sourceCardId: sid, status: 'fading' };
    buguBeaconTimer = setTimeout(() => {
      if (activeBuguBeacon?.sourceCardId === sid) activeBuguBeacon = null;
    }, 2500);
  }

  function resetBeaconSafetyTimer(sourceCardId: string) {
    if (buguBeaconTimer) clearTimeout(buguBeaconTimer);
    buguBeaconTimer = setTimeout(() => {
      if (activeBuguBeacon?.sourceCardId === sourceCardId) finishBuguBeacon();
    }, 90000);
  }

  let buguBabelCount = 0;

  function startBuguRadar(sourceCardId: string) {
    radarBackendFinished = false;
    buguBabelCount = 0;

    // Check if mother card is visible, if not, summon it to the current screen
    const motherCard = cards.find(c => c.id === sourceCardId);
    const viewportY = Math.round($scrollExpansion * 300);

    if (motherCard && Math.abs(motherCard.y - viewportY) > 260) {
      // Mother card is out of screen. Remove it temporarily.
      let currentCards = cards.filter(c => c.id !== sourceCardId);
      const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(motherCard, false, $scrollExpansion, currentCards, memoryPool, sourceCardId, viewportY);
      if (updatedMemoryPool) memoryPool = updatedMemoryPool;
      cards = [...updatedActiveCards.filter(c => c.id !== stagedCard.id), { ...stagedCard, isVIP: true, decayDuration: Math.max(stagedCard.decayDuration, 12 * 3600 * 1000) }];
    } else {
      cards = cards.map(c => c.id === sourceCardId ? { ...c, isVIP: true, decayDuration: Math.max(c.decayDuration, 12 * 3600 * 1000) } : c);
    }

    activeBuguBeacon = { sourceCardId, status: 'scanning' };
    resetBeaconSafetyTimer(sourceCardId);
  }

  function stageOneBabelCard(c: any) {
    const newCard = toCosmicCard(c, 'babel', 'bugu', MACRO_TTL_MS);
    const substantive = (newCard.content || newCard.originalText || newCard.translatedText || '').trim();
    if (!substantive) return;

    buguBabelCount++;
    const viewportY = Math.round($scrollExpansion * 300);
    const targetY = buguBabelCount <= 2 ? viewportY : viewportY + 480;

    memoryPool = [newCard, ...memoryPool.filter(m => m.id !== newCard.id)];
    const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(newCard, true, $scrollExpansion, cards, memoryPool, activeBuguBeacon?.sourceCardId, targetY);
    if (updatedMemoryPool) memoryPool = updatedMemoryPool;

    cards = [...updatedActiveCards.filter(card => card.id !== stagedCard.id), stagedCard];
  }

  function processBabelQueue() {
    if (babelQueueTimer) return;
    if (babelQueue.length === 0) {
      if (radarBackendFinished) finishBuguBeacon();
      return;
    }
    stageOneBabelCard(babelQueue.shift());
    if (babelQueue.length === 0 && radarBackendFinished) {
      finishBuguBeacon();
      return;
    }
    babelQueueTimer = setTimeout(() => {
      babelQueueTimer = null;
      processBabelQueue();
    }, 850);
  }

  async function handleExecuteCommand(cmd: CommandDef, card: CosmicCard) {
    if (cmd.id.startsWith('bugu:')) startBuguRadar(card.id);
    await ExecuteModuleCommand(cmd.id, { cardId: card.id, card, content: card.content || card.title, id: card.id, text: card.content || card.title });
  }

  // --- Tilt, Focus, & Keydown Delegators ---
  let globalTilt = spring({ rx: 0, ry: 0 }, { stiffness: 0.04, damping: 0.25 });

  function handleMouseMove(e: MouseEvent) {
    resetActivityTimer();
    resetTideTimer();
    const mX = (e.clientX / window.innerWidth) * 2 - 1, mY = (e.clientY / window.innerHeight) * 2 - 1;
    globalTilt.set(focusedCardId === null ? { rx: mY * 15, ry: -mX * 15 } : { rx: 0, ry: 0 });
  }

  function toggleFocus(id: string) {
    resetTideTimer();
    focusedCardId = focusedCardId === id ? null : id;
    focusNavDirection = null;
    if (focusedCardId !== null) {
      globalTilt.set({ rx: 0, ry: 0 }, { hard: true });
      omniBarRef?.blurInput();
      omniBarRef?.closeCommandSelect();
      if (searchSession.isActive && !searchSession.isConfirmed) { revertSearchView(); omniBarRef?.clearText(); }
    }
  }

  function navigateFocus(direction: 'prev' | 'next') {
    if (focusedCardId === null) return;

    // Create an ordered unique list of all available cards
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

    // If target card is not in current active `cards`, add it to the stage
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

    resetTideTimer();
  }

  function handleWindowClick(e: MouseEvent) {
    resetTideTimer();
    const target = e.target as Element | null;
    if (target?.closest('.omnibar-container')) return;
    omniBarRef?.blurInput();
    omniBarRef?.closeCommandSelect();
    if (searchSession.isActive && !searchSession.isConfirmed) { revertSearchView(); omniBarRef?.clearText(); }
    if (target && !target.closest('.card') && !target.closest('.nav-button')) focusedCardId = null;
  }

  function handleWindowKeydown(e: KeyboardEvent) {
    resetActivityTimer();
    if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'r') {
      e.preventDefault();
      restoreConfirmedSearch();
      return;
    }
    if (e.key === 'Escape') {
      if (focusedCardId !== null) { focusedCardId = null; return; }
      if (omniBarRef?.handleEscape()) return;
    }
    if (focusedCardId !== null) {
      if (e.key === 'ArrowLeft') { navigateFocus('prev'); return; }
      if (e.key === 'ArrowRight') { navigateFocus('next'); return; }
    }
    if (e.key === 'Enter') omniBarRef?.handleGlobalEnter();
  }

  // --- Metabolic Loop & Lifecycle ---
  function resetActivityTimer() {
    isStageSleeping = false;
    if (activityIdleTimer) clearTimeout(activityIdleTimer);
    activityIdleTimer = setTimeout(() => {
      if (focusedCardId === null && !searchSession.isActive && !isOmnibarFocused) {
        isStageSleeping = true;
      }
    }, 3000);
  }

  function handleWindowFocus() {
    isWindowFocused = true;
    resetActivityTimer();
  }

  function handleWindowBlur() {
    isWindowFocused = false;
  }

  function handleVisibilityChange() {
    isWindowFocused = !document.hidden;
    if (isWindowFocused) resetActivityTimer();
  }

  onMount(() => {
    window.addEventListener('focus', handleWindowFocus);
    window.addEventListener('blur', handleWindowBlur);
    document.addEventListener('visibilitychange', handleVisibilityChange);
    resetActivityTimer();

    GetRegisteredCommands().then(cmds => { if (Array.isArray(cmds)) availableCommands = cmds; }).catch(console.warn);

    Promise.all([LoadLocalCards(), GetTempCards()]).then(([localCards, tempCards]) => {
      if (tempCards?.length) memoryPool = tempCards.map(c => ({ ...toCosmicCard(c, 'babel', 'bugu', MACRO_TTL_MS), isFresh: false, spawnTime: 0, decayDuration: 0, z: 50 }));
      if (localCards?.length) localVaultPool = localCards.map(c => ({ ...toCosmicCard(c, c.type || 'thought', 'local'), isFresh: false, spawnTime: 0, decayDuration: 0, z: 50 }));
      replenishStage();
    }).catch(err => console.error("loadCards error:", err));

    EventsOn("thought:captured", (data: any) => {
      const cardId = data.DocID || ('thought_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7));
      const images = data.Images || data.images;
      const isChat = data.Type === 'chat_log' || isChatLogFormat(data.Text);
      const cardTitle = data.Title || data.title;

      // Check if this card was optimistically staged or already exists on stage
      const existingCard = cards.find(c => c.id === cardId || (isChat && c.isEnriching));
      if (existingCard) {
        cards = cards.map(c => {
          if (c.id === existingCard.id || c.id === cardId) {
            return {
              ...c,
              id: cardId,
              title: cardTitle || (c.title === '正在提炼主题...' ? cardTitle || c.title : c.title),
              type: isChat ? 'chat_log' : c.type,
              content: data.Text || c.content,
              images: images || c.images,
              participants: data.Participants || data.participants || c.participants,
              startedAt: data.StartedAt || data.startedAt || c.startedAt,
              endedAt: data.EndedAt || data.endedAt || c.endedAt,
              isEnriching: false,
            };
          }
          return c;
        });

        localVaultPool = localVaultPool.map(c => {
          if (c.id === existingCard.id || c.id === cardId) {
            return {
              ...c,
              id: cardId,
              title: cardTitle || (c.title === '正在提炼主题...' ? cardTitle || c.title : c.title),
              type: isChat ? 'chat_log' : c.type,
              content: data.Text || c.content,
              images: images || c.images,
              participants: data.Participants || data.participants || c.participants,
              startedAt: data.StartedAt || data.startedAt || c.startedAt,
              endedAt: data.EndedAt || data.endedAt || c.endedAt,
              isEnriching: false,
            };
          }
          return c;
        });

        if (latestManualCardId === existingCard.id) {
          latestManualCardId = cardId;
        }
        return;
      }

      if (latestManualCardId) {
        const prevVIP = cards.find(c => c.id === latestManualCardId);
        if (prevVIP && !isProtectedCard({ ...prevVIP, isVIP: false }, cards, activeBuguBeacon?.sourceCardId)) {
          prevVIP.isVIP = false;
          prevVIP.decayDuration = BASE_STAGE_DECAY_MS;
        }
      }
      latestManualCardId = cardId;
      const baseCard = toCosmicCard({ 
        id: cardId, 
        title: cardTitle,
        type: isChat ? 'chat_log' : 'thought',
        content: data.Text, 
        images,
        participants: data.Participants || data.participants,
        startedAt: data.StartedAt || data.startedAt,
        endedAt: data.EndedAt || data.endedAt,
      }, isChat ? 'chat_log' : 'thought', 'manual');
      const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(baseCard, true, $scrollExpansion, cards, memoryPool, activeBuguBeacon?.sourceCardId);
      if (updatedMemoryPool) memoryPool = updatedMemoryPool;
      const newCard = { ...stagedCard, images: data.Images || data.images || stagedCard.images, isVIP: true, decayDuration: 12 * 3600 * 1000, z: 12 };
      localVaultPool = [newCard, ...localVaultPool.filter(c => c.id !== newCard.id)];
      cards = [...updatedActiveCards.filter(c => c.id !== newCard.id), newCard];
    });

    EventsOn("chatlog:card_spawned", (c: any) => {
      const card = toCosmicCard(c, 'chat_log', 'bugu', MACRO_TTL_MS);
      memoryPool = [card, ...memoryPool.filter(item => item.id !== card.id)];
      replenishStage();
    });

    EventsOn("bugu:card_spawned", (c: any) => {
      if (activeBuguBeacon) resetBeaconSafetyTimer(activeBuguBeacon.sourceCardId);
      babelQueue.push(c);
      processBabelQueue();
    });

    EventsOn("bugu:radar_completed", () => {
      radarBackendFinished = true;
      if (babelQueue.length === 0 && !babelQueueTimer) finishBuguBeacon();
    });

    const metabolicInterval = setInterval(() => {
      if (searchSession.isActive && !searchSession.isConfirmed) return;
      const now = Date.now();
      memoryPool = memoryPool.filter(c => !c.macroExpiresAt || c.macroExpiresAt > now);

      cards.forEach(c => {
        if (c.id === focusedCardId) return;
        if (c.isFresh && now - c.spawnTime > 8 * 60 * 1000) c.isFresh = false;
        if (!isProtectedCard(c, cards, activeBuguBeacon?.sourceCardId) && !c.isFadingOut && (now - c.spawnTime > c.decayDuration)) {
          c.isFadingOut = true;
          setTimeout(() => { cards = cards.filter(card => card.id !== c.id); }, 2500);
        }
      });
      replenishStage();
    }, 2000);



    return () => {
      clearInterval(metabolicInterval);
      if (tideTimer) clearTimeout(tideTimer);
      if (scrollIdleTimer) clearTimeout(scrollIdleTimer);
      if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
      if (babelQueueTimer) clearTimeout(babelQueueTimer);
      if (buguBeaconTimer) clearTimeout(buguBeaconTimer);
      window.removeEventListener('focus', handleWindowFocus);
      window.removeEventListener('blur', handleWindowBlur);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      if (activityIdleTimer) clearTimeout(activityIdleTimer);
    };
  });
</script>

<svelte:window on:keydown={handleWindowKeydown} on:mousemove={handleMouseMove} on:wheel={handleWheel} on:click={handleWindowClick} />

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="container" on:click={handleWindowClick}>
  <CosmicStage
    {cards} buguBeacon={activeBuguBeacon} buguSession={activeBuguSession}
    {focusNavDirection}
    globalTilt={$globalTilt} scrollExpansion={$scrollExpansion} {focusedCardId} {hoveredCardId} {isOmnibarFocused}
    isSearchActive={searchSession.isActive} {isStageSleeping} {isWindowFocused}
    {lastScrollDirection}
    onToggleFocus={toggleFocus} onHoverCard={(id) => { hoveredCardId = id; resetTideTimer(); }}
    onLeaveCard={() => hoveredCardId = null} onBgClick={handleWindowClick}
    on:navigate={(e) => navigateFocus(e.detail)}
  />

  <OmniBar
    bind:this={omniBarRef} {availableCommands} {searchSession} {focusedCardId} {cards} {localVaultPool} {capturePending}
    {isScrolling}
    bind:isOmnibarFocused onCaptureThought={handleCaptureThought} onCaptureChatLog={handleCaptureChatLog} onFreeSearch={handleFreeSearch}
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
