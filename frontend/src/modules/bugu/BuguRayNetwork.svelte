<!--
  ======================================================================================
  BuguRayNetwork.svelte - Bugu Optical Ray Network & Dynamic Laser Fibers
  Task: bugu_babel_meta_and_bezier_persistence
  ======================================================================================
-->
<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import type { BuguBeacon } from './types';
  import BuguRadarBeacon from './BuguRadarBeacon.svelte';

  export let cards: CosmicCard[] = [];
  export let beacon: BuguBeacon | null = null;
  // Legacy session prop fallback
  export let session: any = null;

  $: activeBeacon = beacon || (session ? { sourceCardId: session.sourceCardId, status: session.status } : null);

  $: beaconCard = activeBeacon 
    ? cards.find(c => c.id === activeBeacon.sourceCardId && !c.isFadingOut) 
    : null;

  // Persistently derive all active laser rays between mother cards and their Babel echo children
  $: activeRays = (() => {
    const list: Array<{
      id: string;
      source: CosmicCard;
      target: CosmicCard;
    }> = [];

    for (const child of cards) {
      if (child.type === 'babel' && child.echoOf && !child.isFadingOut) {
        const mother = cards.find(c => c.id === child.echoOf && !c.isFadingOut);
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

  // Compute organic cubic bezier paths connecting source to each target
  function calculatePath(x0: number, y0: number, x1: number, y1: number): string {
    const dx = x1 - x0;
    const dy = y1 - y0;
    const dist = Math.sqrt(dx * dx + dy * dy);

    if (dist < 1) return `M ${x0} ${y0}`;

    // Perpendicular vector for organic arc curve
    const nx = -dy / dist;
    const ny = dx / dist;
    const curvature = Math.min(dist * 0.22, 55);

    const cx0 = x0 + dx * 0.35 + nx * curvature;
    const cy0 = y0 + dy * 0.35 + ny * curvature;
    const cx1 = x0 + dx * 0.65 + nx * curvature;
    const cy1 = y0 + dy * 0.65 + ny * curvature;

    return `M ${x0} ${y0} C ${cx0} ${cy0}, ${cx1} ${cy1}, ${x1} ${y1}`;
  }
</script>

<div class="bugu-ray-network">
  <!-- Mother Card Radar Beacon (Vanishes after scanning) -->
  {#if beaconCard && activeBeacon}
    <BuguRadarBeacon x={beaconCard.x} y={beaconCard.y} status={activeBeacon.status} />
  {/if}

  <!-- Persistent SVG Laser Fibers -->
  {#if activeRays.length > 0}
    <svg class="bugu-ray-svg" viewBox="-2000 -2000 4000 4000">
      <defs>
        <!-- Amber gold neon glow filter -->
        <filter id="bugu-laser-glow" x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur in="SourceGraphic" stdDeviation="3.5" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>

        <!-- Gradient from mother to child -->
        <linearGradient id="bugu-ray-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#F59E0B" stop-opacity="0.95" />
          <stop offset="70%" stop-color="#FBBF24" stop-opacity="0.85" />
          <stop offset="100%" stop-color="#FDE68A" stop-opacity="0.95" />
        </linearGradient>
      </defs>

      {#each activeRays as ray (ray.id)}
        {@const pathD = calculatePath(ray.source.x, ray.source.y, ray.target.x, ray.target.y)}
        <!-- Ambient outer glow line -->
        <path
          d={pathD}
          class="ray-glow"
        />

        <!-- Core animated pulse laser line -->
        <path
          d={pathD}
          class="ray-core"
        />

        <!-- Optical anchor node at child target -->
        <circle 
          cx={ray.target.x} 
          cy={ray.target.y} 
          r="4.5" 
          class="ray-node"
        />
      {/each}
    </svg>
  {/if}
</div>

<style>
  .bugu-ray-network {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
    z-index: 1;
  }

  .bugu-ray-svg {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 4000px;
    height: 4000px;
    transform: translate(-50%, -50%);
    overflow: visible;
    pointer-events: none;
  }

  /* Wide ambient halo */
  .ray-glow {
    fill: none;
    stroke: rgba(245, 158, 11, 0.32);
    stroke-width: 6px;
    stroke-linecap: round;
    filter: url(#bugu-laser-glow);
    transition: opacity 0.5s ease-out;
  }

  /* Sharp golden pulsing core */
  .ray-core {
    fill: none;
    stroke: url(#bugu-ray-gradient);
    stroke-width: 2px;
    stroke-linecap: round;
    stroke-dasharray: 8, 8;
    animation: laser-flow 1.2s linear infinite;
    filter: drop-shadow(0 0 5px rgba(245, 158, 11, 0.95));
    transition: opacity 0.5s ease-out;
  }

  @keyframes laser-flow {
    from {
      stroke-dashoffset: 32;
    }
    to {
      stroke-dashoffset: 0;
    }
  }

  /* Sparkling anchor node at target card */
  .ray-node {
    fill: #FDE68A;
    stroke: #F59E0B;
    stroke-width: 1.5px;
    filter: drop-shadow(0 0 6px #F59E0B);
    animation: node-pulse 2s ease-in-out infinite alternate;
  }

  @keyframes node-pulse {
    from {
      r: 3.5px;
      opacity: 0.7;
    }
    to {
      r: 5.5px;
      opacity: 1;
    }
  }
</style>
