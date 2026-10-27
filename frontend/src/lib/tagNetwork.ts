/**
 * Finnegan Cosmic Stage - Tag Association & Resurgence Physics Engine
 * Task: tag_association_display_network_103
 *
 * Layer 2 Pure Mathematical Engine:
 * - Keyword/tag extraction & normalization (stripping manual '*' markers)
 * - Tag matching across frontmatter keywords and legacy tags
 * - Intrinsic age-determined Z-depth calculation (z_target)
 * - Gravitational sinking interpolation: z(t) = z_target - (z_target - 10) * (remainingMs / 120000)
 * - Zero DOM dependencies
 */

import type { CosmicCard } from '../types/card';

/**
 * Normalizes a tag string by stripping trailing asterisks and trimming whitespace.
 */
export function normalizeTag(tag: string): string {
  if (!tag || typeof tag !== 'string') return '';
  return tag.trim().replace(/\*+$/, '').trim();
}

/**
 * Determines whether a card possesses a manual tag (ending with '*' in keywords, or in tags).
 */
export function cardHasManualTag(card: CosmicCard | any, queryTag: string): boolean {
  if (!card || !queryTag) return false;
  const cleanQuery = normalizeTag(queryTag).toLowerCase();
  if (!cleanQuery) return false;

  // 1. Check card.tags / card.Tags
  const rawTags = card.tags || card.Tags;
  if (Array.isArray(rawTags)) {
    for (const t of rawTags) {
      if (typeof t === 'string' && normalizeTag(t).toLowerCase() === cleanQuery) {
        return true;
      }
    }
  }

  // 2. Check card.keywords / card.Keywords (Record<string, string[]>)
  // Only items with '*' provenance marker are recognized as manual tags
  const kwSource = card.keywords || card.Keywords;
  if (kwSource && typeof kwSource === 'object') {
    for (const list of Object.values(kwSource)) {
      if (Array.isArray(list)) {
        for (const item of list) {
          if (typeof item === 'string' && item.trim().endsWith('*')) {
            if (normalizeTag(item).toLowerCase() === cleanQuery) {
              return true;
            }
          }
        }
      }
    }
  }

  return false;
}

/**
 * Determines whether a card possesses a given keyword/tag in ANY form
 * (with or without '*'). Used for open vault recall and matching.
 */
export function cardHasKeyword(card: CosmicCard | any, queryTag: string): boolean {
  if (!card || !queryTag) return false;
  const cleanQuery = normalizeTag(queryTag).toLowerCase();
  if (!cleanQuery) return false;

  // 1. Check card.tags / card.Tags
  const rawTags = card.tags || card.Tags;
  if (Array.isArray(rawTags)) {
    for (const t of rawTags) {
      if (typeof t === 'string' && normalizeTag(t).toLowerCase() === cleanQuery) {
        return true;
      }
    }
  }

  // 2. Check card.keywords: open match across all 4D dimensions
  const kwSource = card.keywords || card.Keywords;
  if (kwSource && typeof kwSource === 'object') {
    for (const list of Object.values(kwSource)) {
      if (Array.isArray(list)) {
        for (const item of list) {
          if (typeof item === 'string' && normalizeTag(item).toLowerCase() === cleanQuery) {
            return true;
          }
        }
      }
    }
  }

  return false;
}

/**
 * Backwards compatible alias: defaults to open keyword check.
 */
export const cardHasTag = cardHasKeyword;

/**
 * Extracts and deduplicates only manual tags (ending with '*') from a card.
 */
export function getCardManualTags(card: CosmicCard | any): string[] {
  if (!card) return [];
  const set = new Set<string>();

  const rawTags = card.tags || card.Tags;
  if (Array.isArray(rawTags)) {
    for (const t of rawTags) {
      if (typeof t === 'string') {
        const clean = normalizeTag(t);
        if (clean) set.add(clean);
      }
    }
  }

  const kwSource = card.keywords || card.Keywords;
  if (kwSource && typeof kwSource === 'object') {
    for (const list of Object.values(kwSource)) {
      if (Array.isArray(list)) {
        for (const item of list) {
          if (typeof item === 'string' && item.trim().endsWith('*')) {
            const clean = normalizeTag(item);
            if (clean) set.add(clean);
          }
        }
      }
    }
  }

  return Array.from(set);
}

