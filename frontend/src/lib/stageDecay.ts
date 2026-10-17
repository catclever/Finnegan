/**
 * ======================================================================================
 * Finnegan Cosmic Stage - Stage Lifecycle, Multi-Screen Overflow & Metabolic Decay Logic
 * Task: night_shift_overflow_scroll_focus_018
 *
 * [Bug Fix & Feature Documentation]
 * 1. Specific Problems Fixed:
 *    - Bugu Mother cards could previously be evicted or faded out when a rapid stream of
 *      Babel echo cards arrived or when manual VIP focus shifted.
 *    - High-volume Bugu echo cards crowded Screen 1 or triggered premature card eviction
 *      instead of overflowing into Deep Space (Screen 2 and Screen 3).
 *    - Gravitational Tide scrolling lacked multi-screen range (>1.0) and suffered from
 *      spring target desynchronization on rapid wheel impulses.
 * 2. Resolution & Logic:
 *    - Added `isProtectedCard` to protect VIP cards, the active Bugu radar source card,
 *      and any card that is the mother (`c.echoOf === card.id`) of an active stage echo.
 *    - Enhanced `calculateSpatialPlacement` to detect when Screen 1 (`y < 300`) reaches
 *      its safe capacity (`8` for Bugu echoes, `BASE_CAPACITY` otherwise) and automatically
 *      allocate overflow cards into Screen 2 (`y in [320, 650]`, up to 8 cards) and
 *      Screen 3 (`y in [650, 1000]`) using Mitchell's 36-sample best-candidate algorithm.
 *    - Added `getMaxScrollExpansion` (supporting 2.2 up to 3.5) and `computeNextScrollTarget`
 *      with deltaMode normalization and impulse soft-clamping for silky 60fps scrolling.
 *    - Consolidated `toCosmicCard` factory here to keep `App.svelte` concise (<300 lines).
 * 3. Caveats & Future Considerations:
 *    - `TIDE_OFFSET_PX` (300px per scrollExpansion unit) is the shared constant coupling
 *      `.scene` vertical translation, deep-space scroll cues, and `UICard` focus offset.
 * ======================================================================================
 */

import type { CosmicCard, SearchSession } from '../types/card';

// --- Cosmic Stage Configuration Constants ---
export const BASE_CAPACITY = 10;                      // Base single-screen card count (8~12)
export const MACRO_TTL_MS = 24 * 3600 * 1000;         // 24 hours macro TTL for in-memory temporary cards
export const BASE_STAGE_DECAY_MS = 3.5 * 3600 * 1000; // 3.5 hours base stage half-life
export const TIDE_OFFSET_PX = 300;                    // Vertical pixel shift per 1.0 scrollExpansion unit

export interface StageCoordinates {
  x: number;
  y: number;
  rotation: number;
  z: number;
}

export const OMNIBAR_DEADZONE = { halfWidth: 290, halfHeight: 32 };
export const CARD_CLEARANCE = { dx: 220, dy: 200 };
export const COMFORT_NORM_DIST = 1.15; // Minimum safe distance threshold for comfortable placement

export interface ScreenZone {
  minY: number;
  maxY: number;
  centerY: number;
}

export function getDynamicScreenZones(targetY: number): ScreenZone[] {
  return [
    { minY: targetY - 240, maxY: targetY + 240, centerY: targetY },
    { minY: targetY + 280, maxY: targetY + 720, centerY: targetY + 480 },
    { minY: targetY + 760, maxY: targetY + 1160, centerY: targetY + 960 },
    { minY: targetY + 1200, maxY: targetY + 1600, centerY: targetY + 1440 }
  ];
}

/**
 * Protects VIP cards, active Bugu radar source card, and any Mother card of an active stage echo.
 */
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
 * Supports at least 2.2 screens of scroll depth, scaling up to 3.5 when deep cards exist.
 */
