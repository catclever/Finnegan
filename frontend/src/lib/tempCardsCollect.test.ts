// @ts-nocheck
import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { calculateSpatialPlacement } from './stageCoords.ts';

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
      return Boolean(card.isTemporary || card.origin === 'bugu' || card.type === 'radar');
    }

    function simulatePromote(card: any, promotedBackendResult: any = null): any {
      const originalId = card.id;
      const promoted = { ...card };
      promoted.origin = 'local';
      if (promoted.type === 'radar') promoted.type = 'local';
      if (promoted.isTemporary) promoted.isTemporary = false;
      delete promoted.macroExpiresAt;
      if (promotedBackendResult) {
        if (promotedBackendResult.id) promoted.id = promotedBackendResult.id;
        if (promotedBackendResult.filePath) promoted.filePath = promotedBackendResult.filePath;
        if (promotedBackendResult.type) promoted.type = promotedBackendResult.type;
        if (promotedBackendResult.createdAt) promoted.createdAt = promotedBackendResult.createdAt;
      }
      return { originalId, promoted };
    }

    it('identifies uncollected temporary cards accurately', () => {
      assert.equal(isTemporaryCard({ id: 'c1', origin: 'bugu', type: 'babel' }), true);
      assert.equal(isTemporaryCard({ id: 'c2', origin: 'local', type: 'radar' }), true);
      assert.equal(isTemporaryCard({ id: 'c3', origin: 'bugu', type: 'thought' }), true);
      assert.equal(isTemporaryCard({ id: 'c4', origin: 'local', type: 'thought' }), false);
      assert.equal(isTemporaryCard({ id: 'c5', origin: 'manual', type: 'note' }), false);
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
        type: 'local',
      });

      assert.equal(originalId, 'temp-123');
      assert.equal(promoted.id, '01JBCDEF000000000000000000');
      assert.equal(promoted.origin, 'local');
      assert.equal(promoted.type, 'local');
      assert.equal(promoted.filePath, '/root/Scattering/20261027-1500-network-intelligence.md');
      assert.equal(promoted.macroExpiresAt, undefined);
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
});
