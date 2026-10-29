import type { CosmicCard } from '../../types/card';
import type { BabelSessionCallbacks, BabelSessionController } from './types';
// @ts-ignore
import { MACRO_TTL_MS } from '../../lib/stageDecay.ts';

export function createBabelSessionController(callbacks: BabelSessionCallbacks): BabelSessionController {
  let babelQueue: Array<{ data: any; origin?: string }> = [];
  let queueTimer: any = null;
  let isBackendFinished = false;
  let stagedCount = 0;

  function stageOneBabelCard(item: { data: any; origin?: string }) {
    const origin = item.origin || item.data?.origin || 'bugu';
    const newCard = callbacks.toCosmicCard(item.data, 'babel', origin, MACRO_TTL_MS);

    // Filter out completely empty cards
    const substantive = (newCard.content || newCard.originalText || newCard.translatedText || '').trim();
    if (!substantive) return;

    stagedCount++;
    const scrollExpansion = callbacks.getScrollExpansion();
    const viewportY = Math.round(scrollExpansion * 300);
    // Viewport-adaptive staggering: first 2 cards within immediate viewport, subsequent cards staggered lower
    const targetY = stagedCount <= 2 ? viewportY : viewportY + 480;

    let cards = callbacks.getCards();
    let memoryPool = callbacks.getMemoryPool();

    memoryPool = [newCard, ...memoryPool.filter(m => m.id !== newCard.id)];
    const sourceCardId = newCard.echoOf;

    const { stagedCard, updatedActiveCards, updatedMemoryPool } = callbacks.stageCandidate(
      newCard,
      true,
      scrollExpansion,
      cards,
      memoryPool,
      sourceCardId,
      targetY
    );

    if (updatedMemoryPool) memoryPool = updatedMemoryPool;
    cards = [...updatedActiveCards.filter((card: any) => card.id !== stagedCard.id), stagedCard];

    callbacks.onUpdateStage(cards, memoryPool);
    if (callbacks.onCardStaged) {
      callbacks.onCardStaged(stagedCard);
    }
  }

  function processQueue() {
    if (queueTimer) return;
    if (babelQueue.length === 0) {
      if (isBackendFinished && callbacks.onQueueEmpty) {
        callbacks.onQueueEmpty();
      }
      return;
    }

    const nextItem = babelQueue.shift();
    if (nextItem) {
      stageOneBabelCard(nextItem);
    }

    if (babelQueue.length === 0) {
      if (isBackendFinished && callbacks.onQueueEmpty) {
        callbacks.onQueueEmpty();
      }
      return;
    }

    queueTimer = setTimeout(() => {
      queueTimer = null;
      processQueue();
    }, 850);
  }

  function enqueue(cardData: any, origin?: string) {
    babelQueue.push({ data: cardData, origin });
    processQueue();
  }

  function finishSession() {
    isBackendFinished = true;
    if (babelQueue.length === 0 && !queueTimer) {
      if (callbacks.onQueueEmpty) {
        callbacks.onQueueEmpty();
      }
    }
  }

  function reset() {
    if (queueTimer) {
      clearTimeout(queueTimer);
      queueTimer = null;
    }
    babelQueue = [];
    isBackendFinished = false;
    stagedCount = 0;
  }

  function destroy() {
    reset();
  }

  return {
    enqueue,
    finishSession,
    reset,
    getQueueLength: () => babelQueue.length,
    destroy,
  };
}
