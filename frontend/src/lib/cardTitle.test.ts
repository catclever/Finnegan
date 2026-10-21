import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { getCardDisplayTitle, getCardParamDisplayTitle, isGenericTitle } from './cardTitle.ts';
import type { CosmicCard } from '../types/card';

describe('Card Title & Param Display Title Engine', () => {
  const baseCard: CosmicCard = {
    id: 'test-card-1',
    type: 'thought',
    title: 'Thought',
    content: 'Deep learning neural networks and transformers are shaping the future of synthetic intelligence.',
    x: 0,
    y: 0,
    rotation: 0,
    z: 10,
    spawnTime: 1000,
    decayDuration: 3600000,
    isFresh: true,
    isVIP: false,
    origin: 'local'
  };

  describe('isGenericTitle', () => {
    it('identifies placeholder titles', () => {
      assert.equal(isGenericTitle('Thought'), true);
      assert.equal(isGenericTitle('thought'), true);
      assert.equal(isGenericTitle('Card'), true);
      assert.equal(isGenericTitle('Untitled Card'), true);
      assert.equal(isGenericTitle('Babel Echo'), true);
      assert.equal(isGenericTitle(null), true);
      assert.equal(isGenericTitle(''), true);
      assert.equal(isGenericTitle('  '), true);
    });

    it('identifies custom non-generic titles', () => {
      assert.equal(isGenericTitle('Quantum Computing'), false);
      assert.equal(isGenericTitle('Thought Experiment on Mars'), false);
      assert.equal(isGenericTitle('Cardiology Notes'), false);
    });
  });

  describe('getCardDisplayTitle', () => {
    it('returns empty string for null card', () => {
      assert.equal(getCardDisplayTitle(null), '');
      assert.equal(getCardDisplayTitle(undefined), '');
    });

    it('returns explicit custom title', () => {
      const card: CosmicCard = { ...baseCard, title: 'My Custom Note' };
      assert.equal(getCardDisplayTitle(card), 'My Custom Note');
    });

    it('derives from first line of content if title is generic placeholder', () => {
      const card: CosmicCard = {
        ...baseCard,
        title: 'Thought',
        content: '# Research on Antigravity\nSecond line details'
      };
      assert.equal(getCardDisplayTitle(card), 'Research on Antigrav...');
    });
  });

  describe('getCardParamDisplayTitle', () => {
    it('returns empty string for null card', () => {
      assert.equal(getCardParamDisplayTitle(null, 'test'), '');
    });

    it('returns default title when query is empty or whitespace', () => {
      const card: CosmicCard = { ...baseCard, title: 'Deep Tech Roadmap' };
      assert.equal(getCardParamDisplayTitle(card, ''), 'Deep Tech Roadmap');
      assert.equal(getCardParamDisplayTitle(card, '   '), 'Deep Tech Roadmap');
      assert.equal(getCardParamDisplayTitle(card, null), 'Deep Tech Roadmap');
    });

    it('returns full title when custom title contains query', () => {
      const card: CosmicCard = {
        ...baseCard,
        title: 'Bugu Radar Protocol Specification',
        content: 'Unrelated body text here'
      };
      assert.equal(getCardParamDisplayTitle(card, 'radar'), 'Bugu Radar Protocol Specification');
      assert.equal(getCardParamDisplayTitle(card, 'Protocol'), 'Bugu Radar Protocol Specification');
    });

    it('does not return generic placeholder title even if it contains query', () => {
      const card: CosmicCard = {
        ...baseCard,
        title: 'Thought',
        content: 'This thought describes machine learning architecture in detail.'
      };
      // Query "thought" matches generic title "Thought", but should search content instead!
      const res = getCardParamDisplayTitle(card, 'thought');
      assert.ok(res.startsWith('...'));
      assert.ok(res.endsWith('...'));
      assert.ok(res.includes('thought describes machine'));
    });

    it('extracts excerpt from content when title is generic or does not match', () => {
      const card: CosmicCard = {
        ...baseCard,
        title: 'Thought',
        content: 'Preceding words here and transformers are shaping the future of synthetic intelligence.'
      };
      const res = getCardParamDisplayTitle(card, 'transformers');
      assert.ok(res.startsWith('...'));
      assert.ok(res.endsWith('...'));
      assert.ok(res.includes('transformers'));
    });

    it('handles query near the start of content gracefully', () => {
      const card: CosmicCard = {
        ...baseCard,
        title: 'Card',
        content: 'Transformers at the beginning of sentence.'
      };
      const res = getCardParamDisplayTitle(card, 'Transformers');
      assert.ok(res.startsWith('...'));
      assert.ok(res.endsWith('...'));
      assert.ok(res.includes('Transformers'));
    });

    it('handles query across newlines in content without breaking into multiple lines', () => {
      const card: CosmicCard = {
        ...baseCard,
        title: 'Thought',
        content: '# Header\n\nParagraph 1 text\n\nTarget keyword located here with context.'
      };
      const res = getCardParamDisplayTitle(card, 'keyword');
      assert.ok(res.startsWith('...'));
      assert.ok(res.endsWith('...'));
      assert.ok(!res.includes('\n'), 'display title must not contain raw newlines');
      assert.ok(res.includes('keyword'));
    });

    it('strictly preserves card content untouched', () => {
      const originalContent = 'Exact immutable content that must never be modified by excerpt generation.';
      const card: CosmicCard = {
        ...baseCard,
        title: 'Thought',
        content: originalContent
      };
      getCardParamDisplayTitle(card, 'immutable');
      assert.equal(card.content, originalContent);
    });

    it('falls back to getCardDisplayTitle when query does not match title or content', () => {
      const card: CosmicCard = {
        ...baseCard,
        title: 'My Custom Note',
        content: 'Some content'
      };
      assert.equal(getCardParamDisplayTitle(card, 'xyznonexistent'), 'My Custom Note');
    });
  });
});
