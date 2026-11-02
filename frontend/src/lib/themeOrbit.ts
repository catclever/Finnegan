/**
 * Finnegan Cosmic Stage - Theme Thread & Orbital Layout Engine
 * Layer 2 Pure Calculation & Layout Domain Logic
 * Task: theme_carousel_mode_150
 */

import type { CosmicCard, ThemeThread, ThreadRef } from '../types/card';
export type { ThemeThread, ThreadRef } from '../types/card';

export interface ThemeColorPalette {
  name: string;
  primary: string;
  glow: string;
  bgGradient: string;
  bgSolid: string;
  border: string;
  text: string;
}

export const THEME_PALETTES: ThemeColorPalette[] = [
  {
    name: 'rose',
    primary: '#f43f5e',
    glow: 'rgba(244, 63, 94, 0.45)',
    bgGradient: 'linear-gradient(135deg, rgba(244, 63, 94, 0.22), rgba(251, 113, 133, 0.14))',
    bgSolid: 'rgba(244, 63, 94, 0.88)',
    border: 'rgba(244, 63, 94, 0.55)',
    text: '#e11d48'
  },
  {
    name: 'amber',
    primary: '#f59e0b',
    glow: 'rgba(245, 158, 11, 0.45)',
    bgGradient: 'linear-gradient(135deg, rgba(245, 158, 11, 0.22), rgba(251, 191, 36, 0.14))',
    bgSolid: 'rgba(245, 158, 11, 0.88)',
    border: 'rgba(245, 158, 11, 0.55)',
    text: '#d97706'
  },
  {
    name: 'emerald',
    primary: '#10b981',
    glow: 'rgba(16, 185, 129, 0.45)',
    bgGradient: 'linear-gradient(135deg, rgba(16, 185, 129, 0.22), rgba(52, 211, 153, 0.14))',
    bgSolid: 'rgba(16, 185, 129, 0.88)',
    border: 'rgba(16, 185, 129, 0.55)',
    text: '#059669'
  },
  {
    name: 'sky',
    primary: '#0ea5e9',
    glow: 'rgba(14, 165, 233, 0.45)',
    bgGradient: 'linear-gradient(135deg, rgba(14, 165, 233, 0.22), rgba(56, 189, 248, 0.14))',
    bgSolid: 'rgba(14, 165, 233, 0.88)',
    border: 'rgba(14, 165, 233, 0.55)',
    text: '#0284c7'
  },
  {
    name: 'violet',
    primary: '#8b5cf6',
    glow: 'rgba(139, 92, 246, 0.45)',
    bgGradient: 'linear-gradient(135deg, rgba(139, 92, 246, 0.22), rgba(167, 139, 250, 0.14))',
    bgSolid: 'rgba(139, 92, 246, 0.88)',
    border: 'rgba(139, 92, 246, 0.55)',
    text: '#7c3aed'
  },
  {
    name: 'fuchsia',
    primary: '#d946ef',
    glow: 'rgba(217, 70, 239, 0.45)',
    bgGradient: 'linear-gradient(135deg, rgba(217, 70, 239, 0.22), rgba(232, 121, 249, 0.14))',
    bgSolid: 'rgba(217, 70, 239, 0.88)',
    border: 'rgba(217, 70, 239, 0.55)',
    text: '#c026d3'
  },
  {
    name: 'orange',
    primary: '#f97316',
    glow: 'rgba(249, 115, 22, 0.45)',
    bgGradient: 'linear-gradient(135deg, rgba(249, 115, 22, 0.22), rgba(251, 146, 60, 0.14))',
    bgSolid: 'rgba(249, 115, 22, 0.88)',
    border: 'rgba(249, 115, 22, 0.55)',
    text: '#ea580c'
  }
];

export function getThemePalette(themeId: string, index?: number): ThemeColorPalette {
  if (typeof index === 'number' && index >= 0) {
    return THEME_PALETTES[index % THEME_PALETTES.length];
  }
  let hash = 0;
  for (let i = 0; i < (themeId || '').length; i++) {
    hash = (hash << 5) - hash + themeId.charCodeAt(i);
    hash |= 0;
  }
  const idx = Math.abs(hash) % THEME_PALETTES.length;
  return THEME_PALETTES[idx];
}

/**
 * Extracts a numeric creation timestamp for chronological card sorting.
 */
