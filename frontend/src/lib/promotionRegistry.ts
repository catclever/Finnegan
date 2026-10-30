/**
 * Finnegan Cosmic Stage - Module Overridden Promotion Registry
 * Task: module_overridden_promotion_and_scattered_deck_stacking_140
 *
 * Defines strategy for promoting temporary network cards to permanent local cards:
 * - Default: Standalone collection with open-book icon.
 * - Modules (e.g. bugu, atproto): In-place expandable comment strip with anchor comment icon,
 *   attaching as child card to mother card with user comment.
 */

export type PromotionIconType = 'default_book' | 'anchor_comment';

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

// Global strategy registry
const strategyMap = new Map<string, PromotionStrategy>();

// Register default built-in module strategies
strategyMap.set('bugu', BUGU_PROMOTION_STRATEGY);
strategyMap.set('atproto', ATPROTO_PROMOTION_STRATEGY);

/**
 * Registers a new promotion strategy dynamically for a module or origin.
 */
export function registerPromotionStrategy(key: string, strategy: PromotionStrategy): void {
  strategyMap.set(key.toLowerCase().trim(), strategy);
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
  strategyMap.set('bugu', BUGU_PROMOTION_STRATEGY);
  strategyMap.set('atproto', ATPROTO_PROMOTION_STRATEGY);
}
