import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { filterCandidateCard, partitionCardTiers, rankSearchCandidates } from './paramCandidateEngine.ts';
import type { CosmicCard, ParamSpec } from '../types/card';

describe('Parameter Candidate Engine (3-Tier Hierarchy & Filter)', () => {
  const thoughtCard: CosmicCard = {
    id: 'thought-1',
    type: 'thought',
    title: 'Epiphany on Architecture',
    content: 'Consciousness stream architecture should be layered and decoupled.',
    x: 0,
    y: 0,
    rotation: 0,
    z: 10,
    createdAt: '2026-10-27T10:00:00Z',
    spawnTime: 1000,
    decayDuration: 3600000,
    isFresh: true,
    isVIP: false,
    origin: 'local',
    keywords: {
      concept: ['架构解耦', '意识流'],
      motif: ['深度重构']
    }
  };

  const noteCard: CosmicCard = {
    id: 'note-1',
    type: 'note',
    title: 'Redis Setup Command',
    content: 'redis-server --port 6379',
    x: 0,
    y: 0,
    rotation: 0,
    z: 20,
    createdAt: '2026-10-27T11:00:00Z',
    spawnTime: 2000,
    decayDuration: 3600000,
    isFresh: true,
    isVIP: false,
    origin: 'local'
  };

  const emptyKeywordCard: CosmicCard = {
    id: 'thought-raw',
    type: 'thought',
    title: 'Unprocessed Thought',
    content: 'A quick thought that just arrived.',
    x: 0,
    y: 0,
    rotation: 0,
    z: 30,
    createdAt: '2026-10-27T12:00:00Z',
    spawnTime: 3000,
    decayDuration: 3600000,
    isFresh: true,
    isVIP: false,
    origin: 'local'
  };

  describe('filterCandidateCard', () => {
    it('accepts all cards when spec is empty or has no filters', () => {
      assert.equal(filterCandidateCard(thoughtCard, null), true);
      assert.equal(filterCandidateCard(noteCard, {} as ParamSpec), true);
    });

    it('filters out excluded types', () => {
      const spec: ParamSpec = {
        name: 'card',
        type: 'card',
        description: 'Test',
        excludedTypes: ['note']
      };
      assert.equal(filterCandidateCard(thoughtCard, spec), true);
      assert.equal(filterCandidateCard(noteCard, spec), false);
    });

    it('enforces allowed types', () => {
      const spec: ParamSpec = {
        name: 'card',
        type: 'card',
        description: 'Test',
        allowedTypes: ['note']
      };
      assert.equal(filterCandidateCard(thoughtCard, spec), false);
      assert.equal(filterCandidateCard(noteCard, spec), true);
    });

    it('enforces keyword extraction when requireKeywords is true', () => {
      const spec: ParamSpec = {
        name: 'card',
        type: 'card',
        description: 'Bugu Radar Spec',
        requireKeywords: true,
        excludedTypes: ['note']
      };
      assert.equal(filterCandidateCard(thoughtCard, spec), true);
      assert.equal(filterCandidateCard(noteCard, spec), false);
      assert.equal(filterCandidateCard(emptyKeywordCard, spec), false);
    });
  });

  describe('partitionCardTiers', () => {
    const screen1Card: CosmicCard = {
      ...thoughtCard,
      id: 'screen1-card',
      content: 'Screen 1 thought content',
      y: 50,
      createdAt: '2026-10-27T12:00:00Z'
    };

    const screen2Card: CosmicCard = {
      ...thoughtCard,
      id: 'screen2-card',
      content: 'Screen 2 thought content',
      y: 490,
      createdAt: '2026-10-27T11:00:00Z'
    };

    const vaultCard: CosmicCard = {
      ...thoughtCard,
      id: 'vault-card',
      content: 'Vault thought content',
      y: 0,
      createdAt: '2026-10-27T09:00:00Z'
    };

    it('correctly partitions cards across Tier 1, Tier 2, and Tier 3', () => {
      // scrollExpansion = 0 => viewportY = 0.
      // screen1Card: y=50 (|50 - 0| <= 260) => Tier 1 (current_screen)
      // screen2Card: y=490 (|490 - 0| > 260) => Tier 2 (other_screens)
      // vaultCard: not in stageCards => Tier 3 (vault)
      const stageCards = [screen1Card, screen2Card];
      const localVaultPool = [screen1Card, vaultCard]; // screen1Card is in vault too, must be deduplicated

      const res = partitionCardTiers(stageCards, localVaultPool, 0, null, 10);

      assert.equal(res.items.length, 3);
      assert.equal(res.items[0].card.id, 'screen1-card');
      assert.equal(res.items[0].tier, 'current_screen');

      assert.equal(res.items[1].card.id, 'screen2-card');
      assert.equal(res.items[1].tier, 'other_screens');
      assert.equal(res.items[1].screenIndex, 2);

      assert.equal(res.items[2].card.id, 'vault-card');
      assert.equal(res.items[2].tier, 'vault');

      assert.equal(res.tierCounts.current, 1);
      assert.equal(res.tierCounts.other, 1);
      assert.equal(res.tierCounts.vault, 1);
      assert.equal(res.totalVaultAvailable, 1);
    });

    it('adapts when scrollExpansion moves viewport to Screen 2', () => {
      // scrollExpansion = 1.6 => viewportY = round(1.6 * 300) = 480.
      // screen2Card: y=490 (|490 - 480| = 10 <= 260) => Tier 1!
      // screen1Card: y=50 (|50 - 480| = 430 > 260) => Tier 2!
      const stageCards = [screen1Card, screen2Card];
      const res = partitionCardTiers(stageCards, [], 1.6, null, 10);

      assert.equal(res.items.length, 2);
      assert.equal(res.items[0].card.id, 'screen2-card');
      assert.equal(res.items[0].tier, 'current_screen');

      assert.equal(res.items[1].card.id, 'screen1-card');
      assert.equal(res.items[1].tier, 'other_screens');
    });

    it('applies command filtering during partitioning', () => {
      const stageCards = [screen1Card, noteCard];
      const spec: ParamSpec = {
        name: 'card',
        type: 'card',
        description: 'Bugu',
        excludedTypes: ['note']
      };
      const res = partitionCardTiers(stageCards, [], 0, spec, 10);
      assert.equal(res.items.length, 1);
      assert.equal(res.items[0].card.id, 'screen1-card');
    });
  });

  describe('rankSearchCandidates', () => {
    const stageCurrent: CosmicCard = { ...thoughtCard, id: 'stage-cur', content: 'Current hit content', y: 0 };
    const stageDeep: CosmicCard = { ...thoughtCard, id: 'stage-deep', content: 'Deep hit content', y: 600 };
    const vaultOnly: CosmicCard = { ...thoughtCard, id: 'vault-only', content: 'Vault only content', y: 0 };

    it('sorts search hits by tier priority: Current Screen > Other Screens > Vault', () => {
      // Results from search engine
      const searchResults = [
        { id: 'vault-only', title: 'Vault Hit', content: 'Vault only content' },
        { id: 'stage-deep', title: 'Deep Hit', content: 'Deep hit content' },
        { id: 'stage-cur', title: 'Current Hit', content: 'Current hit content' }
      ];

      const ranked = rankSearchCandidates(searchResults, [stageCurrent, stageDeep], 0, null, 10);

      assert.equal(ranked.length, 3);
      assert.equal(ranked[0].card.id, 'stage-cur');
      assert.equal(ranked[0].tier, 'current_screen');

      assert.equal(ranked[1].card.id, 'stage-deep');
      assert.equal(ranked[1].tier, 'other_screens');

      assert.equal(ranked[2].card.id, 'vault-only');
      assert.equal(ranked[2].tier, 'vault');
    });
  });
});
