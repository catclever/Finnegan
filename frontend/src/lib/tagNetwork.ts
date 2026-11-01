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

import type { CosmicCard, AuraTrace } from '../types/card';
import { stageCandidate, MACRO_TTL_MS } from './stageDecay';

/**
 * Normalizes a tag string by stripping leading hashtags, trailing asterisks, and trimming whitespace.
 */
export function normalizeTag(tag: string): string {
  if (!tag || typeof tag !== 'string') return '';
  return tag.trim().replace(/^#+/, '').replace(/\*+$/, '').trim();
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
 * Backwards compatible alias: defaults to manual tag check.
 */
export const cardHasTag = cardHasManualTag;

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
 * Queries memoryPool and localVaultPool for cards that have the keyword/tag (or any of the tags)
 * and are NOT currently present on the active stage.
 * Supports single tag string or multi-tag Iterable (Union / OR relationship).
 */
export function findUnstagedMatchingCards(
  tagOrTags: string | Iterable<string>,
  stageCards: CosmicCard[],
  memoryPool: CosmicCard[],
  localVaultPool: CosmicCard[]
): CosmicCard[] {
  const tags = typeof tagOrTags === 'string' ? [tagOrTags] : Array.from(tagOrTags);
  const stageIdSet = new Set(stageCards.map(c => c.id));
  const seenIds = new Set<string>();
  const matches: CosmicCard[] = [];

  for (const c of memoryPool) {
    if (!stageIdSet.has(c.id) && !seenIds.has(c.id) && tags.some(t => cardHasKeyword(c, t) || cardHasTag(c, t))) {
      matches.push(c);
      seenIds.add(c.id);
    }
  }

  for (const c of localVaultPool) {
    if (!stageIdSet.has(c.id) && !seenIds.has(c.id) && tags.some(t => cardHasKeyword(c, t) || cardHasTag(c, t))) {
      matches.push(c);
      seenIds.add(c.id);
    }
  }

  return matches;
}

/**
 * Determines whether a card matches ANY of the provided query tags or keywords.
 * Union (OR) relationship.
 */
export function cardMatchesAnyTag(card: CosmicCard | any, queryTags: Iterable<string>): boolean {
  if (!card) return false;
  for (const tag of queryTags) {
    if (cardHasTag(card, tag) || cardHasKeyword(card, tag)) {
      return true;
    }
  }
  return false;
}

/**
 * Returns all tags from queryTags that the card matches.
 */
export function getCardMatchedTags(card: CosmicCard | any, queryTags: Iterable<string>): string[] {
  if (!card) return [];
  const matched: string[] = [];
  for (const tag of queryTags) {
    if (cardHasTag(card, tag) || cardHasKeyword(card, tag)) {
      matched.push(tag);
    }
  }
  return matched;
}

/**
 * Preset vibrant palette maximizing hue distance on the 360° color wheel.
 * e.g. hsl(199, 90%, 60%), hsl(280, 90%, 62%), hsl(150, 90%, 50%), hsl(38, 95%, 55%), hsl(340, 90%, 60%)
 */
export const VIBRANT_TAG_PALETTE = [
  'hsl(199, 90%, 60%)', // Ice Blue / Cyan (~199 deg)
  'hsl(280, 90%, 62%)', // Vivid Purple (~280 deg)
  'hsl(150, 90%, 50%)', // Emerald Green (~150 deg)
  'hsl(38, 95%, 55%)',  // Amber / Golden Orange (~38 deg)
  'hsl(340, 90%, 60%)', // Rose / Magenta (~340 deg)
];

/**
 * Picks a distinct vibrant color for a new tag:
 * 1. Checks if any color in VIBRANT_TAG_PALETTE is not yet in existingColors.
 * 2. If all preset colors are in use, calculates the maximum angular gap between existing
 *    hues on the 360° color wheel and picks the midpoint of that gap.
 */
export function getDistinctTagColor(existingColors: Iterable<string> | string[] = []): string {
  const used = Array.from(existingColors).map(c => c.trim().toLowerCase());
  const usedSet = new Set(used);

  for (const preset of VIBRANT_TAG_PALETTE) {
    if (!usedSet.has(preset.toLowerCase())) {
      return preset;
    }
  }

  // Parse hues from existing colors
  const hues: number[] = [];
  for (const color of used) {
    const match = color.match(/hsl\(\s*([\d.]+)/i);
    if (match) {
      const hue = parseFloat(match[1]);
      if (!isNaN(hue)) {
        hues.push(((hue % 360) + 360) % 360);
      }
    }
  }

  if (hues.length === 0) {
    return VIBRANT_TAG_PALETTE[0];
  }

  const sortedHues = Array.from(new Set(hues)).sort((a, b) => a - b);
  if (sortedHues.length === 1) {
    const nextHue = (sortedHues[0] + 180) % 360;
    return `hsl(${Math.round(nextHue)}, 90%, 60%)`;
  }

  let maxGap = -1;
  let maxStartHue = sortedHues[0];

  for (let i = 0; i < sortedHues.length; i++) {
    const h1 = sortedHues[i];
    const h2 = sortedHues[(i + 1) % sortedHues.length];
    const gap = (h2 - h1 + 360) % 360;
    if (gap > maxGap) {
      maxGap = gap;
      maxStartHue = h1;
    }
  }

  const newHue = (maxStartHue + maxGap / 2) % 360;
  return `hsl(${Math.round(newHue)}, 90%, 60%)`;
}

/**
 * Alias for getDistinctTagColor.
 */
export const assignTagColor = getDistinctTagColor;

export interface TagConstellationParams {
  rawTag: string;
  activeConstellationTags: Map<string, string>;
  cards: CosmicCard[];
  memoryPool: CosmicCard[];
  localVaultPool: CosmicCard[];
  scrollExpansion: number;
  sourceCardId?: string | null;
  macroTtlMs?: number;
  now?: number;
}

export interface TagConstellationResult {
  activeConstellationTags: Map<string, string>;
  cards: CosmicCard[];
  memoryPool: CosmicCard[];
  shouldClearFocus: boolean;
  didActivate: boolean;
}

/**
 * Activates a constellation tag, elevates matching stage cards to foreground depth (z: 10),
 * and stages unstaged matching cards from memoryPool and localVaultPool near the viewport center.
 */
export function activateTagConstellation(params: TagConstellationParams): TagConstellationResult {
  const {
    rawTag,
    activeConstellationTags,
    cards,
    memoryPool,
    localVaultPool,
    scrollExpansion,
    sourceCardId,
    macroTtlMs = MACRO_TTL_MS,
    now = Date.now(),
  } = params;

  const tag = normalizeTag(rawTag);
  if (!tag) {
    return {
      activeConstellationTags,
      cards,
      memoryPool,
      shouldClearFocus: false,
      didActivate: false,
    };
  }

  const nextActiveTags = new Map(activeConstellationTags);
  const tagColor = getDistinctTagColor(nextActiveTags.values());
  nextActiveTags.set(tag, tagColor);

  // Elevate cards already on stage that possess this tag or keyword:
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
    return {
      activeConstellationTags: nextActiveTags,
      cards: currentCards,
      memoryPool,
      shouldClearFocus: true,
      didActivate: true,
    };
  }

  let nextMemoryPool = [...memoryPool];
  const viewportCenterY = Math.round(scrollExpansion * 300);

  for (const candidate of unstaged) {
    const res = stageCandidate(
      candidate,
      false,
      scrollExpansion,
      currentCards,
      nextMemoryPool,
      sourceCardId || null,
      viewportCenterY
    );

    currentCards = res.updatedActiveCards;
    nextMemoryPool = res.updatedMemoryPool;

    const tracesToAdd: AuraTrace[] = [];
    for (const [activeTag, color] of nextActiveTags.entries()) {
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

    if (!nextMemoryPool.some(m => m.id === candidate.id)) {
      nextMemoryPool = [
        {
          ...candidate,
          macroExpiresAt: now + macroTtlMs,
        },
        ...nextMemoryPool,
      ];
    }
  }

  return {
    activeConstellationTags: nextActiveTags,
    cards: currentCards,
    memoryPool: nextMemoryPool,
    shouldClearFocus: true,
    didActivate: true,
  };
}

/**
 * Toggles a constellation tag on or off.
 * If already active, toggles off and clears its resurgence neon traces.
 * If inactive, activates it and recalls unstaged candidates.
 */
export function toggleTagConstellation(params: TagConstellationParams): TagConstellationResult {
  const { rawTag, activeConstellationTags, cards, memoryPool } = params;
  const tag = normalizeTag(rawTag);
  if (!tag) {
    return {
      activeConstellationTags,
      cards,
      memoryPool,
      shouldClearFocus: false,
      didActivate: false,
    };
  }

  if (activeConstellationTags.has(tag)) {
    const nextActiveTags = new Map(activeConstellationTags);
    nextActiveTags.delete(tag);
    const targetPrefix = 'tag_resurgence_' + tag + '_';
    const updatedCards = cards.map(c => ({
      ...c,
      auraTraces: (c.auraTraces || []).filter(t => t.searchId !== 'tag_resurgence_' + tag + '_' + c.id && !t.searchId.startsWith(targetPrefix)),
    }));
    return {
      activeConstellationTags: nextActiveTags,
      cards: updatedCards,
      memoryPool,
      shouldClearFocus: false,
      didActivate: false,
    };
  }

  return activateTagConstellation(params);
}


