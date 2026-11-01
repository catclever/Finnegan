/**
  * Finnegan Cosmic Stage - Tag Autocomplete & Parsing Engine
  * Layer 2 Pure Function Engine (Zero DOM Dependencies)
  */

import type { CosmicCard } from '../types/card';

/**
 * Collects and deduplicates all unique available tags across stage cards and local vault pool.
 */
export function collectAllAvailableTags(
  cards: CosmicCard[] = [],
  localVaultPool: CosmicCard[] = []
): string[] {
  const set = new Set<string>();
  for (const c of [...cards, ...localVaultPool]) {
    if (c?.tags && Array.isArray(c.tags)) {
      for (const t of c.tags) {
        if (typeof t === 'string') {
          const clean = t.replace(/\*+$/, '').trim();
          if (clean) set.add(clean);
        }
      }
    }
    if (c?.keywords && typeof c.keywords === 'object') {
      for (const list of Object.values(c.keywords)) {
        if (Array.isArray(list)) {
          for (const kw of list) {
            if (typeof kw === 'string') {
              const clean = kw.replace(/\*+$/, '').trim();
              if (clean) set.add(clean);
            }
          }
        }
      }
    }
  }
  return Array.from(set).sort((a, b) => a.localeCompare(b));
}

/**
 * Filters and limits tags matching the user's tag query prefix/substring.
 */
export function filterMatchedTags(
  allTags: string[] = [],
  tagQuery: string = '',
  limit: number = 15
): string[] {
  if (!allTags || allTags.length === 0) return [];
  const query = (tagQuery || '').trim().toLowerCase();
  if (query === '') {
    return allTags.slice(0, limit);
  }
  return allTags
    .filter(t => t.toLowerCase().includes(query))
    .slice(0, limit);
}

export interface TagTriggerResult {
  isTriggered: boolean;
  tagQuery: string;
  tagMatchStart: number;
}

/**
 * Detects whether the cursor in omnibar text is preceded by a '#' tag trigger token.
 */
export function detectTagTriggerAtCursor(
  omniText: string,
  cursorPos: number
): TagTriggerResult {
  if (typeof omniText !== 'string' || cursorPos < 0) {
    return { isTriggered: false, tagQuery: '', tagMatchStart: -1 };
  }
  const textBeforeCursor = omniText.slice(0, cursorPos);
  const hashMatch = textBeforeCursor.match(/(?:^|\s)#([^\s#]*)$/);
  if (!hashMatch) {
    return { isTriggered: false, tagQuery: '', tagMatchStart: -1 };
  }
  return {
    isTriggered: true,
    tagQuery: hashMatch[1],
    tagMatchStart: textBeforeCursor.lastIndexOf('#'),
  };
}

export interface TagSelectionResult {
  newText: string;
  newCursor: number;
}

/**
 * Applies a selected tag token to the text, formatting spaces as hyphens and appending a trailing space.
 */
export function applyTagSelectionToText(
  omniText: string,
  tagMatchStart: number,
  cursorPos: number,
  tag: string
): TagSelectionResult {
  if (tagMatchStart < 0) {
    return { newText: omniText, newCursor: cursorPos };
  }
  const before = omniText.slice(0, tagMatchStart);
  const after = omniText.slice(cursorPos);
  const tagWithFormat = tag.includes(' ') ? tag.replace(/\s+/g, '-') : tag;
  const inserted = `#${tagWithFormat} `;
  const newText = `${before}${inserted}${after}`;
  const newCursor = (before + inserted).length;
  return { newText, newCursor };
}

export interface AtTriggerResult {
  isTriggered: boolean;
  cardQuery: string;
  cardMatchStart: number;
}

/**
 * Detects whether the cursor in omnibar text is preceded by a '@' card association trigger token.
 */
export function detectAtTriggerAtCursor(
  omniText: string,
  cursorPos: number
): AtTriggerResult {
  if (typeof omniText !== 'string' || cursorPos < 0) {
    return { isTriggered: false, cardQuery: '', cardMatchStart: -1 };
  }
  const textBeforeCursor = omniText.slice(0, cursorPos);
  const atMatch = textBeforeCursor.match(/(?:^|\s)@([^\s@]*)$/);
  if (!atMatch) {
    return { isTriggered: false, cardQuery: '', cardMatchStart: -1 };
  }
  return {
    isTriggered: true,
    cardQuery: atMatch[1],
    cardMatchStart: textBeforeCursor.lastIndexOf('@'),
  };
}

export interface AtSelectionResult {
  newText: string;
  newCursor: number;
}

/**
 * Applies a selected card candidate to the text, inserting @Title and appending a trailing space.
 */
export function applyAtSelectionToText(
  omniText: string,
  cardMatchStart: number,
  cursorPos: number,
  title: string
): AtSelectionResult {
  if (cardMatchStart < 0) {
    return { newText: omniText, newCursor: cursorPos };
  }
  const before = omniText.slice(0, cardMatchStart);
  const after = omniText.slice(cursorPos);
  const cleanTitle = (title || 'Card').trim();
  const inserted = `@${cleanTitle} `;
  const newText = `${before}${inserted}${after}`;
  const newCursor = (before + inserted).length;
  return { newText, newCursor };
}
