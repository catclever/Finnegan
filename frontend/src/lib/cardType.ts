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
  submitLabel?: string;
  badgeColor?: string;
  isExternal?: boolean;
  moduleName?: string;
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
  const rawPrefix = def.prefix || `:${def.type}`;
  return {
    type: def.type,
    prefix: rawPrefix.startsWith(':') ? rawPrefix : `:${rawPrefix}`,
    label: def.name || def.type,
    description: def.description || '',
    icon: def.icon || '📄',
    isCore: isCoreCardType(def.type),
    submitLabel: def.submitLabel,
    badgeColor: def.badgeColor,
    moduleName: def.moduleName,
  };
}

const BUILTIN_MODULE_CARD_DEFAULTS: Record<string, CardTypeOption> = {
  diary: {
    type: 'diary',
    prefix: ':diary',
    label: '羊皮卷日记',
    description: '流年手账：时辰天象与地理印记',
    icon: '📜',
    isCore: false,
    submitLabel: '封存手账',
    moduleName: 'diary',
  },
};

function normalizePrefix(prefix: string | undefined, type: string): string {
  const raw = (prefix || '').trim() || `:${type}`;
  return raw.startsWith(':') ? raw : `:${raw}`;
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
  const seenTypes = new Set<string>(options.map(o => o.type));

  // Default fallback before manifests load
  if (!Array.isArray(activeManifests)) {
    for (const key of ['diary']) {
      if (!seenTypes.has(key)) {
        seenTypes.add(key);
        options.push({ ...BUILTIN_MODULE_CARD_DEFAULTS[key] });
      }
    }
    return options;
  }

  // Dynamically discover selectable card types from all active module manifests
  for (const m of activeManifests) {
    if (!m) continue;
    const specs = Array.isArray(m.cardTypes) ? m.cardTypes : [];

    for (const spec of specs) {
      if (!spec?.type || typeof spec.type !== 'string') continue;
      const cleanType = spec.type.trim();
      if (!cleanType || seenTypes.has(cleanType)) continue;

      const builtinPreset = BUILTIN_MODULE_CARD_DEFAULTS[cleanType] || (m.name ? BUILTIN_MODULE_CARD_DEFAULTS[m.name] : undefined);
      const isBuiltinModuleMatch = Boolean(builtinPreset) && spec.selectable !== false;
      const isSelectable = spec.selectable === true || Boolean(spec.prefix) || isBuiltinModuleMatch;
      if (!isSelectable) continue;

      seenTypes.add(cleanType);
      options.push({
        type: cleanType,
        prefix: normalizePrefix(spec.prefix || builtinPreset?.prefix, cleanType),
        label: spec.name || builtinPreset?.label || cleanType,
        description: spec.description || m.description || builtinPreset?.description || '',
        icon: spec.icon || builtinPreset?.icon || '🧩',
        isCore: isCoreCardType(cleanType),
        submitLabel: spec.submitLabel || builtinPreset?.submitLabel || '入库',
        badgeColor: spec.badgeColor || m.themeColor || builtinPreset?.badgeColor,
        isExternal: Boolean(m.isExternal),
        moduleName: m.name,
      });
    }

    // Legacy fallback: manifest matches 'diary' by module name with empty cardTypes
    if (specs.length === 0 && m.name === 'diary' && !seenTypes.has(m.name)) {
      const preset = BUILTIN_MODULE_CARD_DEFAULTS[m.name];
      seenTypes.add(m.name);
      options.push({
        ...preset,
        description: m.description || preset.description,
        badgeColor: m.themeColor || preset.badgeColor,
        isExternal: Boolean(m.isExternal),
        moduleName: m.name,
      });
    }
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
