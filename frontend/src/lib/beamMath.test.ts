import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { computeAnimatedBeamPath, computeCubicBezierPoint, getCardScale, DOCKING_INSET_PX } from './beamMath.ts';
import type { CosmicCard } from '../types/card';

function createMockCard(id: string, x: number, y: number, z: number = 0): CosmicCard {
  return {
    id,
    title: `Card ${id}`,
    content: 'Content',
    type: 'thought',
    origin: 'local',
    decayDuration: 1000,
    createdAt: Date.now(),
    x,
    y,
    z,
    isFadingOut: false,
  };
}

describe('Cosmic Beam Math & Cubic Bezier Docking Engine', () => {
  describe('getCardScale', () => {
    it('returns 1.08 when card is at z=0 (foreground)', () => {
      const card = createMockCard('c1', 0, 0, 0);
      assert.equal(Math.round(getCardScale(card) * 100) / 100, 1.08);
    });

    it('returns 0.64 when card is at z=100 (deep space)', () => {
      const card = createMockCard('c2', 0, 0, 100);
      assert.equal(Math.round(getCardScale(card) * 100) / 100, 0.64);
    });

    it('clamps z correctly if outside [0, 100]', () => {
      const cardLow = createMockCard('c_low', 0, 0, -50);
      const cardHigh = createMockCard('c_high', 0, 0, 150);
      assert.equal(Math.round(getCardScale(cardLow) * 100) / 100, 1.08);
      assert.equal(Math.round(getCardScale(cardHigh) * 100) / 100, 0.64);
    });
  });

  describe('computeCubicBezierPoint', () => {
    it('computes endpoint values at t=0 and t=1', () => {
      const p0 = 10, p1 = 20, p2 = 80, p3 = 100;
      assert.equal(computeCubicBezierPoint(0, p0, p1, p2, p3), 10);
      assert.equal(computeCubicBezierPoint(1, p0, p1, p2, p3), 100);
    });

    it('computes midpoint at t=0.5 accurately', () => {
      // Linear case: 0, 33.33, 66.66, 100 -> mid = 50
      const mid = computeCubicBezierPoint(0.5, 0, 33.333333, 66.666667, 100);
      assert.equal(Math.round(mid), 50);
    });
  });

  describe('computeAnimatedBeamPath', () => {
    it('calculates horizontal dominant path between source and target', () => {
      const source = createMockCard('src', 0, 0, 0);
      const target = createMockCard('tgt', 500, 50, 0);

      const res = computeAnimatedBeamPath(source, target);

      // Horizontal dominant: rawDx (500) > rawDy (50)
      // Source exits right edge: x1 > source.x
      // Target enters left edge: x2 < target.x
      assert.ok(res.x1 > source.x);
      assert.ok(res.x2 < target.x);
      assert.ok(res.path.startsWith('M '));
      assert.ok(res.path.includes(' C '));
      // Midpoint should lie roughly halfway between x1 and x2
      assert.ok(res.midX > res.x1 && res.midX < res.x2);
    });

    it('calculates vertical dominant path for vertical placement', () => {
      const source = createMockCard('src', 0, 0, 0);
      const target = createMockCard('tgt', 30, 600, 0);

      const res = computeAnimatedBeamPath(source, target);

      // Vertical dominant: rawDy (600) > rawDx (30)
      // Source exits bottom edge: y1 > source.y
      // Target enters top edge: y2 < target.y
      assert.ok(res.y1 > source.y);
      assert.ok(res.y2 < target.y);
      assert.ok(res.midY > res.y1 && res.midY < res.y2);
    });

    it('handles degenerate cases with dist < 1 gracefully', () => {
      const source = createMockCard('src', 100, 200, 0);
      const target = createMockCard('tgt', 100, 200, 0);

      const res = computeAnimatedBeamPath(source, target);
      assert.equal(res.x1, 100);
      assert.equal(res.y1, 200);
      assert.equal(res.x2, 100);
      assert.equal(res.y2, 200);
    });
  });
});
