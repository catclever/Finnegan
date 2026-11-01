/**
 * Finnegan Bugu Module - Amber Gold Border Beam Preset
 *
 * Defines and registers the Bugu module's traveling border beam:
 * - Clean single-segment amber gold comet tail (290° ~ 360°)
 * - 4.6s steady rotation period
 * - Zero surface bleed (surfaceScan: false) with opaque mask
 */

import { registerBorderBeam, type BorderBeamPreset } from '../../lib/borderBeamRegistry.ts';

export const BUGU_BEAM_PRESET: BorderBeamPreset = {
  id: 'bugu',
  gradient: `conic-gradient(
    from 0deg,
    transparent 0deg,
    transparent 290deg,
    rgba(245, 158, 11, 0) 295deg,
    rgba(245, 158, 11, 0.25) 325deg,
    rgba(251, 191, 36, 0.75) 350deg,
    rgba(254, 240, 138, 1) 360deg
  )`,
  duration: '4.6s',
  surfaceScan: false, // 严格单段彗尾，卡面完全阻断透光
  maskBackground: 'linear-gradient(135deg, #eef5fc 0%, #e1ebf7 100%)',
  maskTemporaryBackground: 'linear-gradient(135deg, #eef5fc 0%, #e1ebf7 100%)',
};

// Self-registering on import
registerBorderBeam('bugu', BUGU_BEAM_PRESET);