export function getMaxScrollExpansion(activeCards: CosmicCard[] = []): number {
  const solidCards = activeCards.filter(c => !c.isFadingOut && typeof c.y === 'number' && !isNaN(c.y));
  if (solidCards.length === 0) return 2.2;
  const maxCardY = Math.max(...solidCards.map(c => c.y));
  if (!isFinite(maxCardY)) return 2.2;
  return Math.max(2.2, Math.min(3.5, (maxCardY - 120) / TIDE_OFFSET_PX));
}

/**
 * Normalizes wheel deltaMode, soft-clamps impulse with asymmetric responsiveness, and accumulates onto `currentTarget`.
 * Returning to top (upward gesture, deltaY < 0) has higher impulse capacity and sensitivity to smoothly reach Screen 1 in 1-2 gestures.
 */
export function computeNextScrollTarget(
  currentTarget: number,
  deltaY: number,
  deltaMode: number = 0,
  activeCards: CosmicCard[] = []
): number {
  const normDelta = deltaMode === 1 ? deltaY * 28 : deltaMode === 2 ? deltaY * 300 : deltaY;
  const isUpward = normDelta < 0;
  const maxImpulse = isUpward ? 360 : 260;
  const impulseFactor = isUpward ? 0.0032 : 0.0022;
  const clampedImpulse = Math.max(-maxImpulse, Math.min(maxImpulse, normDelta)) * impulseFactor;
  const maxScroll = getMaxScrollExpansion(activeCards);
  return Math.max(0, Math.min(maxScroll, currentTarget + clampedImpulse));
}

