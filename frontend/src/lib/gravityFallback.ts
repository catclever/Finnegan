/**
 * Finnegan Cosmic Stage - Universal Gravitational Fallback Engine
 * Task: gravity_fallback_unlock_extinction_cooldown
 *
 * Sequence of Gravitational Fallback:
 * 1. Locked State (Light is Shining):
 *    As long as any aura trace (search glow, tag neon, etc.) is active, the card is STRICTLY LOCKED in the foreground.
 * 2. Unlocked Moment (Aura Extinction):
 *    The moment the light naturally extinguishes (e.g. after search aura duration 15m, tag aura 2m),
 *    the card transitions to the Unlocked state.
 * 3. 10-Minute Cooldown Buffer after Extinction:
 *    A 10-minute quiet cooldown window begins immediately upon aura extinction.
 *    The card remains protected in the foreground during these 10 minutes.
 * 4. Window Focus Gravitational Sinking (Strictly "Only Backward, Never Forward"):
 *    Only when the light has extinguished AND the 10-minute cooldown has elapsed,
 *    upon the user returning to Finnegan (handleWindowFocus), the card's natural intrinsic depth
 *    (calculateIntrinsicTargetZ) is computed. If intrinsicZ > currentZ + 5, animSpring smoothly
 *    sinks the card into its natural historical depth in deep space!
 */

import type { CosmicCard } from '../types/card';
import { calculateIntrinsicTargetZ } from './tagNetwork.ts';

/** Default duration for search auras (15 minutes) */
export const DEFAULT_SEARCH_AURA_DURATION = 15 * 60 * 1000;

/** 10 minutes quiet cooldown buffer after aura extinction */
export const COOLDOWN_AFTER_EXTINCTION_MS = 10 * 60 * 1000;

/** Backward-compatibility alias for 10 minutes */
export const ELEVATED_PROTECTION_MS = COOLDOWN_AFTER_EXTINCTION_MS;

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
 * Checks whether any optical aura trace on the card is currently active (light is shining).
 */
export function hasActiveAura(card: CosmicCard | any, now: number = Date.now()): boolean {
  if (!card || !card.auraTraces || card.auraTraces.length === 0) return false;
  return card.auraTraces.some((t: any) => {
    const duration = typeof t.duration === 'number' ? t.duration : DEFAULT_SEARCH_AURA_DURATION;
    const hitTime = typeof t.hitTime === 'number' ? t.hitTime : 0;
    return (now - hitTime) < duration;
  });
}

/**
 * Returns the timestamp when the card's latest aura extinguished (or elevation occurred).
 * Returns null if the card was never endowed with an aura or elevation timestamp.
 */
export function getAuraExtinctionTime(card: CosmicCard | any): number | null {
  if (!card) return null;
  const traces = card.auraTraces;
  if (Array.isArray(traces) && traces.length > 0) {
    const extinctionTimes = traces.map((t: any) => {
      const duration = typeof t.duration === 'number' ? t.duration : DEFAULT_SEARCH_AURA_DURATION;
      const hitTime = typeof t.hitTime === 'number' ? t.hitTime : 0;
      return hitTime + duration;
    });
    return Math.max(...extinctionTimes);
  }
  return typeof card.elevatedAt === 'number' ? card.elevatedAt : null;
}

/**
 * Determines whether a card is currently protected from gravitational sinking.
 * A card is protected IF:
 * 1. Its aura light is still shining (hasActiveAura is true); OR
 * 2. Its aura light has extinguished, BUT the 10-minute cooldown buffer after extinction has not yet elapsed.
 */
export function isElevationProtectionActive(card: CosmicCard | any, now: number = Date.now()): boolean {
  if (!card) return false;

  // 1. As long as the light is on, the card is strictly locked in the foreground
  if (hasActiveAura(card, now)) {
    return true;
  }

  // 2. Obtain the moment the light extinguished (unlockedAt)
  const extinctionTime = getAuraExtinctionTime(card);
  if (extinctionTime === null) {
    return false;
  }

  // 3. Must endure the 10-minute cooldown buffer after extinction
  return (now - extinctionTime) < COOLDOWN_AFTER_EXTINCTION_MS;
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
 * Applies gravitational fallback on an individual card if its light has extinguished AND 10m cooldown elapsed.
 * Visual traces (if any residual) are decoupled and safely preserved.
 */
export function applyGravitationalFallback(
  card: CosmicCard,
  now: number = Date.now()
): CosmicCard {
  if (!card) return card;
  // Temporary cards follow their own metabolic half-life decay, not permanent gravitational fallback
  if (isTemporaryCard(card)) return card;

  // If light is still active or within 10-minute cooldown after extinction, keep in place
  if (isElevationProtectionActive(card, now)) {
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
