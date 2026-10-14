<!--
  ======================================================================================
  BuguRayNetwork.svelte - Bugu Directed Decaying Cross-Screen Rays & Radar Beacon
  Task: temporary_card_and_bugu_ray_refine_028
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
  // Legacy session prop fallback
  export let session: any = null;
  $: void scrollExpansion;

  $: activeBeacon = beacon || (session ? { sourceCardId: session.sourceCardId, status: session.status } : null);

  $: beaconCard = activeBeacon 
    ? cards.find(c => c.id === activeBeacon.sourceCardId && !c.isFadingOut) 
    : null;

  // Requirement 6: Cross-screen connectivity without on-screen clipping
  $: availableRays = (() => {
    const list: Array<{
      id: string;
      source: CosmicCard;
      target: CosmicCard;
    }> = [];

    for (const child of cards) {
      if (child.type === 'babel' && child.echoOf && !child.isFadingOut) {
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

  // Requirement 4 & 7: Asynchronous staggered ray scheduler
  const SINGLE_PULSE_MS = 1300;
  const MIN_NEXT_DELAY_MS = SINGLE_PULSE_MS; // Lower bound is duration of 1 pulse
  const RANDOM_ADDITIONAL_MS = 900;          // Range [1300ms, 2200ms]

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

    // Requirement 7: Write the random delay immediately when ray is launched
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

  $: renderedRays = activeInstances
    .map(inst => {
      const source = cards.find(c => c.id === inst.sourceId && !c.isFadingOut);
      const target = cards.find(c => c.id === inst.targetId && !c.isFadingOut);
      return source && target
        ? {
            instanceId: inst.instanceId,
            id: inst.rayId,
            source,
            target
          }
        : null;
    })
    .filter(Boolean) as Array<{
      instanceId: number;
      id: string;
      source: CosmicCard;
      target: CosmicCard;
    }>;

  onDestroy(() => {
    if (nextSpawnTimer) clearTimeout(nextSpawnTimer);
  });
</script>

<!-- Requirement 5: Lowest 3D z-plane behind all cards (translateZ(-25px), z-index: 0) -->
<div class="bugu-ray-network">
  <!-- Mother Card Radar Beacon (Remains active until all Babel cards have appeared) -->
  {#if beaconCard && activeBeacon}
    <BuguRadarBeacon x={beaconCard.x} y={beaconCard.y} status={activeBeacon.status} />
  {/if}

  <!-- Multi-Screen Laser Beams (Behind Cards in 3D, Staggered Directed Flow) -->
  {#if renderedRays.length > 0}
    <svg class="bugu-flat-svg" viewBox="-800 -500 1600 2800">
      {#each renderedRays as ray (ray.instanceId)}
        <g class="ray-instance">
          <!-- Faint Hairline Guide Thread (0.8px) that dims over 3.9s -->
          <line
            x1={ray.source.x}
            y1={ray.source.y}
            x2={ray.target.x}
            y2={ray.target.y}
            class="ray-guide"
          />
          <!-- Dynamic Directed Laser Beam (1.1px) flowing Mother -> Child -->
          <line
            x1={ray.source.x}
            y1={ray.source.y}
            x2={ray.target.x}
            y2={ray.target.y}
            pathLength="100"
            class="ray-beam"
            on:animationend={() => handleRayFinished(ray.instanceId)}
          />
        </g>
      {/each}
    </svg>
  {/if}
</div>

<style>
  /* Requirement 5: Deepest 3D z-plane behind cards */
  .bugu-ray-network {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
    z-index: 0;
    transform: translateZ(-25px);
    transform-style: preserve-3d;
  }

  /* Requirement 6: Spans entire multi-screen stage from Screen 1 down to Screen 4 */
  .bugu-flat-svg {
    position: absolute;
    top: 0;
    left: 50%;
    width: 1600px;
    height: 2800px;
    transform: translate(-50%, 0);
    overflow: visible;
    pointer-events: none;
  }

  .ray-instance {
    pointer-events: none;
  }

  /* Requirement 4: Delicate 0.8px Guide Line: dims progressively over 3 pulses (3.9s) */
  .ray-guide {
    stroke: rgba(245, 158, 11, 0.18);
    stroke-width: 0.8px;
    stroke-linecap: round;
    animation: bugu-guide-decay 3.9s ease-in-out forwards;
  }

  /* Requirement 4: Fine 1.1px Directed Laser Beam: 3 pulses from Mother (0) to Child (100), progressively dimmer */
  .ray-beam {
    stroke: #fbbf24;
    stroke-width: 1.1px;
    stroke-linecap: round;
    stroke-dasharray: 40 100;
    filter: drop-shadow(0 0 2.5px rgba(245, 158, 11, 0.8));
    animation: bugu-directed-beam 3.9s ease-in-out forwards;
  }

  @keyframes bugu-guide-decay {
    0% { opacity: 0; }
    4% { opacity: 0.28; }
    33% { opacity: 0.28; }
    38% { opacity: 0.16; }
    66% { opacity: 0.16; }
    71% { opacity: 0.08; }
    94% { opacity: 0.08; }
    100% { opacity: 0; }
  }

  @keyframes bugu-directed-beam {
    /* Pulse 1: Bright (0s ~ 1.3s) */
    0% {
      stroke-dashoffset: 40;
      opacity: 0;
    }
    4% {
      opacity: 0.95;
    }
    27% {
      opacity: 0.95;
    }
    33.3% {
      stroke-dashoffset: -100;
      opacity: 0;
    }

    /* Pulse 2: Medium (1.3s ~ 2.6s) */
    33.4% {
      stroke-dashoffset: 40;
      opacity: 0;
    }
    38% {
      opacity: 0.55;
    }
    60% {
      opacity: 0.55;
    }
    66.6% {
      stroke-dashoffset: -100;
      opacity: 0;
    }

    /* Pulse 3: Faint (2.6s ~ 3.9s) */
    66.7% {
      stroke-dashoffset: 40;
      opacity: 0;
    }
    72% {
      opacity: 0.22;
    }
    94% {
      opacity: 0.22;
    }
    100% {
      stroke-dashoffset: -100;
      opacity: 0;
    }
  }
</style>
