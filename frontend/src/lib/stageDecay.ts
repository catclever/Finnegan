/**
 * Finnegan Cosmic Stage - Stage Lifecycle & Metabolic Decay Logic
 * Task: frontend_logic_refactor_011
 */

import type { CosmicCard, SearchSession } from '../types/card';

// --- Cosmic Stage Configuration Constants ---
export const BASE_CAPACITY = 10;                      // Base single-screen card count (8~12)
export const MACRO_TTL_MS = 24 * 3600 * 1000;         // 24 hours macro TTL for in-memory temporary cards
export const BASE_STAGE_DECAY_MS = 3.5 * 3600 * 1000; // 3.5 hours base stage half-life

/**
 * Calculates dynamic half-life decay duration (ms) for a card based on:
 * - VIP manual thought status (12 hours protected)
 * - Origin base duration (local: ~2h, others: 3.5h)
 * - Depth factor (closer = faster decay, farther = slower decay)
 * - Queue pressure from unshown resting cards in memoryPool
 * - Random jitter (±20%)
 */
export function calculateDecayDuration(
  origin: 'manual' | 'bugu' | 'local',
  z: number,
  isVIP: boolean,
  activeCards: CosmicCard[] = [],
  memoryPool: CosmicCard[] = []
): number {
  if (origin === 'manual' && isVIP) {
    return 12 * 3600 * 1000; // VIP protected half-life
  }

  let typeBase = BASE_STAGE_DECAY_MS;
  if (origin === 'local') {
    typeBase = 2 * 3600 * 1000; // Local memories stay ~2 hours
  }

  // Depth factor (closer = faster decay, farther = slower decay)
  const depthFactor = 0.6 + (z / 100) * 0.9;

  // Queue pressure: if memory pool has unshown cards waiting
  const activeIds = new Set(activeCards.map(c => c.id));
  const restingCount = memoryPool.filter(mc => !activeIds.has(mc.id)).length;
  const pressureFactor = Math.max(0.4, 1 / (1 + restingCount * 0.12));

  // Random jitter ±20%
  const jitter = 0.8 + Math.random() * 0.4;

  return Math.round(typeBase * depthFactor * pressureFactor * jitter);
}

export interface StageCoordinates {
  x: number;
  y: number;
  rotation: number;
  z: number;
}

// Stage & Omnibar Geometry Constants
export const OMNIBAR_DEADZONE = {
  halfWidth: 340,  // Omnibar width 580px -> half is 290px + 50px clearance
  halfHeight: 110  // Omnibar height ~55px -> half is 27px + card half 95px
};

// Safe inter-card clearances (Cards are 200px wide x 190px high)
export const CARD_CLEARANCE = {
  dx: 220, // Horizontal clearance between centers
  dy: 200  // Vertical clearance between centers
};

/**
 * Calculates a balanced, non-overlapping spatial placement for a card on the cosmic stage.
 * Uses best-candidate Poisson-disc sampling across 4 natural celestial zones (Top, Bottom, Left, Right)
 * while strictly excluding the central Omnibar deadzone and avoiding collisions with active cards.
 */
