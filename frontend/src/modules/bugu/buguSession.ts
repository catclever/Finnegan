import type { CosmicCard } from '../../types/card';
import type { BuguSession, BuguBeacon } from './types';
import { MACRO_TTL_MS } from '../../lib/stageDecay';

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
  let babelQueue: any[] = [];
  let babelQueueTimer: any = null;
  let radarBackendFinished = false;
  let buguBabelCount = 0;

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

  function startRadar(sourceCardId: string, cards: CosmicCard[], memoryPool: CosmicCard[], localVaultPool: CosmicCard[]) {
    radarBackendFinished = false;
    buguBabelCount = 0;

    // The mother card has already been guaranteed into active viewport by global ensureCardStagedInViewport
    let nextCards = cards.map(c => c.id === sourceCardId
      ? { ...c, isVIP: true, decayDuration: Math.max(c.decayDuration, 12 * 3600 * 1000) }
      : c
    );

    callbacks.onUpdateStage(nextCards, memoryPool);
    updateBeacon({ sourceCardId, status: 'scanning' });
    resetBeaconSafetyTimer(sourceCardId);
  }

  function stageOneBabelCard(c: any) {
    const newCard = callbacks.toCosmicCard(c, 'babel', 'bugu', MACRO_TTL_MS);
    const substantive = (newCard.content || newCard.originalText || newCard.translatedText || '').trim();
    if (!substantive) return;

    buguBabelCount++;
    const scrollExpansion = callbacks.getScrollExpansion();
    const viewportY = Math.round(scrollExpansion * 300);
    const targetY = buguBabelCount <= 2 ? viewportY : viewportY + 480;

    let cards = callbacks.getCards();
    let memoryPool = callbacks.getMemoryPool();

    memoryPool = [newCard, ...memoryPool.filter(m => m.id !== newCard.id)];
    const { stagedCard, updatedActiveCards, updatedMemoryPool } = callbacks.stageCandidate(
      newCard, true, scrollExpansion, cards, memoryPool, activeBuguBeacon?.sourceCardId, targetY
    );

    if (updatedMemoryPool) memoryPool = updatedMemoryPool;
    cards = [...updatedActiveCards.filter((card: any) => card.id !== stagedCard.id), stagedCard];

    callbacks.onUpdateStage(cards, memoryPool);
  }

  function processBabelQueue() {
    if (babelQueueTimer) return;
    if (babelQueue.length === 0) {
      if (radarBackendFinished) finishBuguBeacon();
      return;
    }
    stageOneBabelCard(babelQueue.shift());
    if (babelQueue.length === 0 && radarBackendFinished) {
      finishBuguBeacon();
      return;
    }
    babelQueueTimer = setTimeout(() => {
      babelQueueTimer = null;
      processBabelQueue();
    }, 850);
  }

  function resetBeaconSafetyTimerExternal() {
    if (activeBuguBeacon) {
      resetBeaconSafetyTimer(activeBuguBeacon.sourceCardId);
    }
  }

  function enqueueBabelCard(c: any) {
    if (activeBuguBeacon) resetBeaconSafetyTimerExternal();
    babelQueue.push(c);
    processBabelQueue();
  }

  function finishRadar() {
    radarBackendFinished = true;
    if (babelQueue.length === 0 && !babelQueueTimer) {
      finishBuguBeacon();
    }
  }

  return {
    startRadar,
    finishRadar,
    enqueueBabelCard,
    resetBeaconSafetyTimer: resetBeaconSafetyTimerExternal,
    destroy: () => {
      if (babelQueueTimer) clearTimeout(babelQueueTimer);
      if (buguBeaconTimer) clearTimeout(buguBeaconTimer);
    }
  };
}
