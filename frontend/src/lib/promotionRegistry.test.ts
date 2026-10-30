import { describe, it, beforeEach } from 'node:test';
import assert from 'node:assert';
import {
  DEFAULT_PROMOTION_STRATEGY,
  BUGU_PROMOTION_STRATEGY,
  ATPROTO_PROMOTION_STRATEGY,
  getPromotionStrategy,
  getCardPromotionStrategy,
  registerPromotionStrategy,
  resetPromotionStrategies
} from './promotionRegistry.ts';

describe('Promotion Strategy Registry Engine', () => {
  beforeEach(() => {
    resetPromotionStrategies();
  });

  it('provides sensible default strategy for unknown or empty keys', () => {
    const s1 = getPromotionStrategy(null);
    assert.strictEqual(s1.icon, 'default_book');
    assert.strictEqual(s1.hasCommentPrompt, false);
    assert.strictEqual(s1.bindMotherCard, false);

    const s2 = getPromotionStrategy('unknown_module');
    assert.strictEqual(s2.icon, 'default_book');
    assert.strictEqual(s2.hasCommentPrompt, false);
  });

  it('retrieves bugu and atproto strategies accurately', () => {
    const bugu = getPromotionStrategy('bugu');
    assert.strictEqual(bugu.icon, 'anchor_comment');
    assert.strictEqual(bugu.hasCommentPrompt, true);
    assert.strictEqual(bugu.bindMotherCard, true);
    assert.ok(bugu.placeholder.includes('批注'));

    const atproto = getPromotionStrategy('atproto');
    assert.strictEqual(atproto.icon, 'anchor_comment');
    assert.strictEqual(atproto.hasCommentPrompt, true);
    assert.strictEqual(atproto.bindMotherCard, true);
  });

  it('resolves strategy for cards based on origin or type', () => {
    const cardBugu = { id: 'c1', origin: 'bugu', type: 'babel' };
    const strat1 = getCardPromotionStrategy(cardBugu);
    assert.strictEqual(strat1.icon, 'anchor_comment');
    assert.strictEqual(strat1.hasCommentPrompt, true);

    const cardLocal = { id: 'c2', origin: 'local', type: 'thought' };
    const strat2 = getCardPromotionStrategy(cardLocal);
    assert.strictEqual(strat2.icon, 'default_book');
    assert.strictEqual(strat2.hasCommentPrompt, false);
  });

  it('allows dynamic registration of custom promotion strategies', () => {
    registerPromotionStrategy('research', {
      icon: 'anchor_comment',
      tooltip: '沉淀学术文献附注',
      hasCommentPrompt: true,
      placeholder: '记录文献提炼想法...',
      bindMotherCard: true
    });

    const custom = getPromotionStrategy('research');
    assert.strictEqual(custom.tooltip, '沉淀学术文献附注');
    assert.strictEqual(custom.hasCommentPrompt, true);
  });
});
