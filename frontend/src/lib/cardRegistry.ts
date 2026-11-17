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

import type { CardTypeDefinition, CardTypeSpec, ModuleManifest } from '../types/card';

export const DEFAULT_CARD_TYPE = 'thought';

export const CORE_CARD_TYPES = ['thought', 'note', 'chat_log'] as const;
export const SHARED_CARD_TYPES = ['babel'] as const;

export const BUILTIN_CARD_DEFINITIONS: Record<string, CardTypeDefinition> = {
  thought: {
    type: 'thought',
    name: '灵感',
    description: '核心底座：随手捕获的灵感与意识流卡片 (默认底座，无需前缀)',
    isDefault: true,
    defaultTitle: 'Thought',
    icon: '💡',
    selectable: false
  },
  note: {
    type: 'note',
    name: '结构长文',
    description: '确定性事实备忘、清单与代码长文',
    defaultTitle: 'Note',
    icon: '📝',
    prefix: ':note',
    selectable: true
  },
  chat_log: {
    type: 'chat_log',
    name: '对话录',
    description: '核心底座：人机多轮对话与对话剧本卡片 (格式驱动)',
    defaultTitle: 'Chat Log',
    icon: '💬',
    selectable: false
  },
  babel: {
    type: 'babel',
    name: '巴别塔',
    description: '通用共享：跨语种双语情报与溯源回声卡片 (回声驱动)',
    defaultTitle: 'Babel Echo',
    icon: '🗼',
    selectable: false
  },
  book: {
    type: 'book',
    name: '读书笔记',
    description: '藏书票书摘：原文引文与个人批注',
    defaultTitle: 'Reading Note',
    icon: '📖',
    prefix: ':book',
    selectable: true
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
   * Unregisters a card shape component and its metadata definition.
   * If unregistered, looking up this type safely falls back to the default component (Thought).
   */
  unregister(type: string): void {
    if (!type || typeof type !== 'string') return;
    const cleanType = type.trim();
    this.components.delete(cleanType);
    this.definitions.delete(cleanType);
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
   * Checks whether a specific card type has an explicit metadata definition.
   */
  hasDefinition(type?: string | null): boolean {
    if (!type || typeof type !== 'string') return false;
    return this.definitions.has(type.trim());
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
   * Retrieves only the card type definitions that are explicitly marked as selectable in UI.
   */
  getSelectableDefinitions(): CardTypeDefinition[] {
    return Array.from(this.definitions.values()).filter(def => def.selectable === true);
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

/**
 * Appends a cache-busting query parameter (?t=... or &t=...) when cacheBust is provided.
 */
export function appendCacheBust(url: string, cacheBust?: number): string {
  if (!url || !cacheBust) return url;
  const sep = url.includes('?') ? '&' : '?';
  return `${url}${sep}t=${cacheBust}`;
}

/**
 * Builds a normalized CardTypeDefinition from a ModuleManifest CardTypeSpec,
 * preserving existing metadata fields when spec omits them.
 */
export function buildCardTypeDefinitionFromSpec(
  spec: CardTypeSpec,
  manifest: ModuleManifest,
  existing?: CardTypeDefinition
): CardTypeDefinition {
  const cleanType = spec.type.trim();
  const baseName = existing?.name || spec.name || cleanType;
  return {
    ...(existing || {}),
    type: cleanType,
    name: spec.name || baseName,
    defaultTitle: existing?.defaultTitle || spec.name || cleanType,
    description: spec.description ?? existing?.description ?? manifest.description,
    icon: spec.icon ?? existing?.icon,
    prefix: spec.prefix ?? existing?.prefix,
    submitLabel: spec.submitLabel ?? existing?.submitLabel,
    badgeColor: spec.badgeColor ?? existing?.badgeColor ?? manifest.themeColor,
    selectable: spec.selectable !== undefined ? spec.selectable : existing?.selectable,
    customElementTag: spec.customElementTag ?? existing?.customElementTag,
    shellStyle: spec.shellStyle ?? existing?.shellStyle,
    moduleName: spec.moduleName ?? manifest.name ?? existing?.moduleName,
  };
}

export interface ExternalCardSpecEntry {
  manifest: ModuleManifest;
  spec: CardTypeSpec;
}

export interface ReconcileExternalResult {
  removedUITypes: string[];
  removedDefTypes: string[];
  activeUISpecs: ExternalCardSpecEntry[];
}

/**
 * Pure reconciliation of external card definitions and UI specs against active manifests.
 * Unregisters any previously tracked external types absent from manifests so cards fall back to ThoughtCardBody.
 */
export function reconcileExternalCardTypes(
  registry: CardRegistry<any>,
  manifests?: ModuleManifest[] | null,
  externalRegisteredTypes?: Set<string>,
  externalDefinitionTypes?: Set<string>
): ReconcileExternalResult {
  const manifestList = Array.isArray(manifests) ? manifests : [];
  const nextUITypes = new Set<string>();
  const nextManifestTypes = new Set<string>();
  const activeUISpecs: ExternalCardSpecEntry[] = [];

  for (const m of manifestList) {
    if (!m?.cardTypes || !Array.isArray(m.cardTypes)) continue;
    for (const spec of m.cardTypes) {
      if (!spec?.type || typeof spec.type !== 'string') continue;
      const cleanType = spec.type.trim();
      if (!cleanType) continue;
      nextManifestTypes.add(cleanType);
      if (spec.uiEntry || spec.customElementTag) {
        nextUITypes.add(cleanType);
        activeUISpecs.push({ manifest: m, spec: { ...spec, type: cleanType } });
      }
    }
  }

  const removedUITypes: string[] = [];
  if (externalRegisteredTypes) {
    for (const prevType of Array.from(externalRegisteredTypes)) {
      if (!nextUITypes.has(prevType)) {
        externalRegisteredTypes.delete(prevType);
        registry.unregister(prevType);
        removedUITypes.push(prevType);
      }
    }
  }

  const removedDefTypes: string[] = [];
  if (externalDefinitionTypes) {
    for (const prevDefType of Array.from(externalDefinitionTypes)) {
      if (!nextManifestTypes.has(prevDefType) && !(prevDefType in BUILTIN_CARD_DEFINITIONS)) {
        externalDefinitionTypes.delete(prevDefType);
        registry.unregister(prevDefType);
        removedDefTypes.push(prevDefType);
      }
    }
  }

  for (const m of manifestList) {
    if (!m?.cardTypes || !Array.isArray(m.cardTypes)) continue;
    for (const spec of m.cardTypes) {
      if (!spec?.type || typeof spec.type !== 'string') continue;
      const cleanType = spec.type.trim();
      if (!cleanType) continue;
      const existing = registry.hasDefinition(cleanType) ? registry.getDefinition(cleanType) : undefined;
      const def = buildCardTypeDefinitionFromSpec({ ...spec, type: cleanType }, m, existing);
      registry.registerDefinition(def);
      if (!(cleanType in BUILTIN_CARD_DEFINITIONS)) {
        externalDefinitionTypes?.add(cleanType);
      }
    }
  }

  return { removedUITypes, removedDefTypes, activeUISpecs };
}
