/**
 * Finnegan ATProto / Bluesky Module - Ice Blue Border Beam Preset
 *
 * Defines and registers the ATProto / Bluesky module's traveling border beam:
 * - Clean single-segment ice blue comet tail (290° ~ 360°)
 * - 4.6s steady rotation period
 * - Zero surface bleed (surfaceScan: false) with opaque mask
 */

import { registerBorderBeam, type BorderBeamPreset } from '../../lib/borderBeamRegistry.ts';

export const BSKY_BEAM_PRESET: BorderBeamPreset = {
  id: 'atproto',
  gradient: `conic-gradient(
    from 0deg,
    transparent 0deg,
    transparent 290deg,
    rgba(56, 189, 248, 0) 295deg,
    rgba(56, 189, 248, 0.25) 325deg,
    rgba(125, 211, 252, 0.75) 350deg,
    rgba(224, 242, 254, 1) 360deg
  )`,
  duration: '4.6s',
  surfaceScan: false, // 严格单段彗尾，卡面完全阻断透光
  maskBackground: 'linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%)',
  maskTemporaryBackground: 'linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%)',
};

// Self-registering for both 'atproto' and 'bsky' aliases
registerBorderBeam('atproto', BSKY_BEAM_PRESET);
registerBorderBeam('bsky', BSKY_BEAM_PRESET);
