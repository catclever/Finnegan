<!--
  ======================================================================================
  BuguRayNetwork.svelte - Bugu Directed Decaying Cross-Screen Rays & Radar Beacon
  Task: bugu_ray_exact_center_and_pure_amber_decay_030
  ======================================================================================
-->
<script lang="ts">
  import { onDestroy } from 'svelte';
  import type { CosmicCard } from '../../types/card';
  import type { BuguBeacon } from './types';
  import BuguRadarBeacon from './BuguRadarBeacon.svelte';

  export let cards: CosmicCard[] = [];
  export let beacon: BuguBeacon | null = null;
  export let scrollExpansion: number = 0;
  export let hoveredMotherId: string | null = null;
  // Legacy session prop fallback
  export let session: any = null;
  $: void scrollExpansion;

  $: activeBeacon = beacon || (session ? { sourceCardId: session.sourceCardId, status: session.status } : null);

  $: beaconCard = activeBeacon 
    ? cards.find(c => c.id === activeBeacon.sourceCardId && !c.isFadingOut) 
    : null;

  // Requirement 1 & 6: Cross-screen connectivity from mother center to child center
  $: availableRays = (() => {
    const list: Array<{
      id: string;
      source: CosmicCard;
      target: CosmicCard;
    }> = [];

    for (const child of cards) {
      if (child.origin === 'bugu' && child.echoOf && !child.isFadingOut) {
        const mother = cards.find(
          c => c.id === child.echoOf && !c.isFadingOut
        );
        if (mother) {
          list.push({
            id: `${mother.id}->${child.id}`,
            source: mother,
            target: child
          });
        }
      }
    }
    return list;
  })();

  // Requirement 3: Synchronized 3-cycle decay (4.8s per cycle, 14.4s total - 2x duration)
  const SINGLE_PULSE_MS = 4800;
  const MIN_NEXT_DELAY_MS = SINGLE_PULSE_MS; // Lower bound is duration of 1 pulse
  const RANDOM_ADDITIONAL_MS = 2400;         // Range [4800ms, 7200ms]

  interface ActiveRayInstance {
    instanceId: number;
    rayId: string;
    sourceId: string;
    targetId: string;
  }

  let activeInstances: ActiveRayInstance[] = [];
  let nextInstanceId = 0;
  let nextSpawnTimer: any = null;
  let lastSpawnedRayId: string = '';

  // To track hovered items without changing ActiveRayInstance interface which doesn't allow new props
  $: isMotherHovered = (id: string) => hoveredMotherId !== null && hoveredMotherId === id;

  function spawnNextRay() {
    if (availableRays.length === 0) {
      nextSpawnTimer = null;
      return;
    }

    // Pick candidate: prefer a ray different from last spawned, or random
    const candidates = availableRays.filter(r => r.id !== lastSpawnedRayId);
    const selected = candidates.length > 0 
      ? candidates[Math.floor(Math.random() * candidates.length)]
      : availableRays[0];

    const id = ++nextInstanceId;
    lastSpawnedRayId = selected.id;
    activeInstances = [
      ...activeInstances,
      {
        instanceId: id,
        rayId: selected.id,
        sourceId: selected.source.id,
        targetId: selected.target.id
      }
    ];

    // Schedule next ray at a random interval whose lower bound is 1 pulse duration
    const delay = MIN_NEXT_DELAY_MS + Math.floor(Math.random() * RANDOM_ADDITIONAL_MS);
    nextSpawnTimer = setTimeout(() => {
      nextSpawnTimer = null;
      spawnNextRay();
    }, delay);
  }

  // Reactive trigger: ensure scheduler runs whenever availableRays exist
  $: {
    if (availableRays.length > 0) {
      if (!nextSpawnTimer && activeInstances.length === 0) {
        spawnNextRay();
      }
    } else {
      if (nextSpawnTimer) {
        clearTimeout(nextSpawnTimer);
        nextSpawnTimer = null;
      }
      if (activeInstances.length > 0) {
        activeInstances = [];
      }
    }
  }

  function handleRayFinished(instanceId: number) {
    activeInstances = activeInstances.filter(inst => inst.instanceId !== instanceId);
  }

  function getCardScale(card: CosmicCard): number {
    const zNorm = Math.max(0, Math.min(1, (card.z || 0) / 100));
    return 1.08 - (zNorm * 0.44);
  }

  // Refined Docking Inset: 2px close to card edge so aperture circle sits right on border
  const DOCKING_INSET_PX = 2;

  function computeAnimatedBeamPath(
    source: CosmicCard,
    target: CosmicCard
  ): { path: string; x1: number; y1: number; x2: number; y2: number; midX: number; midY: number } {
    const rawDx = target.x - source.x;
    const rawDy = target.y - source.y;
    const dist = Math.hypot(rawDx, rawDy);

    if (dist < 1) {
      return {
        path: `M ${source.x} ${source.y} L ${target.x} ${target.y}`,
        x1: source.x,
        y1: source.y,
        x2: target.x,
        y2: target.y,
        midX: source.x,
        midY: source.y
      };
    }

    const sourceScale = getCardScale(source);
    const targetScale = getCardScale(target);

    const sourceHalfW = Math.max(16, (200 * sourceScale) / 2 - DOCKING_INSET_PX);
    const sourceHalfH = Math.max(16, (190 * sourceScale) / 2 - DOCKING_INSET_PX);

    const targetHalfW = Math.max(16, (200 * targetScale) / 2 - DOCKING_INSET_PX);
    const targetHalfH = Math.max(16, (190 * targetScale) / 2 - DOCKING_INSET_PX);

    let startX: number, startY: number, endX: number, endY: number;
    let c1x: number, c1y: number, c2x: number, c2y: number;

    const isHorizontal = Math.abs(rawDx) >= Math.abs(rawDy);

    if (isHorizontal) {
      // Exit from horizontal side (left or right)
      if (rawDx >= 0) {
        startX = source.x + sourceHalfW;
        endX = target.x - targetHalfW;
      } else {
        startX = source.x - sourceHalfW;
        endX = target.x + targetHalfW;
      }
      // Gentle vertical alignment adjustment
      const maxSagY = Math.min(24, Math.abs(rawDy) * 0.2);
      startY = source.y + (rawDy > 0 ? maxSagY : -maxSagY);
      endY = target.y + (rawDy > 0 ? -maxSagY : maxSagY);

      const dx = endX - startX;
      const dy = endY - startY;
      const curvature = Math.min(180, Math.max(40, Math.abs(dx) * 0.45));

      c1x = startX + (dx >= 0 ? curvature : -curvature);
      c1y = startY;
      c2x = endX - (dx >= 0 ? curvature : -curvature);
      c2y = endY;
    } else {
      // Exit from vertical side (top or bottom, e.g. cross-screen gravitational tide)
      if (rawDy >= 0) {
        startY = source.y + sourceHalfH;
        endY = target.y - targetHalfH;
      } else {
        startY = source.y - sourceHalfH;
        endY = target.y + targetHalfH;
      }
      // Gentle horizontal alignment adjustment
      const maxSagX = Math.min(24, Math.abs(rawDx) * 0.2);
      startX = source.x + (rawDx > 0 ? maxSagX : -maxSagX);
      endX = target.x + (rawDx > 0 ? -maxSagX : maxSagX);

      const dx = endX - startX;
      const dy = endY - startY;
      const curvature = Math.min(180, Math.max(40, Math.abs(dy) * 0.45));

      c1x = startX;
      c1y = startY + (dy >= 0 ? curvature : -curvature);
      c2x = endX;
      c2y = endY - (dy >= 0 ? curvature : -curvature);
    }

    const path = `M ${startX.toFixed(1)} ${startY.toFixed(1)} C ${c1x.toFixed(1)} ${c1y.toFixed(1)}, ${c2x.toFixed(1)} ${c2y.toFixed(1)}, ${endX.toFixed(1)} ${endY.toFixed(1)}`;

    const t = 0.5;
    const mt = 1 - t;
    const midX = mt * mt * mt * startX + 3 * mt * mt * t * c1x + 3 * mt * t * t * c2x + t * t * t * endX;
    const midY = mt * mt * mt * startY + 3 * mt * mt * t * c1y + 3 * mt * t * t * c2y + t * t * t * endY;

    return { path, x1: startX, y1: startY, x2: endX, y2: endY, midX, midY };
  }

  $: renderedRays = activeInstances
    .map(inst => {
      const source = cards.find(c => c.id === inst.sourceId && !c.isFadingOut);
      const target = cards.find(c => c.id === inst.targetId && !c.isFadingOut);
      if (!source || !target) return null;
      const beam = computeAnimatedBeamPath(source, target);
      return {
        instanceId: inst.instanceId,
        id: inst.rayId,
        source,
        target,
        path: beam.path,
        x1: beam.x1,
        y1: beam.y1,
        x2: beam.x2,
        y2: beam.y2,
        midX: beam.midX,
        midY: beam.midY
      };
    })
    .filter(Boolean) as Array<{
      instanceId: number;
      id: string;
      source: CosmicCard;
      target: CosmicCard;
      path: string;
      x1: number;
      y1: number;
      x2: number;
      y2: number;
      midX: number;
      midY: number;
    }>;

  onDestroy(() => {
    if (nextSpawnTimer) clearTimeout(nextSpawnTimer);
  });
