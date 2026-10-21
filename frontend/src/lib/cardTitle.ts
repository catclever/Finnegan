import type { CosmicCard } from '../types/card';

const GENERIC_TITLES = new Set(['thought', 'card', 'untitled card', 'babel echo']);

export function isGenericTitle(title?: string | null): boolean {
  if (!title || !title.trim()) return true;
  return GENERIC_TITLES.has(title.trim().toLowerCase());
}

/**
 * Derives default display title for a card:
 * If an explicit non-generic title exists, uses it;
 * otherwise extracts first non-empty line of content, capped to 20 chars.
 */
export function getCardDisplayTitle(c: CosmicCard | null | undefined): string {
  if (!c) return '';
  if (c.title && !isGenericTitle(c.title)) {
    return c.title;
  }
  const raw = c.content || c.translatedText || c.originalText || '';
  const clean = raw.split('\n')[0].replace(/^[#\s\-*]+/, '').trim();
  if (clean) {
    return clean.length > 20 ? clean.slice(0, 20) + '...' : clean;
  }
  return c.title || (c.type === 'babel' ? 'Babel Echo' : 'Thought');
}

/**
 * Returns the single-line parameter candidate title for Raycast/Alfred style dropdown:
 * 1. If !query, returns getCardDisplayTitle(c).
 * 2. If c.title contains query and is not a generic placeholder, returns full c.title.
 * 3. Otherwise searches content for first hit of query, extracting [hit - 12, hit + query.len + 30]
 *    formatted as '...snippet...'.
 * 4. Falls back to getCardDisplayTitle(c) if no match found.
 * Strictly avoids modifying c.content or any underlying card data.
 */
export function getCardParamDisplayTitle(c: CosmicCard | null | undefined, query?: string | null): string {
  if (!c) return '';
  const q = (query || '').trim();
  if (!q) {
    return getCardDisplayTitle(c);
  }

  const title = (c.title || '').trim();
  const qLower = q.toLowerCase();

  // If title contains query and is not a generic placeholder (Thought, Card, etc.), return full title
  if (title && !isGenericTitle(title) && title.toLowerCase().includes(qLower)) {
    return c.title;
  }

  // Otherwise locate the first occurrence of query in the content/body
  const rawContent = c.content || c.translatedText || c.originalText || '';
  const normalizedContent = rawContent.replace(/\r?\n+/g, ' ');
  let hitIdx = normalizedContent.toLowerCase().indexOf(qLower);
  let source = normalizedContent;

  if (hitIdx === -1) {
    hitIdx = rawContent.toLowerCase().indexOf(qLower);
    source = rawContent;
  }

  if (hitIdx !== -1) {
    const start = Math.max(0, hitIdx - 12);
    const end = Math.min(source.length, hitIdx + q.length + 30);
    const excerpt = source.slice(start, end).trim();
    return `...${excerpt}...`;
  }

  return getCardDisplayTitle(c);
}
