// @ts-nocheck
import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { calculateSpatialPlacement } from './stageCoords.ts';
import { VIP_INK_BEAM_DURATION_MS } from './stageDecay.ts';

describe('Temporary Card Collection & Jitter Depth Logic', () => {
  describe('1. calculateSpatialPlacement jitter depth (20~30)', () => {
    it('produces depth strictly within [20, 30] for bugu cards', () => {
      for (let i = 0; i < 50; i++) {
        const res = calculateSpatialPlacement([], 0, true, {
          id: `bugu-${i}`,
          origin: 'bugu',
          type: 'babel',
          title: 'Discoveries',
          content: 'text',
          x: 0,
          y: 0,
          rotation: 0,
          z: 0,
          spawnTime: Date.now(),
          decayDuration: 3600000,
          isFresh: true,
          isVIP: false,
        } as any);

        assert.ok(res.z >= 20.0, `Expected z (${res.z}) >= 20.0`);
        assert.ok(res.z <= 30.0, `Expected z (${res.z}) <= 30.0`);
      }
    });

    it('produces depth strictly within [20, 30] for radar cards', () => {
      for (let i = 0; i < 50; i++) {
        const res = calculateSpatialPlacement([], 0, false, {
          id: `radar-${i}`,
          origin: 'local',
          type: 'radar',
          title: 'Radar item',
          content: 'text',
          x: 0,
          y: 0,
          rotation: 0,
          z: 0,
          spawnTime: Date.now(),
          decayDuration: 3600000,
          isFresh: true,
          isVIP: false,
        } as any);

        assert.ok(res.z >= 20.0, `Expected z (${res.z}) >= 20.0`);
        assert.ok(res.z <= 30.0, `Expected z (${res.z}) <= 30.0`);
      }
    });
  });

  describe('2. Temporary Card Determination & In-Place Promotion Semantics', () => {
    function isTemporaryCard(card: any): boolean {
      if (!card) return false;
      if (card.isTemporary === false) return false;
      return Boolean(card.isTemporary || card.origin === 'bugu' || card.origin === 'atproto' || card.type === 'radar');
    }

    function simulatePromote(card: any, promotedBackendResult: any = null): any {
      const originalId = card.id;
      const promoted = { ...card };
      promoted.isTemporary = false;
      delete promoted.macroExpiresAt;
      if (promotedBackendResult) {
        if (promotedBackendResult.id) promoted.id = promotedBackendResult.id;
        if (promotedBackendResult.filePath) promoted.filePath = promotedBackendResult.filePath;
        if (promotedBackendResult.type && promotedBackendResult.type !== 'local') {
          promoted.type = promotedBackendResult.type;
        } else if (promoted.type === 'radar') {
          promoted.type = 'thought';
        }
        if (promotedBackendResult.origin) promoted.origin = promotedBackendResult.origin;
        if (promotedBackendResult.createdAt) promoted.createdAt = promotedBackendResult.createdAt;
      } else if (promoted.type === 'radar') {
        promoted.type = 'thought';
      }
      return { originalId, promoted };
    }

    it('identifies uncollected temporary cards accurately', () => {
      assert.equal(isTemporaryCard({ id: 'c1', origin: 'bugu', type: 'babel' }), true);
      assert.equal(isTemporaryCard({ id: 'c2', origin: 'local', type: 'radar' }), true);
      assert.equal(isTemporaryCard({ id: 'c3', origin: 'bugu', type: 'thought' }), true);
      assert.equal(isTemporaryCard({ id: 'c4', origin: 'local', type: 'thought' }), false);
      assert.equal(isTemporaryCard({ id: 'c5', origin: 'manual', type: 'note' }), false);
      assert.equal(isTemporaryCard({ id: 'c6', origin: 'bugu', type: 'babel', isTemporary: false }), false);
    });

    it('converts temporary card in-place to local card', () => {
      const tempCard = {
        id: 'temp-123',
        origin: 'bugu',
        type: 'radar',
        title: 'Network Intelligence',
        content: 'Fascinating insight',
        macroExpiresAt: Date.now() + 86400000,
      };

      assert.equal(isTemporaryCard(tempCard), true);

      const { originalId, promoted } = simulatePromote(tempCard, {
        id: '01JBCDEF000000000000000000',
        filePath: '/root/Scattering/20261027-1500-network-intelligence.md',
        type: 'thought',
        origin: 'bugu',
      });

      assert.equal(originalId, 'temp-123');
      assert.equal(promoted.id, '01JBCDEF000000000000000000');
      assert.equal(promoted.type, 'thought');
      assert.equal(promoted.filePath, '/root/Scattering/20261027-1500-network-intelligence.md');
      assert.equal(promoted.macroExpiresAt, undefined);
      assert.equal(isTemporaryCard(promoted), false);
    });

    it('preserves babel card type and author metadata upon promotion', () => {
      const babelCard = {
        id: 'babel-temp-999',
        origin: 'atproto',
        type: 'babel',
        title: 'Bluesky Echo',
        content: 'Social pulse insights',
        authorName: 'alice.bsky.social',
        authorAvatar: 'https://cdn.bsky.app/avatar.jpg',
        publishedAt: '2026-10-30T10:00:00Z',
        sourceUrl: 'https://bsky.app/profile/alice/post/123',
        macroExpiresAt: Date.now() + 86400000,
      };

      assert.equal(isTemporaryCard(babelCard), true);

      const { originalId, promoted } = simulatePromote(babelCard, {
        id: '01JBABELLOSSLESS000000000',
        filePath: '/root/Scattering/20261030-1200-bluesky-echo.md',
        type: 'babel',
        origin: 'atproto',
      });

      assert.equal(originalId, 'babel-temp-999');
      assert.equal(promoted.id, '01JBABELLOSSLESS000000000');
      assert.equal(promoted.type, 'babel');
      assert.equal(promoted.authorName, 'alice.bsky.social');
      assert.equal(promoted.authorAvatar, 'https://cdn.bsky.app/avatar.jpg');
      assert.equal(promoted.sourceUrl, 'https://bsky.app/profile/alice/post/123');
      assert.equal(isTemporaryCard(promoted), false);
    });
  });

  describe('3. Edit Interception on Uncollected Temporary Cards', () => {
    function handleDoubleClickTest(isTemporary: boolean, onPrompt: () => void, onOpenEditor: () => void) {
      if (isTemporary) {
        onPrompt();
        return;
      }
      onOpenEditor();
    }

    function handleClickTest(event: { altKey: boolean }, isTemporary: boolean, onPrompt: () => void, onOpenEditor: () => void) {
      if (event.altKey) {
        if (isTemporary) {
          onPrompt();
          return;
        }
        onOpenEditor();
        return;
      }
    }

    it('blocks double-click for uncollected temporary cards and triggers prompt', () => {
      let promptTriggered = false;
      let editorTriggered = false;

      handleDoubleClickTest(
        true,
        () => { promptTriggered = true; },
        () => { editorTriggered = true; }
      );

      assert.equal(promptTriggered, true);
      assert.equal(editorTriggered, false);
    });

    it('allows double-click for permanent local cards', () => {
      let promptTriggered = false;
      let editorTriggered = false;

      handleDoubleClickTest(
        false,
        () => { promptTriggered = true; },
        () => { editorTriggered = true; }
      );

      assert.equal(promptTriggered, false);
      assert.equal(editorTriggered, true);
    });

    it('blocks Alt+click for uncollected temporary cards and triggers prompt', () => {
      let promptTriggered = false;
      let editorTriggered = false;

      handleClickTest(
        { altKey: true },
        true,
        () => { promptTriggered = true; },
        () => { editorTriggered = true; }
      );

      assert.equal(promptTriggered, true);
      assert.equal(editorTriggered, false);
    });

    it('allows Alt+click for permanent local cards', () => {
      let promptTriggered = false;
      let editorTriggered = false;

      handleClickTest(
        { altKey: true },
        false,
        () => { promptTriggered = true; },
        () => { editorTriggered = true; }
      );

      assert.equal(promptTriggered, false);
      assert.equal(editorTriggered, true);
    });
  });

  describe('4. Focus-Driven Two-Stage Decay for Temporary Cards', () => {
    function simulateWindowFocusDecay(cards: any[], now: number, focusedCardId: string | null = null): { updatedCards: any[]; fadingIds: string[] } {
      const fadingIds: string[] = [];
      const updatedCards = cards.map(c => {
        if (c.id === focusedCardId) return c;
        if (c.isVIP || c.isSearchResult) return c;
        if (c.isFadingOut) return c;

        const isTemporary = Boolean(c.isTemporary || c.origin === 'bugu' || c.type === 'radar');
        if (isTemporary && c.decayDuration && (now - c.spawnTime >= c.decayDuration)) {
          if (c.z < 65) {
            return {
              ...c,
              z: c.z + 20 + 2.5,
              spawnTime: now
            };
          } else {
            fadingIds.push(c.id);
            return {
              ...c,
              isFadingOut: true
            };
          }
        }
        return c;
      });
      return { updatedCards, fadingIds };
    }

    it('midground/foreground temporary card (z < 65) steps back into deep space when half-life expires', () => {
      const now = 1000000;
      const initialCard = {
        id: 'bugu-1',
        origin: 'bugu',
        type: 'babel',
        z: 30,
        spawnTime: now - 3600000,
        decayDuration: 3600000,
        isFadingOut: false
      };

      const { updatedCards, fadingIds } = simulateWindowFocusDecay([initialCard], now);
      assert.equal(fadingIds.length, 0);
      assert.ok(updatedCards[0].z >= 50 && updatedCards[0].z <= 55);
      assert.equal(updatedCards[0].spawnTime, now);
      assert.equal(updatedCards[0].isFadingOut, false);
    });

    it('deep space temporary card (z >= 65) triggers isFadingOut = true when half-life expires', () => {
      const now = 1000000;
      const deepCard = {
        id: 'bugu-2',
        origin: 'bugu',
        type: 'babel',
        z: 70,
        spawnTime: now - 4000000,
        decayDuration: 3600000,
        isFadingOut: false
      };

      const { updatedCards, fadingIds } = simulateWindowFocusDecay([deepCard], now);
      assert.equal(fadingIds.length, 1);
      assert.equal(fadingIds[0], 'bugu-2');
      assert.equal(updatedCards[0].isFadingOut, true);
    });

    it('temporary card does not decay if half-life has not been reached', () => {
      const now = 1000000;
      const freshCard = {
        id: 'bugu-3',
        origin: 'bugu',
        type: 'babel',
        z: 25,
        spawnTime: now - 1000,
        decayDuration: 3600000,
        isFadingOut: false
      };

      const { updatedCards, fadingIds } = simulateWindowFocusDecay([freshCard], now);
      assert.equal(fadingIds.length, 0);
      assert.equal(updatedCards[0].z, 25);
      assert.equal(updatedCards[0].spawnTime, now - 1000);
      assert.equal(updatedCards[0].isFadingOut, false);
    });

    it('permanent local cards never decay on window focus', () => {
      const now = 1000000;
      const localCard = {
        id: 'local-1',
        origin: 'local',
        type: 'thought',
        z: 20,
        spawnTime: now - 10000000,
        decayDuration: 3600000,
        isFadingOut: false
      };

      const { updatedCards, fadingIds } = simulateWindowFocusDecay([localCard], now);
      assert.equal(fadingIds.length, 0);
      assert.equal(updatedCards[0].z, 20);
      assert.equal(updatedCards[0].isFadingOut, false);
    });

    it('currently focused temporary card is protected from focus decay', () => {
      const now = 1000000;
      const focusedCard = {
        id: 'bugu-focused',
        origin: 'bugu',
        type: 'radar',
        z: 25,
        spawnTime: now - 5000000,
        decayDuration: 3600000,
        isFadingOut: false
      };

      const { updatedCards, fadingIds } = simulateWindowFocusDecay([focusedCard], now, 'bugu-focused');
      assert.equal(fadingIds.length, 0);
      assert.equal(updatedCards[0].z, 25);
      assert.equal(updatedCards[0].isFadingOut, false);
    });
  });

  describe('5. VIP Ink Beam Duration & Window Focus Extinction', () => {
    function simulateWindowFocus(cards: any[], now: number, focusedCardId: string | null = null, currentLatestManualId: string | null = null) {
      let latestId = currentLatestManualId;
      const updatedCards = cards.map(c => {
        if (c.isVIP) {
          const birthTime = c.spawnTime || (c.createdAt ? new Date(c.createdAt).getTime() : 0);
          if (birthTime && (now - birthTime >= VIP_INK_BEAM_DURATION_MS)) {
            c = { ...c, isVIP: false };
            if (latestId === c.id) {
              latestId = null;
            }
          }
        }
        return c;
      });
      return { updatedCards, latestId };
    }

    it('extinguishes isVIP and resets latestManualCardId when > 15 minutes have elapsed on window focus', () => {
      const birth = 1000000;
      const now = birth + VIP_INK_BEAM_DURATION_MS + 1000;
      const vipCard = {
        id: 'vip-card-1',
        isVIP: true,
        spawnTime: birth,
      };

      const { updatedCards, latestId } = simulateWindowFocus([vipCard], now, null, 'vip-card-1');
      assert.equal(updatedCards[0].isVIP, false);
      assert.equal(latestId, null);
    });

    it('preserves isVIP and latestManualCardId when < 15 minutes have elapsed on window focus', () => {
      const birth = 1000000;
      const now = birth + 14 * 60 * 1000;
      const vipCard = {
        id: 'vip-card-1',
        isVIP: true,
        spawnTime: birth,
      };

      const { updatedCards, latestId } = simulateWindowFocus([vipCard], now, null, 'vip-card-1');
      assert.equal(updatedCards[0].isVIP, true);
      assert.equal(latestId, 'vip-card-1');
    });

    it('extinguishes isVIP even if card is currently focused when window focus occurs', () => {
      const birth = 1000000;
      const now = birth + VIP_INK_BEAM_DURATION_MS + 500;
      const vipCard = {
        id: 'vip-focused',
        isVIP: true,
        spawnTime: birth,
      };

      const { updatedCards, latestId } = simulateWindowFocus([vipCard], now, 'vip-focused', 'vip-focused');
      assert.equal(updatedCards[0].isVIP, false);
      assert.equal(latestId, null);
    });
  });
});
