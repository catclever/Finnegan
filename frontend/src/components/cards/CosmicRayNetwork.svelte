<!--
  ======================================================================================
  CosmicRayNetwork.svelte - Universal Cosmic Ray Network & Staggered Cascading Flow
  Task: universal_cosmic_ray_network_staggered_cascade_132
  Unifies Mother-Child Radar Rays (Bugu/Bluesky) and Tag Synapse Rays into a single,
  high-performance ethereal beam engine with staggered cascading photon pulses.
  ======================================================================================
-->
<script lang="ts">
  import { onDestroy } from 'svelte';
  import type { CosmicCard } from '../../types/card';
  import type { BuguBeacon } from '../../modules/bugu/types';
  import BuguRadarBeacon from '../../modules/bugu/BuguRadarBeacon.svelte';
  import { computeAnimatedBeamPath, type BeamCoordinates } from '../../lib/beamMath';
  import { cardHasTag, cardHasKeyword, getCardTags } from '../../lib/tagNetwork';

  export let cards: CosmicCard[] = [];
  export let beacon: BuguBeacon | null = null;
  export let scrollExpansion: number = 0;
  export let hoveredMotherId: string | null = null;
  export let hoveredCardId: string | null = null;
  export let focusedCardId: string | null = null;
  export let activeConstellationTag: string | null = null;
  // Legacy session prop fallback
  export let session: any = null;
  $: void scrollExpansion;
  $: void activeConstellationTag;

  $: activeBeacon = beacon || (session ? { sourceCardId: session.sourceCardId, status: session.status } : null);

  $: beaconCard = activeBeacon 
    ? cards.find(c => c.id === activeBeacon.sourceCardId && !c.isFadingOut) 
    : null;

  interface CosmicRay {
    id: string;
    type: 'mother_child' | 'synapse';
    origin: string; // 'bugu' | 'atproto' | 'synapse'
    source: CosmicCard;
    target: CosmicCard;
    badgeText?: string;
    badgeTitle?: string;
    coords: BeamCoordinates;
    staggerIndex: number;
    isPermanent: boolean;
  }

  // 1. Mother-Child Rays (Bugu & Bluesky)
  $: motherChildRayDefs = (() => {
    const list: Array<{
      id: string;
      source: CosmicCard;
      target: CosmicCard;
      origin: string;
      matchedAnchor?: string;
      echoReason?: string;
    }> = [];

    for (const child of cards) {
      if ((child.origin === 'bugu' || child.origin === 'atproto') && child.echoOf && !child.isFadingOut) {
        const mother = cards.find(c => c.id === child.echoOf && !c.isFadingOut);
        if (mother) {
          list.push({
            id: `mc:${mother.id}->${child.id}`,
            source: mother,
            target: child,
            origin: child.origin,
            matchedAnchor: child.matchedAnchor,
            echoReason: child.echoReason,
          });
        }
      }
    }
    return list;
  })();

  // 2. Synapse Rays (Hovered Tag Association Network)
  $: synapseRayDefs = (() => {
    if (focusedCardId !== null || !hoveredCardId) return [];

    const hoveredCard = cards.find(c => c.id === hoveredCardId && !c.isFadingOut);
    if (!hoveredCard) return [];

    const hoveredTags = getCardTags(hoveredCard);
    if (hoveredTags.length === 0) return [];

    const list: Array<{
      id: string;
      source: CosmicCard;
      target: CosmicCard;
      tag: string;
    }> = [];

    const seen = new Set<string>();
    for (const other of cards) {
      if (other.id !== hoveredCard.id && !other.isFadingOut) {
        const sharedTag = hoveredTags.find(tag => cardHasTag(other, tag) || cardHasKeyword(other, tag));
        if (sharedTag) {
          const key = [hoveredCard.id, other.id].sort().join('--');
          if (!seen.has(key)) {
            seen.add(key);
            list.push({
              id: `syn:${key}`,
              source: hoveredCard,
              target: other,
              tag: sharedTag,
            });
          }
        }
      }
    }
    return list;
  })();

  // Periodic ambient pulse scheduler for Mother-Child rays
  const SINGLE_PULSE_MS = 4800;
  const MIN_NEXT_DELAY_MS = SINGLE_PULSE_MS;
  const RANDOM_ADDITIONAL_MS = 2400;

  let activeAmbientRayIds: string[] = [];
  let ambientSpawnTimer: any = null;
  let lastSpawnedId = '';

  function spawnNextAmbientRay() {
    if (motherChildRayDefs.length === 0) {
      ambientSpawnTimer = null;
      return;
    }

    const candidates = motherChildRayDefs.filter(r => r.id !== lastSpawnedId);
    const selected = candidates.length > 0 
      ? candidates[Math.floor(Math.random() * candidates.length)]
      : motherChildRayDefs[0];

    lastSpawnedId = selected.id;
    activeAmbientRayIds = [selected.id];

    const delay = MIN_NEXT_DELAY_MS + Math.floor(Math.random() * RANDOM_ADDITIONAL_MS);
    ambientSpawnTimer = setTimeout(() => {
      ambientSpawnTimer = null;
      spawnNextAmbientRay();
    }, delay);
  }

  $: {
    if (motherChildRayDefs.length > 0) {
      if (!ambientSpawnTimer && activeAmbientRayIds.length === 0) {
        spawnNextAmbientRay();
      }
    } else {
      if (ambientSpawnTimer) {
        clearTimeout(ambientSpawnTimer);
        ambientSpawnTimer = null;
      }
      activeAmbientRayIds = [];
    }
  }

  onDestroy(() => {
    if (ambientSpawnTimer) clearTimeout(ambientSpawnTimer);
  });

  // Consolidated Rendered Rays with Staggered Cascading
  $: renderedRays = (() => {
    // Hide all rays during full modal focus
    if (focusedCardId !== null) return [];

    const rays: CosmicRay[] = [];
    let cascadeIndex = 0;

    // A. Mother-Child Rays
    for (const def of motherChildRayDefs) {
      const isMotherHovered = hoveredMotherId !== null && (hoveredMotherId === def.source.id || hoveredMotherId === def.target.id);
      const isScanning = activeBeacon && activeBeacon.sourceCardId === def.source.id && activeBeacon.status === 'scanning';
      const isAmbientActive = activeAmbientRayIds.includes(def.id);

      // Render if hovered, during active scanning, or triggered by ambient pulse
      if (isMotherHovered || isScanning || isAmbientActive) {
        const coords = computeAnimatedBeamPath(def.source, def.target);
        rays.push({
          id: def.id,
          type: 'mother_child',
          origin: def.origin,
          source: def.source,
          target: def.target,
          badgeText: def.matchedAnchor,
          badgeTitle: def.echoReason || def.matchedAnchor,
          coords,
          staggerIndex: cascadeIndex++,
          isPermanent: Boolean(isMotherHovered),
        });
      }
    }

    // B. Synapse Rays (Cascading Ice Blue)
    for (const syn of synapseRayDefs) {
      const coords = computeAnimatedBeamPath(syn.source, syn.target);
      rays.push({
        id: syn.id,
        type: 'synapse',
        origin: 'atproto', // Synapse rays use ice blue theme
        source: syn.source,
        target: syn.target,
        badgeText: syn.tag,
        badgeTitle: `关联标签: ${syn.tag}`,
        coords,
        staggerIndex: cascadeIndex++,
        isPermanent: true, // While hovered, maintain steady connection
      });
    }

    return rays;
  })();
