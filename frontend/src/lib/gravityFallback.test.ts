/**
 * Finnegan Cosmic Stage - Universal Gravitational Fallback Engine Tests
 * Task: gravity_fallback_unlock_extinction_cooldown
 *
 * Verifies:
 * 1. Active aura lock (hasActiveAura): As long as light is on, card is strictly locked in foreground.
 * 2. Extinction & 10-minute cooldown (isElevationProtectionActive):
 *    - Phase 1: Light shining -> locked & protected.
 *    - Phase 2: Light just extinguished -> enters 10m cooldown buffer, still protected.
 *    - Phase 3: In 10m cooldown buffer -> still protected.
 *    - Phase 4: > 10m past extinction -> unlocked & eligible for gravitational sinking.
 * 3. Natural intrinsic depth calculation & sinking (calculateGravitationalFallbackZ).
 * 4. Strict "only backward, not forward (intrinsicZ > z + 5)" rule.
 * 5. Optical aura & visual cue decoupling (auraTraces & isSearchResult preservation).
 * 6. Window focus fallback batch processing (processWindowFocusGravitationalFallback).
 * 7. Search & tag elevation integration (transformSearchResults & calculateSearchRepulsion).
 */

// @ts-nocheck
import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  DEFAULT_SEARCH_AURA_DURATION,
  COOLDOWN_AFTER_EXTINCTION_MS,
  ELEVATED_PROTECTION_MS,
  isTemporaryCard,
  isPermanentCard,
  hasActiveAura,
  getAuraExtinctionTime,
  isElevationProtectionActive,
  calculateGravitationalFallbackZ,
  applyGravitationalFallback,
  processWindowFocusGravitationalFallback,
} from './gravityFallback.ts';
import { calculateIntrinsicTargetZ } from './tagNetwork.ts';
import { transformSearchResults, calculateSearchRepulsion, createAuraTrace } from './auraSearch.ts';

