import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { isSingleWebURL, SINGLE_URL_REGEX, prependCapturedCardRef } from './webcapture.ts';
// @ts-ignore
import { mergeCapturedThought } from './cardCapture.ts';

describe('Web Capture URL Detection Engine', () => {
  describe('SINGLE_URL_REGEX', () => {
    it('matches standard http and https URLs', () => {
      assert.equal(SINGLE_URL_REGEX.test('https://example.com'), true);
      assert.equal(SINGLE_URL_REGEX.test('http://example.com'), true);
      assert.equal(SINGLE_URL_REGEX.test('https://news.ycombinator.com/item?id=12345'), true);
      assert.equal(SINGLE_URL_REGEX.test('https://x.com/user/status/9876543210#ref'), true);
      assert.equal(SINGLE_URL_REGEX.test('http://localhost:3000/path'), true);
    });

    it('rejects URLs with whitespace inside or surrounding words', () => {
      assert.equal(SINGLE_URL_REGEX.test('https://example.com with notes'), false);
      assert.equal(SINGLE_URL_REGEX.test('note https://example.com'), false);
      assert.equal(SINGLE_URL_REGEX.test('https://a.com https://b.com'), false);
    });

    it('rejects non-http/https schemes and plain text', () => {
      assert.equal(SINGLE_URL_REGEX.test('ftp://files.example.com'), false);
      assert.equal(SINGLE_URL_REGEX.test('file:///Users/kael/test.txt'), false);
      assert.equal(SINGLE_URL_REGEX.test('mailto:user@example.com'), false);
      assert.equal(SINGLE_URL_REGEX.test('just a regular thought'), false);
      assert.equal(SINGLE_URL_REGEX.test(''), false);
    });
  });

  describe('isSingleWebURL', () => {
    it('returns true for single URL without images', () => {
      assert.equal(isSingleWebURL('https://example.com'), true);
      assert.equal(isSingleWebURL('  https://example.com/sub/page?key=val  '), true);
      assert.equal(isSingleWebURL('\nhttps://example.com\n'), true);
    });

    it('returns false when attachedImagesCount > 0', () => {
      assert.equal(isSingleWebURL('https://example.com', 1), false);
      assert.equal(isSingleWebURL('https://example.com', 3), false);
    });

    it('returns false for plain text or mixed content', () => {
      assert.equal(isSingleWebURL('Hello world'), false);
      assert.equal(isSingleWebURL('https://example.com is an interesting read'), false);
      assert.equal(isSingleWebURL('Check this: https://example.com'), false);
      assert.equal(isSingleWebURL('https://example.com\nmy thought below'), false);
      assert.equal(isSingleWebURL(''), false);
      assert.equal(isSingleWebURL('   '), false);
    });
  });

  describe('prependCapturedCardRef', () => {
    it('prepends @title to follow-up thought text when capturedCard is present', () => {
      const card = { id: 'note_1', title: 'DeepSeek R1 论文解读', type: 'note' };
      assert.equal(
        prependCapturedCardRef('这篇文章提到的强化学习很有启发', card),
        '@DeepSeek R1 论文解读 这篇文章提到的强化学习很有启发'
      );
    });

    it('does not duplicate @title if already present in text', () => {
      const card = { id: 'note_1', title: 'DeepSeek R1 论文解读', type: 'note' };
      assert.equal(
        prependCapturedCardRef('@DeepSeek R1 论文解读 继续补充一点想法', card),
        '@DeepSeek R1 论文解读 继续补充一点想法'
      );
    });

    it('returns trimmed text unchanged when capturedCard is null', () => {
      assert.equal(prependCapturedCardRef('  普通灵感  ', null), '普通灵感');
    });
  });

  describe('mergeCapturedThought for Web Captured Cards', () => {
    it('immediately stages a web-captured note card with SourceURL and VIP status', () => {
      const res = mergeCapturedThought({
        data: {
          DocID: 'web_note_123',
          Title: 'Example Web Article',
          Text: 'Full article markdown body',
          Type: 'note',
          SourceURL: 'https://example.com/article',
          CapturedAt: '2026-11-06T12:00:00Z',
        },
        cards: [],
        memoryPool: [],
        localVaultPool: [],
        scrollExpansion: 0,
        latestManualCardId: null,
      });

      assert.equal(res.cards.length, 1);
      assert.equal(res.cards[0].id, 'web_note_123');
      assert.equal(res.cards[0].type, 'note');
      assert.equal(res.cards[0].title, 'Example Web Article');
      assert.equal(res.cards[0].sourceUrl, 'https://example.com/article');
      assert.equal(res.cards[0].isVIP, true);
      assert.equal(res.latestManualCardId, 'web_note_123');
    });
  });
});
