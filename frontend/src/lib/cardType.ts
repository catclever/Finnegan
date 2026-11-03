/**
 * Card Type Prefix Parser & Dropdown Options Engine
 * 
 * Implements prefix syntax `:cardType ` (no leading space at start, single space at end)
 * reserved for generating different card types without command system coupling.
 * Also provides card type dropdown trigger detection and dynamic option listing.
 */

import type { CardTypeDefinition, ModuleManifest } from '../types/card';
import { BUILTIN_CARD_DEFINITIONS, isCoreCardType } from './cardRegistry.ts';

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
 * Converts a CardTypeDefinition into a UI CardTypeOption for dropdown display.
 */
export function definitionToCardTypeOption(def: CardTypeDefinition): CardTypeOption {
  return {
    type: def.type,
    prefix: def.prefix || `:${def.type}`,
    label: def.name || def.type,
    description: def.description || '',
    icon: def.icon || '📄',
    isCore: isCoreCardType(def.type),
  };
}

/**
 * Returns available card type options based on selectable registry definitions and active module manifests.
 * Default thought cards (handled with zero-prefix) and format-driven chat_log cards are NOT selectable in the dropdown.
 */
export function getAvailableCardTypeOptions(
  activeManifests?: ModuleManifest[] | null,
  customDefinitions?: CardTypeDefinition[]
): CardTypeOption[] {
  // If explicit definitions are provided, use only selectable ones
  if (Array.isArray(customDefinitions) && customDefinitions.length > 0) {
    return customDefinitions
      .filter(def => def.selectable === true)
      .map(definitionToCardTypeOption);
  }

  // Derive base selectable options from built-in card registry definitions
  const options: CardTypeOption[] = Object.values(BUILTIN_CARD_DEFINITIONS)
    .filter(def => def.selectable === true)
    .map(definitionToCardTypeOption);

  // Modular extension: curio
  const hasCurio = Array.isArray(activeManifests)
    ? activeManifests.some(
        m => m?.name === 'curio' || (m?.cardTypes && m.cardTypes.some(ct => ct?.type === 'curio'))
      )
    : true; // Default fallback before manifests load

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
