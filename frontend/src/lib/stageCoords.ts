import type { CosmicCard } from '../types/card';

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
 * Supports at least 2.2 screens of scroll depth, scaling up to 5.0 when deep cards or pool cards exist.
 */
export function getMaxScrollExpansion(activeCards: CosmicCard[] = [], totalAvailable: number = 0): number {
  const solidCards = activeCards.filter(c => !c.isFadingOut && typeof c.y === 'number' && !isNaN(c.y));
  const cardBasedMax = solidCards.length === 0 ? 2.2 : Math.max(2.2, Math.min(5.0, (Math.max(...solidCards.map(c => c.y)) - 120) / TIDE_OFFSET_PX));
  const poolBasedMax = totalAvailable > 10 ? Math.min(5.0, 2.2 + (totalAvailable - 10) * 0.12) : 2.2;
  return Math.min(5.0, Math.max(cardBasedMax, poolBasedMax));
}

/**
 * Normalizes wheel deltaMode, soft-clamps impulse with asymmetric responsiveness, and accumulates onto currentTarget.
 * Returning to top (upward gesture, deltaY < 0) has higher impulse capacity and sensitivity to smoothly reach Screen 1 in 1-2 gestures.
 */
export function computeNextScrollTarget(
  currentTarget: number,
  deltaY: number,
  deltaMode: number = 0,
  activeCards: CosmicCard[] = [],
  totalAvailable: number = 0
): number {
  const normDelta = deltaMode === 1 ? deltaY * 28 : deltaMode === 2 ? deltaY * 300 : deltaY;
  const isUpward = normDelta < 0;
  const maxImpulse = isUpward ? 360 : 260;
  const impulseFactor = isUpward ? 0.0032 : 0.0022;
  const clampedImpulse = Math.max(-maxImpulse, Math.min(maxImpulse, normDelta)) * impulseFactor;
  const maxScroll = getMaxScrollExpansion(activeCards, totalAvailable);
  return Math.max(0, Math.min(maxScroll, currentTarget + clampedImpulse));
}

/**
 * Calculates optimal 3D spatial placement for a new or staged card using Mitchell's best-candidate sampling.
 * Avoids omnibar deadzone and neighboring cards across dynamic screen zones.
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
  const screenZones = explicitTargetY !== undefined
    ? [{ minY: explicitTargetY - 240, maxY: explicitTargetY + 240, centerY: explicitTargetY }]
    : getDynamicScreenZones(viewCenterY);

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
  const finalPoint = chosenPoint || globalBestPoint || { x: Math.random() < 0.5 ? -380 : 380, y: viewCenterY };
  const finalX = Math.round(finalPoint.x);
  const finalY = Math.round(finalPoint.y);
  const rotation = Number(((Math.random() - 0.5) * 50).toFixed(1));
  const z = (_origin === 'manual' && isNewlyCreated)
    ? Number((8 + Math.random() * 20).toFixed(1))
    : Number((15 + Math.random() * 70).toFixed(1));

  return { x: finalX, y: finalY, rotation, z };
}
