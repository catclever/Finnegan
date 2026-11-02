/**
 * Card Type Prefix Parser & Dropdown Options Engine
 * 
 * Implements prefix syntax `:cardType ` (no leading space at start, single space at end)
 * reserved for generating different card types without command system coupling.
 * Also provides card type dropdown trigger detection and dynamic option listing.
 */

import type { ModuleManifest } from '../types/card';

export interface ParsedCardType {
  hasTypePrefix: boolean;
  type: string | null;
  content: string;
}

export interface CardTypeOption {
  type: string;
  prefix: string; // e.g. ":thought", ":note", ":curio", ":chat_log"
  label: string;  // e.g. "灵感碎片", "结构长文", "奇物标本"
  description: string;
  icon: string;
  isCore?: boolean;
}

export interface CardTypeTrigger {
  isTriggered: boolean;
  query: string;
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

/**
 * Detects if the current text is triggering the card type dropdown.
 * Matches when text starts with ':' and does NOT yet contain a trailing space (e.g. ":", ":c", ":curio").
 */
export function detectCardTypeTrigger(text: string): CardTypeTrigger {
  if (typeof text !== 'string') {
    return { isTriggered: false, query: '' };
  }
  const match = text.match(/^:([a-zA-Z0-9_-]*)$/);
  if (match) {
    return {
      isTriggered: true,
      query: match[1].toLowerCase(),
    };
  }
  return {
    isTriggered: false,
    query: '',
  };
}

/**
 * Returns available card type options based on core types and active module manifests.
 */
export function getAvailableCardTypeOptions(activeManifests?: ModuleManifest[] | null): CardTypeOption[] {
  const options: CardTypeOption[] = [
    {
      type: 'thought',
      prefix: ':thought',
      label: '灵感碎片',
      description: '即时思考与散落意识流 (默认底座)',
      icon: '💡',
      isCore: true,
    },
    {
      type: 'note',
      prefix: ':note',
      label: '结构长文',
      description: '代码备忘、清单与结构化长文',
      icon: '📝',
      isCore: true,
    },
    {
      type: 'chat_log',
      prefix: ':chat_log',
      label: '对话录',
      description: '剧本式双人对话与多轮交谈',
      icon: '💬',
      isCore: true,
    },
  ];

  // If activeManifests is provided, check for modular extensions (like curio)
  if (Array.isArray(activeManifests)) {
    const hasCurio = activeManifests.some(
      m => m?.name === 'curio' || (m?.cardTypes && m.cardTypes.some(ct => ct?.type === 'curio'))
    );
    if (hasCurio) {
      options.push({
        type: 'curio',
        prefix: ':curio',
        label: '奇物标本',
        description: '商品网页快照与恋物藏品 (Curio 模块驱动)',
        icon: '🏺',
        isCore: false,
      });
    }
  } else {
    // Default fallback when manifests have not yet loaded
    options.push({
      type: 'curio',
      prefix: ':curio',
      label: '奇物标本',
      description: '商品网页快照与恋物藏品 (Curio 模块驱动)',
      icon: '🏺',
      isCore: false,
    });
  }

  return options;
}

/**
 * Filters card type options by query string.
 */
export function filterCardTypeOptions(options: CardTypeOption[], query: string): CardTypeOption[] {
  if (!Array.isArray(options)) return [];
  const clean = (query || '').trim().toLowerCase();
  if (!clean) return options;

  return options.filter(opt => 
    opt.type.toLowerCase().includes(clean) ||
    opt.prefix.toLowerCase().includes(clean) ||
    opt.label.toLowerCase().includes(clean) ||
    opt.description.toLowerCase().includes(clean)
  );
}
