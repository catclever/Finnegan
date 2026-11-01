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
  import {
    buildMotherChildRayDefs,
    buildSynapseRayDefs,
    assembleRenderedRays,
    type ActiveRayInstance,
  } from '../../lib/cosmicRayEngine';

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

  // 1. Mother-Child Rays (Bugu & Bluesky)
  $: motherChildRayDefs = buildMotherChildRayDefs(cards);

  // 2. Synapse Rays (Hovered Tag Association Network)
  $: synapseRayDefs = buildSynapseRayDefs(cards, hoveredCardId, focusedCardId);

  // Periodic ambient pulse scheduler for Mother-Child rays: 10s ~ 30s random single-pulse interval
  const MIN_AMBIENT_INTERVAL_MS = 10000;
  const MAX_AMBIENT_INTERVAL_MS = 30000;

  let activeInstance: ActiveRayInstance | null = null;
  let nextInstanceId = 0;
  let ambientSpawnTimer: any = null;
  let lastSpawnedId = '';

  function getRandomAmbientDelay(): number {
    return MIN_AMBIENT_INTERVAL_MS + Math.floor(Math.random() * (MAX_AMBIENT_INTERVAL_MS - MIN_AMBIENT_INTERVAL_MS));
  }

  function spawnNextAmbientRay() {
    if (motherChildRayDefs.length === 0) {
      ambientSpawnTimer = null;
      activeInstance = null;
      return;
    }

    const candidates = motherChildRayDefs.filter(r => r.id !== lastSpawnedId);
    const selected = candidates.length > 0 
      ? candidates[Math.floor(Math.random() * candidates.length)]
      : motherChildRayDefs[0];

    const id = ++nextInstanceId;
    lastSpawnedId = selected.id;

    // Single-pulse strictly enforced: only 1 ambient ray active at a time
    activeInstance = {
      instanceId: id,
      rayId: selected.id,
      sourceId: selected.source.id,
      targetId: selected.target.id,
    };

    // Schedule next pulse event at random 10s ~ 30s interval
    const nextDelay = getRandomAmbientDelay();
    ambientSpawnTimer = setTimeout(() => {
      ambientSpawnTimer = null;
      spawnNextAmbientRay();
    }, nextDelay);
  }

  function handleRayFinished(instanceId: number) {
    if (activeInstance && activeInstance.instanceId === instanceId) {
      activeInstance = null;
    }
  }

  $: {
    if (motherChildRayDefs.length > 0) {
      if (!ambientSpawnTimer && activeInstance === null) {
        const initialDelay = getRandomAmbientDelay();
        ambientSpawnTimer = setTimeout(() => {
          ambientSpawnTimer = null;
          spawnNextAmbientRay();
        }, initialDelay);
      }
    } else {
      if (ambientSpawnTimer) {
        clearTimeout(ambientSpawnTimer);
        ambientSpawnTimer = null;
      }
      activeInstance = null;
    }
  }

  onDestroy(() => {
    if (ambientSpawnTimer) clearTimeout(ambientSpawnTimer);
  });

  // Consolidated Rendered Rays with Staggered Cascading
  $: renderedRays = assembleRenderedRays({
    cards,
    focusedCardId,
    hoveredCardId,
    motherChildRayDefs,
    synapseRayDefs,
    activeInstance,
  });
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
            style="transform: translate(calc({ray.coords.midX}px - 50%), calc({ray.coords.midY}px - 50%)) translateZ(8px);"
          >
            <div 
              class="ray-anchor-badge"
              class:is-bsky={ray.origin === 'atproto'}
              class:is-synapse={ray.type === 'synapse'}
              class:permanent-badge={ray.isPermanent}
              style="animation-delay: {ray.staggerIndex * 90}ms;"
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
            on:animationend={() => {
              if (!ray.isPermanent && ray.instanceId) {
                handleRayFinished(ray.instanceId);
              }
            }}
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

  /* Amber Aperture Rings & Dots (Bugu) - Hardware RadialGradient, Zero CPU Drop-Shadow */
  .glow-ring {
    transform-origin: 0 0;
  }

  .core-dot {
    transform-origin: 0 0;
  }

  /* Ice Blue Aperture Rings & Dots (Bluesky & Synapse) */
  .is-bsky .glow-ring,
  .is-synapse .glow-ring {
    transform-origin: 0 0;
  }

  .is-bsky .core-dot,
  .is-synapse .core-dot {
    transform-origin: 0 0;
  }

  /* Beam Track: Delicate Ethereal Fiber */
  .beam-track {
    fill: none;
    stroke: rgba(245, 158, 11, 0.45);
    stroke-width: 1px;
    stroke-linecap: round;
    animation: cosmic-track-decay 14.4s ease-in-out forwards;
  }

  .is-bsky .beam-track,
  .is-synapse .beam-track {
    stroke: rgba(56, 189, 248, 0.45);
  }

  /* Traveling Photon Pulse: Brilliant GPU Laser Filament */
  .beam-pulse {
    fill: none;
    stroke: #fbbf24;
    stroke-width: 1.8px;
    stroke-linecap: round;
    stroke-dasharray: 24 100;
    animation: cosmic-beam-pulse 14.4s ease-in-out forwards;
  }

  .is-bsky .beam-pulse,
  .is-synapse .beam-pulse {
    stroke: #7dd3fc;
  }

  /* Permanent Hovered States */
  .permanent-track {
    animation: none !important;
    opacity: 0.35 !important;
  }

  .permanent-pulse {
    animation: cosmic-beam-pulse-continuous 1.8s linear infinite !important;
    opacity: 0.95 !important;
  }

  .permanent-bloom {
    animation: none !important;
    opacity: 0.92 !important;
    transform: scale(1.25) !important;
  }

  .source-bloom {
    animation: cosmic-source-bloom 14.4s ease-in-out forwards;
  }

  .target-bloom {
    animation: cosmic-target-bloom 14.4s ease-in-out forwards;
  }

  /* Keyframe Animations (Synchronized 14.4s 3-Cycle Physical Clock) */
  @keyframes cosmic-track-decay {
    /* Cycle 1 (0 ~ 4.8s): Base Track Opacity 0.22 */
    0% { opacity: 0; }
    3% { opacity: 0.22; }
    30% { opacity: 0.22; }
    33.3% { opacity: 0.22; }

    /* Cycle 2 (4.8s ~ 9.6s): Base Track Opacity 0.12 */
    33.4% { opacity: 0.12; }
    63% { opacity: 0.12; }
    66.6% { opacity: 0.12; }

    /* Cycle 3 (9.6s ~ 14.4s): Base Track Opacity 0.04 */
    66.7% { opacity: 0.04; }
    96% { opacity: 0.04; }
    100% { opacity: 0; }
  }

  @keyframes cosmic-beam-pulse {
    /* Cycle 1 (0 ~ 4.8s): Pulse Peak Opacity 0.92 */
    0%, 4% {
      stroke-dashoffset: 24;
      opacity: 0;
    }
    5% {
      stroke-dashoffset: 24;
      opacity: 0.92;
    }
    7% {
      stroke-dashoffset: 10;
      opacity: 0.92;
    }
    22% {
      stroke-dashoffset: -85;
      opacity: 0.92;
    }
    25% {
      stroke-dashoffset: -100;
      opacity: 0.92;
    }
    26.5% {
      stroke-dashoffset: -105;
      opacity: 0;
    }
    33.3% {
      stroke-dashoffset: -105;
      opacity: 0;
    }

    /* Cycle 2 (4.8s ~ 9.6s): Pulse Peak Opacity 0.58 */
    33.4%, 37.3% {
      stroke-dashoffset: 24;
      opacity: 0;
    }
    38.3% {
      stroke-dashoffset: 24;
      opacity: 0.58;
    }
    40.3% {
      stroke-dashoffset: 10;
      opacity: 0.58;
    }
    55.3% {
      stroke-dashoffset: -85;
      opacity: 0.58;
    }
    58.3% {
      stroke-dashoffset: -100;
      opacity: 0.58;
    }
    59.8% {
      stroke-dashoffset: -105;
      opacity: 0;
    }
    66.6% {
      stroke-dashoffset: -105;
      opacity: 0;
    }

    /* Cycle 3 (9.6s ~ 14.4s): Pulse Peak Opacity 0.28 */
    66.7%, 70.7% {
      stroke-dashoffset: 24;
      opacity: 0;
    }
    71.7% {
      stroke-dashoffset: 24;
      opacity: 0.28;
    }
    73.7% {
      stroke-dashoffset: 10;
      opacity: 0.28;
    }
    88.7% {
      stroke-dashoffset: -85;
      opacity: 0.28;
    }
    91.7% {
      stroke-dashoffset: -100;
      opacity: 0.28;
    }
    93.2% {
      stroke-dashoffset: -105;
      opacity: 0;
    }
    100% {
      stroke-dashoffset: -105;
      opacity: 0;
    }
  }

  /* Continuous seamless linear loop when permanently hovered */
  @keyframes cosmic-beam-pulse-continuous {
    0% {
      stroke-dashoffset: 24;
      opacity: 0.95;
    }
    100% {
      stroke-dashoffset: -100;
      opacity: 0.95;
    }
  }

  @keyframes cosmic-source-bloom {
    /* Cycle 1 (0 ~ 4.8s): Charges up, PEAKS at launch (4% ~ 7%), fades as pulse moves away */
    0% {
      opacity: 0;
      transform: scale(0.35);
    }
    4% {
      opacity: 0.92;
      transform: scale(1.25);
    }
    7% {
      opacity: 0.92;
      transform: scale(1.15);
    }
    11% {
      opacity: 0.65;
      transform: scale(0.95);
    }
    15% {
      opacity: 0.30;
      transform: scale(0.70);
    }
    18% {
      opacity: 0;
      transform: scale(0.40);
    }
    33.3% {
      opacity: 0;
      transform: scale(0.40);
    }

    /* Cycle 2 (4.8s ~ 9.6s) */
    33.4% {
      opacity: 0;
      transform: scale(0.35);
    }
    37.3% {
      opacity: 0.58;
      transform: scale(1.25);
    }
    40.3% {
      opacity: 0.58;
      transform: scale(1.15);
    }
    44.3% {
      opacity: 0.40;
      transform: scale(0.95);
    }
    48.3% {
      opacity: 0.18;
      transform: scale(0.70);
    }
    51.3% {
      opacity: 0;
      transform: scale(0.40);
    }
    66.6% {
      opacity: 0;
      transform: scale(0.40);
    }

    /* Cycle 3 (9.6s ~ 14.4s) */
    66.7% {
      opacity: 0;
      transform: scale(0.35);
    }
    70.7% {
      opacity: 0.28;
      transform: scale(1.25);
    }
    73.7% {
      opacity: 0.28;
      transform: scale(1.15);
    }
    77.7% {
      opacity: 0.18;
      transform: scale(0.95);
    }
    81.7% {
      opacity: 0.08;
      transform: scale(0.70);
    }
    84.7% {
      opacity: 0;
      transform: scale(0.40);
    }
    100% {
      opacity: 0;
      transform: scale(0.40);
    }
  }

  @keyframes cosmic-target-bloom {
    /* Cycle 1 (0 ~ 4.8s): Warms up as pulse approaches, PEAKS upon arrival (25%), fades till 33.3% */
    0%, 15% {
      opacity: 0;
      transform: scale(0.40);
    }
    18% {
      opacity: 0.20;
      transform: scale(0.70);
    }
    22% {
      opacity: 0.65;
      transform: scale(0.95);
    }
    25% {
      opacity: 0.92;
      transform: scale(1.25);
    }
    27.5% {
      opacity: 0.68;
      transform: scale(1.10);
    }
    31% {
      opacity: 0.18;
      transform: scale(0.75);
    }
    33.3% {
      opacity: 0;
      transform: scale(0.45);
    }

    /* Cycle 2 (4.8s ~ 9.6s): Warms up, PEAKS upon arrival (58.3%), fades till 66.6% */
    33.4%, 48.3% {
      opacity: 0;
      transform: scale(0.40);
    }
    51.3% {
      opacity: 0.12;
      transform: scale(0.70);
    }
    55.3% {
      opacity: 0.40;
      transform: scale(0.95);
    }
    58.3% {
      opacity: 0.58;
      transform: scale(1.25);
    }
    60.8% {
      opacity: 0.42;
      transform: scale(1.10);
    }
    64.3% {
      opacity: 0.12;
      transform: scale(0.75);
    }
    66.6% {
      opacity: 0;
      transform: scale(0.45);
    }

    /* Cycle 3 (9.6s ~ 14.4s): Warms up, PEAKS upon arrival (91.7%), fades till 100% */
    66.7%, 81.7% {
      opacity: 0;
      transform: scale(0.40);
    }
    84.7% {
      opacity: 0.06;
      transform: scale(0.70);
    }
    88.7% {
      opacity: 0.18;
      transform: scale(0.95);
    }
    91.7% {
      opacity: 0.28;
      transform: scale(1.25);
    }
    94.2% {
      opacity: 0.18;
      transform: scale(1.10);
    }
    97.7% {
      opacity: 0.05;
      transform: scale(0.75);
    }
    100% {
      opacity: 0;
      transform: scale(0.45);
    }
  }

  /* Anchor Badges */
  .ray-anchor-badge-container {
    position: absolute;
    top: 0;
    left: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: none;
    z-index: 10;
  }

  .ray-anchor-badge {
    position: relative;
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
    transform-origin: center center;
    user-select: none;
    pointer-events: auto;
    cursor: default;
    transition: transform 0.25s ease, opacity 0.25s ease, box-shadow 0.25s ease;
    animation: cosmic-badge-bloom 14.4s ease-in-out forwards;
  }

  .ray-anchor-badge.is-bsky,
  .ray-anchor-badge.is-synapse {
    color: #38bdf8;
    border-color: rgba(56, 189, 248, 0.4);
    box-shadow: 0 0 12px rgba(56, 189, 248, 0.25), 0 2px 6px rgba(0, 0, 0, 0.4);
  }

  .ray-anchor-badge:hover {
    opacity: 1 !important;
    transform: scale(1.06) !important;
  }

  .ray-anchor-badge:not(.is-bsky):not(.is-synapse):hover {
    box-shadow: 0 0 16px rgba(245, 158, 11, 0.5), 0 2px 8px rgba(0, 0, 0, 0.5);
  }

  .ray-anchor-badge.is-bsky:hover,
  .ray-anchor-badge.is-synapse:hover {
    box-shadow: 0 0 16px rgba(56, 189, 248, 0.5), 0 2px 8px rgba(0, 0, 0, 0.5);
  }

  .permanent-badge {
    animation: none !important;
    opacity: 1 !important;
  }

  @keyframes cosmic-badge-bloom {
    /* Cycle 1 (0 ~ 4.8s): Ambient whisper (0.10), blooms to 1.0 when photon pulse passes through middle at 14%-17%, returns to ambient (0.10) */
    0% {
      opacity: 0;
      transform: scale(0.9);
    }
    3% {
      opacity: 0.10;
      transform: scale(1);
    }
    11% {
      opacity: 0.10;
      transform: scale(1);
    }
    13% {
      opacity: 0.75;
      transform: scale(1.03);
    }
    14% {
      opacity: 1;
      transform: scale(1.06);
    }
    17% {
      opacity: 1;
      transform: scale(1.06);
    }
    19% {
      opacity: 0.65;
      transform: scale(1.02);
    }
    21% {
      opacity: 0.10;
      transform: scale(1);
    }
    33.3% {
      opacity: 0.10;
      transform: scale(1);
    }

    /* Cycle 2 (4.8s ~ 9.6s): Second pulse wave */
    33.4% {
      opacity: 0.10;
      transform: scale(1);
    }
    44% {
      opacity: 0.10;
      transform: scale(1);
    }
    46% {
      opacity: 0.70;
      transform: scale(1.02);
    }
    47% {
      opacity: 1;
      transform: scale(1.05);
    }
    50% {
      opacity: 1;
      transform: scale(1.05);
    }
    52% {
      opacity: 0.60;
      transform: scale(1.01);
    }
    54% {
      opacity: 0.08;
      transform: scale(1);
    }
    66.6% {
      opacity: 0.08;
      transform: scale(1);
    }

    /* Cycle 3 (9.6s ~ 14.4s): Third pulse wave & final fade */
    66.7% {
      opacity: 0.08;
      transform: scale(1);
    }
    78% {
      opacity: 0.08;
      transform: scale(1);
    }
    80% {
      opacity: 1;
      transform: scale(1.03);
    }
    83% {
      opacity: 1;
      transform: scale(1.03);
    }
    85% {
      opacity: 0.45;
      transform: scale(1.01);
    }
    87% {
      opacity: 0.08;
      transform: scale(1);
    }
    94% {
      opacity: 0.03;
      transform: scale(0.96);
    }
    100% {
      opacity: 0;
      transform: scale(0.9);
    }
  }
</style>