</script>

<!-- Requirement 4: Normalized 3D z-depth (-5px) & z-index: -1, seamlessly beneath cards -->
<div class="bugu-ray-network">
  <!-- Mother Card Radar Beacon (Remains active until all Babel cards have appeared) -->
  {#if beaconCard && activeBeacon}
    <BuguRadarBeacon x={beaconCard.x} y={beaconCard.y} status={activeBeacon.status} />
  {/if}

  <!-- Magic UI Animated Beam: Refined Fiber, Bezier Curvature & Synchronized Blooming Aperture Nodes -->
  {#if renderedRays.length > 0}
    <div class="bugu-flat-html">
      {#each renderedRays as ray (ray.instanceId)}
        {#if ray.target.matchedAnchor}
          <div 
             class="ray-anchor-badge-container"
             style="transform: translate(calc({ray.midX}px - 50%), calc({ray.midY}px - 50%)) translateZ(8px);"
          >
            <div 
              class="ray-anchor-badge"
              class:permanent-badge={isMotherHovered(ray.source.id)}
              title={ray.target.echoReason || ray.target.matchedAnchor}
            >
              {ray.target.matchedAnchor}
            </div>
          </div>
        {/if}
      {/each}
    </div>

    <svg class="bugu-flat-svg">
      <defs>
        <!-- Soft glowing radial gradient for aperture nodes -->
        <radialGradient id="bugu-aperture-glow" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#f59e0b" stop-opacity="1" />
          <stop offset="35%" stop-color="#f59e0b" stop-opacity="0.65" />
          <stop offset="70%" stop-color="#f59e0b" stop-opacity="0.2" />
          <stop offset="100%" stop-color="#f59e0b" stop-opacity="0" />
        </radialGradient>
      </defs>

      {#each renderedRays as ray (ray.instanceId)}
        <g class="ray-instance">
          <!-- Source Node Aperture Bloom (Erupts before launch, dissolves as beam flies) -->
          <g class="aperture-group" transform="translate({ray.x1}, {ray.y1})">
            <circle cx="0" cy="0" r="22" fill="url(#bugu-aperture-glow)" class="glow-ring source-bloom" class:permanent-bloom={isMotherHovered(ray.source.id)} />
            <circle cx="0" cy="0" r="3.2" fill="#f59e0b" class="core-dot source-bloom" class:permanent-bloom={isMotherHovered(ray.source.id)} />
          </g>

          <!-- Magic UI Style: Static ambient beam path in pure Amber Gold (#f59e0b) -->
          <path
            d={ray.path}
            class="beam-track" class:permanent-track={isMotherHovered(ray.source.id)}
          />

          <!-- Magic UI Style: Traveling energy pulse in pure Amber Gold (#f59e0b) flowing Mother -> Child -->
          <path
            d={ray.path}
            pathLength="100"
            class="beam-pulse" class:permanent-pulse={isMotherHovered(ray.source.id)}
            on:animationend={() => { if (!isMotherHovered(ray.source.id)) handleRayFinished(ray.instanceId); }}
          />

          <!-- Target Node Aperture Bloom (Warms up as beam approaches, peaks upon arrival, fades out) -->
          <g class="aperture-group" transform="translate({ray.x2}, {ray.y2})">
            <circle cx="0" cy="0" r="22" fill="url(#bugu-aperture-glow)" class="glow-ring target-bloom" class:permanent-bloom={isMotherHovered(ray.source.id)} />
            <circle cx="0" cy="0" r="3.2" fill="#f59e0b" class="core-dot target-bloom" class:permanent-bloom={isMotherHovered(ray.source.id)} />
          </g>
        </g>
      {/each}
    </svg>
  {/if}
</div>

<style>
  /* Requirement 4: Normalized 3D z-depth (-5px) & z-index: -1, eliminating parallax while staying beneath cards */
  .bugu-ray-network {
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

  /* Origin anchored at top: 50%; left: 50% with width: 0; height: 0.
     No viewBox distortion: exact pixel tracking across 3D Cosmic Stage */
  .bugu-flat-svg {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    overflow: visible;
    pointer-events: none;
  }

  .ray-instance {
    pointer-events: none;
  }

  /* Aperture Node Styles */
  .aperture-group {
    pointer-events: none;
  }

  .glow-ring {
    transform-origin: 0 0;
    filter: drop-shadow(0 0 6px rgba(245, 158, 11, 0.95)) drop-shadow(0 0 16px rgba(217, 119, 6, 0.6));
  }

  .core-dot {
    transform-origin: 0 0;
    filter: drop-shadow(0 0 3px rgba(245, 158, 11, 1));
  }

  .source-bloom {
    animation: bugu-source-bloom 14.4s ease-in-out forwards;
  }

  .target-bloom {
    animation: bugu-target-bloom 14.4s ease-in-out forwards;
  }

  /* Refined Magic UI Track: Thinner (0.9px), subtle and elegant */
  .beam-track {
    fill: none;
    stroke: #f59e0b;
    stroke-width: 0.9px;
    stroke-linecap: round;
    filter: drop-shadow(0 0 2px rgba(245, 158, 11, 0.35));
    animation: bugu-track-decay 14.4s ease-in-out forwards;
  }

  /* Refined Magic UI Pulse: Delicate (1.6px), crisp high-precision laser filament */
  .beam-pulse {
    fill: none;
    stroke: #f59e0b;
    stroke-width: 1.6px;
    stroke-linecap: round;
    stroke-dasharray: 24 100;
    filter: drop-shadow(0 0 4px rgba(245, 158, 11, 0.95)) drop-shadow(0 0 10px rgba(217, 119, 6, 0.5));
    animation: bugu-beam-pulse 14.4s ease-in-out forwards;
  }

  @keyframes bugu-source-bloom {
    /* Cycle 1 (0 ~ 4.8s): Charges up, PEAKS at launch (4% ~ 7%), fades as pulse moves away (7% ~ 18%) */
    0% {
      opacity: 0;
      transform: scale(0.35);
    }
    4% {
      /* Pulse Launch Moment: Maximum Radiance */
      opacity: 0.92;
      transform: scale(1.25);
    }
    7% {
      /* Pulse begins traveling into deep space */
      opacity: 0.92;
      transform: scale(1.15);
    }
    11% {
      /* Pulse is moving further away */
      opacity: 0.65;
      transform: scale(0.95);
    }
    15% {
      opacity: 0.30;
      transform: scale(0.70);
    }
    18% {
      /* Pulse is midway, source fully dissolves */
      opacity: 0;
      transform: scale(0.40);
    }
    33.3% {
      opacity: 0;
      transform: scale(0.40);
    }

    /* Cycle 2 (4.8s ~ 9.6s): Charges up, PEAKS at launch (37.3% ~ 40.3%), fades as pulse moves away */
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

    /* Cycle 3 (9.6s ~ 14.4s): Charges up, PEAKS at launch (70.7% ~ 73.7%), fades as pulse moves away */
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

  @keyframes bugu-target-bloom {
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
      /* Arrival Peak Moment */
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
      /* Arrival Peak Moment */
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
      /* Arrival Peak Moment */
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

  @keyframes bugu-track-decay {
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

  @keyframes bugu-beam-pulse {
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

  .bugu-flat-html {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    overflow: visible;
    pointer-events: none;
    z-index: 10;
  }

  .ray-anchor-badge-container {
    position: absolute;
    pointer-events: none;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .ray-anchor-badge {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: #fef3c7;
    background: transparent;
    border: 1px solid transparent;
    backdrop-filter: blur(0px);
    -webkit-backdrop-filter: blur(0px);
    padding: 1.5px 8px;
    border-radius: 6px;
    box-shadow: none;
    text-shadow: 0 0 3px rgba(245, 158, 11, 0.25);
    white-space: nowrap;
    transform-origin: center center;
    pointer-events: auto;
    cursor: default;
    user-select: none;
    transition: background 0.25s ease, border-color 0.25s ease, backdrop-filter 0.25s ease, -webkit-backdrop-filter 0.25s ease, box-shadow 0.25s ease, transform 0.25s ease, opacity 0.25s ease, text-shadow 0.25s ease;
    animation: bugu-badge-decay 14.4s ease-in-out forwards;
  }

  .ray-anchor-badge:hover {
    opacity: 1 !important;
    background: rgba(15, 23, 42, 0.88) !important;
    border-color: rgba(245, 158, 11, 0.75) !important;
    backdrop-filter: blur(8px) !important;
    -webkit-backdrop-filter: blur(8px) !important;
    transform: scale(1.06) !important;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.6), 0 0 16px rgba(245, 158, 11, 0.45) !important;
    text-shadow: 0 0 8px rgba(245, 158, 11, 0.85), 0 0 14px rgba(251, 191, 36, 0.5) !important;
  }

  @keyframes bugu-badge-decay {
    /* Cycle 1 (0 ~ 4.8s): Ambient whisper (0.10), frosted glass & full radiance blooms when pulse passes (13%~18%), dissolves back to subtle ambient (0.09) */
    0% {
      opacity: 0;
      transform: scale(0.9);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    3% {
      opacity: 0.10;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    11% {
      opacity: 0.10;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    13% {
      /* Pulse approaches: frosted capsule begins blooming, text wakes up */
      opacity: 0.75;
      transform: scale(1.03);
      background: rgba(15, 23, 42, 0.5);
      border-color: rgba(245, 158, 11, 0.4);
      backdrop-filter: blur(4px);
      -webkit-backdrop-filter: blur(4px);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4), 0 0 8px rgba(245, 158, 11, 0.25);
    }
    14% {
      /* Pulse packet rushes across badge: full frosted glass capsule manifestation and 100% radiance */
      opacity: 1;
      transform: scale(1.06);
      background: rgba(15, 23, 42, 0.88);
      border-color: rgba(245, 158, 11, 0.75);
      backdrop-filter: blur(8px);
      -webkit-backdrop-filter: blur(8px);
      box-shadow: 0 2px 12px rgba(0, 0, 0, 0.6), 0 0 16px rgba(245, 158, 11, 0.4);
      text-shadow: 0 0 8px rgba(245, 158, 11, 0.8), 0 0 14px rgba(251, 191, 36, 0.5);
    }
    17% {
      opacity: 1;
      transform: scale(1.06);
      background: rgba(15, 23, 42, 0.88);
      border-color: rgba(245, 158, 11, 0.75);
      backdrop-filter: blur(8px);
      -webkit-backdrop-filter: blur(8px);
      box-shadow: 0 2px 12px rgba(0, 0, 0, 0.6), 0 0 16px rgba(245, 158, 11, 0.4);
      text-shadow: 0 0 8px rgba(245, 158, 11, 0.8), 0 0 14px rgba(251, 191, 36, 0.5);
    }
    19% {
      /* Pulse leaves: frosted capsule begins dissolving, text softens */
      opacity: 0.65;
      transform: scale(1.02);
      background: rgba(15, 23, 42, 0.4);
      border-color: rgba(245, 158, 11, 0.25);
      backdrop-filter: blur(3px);
      -webkit-backdrop-filter: blur(3px);
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.2);
    }
    21% {
      /* Restores to subtle transparent ambient presence */
      opacity: 0.09;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
      text-shadow: 0 0 3px rgba(245, 158, 11, 0.2);
    }
    33.3% {
      opacity: 0.09;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }

    /* Cycle 2 (4.8s ~ 9.6s): Second pulse wave */
    33.4% {
      opacity: 0.09;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    44% {
      opacity: 0.09;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    46% {
      /* Pulse 2 approaches */
      opacity: 0.70;
      transform: scale(1.02);
      background: rgba(15, 23, 42, 0.45);
      border-color: rgba(245, 158, 11, 0.35);
      backdrop-filter: blur(4px);
      -webkit-backdrop-filter: blur(4px);
    }
    47% {
      /* Pulse 2 crosses midpoint: frosted capsule blooms */
      opacity: 1;
      transform: scale(1.05);
      background: rgba(15, 23, 42, 0.85);
      border-color: rgba(245, 158, 11, 0.7);
      backdrop-filter: blur(8px);
      -webkit-backdrop-filter: blur(8px);
      box-shadow: 0 2px 12px rgba(0, 0, 0, 0.6), 0 0 14px rgba(245, 158, 11, 0.4);
      text-shadow: 0 0 8px rgba(245, 158, 11, 0.7), 0 0 14px rgba(251, 191, 36, 0.4);
    }
    50% {
      opacity: 1;
      transform: scale(1.05);
      background: rgba(15, 23, 42, 0.85);
      border-color: rgba(245, 158, 11, 0.7);
      backdrop-filter: blur(8px);
      -webkit-backdrop-filter: blur(8px);
      box-shadow: 0 2px 12px rgba(0, 0, 0, 0.6), 0 0 14px rgba(245, 158, 11, 0.4);
      text-shadow: 0 0 8px rgba(245, 158, 11, 0.7), 0 0 14px rgba(251, 191, 36, 0.4);
    }
    52% {
      /* Pulse 2 dissolves */
      opacity: 0.60;
      transform: scale(1.01);
      background: rgba(15, 23, 42, 0.35);
      border-color: rgba(245, 158, 11, 0.2);
      backdrop-filter: blur(3px);
      -webkit-backdrop-filter: blur(3px);
    }
    54% {
      opacity: 0.08;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    66.6% {
      opacity: 0.08;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }

    /* Cycle 3 (9.6s ~ 14.4s): Final gentle whisper */
    66.7% {
      opacity: 0.08;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    78% {
      opacity: 0.08;
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    80% {
      /* Pulse 3 subtle bloom */
      opacity: 0.90;
      transform: scale(1.03);
      background: rgba(15, 23, 42, 0.82);
      border-color: rgba(245, 158, 11, 0.6);
      backdrop-filter: blur(7px);
      -webkit-backdrop-filter: blur(7px);
      box-shadow: 0 2px 10px rgba(0, 0, 0, 0.5), 0 0 12px rgba(245, 158, 11, 0.3);
      text-shadow: 0 0 7px rgba(245, 158, 11, 0.55);
    }
    83% {
      opacity: 0.90;
      transform: scale(1.03);
      background: rgba(15, 23, 42, 0.82);
      border-color: rgba(245, 158, 11, 0.6);
      backdrop-filter: blur(7px);
      -webkit-backdrop-filter: blur(7px);
      box-shadow: 0 2px 10px rgba(0, 0, 0, 0.5), 0 0 12px rgba(245, 158, 11, 0.3);
      text-shadow: 0 0 7px rgba(245, 158, 11, 0.55);
    }
    85% {
      opacity: 0.45;
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    87% {
      opacity: 0.06;
      transform: scale(1);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    94% {
      opacity: 0.03;
      transform: scale(0.96);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
    100% {
      opacity: 0;
      transform: scale(0.9);
      background: transparent;
      border-color: transparent;
      box-shadow: none;
      backdrop-filter: blur(0px);
      -webkit-backdrop-filter: blur(0px);
    }
  }
  :global(.permanent-track) {
    animation: none !important;
    opacity: 0.22 !important;
  }
  :global(.permanent-pulse) {
    animation: none !important;
    opacity: 0.92 !important;
    stroke-dashoffset: 0 !important;
  }
  :global(.permanent-bloom) {
    animation: none !important;
    opacity: 0.92 !important;
    transform: scale(1.25) !important;
  }
  :global(.permanent-badge) {
    animation: none !important;
    opacity: 1 !important;
    transform: scale(1) !important;
    background: rgba(15, 23, 42, 0.88) !important;
    border: 1px solid rgba(245, 158, 11, 0.75) !important;
    backdrop-filter: blur(8px) !important;
    -webkit-backdrop-filter: blur(8px) !important;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.5), 0 0 12px rgba(245, 158, 11, 0.35) !important;
    text-shadow: 0 0 8px rgba(245, 158, 11, 0.7) !important;
  }
</style>
