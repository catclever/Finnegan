/**
 * Finnegan Cosmic Stage - Border Beam Registry & Morphic Dispatch Engine
 *
 * Decouples traveling border beam styles and visual themes from components:
 * - Module-injected presets: Bugu (amber single-segment), ATProto/Bluesky (ice-blue single-segment)
 * - Core base presets: Ink wash (水墨双段飞白，独享卡面微透漫射扫掠), Aura (搜索共振多色流转)
 * - Pure registry & resolver: cards declare origin/state, resolver matches registered preset.
 */

export interface BorderBeamPreset {
  id: string;
  gradient: string;
  duration: string;
  surfaceScan: boolean;
  maskBackground?: string;
  maskTemporaryBackground?: string;
}

export interface BorderBeamContext {
  hasAura?: boolean;
  auraStage?: number;
  conicGradient?: string;
  isFocused?: boolean;
  isRecentVIP?: boolean;
  isTemporary?: boolean;
  isBuguScanning?: boolean;
  isBskyScanning?: boolean;
}

export interface ResolvedBorderBeam extends BorderBeamPreset {
  delay: number;
}

// Global registry for modular border beams
const beamRegistry = new Map<string, BorderBeamPreset>();

/**
 * Register a border beam preset for a module or source origin.
 */
export function registerBorderBeam(key: string, preset: BorderBeamPreset): void {
  beamRegistry.set(key.toLowerCase().trim(), preset);
}

/**
 * Retrieve a registered border beam preset.
 */
export function getBorderBeam(key: string): BorderBeamPreset | undefined {
  return beamRegistry.get(key.toLowerCase().trim());
}

/**
 * Retrieve all registered presets.
 */
export function getAllBorderBeams(): BorderBeamPreset[] {
  return Array.from(beamRegistry.values());
}

/**
 * Clear all registered presets (primarily for unit testing reset).
 */
export function clearBorderBeams(): void {
  beamRegistry.clear();
}

// --- Built-in Core Presets ---

export const INK_BEAM_PRESET: BorderBeamPreset = {
  id: 'ink',
  gradient: `conic-gradient(
    from 0deg,
    transparent 0deg,
    transparent 135deg,
    rgba(148, 163, 184, 0.20) 155deg,
    rgba(100, 116, 139, 0.45) 175deg,
    transparent 185deg,
    transparent 220deg,
    rgba(148, 163, 184, 0.22) 245deg,
    rgba(100, 116, 139, 0.52) 280deg,
    rgba(71, 85, 105, 0.72) 315deg,
    rgba(71, 85, 105, 0.88) 345deg,
    rgba(51, 65, 85, 0.88) 355deg,
    transparent 360deg
  )`,
  duration: '11.4s',
  surfaceScan: true, // 独享：水墨飞白微透漫射扫描光影
  maskBackground: 'rgba(255, 255, 255, 0.91)',
  maskTemporaryBackground: 'linear-gradient(135deg, rgba(226, 236, 248, 0.88) 0%, rgba(210, 222, 238, 0.80) 100%)',
};

// Register core ink preset
registerBorderBeam('ink', INK_BEAM_PRESET);

/**
 * Pure dispatch function that resolves which border beam preset applies to a given card.
 */
export function resolveCardBorderBeam(card: any, ctx: BorderBeamContext = {}): ResolvedBorderBeam | null {
  if (!card) return null;

  const cardId = card.id || '';
  const animDelay = -((cardId ? cardId.charCodeAt(cardId.length - 1) : 0) % 5);
  const hash = cardId ? cardId.split('').reduce((a: number, b: string) => a + b.charCodeAt(0), 0) : 0;
  const auraDelay = -((hash % 40) / 10);

  // 1. Priority: Aura Search/Tag Resonance Glow
  if (ctx.hasAura && (ctx.auraStage ?? 0) < 3) {
    return {
      id: 'aura',
      gradient: ctx.conicGradient || '',
      duration: ctx.auraStage === 2 ? '9s' : '4s',
      delay: auraDelay,
      surfaceScan: false,
      maskBackground: 'rgba(255, 255, 255, 0.98)',
      maskTemporaryBackground: 'linear-gradient(135deg, #e2ecf8 0%, #d2def0 100%)',
    };
  }

  // 2. Focused state: all edge traveling beams are suppressed in favor of focused frame
  if (ctx.isFocused) {
    return null;
  }

  // 3. Active Module Radar / Echo Scanning on Mother Card
  if (ctx.isBuguScanning) {
    const buguPreset = beamRegistry.get('bugu');
    if (buguPreset) {
      return {
        ...buguPreset,
        delay: animDelay,
      };
    }
  }

  if (ctx.isBskyScanning) {
    const bskyPreset = beamRegistry.get('atproto') || beamRegistry.get('bsky');
    if (bskyPreset) {
      return {
        ...bskyPreset,
        delay: animDelay,
      };
    }
  }

  // 4. Module Origin Traveling Beam (e.g. Bugu, Bluesky/ATProto)
  if (card.origin) {
    const originKey = String(card.origin).toLowerCase().trim();
    const preset = beamRegistry.get(originKey);
    if (preset) {
      return {
        ...preset,
        delay: animDelay,
      };
    }
  }

  // 5. Recent VIP New Card Inspiration (水墨双段飞白)
  if (ctx.isRecentVIP) {
    const inkPreset = beamRegistry.get('ink') || INK_BEAM_PRESET;
    return {
      ...inkPreset,
      delay: animDelay,
    };
  }

  return null;
}