export function calculateSpatialPlacement(
  activeCards: CosmicCard[] = [],
  scrollExpansion: number = 0,
  isNewlyCreated: boolean = false
): StageCoordinates {
  const solidCards = activeCards.filter(c => !c.isFadingOut);

  // Define 4 natural celestial zones orbiting around the central Omnibar deadzone
  const bottomMaxY = 270 + scrollExpansion * 600;

  // 4 Celestial Zones:
  // 0: Top Belt       (above Omnibar)
  // 1: Bottom Belt    (below Omnibar, expands downwards with scroll)
  // 2: Left Flank     (left of Omnibar)
  // 3: Right Flank    (right of Omnibar)
  const sampleFromZone = (zoneIndex: number): { x: number; y: number } => {
    switch (zoneIndex) {
      case 0: // Top Belt
        return {
          x: -520 + Math.random() * 1040,
          y: -270 + Math.random() * 140 // [-270, -130]
        };
      case 1: // Bottom Belt
        if (scrollExpansion > 0.05 && Math.random() < 0.6) {
          return {
            x: -520 + Math.random() * 1040,
            y: 190 + Math.random() * (bottomMaxY - 190)
          };
        }
        return {
          x: -520 + Math.random() * 1040,
          y: 135 + Math.random() * (bottomMaxY - 135)
        };
      case 2: // Left Flank
        return {
          x: -520 + Math.random() * 170, // [-520, -350]
          y: -130 + Math.random() * 260  // [-130, 130]
        };
      case 3: // Right Flank
      default:
        return {
          x: 350 + Math.random() * 170,  // [350, 520]
          y: -130 + Math.random() * 260  // [-130, 130]
        };
    }
  };

  // Best candidate sampling (Mitchell's algorithm)
  const SAMPLES = 36;
  let bestCandidate: { x: number; y: number } | null = null;
  let maxMinDistance = -Infinity;

  for (let i = 0; i < SAMPLES; i++) {
    let zone: number;
    if (scrollExpansion > 0.05 && Math.random() < 0.45) {
      zone = 1; // Bias towards expanded bottom
    } else {
      zone = i % 4;
    }

    const pt = sampleFromZone(zone);

    // Strict deadzone guard: never allow coordinates within Omnibar area
    if (Math.abs(pt.x) < OMNIBAR_DEADZONE.halfWidth && Math.abs(pt.y) < OMNIBAR_DEADZONE.halfHeight) {
      continue;
    }

    // Measure normalized elliptical distance against all existing solid cards
    let minCandidateDist = Infinity;
    for (const other of solidCards) {
      const dx = Math.abs(pt.x - other.x);
      const dy = Math.abs(pt.y - other.y);
      const normDist = Math.sqrt(
        Math.pow(dx / CARD_CLEARANCE.dx, 2) + 
        Math.pow(dy / CARD_CLEARANCE.dy, 2)
      );
      if (normDist < minCandidateDist) {
        minCandidateDist = normDist;
      }
    }

    if (minCandidateDist > maxMinDistance) {
      maxMinDistance = minCandidateDist;
      bestCandidate = pt;
      // Early exit if ideal spacious clearance (normDist >= 1.35) found after sufficient sampling
      if (minCandidateDist >= 1.35 && i >= 16) {
        break;
      }
    }
  }

  // Fallback if all points were rejected
  const finalX = bestCandidate ? Math.round(bestCandidate.x) : (Math.random() < 0.5 ? -390 : 390);
  const finalY = bestCandidate ? Math.round(bestCandidate.y) : (Math.random() < 0.5 ? -180 : 180);

  // Wild dynamic tilt: ±25° (artistic celestial scattering)
  const rotation = Number(((Math.random() - 0.5) * 50).toFixed(1));

  // Depth layering:
  // Fresh newly created cards (Bugu echoes, thoughts) spawn in foreground (Z: 10 ~ 26)
  // Replenished/vault background cards spawn in midground/far ground (Z: 35 ~ 85)
  const z = isNewlyCreated
    ? Number((10 + Math.random() * 16).toFixed(1))
    : Number((35 + Math.random() * 50).toFixed(1));

  return { x: finalX, y: finalY, rotation, z };
}

/**
 * Stages a candidate card into active 3D coordinates on stage, computing initial spawn parameters,
 * anti-collision spatial placement, and half-life decay duration.
 */
export function stageCandidate(
  candidate: CosmicCard,
  isNewlyCreated: boolean = false,
  scrollExpansion: number = 0,
  activeCards: CosmicCard[] = [],
  memoryPool: CosmicCard[] = []
): CosmicCard {
  const coords = calculateSpatialPlacement(activeCards, scrollExpansion, isNewlyCreated);

  // Preserve search result Z if present
  const z = (candidate.z !== undefined && candidate.isSearchResult) ? candidate.z : coords.z;
  const spawnTime = Date.now();
  const decayDuration = calculateDecayDuration(candidate.origin, z, candidate.isVIP, activeCards, memoryPool);

  return {
    ...candidate,
    x: coords.x,
    y: coords.y,
    rotation: coords.rotation,
    z,
    spawnTime,
    decayDuration,
    isFresh: isNewlyCreated,
    isFadingOut: false
  };
}

/**
 * Returns memory cards that are currently not on stage and have not exceeded macro TTL.
 */
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

