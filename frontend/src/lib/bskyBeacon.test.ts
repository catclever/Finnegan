import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import type { BskyBeacon } from '../modules/atproto/types';

describe('ATProto Butterfly Beacon Lifecycle & Direct Search Engine', () => {
  describe('Beacon State Machine', () => {
    function createBeaconController() {
      let activeBskyBeacon: BskyBeacon | null = null;
      let bskyBeaconTimer: any = null;

      function startBskyEcho(sourceCardId?: string | null) {
        if (bskyBeaconTimer) {
          clearTimeout(bskyBeaconTimer);
          bskyBeaconTimer = null;
        }
        const initialCount = Math.floor(Math.random() * 4) + 2;
        activeBskyBeacon = {
          sourceCardId: sourceCardId || null,
          status: 'scanning',
          butterflyCount: initialCount,
        };
        bskyBeaconTimer = setTimeout(() => {
          finishBskyBeacon();
        }, 35000);
      }

      function incrementBskyButterfly(forceIncrement = false) {
        if (activeBskyBeacon && activeBskyBeacon.status !== 'fading' && activeBskyBeacon.status !== 'completed') {
          if ((forceIncrement || Math.random() < 0.8) && activeBskyBeacon.butterflyCount < 8) {
            activeBskyBeacon = {
              ...activeBskyBeacon,
              butterflyCount: activeBskyBeacon.butterflyCount + 1,
            };
          }
        }
      }

      function finishBskyBeacon() {
        if (bskyBeaconTimer) {
          clearTimeout(bskyBeaconTimer);
          bskyBeaconTimer = null;
        }
        if (activeBskyBeacon && activeBskyBeacon.status !== 'fading' && activeBskyBeacon.status !== 'completed') {
          activeBskyBeacon = { ...activeBskyBeacon, status: 'completed' };
        }
      }

      return {
        getBeacon: () => activeBskyBeacon,
        getTimer: () => bskyBeaconTimer,
        startBskyEcho,
        incrementBskyButterfly,
        finishBskyBeacon,
        transitionToFading: () => {
          if (activeBskyBeacon) {
            activeBskyBeacon = { ...activeBskyBeacon, status: 'fading' };
          }
        },
        resetTimer: () => {
          if (bskyBeaconTimer) clearTimeout(bskyBeaconTimer);
        }
      };
    }

    test('startBskyEcho initializes beacon with scanning status and 2-5 butterflies', () => {
      const ctrl = createBeaconController();
      ctrl.startBskyEcho('card-123');
      const beacon = ctrl.getBeacon();
      assert.ok(beacon);
      assert.equal(beacon.sourceCardId, 'card-123');
      assert.equal(beacon.status, 'scanning');
      assert.ok(beacon.butterflyCount >= 2 && beacon.butterflyCount <= 5);
      ctrl.resetTimer();
    });

    test('startBskyEcho supports direct search with null/undefined sourceCardId', () => {
      const ctrl = createBeaconController();
      ctrl.startBskyEcho(null);
      const beacon = ctrl.getBeacon();
      assert.ok(beacon);
      assert.equal(beacon.sourceCardId, null);
      assert.equal(beacon.status, 'scanning');
      ctrl.resetTimer();
    });

    test('incrementBskyButterfly increments butterfly count up to max 8', () => {
      const ctrl = createBeaconController();
      ctrl.startBskyEcho('card-test');
      for (let i = 0; i < 20; i++) {
        ctrl.incrementBskyButterfly(true);
      }
      const beacon = ctrl.getBeacon();
      assert.ok(beacon);
      assert.equal(beacon.butterflyCount, 8);
      ctrl.resetTimer();
    });

    test('finishBskyBeacon transitions beacon to completed status to stop rotation', () => {
      const ctrl = createBeaconController();
      ctrl.startBskyEcho('card-test');
      ctrl.finishBskyBeacon();
      const beacon = ctrl.getBeacon();
      assert.ok(beacon);
      assert.equal(beacon.status, 'completed');
      assert.equal(ctrl.getTimer(), null);
    });

    test('incrementBskyButterfly does nothing when beacon is in completed status', () => {
      const ctrl = createBeaconController();
      ctrl.startBskyEcho('card-test');
      ctrl.finishBskyBeacon();
      const countBefore = ctrl.getBeacon()?.butterflyCount;
      ctrl.incrementBskyButterfly(true);
      assert.equal(ctrl.getBeacon()?.butterflyCount, countBefore);
    });

    test('beacon transitions from completed to fading for graceful scatter', () => {
      const ctrl = createBeaconController();
      ctrl.startBskyEcho('card-test');
      ctrl.finishBskyBeacon();
      assert.equal(ctrl.getBeacon()?.status, 'completed');
      ctrl.transitionToFading();
      assert.equal(ctrl.getBeacon()?.status, 'fading');
    });
  });

  describe('Direct Search Keyboard Navigation Logic', () => {
    function computeKeydownNavigation(opts: {
      key: 'ArrowUp' | 'ArrowDown' | 'Enter';
      commandId: string;
      omniText: string;
      highlightedCardIndex: number;
      candidatesLength: number;
    }) {
      const isBskyDirectSearch = opts.commandId === 'atproto:echo' && Boolean(opts.omniText.trim());
      let nextIndex = opts.highlightedCardIndex;
      let executedDirectSearch = false;
      let lockedCardIndex: number | null = null;

      if (opts.key === 'ArrowDown') {
        if (isBskyDirectSearch && opts.highlightedCardIndex === -1) {
          if (opts.candidatesLength > 0) {
            nextIndex = 0;
          }
        } else if (opts.candidatesLength > 0) {
          nextIndex = Math.min(opts.highlightedCardIndex + 1, opts.candidatesLength - 1);
        }
      } else if (opts.key === 'ArrowUp') {
        if (isBskyDirectSearch) {
          if (opts.highlightedCardIndex <= 0) {
            nextIndex = -1;
          } else {
            nextIndex = opts.highlightedCardIndex - 1;
          }
        } else if (opts.candidatesLength > 0) {
          nextIndex = Math.max(0, opts.highlightedCardIndex - 1);
        }
      } else if (opts.key === 'Enter') {
        if (isBskyDirectSearch && (opts.highlightedCardIndex === -1 || opts.candidatesLength === 0)) {
          executedDirectSearch = true;
        } else if (opts.candidatesLength > 0 && opts.highlightedCardIndex >= 0 && opts.highlightedCardIndex < opts.candidatesLength) {
          lockedCardIndex = opts.highlightedCardIndex;
        }
      }

      return { nextIndex, executedDirectSearch, lockedCardIndex };
    }

    test('ArrowUp from 0 navigates to -1 for atproto:echo direct search', () => {
      const res = computeKeydownNavigation({
        key: 'ArrowUp',
        commandId: 'atproto:echo',
        omniText: 'AI agents',
        highlightedCardIndex: 0,
        candidatesLength: 3,
      });
      assert.equal(res.nextIndex, -1);
    });

    test('ArrowDown from -1 navigates back to 0 when candidates exist', () => {
      const res = computeKeydownNavigation({
        key: 'ArrowDown',
        commandId: 'atproto:echo',
        omniText: 'AI agents',
        highlightedCardIndex: -1,
        candidatesLength: 3,
      });
      assert.equal(res.nextIndex, 0);
    });

    test('Enter at -1 executes direct search with omniText', () => {
      const res = computeKeydownNavigation({
        key: 'Enter',
        commandId: 'atproto:echo',
        omniText: 'AI agents',
        highlightedCardIndex: -1,
        candidatesLength: 3,
      });
      assert.equal(res.executedDirectSearch, true);
      assert.equal(res.lockedCardIndex, null);
    });

    test('Enter when candidates length is 0 executes direct search directly', () => {
      const res = computeKeydownNavigation({
        key: 'Enter',
        commandId: 'atproto:echo',
        omniText: 'novel keywords not in vault',
        highlightedCardIndex: 0,
        candidatesLength: 0,
      });
      assert.equal(res.executedDirectSearch, true);
    });

    test('Enter on normal candidate locks candidate card', () => {
      const res = computeKeydownNavigation({
        key: 'Enter',
        commandId: 'atproto:echo',
        omniText: 'test',
        highlightedCardIndex: 1,
        candidatesLength: 3,
      });
      assert.equal(res.executedDirectSearch, false);
      assert.equal(res.lockedCardIndex, 1);
    });
  });
});
