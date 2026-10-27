/**
 * Card Type Prefix Parser
 * 
 * Implements prefix syntax `:cardType ` (no leading space at start, single space at end)
 * reserved for generating different card types without command system coupling.
 */

export interface ParsedCardType {
  hasTypePrefix: boolean;
  type: string | null;
  content: string;
}

const CARD_TYPE_PREFIX_REGEX = /^:([a-zA-Z0-9_-]+)\s([\s\S]*)$/;

export function parseCardTypePrefix(text: string): ParsedCardType {
  const match = text.match(CARD_TYPE_PREFIX_REGEX);
  if (!match) {
    return {
      hasTypePrefix: false,
      type: null,
      content: text,
    };
  }

  return {
    hasTypePrefix: true,
    type: match[1].toLowerCase(),
    content: match[2],
  };
}
