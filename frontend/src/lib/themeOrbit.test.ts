import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import type { CosmicCard } from '../types/card.ts';
// @ts-ignore
import {
  extractCardThemes,
  computeOrbitalButtonLayout,
  getCardTimestamp,
  findAvailableTheme,
  recordThemeEmit,
  clearThemeEmitHistory,
  getThemeCooldownRemaining,
  THEME_COOLDOWN_MS
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

    it('transitive chain C -> B -> A resolves to single unified theme A for all cards', () => {
      const cardA: CosmicCard = {
        id: 'card-a',
        title: 'Theme Root A',
        content: 'I am root',
        createdAt: '2026-10-01T10:00:00Z',
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const cardB: CosmicCard = {
        id: 'card-b',
        title: 'Middle B',
        content: 'References @Theme Root A',
        createdAt: '2026-10-02T10:00:00Z',
        threads: [{ id: 'card-a', title: 'Theme Root A' }],
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const cardC: CosmicCard = {
        id: 'card-c',
        title: 'Leaf C',
        content: 'References @Middle B',
        createdAt: '2026-10-03T10:00:00Z',
        threads: [{ id: 'card-b', title: 'Middle B' }],
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const pool = [cardA, cardB, cardC];

      // Focusing on A, B, or C should all yield the exact same single theme group 'card-a' with [A, B, C]
      const fromA = extractCardThemes(cardA, pool);
      assert.equal(fromA.length, 1);
      assert.equal(fromA[0].id, 'card-a');
      assert.equal(fromA[0].title, 'Theme Root A');
      assert.deepEqual(fromA[0].cards.map(c => c.id), ['card-a', 'card-b', 'card-c']);

      const fromB = extractCardThemes(cardB, pool);
      assert.equal(fromB.length, 1);
      assert.equal(fromB[0].id, 'card-a');
      assert.deepEqual(fromB[0].cards.map(c => c.id), ['card-a', 'card-b', 'card-c']);

      const fromC = extractCardThemes(cardC, pool);
      assert.equal(fromC.length, 1);
      assert.equal(fromC[0].id, 'card-a');
      assert.deepEqual(fromC[0].cards.map(c => c.id), ['card-a', 'card-b', 'card-c']);

      // Extended chain D -> C -> B -> A
      const cardD: CosmicCard = {
        id: 'card-d',
        title: 'Leaf D',
        content: 'References @Leaf C',
        createdAt: '2026-10-04T10:00:00Z',
        threads: [{ id: 'card-c', title: 'Leaf C' }],
        x: 0, y: 0, rotation: 0, z: 0,
        spawnTime: 0, decayDuration: 0, isFresh: false, isVIP: false, origin: 'manual'
      };

      const pool4 = [cardA, cardB, cardC, cardD];
      const fromD = extractCardThemes(cardD, pool4);
      assert.equal(fromD.length, 1);
      assert.equal(fromD[0].id, 'card-a');
      assert.deepEqual(fromD[0].cards.map(c => c.id), ['card-a', 'card-b', 'card-c', 'card-d']);
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

  describe('Theme Barrage Cooldown & Non-Consecutive Selection (30s Window)', () => {
    it('returns null when themes list is empty', () => {
      clearThemeEmitHistory();
      assert.equal(findAvailableTheme([]), null);
    });

    it('enforces 30s cooldown on a single theme card', () => {
      clearThemeEmitHistory();
      const themes = [{ id: 'theme-1', title: '读书笔记' }];
      const t0 = 100_000;

      // First emit succeeds
      const chosen1 = findAvailableTheme(themes, t0);
      assert.equal(chosen1?.id, 'theme-1');
      recordThemeEmit('theme-1', t0);

      // Within 30s (e.g. at 5s, 15s, 29s), cannot emit again
      assert.equal(findAvailableTheme(themes, t0 + 5_000), null);
      assert.equal(findAvailableTheme(themes, t0 + 15_000), null);
      assert.equal(findAvailableTheme(themes, t0 + 29_999), null);
      assert.equal(getThemeCooldownRemaining('theme-1', t0 + 10_000), 20_000);

      // At or after 30s, cooldown expires and theme can emit again
      const chosen2 = findAvailableTheme(themes, t0 + 30_000);
      assert.equal(chosen2?.id, 'theme-1');

      // bypassCooldown allows instant emit on card switch/focus even if within 30s
      recordThemeEmit('theme-1', t0 + 30_000);
      assert.equal(findAvailableTheme(themes, t0 + 31_000, false), null);
      assert.equal(findAvailableTheme(themes, t0 + 31_000, true)?.id, 'theme-1');
    });

    it('prevents consecutive back-to-back duplicate emits across multiple themes', () => {
      clearThemeEmitHistory();
      const themes = [
        { id: 'theme-1', title: '读书笔记' },
        { id: 'theme-2', title: '代码架构' }
      ];
      const t0 = 100_000;

      // 1. First choice can be either theme
      const chosen1 = findAvailableTheme(themes, t0);
      assert.ok(chosen1 !== null);
      recordThemeEmit(chosen1.id, t0);

      // 2. Next choice at t0 + 2s must pick the other theme (not on cooldown, and non-consecutive)
      const chosen2 = findAvailableTheme(themes, t0 + 2_000);
      assert.ok(chosen2 !== null);
      assert.notEqual(chosen2.id, chosen1.id);
      recordThemeEmit(chosen2.id, t0 + 2_000);

      // 3. Now both themes are on cooldown at t0 + 5s
      assert.equal(findAvailableTheme(themes, t0 + 5_000), null);

      // 4. At t0 + 30_001s, theme-1 expires first and is the only available
      const chosen3 = findAvailableTheme(themes, t0 + 30_001);
      assert.equal(chosen3?.id, chosen1.id);
    });
  });
});
