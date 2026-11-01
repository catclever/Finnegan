import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import type { CosmicCard } from '../types/card.ts';
// @ts-ignore
import {
  extractCardThemes,
  computeOrbitalButtonLayout,
  getCardTimestamp,
} from './themeOrbit.ts';

describe('Finnegan Theme Thread & Orbital Layout Engine (Task 150)', () => {
  describe('extractCardThemes', () => {
    it('returns empty array when card is null or undefined or has no themes', () => {
      assert.deepEqual(extractCardThemes(null as any), []);
      assert.deepEqual(extractCardThemes(undefined as any), []);

      const plainCard: CosmicCard = {
        id: 'c1',
        title: 'Plain Thought',
        content: 'No threads',
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };
      assert.deepEqual(extractCardThemes(plainCard, []), []);
    });

    it('extracts theme declared in card own threads', () => {
      const rootCard: CosmicCard = {
        id: 'theme-root',
        title: '架构重构',
        content: 'Theme root content',
        createdAt: '2026-10-01T10:00:00Z',
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const cardA: CosmicCard = {
        id: 'card-a',
        title: 'Thought A',
        content: 'Mentions @架构重构',
        createdAt: '2026-10-02T10:00:00Z',
        threads: [{ id: 'theme-root', title: '架构重构' }],
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const cardB: CosmicCard = {
        id: 'card-b',
        title: 'Thought B',
        content: 'Also mentions @架构重构',
        createdAt: '2026-10-03T10:00:00Z',
        threads: [{ id: 'theme-root', title: '架构重构' }],
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const themes = extractCardThemes(cardA, [rootCard, cardA, cardB]);
      assert.equal(themes.length, 1);
      assert.equal(themes[0].id, 'theme-root');
      assert.equal(themes[0].title, '架构重构');

      // Cards must be sorted chronologically: rootCard (Oct 1) -> cardA (Oct 2) -> cardB (Oct 3)
      assert.equal(themes[0].cards.length, 3);
      assert.equal(themes[0].cards[0].id, 'theme-root');
      assert.equal(themes[0].cards[1].id, 'card-a');
      assert.equal(themes[0].cards[2].id, 'card-b');
    });

    it('extracts theme when other cards point to this card as the root', () => {
      const rootCard: CosmicCard = {
        id: 'my-theme-id',
        title: '分布式思维',
        content: 'Central note',
        createdAt: '2026-10-10T12:00:00Z',
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const childCard: CosmicCard = {
        id: 'child-1',
        title: 'Child 1',
        content: 'Points to root',
        createdAt: '2026-10-11T12:00:00Z',
        threads: [{ id: 'my-theme-id', title: '分布式思维' }],
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      // Calling on rootCard should detect it is the root of the theme
      const themes = extractCardThemes(rootCard, [rootCard, childCard]);
      assert.equal(themes.length, 1);
      assert.equal(themes[0].id, 'my-theme-id');
      assert.equal(themes[0].title, '分布式思维');
      assert.equal(themes[0].cards.length, 2);
      assert.equal(themes[0].cards[0].id, 'my-theme-id');
      assert.equal(themes[0].cards[1].id, 'child-1');
    });

    it('handles multiple themes for a single card', () => {
      const card: CosmicCard = {
        id: 'multi-card',
        title: 'Hybrid Note',
        content: 'Cross theme note',
        createdAt: '2026-10-15T00:00:00Z',
        threads: [
          { id: 't1', title: 'Theme 1' },
          { id: 't2', title: 'Theme 2' }
        ],
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const themes = extractCardThemes(card, [card]);
      assert.equal(themes.length, 2);
      assert.deepEqual(themes.map(t => t.id).sort(), ['t1', 't2']);
    });
  });

  describe('computeOrbitalButtonLayout', () => {
    it('returns empty array when totalThemes is 0 or negative', () => {
      assert.deepEqual(computeOrbitalButtonLayout(0, false), []);
      assert.deepEqual(computeOrbitalButtonLayout(-1, true), []);
    });

    it('returns single centered button for totalThemes = 1', () => {
      // Left button: inward points right (+X)
      const leftLayout = computeOrbitalButtonLayout(1, false, 72);
      assert.equal(leftLayout.length, 1);
      assert.equal(leftLayout[0].x, 72);
      assert.equal(leftLayout[0].y, 0);

      // Right button: inward points left (-X)
      const rightLayout = computeOrbitalButtonLayout(1, true, 72);
      assert.equal(rightLayout.length, 1);
      assert.equal(rightLayout[0].x, -72);
      assert.equal(rightLayout[0].y, 0);
    });

    it('calculates symmetrical inward arc for multiple themes', () => {
      const layoutLeft = computeOrbitalButtonLayout(3, false, 72);
      assert.equal(layoutLeft.length, 3);

      // Middle button is along horizontal axis y = 0, pointing inward (+X)
      assert.equal(layoutLeft[1].y, 0);
      assert.equal(layoutLeft[1].x, 72);

      // Top and bottom buttons are symmetrical around y = 0
      assert.equal(layoutLeft[0].y, -layoutLeft[2].y);
      assert.equal(layoutLeft[0].x, layoutLeft[2].x);
      assert.ok(layoutLeft[0].x > 0, 'Left side buttons must all have positive X (pointing inward)');

      // Right side nav button
      const layoutRight = computeOrbitalButtonLayout(3, true, 72);
      assert.equal(layoutRight.length, 3);
      assert.equal(layoutRight[1].y, 0);
      assert.equal(layoutRight[1].x, -72);
      assert.ok(layoutRight[0].x < 0, 'Right side buttons must all have negative X (pointing inward)');
    });
  });
});
