import type { BskyBeacon } from './types';

export interface BskySessionCallbacks {
  onBeaconChange: (beacon: BskyBeacon | null) => void;
}

export function createBskySessionController(callbacks: BskySessionCallbacks) {
  let activeBskyBeacon: BskyBeacon | null = null;
  let bskyBeaconTimer: any = null;
  let fadeTimer: any = null;
  let cleanupTimer: any = null;

  function updateBeacon(b: BskyBeacon | null) {
    activeBskyBeacon = b;
    callbacks.onBeaconChange(b);
  }

  function startEcho(sourceCardId?: string | null) {
    destroyTimers();
    const initialCount = Math.floor(Math.random() * 4) + 2;
    updateBeacon({
      sourceCardId: sourceCardId || null,
      status: 'scanning',
      butterflyCount: initialCount,
    });
    bskyBeaconTimer = setTimeout(() => {
      finishBeacon();
    }, 35000);
  }

  function incrementButterfly(forceIncrement = false) {
    if (activeBskyBeacon && activeBskyBeacon.status !== 'fading' && activeBskyBeacon.status !== 'completed') {
      if ((forceIncrement || Math.random() < 0.8) && activeBskyBeacon.butterflyCount < 8) {
        updateBeacon({
          ...activeBskyBeacon,
          butterflyCount: activeBskyBeacon.butterflyCount + 1,
        });
      }
    }
  }

  function finishBeacon() {
    if (bskyBeaconTimer) {
      clearTimeout(bskyBeaconTimer);
      bskyBeaconTimer = null;
    }
    if (activeBskyBeacon && activeBskyBeacon.status !== 'fading' && activeBskyBeacon.status !== 'completed') {
      updateBeacon({ ...activeBskyBeacon, status: 'completed' });
      fadeTimer = setTimeout(() => {
        if (activeBskyBeacon?.status === 'completed') {
          updateBeacon({ ...activeBskyBeacon, status: 'fading' });
          cleanupTimer = setTimeout(() => {
            if (activeBskyBeacon?.status === 'fading') {
              updateBeacon(null);
            }
          }, 2500);
        }
      }, 1800);
    }
  }

  function destroyTimers() {
    if (bskyBeaconTimer) {
      clearTimeout(bskyBeaconTimer);
      bskyBeaconTimer = null;
    }
    if (fadeTimer) {
      clearTimeout(fadeTimer);
      fadeTimer = null;
    }
    if (cleanupTimer) {
      clearTimeout(cleanupTimer);
      cleanupTimer = null;
    }
  }

  function destroy() {
    destroyTimers();
    updateBeacon(null);
  }

  return {
    startEcho,
    incrementButterfly,
    finishBeacon,
    destroy,
    getBeacon: () => activeBskyBeacon,
  };
}