/**
 * Normalizes raw card payloads from backend/IPC events into a typed CosmicCard instance.
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
    images,
    isEnriching: c.isEnriching !== undefined ? c.isEnriching : false,
    x: 0, y: 0, rotation: 0, z: 50,
    spawnTime: Date.now(), decayDuration: BASE_STAGE_DECAY_MS, isFresh: true, isVIP: false, origin,
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
export function calculateSpatialPlacement(
  activeCards: CosmicCard[] = [],
  _scrollExpansion: number = 0,
  isNewlyCreated: boolean = false,
  _origin?: string,
  explicitTargetY?: number
): StageCoordinates {
  const solidCards = activeCards.filter(c => !c.isFadingOut);

  const viewCenterY = explicitTargetY !== undefined ? explicitTargetY : Math.round(_scrollExpansion * 300);
  const screenZones = getDynamicScreenZones(viewCenterY);

  const SAMPLES_PER_ZONE = 28;
  let chosenPoint: { x: number; y: number } | null = null;
  let globalBestPoint: { x: number; y: number } | null = null;
  let globalMaxDistance = -Infinity;

  for (const zone of screenZones) {
    let zoneBestPoint: { x: number; y: number } | null = null;
    let zoneMaxDistance = -Infinity;

    for (let i = 0; i < SAMPLES_PER_ZONE; i++) {
      const pt = {
        x: -500 + Math.random() * 1000,
        y: zone.minY + Math.random() * (zone.maxY - zone.minY)
      };

      // Deadzone: card center must not be positioned underneath the central Omnibar on any screen
      if (Math.abs(pt.x) < OMNIBAR_DEADZONE.halfWidth && Math.abs(pt.y - zone.centerY) < OMNIBAR_DEADZONE.halfHeight) {
        continue;
      }

      let minCandidateDist = Infinity;
      for (const other of solidCards) {
        const dx = Math.abs(pt.x - other.x);
        const dy = Math.abs(pt.y - other.y);
        const normDist = Math.sqrt(Math.pow(dx / CARD_CLEARANCE.dx, 2) + Math.pow(dy / CARD_CLEARANCE.dy, 2));
        if (normDist < minCandidateDist) minCandidateDist = normDist;
      }

      if (minCandidateDist > zoneMaxDistance) {
        zoneMaxDistance = minCandidateDist;
        zoneBestPoint = pt;
        if (minCandidateDist >= 1.45 && i >= 10) break;
      }
    }

    if (zoneBestPoint && zoneMaxDistance > globalMaxDistance) {
      globalMaxDistance = zoneMaxDistance;
      globalBestPoint = zoneBestPoint;
    }

    // Idea A: If this screen zone has a comfortable, non-overlapping location, settle here!
    if (zoneBestPoint && zoneMaxDistance >= COMFORT_NORM_DIST) {
      chosenPoint = zoneBestPoint;
      break;
    }
  }

  // Fallback to the globally best candidate if all zones were crowded
  const finalPoint = chosenPoint || globalBestPoint || { x: Math.random() < 0.5 ? -380 : 380, y: 0 };
  const finalX = Math.round(finalPoint.x);
  const finalY = Math.round(finalPoint.y);
  const rotation = Number(((Math.random() - 0.5) * 50).toFixed(1));
  const z = (_origin === 'manual' && isNewlyCreated)
    ? Number((8 + Math.random() * 20).toFixed(1))
    : Number((15 + Math.random() * 70).toFixed(1));

  return { x: finalX, y: finalY, rotation, z };
}

/**
 * Stages a candidate card into active 3D coordinates on stage.
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
  const targetY = explicitTargetY !== undefined ? explicitTargetY : Math.round(scrollExpansion * 300);

  let currentActiveCards = [...activeCards];

  function applyCascade(cascadeTargetY: number) {
    const screenSolidCards = currentActiveCards.filter(c =>
      !c.isFadingOut && c.y >= cascadeTargetY - 240 && c.y <= cascadeTargetY + 240
    );

    if (screenSolidCards.length >= 6) {
      const evictable = screenSolidCards
        .filter(c => !isProtectedCard(c, currentActiveCards, activeBuguSourceId))
        .sort((a, b) => a.spawnTime - b.spawnTime);

      if (evictable.length > 0) {
        const victim = evictable[0];
        const nextTargetY = cascadeTargetY + 480;

        const newCoords = calculateSpatialPlacement(currentActiveCards.filter(c => c.id !== victim.id), scrollExpansion, false, victim.origin, nextTargetY);

        currentActiveCards = currentActiveCards.map(c =>
          c.id === victim.id ? { ...c, x: newCoords.x, y: newCoords.y, z: newCoords.z } : c
        );

        applyCascade(nextTargetY);
      }
    }
  }

  applyCascade(targetY);

  const coords = calculateSpatialPlacement(currentActiveCards, scrollExpansion, isNewlyCreated, candidate.origin, targetY);
  const z = candidate.z !== undefined ? candidate.z : coords.z;
  const spawnTime = Date.now();
  const decayDuration = calculateDecayDuration(candidate.origin, z, candidate.isVIP, currentActiveCards, memoryPool);

  const stagedCard = {
    ...candidate,
    x: coords.x, y: coords.y, rotation: coords.rotation, z,
    spawnTime, decayDuration, isFresh: isNewlyCreated, isFadingOut: false
  };

  currentActiveCards.push(stagedCard);

  let updatedMemoryPool = [...memoryPool];

  const activeSolidCards = currentActiveCards.filter(c => !c.isFadingOut);
  if (activeSolidCards.length > 30) {
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

export function getUnstagedMemoryCards(
  memoryPool: CosmicCard[],
  activeCards: CosmicCard[],
  now: number = Date.now()
): CosmicCard[] {
  const activeIds = new Set(activeCards.map(c => c.id));
  return memoryPool.filter(c => !activeIds.has(c.id) && (!c.macroExpiresAt || c.macroExpiresAt > now));
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
    localVaultPool.find(c => !memoryIds.has(c.id) && !activeIds.has(c.id)) ||
    localVaultPool.find(c => !activeIds.has(c.id));

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
      stagedCard.z = 15 + Math.random() * 25;
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
