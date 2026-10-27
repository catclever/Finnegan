/**
 * Finnegan Cosmic Stage - Aura Search & 3D Repulsion Logic
 * Task: frontend_logic_refactor_011
 */

import type { AuraTrace, CosmicCard } from '../types/card';
export { toCosmicCard } from './stageDecay.ts';

/**
 * Generates a distinct aura color using the Color Wheel Max Angular Gap Algorithm
 * across all cards currently holding active aura traces.
 */
export function generateAuraColor(cards: CosmicCard[], memoryPool: CosmicCard[]): string {
  const activeHues: number[] = [];
  const allCards = [...cards, ...memoryPool];

  // Extract hues from all cards with active aura traces
  for (const c of allCards) {
    if (c.auraTraces && c.auraTraces.length > 0) {
      for (const t of c.auraTraces) {
        if (t.color) {
          const match = t.color.match(/hsl\(\s*([\d.]+)/i);
          if (match) {
            const h = ((parseFloat(match[1]) % 360) + 360) % 360;
            activeHues.push(h);
          }
        }
      }
    }
  }

  // If no active hues exist, pick a random angle on the 360deg color wheel
  if (activeHues.length === 0) {
    const randomHue = Math.floor(Math.random() * 360);
    return `hsl(${randomHue}, 95%, 62%)`;
  }

  // Sort unique hues ascending
  const uniqueSortedHues = Array.from(new Set(activeHues.map(h => Math.round(h)))).sort((a, b) => a - b);

  if (uniqueSortedHues.length === 1) {
    const oppHue = (uniqueSortedHues[0] + 180) % 360;
    return `hsl(${oppHue}, 95%, 62%)`;
  }

  // Find the largest angular gap between adjacent hues (including the 360 wrap-around)
  let maxGap = -1;
  let chosenHue = 0;

  for (let i = 0; i < uniqueSortedHues.length; i++) {
    const h1 = uniqueSortedHues[i];
    const h2 = (i === uniqueSortedHues.length - 1) ? (uniqueSortedHues[0] + 360) : uniqueSortedHues[i + 1];
    const gap = h2 - h1;
    if (gap > maxGap) {
      maxGap = gap;
      chosenHue = ((h1 + h2) / 2) % 360;
    }
  }

  return `hsl(${Math.round(chosenHue)}, 95%, 62%)`;
}

/**
 * Creates or updates an aura trace on a card.
 */
export function createAuraTrace(searchId: string, color: string, hitTime: number = Date.now()): AuraTrace {
  return { searchId, color, hitTime };
}

/**
 * Transforms raw backend search results into CosmicCard instances with updated aura traces.
 */
export function transformSearchResults(
  rawResults: any[],
  knownPools: CosmicCard[][],
  searchId: string,
  searchColor: string,
  now: number = Date.now(),
  macroTtlMs: number = 24 * 3600 * 1000,
  baseDecayMs: number = 3.5 * 3600 * 1000
): CosmicCard[] {
  if (!rawResults || rawResults.length === 0) return [];

  // Flatten known cards for fast lookup
  const knownMap = new Map<string, CosmicCard>();
  for (const pool of knownPools) {
    for (const card of pool) {
      if (card && card.id && !knownMap.has(card.id)) {
        knownMap.set(card.id, card);
      }
    }
  }

  return rawResults.map((r: any) => {
    const existing = knownMap.get(r.id);
    const prevTraces = existing?.auraTraces || [];
    const newTraces: AuraTrace[] = [
      ...prevTraces.filter(t => t.searchId !== searchId),
      createAuraTrace(searchId, searchColor, now)
    ];

    const rawImages = r.images || r.Images;
    const images = rawImages ? (Array.isArray(rawImages) ? rawImages : [rawImages]) : undefined;

    if (existing) {
      return {
        ...existing,
        images: existing.images || images,
        isSearchResult: true,
        auraTraces: newTraces,
        spawnTime: now,
        decayDuration: (existing.decayDuration && existing.decayDuration > 0) ? existing.decayDuration : baseDecayMs,
        isFresh: true,
        isFadingOut: false,
        macroExpiresAt: now + macroTtlMs,
        elevatedAt: now,
      };
    }

    let title = r.title;
    if (!title || title === 'Thought') {
      const rawContent = r.content || r.translatedText || r.originalText || '';
      const firstLine = rawContent.split('\n')[0].replace(/^[#\s\-*]+/, '').trim();
      title = firstLine ? (firstLine.length > 20 ? firstLine.slice(0, 20) + '...' : firstLine) : (r.type === 'babel' ? 'Babel Echo' : 'Thought');
    }

    return {
      id: r.id || ('search_' + Math.random().toString(36).slice(2)),
      type: r.type || 'thought',
      title,
      content: r.content || '',
      images,
      originalText: r.originalText || '',
      translatedText: r.translatedText || '',
      sourceUrl: r.sourceUrl || '',
      x: 0,
      y: 0,
      rotation: 0,
      z: 25,
      spawnTime: now,
      decayDuration: baseDecayMs,
      isFresh: true,
      isVIP: false,
      origin: ((r as any).origin || 'local') as 'manual' | 'bugu' | 'local',
      macroExpiresAt: now + macroTtlMs,
      isSearchResult: true,
      auraTraces: newTraces,
      isFadingOut: false,
      elevatedAt: now
    };
  });
}

export interface SearchRepulsionResult {
  foregroundCards: CosmicCard[];
  pushedExistingCards: CosmicCard[];
  rankedResults: CosmicCard[];
  stagedCards: CosmicCard[];
}

/**
 * Calculates 3D repulsion Z coordinate for an existing card:
 * z_new = z_old + 35 + N * 4.
 * If z_new > 100, card fades out into deep space.
 * Otherwise, clamps z into deep background [60, 95].
 */
export function calculate3DRepulsionZ(oldZ: number, N: number): { z: number; isFadingOut: boolean } {
  const zNew = oldZ + 35 + N * 4;
  if (zNew > 100) {
    return { z: zNew, isFadingOut: true };
  }
  const clampedZ = Math.max(60, Math.min(95, zNew));
  return { z: clampedZ, isFadingOut: false };
}

/**
 * Executes 3D Repulsion calculation:
 * - Places top K search results in foreground (z in [15, 35]).
 * - Pushes existing cards back: z_new = z_old + 35 + N * 4.
 * - Cards with z > 100 fade out into deep space.
 */
export function calculateSearchRepulsion(
  currentCards: CosmicCard[],
  searchCards: CosmicCard[],
  targetCapacity: number
): SearchRepulsionResult {
  const N = searchCards.length;
  const K = Math.min(targetCapacity, N);

  const topCards = searchCards.slice(0, K);
  const remainingRanked = searchCards.slice(K);

  const topIds = new Set(topCards.map(c => c.id));
  const foregroundCards: CosmicCard[] = [];

  // Stage top K results in foreground (z in [15, 35])
  topCards.forEach((sc, idx) => {
    const z = 15 + Math.random() * 20; // z in [15, 35]
    const angle = (idx / (K || 1)) * 2 * Math.PI + (Math.random() - 0.5) * 0.4;
    const dist = 140 + Math.random() * 200;
    const x = Math.cos(angle) * dist * 1.4;
    const y = Math.sin(angle) * dist * 0.8;
    const rotation = (Math.random() - 0.5) * 18;

    foregroundCards.push({
      ...sc,
      x,
      y,
      rotation,
      z,
      isFresh: true,
      isFadingOut: false,
      elevatedAt: sc.elevatedAt ?? Date.now()
    });
  });

  // 3D Repulsion: Push existing stage cards back: z_new = z_old + 35 + N * 4
  const pushedExistingCards: CosmicCard[] = [];
  currentCards.forEach(oldCard => {
    if (topIds.has(oldCard.id)) return; // Already staged in foreground
    const repulsion = calculate3DRepulsionZ(oldCard.z, N);
    pushedExistingCards.push({
      ...oldCard,
      z: repulsion.z,
      isFadingOut: repulsion.isFadingOut || oldCard.isFadingOut
    });
  });

  return {
    foregroundCards,
    pushedExistingCards,
    rankedResults: remainingRanked,
    stagedCards: [...foregroundCards, ...pushedExistingCards]
  };
}

/**
 * Commits unallocated search results into memory pool with macro TTL.
 */
export function commitUnallocatedSearchResults(
  rankedResults: CosmicCard[],
  stagedCards: CosmicCard[],
  currentMemoryPool: CosmicCard[],
  macroTtlMs: number = 24 * 3600 * 1000,
  now: number = Date.now()
): CosmicCard[] {
  const stagedIds = new Set(stagedCards.map(c => c.id));
  const unallocated = rankedResults.filter(r => !stagedIds.has(r.id));

  let pool = [...currentMemoryPool];
  for (const c of unallocated) {
    const committedCard: CosmicCard = {
      ...c,
      macroExpiresAt: now + macroTtlMs
    };
    pool = [committedCard, ...pool.filter(m => m.id !== c.id)];
  }
  return pool;
}