export function getCardTimestamp(card: CosmicCard | null | undefined): number {
  if (!card) return 0;
  if (card.createdAt) {
    const t = new Date(card.createdAt).getTime();
    if (!isNaN(t) && t > 0) return t;
  }
  if (typeof card.spawnTime === 'number' && card.spawnTime > 0) {
    return card.spawnTime;
  }
  return 0;
}

/**
 * Given a focused card and all available cards, extracts all theme threads via
 * Transitive Root Resolution (有向链式溯源归组).
 * 
 * Rules:
 * 1. For any card X, walks up c.threads recursively until terminal roots are reached.
 * 2. Every Root R defines a unified theme group containing R and all cards transitively reaching R.
 * 3. Cards in each group are chronologically sorted.
 */
export function extractCardThemes(
  card: CosmicCard | null | undefined,
  allCards: CosmicCard[] = []
): ThemeThread[] {
  if (!card || !card.id) return [];

  // Pool of all known cards including target card
  const cardPoolMap = new Map<string, CosmicCard>();
  for (const c of allCards || []) {
    if (c?.id) cardPoolMap.set(c.id, c);
  }
  cardPoolMap.set(card.id, card);

  const fullPool = Array.from(cardPoolMap.values());

  function getOutgoingThreads(cid: string): ThreadRef[] {
    const c = cardPoolMap.get(cid);
    if (!c || !Array.isArray(c.threads)) return [];
    const res: ThreadRef[] = [];
    for (const th of c.threads) {
      if (!th) continue;
      let targetCard = cardPoolMap.get(th.id);
      if (!targetCard && th.title) {
        targetCard = fullPool.find(p => p.title === th.title);
      }
      const targetId = targetCard ? targetCard.id : th.id;
      if (targetId && targetId !== cid) {
        res.push({
          id: targetId,
          title: targetCard?.title || th.title || '主题'
        });
      }
    }
    return res;
  }

  function findRootRefs(
    cid: string,
    visited: Set<string> = new Set()
  ): Array<{ id: string; title: string }> {
    if (visited.has(cid)) {
      // Cycle break: pick the canonical node (earliest createdAt) from visited cycle
      const cycleCards = Array.from(visited)
        .map(id => cardPoolMap.get(id))
        .filter(Boolean) as CosmicCard[];
      cycleCards.sort((a, b) => {
        const tDiff = getCardTimestamp(a) - getCardTimestamp(b);
        if (tDiff !== 0) return tDiff;
        return a.id.localeCompare(b.id);
      });
      const canonical = cycleCards[0] || cardPoolMap.get(cid);
      const rootId = canonical ? canonical.id : cid;
      const rootTitle = canonical?.title || '主题';
      return [{ id: rootId, title: rootTitle }];
    }

    const nextVisited = new Set(visited);
    nextVisited.add(cid);

    const outgoing = getOutgoingThreads(cid);
    if (outgoing.length === 0) {
      const currentCard = cardPoolMap.get(cid);
      return [{
        id: cid,
        title: currentCard?.title || '主题'
      }];
    }

    const rootsMap = new Map<string, string>();
    for (const th of outgoing) {
      if (!cardPoolMap.has(th.id)) {
        rootsMap.set(th.id, th.title || '主题');
      } else {
        const subRoots = findRootRefs(th.id, nextVisited);
        for (const sr of subRoots) {
          if (!rootsMap.has(sr.id)) {
            rootsMap.set(sr.id, sr.title);
          }
        }
      }
    }

    return Array.from(rootsMap.entries()).map(([id, title]) => ({ id, title }));
  }

  const rootToCardsMap = new Map<string, { title: string; cardIds: Set<string> }>();

  for (const c of fullPool) {
    const hasExplicitThreads = Array.isArray(c.threads) && c.threads.length > 0;
    if (!hasExplicitThreads) continue;

    const roots = findRootRefs(c.id);
    for (const root of roots) {
      if (!rootToCardsMap.has(root.id)) {
        const rootCard = cardPoolMap.get(root.id);
        const title = rootCard?.title || root.title || '主题';
        rootToCardsMap.set(root.id, {
          title,
          cardIds: new Set([c.id])
        });
      } else {
        rootToCardsMap.get(root.id)!.cardIds.add(c.id);
      }
      if (cardPoolMap.has(root.id)) {
        rootToCardsMap.get(root.id)!.cardIds.add(root.id);
      }
    }
  }

  const result: ThemeThread[] = [];
  const cardHasExplicitThreads = Array.isArray(card.threads) && card.threads.length > 0;

  for (const [rootId, entry] of rootToCardsMap.entries()) {
    if (entry.cardIds.has(card.id)) {
      if (entry.cardIds.size >= 2 || cardHasExplicitThreads) {
        const threadCards: CosmicCard[] = [];
        for (const cid of entry.cardIds) {
          const c = cardPoolMap.get(cid);
          if (c) threadCards.push(c);
        }

        threadCards.sort((a, b) => {
          const timeDiff = getCardTimestamp(a) - getCardTimestamp(b);
          if (timeDiff !== 0) return timeDiff;
          return (a.id || '').localeCompare(b.id || '');
        });

        result.push({
          id: rootId,
          title: entry.title,
          cards: threadCards
        });
      }
    }
  }

  return result;
}

