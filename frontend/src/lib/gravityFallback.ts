/**
 * Finnegan Cosmic Stage - Universal Gravitational Fallback Engine
 * Task: gravity_fallback_elevated_cards_112
 *
 * Decouples visual optical aura traces from 3D physical stage depth.
 * When permanent cards are elevated into the foreground by full-text search,
 * tag constellation recall, or command staging:
 * 1. They are tagged with an elevatedAt timestamp and endowed with a 10-minute elevation protection window.
 * 2. When the user returns/focuses the window (handleWindowFocus), for cards whose protection period
 *    has elapsed (now - elevatedAt >= 10 * 60 * 1000):
 * 3. Their intrinsic depth (z_target) is computed based on their creation time (calculateIntrinsicTargetZ).
 * 4. The strict "only backward, not forward" rule is enforced:
 *    only if intrinsicZ > currentZ + 5 will the card smoothly sink back into deep space.
 * 5. Visual cues (auraTraces, isSearchResult, tags, etc.) are strictly preserved intact.
 * 6. UICard's animSpring automatically provides smooth physics interpolation for depth and perspective optics.
 */

import type { CosmicCard } from '../types/card';
import { calculateIntrinsicTargetZ } from './tagNetwork.ts';

/** 10 minutes gravitational elevation protection duration in milliseconds */
export const ELEVATED_PROTECTION_MS = 10 * 60 * 1000;

/**
 * Checks whether a card is a transient/temporary card (e.g. Bugu radar discovery).
 */
export function isTemporaryCard(card: CosmicCard | any): boolean {
  if (!card) return false;
  return Boolean(card.isTemporary || card.origin === 'bugu' || card.type === 'radar');
}

/**
 * Checks whether a card is a permanent / normal card (e.g. Scattering/ or Waving/ thought/note).
 */
export function isPermanentCard(card: CosmicCard | any): boolean {
  return !isTemporaryCard(card);
}

/**
 * Determines whether a card's 10-minute elevation protection window is currently active.
 */
export function isElevationProtectionActive(card: CosmicCard | any, now: number = Date.now()): boolean {
  if (!card || typeof card.elevatedAt !== 'number') return false;
  return (now - card.elevatedAt) < ELEVATED_PROTECTION_MS;
}

/**
 * Calculates the gravitational fallback depth for a permanent card:
 * Strictly enforces "only backward, not forward (intrinsicZ > z + 5)".
 */
export function calculateGravitationalFallbackZ(
  card: CosmicCard | any,
  now: number = Date.now()
): { shouldSink: boolean; targetZ: number; intrinsicZ: number } {
  const intrinsicZ = calculateIntrinsicTargetZ(card, now);
  const currentZ = typeof card?.z === 'number' ? card.z : 0;

  // Strict "only backward, not forward" rule with +5 threshold:
  // Higher Z means deeper into the background in Finnegan perspective space.
  const shouldSink = intrinsicZ > currentZ + 5;
  const targetZ = shouldSink ? intrinsicZ : currentZ;

  return { shouldSink, targetZ, intrinsicZ };
}

/**
 * Applies gravitational fallback on an individual card if protection period has lapsed.
 * Optical auras and visual cues are decoupled and strictly preserved intact.
 */
export function applyGravitationalFallback(
  card: CosmicCard,
  now: number = Date.now()
): CosmicCard {
  if (!card) return card;
  // Temporary cards follow their own metabolic half-life decay, not permanent gravitational fallback
  if (isTemporaryCard(card)) return card;

  const elevatedTime = card.elevatedAt;
  // If not elevated or still within the 10-minute protection window, maintain current state
  if (!elevatedTime || (now - elevatedTime < ELEVATED_PROTECTION_MS)) {
    return card;
  }

  const { shouldSink, targetZ } = calculateGravitationalFallbackZ(card, now);
  if (shouldSink) {
    return {
      ...card,
      z: targetZ,
    };
  }

  return card;
}

/**
 * Processes gravitational fallback on an array of active stage cards on window focus.
 */
export function processWindowFocusGravitationalFallback(
  cards: CosmicCard[],
  now: number = Date.now(),
  focusedCardId?: string | null
): CosmicCard[] {
  return cards.map(c => {
    if (!c) return c;
    if (focusedCardId && c.id === focusedCardId) return c;
    if (c.isFadingOut) return c;
    if (isTemporaryCard(c)) return c;
    return applyGravitationalFallback(c, now);
  });
}
