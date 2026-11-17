import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  extractBookHistory,
  findLinkedAuthor,
  parseBookContentBlocks,
  getSameBookCards,
  type BookMeta,
} from './bookScroll.ts';
import { toCosmicCard } from './stageDecay.ts';
import { mergeCapturedThought, updateCardInPools } from './cardCapture.ts';
import type { CosmicCard } from '../types/card';

function makeCard(overrides: Partial<CosmicCard>): CosmicCard {
  return {
    id: 'card_1',
    type: 'book',
    title: 'Reading Note',
    content: '',
    x: 0,
    y: 0,
    rotation: 0,
    z: 10,
    spawnTime: 1000,
    decayDuration: 3600000,
    isFresh: false,
    isVIP: false,
    origin: 'local',
    ...overrides,
  };
}

describe('Book Scroll & Marginalia Engine (bookScroll.ts)', () => {
  describe('extractBookHistory', () => {
    it('extracts and deduplicates books by normalized title, linking authors and sorting by recency', () => {
      const cards: CosmicCard[] = [
        makeCard({
          id: 'b1',
          type: 'book',
          bookTitle: '《沙丘》',
          bookAuthor: '弗兰克·赫伯特',
          createdAt: '2026-11-01T10:00:00.000Z',
        }),
        makeCard({
          id: 'b2',
          type: 'book',
          bookTitle: '  沙丘  ',
          bookAuthor: '',
          createdAt: '2026-11-05T12:00:00.000Z',
        }),
        makeCard({
          id: 'b3',
          type: 'book',
          bookTitle: 'Dune Messiah',
          bookAuthor: 'Frank Herbert',
          createdAt: '2026-11-06T08:00:00.000Z',
        }),
        makeCard({
          id: 't1',
          type: 'thought',
          bookTitle: '',
          content: 'Just a regular thought',
          createdAt: '2026-11-07T08:00:00.000Z',
        }),
        // Duplicate card id should not double count
        makeCard({
          id: 'b1',
          type: 'book',
          bookTitle: '沙丘',
          bookAuthor: '弗兰克·赫伯特',
          createdAt: '2026-11-01T10:00:00.000Z',
        }),
      ];

      const cached: BookMeta[] = [
        {
          bookTitle: 'Three-Body Problem',
          bookAuthor: 'Cixin Liu',
          lastUsedAt: Date.parse('2026-11-04T00:00:00.000Z'),
          count: 1,
        },
      ];

      const history = extractBookHistory(cards, cached);

      assert.equal(history.length, 3);
      assert.equal(history[0].bookTitle, 'Dune Messiah');
      assert.equal(history[0].bookAuthor, 'Frank Herbert');

      assert.equal(history[1].bookTitle, '沙丘');
      assert.equal(history[1].bookAuthor, '弗兰克·赫伯特');
      assert.equal(history[1].count, 2);

      assert.equal(history[2].bookTitle, 'Three-Body Problem');
      assert.equal(history[2].bookAuthor, 'Cixin Liu');
    });

    it('includes non-book cards if they carry bookTitle metadata', () => {
      const cards: CosmicCard[] = [
        makeCard({
          id: 'n1',
          type: 'thought',
          bookTitle: 'Gödel, Escher, Bach',
          bookAuthor: 'Douglas Hofstadter',
          createdAt: 5000,
        }),
      ];
      const history = extractBookHistory(cards);
      assert.equal(history.length, 1);
      assert.equal(history[0].bookTitle, 'Gödel, Escher, Bach');
      assert.equal(history[0].bookAuthor, 'Douglas Hofstadter');
    });
  });

  describe('findLinkedAuthor', () => {
    const history: BookMeta[] = [
      { bookTitle: '沙丘', bookAuthor: '弗兰克·赫伯特', lastUsedAt: 2000 },
      { bookTitle: 'The Mythical Man-Month', bookAuthor: 'Frederick P. Brooks Jr.', lastUsedAt: 1000 },
    ];

    it('matches case-insensitively and ignores 《》 and extra whitespace', () => {
      assert.equal(findLinkedAuthor('《 沙丘 》', history), '弗兰克·赫伯特');
      assert.equal(findLinkedAuthor('the  mythical man-month', history), 'Frederick P. Brooks Jr.');
    });

    it('returns empty string when book is unknown or input is blank', () => {
      assert.equal(findLinkedAuthor('Unknown Book', history), '');
      assert.equal(findLinkedAuthor('   ', history), '');
    });
  });

  describe('parseBookContentBlocks', () => {
    it('groups contiguous blockquote lines into quotes and non-blockquote paragraphs into comments', () => {
      const body = [
        '> Fear is the mind-killer.',
        '> Fear is the little-death that brings total obliteration.',
        '',
        '这段描写恐惧的句子极具节奏感。',
        '',
        '> I must not fear.',
        '紧随其后的个人批注第一行',
        '紧随其后的个人批注第二行',
      ].join('\n');

      const parsed = parseBookContentBlocks(body);
      assert.equal(parsed.hasQuotes, true);
      assert.deepEqual(parsed.quotes, [
        'Fear is the mind-killer.\nFear is the little-death that brings total obliteration.',
        'I must not fear.',
      ]);
      assert.deepEqual(parsed.comments, [
        '这段描写恐惧的句子极具节奏感。',
        '紧随其后的个人批注第一行\n紧随其后的个人批注第二行',
      ]);
    });

    it('handles content without any blockquotes gracefully', () => {
      const body = '只有个人批注第一段。\n\n只有个人批注第二段。';
      const parsed = parseBookContentBlocks(body);
      assert.equal(parsed.hasQuotes, false);
      assert.deepEqual(parsed.quotes, []);
      assert.deepEqual(parsed.comments, [
        '只有个人批注第一段。',
        '只有个人批注第二段。',
      ]);
    });

    it('handles empty or whitespace-only content', () => {
      assert.deepEqual(parseBookContentBlocks('   \n  '), {
        quotes: [],
        comments: [],
        hasQuotes: false,
      });
    });
  });

  describe('getSameBookCards', () => {
    it('filters type=book matching normalized title, deduplicates by id, and sorts chronologically ascending', () => {
      const allCards: CosmicCard[] = [
        makeCard({
          id: 'c3',
          type: 'book',
          bookTitle: '《  Refactoring 》',
          createdAt: '2026-11-05T15:00:00.000Z',
        }),
        makeCard({
          id: 'c1',
          type: 'book',
          bookTitle: 'refactoring',
          createdAt: '2026-11-01T08:00:00.000Z',
        }),
        makeCard({
          id: 'c1',
          type: 'book',
          bookTitle: 'Refactoring',
          createdAt: '2026-11-01T08:00:00.000Z',
        }),
        makeCard({
          id: 'c_thought',
          type: 'thought',
          bookTitle: 'Refactoring',
          createdAt: '2026-11-02T10:00:00.000Z',
        }),
        makeCard({
          id: 'c2',
          type: 'book',
          bookTitle: 'Clean Code',
          createdAt: '2026-11-03T10:00:00.000Z',
        }),
      ];

      const same = getSameBookCards('REFACTORING', allCards);
      assert.equal(same.length, 2);
      assert.equal(same[0].id, 'c1');
      assert.equal(same[1].id, 'c3');
    });
  });

  describe('Book Card Stage & Capture Wiring', () => {
    it('toCosmicCard preserves bookTitle and bookAuthor across camelCase, PascalCase, and snake_case', () => {
      const pascal = toCosmicCard(
        { ID: 'b_pascal', Type: 'book', Content: '> Quote', BookTitle: '沙丘', BookAuthor: '弗兰克·赫伯特' },
        'book',
        'local'
      );
      assert.equal(pascal.type, 'book');
      assert.equal(pascal.bookTitle, '沙丘');
      assert.equal(pascal.bookAuthor, '弗兰克·赫伯特');

      const snake = toCosmicCard(
        { id: 'b_snake', type: 'book', content: 'Note', book_title: 'Dune', book_author: 'Frank Herbert' },
        'book',
        'manual'
      );
      assert.equal(snake.bookTitle, 'Dune');
      assert.equal(snake.bookAuthor, 'Frank Herbert');
    });

    it('mergeCapturedThought stages data.Type === "book" with bookTitle and bookAuthor onto Cosmic Stage', () => {
      const res = mergeCapturedThought({
        data: {
          DocID: 'book_doc_1',
          Type: 'book',
          Title: '《三体》',
          Text: '> 给岁月以文明。\n\n一句震撼的话。',
          BookTitle: '三体',
          BookAuthor: '刘慈欣',
        },
        cards: [],
        memoryPool: [],
        localVaultPool: [],
        scrollExpansion: 0,
        latestManualCardId: null,
      });

      const staged = res.cards.find((c) => c.id === 'book_doc_1');
      assert.ok(staged);
      assert.equal(staged.type, 'book');
      assert.equal(staged.bookTitle, '三体');
      assert.equal(staged.bookAuthor, '刘慈欣');
      assert.equal(staged.isVIP, true);
    });

    it('updateCardInPools preserves updated bookTitle and bookAuthor', () => {
      const initial = makeCard({ id: 'b_upd', type: 'book', bookTitle: 'Old Book', bookAuthor: 'Old Author' });
      const res = updateCardInPools(
        { id: 'b_upd', type: 'book', content: 'Updated', bookTitle: 'New Book', bookAuthor: 'New Author' },
        { cards: [initial], memoryPool: [], localVaultPool: [initial] }
      );
      assert.equal(res.cards[0].bookTitle, 'New Book');
      assert.equal(res.cards[0].bookAuthor, 'New Author');
      assert.equal(res.localVaultPool[0].bookTitle, 'New Book');
      assert.equal(res.localVaultPool[0].bookAuthor, 'New Author');
    });
  });
});
