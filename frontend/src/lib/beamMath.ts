import type { CosmicCard } from '../types/card';

export interface BeamCoordinates {
  path: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  midX: number;
  midY: number;
  c1x: number;
  c1y: number;
  c2x: number;
  c2y: number;
  dist: number;
}

export const DOCKING_INSET_PX = 2;
export const CARD_WIDTH = 200;
export const CARD_HEIGHT = 190;

export function getCardScale(card: CosmicCard): number {
  const zNorm = Math.max(0, Math.min(1, (card.z || 0) / 100));
  return 1.08 - (zNorm * 0.44);
}

export function computeCubicBezierPoint(
  t: number,
  p0: number,
  p1: number,
  p2: number,
  p3: number
): number {
  const mt = 1 - t;
  return mt * mt * mt * p0 + 3 * mt * mt * t * p1 + 3 * mt * t * t * p2 + t * t * t * p3;
}

export function computeAnimatedBeamPath(
  source: CosmicCard,
  target: CosmicCard,
  dockingInset: number = DOCKING_INSET_PX
): BeamCoordinates {
  const rawDx = target.x - source.x;
  const rawDy = target.y - source.y;
  const dist = Math.hypot(rawDx, rawDy);

  if (dist < 1) {
    return {
      path: `M ${source.x} ${source.y} L ${target.x} ${target.y}`,
      x1: source.x,
      y1: source.y,
      x2: target.x,
      y2: target.y,
      midX: source.x,
      midY: source.y,
      c1x: source.x,
      c1y: source.y,
      c2x: target.x,
      c2y: target.y,
      dist,
    };
  }

  const sourceScale = getCardScale(source);
  const targetScale = getCardScale(target);

  const sourceHalfW = Math.max(16, (CARD_WIDTH * sourceScale) / 2 - dockingInset);
  const sourceHalfH = Math.max(16, (CARD_HEIGHT * sourceScale) / 2 - dockingInset);

  const targetHalfW = Math.max(16, (CARD_WIDTH * targetScale) / 2 - dockingInset);
  const targetHalfH = Math.max(16, (CARD_HEIGHT * targetScale) / 2 - dockingInset);

  let startX: number, startY: number, endX: number, endY: number;
  let c1x: number, c1y: number, c2x: number, c2y: number;

  const isHorizontal = Math.abs(rawDx) >= Math.abs(rawDy);

  if (isHorizontal) {
    if (rawDx >= 0) {
      startX = source.x + sourceHalfW;
      endX = target.x - targetHalfW;
    } else {
      startX = source.x - sourceHalfW;
      endX = target.x + targetHalfW;
    }
    const maxSagY = Math.min(24, Math.abs(rawDy) * 0.2);
    startY = source.y + (rawDy > 0 ? maxSagY : -maxSagY);
    endY = target.y + (rawDy > 0 ? -maxSagY : maxSagY);

    const dx = endX - startX;
    const curvature = Math.min(180, Math.max(40, Math.abs(dx) * 0.45));

    c1x = startX + (dx >= 0 ? curvature : -curvature);
    c1y = startY;
    c2x = endX - (dx >= 0 ? curvature : -curvature);
    c2y = endY;
  } else {
    if (rawDy >= 0) {
      startY = source.y + sourceHalfH;
      endY = target.y - targetHalfH;
    } else {
      startY = source.y - sourceHalfH;
      endY = target.y + targetHalfH;
    }
    const maxSagX = Math.min(24, Math.abs(rawDx) * 0.2);
    startX = source.x + (rawDx > 0 ? maxSagX : -maxSagX);
    endX = target.x + (rawDx > 0 ? -maxSagX : maxSagX);

    const dy = endY - startY;
    const curvature = Math.min(180, Math.max(40, Math.abs(dy) * 0.45));

    c1x = startX;
    c1y = startY + (dy >= 0 ? curvature : -curvature);
    c2x = endX;
    c2y = endY - (dy >= 0 ? curvature : -curvature);
  }

  const path = `M ${startX.toFixed(1)} ${startY.toFixed(1)} C ${c1x.toFixed(1)} ${c1y.toFixed(1)}, ${c2x.toFixed(1)} ${c2y.toFixed(1)}, ${endX.toFixed(1)} ${endY.toFixed(1)}`;

  const midX = computeCubicBezierPoint(0.5, startX, c1x, c2x, endX);
  const midY = computeCubicBezierPoint(0.5, startY, c1y, c2y, endY);

  return {
    path,
    x1: startX,
    y1: startY,
    x2: endX,
    y2: endY,
    midX,
    midY,
    c1x,
    c1y,
    c2x,
    c2y,
    dist,
  };
}