</script>

<div class="cosmic-ray-network">
  <!-- Mother Card Radar Beacon -->
  {#if beaconCard && activeBeacon}
    <BuguRadarBeacon 
      x={beaconCard.x} 
      y={beaconCard.y} 
      status={activeBeacon.status} 
      theme={beaconCard.origin === 'atproto' ? 'ice-blue' : 'amber'}
    />
  {/if}

  {#if renderedRays.length > 0}
    <!-- HTML Layer: Anchor Badges with 3D Depth & Staggered Fade In -->
    <div class="cosmic-flat-html">
      {#each renderedRays as ray (ray.id)}
        {#if ray.badgeText}
          <div 
            class="ray-anchor-badge-container"
            style="transform: translate(calc({ray.coords.midX}px - 50%), calc({ray.coords.midY}px - 50%)) translateZ(8px); animation-delay: {ray.staggerIndex * 90}ms;"
          >
            <div 
              class="ray-anchor-badge"
              class:is-bsky={ray.origin === 'atproto'}
              class:is-synapse={ray.type === 'synapse'}
              class:permanent-badge={ray.isPermanent}
              title={ray.badgeTitle || ray.badgeText}
            >
              {ray.badgeText}
            </div>
          </div>
        {/if}
      {/each}
    </div>

    <!-- SVG Layer: Animated Beams, Cascading Traveling Pulses & Aperture Blooms -->
    <svg class="cosmic-flat-svg">
      <defs>
        <!-- Soft glowing radial gradient for aperture nodes (Amber Bugu) -->
        <radialGradient id="cosmic-amber-bloom" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#f59e0b" stop-opacity="1" />
          <stop offset="35%" stop-color="#f59e0b" stop-opacity="0.65" />
          <stop offset="70%" stop-color="#f59e0b" stop-opacity="0.2" />
          <stop offset="100%" stop-color="#f59e0b" stop-opacity="0" />
        </radialGradient>

        <!-- Soft glowing radial gradient for aperture nodes (Ice Blue Bluesky & Synapse) -->
        <radialGradient id="cosmic-iceblue-bloom" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#bae6fd" stop-opacity="1" />
          <stop offset="35%" stop-color="#38bdf8" stop-opacity="0.65" />
          <stop offset="70%" stop-color="#0ea5e9" stop-opacity="0.2" />
          <stop offset="100%" stop-color="#0284c7" stop-opacity="0" />
        </radialGradient>
      </defs>

      {#each renderedRays as ray (ray.id)}
        <g 
          class="ray-instance" 
          class:is-bsky={ray.origin === 'atproto'}
          class:is-synapse={ray.type === 'synapse'}
          style="--cascade-delay: {ray.staggerIndex * 90}ms;"
        >
          <!-- Source Node Aperture Bloom -->
          <g class="aperture-group" transform="translate({ray.coords.x1}, {ray.coords.y1})">
            <circle 
              cx="0" cy="0" r="22" 
              fill={ray.origin === 'atproto' ? "url(#cosmic-iceblue-bloom)" : "url(#cosmic-amber-bloom)"} 
              class="glow-ring source-bloom" 
              class:permanent-bloom={ray.isPermanent} 
              style="animation-delay: {ray.staggerIndex * 90}ms;"
            />
            <circle 
              cx="0" cy="0" r="3.2" 
              fill={ray.origin === 'atproto' ? "#38bdf8" : "#f59e0b"} 
              class="core-dot source-bloom" 
              class:permanent-bloom={ray.isPermanent} 
              style="animation-delay: {ray.staggerIndex * 90}ms;"
            />
          </g>

          <!-- Ambient Fiber Beam Track (Cascading Fade In) -->
          <path
            d={ray.coords.path}
            class="beam-track" 
            class:permanent-track={ray.isPermanent}
            style="animation-delay: {ray.staggerIndex * 90}ms;"
          />

          <!-- Traveling Photon Pulse (Energy Particle Streaming from Source to Target) -->
          <path
            d={ray.coords.path}
            pathLength="100"
            class="beam-pulse" 
            class:permanent-pulse={ray.isPermanent}
            style="animation-delay: {ray.staggerIndex * 90}ms;"
          />

          <!-- Target Node Aperture Bloom (Erupts upon photon arrival) -->
          <g class="aperture-group" transform="translate({ray.coords.x2}, {ray.coords.y2})">
            <circle 
              cx="0" cy="0" r="22" 
              fill={ray.origin === 'atproto' ? "url(#cosmic-iceblue-bloom)" : "url(#cosmic-amber-bloom)"} 
              class="glow-ring target-bloom" 
              class:permanent-bloom={ray.isPermanent} 
              style="animation-delay: {ray.staggerIndex * 90}ms;"
            />
            <circle 
              cx="0" cy="0" r="3.2" 
              fill={ray.origin === 'atproto' ? "#38bdf8" : "#f59e0b"} 
              class="core-dot target-bloom" 
              class:permanent-bloom={ray.isPermanent} 
              style="animation-delay: {ray.staggerIndex * 90}ms;"
            />
          </g>
        </g>
      {/each}
    </svg>
  {/if}
</div>

<style>
  .cosmic-ray-network {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
    z-index: -1;
    transform: translateZ(-5px);
    transform-style: preserve-3d;
  }

  .cosmic-flat-svg {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    overflow: visible;
    pointer-events: none;
  }

  .cosmic-flat-html {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    pointer-events: none;
    transform-style: preserve-3d;
  }

  .ray-instance {
    pointer-events: none;
  }

  .aperture-group {
    pointer-events: none;
  }

  /* Amber Aperture Rings & Dots (Bugu) */
  .glow-ring {
    transform-origin: 0 0;
    filter: drop-shadow(0 0 6px rgba(245, 158, 11, 0.95)) drop-shadow(0 0 16px rgba(217, 119, 6, 0.6));
  }

  .core-dot {
    transform-origin: 0 0;
    filter: drop-shadow(0 0 3px rgba(245, 158, 11, 1));
  }

  /* Ice Blue Aperture Rings & Dots (Bluesky & Synapse) */
  .is-bsky .glow-ring,
  .is-synapse .glow-ring {
    filter: drop-shadow(0 0 6px rgba(56, 189, 248, 0.95)) drop-shadow(0 0 16px rgba(14, 165, 233, 0.6));
  }

  .is-bsky .core-dot,
  .is-synapse .core-dot {
    filter: drop-shadow(0 0 3px #38bdf8);
  }

  /* Beam Track: Delicate Ethereal Fiber */
  .beam-track {
    fill: none;
    stroke: #f59e0b;
    stroke-width: 0.9px;
    stroke-linecap: round;
    filter: drop-shadow(0 0 2px rgba(245, 158, 11, 0.35));
    animation: cosmic-track-decay 4.8s ease-in-out forwards;
  }

  .is-bsky .beam-track,
  .is-synapse .beam-track {
    stroke: #38bdf8;
    filter: drop-shadow(0 0 2px rgba(56, 189, 248, 0.35));
  }

  /* Traveling Photon Pulse */
  .beam-pulse {
    fill: none;
    stroke: #f59e0b;
    stroke-width: 1.6px;
    stroke-linecap: round;
    stroke-dasharray: 24 100;
    filter: drop-shadow(0 0 4px rgba(245, 158, 11, 0.95)) drop-shadow(0 0 10px rgba(217, 119, 6, 0.5));
    animation: cosmic-beam-pulse 4.8s ease-in-out forwards;
  }

  .is-bsky .beam-pulse,
  .is-synapse .beam-pulse {
    stroke: #7dd3fc;
    filter: drop-shadow(0 0 4px rgba(56, 189, 248, 0.95)) drop-shadow(0 0 10px rgba(14, 165, 233, 0.5));
  }

  /* Permanent Hovered States */
  .permanent-track {
    animation: none !important;
    opacity: 0.8 !important;
  }

  .permanent-pulse {
    animation: cosmic-beam-pulse-loop 2.4s ease-in-out infinite !important;
    opacity: 0.95 !important;
  }

  .permanent-bloom {
    animation: none !important;
    opacity: 0.9 !important;
    transform: scale(1) !important;
  }

  .source-bloom {
    animation: cosmic-source-bloom 4.8s ease-in-out forwards;
  }

  .target-bloom {
    animation: cosmic-target-bloom 4.8s ease-in-out forwards;
  }

  /* Keyframe Animations */
  @keyframes cosmic-track-decay {
    0% {
      opacity: 0;
    }
    10% {
      opacity: 0.85;
    }
    60% {
      opacity: 0.7;
    }
    100% {
      opacity: 0;
    }
  }

  @keyframes cosmic-beam-pulse {
    0% {
      stroke-dashoffset: 100;
      opacity: 0;
    }
    10% {
      opacity: 0.95;
    }
    80% {
      stroke-dashoffset: -24;
      opacity: 0.95;
    }
    100% {
      stroke-dashoffset: -24;
      opacity: 0;
    }
  }

  @keyframes cosmic-beam-pulse-loop {
    0% {
      stroke-dashoffset: 100;
      opacity: 0;
    }
    15% {
      opacity: 0.95;
    }
    85% {
      stroke-dashoffset: -24;
      opacity: 0.95;
    }
    100% {
      stroke-dashoffset: -24;
      opacity: 0;
    }
  }

  @keyframes cosmic-source-bloom {
    0% {
      opacity: 0;
      transform: scale(0.35);
    }
    8% {
      opacity: 0.95;
      transform: scale(1.25);
    }
    20% {
      opacity: 0.6;
      transform: scale(0.9);
    }
    45% {
      opacity: 0;
      transform: scale(0.4);
    }
    100% {
      opacity: 0;
      transform: scale(0.4);
    }
  }

  @keyframes cosmic-target-bloom {
    0%, 35% {
      opacity: 0;
      transform: scale(0.35);
    }
    60% {
      opacity: 0.95;
      transform: scale(1.3);
    }
    80% {
      opacity: 0.6;
      transform: scale(0.9);
    }
    100% {
      opacity: 0;
      transform: scale(0.4);
    }
  }

  /* Anchor Badges */
  .ray-anchor-badge-container {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none;
    z-index: 10;
    animation: cosmic-badge-fade 4.8s ease-in-out forwards;
  }

  .ray-anchor-badge {
    display: inline-flex;
    align-items: center;
    padding: 3px 9px;
    border-radius: 9999px;
    font-size: 11px;
    font-weight: 500;
    letter-spacing: 0.03em;
    background: rgba(15, 23, 42, 0.85);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    color: #f59e0b;
    border: 1px solid rgba(245, 158, 11, 0.4);
    box-shadow: 0 0 12px rgba(245, 158, 11, 0.25), 0 2px 6px rgba(0, 0, 0, 0.4);
    white-space: nowrap;
    user-select: none;
    pointer-events: auto;
    transition: all 0.25s ease;
  }

  .ray-anchor-badge.is-bsky,
  .ray-anchor-badge.is-synapse {
    color: #38bdf8;
    border-color: rgba(56, 189, 248, 0.4);
    box-shadow: 0 0 12px rgba(56, 189, 248, 0.25), 0 2px 6px rgba(0, 0, 0, 0.4);
  }

  .permanent-badge {
    animation: none !important;
    opacity: 1 !important;
  }

  @keyframes cosmic-badge-fade {
    0% {
      opacity: 0;
      transform: scale(0.85);
    }
    15% {
      opacity: 0.95;
      transform: scale(1);
    }
    75% {
      opacity: 0.9;
    }
    100% {
      opacity: 0;
      transform: scale(0.9);
    }
  }
</style>
