import type { CosmicCard, ParamSpec, CandidateCardItem } from '../types/card';

/**
 * Parses date string or number to timestamp.
 */
function parseCardDate(val: any): number {
  if (!val) return 0;
  if (typeof val === 'number') return isNaN(val) ? 0 : val;
  if (val instanceof Date) return isNaN(val.getTime()) ? 0 : val.getTime();
  if (typeof val === 'string') {
    const d = new Date(val);
    return isNaN(d.getTime()) ? 0 : d.getTime();
  }
  return 0;
}

/**
 * Returns creation timestamp of a card.
 */
export function getCardTime(c: CosmicCard | null | undefined): number {
  if (!c) return 0;
  const t = parseCardDate(c.createdAt);
  if (t > 0) return t;
  return typeof c.spawnTime === 'number' ? c.spawnTime : 0;
}

/**
 * Validates a candidate card against a command's ParamSpec requirements.
 * Replaces hardcoded module checks with clean declarative rules.
 */
export function filterCandidateCard(card: CosmicCard, spec?: ParamSpec | null): boolean {
  if (!card) return false;
  const cType = card.type || 'thought';

  // 1. Excluded types check (e.g. ['note'])
  if (spec?.excludedTypes && Array.isArray(spec.excludedTypes)) {
    if (spec.excludedTypes.includes(cType)) {
      return false;
    }
  }

  // 2. Allowed types check
  if (spec?.allowedTypes && Array.isArray(spec.allowedTypes) && spec.allowedTypes.length > 0) {
    if (!spec.allowedTypes.includes(cType)) {
      return false;
    }
  }

  // 3. Keyword extraction requirement check
  if (spec?.requireKeywords) {
    if (cType === 'note') return false;
    const hasKeywords = Boolean(
      card.keywords && 
      Object.values(card.keywords).some(arr => Array.isArray(arr) && arr.length > 0)
    );
    if (!hasKeywords) return false;
  }

  return true;
}

export interface PartitionResult {
  items: CandidateCardItem[];
  tierCounts: { current: number; other: number; vault: number };
  totalVaultAvailable: number;
}

/**
 * Partitions available cards into the 3-Tier Hierarchy:
 * - Tier 1 (current_screen): Active on stage within |c.y - viewportY| <= 260px.
 * - Tier 2 (other_screens): Active on stage with |c.y - viewportY| > 260px.
 * - Tier 3 (vault): Offline in localVaultPool / resting memory, sorted by time descending.
 */
export function partitionCardTiers(
  stageCards: CosmicCard[],
  localVaultPool: CosmicCard[],
  scrollExpansion: number,
  spec?: ParamSpec | null,
  vaultLimit: number = 20
): PartitionResult {
  const viewportY = Math.round(scrollExpansion * 300);

  // Filter stage cards
  const validStageCards = (stageCards || []).filter(c => !c.isFadingOut && filterCandidateCard(c, spec));

  const seenIds = new Set<string>();
  const seenContent = new Set<string>();

  // Tier 1: Current Screen
  const currentTier: CandidateCardItem[] = [];
  const otherTier: CandidateCardItem[] = [];

  for (const c of validStageCards) {
    const isCurrent = Math.abs(c.y - viewportY) <= 260;
    const normContent = (c.content || '').trim();
    if (c.id) seenIds.add(c.id);
    if (normContent) seenContent.add(normContent);

    if (isCurrent) {
      currentTier.push({ card: c, tier: 'current_screen' });
    } else {
      const screenIdx = Math.max(1, Math.round(c.y / 480) + 1);
      otherTier.push({ card: c, tier: 'other_screens', screenIndex: screenIdx });
    }
  }

  // Sort Tier 1 by recency (newest first)
  currentTier.sort((a, b) => getCardTime(b.card) - getCardTime(a.card));

  // Sort Tier 2 by physical distance to current viewport (closest other screen first)
  otherTier.sort((a, b) => {
    const distA = Math.abs(a.card.y - viewportY);
    const distB = Math.abs(b.card.y - viewportY);
    if (distA !== distB) return distA - distB;
    return getCardTime(b.card) - getCardTime(a.card);
  });

  // Tier 3: Vault cards
  const validVault = (localVaultPool || []).filter(c => {
    if (!filterCandidateCard(c, spec)) return false;
    const normContent = (c.content || '').trim();
    if (c.id && seenIds.has(c.id)) return false;
    if (normContent && seenContent.has(normContent)) return false;
    return true;
  });

  // Sort Tier 3 by recency
  validVault.sort((a, b) => getCardTime(b) - getCardTime(a));

  const totalVaultAvailable = validVault.length;
  const pagedVault = validVault.slice(0, vaultLimit);

  const vaultTier: CandidateCardItem[] = pagedVault.map(c => ({
    card: c,
    tier: 'vault'
  }));

  const items = [...currentTier, ...otherTier, ...vaultTier];

  return {
    items,
    tierCounts: {
      current: currentTier.length,
      other: otherTier.length,
      vault: vaultTier.length
    },
    totalVaultAvailable
  };
}

