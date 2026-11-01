import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  registerBorderBeam,
  getBorderBeam,
  getAllBorderBeams,
  resolveCardBorderBeam,
  INK_BEAM_PRESET,
  type BorderBeamPreset
} from './borderBeamRegistry.ts';

// Import modular registrations
import '../modules/bugu/beam.ts';
import '../modules/atproto/beam.ts';

describe('Border Beam Registry & Morphic Dispatch Engine', () => {
  it('has built-in ink preset registered by default with exclusive surfaceScan', () => {
    const ink = getBorderBeam('ink');
    assert.ok(ink, 'ink preset should be registered');
    assert.strictEqual(ink.id, 'ink');
    assert.strictEqual(ink.surfaceScan, true, 'Ink wash exclusively retains surfaceScan: true');
    assert.strictEqual(ink.duration, '11.4s');
  });

  it('verifies Bugu module self-registers amber gold single-segment preset with zero surface scan', () => {
    const bugu = getBorderBeam('bugu');
    assert.ok(bugu, 'bugu preset should be registered');
    assert.strictEqual(bugu.id, 'bugu');
    assert.strictEqual(bugu.surfaceScan, false, 'Bugu must strictly have surfaceScan: false');
    assert.strictEqual(bugu.duration, '4.6s');
    assert.ok(bugu.gradient.includes('rgba(245, 158, 11'), 'gradient contains core amber gold');
    assert.ok(bugu.gradient.includes('transparent 290deg'), 'gradient starts transparent up to 290deg for single tail');
  });

  it('verifies ATProto / Bluesky module self-registers ice blue single-segment preset with zero surface scan', () => {
    const atproto = getBorderBeam('atproto');
    assert.ok(atproto, 'atproto preset should be registered');
    assert.strictEqual(atproto.id, 'atproto');
    assert.strictEqual(atproto.surfaceScan, false, 'ATProto must strictly have surfaceScan: false');
    assert.strictEqual(atproto.duration, '4.6s');
    assert.ok(atproto.gradient.includes('rgba(56, 189, 248'), 'gradient contains core ice blue');

    const bskyAlias = getBorderBeam('bsky');
    assert.ok(bskyAlias, 'bsky alias should also be registered');
    assert.strictEqual(bskyAlias.id, 'atproto');
  });

  it('supports dynamic registration of third-party or future module border beams', () => {
    const customPreset: BorderBeamPreset = {
      id: 'github',
      gradient: 'conic-gradient(from 0deg, transparent 280deg, #10b981 360deg)',
      duration: '5.2s',
      surfaceScan: false,
      maskBackground: '#ffffff'
    };

    registerBorderBeam('github', customPreset);
    const retrieved = getBorderBeam('github');
    assert.deepStrictEqual(retrieved, customPreset);
  });

  it('resolves active beam following strict priority cascade', () => {
    const card = {
      id: 'card-101',
      origin: 'bugu',
      echoOf: 'mother-100',
    };

    // Priority 1: Aura takes precedence over origin
    const auraBeam = resolveCardBorderBeam(card, {
      hasAura: true,
      auraStage: 1,
      conicGradient: 'conic-gradient(from 0deg, #ef4444, #3b82f6)'
    });
    assert.ok(auraBeam);
    assert.strictEqual(auraBeam.id, 'aura');
    assert.strictEqual(auraBeam.duration, '4s');
    assert.strictEqual(auraBeam.gradient, 'conic-gradient(from 0deg, #ef4444, #3b82f6)');

    // Aura Stage 2 slows down to 9s
    const slowAuraBeam = resolveCardBorderBeam(card, {
      hasAura: true,
      auraStage: 2,
      conicGradient: 'conic-gradient(from 0deg, #ef4444, #3b82f6)'
    });
    assert.strictEqual(slowAuraBeam?.duration, '9s');

    // Priority 2: Focused mode suppresses all edge traveling beams
    const focusedBeam = resolveCardBorderBeam(card, {
      isFocused: true,
      hasAura: false
    });
    assert.strictEqual(focusedBeam, null);

    // Priority 3: Module Origin Traveling Beam (Bugu)
    const buguBeam = resolveCardBorderBeam(card, {
      isFocused: false,
      hasAura: false
    });
    assert.ok(buguBeam);
    assert.strictEqual(buguBeam.id, 'bugu');
    assert.strictEqual(buguBeam.surfaceScan, false);
    assert.strictEqual(buguBeam.duration, '4.6s');

    // Bluesky origin
    const bskyCard = {
      id: 'card-202',
      origin: 'atproto',
      echoOf: 'mother-99'
    };
    const bskyBeam = resolveCardBorderBeam(bskyCard, {
      isFocused: false,
      hasAura: false
    });
    assert.ok(bskyBeam);
    assert.strictEqual(bskyBeam.id, 'atproto');
    assert.strictEqual(bskyBeam.surfaceScan, false);

    // Priority 4: Recent VIP Card (水墨飞白)
    const vipCard = {
      id: 'card-vip',
      origin: 'local',
      isVIP: true
    };
    const vipBeam = resolveCardBorderBeam(vipCard, {
      isRecentVIP: true,
      isFocused: false,
      hasAura: false
    });
    assert.ok(vipBeam);
    assert.strictEqual(vipBeam.id, 'ink');
    assert.strictEqual(vipBeam.surfaceScan, true);
    assert.strictEqual(vipBeam.duration, '11.4s');

    // Normal non-echo card without VIP or Aura -> null
    const plainCard = {
      id: 'card-plain',
      origin: 'local'
    };
    const plainBeam = resolveCardBorderBeam(plainCard, {
      isFocused: false,
      hasAura: false,
      isRecentVIP: false
    });
    assert.strictEqual(plainBeam, null);
  });
});
