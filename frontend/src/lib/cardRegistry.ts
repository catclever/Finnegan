/**
 * Finnegan Card Type Registry & Morphic Dispatch Engine
 * Task: card_type_registry_and_babel_decoupling_091
 *
 * Implements Layer 2 pure registry engine:
 * - Decouples card shapes (form) from modules/origins (source).
 * - Core base types: thought (default), note, chat_log (zero external module dependency).
 * - Generic shared shape: babel (bilingual intelligence & source echo, decoupled from bugu).
 * - Safe fallback to thought for missing, null, empty, or unknown types.
 */

import type { CardTypeDefinition } from '../types/card';

export const DEFAULT_CARD_TYPE = 'thought';

export const CORE_CARD_TYPES = ['thought', 'note', 'chat_log'] as const;
export const SHARED_CARD_TYPES = ['babel'] as const;

export const BUILTIN_CARD_DEFINITIONS: Record<string, CardTypeDefinition> = {
  thought: {
    type: 'thought',
    name: '灵感',
    description: '核心底座：随手捕获的灵感与意识流卡片',
    isDefault: true,
    defaultTitle: 'Thought'
  },
  note: {
    type: 'note',
    name: '备忘',
    description: '核心底座：确定性事实备忘、清单与代码速记卡片',
    defaultTitle: 'Note'
  },
  chat_log: {
    type: 'chat_log',
    name: '对话录',
    description: '核心底座：人机多轮对话与对话剧本卡片',
    defaultTitle: 'Chat Log'
  },
  babel: {
    type: 'babel',
    name: '巴别塔',
    description: '通用共享：跨语种双语情报与溯源回声卡片',
    defaultTitle: 'Babel Echo'
  }
};

/**
 * Normalizes card type string; falls back to 'thought' if empty or not a string.
 */
export function normalizeCardType(type?: string | null): string {
  if (!type || typeof type !== 'string') {
    return DEFAULT_CARD_TYPE;
  }
  const clean = type.trim();
  return clean || DEFAULT_CARD_TYPE;
}

/**
 * Checks whether a type is one of the native core base types.
 */
export function isCoreCardType(type?: string | null): boolean {
  const norm = normalizeCardType(type);
  return (CORE_CARD_TYPES as readonly string[]).includes(norm);
}

/**
 * Retrieves the default title for a given card type.
 */
export function getCardDefaultTitle(type?: string | null): string {
  const norm = normalizeCardType(type);
  const def = BUILTIN_CARD_DEFINITIONS[norm];
  if (def?.defaultTitle) {
    return def.defaultTitle;
  }
  return BUILTIN_CARD_DEFINITIONS[DEFAULT_CARD_TYPE].defaultTitle || 'Thought';
}

/**
 * Generic pure CardRegistry class that holds component mappings and metadata.
 */
export class CardRegistry<T = any> {
  private components = new Map<string, T>();
  private definitions = new Map<string, CardTypeDefinition>();
  private defaultType: string;
  private defaultComponent: T | null;

  constructor(defaultComponent?: T, defaultType: string = DEFAULT_CARD_TYPE) {
    this.defaultComponent = defaultComponent ?? null;
    this.defaultType = defaultType;
    this.resetDefinitions();
  }

  private resetDefinitions(): void {
    this.definitions.clear();
    for (const [key, def] of Object.entries(BUILTIN_CARD_DEFINITIONS)) {
      this.definitions.set(key, { ...def });
    }
  }

  /**
   * Registers a card shape component and optional metadata.
   */
  register(type: string, component: T, meta?: Partial<CardTypeDefinition>): void {
    if (!type || typeof type !== 'string') return;
    const cleanType = type.trim();
    if (!cleanType) return;

    this.components.set(cleanType, component);

    const existing = this.definitions.get(cleanType) || {
      type: cleanType,
      name: cleanType,
      defaultTitle: cleanType
    };
    this.definitions.set(cleanType, { ...existing, ...meta, type: cleanType });
  }

  /**
   * Registers or updates a card type metadata definition.
   */
  registerDefinition(def: CardTypeDefinition): void {
    if (!def?.type || typeof def.type !== 'string') return;
    const cleanType = def.type.trim();
    if (!cleanType) return;
    this.definitions.set(cleanType, { ...def, type: cleanType });
  }

  /**
   * Retrieves the registered component for a card type.
   * If type is missing, empty, or not found, safely falls back to default component.
   */
  get(type?: string | null): T {
    if (!type || typeof type !== 'string') {
      return this.getDefault();
    }
    const cleanType = type.trim();
    const found = this.components.get(cleanType);
    if (found !== undefined) {
      return found;
    }
    return this.getDefault();
  }

  /**
   * Checks whether a specific card type has a registered component.
   */
  has(type?: string | null): boolean {
    if (!type || typeof type !== 'string') return false;
    return this.components.has(type.trim());
  }

  /**
   * Gets metadata definition for a card type.
   */
  getDefinition(type?: string | null): CardTypeDefinition {
    const norm = normalizeCardType(type);
    const def = this.definitions.get(norm);
    if (def) return def;
    return this.definitions.get(this.defaultType) || BUILTIN_CARD_DEFINITIONS[DEFAULT_CARD_TYPE];
  }

  /**
   * Returns all registered card type definitions.
   */
  getAllDefinitions(): CardTypeDefinition[] {
    return Array.from(this.definitions.values());
  }

  /**
   * Clears all registered components and restores definitions to builtins.
   */
  clear(): void {
    this.components.clear();
    this.resetDefinitions();
  }

  private getDefault(): T {
    const defaultFromMap = this.components.get(this.defaultType);
    if (defaultFromMap !== undefined) {
      return defaultFromMap;
    }
    if (this.defaultComponent !== null) {
      return this.defaultComponent;
    }
    throw new Error(`CardRegistry has no default component registered for type '${this.defaultType}'.`);
  }
}

/**
 * Factory function to create a new CardRegistry.
 */
export function createCardRegistry<T = any>(defaultComponent?: T, defaultType: string = DEFAULT_CARD_TYPE): CardRegistry<T> {
  return new CardRegistry<T>(defaultComponent, defaultType);
}
