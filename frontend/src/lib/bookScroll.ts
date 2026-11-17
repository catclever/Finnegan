/**
 * Finnegan Cosmic Stage - Book Scroll & Marginalia Pure Engine
 * Task: book_card_frontend_omnibar_and_body_172
 *
 * Layer 2 pure engine (zero DOM dependencies):
 * - Book history extraction & author linking
 * - Excerpt (blockquote `>`) vs personal marginalia block parsing
 * - Same-book card filtering, deduplication, and chronological sorting
 */

import type { CosmicCard } from '../types/card';

export interface BookMeta {
  bookTitle: string;
  bookAuthor: string;
  lastUsedAt?: number;
  count?: number;
}

export interface BookContentBlocks {
  quotes: string[];
  comments: string[];
  hasQuotes: boolean;
}

/**
 * Strips outer book title brackets `《》` and trims whitespace for clean UI display.
 */
export function cleanBookTitleDisplay(title?: string | null): string {
  if (!title || typeof title !== 'string') return '';
  return title.trim().replace(/^《+|》+$/g, '').trim();
}

/**
 * Normalizes a book title for matching/deduplication:
 * strips `《》`, collapses internal whitespace, trims, and lowercases.
 */
export function normalizeBookTitle(title?: string | null): string {
  if (!title || typeof title !== 'string') return '';
  return title
    .replace(/[《》]/g, '')
    .trim()
    .replace(/\s+/g, ' ')
    .toLowerCase();
}

