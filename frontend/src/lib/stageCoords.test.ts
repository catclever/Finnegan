import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { calculateSpatialPlacement, computeClusteredChildCoords, isTemporaryCard, isClusteredChildCard } from './stageCoords.ts';

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

  it('computes clustered child coordinates strictly according to physical deck formula', () => {
    const mother = { x: 100, y: 200, z: 10, rotation: 5.0 };
    const child1 = computeClusteredChildCoords(mother, 1);
    assert.strictEqual(child1.x, 105);
    assert.strictEqual(child1.y, 205);
    assert.strictEqual(child1.z, 12.5);
    assert.strictEqual(child1.rotation, 6.2);

    const child2 = computeClusteredChildCoords(mother, 2);
    assert.strictEqual(child2.x, 110);
    assert.strictEqual(child2.y, 210);
    assert.strictEqual(child2.z, 12.5);
    assert.strictEqual(child2.rotation, 7.4);
  });

  it('automatically clusters collected child card under its mother card during calculateSpatialPlacement', () => {
    const motherCard = {
      id: 'mother-100',
      x: 50,
      y: 60,
      z: 8,
      rotation: 2.0,
      isTemporary: false,
      isFadingOut: false
    };

    const childCard = {
      id: 'child-101',
      echoOf: 'mother-100',
      isTemporary: false,
      isFadingOut: false
    };

    const coord = calculateSpatialPlacement([motherCard as any], 0, false, childCard as any);
    assert.strictEqual(coord.x, 55);
    assert.strictEqual(coord.y, 65);
    assert.strictEqual(coord.z, 10.5);
    assert.strictEqual(coord.rotation, 3.2);
  });

  it('temporary bugu cards with echoOf bypass clustering and scatter freely via spatial placement', () => {
    const motherCard = {
      id: 'mother-100',
      x: 50,
      y: 60,
      z: 8,
      rotation: 2.0,
      isTemporary: false,
      isFadingOut: false
    };

    const tempBuguCard = {
      id: 'bugu-echo-child',
      origin: 'bugu',
      type: 'babel',
      echoOf: 'mother-100',
      isFadingOut: false
      // isTemporary is undefined by default on newly arrived bugu cards
    };

    assert.strictEqual(isTemporaryCard(tempBuguCard), true);
    assert.strictEqual(isClusteredChildCard(tempBuguCard as any, [motherCard as any]), false);

    const coord = calculateSpatialPlacement([motherCard as any], 0, false, tempBuguCard as any);
    // Must NOT be placed at clustered coords (55, 65, 10.5)
    assert.notStrictEqual(coord.x, 55);
    assert.notStrictEqual(coord.y, 65);
    assert.ok(coord.z >= 20 && coord.z <= 30, `z ${coord.z} should be in bugu jitter range [20, 30]`);
  });

  it('determines clustered child card status strictly based on collected state and mother presence', () => {
    const mother = { id: 'm-1', isFadingOut: false };
    const fadingMother = { id: 'm-fading', isFadingOut: true };

    // 1. Temporary card with echoOf -> NOT clustered child
    assert.strictEqual(isClusteredChildCard({ id: 'c1', origin: 'bugu', echoOf: 'm-1' } as any, [mother as any]), false);

    // 2. Collected card with echoOf and solid mother on stage -> IS clustered child
    assert.strictEqual(isClusteredChildCard({ id: 'c2', isTemporary: false, echoOf: 'm-1' } as any, [mother as any]), true);

    // 3. Collected card whose mother is fading out -> NOT clustered child
    assert.strictEqual(isClusteredChildCard({ id: 'c3', isTemporary: false, echoOf: 'm-fading' } as any, [fadingMother as any]), false);

    // 4. Collected card whose mother is not on stage -> NOT clustered child
    assert.strictEqual(isClusteredChildCard({ id: 'c4', isTemporary: false, echoOf: 'm-missing' } as any, [mother as any]), false);

    // 5. Collected card without echoOf -> NOT clustered child
    assert.strictEqual(isClusteredChildCard({ id: 'c5', isTemporary: false } as any, [mother as any]), false);
  });
});
