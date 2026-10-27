/**
 * Finnegan Cosmic Stage - Universal Gravitational Fallback Engine Tests
 * Task: gravity_fallback_elevated_cards_112
 *
 * Verifies:
 * 1. 10-minute elevation protection determination (isElevationProtectionActive)
 * 2. Natural intrinsic depth calculation & sinking (calculateGravitationalFallbackZ)
 * 3. Strict "only backward, not forward (intrinsicZ > z + 5)" rule
 * 4. Optical aura & visual cue decoupling (auraTraces & isSearchResult preservation)
 * 5. Window focus fallback batch processing (processWindowFocusGravitationalFallback)
 * 6. Search & tag elevation injection (transformSearchResults & calculateSearchRepulsion)
 */

// @ts-nocheck
import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  ELEVATED_PROTECTION_MS,
  isTemporaryCard,
  isPermanentCard,
  isElevationProtectionActive,
  calculateGravitationalFallbackZ,
  applyGravitationalFallback,
  processWindowFocusGravitationalFallback,
} from './gravityFallback.ts';
import { calculateIntrinsicTargetZ } from './tagNetwork.ts';
import { transformSearchResults, calculateSearchRepulsion } from './auraSearch.ts';

describe('Universal Gravitational Fallback Engine', () => {
  const DAY_MS = 24 * 3600 * 1000;
  const MINUTE_MS = 60 * 1000;
  const NOW = 1750000000000;

  describe('1. 10-Minute Elevation Protection Window (isElevationProtectionActive)', () => {
    it('defines ELEVATED_PROTECTION_MS as exactly 10 minutes (600,000ms)', () => {
      assert.equal(ELEVATED_PROTECTION_MS, 10 * 60 * 1000);
    });

    it('identifies temporary vs permanent cards correctly', () => {
      assert.equal(isTemporaryCard({ id: 'c1', origin: 'bugu' }), true);
      assert.equal(isTemporaryCard({ id: 'c2', type: 'radar' }), true);
      assert.equal(isTemporaryCard({ id: 'c3', isTemporary: true }), true);
      assert.equal(isTemporaryCard({ id: 'c4', origin: 'local', type: 'thought' }), false);
      assert.equal(isTemporaryCard({ id: 'c5', origin: 'manual', type: 'note' }), false);

      assert.equal(isPermanentCard({ id: 'c4', origin: 'local' }), true);
      assert.equal(isPermanentCard({ id: 'c1', origin: 'bugu' }), false);
    });

    it('returns true when card is within the 10-minute protection window', () => {
      // Just elevated (0ms elapsed)
      const justElevated = { id: 'c1', elevatedAt: NOW };
      assert.equal(isElevationProtectionActive(justElevated, NOW), true);

      // Elevated 5 minutes ago
      const fiveMinAgo = { id: 'c2', elevatedAt: NOW - 5 * MINUTE_MS };
      assert.equal(isElevationProtectionActive(fiveMinAgo, NOW), true);

      // Elevated 9.99 minutes ago (almost expired, but still protected)
      const almostExpired = { id: 'c3', elevatedAt: NOW - 9.99 * MINUTE_MS };
      assert.equal(isElevationProtectionActive(almostExpired, NOW), true);
    });

    it('returns false when the 10-minute protection window has elapsed', () => {
      // Exactly 10 minutes elapsed
      const exactTen = { id: 'c1', elevatedAt: NOW - 10 * MINUTE_MS };
      assert.equal(isElevationProtectionActive(exactTen, NOW), false);

      // 15 minutes elapsed
      const fifteenMinAgo = { id: 'c2', elevatedAt: NOW - 15 * MINUTE_MS };
      assert.equal(isElevationProtectionActive(fifteenMinAgo, NOW), false);

      // 1 day elapsed
      const oneDayAgo = { id: 'c3', elevatedAt: NOW - DAY_MS };
      assert.equal(isElevationProtectionActive(oneDayAgo, NOW), false);
    });

    it('returns false for cards without elevatedAt timestamp', () => {
      assert.equal(isElevationProtectionActive({ id: 'c1' }, NOW), false);
      assert.equal(isElevationProtectionActive({ id: 'c2', elevatedAt: undefined }, NOW), false);
      assert.equal(isElevationProtectionActive(null, NOW), false);
    });
  });

  describe('2. Intrinsic Depth Sinking & Optical Decoupling', () => {
    it('calculates intrinsic depth and triggers sinking for an old card pulled to foreground', () => {
      // Card created 60 days ago: intrinsic Z is ~66..78
      const oldCard: CosmicCard = {
        id: 'old-1',
        title: 'Ancient Architecture Note',
        content: 'Design decisions from two months ago',
        createdAt: new Date(NOW - 60 * DAY_MS).toISOString(),
        origin: 'local',
        type: 'note',
        x: 0,
        y: 0,
        rotation: 0,
        z: 12, // pulled to foreground
        spawnTime: NOW - 20 * MINUTE_MS,
        decayDuration: 12 * 3600 * 1000,
        isFresh: false,
        isVIP: false,
        elevatedAt: NOW - 15 * MINUTE_MS, // 15 min ago (> 10 min protection)
        auraTraces: [{ searchId: 's1', color: 'hsl(210, 95%, 62%)', hitTime: NOW - 15 * MINUTE_MS }],
        isSearchResult: true,
      };

      const { shouldSink, targetZ, intrinsicZ } = calculateGravitationalFallbackZ(oldCard, NOW);
      assert.equal(shouldSink, true);
      assert.ok(targetZ >= 66 && targetZ <= 78, `targetZ (${targetZ}) should match 60-day intrinsic band (66~78)`);
      assert.equal(targetZ, intrinsicZ);

      // Apply fallback and verify
      const fallen = applyGravitationalFallback(oldCard, NOW);
      assert.equal(fallen.z, targetZ);
      assert.ok(fallen.z > oldCard.z + 5, 'Card must sink deeper than z + 5');

      // Optical decoupling: visual aura, search flags, and metadata MUST remain intact
      assert.equal(fallen.isSearchResult, true, 'isSearchResult must be preserved');
      assert.equal(fallen.auraTraces?.length, 1, 'auraTraces must be preserved');
      assert.equal(fallen.auraTraces?.[0].color, 'hsl(210, 95%, 62%)');
      assert.equal(fallen.title, 'Ancient Architecture Note');
      assert.equal(fallen.content, 'Design decisions from two months ago');
    });

    it('protects cards from sinking while within the 10-minute window', () => {
      const protectedCard: CosmicCard = {
        id: 'prot-1',
        title: 'Protected Note',
        content: 'Recently pulled card',
        createdAt: new Date(NOW - 90 * DAY_MS).toISOString(),
        origin: 'local',
        type: 'thought',
        x: 0,
        y: 0,
        rotation: 0,
        z: 15,
        spawnTime: NOW - 5 * MINUTE_MS,
        decayDuration: 3600000,
        isFresh: true,
        isVIP: false,
        elevatedAt: NOW - 5 * MINUTE_MS, // 5 min ago (< 10 min protection)
      };

      const result = applyGravitationalFallback(protectedCard, NOW);
      assert.equal(result.z, 15, 'Must not sink during active 10-minute protection window');
    });
  });

  describe('3. Strict "Only Backward, Not Forward (intrinsicZ > z + 5)" Rule', () => {
    it('strictly forbids moving forward when intrinsic depth is shallower than current depth', () => {
      // Fresh card created 30 minutes ago: intrinsic depth is ~3..11
      // Suppose it is sitting at depth z = 22
      const freshCard: CosmicCard = {
        id: 'fresh-1',
        title: 'Today Thought',
        content: 'Just thought about this',
        createdAt: new Date(NOW - 30 * MINUTE_MS).toISOString(),
        origin: 'local',
        type: 'thought',
        x: 0,
        y: 0,
        rotation: 0,
        z: 22,
        spawnTime: NOW - 20 * MINUTE_MS,
        decayDuration: 3600000,
        isFresh: true,
        isVIP: false,
        elevatedAt: NOW - 12 * MINUTE_MS, // protection elapsed (> 10 min)
      };

      const { shouldSink, targetZ, intrinsicZ } = calculateGravitationalFallbackZ(freshCard, NOW);
      assert.ok(intrinsicZ <= 11, `Today card intrinsicZ (${intrinsicZ}) should be <= 11`);
      // Since intrinsicZ (e.g. 7) is less than current z (22), moving to intrinsicZ would mean jumping forward
      assert.equal(shouldSink, false, 'Should sink must be false when intrinsicZ is shallower');
      assert.equal(targetZ, 22, 'TargetZ must remain at current depth');

      const result = applyGravitationalFallback(freshCard, NOW);
      assert.equal(result.z, 22, 'Card depth must remain at 22 and not jump forward');
    });

    it('enforces the +5 threshold: does not sink for minor perturbations <= 5', () => {
      // Suppose card is at z = 48, and intrinsicZ happens to be 51
      const borderlineCard: CosmicCard = {
        id: 'border-1',
        title: 'Borderline Depth Card',
        content: 'Content',
        createdAt: new Date(NOW - 20 * DAY_MS).toISOString(), // ~48..62
        origin: 'local',
        type: 'thought',
        x: 0,
        y: 0,
        rotation: 0,
        z: 58, // current depth is 58
        spawnTime: NOW - 20 * MINUTE_MS,
        decayDuration: 3600000,
        isFresh: false,
        isVIP: false,
        elevatedAt: NOW - 15 * MINUTE_MS,
      };

      // Even if intrinsicZ is 60 (which is 58 + 2), 60 is NOT > 58 + 5 = 63
      const { shouldSink } = calculateGravitationalFallbackZ(borderlineCard, NOW);
      // Since 20 days intrinsic band is [48, 62], and currentZ is 58,
      // intrinsicZ (<= 62) is NEVER > 58 + 5 = 63!
      assert.equal(shouldSink, false, 'Must not sink when delta <= 5');
    });

    it('sinks properly when intrinsic depth is genuinely deeper (intrinsicZ > z + 5)', () => {
      // Card created 200 days ago: intrinsic Z is ~80..92
      // Pulled to foreground z = 20
      const deepCard: CosmicCard = {
        id: 'deep-1',
        title: 'Old Card',
        content: 'Content',
        createdAt: new Date(NOW - 200 * DAY_MS).toISOString(),
        origin: 'local',
        type: 'thought',
        x: 0,
        y: 0,
        rotation: 0,
        z: 20,
        spawnTime: NOW - 20 * MINUTE_MS,
        decayDuration: 3600000,
        isFresh: false,
        isVIP: false,
        elevatedAt: NOW - 15 * MINUTE_MS,
      };

      const { shouldSink, targetZ } = calculateGravitationalFallbackZ(deepCard, NOW);
      assert.equal(shouldSink, true);
      assert.ok(targetZ >= 80 && targetZ <= 92);
      assert.ok(targetZ > deepCard.z + 5);

      const result = applyGravitationalFallback(deepCard, NOW);
      assert.equal(result.z, targetZ);
    });
  });

  describe('4. Batch Window Focus Processing (processWindowFocusGravitationalFallback)', () => {
    it('processes active stage cards correctly according to all rules', () => {
      const cards: CosmicCard[] = [
        // 1. Permanent elevated card, protected (< 10 min) -> no change
        {
          id: 'card-prot',
          title: 'Protected',
          content: 'c1',
          createdAt: new Date(NOW - 100 * DAY_MS).toISOString(),
          origin: 'local',
          z: 15,
          spawnTime: NOW,
          decayDuration: 3600000,
          isFresh: true,
          isVIP: false,
          elevatedAt: NOW - 4 * MINUTE_MS, // 4 min ago
          x: 0, y: 0, rotation: 0
        },
        // 2. Permanent elevated card, expired (> 10 min), old note -> sinks to intrinsic depth
        {
          id: 'card-sink',
          title: 'Should Sink',
          content: 'c2',
          createdAt: new Date(NOW - 90 * DAY_MS).toISOString(),
          origin: 'local',
          z: 10,
          spawnTime: NOW - 20 * MINUTE_MS,
          decayDuration: 3600000,
          isFresh: false,
          isVIP: false,
          elevatedAt: NOW - 15 * MINUTE_MS, // 15 min ago
          auraTraces: [{ searchId: 's1', color: 'hsl(100, 95%, 62%)', hitTime: NOW - 15 * MINUTE_MS }],
          isSearchResult: true,
          x: 0, y: 0, rotation: 0
        },
        // 3. Permanent elevated card, expired (> 10 min), but created today -> stays at z (only backward)
        {
          id: 'card-today',
          title: 'Today Note',
          content: 'c3',
          createdAt: new Date(NOW - 1 * MINUTE_MS).toISOString(),
          origin: 'local',
          z: 25,
          spawnTime: NOW - 20 * MINUTE_MS,
          decayDuration: 3600000,
          isFresh: true,
          isVIP: false,
          elevatedAt: NOW - 12 * MINUTE_MS,
          x: 0, y: 0, rotation: 0
        },
        // 4. Currently focused card -> strictly untouched
        {
          id: 'card-focused',
          title: 'Focused',
          content: 'c4',
          createdAt: new Date(NOW - 100 * DAY_MS).toISOString(),
          origin: 'local',
          z: 10,
          spawnTime: NOW - 20 * MINUTE_MS,
          decayDuration: 3600000,
          isFresh: false,
          isVIP: false,
          elevatedAt: NOW - 15 * MINUTE_MS,
          x: 0, y: 0, rotation: 0
        },
        // 5. Departing card (isFadingOut: true) -> untouched
        {
          id: 'card-fading',
          title: 'Fading',
          content: 'c5',
          createdAt: new Date(NOW - 100 * DAY_MS).toISOString(),
          origin: 'local',
          z: 10,
          spawnTime: NOW - 20 * MINUTE_MS,
          decayDuration: 3600000,
          isFresh: false,
          isVIP: false,
          isFadingOut: true,
          elevatedAt: NOW - 15 * MINUTE_MS,
          x: 0, y: 0, rotation: 0
        },
        // 6. Temporary card (origin: 'bugu') -> untouched by permanent fallback
        {
          id: 'card-bugu',
          title: 'Bugu Discovery',
          content: 'c6',
          origin: 'bugu',
          type: 'babel',
          z: 10,
          spawnTime: NOW - 20 * MINUTE_MS,
          decayDuration: 3600000,
          isFresh: false,
          isVIP: false,
          elevatedAt: NOW - 15 * MINUTE_MS,
          x: 0, y: 0, rotation: 0
        },
        // 7. Permanent card never elevated -> untouched
        {
          id: 'card-never-elevated',
          title: 'Default stage card',
          content: 'c7',
          origin: 'local',
          z: 50,
          spawnTime: NOW - 50 * MINUTE_MS,
          decayDuration: 3600000,
          isFresh: false,
          isVIP: false,
          x: 0, y: 0, rotation: 0
        }
      ];

      const processed = processWindowFocusGravitationalFallback(cards, NOW, 'card-focused');

      // 1. Protected stays at z = 15
      assert.equal(processed.find(c => c.id === 'card-prot')?.z, 15);

      // 2. Expired old card sinks to > 65
      const sunk = processed.find(c => c.id === 'card-sink');
      assert.ok(sunk && sunk.z >= 66 && sunk.z <= 78, `sunk.z (${sunk?.z}) should be in 66~78`);
      assert.equal(sunk.isSearchResult, true);
      assert.equal(sunk.auraTraces?.length, 1);

      // 3. Today card stays at z = 25 (did not jump forward)
      assert.equal(processed.find(c => c.id === 'card-today')?.z, 25);

      // 4. Focused card stays at z = 10
      assert.equal(processed.find(c => c.id === 'card-focused')?.z, 10);

      // 5. Fading card stays at z = 10
      assert.equal(processed.find(c => c.id === 'card-fading')?.z, 10);

      // 6. Bugu temporary card stays untouched by permanent fallback engine
      assert.equal(processed.find(c => c.id === 'card-bugu')?.z, 10);

      // 7. Never-elevated card stays at z = 50
      assert.equal(processed.find(c => c.id === 'card-never-elevated')?.z, 50);
    });
  });

  describe('5. Search Elevation Integration (transformSearchResults & calculateSearchRepulsion)', () => {
    it('injects elevatedAt: now into both existing and newly transformed search cards', () => {
      const existingCard: CosmicCard = {
        id: 'exist-1',
        title: 'Existing',
        content: 'content',
        origin: 'local',
        z: 50,
        spawnTime: NOW - 3600000,
        decayDuration: 3600000,
        isFresh: false,
        isVIP: false,
        x: 0, y: 0, rotation: 0
      };

      const rawResults = [
        { id: 'exist-1', title: 'Existing' },
        { id: 'new-1', title: 'Newly Found', content: 'hello' }
      ];

      const transformed = transformSearchResults(rawResults, [[existingCard]], 'search_1', 'hsl(200, 95%, 62%)', NOW);
      assert.equal(transformed.length, 2);

      const transformedExisting = transformed.find(c => c.id === 'exist-1');
      assert.equal(transformedExisting?.elevatedAt, NOW, 'Existing card must have elevatedAt injected');
      assert.equal(transformedExisting?.isSearchResult, true);

      const transformedNew = transformed.find(c => c.id === 'new-1');
      assert.equal(transformedNew?.elevatedAt, NOW, 'New card must have elevatedAt injected');
      assert.equal(transformedNew?.isSearchResult, true);
    });

    it('preserves elevatedAt in calculateSearchRepulsion foregroundCards', () => {
      const searchCards: CosmicCard[] = [
        {
          id: 'sc-1',
          title: 'Search Result',
          content: 'text',
          origin: 'local',
          z: 25,
          spawnTime: NOW,
          decayDuration: 3600000,
          isFresh: true,
          isVIP: false,
          elevatedAt: NOW,
          x: 0, y: 0, rotation: 0
        }
      ];

      const repulsion = calculateSearchRepulsion([], searchCards, 10);
      assert.equal(repulsion.foregroundCards.length, 1);
      assert.equal(repulsion.foregroundCards[0].elevatedAt, NOW);
    });
  });
});