function parseTimestampMs(val?: string | number | null): number {
  if (typeof val === 'number') {
    return Number.isFinite(val) ? val : 0;
  }
  if (typeof val === 'string' && val.trim()) {
    const num = Number(val);
    if (Number.isFinite(num) && /^\d+$/.test(val.trim())) {
      return num;
    }
    const parsed = Date.parse(val);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function extractRawBookTitle(card: any): string {
  return card?.bookTitle || card?.BookTitle || card?.book_title || '';
}

function extractRawBookAuthor(card: any): string {
  return card?.bookAuthor || card?.BookAuthor || card?.book_author || '';
}

/**
 * Scans cards (where `c.type === 'book'` or `c.bookTitle`) and optional `cachedList`,
 * deduplicates by normalized `bookTitle`, links each `bookTitle` with its `bookAuthor`,
 * and sorts by most recent usage descending.
 */
export function extractBookHistory(
  cards: CosmicCard[] = [],
  cachedList?: BookMeta[]
): BookMeta[] {
  const map = new Map<string, BookMeta>();
  const seenCardIds = new Set<string>();

  if (Array.isArray(cards)) {
    for (const c of cards) {
      if (!c) continue;
      if (c.id) {
        if (seenCardIds.has(c.id)) continue;
        seenCardIds.add(c.id);
      }

      const rawType = c.type || (c as any).Type;
      const rawTitle = extractRawBookTitle(c);
      if (rawType !== 'book' && !rawTitle) continue;

      const displayTitle = cleanBookTitleDisplay(rawTitle);
      if (!displayTitle) continue;

      const normKey = normalizeBookTitle(displayTitle);
      if (!normKey) continue;

      const author = extractRawBookAuthor(c).trim();
      const ts = Math.max(
        parseTimestampMs(c.createdAt ?? (c as any).CreatedAt),
        parseTimestampMs(c.elevatedAt),
        parseTimestampMs(c.spawnTime),
        0
      );

      const existing = map.get(normKey);
      if (!existing) {
        map.set(normKey, {
          bookTitle: displayTitle,
          bookAuthor: author,
          lastUsedAt: ts,
          count: 1,
        });
      } else {
        existing.count = (existing.count || 0) + 1;
        const prevTs = existing.lastUsedAt || 0;
        if (ts >= prevTs) {
          existing.lastUsedAt = ts;
          existing.bookTitle = displayTitle;
          if (author) {
            existing.bookAuthor = author;
          }
        } else if (!existing.bookAuthor && author) {
          existing.bookAuthor = author;
        }
      }
    }
  }

  if (Array.isArray(cachedList)) {
    for (const item of cachedList) {
      if (!item || !item.bookTitle) continue;
      const displayTitle = cleanBookTitleDisplay(item.bookTitle);
      if (!displayTitle) continue;
      const normKey = normalizeBookTitle(displayTitle);
      if (!normKey) continue;

      const author = (item.bookAuthor || '').trim();
      const ts = item.lastUsedAt || 0;

      const existing = map.get(normKey);
      if (!existing) {
        map.set(normKey, {
          bookTitle: displayTitle,
          bookAuthor: author,
          lastUsedAt: ts,
          count: item.count ?? 0,
        });
      } else {
        const prevTs = existing.lastUsedAt || 0;
        if (ts >= prevTs) {
          existing.lastUsedAt = ts;
          if (author) {
            existing.bookAuthor = author;
          }
        } else if (!existing.bookAuthor && author) {
          existing.bookAuthor = author;
        }
        if ((existing.count || 0) === 0 && typeof item.count === 'number') {
          existing.count = item.count;
        }
      }
    }
  }

  return Array.from(map.values()).sort((a, b) => {
    const timeDiff = (b.lastUsedAt || 0) - (a.lastUsedAt || 0);
    if (timeDiff !== 0) return timeDiff;
    const countDiff = (b.count || 0) - (a.count || 0);
    if (countDiff !== 0) return countDiff;
    return a.bookTitle.localeCompare(b.bookTitle);
  });
}

/**
 * Returns the linked `bookAuthor` when a matching `bookTitle` is found in history.
 */
export function findLinkedAuthor(bookTitle: string, history: BookMeta[]): string {
  const norm = normalizeBookTitle(bookTitle);
  if (!norm || !Array.isArray(history)) return '';
  const match = history.find((item) => normalizeBookTitle(item.bookTitle) === norm);
  return match?.bookAuthor || '';
}

/**
 * Parses Markdown body lines:
 * - Contiguous lines starting with `>` (e.g. `^\s*>\s?(.*)$`) are grouped into `quotes`.
 * - Non-blockquote paragraphs (separated by blank lines or quote blocks) are grouped into `comments`.
 */
export function parseBookContentBlocks(content: string): BookContentBlocks {
  if (!content || typeof content !== 'string') {
    return { quotes: [], comments: [], hasQuotes: false };
  }

  const lines = content.replace(/\r\n/g, '\n').split('\n');
  const quotes: string[] = [];
  const comments: string[] = [];

  let quoteBuffer: string[] = [];
  let commentBuffer: string[] = [];

  const flushQuote = () => {
    if (quoteBuffer.length > 0) {
      const joined = quoteBuffer.join('\n').trim();
      if (joined) {
        quotes.push(joined);
      }
      quoteBuffer = [];
    }
  };

  const flushComment = () => {
    if (commentBuffer.length > 0) {
      const joined = commentBuffer.join('\n').trim();
      if (joined) {
        comments.push(joined);
      }
      commentBuffer = [];
    }
  };

  for (const line of lines) {
    const quoteMatch = line.match(/^\s*>\s?(.*)$/);
    if (quoteMatch) {
      flushComment();
      quoteBuffer.push(quoteMatch[1]);
    } else if (line.trim() === '') {
      flushQuote();
      flushComment();
    } else {
      flushQuote();
      commentBuffer.push(line);
    }
  }

  flushQuote();
  flushComment();

  return {
    quotes,
    comments,
    hasQuotes: quotes.length > 0,
  };
}

/**
 * Filters `c.type === 'book'` matching `bookTitle` (case/whitespace-insensitive, stripping `《》`),
 * deduplicates by `id`, and sorts chronologically ascending.
 */
export function getSameBookCards(bookTitle: string, allCards: CosmicCard[]): CosmicCard[] {
  const normTarget = normalizeBookTitle(bookTitle);
  if (!normTarget || !Array.isArray(allCards)) return [];

  const dedupMap = new Map<string, CosmicCard>();
  for (const c of allCards) {
    if (!c) continue;
    const rawType = c.type || (c as any).Type;
    if (rawType !== 'book') continue;

    const cardTitleNorm = normalizeBookTitle(extractRawBookTitle(c));
    if (cardTitleNorm !== normTarget) continue;

    if (c.id) {
      if (!dedupMap.has(c.id)) {
        dedupMap.set(c.id, c);
      }
    }
  }

  const result = Array.from(dedupMap.values());
  result.sort((a, b) => {
    const tsA = parseTimestampMs(a.createdAt ?? (a as any).CreatedAt) || parseTimestampMs(a.spawnTime);
    const tsB = parseTimestampMs(b.createdAt ?? (b as any).CreatedAt) || parseTimestampMs(b.spawnTime);
    return tsA - tsB;
  });

  return result;
}