/**
 * Finds an unshown card from the local vault pool, inserts it into memoryPool with macro TTL,
 * and returns both the fresh card and the updated memory pool.
 */
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

  // Find candidate from localVaultPool that is not yet in memory and not on stage
  let candidate = localVaultPool.find(c => !memoryIds.has(c.id) && !activeIds.has(c.id));

  // Fallback: any vault card not currently active on stage (or in current staging batch)
  if (!candidate) {
    candidate = localVaultPool.find(c => !activeIds.has(c.id));
  }

  if (candidate) {
    const freshMemoryCard: CosmicCard = {
      ...candidate,
      macroExpiresAt: now + macroTtlMs, // Inserted into memory with 24hr TTL
      isFresh: true,
      spawnTime: 0,
      decayDuration: 0,
      isVIP: false,
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
}

/**
 * Selects candidate cards to replenish the cosmic stage up to target capacity:
 * 1. Prioritizes unallocated search results if search session is active.
 * 2. Replenishes remaining slots using 80/20 balance:
 *    - 80% from resting in-memory pool cards.
 *    - 20% fetched fresh from local vault into memory.
 */
export function replenishStageCandidates({
  targetCapacity,
  cards,
  memoryPool,
  localVaultPool,
  searchSession,
  scrollExpansion = 0
}: ReplenishParams): ReplenishResult {
  const activeCount = cards.filter(c => !c.isFadingOut).length;
  const needed = targetCapacity - activeCount;
  if (needed <= 0) {
    return {
      newCardsToStage: [],
      updatedMemoryPool: memoryPool,
      remainingRankedResults: searchSession?.rankedResults
    };
  }

  const newCardsToStage: CosmicCard[] = [];
  const stagingIds = new Set<string>();
  let currentMemory = [...memoryPool];
  let remainingRanked = searchSession?.rankedResults ? [...searchSession.rankedResults] : undefined;

  // 1. Prioritize remaining unallocated search results if search session is active
  if (searchSession?.isActive && remainingRanked && remainingRanked.length > 0) {
    const takeCount = Math.min(needed, remainingRanked.length);
    const auraBatch = remainingRanked.slice(0, takeCount);
    remainingRanked = remainingRanked.slice(takeCount);

    for (const ac of auraBatch) {
      if (stagingIds.has(ac.id) || cards.some(c => c.id === ac.id)) continue;
      const staged = stageCandidate(ac, true, scrollExpansion, [...cards, ...newCardsToStage], currentMemory);
      staged.z = 15 + Math.random() * 25; // Search cards staged in foreground/midground
      newCardsToStage.push(staged);
      stagingIds.add(staged.id);
    }
  }

  const remainingNeeded = needed - newCardsToStage.length;
  if (remainingNeeded > 0) {
    const unstagedMemory = getUnstagedMemoryCards(currentMemory, cards).filter(c => !stagingIds.has(c.id));
    if (unstagedMemory.length > 0) {
      let memSlots: number;
      if (remainingNeeded === 1) {
        memSlots = Math.random() < 0.8 ? 1 : 0;
      } else {
        memSlots = Math.min(unstagedMemory.length, Math.round(remainingNeeded * 0.8));
      }
      const newSlots = remainingNeeded - memSlots;

      // Take 80% from resting memory cards
      const shuffledMemory = [...unstagedMemory].sort(() => Math.random() - 0.5);
      for (let i = 0; i < memSlots && i < shuffledMemory.length; i++) {
        const item = shuffledMemory[i];
        if (!stagingIds.has(item.id) && !cards.some(c => c.id === item.id)) {
          const staged = stageCandidate(item, false, scrollExpansion, [...cards, ...newCardsToStage], currentMemory);
          newCardsToStage.push(staged);
          stagingIds.add(staged.id);
        }
      }

      // Fetch remaining 20% new cards into memory and display
      for (let i = 0; i < newSlots; i++) {
        const { card: fetched, updatedMemoryPool } = fetchNewCardIntoMemory(
          localVaultPool,
          currentMemory,
          cards,
          stagingIds
        );
        currentMemory = updatedMemoryPool;
        if (fetched) {
          const staged = stageCandidate(fetched, true, scrollExpansion, [...cards, ...newCardsToStage], currentMemory);
          newCardsToStage.push(staged);
          stagingIds.add(staged.id);
        } else {
          const backup = shuffledMemory.find(m => !stagingIds.has(m.id) && !cards.some(c => c.id === m.id));
          if (backup) {
            const staged = stageCandidate(backup, false, scrollExpansion, [...cards, ...newCardsToStage], currentMemory);
            newCardsToStage.push(staged);
            stagingIds.add(staged.id);
          }
        }
      }
    } else {
      // Fallback: fetch available cards from vault into memory (never duplicate)
      for (let i = 0; i < remainingNeeded; i++) {
        const { card: fetched, updatedMemoryPool } = fetchNewCardIntoMemory(
          localVaultPool,
          currentMemory,
          cards,
          stagingIds
        );
        currentMemory = updatedMemoryPool;
        if (fetched) {
          const staged = stageCandidate(fetched, true, scrollExpansion, [...cards, ...newCardsToStage], currentMemory);
          newCardsToStage.push(staged);
          stagingIds.add(staged.id);
        } else {
          // No more unique cards available! Stop staging to prevent duplicates
          break;
        }
      }
    }
  }

  return {
    newCardsToStage,
    updatedMemoryPool: currentMemory,
    remainingRankedResults: remainingRanked
  };
}
