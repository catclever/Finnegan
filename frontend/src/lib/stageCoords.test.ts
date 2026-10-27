import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { calculateSpatialPlacement } from './stageCoords.ts';

describe('Cosmic Stage Coordinates - Bugu/Radar Jitter Depth', () => {
  it('calculates initial depth with 20~30 random jitter for bugu origin cards', () => {
    for (let i = 0; i < 30; i++) {
      const coord = calculateSpatialPlacement([], 0, true, {
        id: `bugu-card-${i}`,
        origin: 'bugu',
        type: 'thought',
        title: 'Bugu discovery',
        content: 'content',
        x: 0,
        y: 0,
        rotation: 0,
        z: 0,
        spawnTime: Date.now(),
        decayDuration: 3600000,
        isFresh: true,
        isVIP: false,
      } as any);

      assert.ok(coord.z >= 20.0, `z ${coord.z} should be >= 20.0`);
      assert.ok(coord.z <= 30.0, `z ${coord.z} should be <= 30.0`);
    }
  });

  it('calculates initial depth with 20~30 random jitter for radar type cards', () => {
    for (let i = 0; i < 30; i++) {
      const coord = calculateSpatialPlacement([], 0, false, {
        id: `radar-card-${i}`,
        origin: 'local',
        type: 'radar',
        title: 'Radar item',
        content: 'content',
        x: 0,
        y: 0,
        rotation: 0,
        z: 0,
        spawnTime: Date.now(),
        decayDuration: 3600000,
        isFresh: true,
        isVIP: false,
      } as any);

      assert.ok(coord.z >= 20.0, `z ${coord.z} should be >= 20.0`);
      assert.ok(coord.z <= 30.0, `z ${coord.z} should be <= 30.0`);
    }
  });
});