export interface OrbitalButtonPosition {
  x: number;
  y: number;
  angleDeg: number;
}

/**
 * Calculates (x, y) angular offsets around the center of the nav button.
 * - Radius R ≈ 72px.
 * - Forms an inward arc pointing toward the center of the screen:
 *   - For nav-left (left side of screen): inward arc points rightward (+X).
 *   - For nav-right (right side of screen): inward arc points leftward (-X).
 */
export function computeOrbitalButtonLayout(
  totalThemes: number,
  isRightSide: boolean,
  radius: number = 72
): OrbitalButtonPosition[] {
  if (totalThemes <= 0) return [];

  if (totalThemes === 1) {
    const x = isRightSide ? -radius : radius;
    return [{ x, y: 0, angleDeg: 0 }];
  }

  // Symmetrical angular fan: up to 130 degrees total spread
  const maxSpread = Math.min(130, 36 * (totalThemes - 1));
  const startAngle = -maxSpread / 2;
  const step = maxSpread / (totalThemes - 1);

  const positions: OrbitalButtonPosition[] = [];

  for (let i = 0; i < totalThemes; i++) {
    const angleDeg = startAngle + i * step;
    const rad = (angleDeg * Math.PI) / 180;

    // Normalizing X offset direction: inward toward screen center
    const cosVal = Math.cos(rad);
    const sinVal = Math.sin(rad);

    const x = Math.round(isRightSide ? -radius * cosVal : radius * cosVal);
    const y = Math.round(radius * sinVal);

    positions.push({
      x,
      y,
      angleDeg: Math.round(angleDeg)
    });
  }

  return positions;
}

/**
 * Shared Barrage Cooldown & Non-Consecutive Selection Manager
 * Enforces a 30s cooldown per theme and prevents back-to-back duplicate bubbles.
 */
export const THEME_COOLDOWN_MS = 30_000;
const themeLastEmitTime = new Map<string, number>();
let lastEmittedThemeId: string | null = null;

export function recordThemeEmit(themeId: string, now: number = Date.now()): void {
  themeLastEmitTime.set(themeId, now);
  lastEmittedThemeId = themeId;
}

export function clearThemeEmitHistory(): void {
  themeLastEmitTime.clear();
  lastEmittedThemeId = null;
}

export function getThemeCooldownRemaining(themeId: string, now: number = Date.now()): number {
  const last = themeLastEmitTime.get(themeId) || 0;
  return Math.max(0, THEME_COOLDOWN_MS - (now - last));
}

export function findAvailableTheme(
  themes: Array<{ id: string; title: string; cards?: any[] }>,
  now: number = Date.now(),
  bypassCooldown: boolean = false
): { id: string; title: string; cards?: any[] } | null {
  if (!themes || themes.length === 0) return null;

  // 1. Filter out themes currently within 30s cooldown window (unless bypassCooldown is true)
  let available = bypassCooldown 
    ? [...themes]
    : themes.filter(t => (now - (themeLastEmitTime.get(t.id) || 0)) >= THEME_COOLDOWN_MS);

  if (available.length === 0) {
    if (bypassCooldown) {
      available = [...themes];
    } else {
      return null;
    }
  }

  // 2. If multiple available, avoid back-to-back consecutive repeats with lastEmittedThemeId
  if (available.length > 1 && lastEmittedThemeId) {
    const nonConsecutive = available.filter(t => t.id !== lastEmittedThemeId);
    if (nonConsecutive.length > 0) {
      available = nonConsecutive;
    }
  }

  // Pick randomly among available candidates
  return available[Math.floor(Math.random() * available.length)];
}
