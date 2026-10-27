import type { CosmicCard } from '../types/card';

export const TIDE_OFFSET_PX = 300;                    // Vertical pixel shift per 1.0 scrollExpansion unit

export interface StageCoordinates {
  x: number;
  y: number;
  rotation: number;
  z: number;
}

export const OMNIBAR_DEADZONE = { halfWidth: 290, halfHeight: 32 };
export const CARD_CLEARANCE = { dx: 290, dy: 270 };
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
import { MACRO_TTL_MS } from './stageDecay';

export function calculateSpatialPlacement(
  activeCards: CosmicCard[] = [],
  _scrollExpansion: number = 0,
  isNewlyCreated: boolean = false,
  candidate: CosmicCard | null = null,
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

  const isManualNew = isNewlyCreated && candidate?.origin === 'manual';
  const sampleXRange = isManualNew ? 340 : 500;

  for (const zone of screenZones) {
    let zoneBestPoint: { x: number; y: number } | null = null;
    let zoneMaxDistance = -Infinity;

    for (let i = 0; i < SAMPLES_PER_ZONE; i++) {
      const pt = {
        x: -sampleXRange + Math.random() * (sampleXRange * 2),
        y: zone.minY + Math.random() * (zone.maxY - zone.minY)
      };

      // Center-proximate placement for newly created manual cards: keep within 380px radius to avoid peripheral edge blur (> 400px)
      if (isManualNew) {
        const viewY = pt.y - zone.centerY;
        const distFromCenter = Math.sqrt(pt.x * pt.x + viewY * viewY);
        if (distFromCenter > 380) {
          continue;
        }
      }

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
  const fallbackX = isManualNew ? (Math.random() < 0.5 ? -260 : 260) : (Math.random() < 0.5 ? -380 : 380);
  const finalPoint = chosenPoint || globalBestPoint || { x: fallbackX, y: viewCenterY };
  const finalX = Math.round(finalPoint.x);
  const finalY = Math.round(finalPoint.y);
  // Natural desk-scatter rotation (±7 deg) instead of disorienting ±25 deg
  const rotation = Number(((Math.random() - 0.5) * 14).toFixed(1));

  let z = 50;
  if (candidate) {
    if ((isNewlyCreated && candidate.origin === 'manual') || candidate.isVIP) {
      // Extreme foreground: sits at the absolute closest layer
      z = Number((1.5 + Math.random() * 3.5).toFixed(1)); // z: 1.5 ~ 5.0
    } else if (candidate.isSearchResult) {
      z = Math.random() * 15;
    } else if (candidate.origin === 'bugu' || candidate.type === 'radar') {
      // Ephemeral network cards decay by TTL ratio
      const now = Date.now();
      let ratio = 1;
      if (candidate.macroExpiresAt) {
        ratio = Math.max(0, Math.min(1, (candidate.macroExpiresAt - now) / MACRO_TTL_MS));
      }
      z = 85 - (ratio * 65);
    } else {
      // Permanent user memories & local cards: Strictly mapped by temporal recency
      const now = Date.now();
      const createdAtDate = candidate.createdAt ? new Date(candidate.createdAt).getTime() : now;
      const ageMs = Math.max(0, now - createdAtDate);
      const ageDays = ageMs / (1000 * 60 * 60 * 24);

      if (ageDays > 365) {
        // 回光返照 (Resurgence Echo of memories > 1 year): ~42..52
        z = 42 + Math.random() * 10;
      } else if (ageDays <= 1) {
        z = 3 + Math.random() * 8;   // 今日 (0~24h): z 3 ~ 11 -> scale ~1.20x - 1.25x
      } else if (ageDays <= 3) {
        z = 16 + Math.random() * 10; // 近 3 天: z 16 ~ 26 -> scale ~1.02x - 1.09x
      } else if (ageDays <= 7) {
        z = 30 + Math.random() * 12; // 近 1 周: z 30 ~ 42 -> scale ~0.86x - 0.93x
      } else if (ageDays <= 30) {
        z = 48 + Math.random() * 14; // 近 1 个月 (如 11天前): z 48 ~ 62 -> scale ~0.67x - 0.73x
      } else if (ageDays <= 90) {
        z = 66 + Math.random() * 12; // 近 3 个月: z 66 ~ 78 -> scale ~0.52x - 0.58x
      } else {
        z = 80 + Math.random() * 12; // 3 个月 ~ 1 年: z 80 ~ 92 -> scale ~0.43x - 0.48x
      }
    }
  } else {
    z = isNewlyCreated
      ? Number((8 + Math.random() * 20).toFixed(1))
      : Number((15 + Math.random() * 70).toFixed(1));
  }
  z = Number(z.toFixed(1));

  return { x: finalX, y: finalY, rotation, z };
}
