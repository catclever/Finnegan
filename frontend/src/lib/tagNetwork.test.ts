// @ts-nocheck
import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import {
  normalizeTag,
  cardHasTag,
  getCardTags,
  calculateIntrinsicTargetZ,
  calculateResurgenceZ,
  findUnstagedMatchingCards,
} from './tagNetwork.ts';

describe('Finnegan Tag Association & Resurgence Physics', () => {
  describe('normalizeTag', () => {
    it('strips trailing asterisks and trims whitespace', () => {
      assert.equal(normalizeTag('WebKitGTK 4.1*'), 'WebKitGTK 4.1');
      assert.equal(normalizeTag('  松弛感**  '), '松弛感');
      assert.equal(normalizeTag('alpha'), 'alpha');
      assert.equal(normalizeTag(''), '');
    });
  });

  describe('cardHasTag', () => {
    it('matches tags in keywords regardless of asterisks or case', () => {
      const card: any = {
        id: 'c1',
        keywords: {
          object: ['WebKitGTK 4.1*'],
          mood: ['松弛感*'],
        },
      };

      assert.ok(cardHasTag(card, 'WebKitGTK 4.1'));
      assert.ok(cardHasTag(card, 'webkitgtk 4.1*'));
      assert.ok(cardHasTag(card, '松弛感'));
      assert.ok(!cardHasTag(card, 'OtherTag'));
    });

    it('matches legacy tags array', () => {
      const card: any = {
        id: 'c2',
        tags: ['rust*', 'pve-native'],
      };

      assert.ok(cardHasTag(card, 'Rust'));
      assert.ok(cardHasTag(card, 'pve-native'));
      assert.ok(!cardHasTag(card, 'unknown'));
    });

    it('strictly ignores AI keywords that lack an asterisk marker', () => {
      const card: any = {
        id: 'c_ai',
        keywords: {
          object: ['AI提取物无星号'],
          mood: ['人工标记*'],
        },
      };

      assert.ok(!cardHasTag(card, 'AI提取物无星号'));
      assert.ok(cardHasTag(card, '人工标记'));
    });
  });

  describe('getCardTags', () => {
    it('aggregates and deduplicates only manual tags ending with asterisk', () => {
      const card: any = {
        id: 'c3',
        tags: ['WebkitGTK 4.1*', 'Rust'],
        keywords: {
          object: ['AI提取物无星号'],
          mood: ['松弛感*'],
        },
      };

      const tags = getCardTags(card);
      assert.ok(tags.includes('WebKitGTK 4.1') || tags.includes('WebkitGTK 4.1'));
      assert.ok(tags.includes('Rust'));
      assert.ok(tags.includes('松弛感'));
      assert.ok(!tags.includes('AI提取物无星号'));
      assert.ok(!tags.some(t => t.endsWith('*')));
    });
  });

  describe('calculateIntrinsicTargetZ', () => {
    const DAY_MS = 24 * 3600 * 1000;
    const now = 1700000000000;

    it('assigns correct depth bands based on card age', () => {
      // 0 days (today): 3 ~ 11
      const todayCard = { createdAt: new Date(now - 0.2 * DAY_MS).toISOString() };
      const zToday = calculateIntrinsicTargetZ(todayCard, now);
      assert.ok(zToday >= 3 && zToday <= 11);

      // 2 days: 16 ~ 26
      const twoDaysCard = { createdAt: new Date(now - 2 * DAY_MS).toISOString() };
      const z2 = calculateIntrinsicTargetZ(twoDaysCard, now);
      assert.ok(z2 >= 16 && z2 <= 26);

      // 5 days: 30 ~ 42
      const fiveDaysCard = { createdAt: new Date(now - 5 * DAY_MS).toISOString() };
      const z5 = calculateIntrinsicTargetZ(fiveDaysCard, now);
      assert.ok(z5 >= 30 && z5 <= 42);

      // 20 days: 48 ~ 62
      const twentyDaysCard = { createdAt: new Date(now - 20 * DAY_MS).toISOString() };
      const z20 = calculateIntrinsicTargetZ(twentyDaysCard, now);
      assert.ok(z20 >= 48 && z20 <= 62);

      // 60 days: 66 ~ 78
      const sixtyDaysCard = { createdAt: new Date(now - 60 * DAY_MS).toISOString() };
      const z60 = calculateIntrinsicTargetZ(sixtyDaysCard, now);
      assert.ok(z60 >= 66 && z60 <= 78);

      // 200 days: 80 ~ 92
      const twoHundredDaysCard = { createdAt: new Date(now - 200 * DAY_MS).toISOString() };
      const z200 = calculateIntrinsicTargetZ(twoHundredDaysCard, now);
      assert.ok(z200 >= 80 && z200 <= 92);

      // 400 days (> 1 year resurgence echo): 42 ~ 52
      const oldCard = { createdAt: new Date(now - 400 * DAY_MS).toISOString() };
      const zOld = calculateIntrinsicTargetZ(oldCard, now);
      assert.ok(zOld >= 42 && zOld <= 52);
    });
  });

  describe('calculateResurgenceZ', () => {
    it('interpolates smoothly from initialZ (10) down to z_target based on elapsed remainingMs', () => {
      const targetZ = 50;
      const initialZ = 10;
      const durationMs = 120000;

      // At start (t = 0, remainingMs = 120000): z = 10
      assert.equal(calculateResurgenceZ(targetZ, 120000, durationMs, initialZ), 10);

      // At midpoint (remainingMs = 60000): z = 50 - (50 - 10) * 0.5 = 30
      assert.equal(calculateResurgenceZ(targetZ, 60000, durationMs, initialZ), 30);

      // At three-quarter (remainingMs = 30000): z = 50 - (50 - 10) * 0.25 = 40
      assert.equal(calculateResurgenceZ(targetZ, 30000, durationMs, initialZ), 40);

      // At end (remainingMs = 0): z = 50
      assert.equal(calculateResurgenceZ(targetZ, 0, durationMs, initialZ), 50);
    });
  });

  describe('findUnstagedMatchingCards', () => {
    it('returns only matching cards from memory and vault pools that are NOT on stage', () => {
      const stage = [
        { id: 'c1', keywords: { object: ['tagA*'] } } as any,
      ];
      const memory = [
        { id: 'c1', keywords: { object: ['tagA*'] } } as any, // already on stage
        { id: 'c2', keywords: { object: ['tagA*'] } } as any, // not on stage
        { id: 'c3', keywords: { object: ['tagB*'] } } as any, // not matching
      ];
      const vault = [
        { id: 'c4', keywords: { mood: ['tagA*'] } } as any, // not on stage
        { id: 'c2', keywords: { object: ['tagA*'] } } as any, // duplicate of memory
      ];

      const unstaged = findUnstagedMatchingCards('tagA', stage, memory, vault);
      assert.equal(unstaged.length, 2);
      assert.deepEqual(unstaged.map(c => c.id).sort(), ['c2', 'c4']);
    });
  });
});
