import type { CosmicCard } from '../../types/card';

export interface BabelSessionCallbacks {
  getCards: () => CosmicCard[];
  getMemoryPool: () => CosmicCard[];
  getScrollExpansion: () => number;
  onUpdateStage: (cards: CosmicCard[], memoryPool?: CosmicCard[]) => void;
  toCosmicCard: (data: any, type: string, source: string, ttl: number) => CosmicCard;
  stageCandidate: (
    card: CosmicCard,
    isVisible: boolean,
    scrollExp: number,
    cards: CosmicCard[],
    memPool: CosmicCard[],
    srcId?: string,
    yOverride?: number
  ) => { stagedCard: CosmicCard; updatedActiveCards: CosmicCard[]; updatedMemoryPool?: CosmicCard[] };
  onCardStaged?: (card: CosmicCard) => void;
  onQueueEmpty?: () => void;
}

export interface BabelSessionController {
  enqueue: (cardData: any, origin?: string) => void;
  finishSession: () => void;
  reset: () => void;
  getQueueLength: () => number;
  destroy: () => void;
}