/**
 * Organizes search results into the 3-Tier Hierarchy while preserving search relevance within tiers.
 */
export function rankSearchCandidates(
  searchResults: any[],
  stageCards: CosmicCard[],
  scrollExpansion: number,
  spec?: ParamSpec | null,
  limit: number = 30
): CandidateCardItem[] {
  const viewportY = Math.round(scrollExpansion * 300);
  const stageMap = new Map<string, CosmicCard>();
  const stageContentMap = new Map<string, CosmicCard>();

  for (const sc of stageCards || []) {
    if (sc.id) stageMap.set(sc.id, sc);
    const trimmed = (sc.content || '').trim();
    if (trimmed) stageContentMap.set(trimmed, sc);
  }

  const seenIds = new Set<string>();
  const seenContent = new Set<string>();

  const currentHits: CandidateCardItem[] = [];
  const otherHits: CandidateCardItem[] = [];
  const vaultHits: CandidateCardItem[] = [];

  for (const r of searchResults || []) {
    const cardObj: CosmicCard = {
      id: r.id,
      type: r.type || 'thought',
      title: r.title || 'Card',
      content: r.content || '',
      originalText: r.originalText || '',
      translatedText: r.translatedText || '',
      sourceUrl: r.sourceUrl || '',
      createdAt: r.createdAt || r.CreatedAt,
      x: 0, y: 0, rotation: 0, z: 50,
      keywords: r.keywords || r.Keywords,
      spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false,
      origin: ((r as any).origin || 'local') as 'manual' | 'bugu' | 'local'
    };

    if (!filterCandidateCard(cardObj, spec)) continue;

    const normContent = (cardObj.content || '').trim();
    if (cardObj.id && seenIds.has(cardObj.id)) continue;
    if (normContent && seenContent.has(normContent)) continue;
    if (cardObj.id) seenIds.add(cardObj.id);
    if (normContent) seenContent.add(normContent);

    // Check if mounted on stage
    const matchedStageCard = (cardObj.id ? stageMap.get(cardObj.id) : null) ||
      (normContent ? stageContentMap.get(normContent) : null);

    if (matchedStageCard && !matchedStageCard.isFadingOut) {
      const isCurrent = Math.abs(matchedStageCard.y - viewportY) <= 260;
      if (isCurrent) {
        currentHits.push({ card: { ...cardObj, y: matchedStageCard.y }, tier: 'current_screen' });
      } else {
        const screenIdx = Math.max(1, Math.round(matchedStageCard.y / 480) + 1);
        otherHits.push({ card: { ...cardObj, y: matchedStageCard.y }, tier: 'other_screens', screenIndex: screenIdx });
      }
    } else {
      vaultHits.push({ card: cardObj, tier: 'vault' });
    }
  }

  return [...currentHits, ...otherHits, ...vaultHits].slice(0, limit);
}
