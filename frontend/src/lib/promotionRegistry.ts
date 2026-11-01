/**
 * Finnegan Cosmic Stage - Module Overridden Promotion Registry
 * Task: module_overridden_promotion_and_scattered_deck_stacking_140
 *
 * Defines strategy for promoting temporary network cards to permanent local cards:
 * - Default: Standalone collection with open-book icon.
 * - Modules (e.g. bugu, atproto): In-place expandable comment strip with anchor comment icon,
 *   attaching as child card to mother card with user comment.
 */

import type { ModuleManifest } from '../types/card';

export type PromotionIconType = 'default_book' | 'anchor_comment' | string;

export interface PromotionStrategy {
  icon: PromotionIconType;
  tooltip: string;
  hasCommentPrompt: boolean;
  placeholder: string;
  bindMotherCard: boolean;
}

export const DEFAULT_PROMOTION_STRATEGY: PromotionStrategy = {
  icon: 'default_book',
  tooltip: '收集转正为此地卡片',
  hasCommentPrompt: false,
  placeholder: '',
  bindMotherCard: false,
};

export const BUGU_PROMOTION_STRATEGY: PromotionStrategy = {
  icon: 'anchor_comment',
  tooltip: '附着母卡收录（附带批注）',
  hasCommentPrompt: true,
  placeholder: '留下附着批注（回车直接收录）...',
  bindMotherCard: true,
};

export const ATPROTO_PROMOTION_STRATEGY: PromotionStrategy = {
  icon: 'anchor_comment',
  tooltip: '附着母卡收录（附带批注）',
  hasCommentPrompt: true,
  placeholder: '留下附着批注（回车直接收录）...',
  bindMotherCard: true,
};

// Global strategy, theme, and beacon registries
const strategyMap = new Map<string, PromotionStrategy>();
const themeColorMap = new Map<string, string>();
const beaconTypeMap = new Map<string, string>();

function initDefaultPresets(): void {
  strategyMap.set('bugu', BUGU_PROMOTION_STRATEGY);
  strategyMap.set('atproto', ATPROTO_PROMOTION_STRATEGY);

  themeColorMap.set('bugu', '#f59e0b');
  themeColorMap.set('atproto', '#38bdf8');

  beaconTypeMap.set('bugu', 'radar');
  beaconTypeMap.set('bugu:radar', 'radar');
  beaconTypeMap.set('atproto', 'butterfly');
  beaconTypeMap.set('atproto:echo', 'butterfly');
  beaconTypeMap.set('bsky', 'butterfly');
}

// Register default built-in module presets
initDefaultPresets();

/**
 * Registers a new promotion strategy dynamically for a module or origin.
 */
export function registerPromotionStrategy(key: string, strategy: PromotionStrategy): void {
  strategyMap.set(key.toLowerCase().trim(), strategy);
}

/**
 * Hydrates dynamic module manifests from backend (both built-in and external plugins).
 */
export function hydrateModuleManifests(manifests?: ModuleManifest[] | null): void {
  if (!Array.isArray(manifests)) return;
  for (const m of manifests) {
    if (!m || !m.name) continue;
    const modKey = m.name.toLowerCase().trim();

    if (m.promotionStrategy) {
      const strat: PromotionStrategy = {
        icon: (m.promotionStrategy.icon as PromotionIconType) || 'default_book',
        tooltip: m.promotionStrategy.tooltip || DEFAULT_PROMOTION_STRATEGY.tooltip,
        hasCommentPrompt: Boolean(m.promotionStrategy.hasCommentPrompt),
        placeholder: m.promotionStrategy.placeholder || '',
        bindMotherCard: Boolean(m.promotionStrategy.bindMotherCard),
      };
      strategyMap.set(modKey, strat);
      if (Array.isArray(m.cardTypes)) {
        for (const ct of m.cardTypes) {
          if (ct?.type) {
            strategyMap.set(ct.type.toLowerCase().trim(), strat);
          }
        }
      }
    }

    if (m.themeColor) {
      themeColorMap.set(modKey, m.themeColor);
    }

    if (m.beaconType) {
      beaconTypeMap.set(modKey, m.beaconType);
      if (Array.isArray(m.commands)) {
        for (const cmd of m.commands) {
          if (cmd?.id) {
            beaconTypeMap.set(cmd.id.toLowerCase().trim(), m.beaconType);
          }
          if (cmd?.namespace) {
            beaconTypeMap.set(cmd.namespace.toLowerCase().trim(), m.beaconType);
          }
        }
      }
    }
  }
}

/**
 * Retrieves the registered theme color for a module/origin.
 */
export function getModuleThemeColor(origin?: string | null): string | undefined {
  if (!origin) return undefined;
  return themeColorMap.get(origin.toLowerCase().trim());
}

/**
 * Retrieves the registered beacon type ('radar' | 'butterfly' | 'none') for a module, origin, or command ID.
 */
export function getModuleBeaconType(originOrCmd?: string | null): string | undefined {
  if (!originOrCmd) return undefined;
  const normalized = originOrCmd.toLowerCase().trim();
  if (beaconTypeMap.has(normalized)) {
    return beaconTypeMap.get(normalized);
  }
  const colonIdx = normalized.indexOf(':');
  if (colonIdx > 0) {
    const prefix = normalized.slice(0, colonIdx);
    if (beaconTypeMap.has(prefix)) {
      return beaconTypeMap.get(prefix);
    }
  }
  return undefined;
}

/**
 * Retrieves a promotion strategy by key (module or origin).
 * Falls back to DEFAULT_PROMOTION_STRATEGY if key is empty or not registered.
 */
export function getPromotionStrategy(key?: string | null): PromotionStrategy {
  if (!key) return DEFAULT_PROMOTION_STRATEGY;
  const normalized = key.toLowerCase().trim();
  return strategyMap.get(normalized) || DEFAULT_PROMOTION_STRATEGY;
}

/**
 * Resolves promotion strategy for a specific card by inspecting origin and type.
 */
export function getCardPromotionStrategy(card?: any): PromotionStrategy {
  if (!card) return DEFAULT_PROMOTION_STRATEGY;
  if (card.origin) {
    const originKey = String(card.origin).toLowerCase().trim();
    if (strategyMap.has(originKey)) {
      return strategyMap.get(originKey)!;
    }
  }
  if (card.type) {
    const typeKey = String(card.type).toLowerCase().trim();
    if (strategyMap.has(typeKey)) {
      return strategyMap.get(typeKey)!;
    }
  }
  return DEFAULT_PROMOTION_STRATEGY;
}

/**
 * Clears custom registrations and resets to built-in presets (useful for test isolation).
 */
export function resetPromotionStrategies(): void {
  strategyMap.clear();
  themeColorMap.clear();
  beaconTypeMap.clear();
  initDefaultPresets();
}
