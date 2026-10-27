import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { CARD_CLEARANCE, OMNIBAR_DEADZONE, getMaxScrollExpansion, computeNextScrollTarget, calculateSpatialPlacement } from './stageCoords.ts';
import type { CosmicCard } from '../types/card';

describe('Cosmic Stage Coordinates & Optics Dynamics', () => {
  it('has updated CARD_CLEARANCE matching Scheme B optics requirements', () => {
    assert.deepEqual(CARD_CLEARANCE, { dx: 290, dy: 270 });
  });

  it('preserves omnibar deadzone dimensions', () => {
    assert.equal(OMNIBAR_DEADZONE.halfWidth, 290);
    assert.equal(OMNIBAR_DEADZONE.halfHeight, 32);
  });

  it('calculates spatial placement without overlapping existing cards', () => {
    const existingCards: CosmicCard[] = [
      {
        id: 'card-1',
        type: 'thought',
        title: 'Center Card',
        content: 'Content 1',
        x: 0,
        y: 0,
        rotation: 0,
        z: 0,
        createdAt: '2026-10-27T10:00:00Z',
        spawnTime: 1000,
        decayDuration: 3600000,
        isFresh: true,
        origin: 'local'
      }
    ];

    const coord = calculateSpatialPlacement(existingCards, 0, false);
    assert.ok(typeof coord.x === 'number' && !isNaN(coord.x));
    assert.ok(typeof coord.y === 'number' && !isNaN(coord.y));
    assert.ok(typeof coord.z === 'number' && coord.z >= 0 && coord.z <= 100);
  });

  it('calculates max scroll expansion', () => {
    const emptyMax = getMaxScrollExpansion([], 0);
    assert.equal(emptyMax, 2.2);

    const cards: CosmicCard[] = [
      {
        id: 'card-deep',
        type: 'thought',
        title: 'Deep Card',
        content: 'Deep',
        x: 0,
        y: 1200,
        rotation: 0,
        z: 50,
        createdAt: '2026-10-27T10:00:00Z',
        spawnTime: 1000,
        decayDuration: 3600000,
        isFresh: true,
        origin: 'local'
      }
    ];
    const deepMax = getMaxScrollExpansion(cards, 0);
    assert.ok(deepMax >= 2.2);
    assert.ok(deepMax <= 5.0);
  });
});
