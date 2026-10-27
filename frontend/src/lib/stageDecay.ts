import type { CosmicCard, SearchSession } from '../types/card';
import { calculateSpatialPlacement } from './stageCoords';

// --- Cosmic Stage Configuration Constants ---
export const BASE_CAPACITY = 10;                      // Base single-screen card count (8~12)
export const MACRO_TTL_MS = 24 * 3600 * 1000;         // 24 hours macro TTL for in-memory temporary cards
export const BASE_STAGE_DECAY_MS = 3.5 * 3600 * 1000; // 3.5 hours base stage half-life
export function isProtectedCard(
  card: CosmicCard,
  activeCards: CosmicCard[],
  activeBuguSourceId?: string | null
): boolean {
  if (card.isVIP) return true;
  if (card.isSearchResult) return true;
  if (activeBuguSourceId && card.id === activeBuguSourceId) return true;
  return activeCards.some(c => !c.isFadingOut && Boolean(c.echoOf) && c.echoOf === card.id);
}

/**
 * Supports at least 2.2 screens of scroll depth, scaling up to 5.0 when deep cards or pool cards exist.
 */
export function toCosmicCard(
  c: any,
  type: string,
  origin: 'manual' | 'bugu' | 'local',
  ttl?: number
): CosmicCard {
  const rawContent = c.content || c.Content || c.translatedText || c.TranslatedText || c.translated_text || c.originalText || '';
  let title = c.title || c.Title;
  if (!title || title === 'Thought') {
    const firstLine = rawContent.split('\n')[0].replace(/^[#\s\-*]+/, '').trim();
    title = firstLine ? (firstLine.length > 20 ? firstLine.slice(0, 20) + '...' : firstLine) : (type === 'babel' ? 'Babel Echo' : 'Thought');
  }
  const rawImages = c.images || c.Images;
  const images = rawImages ? (Array.isArray(rawImages) ? rawImages : [rawImages]) : undefined;
  return {
    id: c.id || c.ID || (type + '_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7)),
    type: c.type || c.Type || type,
    title,
    content: rawContent,
    originalText: c.originalText || c.OriginalText || c.original_text || '',
    translatedText: c.translatedText || c.TranslatedText || c.translated_text || '',
    sourceUrl: c.sourceUrl || c.SourceURL || c.source_url || '',
    publishedAt: c.publishedAt || c.PublishedAt || c.published_at || '',
    authorName: c.authorName || c.AuthorName || c.author_name || '',
    authorAvatar: c.authorAvatar || c.AuthorAvatar || c.author_avatar || '',
    createdAt: c.createdAt || c.CreatedAt || c.created_at || new Date().toISOString(),
    participants: c.participants || c.Participants,
    startedAt: c.startedAt || c.StartedAt || c.started_at,
    endedAt: c.endedAt || c.EndedAt || c.ended_at,
    echoOf: c.echoOf || c.EchoOf || c.echo_of || '',
    matchedAnchor: c.matchedAnchor || c.MatchedAnchor || c.matched_anchor || '',
    echoReason: c.echoReason || c.EchoReason || c.echo_reason || '',
    images,
    keywords: c.keywords || c.Keywords,
    tags: c.tags || c.Tags || [],
    isEnriching: c.isEnriching !== undefined ? c.isEnriching : false,
    filePath: c.filePath || c.FilePath || '',
    x: 0, y: 0, rotation: 0, z: 50,
    spawnTime: Date.now(), decayDuration: BASE_STAGE_DECAY_MS, isFresh: true, isVIP: false, origin: c.origin || origin,
    macroExpiresAt: ttl ? Date.now() + ttl : undefined
  };
}

/**
 * Calculates dynamic half-life decay duration (ms) for a card.
 */
export function calculateDecayDuration(
  origin: 'manual' | 'bugu' | 'local',
  z: number,
  isVIP: boolean,
  activeCards: CosmicCard[] = [],
  memoryPool: CosmicCard[] = []
): number {
  if (isVIP) return 12 * 3600 * 1000;
  const typeBase = origin === 'local' ? 2 * 3600 * 1000 : BASE_STAGE_DECAY_MS;
  const depthFactor = 0.6 + (z / 100) * 0.9;
  const activeIds = new Set(activeCards.map(c => c.id));
  const restingCount = memoryPool.filter(mc => !activeIds.has(mc.id)).length;
  const pressureFactor = Math.max(0.4, 1 / (1 + restingCount * 0.12));
  const jitter = 0.8 + Math.random() * 0.4;
  return Math.round(typeBase * depthFactor * pressureFactor * jitter);
}

/**
 * Calculates non-overlapping spatial placement across screens using Idea A (Dynamic Spatial Density).
 * Probes Screen 1 first; if all candidate points have safety distance < COMFORT_NORM_DIST (1.15),
 * elastically cascades to Screen 2, then Screen 3, completely eliminating hard-coded card count limits.
 * The central deadzone prevents card centers from being positioned underneath the Omnibar on all screens.
 */
export interface StageResult {
  stagedCard: CosmicCard;
  updatedActiveCards: CosmicCard[];
  updatedMemoryPool?: CosmicCard[];
}

export function stageCandidate(
  candidate: CosmicCard,
  isNewlyCreated: boolean = false,
  scrollExpansion: number = 0,
  activeCards: CosmicCard[] = [],
  memoryPool: CosmicCard[] = [],
  activeBuguSourceId?: string | null,
  explicitTargetY?: number
): StageResult {
  let baseTargetY = explicitTargetY !== undefined ? explicitTargetY : Math.round(scrollExpansion * 300);

  let currentActiveCards = [...activeCards];
  let updatedMemoryPool = [...memoryPool];

  let placementTargetY = baseTargetY;
  if (candidate.isSearchResult) {
    while (true) {
      const screenSearchCards = currentActiveCards.filter(c =>
        !c.isFadingOut && c.isSearchResult && c.y >= placementTargetY - 240 && c.y <= placementTargetY + 240
      );
      if (screenSearchCards.length >= 5) {
        placementTargetY += 480;
      } else {
        break;
      }
    }
  }

  function applyCascade(cascadeTargetY: number) {
    const screenSolidCards = currentActiveCards.filter(c =>
      !c.isFadingOut && c.y >= cascadeTargetY - 240 && c.y <= cascadeTargetY + 240
    );

    if (screenSolidCards.length >= 6) {
      const evictable = screenSolidCards
        .filter(c => !isProtectedCard(c, currentActiveCards, activeBuguSourceId))
        .sort((a, b) => b.z - a.z);

      if (evictable.length > 0) {
        const victim = evictable[0];

        if (victim.z >= 85) {
          currentActiveCards = currentActiveCards.map(c =>
             c.id === victim.id ? { ...c, isFadingOut: true } : c
          );
          updatedMemoryPool = [{ ...victim, isFadingOut: false, isVIP: false, macroExpiresAt: Date.now() + MACRO_TTL_MS }, ...updatedMemoryPool.filter(c => c.id !== victim.id)];
        } else {
          const nextTargetY = cascadeTargetY + 480;

          const newCoords = calculateSpatialPlacement(currentActiveCards.filter(c => c.id !== victim.id), scrollExpansion, false, victim, nextTargetY);

          currentActiveCards = currentActiveCards.map(c =>
            c.id === victim.id ? { ...c, x: newCoords.x, y: newCoords.y, z: newCoords.z } : c
          );

          applyCascade(nextTargetY);
        }
      }
    }
  }

  applyCascade(placementTargetY);

  const coords = calculateSpatialPlacement(currentActiveCards, scrollExpansion, isNewlyCreated, candidate, placementTargetY);
  const z = coords.z;
  const spawnTime = Date.now();
  const decayDuration = calculateDecayDuration(candidate.origin, z, candidate.isVIP, currentActiveCards, memoryPool);

  const stagedCard = {
    ...candidate,
    x: coords.x, y: coords.y, rotation: coords.rotation, z,
    spawnTime, decayDuration, isFresh: isNewlyCreated, isFadingOut: false
  };

  currentActiveCards.push(stagedCard);

  const activeSolidCards = currentActiveCards.filter(c => !c.isFadingOut);
  const maxSafeActive = Math.max(36, Math.round(BASE_CAPACITY * (1 + scrollExpansion * 0.6)) + 4);
  if (activeSolidCards.length > maxSafeActive) {
     const toEvict = activeSolidCards
        .filter(c => !isProtectedCard(c, currentActiveCards, activeBuguSourceId))
        .sort((a, b) => (a.z - b.z) || (a.spawnTime - b.spawnTime));

     if (toEvict.length > 0) {
        const victim = toEvict[0];
        currentActiveCards = currentActiveCards.map(c =>
           c.id === victim.id ? { ...c, isFadingOut: true } : c
        );
        updatedMemoryPool = [{ ...victim, isFadingOut: false, isVIP: false, macroExpiresAt: Date.now() + MACRO_TTL_MS }, ...updatedMemoryPool.filter(c => c.id !== victim.id)];
     }
  }

  return { stagedCard, updatedActiveCards: currentActiveCards, updatedMemoryPool };
}

/**
 * Ensures a card is staged and present in the user's active viewport (current screen):
 * 1. If card is already on stage within the viewport comfort zone (|c.y - viewportY| <= 260),
 *    illuminates/elevates it (VIP protection) without displacing its coordinates.
 * 2. If card is on another screen (|c.y - viewportY| > 260) or not on stage at all (vault/memory),
 *    summons it directly to the active viewport (explicitTargetY = viewportY), running Mitchell
 *    best-candidate spatial placement and triggering downward domino cascade if screen is full.
 */
export function ensureCardStagedInViewport(
  card: CosmicCard,
  scrollExpansion: number,
  cards: CosmicCard[],
  memoryPool: CosmicCard[],
  options?: { isVIP?: boolean; decayDuration?: number }
): { updatedCards: CosmicCard[]; updatedMemoryPool: CosmicCard[]; stagedCard: CosmicCard } {
  const viewportY = Math.round(scrollExpansion * 300);
  const existingCard = cards.find(c => c.id === card.id);

  let nextCards = [...cards];
  let nextMemoryPool = [...memoryPool];

  const isVIP = options?.isVIP ?? true;
  const decayDuration = options?.decayDuration ?? (12 * 3600 * 1000);

  // Case 1: Already on stage and within current viewport comfort zone
  if (existingCard && Math.abs(existingCard.y - viewportY) <= 260) {
    nextCards = nextCards.map(c => c.id === card.id
      ? { ...c, isVIP: isVIP || c.isVIP, decayDuration: Math.max(c.decayDuration || 0, decayDuration) }
      : c
    );
    return {
      updatedCards: nextCards,
      updatedMemoryPool: nextMemoryPool,
      stagedCard: nextCards.find(c => c.id === card.id)!
    };
  }

  // Case 2: In other screens or offline in vault/memory -> summon into active viewport
  let currentCards = nextCards.filter(c => c.id !== card.id);
  const { stagedCard, updatedActiveCards, updatedMemoryPool: newMemPool } = stageCandidate(
    card,
    false,
    scrollExpansion,
    currentCards,
    nextMemoryPool,
    undefined,
    viewportY
  );

  if (newMemPool) nextMemoryPool = newMemPool;
  const finalStagedCard = {
    ...stagedCard,
    isVIP: isVIP || stagedCard.isVIP,
    decayDuration: Math.max(stagedCard.decayDuration || 0, decayDuration)
  };

  nextCards = [...updatedActiveCards.filter(c => c.id !== finalStagedCard.id), finalStagedCard];

  return {
    updatedCards: nextCards,
    updatedMemoryPool: nextMemoryPool,
    stagedCard: finalStagedCard
  };
}

export function getUnstagedMemoryCards(
  memoryPool: CosmicCard[],
  activeCards: CosmicCard[],
  now: number = Date.now()
): CosmicCard[] {
  const activeIds = new Set(activeCards.map(c => c.id));
  return memoryPool.filter(c => c.type !== 'note' && !activeIds.has(c.id) && (!c.macroExpiresAt || c.macroExpiresAt > now));
}

export interface FetchCardResult {
  card: CosmicCard | null;
  updatedMemoryPool: CosmicCard[];
}

export function fetchNewCardIntoMemory(
  localVaultPool: CosmicCard[],
  memoryPool: CosmicCard[],
  activeCards: CosmicCard[],
  stagingIds: Set<string> = new Set(),
  macroTtlMs: number = MACRO_TTL_MS,
  now: number = Date.now()
): FetchCardResult {
  const memoryIds = new Set(memoryPool.map(c => c.id));
  const activeIds = new Set([...activeCards.map(c => c.id), ...stagingIds]);
  const candidate =
    localVaultPool.find(c => c.type !== 'note' && !memoryIds.has(c.id) && !activeIds.has(c.id)) ||
    localVaultPool.find(c => c.type !== 'note' && !activeIds.has(c.id));

  if (candidate) {
    const freshMemoryCard: CosmicCard = {
      ...candidate,
      macroExpiresAt: now + macroTtlMs,
      isFresh: true, spawnTime: 0, decayDuration: 0, isVIP: false,
      origin: candidate.origin || 'local'
    };
    const updatedMemoryPool = [freshMemoryCard, ...memoryPool.filter(m => m.id !== freshMemoryCard.id)];
    return { card: freshMemoryCard, updatedMemoryPool };
  }
  return { card: null, updatedMemoryPool: memoryPool };
}

export interface ReplenishParams {
  targetCapacity: number;
  cards: CosmicCard[];
  memoryPool: CosmicCard[];
  localVaultPool: CosmicCard[];
  searchSession?: SearchSession;
  scrollExpansion?: number;
}

export interface ReplenishResult {
  newCardsToStage: CosmicCard[];
  updatedMemoryPool: CosmicCard[];
  remainingRankedResults?: CosmicCard[];
  updatedCards?: CosmicCard[];
}

export function replenishStageCandidates({
  targetCapacity, cards, memoryPool, localVaultPool, searchSession, scrollExpansion = 0
}: ReplenishParams): ReplenishResult {
  const activeCount = cards.filter(c => !c.isFadingOut).length;
  const needed = targetCapacity - activeCount;
  if (needed <= 0) {
    return { newCardsToStage: [], updatedMemoryPool: memoryPool, remainingRankedResults: searchSession?.rankedResults };
  }

  const newCardsToStage: CosmicCard[] = [];
  const stagingIds = new Set<string>();
  let currentMemory = [...memoryPool];
  let currentCards = [...cards];
  let remainingRanked = searchSession?.rankedResults ? [...searchSession.rankedResults] : undefined;

  if (searchSession?.isActive && remainingRanked && remainingRanked.length > 0) {
    const takeCount = Math.min(needed, remainingRanked.length);
    const auraBatch = remainingRanked.slice(0, takeCount);
    remainingRanked = remainingRanked.slice(takeCount);

    for (const ac of auraBatch) {
      if (stagingIds.has(ac.id) || currentCards.some(c => c.id === ac.id)) continue;
      const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(ac, true, scrollExpansion, currentCards, currentMemory);
      newCardsToStage.push(stagedCard);
      stagingIds.add(stagedCard.id);
      currentCards = updatedActiveCards;
      if (updatedMemoryPool) currentMemory = updatedMemoryPool;
    }
  }

  const remainingNeeded = needed - newCardsToStage.length;
  if (remainingNeeded > 0 && !searchSession?.isActive) {
    const unstagedMemory = getUnstagedMemoryCards(currentMemory, cards).filter(c => !stagingIds.has(c.id));
    if (unstagedMemory.length > 0) {
      const memSlots = remainingNeeded === 1
        ? (Math.random() < 0.8 ? 1 : 0)
        : Math.min(unstagedMemory.length, Math.round(remainingNeeded * 0.8));
      const newSlots = remainingNeeded - memSlots;

      const shuffledMemory = [...unstagedMemory].sort(() => Math.random() - 0.5);
      for (let i = 0; i < memSlots && i < shuffledMemory.length; i++) {
        const item = shuffledMemory[i];
        if (!stagingIds.has(item.id) && !currentCards.some(c => c.id === item.id)) {
          const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(item, false, scrollExpansion, currentCards, currentMemory);
          newCardsToStage.push(stagedCard);
          stagingIds.add(stagedCard.id);
          currentCards = updatedActiveCards;
          if (updatedMemoryPool) currentMemory = updatedMemoryPool;
        }
      }

      for (let i = 0; i < newSlots; i++) {
        const { card: fetched, updatedMemoryPool: fetchMemPool } = fetchNewCardIntoMemory(localVaultPool, currentMemory, cards, stagingIds);
        currentMemory = fetchMemPool;
        if (fetched) {
          const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(fetched, true, scrollExpansion, currentCards, currentMemory);
          newCardsToStage.push(stagedCard);
          stagingIds.add(stagedCard.id);
          currentCards = updatedActiveCards;
          if (updatedMemoryPool) currentMemory = updatedMemoryPool;
        } else {
          const backup = shuffledMemory.find(m => !stagingIds.has(m.id) && !currentCards.some(c => c.id === m.id));
          if (backup) {
            const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(backup, false, scrollExpansion, currentCards, currentMemory);
            newCardsToStage.push(stagedCard);
            stagingIds.add(stagedCard.id);
            currentCards = updatedActiveCards;
            if (updatedMemoryPool) currentMemory = updatedMemoryPool;
          }
        }
      }
    } else {
      for (let i = 0; i < remainingNeeded; i++) {
        const { card: fetched, updatedMemoryPool: fetchMemPool } = fetchNewCardIntoMemory(localVaultPool, currentMemory, cards, stagingIds);
        currentMemory = fetchMemPool;
        if (fetched) {
          const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(fetched, true, scrollExpansion, currentCards, currentMemory);
          newCardsToStage.push(stagedCard);
          stagingIds.add(stagedCard.id);
          currentCards = updatedActiveCards;
          if (updatedMemoryPool) currentMemory = updatedMemoryPool;
        } else {
          break;
        }
      }
    }
  }

  return { newCardsToStage, updatedMemoryPool: currentMemory, remainingRankedResults: remainingRanked, updatedCards: currentCards };
}