/**
 * Backwards compatible alias: returns manual tags.
 */
export const getCardTags = getCardManualTags;

/**
 * Extracts and deduplicates ALL keywords and tags from a card (with or without '*').
 */
export function getCardAllKeywords(card: CosmicCard | any): string[] {
  if (!card) return [];
  const set = new Set<string>();

  const rawTags = card.tags || card.Tags;
  if (Array.isArray(rawTags)) {
    for (const t of rawTags) {
      if (typeof t === 'string') {
        const clean = normalizeTag(t);
        if (clean) set.add(clean);
      }
    }
  }

  const kwSource = card.keywords || card.Keywords;
  if (kwSource && typeof kwSource === 'object') {
    for (const list of Object.values(kwSource)) {
      if (Array.isArray(list)) {
        for (const item of list) {
          if (typeof item === 'string') {
            const clean = normalizeTag(item);
            if (clean) set.add(clean);
          }
        }
      }
    }
  }

  return Array.from(set);
}

/**
 * Calculates a permanent card's intrinsic age-determined depth z_target based on temporal recency.
 * Modeled after stageCoords.ts calculateSpatialPlacement() rules.
 */
export function calculateIntrinsicTargetZ(card: CosmicCard | any, now: number = Date.now()): number {
  const createdAtDate = card?.createdAt ? new Date(card.createdAt).getTime() : now;
  const ageMs = Math.max(0, now - createdAtDate);
  const ageDays = ageMs / (1000 * 60 * 60 * 24);

  if (ageDays > 365) {
    // Resurgence Echo of memories > 1 year: ~42..52
    return Number((42 + Math.random() * 10).toFixed(1));
  } else if (ageDays <= 1) {
    // Today (0~24h): z 3 ~ 11
    return Number((3 + Math.random() * 8).toFixed(1));
  } else if (ageDays <= 3) {
    // Past 3 days: z 16 ~ 26
    return Number((16 + Math.random() * 10).toFixed(1));
  } else if (ageDays <= 7) {
    // Past 1 week: z 30 ~ 42
    return Number((30 + Math.random() * 12).toFixed(1));
  } else if (ageDays <= 30) {
    // Past 1 month: z 48 ~ 62
    return Number((48 + Math.random() * 14).toFixed(1));
  } else if (ageDays <= 90) {
    // Past 3 months: z 66 ~ 78
    return Number((66 + Math.random() * 12).toFixed(1));
  } else {
    // 3 months ~ 1 year: z 80 ~ 92
    return Number((80 + Math.random() * 12).toFixed(1));
  }
}

/**
 * Gravitational Sinking Formula:
 * During the 120,000ms resurgence period, interpolate effective Z depth from initial Z (10)
 * down to its intrinsic age-determined depth z_target based on elapsed time:
 * z(t) = z_target - (z_target - 10) * (remainingMs / 120000).
 */
export function calculateResurgenceZ(
  targetZ: number,
  remainingMs: number,
  durationMs: number = 120000,
  initialZ: number = 10
): number {
  const clampedRemaining = Math.max(0, Math.min(durationMs, remainingMs));
  const ratio = clampedRemaining / durationMs;
  const z = targetZ - (targetZ - initialZ) * ratio;
  return Number(z.toFixed(2));
}

/**
 * Queries memoryPool and localVaultPool for cards that have the keyword/tag
 * and are NOT currently present on the active stage.
 */
export function findUnstagedMatchingCards(
  tag: string,
  stageCards: CosmicCard[],
  memoryPool: CosmicCard[],
  localVaultPool: CosmicCard[]
): CosmicCard[] {
  const stageIdSet = new Set(stageCards.map(c => c.id));
  const seenIds = new Set<string>();
  const matches: CosmicCard[] = [];

  for (const c of memoryPool) {
    if (!stageIdSet.has(c.id) && !seenIds.has(c.id) && cardHasTag(c, tag)) {
      matches.push(c);
      seenIds.add(c.id);
    }
  }

  for (const c of localVaultPool) {
    if (!stageIdSet.has(c.id) && !seenIds.has(c.id) && cardHasTag(c, tag)) {
      matches.push(c);
      seenIds.add(c.id);
    }
  }

  return matches;
}
