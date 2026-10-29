import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { createBabelSessionController } from '../modules/babel/babelSession.ts';
import type { CosmicCard } from '../types/card';

describe('Babel Session Controller & Decoupled Queue Engine', () => {
  it('enqueues and stages a valid Babel card into memory pool and active cards', () => {
    let cards: CosmicCard[] = [];
    let memoryPool: CosmicCard[] = [];
    let updateStageCalled = false;
    let cardStagedEvent: CosmicCard | null = null;

    const controller = createBabelSessionController({
      getCards: () => cards,
      getMemoryPool: () => memoryPool,
      getScrollExpansion: () => 0,
      onUpdateStage: (newCards, newMem) => {
        updateStageCalled = true;
        cards = newCards;
        if (newMem) memoryPool = newMem;
      },
      toCosmicCard: (data, type, source, ttl) => ({
        id: data.id || 'b1',
        title: data.title || 'Echo',
        content: data.content || 'Content text',
        type: type as any,
        origin: source,
        decayDuration: ttl,
        createdAt: Date.now(),
        x: 0,
        y: 0,
        z: 0,
        isFadingOut: false,
      }),
      stageCandidate: (card, isVisible, scrollExp, activeCards, memPool, srcId, yOverride) => {
        return {
          stagedCard: { ...card, y: yOverride ?? 0 },
          updatedActiveCards: activeCards,
          updatedMemoryPool: memPool,
        };
      },
      onCardStaged: (c) => {
        cardStagedEvent = c;
      },
    });

    controller.enqueue({ id: 'babel_1', content: 'Substantive echo text', origin: 'bugu' });

    assert.equal(updateStageCalled, true);
    assert.equal(cards.length, 1);
    assert.equal(cards[0].id, 'babel_1');
    assert.equal(cards[0].origin, 'bugu');
    assert.equal(cardStagedEvent?.id, 'babel_1');

    controller.destroy();
  });

  it('filters out empty/blank cards silently without staging', () => {
    let cards: CosmicCard[] = [];
    let memoryPool: CosmicCard[] = [];
    let updateStageCalled = false;

    const controller = createBabelSessionController({
      getCards: () => cards,
      getMemoryPool: () => memoryPool,
      getScrollExpansion: () => 0,
      onUpdateStage: () => {
        updateStageCalled = true;
      },
      toCosmicCard: (data, type, source, ttl) => ({
        id: data.id || 'b_empty',
        title: 'Empty',
        content: data.content || '',
        originalText: data.originalText || '',
        translatedText: data.translatedText || '',
        type: type as any,
        origin: source,
        decayDuration: ttl,
        createdAt: Date.now(),
        x: 0,
        y: 0,
        z: 0,
        isFadingOut: false,
      }),
      stageCandidate: (card) => ({ stagedCard: card, updatedActiveCards: [] }),
    });

    controller.enqueue({ id: 'empty_1', content: '   ' });
    assert.equal(updateStageCalled, false);
    assert.equal(cards.length, 0);

    controller.destroy();
  });

  it('notifies onQueueEmpty when session is finished and queue is exhausted', () => {
    let emptyCalled = false;

    const controller = createBabelSessionController({
      getCards: () => [],
      getMemoryPool: () => [],
      getScrollExpansion: () => 0,
      onUpdateStage: () => {},
      toCosmicCard: (data) => ({
        id: data.id,
        title: 'Babel',
        content: data.content,
        type: 'babel',
        origin: 'atproto',
        decayDuration: 1000,
        createdAt: Date.now(),
        x: 0,
        y: 0,
        z: 0,
        isFadingOut: false,
      }),
      stageCandidate: (card) => ({ stagedCard: card, updatedActiveCards: [] }),
      onQueueEmpty: () => {
        emptyCalled = true;
      },
    });

    controller.enqueue({ id: 'card_1', content: 'Bluesky post', origin: 'atproto' });
    controller.finishSession();

    assert.equal(emptyCalled, true);
    controller.destroy();
  });
});
