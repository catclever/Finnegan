import type { CosmicCard } from '../../types/card';
import type { BuguSession, BuguBeacon } from './types';
import { createBabelSessionController } from '../babel/babelSession';

export interface BuguSessionCallbacks {
  getCards: () => CosmicCard[];
  getMemoryPool: () => CosmicCard[];
  getScrollExpansion: () => number;
  onBeaconChange: (beacon: BuguBeacon | null) => void;
  onSessionChange: (session: BuguSession | null) => void;
  onUpdateStage: (cards: CosmicCard[], memoryPool?: CosmicCard[]) => void;
  toCosmicCard: (data: any, type: string, source: string, ttl: number) => CosmicCard;
  stageCandidate: (card: CosmicCard, isVisible: boolean, scrollExp: number, cards: CosmicCard[], memPool: CosmicCard[], srcId?: string, yOverride?: number) => any;
}

export function createBuguSessionController(callbacks: BuguSessionCallbacks) {
  let activeBuguBeacon: BuguBeacon | null = null;
  let buguBeaconTimer: any = null;

  function updateBeacon(b: BuguBeacon | null) {
    activeBuguBeacon = b;
    callbacks.onBeaconChange(b);
  }

  function finishBuguBeacon() {
    if (buguBeaconTimer) clearTimeout(buguBeaconTimer);
    if (!activeBuguBeacon || activeBuguBeacon.status === 'fading') return;
    const sid = activeBuguBeacon.sourceCardId;
    updateBeacon({ sourceCardId: sid, status: 'fading' });
    buguBeaconTimer = setTimeout(() => {
      if (activeBuguBeacon?.sourceCardId === sid) updateBeacon(null);
    }, 2500);
  }

  function resetBeaconSafetyTimer(sourceCardId: string) {
    if (buguBeaconTimer) clearTimeout(buguBeaconTimer);
    buguBeaconTimer = setTimeout(() => {
      if (activeBuguBeacon?.sourceCardId === sourceCardId) finishBuguBeacon();
    }, 90000);
  }

  // Universal Babel Session manages queuing and stage placement
  const babelSession = createBabelSessionController({
    getCards: callbacks.getCards,
    getMemoryPool: callbacks.getMemoryPool,
    getScrollExpansion: callbacks.getScrollExpansion,
    onUpdateStage: callbacks.onUpdateStage,
    toCosmicCard: callbacks.toCosmicCard,
    stageCandidate: callbacks.stageCandidate,
    onQueueEmpty: () => {
      finishBuguBeacon();
    },
  });

  function startRadar(sourceCardId: string, cards: CosmicCard[], memoryPool: CosmicCard[], localVaultPool: CosmicCard[]) {
    babelSession.reset();

    // The mother card has already been guaranteed into active viewport by global ensureCardStagedInViewport
    let nextCards = cards.map(c => c.id === sourceCardId
      ? { ...c, isVIP: true, decayDuration: Math.max(c.decayDuration, 12 * 3600 * 1000) }
      : c
    );

    callbacks.onUpdateStage(nextCards, memoryPool);
    updateBeacon({ sourceCardId, status: 'scanning' });
    resetBeaconSafetyTimer(sourceCardId);
  }

  function resetBeaconSafetyTimerExternal() {
    if (activeBuguBeacon) {
      resetBeaconSafetyTimer(activeBuguBeacon.sourceCardId);
    }
  }

  function enqueueBabelCard(c: any, origin: string = 'bugu') {
    if (activeBuguBeacon) resetBeaconSafetyTimerExternal();
    babelSession.enqueue(c, origin);
  }

  function finishRadar() {
    babelSession.finishSession();
  }

  return {
    startRadar,
    finishRadar,
    enqueueBabelCard,
    getBabelSession: () => babelSession,
    resetBeaconSafetyTimer: resetBeaconSafetyTimerExternal,
    destroy: () => {
      if (buguBeaconTimer) clearTimeout(buguBeaconTimer);
      babelSession.destroy();
    }
  };
}