describe('Universal Gravitational Fallback Engine', () => {
  const DAY_MS = 24 * 3600 * 1000;
  const MINUTE_MS = 60 * 1000;
  const NOW = 1750000000000;

  describe('1. Active Aura & 10-Minute Post-Extinction Cooldown', () => {
    it('defines standard constants: 15m default search duration, 10m cooldown after extinction', () => {
      assert.equal(DEFAULT_SEARCH_AURA_DURATION, 15 * 60 * 1000);
      assert.equal(COOLDOWN_AFTER_EXTINCTION_MS, 10 * 60 * 1000);
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

    it('detects active aura (light is shining)', () => {
      // 1. Search aura with default duration (15m), hit 5m ago -> active
      const searchActive = {
        id: 'c1',
        auraTraces: [{ searchId: 's1', color: 'hsl(200, 95%, 62%)', hitTime: NOW - 5 * MINUTE_MS, duration: 15 * MINUTE_MS }]
      };
      assert.equal(hasActiveAura(searchActive, NOW), true);

      // 2. Search aura with default duration (15m), hit 16m ago -> extinguished
      const searchExtinguished = {
        id: 'c2',
        auraTraces: [{ searchId: 's1', color: 'hsl(200, 95%, 62%)', hitTime: NOW - 16 * MINUTE_MS, duration: 15 * MINUTE_MS }]
      };
      assert.equal(hasActiveAura(searchExtinguished, NOW), false);

      // 3. Tag aura with 2m duration, hit 1m ago -> active
      const tagActive = {
        id: 'c3',
        auraTraces: [{ searchId: 'tag_1', color: '#38bdf8', hitTime: NOW - 1 * MINUTE_MS, duration: 2 * MINUTE_MS }]
      };
      assert.equal(hasActiveAura(tagActive, NOW), true);

      // 4. Tag aura with 2m duration, hit 3m ago -> extinguished
      const tagExtinguished = {
        id: 'c4',
        auraTraces: [{ searchId: 'tag_1', color: '#38bdf8', hitTime: NOW - 3 * MINUTE_MS, duration: 2 * MINUTE_MS }]
      };
      assert.equal(hasActiveAura(tagExtinguished, NOW), false);
    });

    it('calculates the exact moment when the latest aura extinguished (getAuraExtinctionTime)', () => {
      const card = {
        id: 'c1',
        auraTraces: [
          { searchId: 's1', hitTime: NOW - 20 * MINUTE_MS, duration: 15 * MINUTE_MS }, // extinguished at NOW - 5m
          { searchId: 's2', hitTime: NOW - 10 * MINUTE_MS, duration: 15 * MINUTE_MS }, // will extinguish at NOW + 5m
        ]
      };
      const extTime = getAuraExtinctionTime(card);
      assert.equal(extTime, NOW + 5 * MINUTE_MS);
    });

    it('strictly enforces: Light On -> Locked; Light Off -> 10m Cooldown -> Unlocked (isElevationProtectionActive)', () => {
      // Phase 1: Light is shining (search hit 8m ago, duration 15m). 7m remaining on light.
      const lightOn = {
        id: 'p1',
        auraTraces: [{ searchId: 's1', hitTime: NOW - 8 * MINUTE_MS, duration: 15 * MINUTE_MS }]
      };
      assert.equal(isElevationProtectionActive(lightOn, NOW), true, 'Must be locked while light is on');

      // Phase 2: Light just extinguished 0.5m ago (hit 15.5m ago, duration 15m). Within 10m cooldown buffer.
      const justExtinguished = {
        id: 'p2',
        auraTraces: [{ searchId: 's1', hitTime: NOW - 15.5 * MINUTE_MS, duration: 15 * MINUTE_MS }]
      };
      assert.equal(hasActiveAura(justExtinguished, NOW), false, 'Light is extinguished');
      assert.equal(isElevationProtectionActive(justExtinguished, NOW), true, 'Must remain protected in 10m cooldown buffer');

      // Phase 3: Light extinguished 6m ago (hit 21m ago, duration 15m). 4m remaining in cooldown buffer.
      const halfwayCooldown = {
        id: 'p3',
        auraTraces: [{ searchId: 's1', hitTime: NOW - 21 * MINUTE_MS, duration: 15 * MINUTE_MS }]
      };
      assert.equal(hasActiveAura(halfwayCooldown, NOW), false);
      assert.equal(isElevationProtectionActive(halfwayCooldown, NOW), true, 'Must remain protected halfway through cooldown');

      // Phase 4: Light extinguished 10.1m ago (hit 25.1m ago, duration 15m). Cooldown expired!
      const cooldownExpired = {
        id: 'p4',
        auraTraces: [{ searchId: 's1', hitTime: NOW - 25.1 * MINUTE_MS, duration: 15 * MINUTE_MS }]
      };
      assert.equal(hasActiveAura(cooldownExpired, NOW), false);
      assert.equal(isElevationProtectionActive(cooldownExpired, NOW), false, 'Protection must expire after 10m cooldown');
    });

    it('supports cards elevated via elevatedAt without aura traces (10m grace period)', () => {
      const elevated5mAgo = { id: 'c1', elevatedAt: NOW - 5 * MINUTE_MS };
      assert.equal(isElevationProtectionActive(elevated5mAgo, NOW), true);

      const elevated12mAgo = { id: 'c2', elevatedAt: NOW - 12 * MINUTE_MS };
      assert.equal(isElevationProtectionActive(elevated12mAgo, NOW), false);
    });

    it('returns false for cards with no elevation or aura traces', () => {
      assert.equal(isElevationProtectionActive({ id: 'c1' }, NOW), false);
      assert.equal(isElevationProtectionActive(null, NOW), false);
    });
  });

  describe('2. Intrinsic Depth Sinking & Optical Decoupling', () => {
    it('sinks an old card after light extinguishes AND 10m cooldown buffer elapses', () => {
      // Card created 60 days ago: intrinsic Z is ~66..78
      // Hit 30m ago (15m light + 15m past extinction = > 10m cooldown)
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
        spawnTime: NOW - 30 * MINUTE_MS,
        decayDuration: 12 * 3600 * 1000,
        isFresh: false,
        isVIP: false,
        auraTraces: [{ searchId: 's1', color: 'hsl(210, 95%, 62%)', hitTime: NOW - 30 * MINUTE_MS, duration: 15 * MINUTE_MS }],
        isSearchResult: true,
      };

      assert.equal(isElevationProtectionActive(oldCard, NOW), false, 'Protection must be elapsed');

      const { shouldSink, targetZ, intrinsicZ } = calculateGravitationalFallbackZ(oldCard, NOW);
      assert.equal(shouldSink, true);
      assert.ok(targetZ >= 66 && targetZ <= 78, `targetZ (${targetZ}) should match 60-day intrinsic band (66~78)`);
      assert.equal(targetZ, intrinsicZ);

      // Apply fallback and verify
      const fallen = applyGravitationalFallback(oldCard, NOW);
      assert.ok(fallen.z >= 66 && fallen.z <= 78, `fallen.z (${fallen.z}) should match 60-day intrinsic band (66~78)`);
      assert.ok(fallen.z > oldCard.z + 5, 'Card must sink deeper than z + 5');

      // Optical decoupling: visual aura trace array and search result flag are safely preserved
      assert.equal(fallen.isSearchResult, true, 'isSearchResult must be preserved');
      assert.equal(fallen.auraTraces?.length, 1, 'auraTraces must be preserved');
      assert.equal(fallen.auraTraces?.[0].color, 'hsl(210, 95%, 62%)');
    });

    it('protects cards from sinking while light is still shining', () => {
      const cardWithActiveLight: CosmicCard = {
        id: 'active-light-1',
        title: 'Recently Searched',
        content: 'Glowing note',
        createdAt: new Date(NOW - 90 * DAY_MS).toISOString(),
        origin: 'local',
        type: 'thought',
        x: 0, y: 0, rotation: 0,
        z: 15,
        spawnTime: NOW - 5 * MINUTE_MS,
        auraTraces: [{ searchId: 's1', color: 'hsl(180, 95%, 62%)', hitTime: NOW - 5 * MINUTE_MS, duration: 15 * MINUTE_MS }],
      };

      const result = applyGravitationalFallback(cardWithActiveLight, NOW);
      assert.equal(result.z, 15, 'Must not sink while light is active');
    });

    it('protects cards from sinking during the 10-minute cooldown buffer after light extinguishes', () => {
      const cardInCooldown: CosmicCard = {
        id: 'cooldown-1',
        title: 'Extinguished but cooling',
        content: 'Content',
        createdAt: new Date(NOW - 90 * DAY_MS).toISOString(),
        origin: 'local',
        type: 'thought',
        x: 0, y: 0, rotation: 0,
        z: 15,
        spawnTime: NOW - 18 * MINUTE_MS,
        // Extinguished 3m ago (hit 18m ago, duration 15m). Inside 10m cooldown buffer.
        auraTraces: [{ searchId: 's1', color: 'hsl(180, 95%, 62%)', hitTime: NOW - 18 * MINUTE_MS, duration: 15 * MINUTE_MS }],
      };

      const result = applyGravitationalFallback(cardInCooldown, NOW);
      assert.equal(result.z, 15, 'Must not sink during 10m cooldown after extinction');
    });
  });

  describe('3. Strict "Only Backward, Not Forward (intrinsicZ > z + 5)" Rule', () => {
    it('strictly forbids moving forward when intrinsic depth is shallower than current depth', () => {
      // Fresh card created 30 minutes ago: intrinsic depth is ~3..11
      // Sitting at depth z = 22, hit 30m ago (cooldown elapsed)
      const freshCard: CosmicCard = {
        id: 'fresh-1',
        title: 'Today Thought',
        content: 'Just thought about this',
        createdAt: new Date(NOW - 30 * MINUTE_MS).toISOString(),
        origin: 'local',
        type: 'thought',
        x: 0, y: 0, rotation: 0,
        z: 22,
        spawnTime: NOW - 30 * MINUTE_MS,
        auraTraces: [{ searchId: 's1', hitTime: NOW - 30 * MINUTE_MS, duration: 15 * MINUTE_MS }],
      };

      const { shouldSink, targetZ, intrinsicZ } = calculateGravitationalFallbackZ(freshCard, NOW);
      assert.ok(intrinsicZ <= 11, `Today card intrinsicZ (${intrinsicZ}) should be <= 11`);
      assert.equal(shouldSink, false, 'Should sink must be false when intrinsicZ is shallower');
      assert.equal(targetZ, 22, 'TargetZ must remain at current depth');

      const result = applyGravitationalFallback(freshCard, NOW);
      assert.equal(result.z, 22, 'Card depth must remain at 22 and not jump forward');
    });

    it('enforces the +5 threshold: does not sink for minor perturbations <= 5', () => {
      const borderlineCard: CosmicCard = {
        id: 'border-1',
        title: 'Borderline Depth Card',
        content: 'Content',
        createdAt: new Date(NOW - 20 * DAY_MS).toISOString(), // ~48..62
        origin: 'local',
        type: 'thought',
        x: 0, y: 0, rotation: 0,
        z: 58,
        spawnTime: NOW - 30 * MINUTE_MS,
        auraTraces: [{ searchId: 's1', hitTime: NOW - 30 * MINUTE_MS, duration: 15 * MINUTE_MS }],
      };

      const { shouldSink } = calculateGravitationalFallbackZ(borderlineCard, NOW);
      assert.equal(shouldSink, false, 'Must not sink when delta <= 5');
    });
  });

  describe('4. Batch Window Focus Processing (processWindowFocusGravitationalFallback)', () => {
    it('processes active stage cards correctly according to all 4 phases', () => {
      const cards: CosmicCard[] = [
        // 1. Permanent card with ACTIVE light (hit 5m ago, duration 15m) -> strictly stays at z = 15
        {
          id: 'card-light-on',
          title: 'Light On',
          content: 'c1',
          createdAt: new Date(NOW - 100 * DAY_MS).toISOString(),
          origin: 'local',
          z: 15,
          spawnTime: NOW - 5 * MINUTE_MS,
          auraTraces: [{ searchId: 's1', color: 'hsl(180, 95%, 62%)', hitTime: NOW - 5 * MINUTE_MS, duration: 15 * MINUTE_MS }],
          x: 0, y: 0, rotation: 0
        },
        // 2. Permanent card in COOLDOWN buffer (hit 18m ago, duration 15m -> ext 3m ago < 10m) -> stays at z = 15
        {
          id: 'card-cooldown',
          title: 'Cooldown Active',
          content: 'c2',
          createdAt: new Date(NOW - 100 * DAY_MS).toISOString(),
          origin: 'local',
          z: 15,
          spawnTime: NOW - 18 * MINUTE_MS,
          auraTraces: [{ searchId: 's1', color: 'hsl(180, 95%, 62%)', hitTime: NOW - 18 * MINUTE_MS, duration: 15 * MINUTE_MS }],
          x: 0, y: 0, rotation: 0
        },
        // 3. Permanent card with expired cooldown (hit 30m ago, ext 15m ago > 10m) -> SINKS to intrinsic depth
        {
          id: 'card-sink',
          title: 'Should Sink',
          content: 'c3',
          createdAt: new Date(NOW - 90 * DAY_MS).toISOString(),
          origin: 'local',
          z: 10,
          spawnTime: NOW - 30 * MINUTE_MS,
          auraTraces: [{ searchId: 's1', color: 'hsl(100, 95%, 62%)', hitTime: NOW - 30 * MINUTE_MS, duration: 15 * MINUTE_MS }],
          isSearchResult: true,
          x: 0, y: 0, rotation: 0
        },
        // 4. Permanent card with expired cooldown, but created today -> stays at z = 25 (did not jump forward)
        {
          id: 'card-today',
          title: 'Today Note',
          content: 'c4',
          createdAt: new Date(NOW - 1 * MINUTE_MS).toISOString(),
          origin: 'local',
          z: 25,
          spawnTime: NOW - 30 * MINUTE_MS,
          auraTraces: [{ searchId: 's1', hitTime: NOW - 30 * MINUTE_MS, duration: 15 * MINUTE_MS }],
          x: 0, y: 0, rotation: 0
        },
        // 5. Currently focused card -> strictly untouched
        {
          id: 'card-focused',
          title: 'Focused',
          content: 'c5',
          createdAt: new Date(NOW - 100 * DAY_MS).toISOString(),
          origin: 'local',
          z: 10,
          spawnTime: NOW - 30 * MINUTE_MS,
          auraTraces: [{ searchId: 's1', hitTime: NOW - 30 * MINUTE_MS, duration: 15 * MINUTE_MS }],
          x: 0, y: 0, rotation: 0
        },
        // 6. Temporary card (origin: 'bugu') -> untouched by permanent fallback engine
        {
          id: 'card-bugu',
          title: 'Bugu Discovery',
          content: 'c6',
          origin: 'bugu',
          type: 'babel',
          z: 10,
          spawnTime: NOW - 30 * MINUTE_MS,
          x: 0, y: 0, rotation: 0
        }
      ];

      const processed = processWindowFocusGravitationalFallback(cards, NOW, 'card-focused');

      // 1. Light-on stays at z = 15
      assert.equal(processed.find(c => c.id === 'card-light-on')?.z, 15);

      // 2. Cooldown stays at z = 15
      assert.equal(processed.find(c => c.id === 'card-cooldown')?.z, 15);

      // 3. Expired old card sinks to > 65
      const sunk = processed.find(c => c.id === 'card-sink');
      assert.ok(sunk && sunk.z >= 66 && sunk.z <= 78, `sunk.z (${sunk?.z}) should be in 66~78`);

      // 4. Today card stays at z = 25 (did not jump forward)
      assert.equal(processed.find(c => c.id === 'card-today')?.z, 25);

      // 5. Focused card stays at z = 10
      assert.equal(processed.find(c => c.id === 'card-focused')?.z, 10);

      // 6. Bugu temporary card stays untouched by permanent fallback engine
      assert.equal(processed.find(c => c.id === 'card-bugu')?.z, 10);
    });
  });

  describe('5. Search Elevation Integration', () => {
    it('sets duration: DEFAULT_SEARCH_AURA_DURATION on new search aura traces', () => {
      const trace = createAuraTrace('s1', 'hsl(200, 95%, 62%)', NOW);
      assert.equal(trace.duration, 15 * 60 * 1000);
      assert.equal(trace.hitTime, NOW);
    });

    it('injects traces with duration in transformSearchResults', () => {
      const rawResults = [{ id: 'new-1', title: 'Newly Found', content: 'hello' }];
      const transformed = transformSearchResults(rawResults, [[]], 'search_1', 'hsl(200, 95%, 62%)', NOW);
      assert.equal(transformed.length, 1);
      assert.equal(transformed[0].auraTraces?.[0].duration, 15 * 60 * 1000);
    });
  });
});
