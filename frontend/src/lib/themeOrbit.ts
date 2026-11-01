/**
 * Finnegan Cosmic Stage - Theme Thread & Orbital Layout Engine
 * Layer 2 Pure Calculation & Layout Domain Logic
 * Task: theme_carousel_mode_150
 */

import type { CosmicCard, ThemeThread } from '../types/card';

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
 * Given a focused card and all available cards, extracts all theme threads the card belongs to.
 * A card belongs to:
 * 1. Any theme thread explicitly declared in its own `card.threads`.
 * 2. Itself as a root theme if other cards point to this card via their `threads`.
 * 
 * Each ThemeThread contains `{ id, title, cards }` with cards sorted chronologically.
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

  // Map of themeId -> { title: string, cardIds: Set<string> }
  const themeMap = new Map<string, { title: string; cardIds: Set<string> }>();

  // 1. Check card's own threads
  if (Array.isArray(card.threads)) {
    for (const th of card.threads) {
      if (!th || !th.id) continue;
      const themeId = th.id;
      const themeTitle = th.title || '主题';

      if (!themeMap.has(themeId)) {
        themeMap.set(themeId, {
          title: themeTitle,
          cardIds: new Set([card.id])
        });
      } else {
        themeMap.get(themeId)!.cardIds.add(card.id);
      }
    }
  }

  // 2. Check other cards that point to this card or share this card's threads
  for (const other of fullPool) {
    if (!other || !Array.isArray(other.threads)) continue;

    for (const th of other.threads) {
      if (!th || !th.id) continue;

      // Case 2a: other card points to `card.id` as its theme root
      if (th.id === card.id) {
        const rootTitle = card.title || th.title || '主题';
        if (!themeMap.has(card.id)) {
          themeMap.set(card.id, {
            title: rootTitle,
            cardIds: new Set([card.id, other.id])
          });
        } else {
          themeMap.get(card.id)!.cardIds.add(other.id);
        }
      }

      // Case 2b: other card belongs to one of `card`'s active themes
      if (themeMap.has(th.id)) {
        themeMap.get(th.id)!.cardIds.add(other.id);
      }
    }
  }

  // 3. For any theme where themeId exists as a card in fullPool, ensure that root card is included
  for (const [themeId, entry] of themeMap.entries()) {
    if (cardPoolMap.has(themeId)) {
      entry.cardIds.add(themeId);
    }
  }

  // 4. Assemble and sort each ThemeThread chronologically
  const result: ThemeThread[] = [];

  for (const [themeId, entry] of themeMap.entries()) {
    const threadCards: CosmicCard[] = [];
    for (const cid of entry.cardIds) {
      const c = cardPoolMap.get(cid);
      if (c) threadCards.push(c);
    }

    // Sort chronologically ascending (earliest to latest)
    threadCards.sort((a, b) => {
      const timeDiff = getCardTimestamp(a) - getCardTimestamp(b);
      if (timeDiff !== 0) return timeDiff;
      return (a.id || '').localeCompare(b.id || '');
    });

    result.push({
      id: themeId,
      title: entry.title,
      cards: threadCards
    });
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
