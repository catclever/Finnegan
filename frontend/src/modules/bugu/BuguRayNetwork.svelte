<!--
  ======================================================================================
  BuguRayNetwork.svelte - Bugu Optical Ray Network & Dynamic Laser Fibers
  Task: bugu_ray_network_and_beacon
  ======================================================================================
-->
<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import type { BuguSession } from './types';
  import BuguRadarBeacon from './BuguRadarBeacon.svelte';

  export let session: BuguSession;
  export let cards: CosmicCard[] = [];

  $: sourceCard = cards.find(c => c.id === session.sourceCardId);

  $: targetCards = session.targetCardIds
    .map(id => cards.find(c => c.id === id))
    .filter((c): c is CosmicCard => Boolean(c));

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

<div class="bugu-ray-network" class:fading={session.status === 'fading'}>
  <!-- Mother Card Radar Beacon -->
  {#if sourceCard}
    <BuguRadarBeacon x={sourceCard.x} y={sourceCard.y} status={session.status} />
  {/if}

  <!-- SVG Laser Fibers -->
  {#if sourceCard && targetCards.length > 0}
    <svg class="bugu-ray-svg">
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
          <stop offset="0%" stop-color="#F59E0B" stop-opacity="0.9" />
          <stop offset="70%" stop-color="#FBBF24" stop-opacity="0.75" />
          <stop offset="100%" stop-color="#FDE68A" stop-opacity="0.95" />
        </linearGradient>
      </defs>

      {#each targetCards as target (target.id)}
        {@const pathD = calculatePath(sourceCard.x, sourceCard.y, target.x, target.y)}
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
          cx={target.x} 
          cy={target.y} 
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
    transition: opacity 2.5s ease-out;
  }

  .bugu-ray-network.fading {
    opacity: 0;
  }

  .bugu-ray-svg {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    overflow: visible;
    pointer-events: none;
  }

  /* Wide ambient halo */
  .ray-glow {
    fill: none;
    stroke: rgba(245, 158, 11, 0.28);
    stroke-width: 6px;
    stroke-linecap: round;
    filter: url(#bugu-laser-glow);
  }

  /* Sharp golden pulsing core */
  .ray-core {
    fill: none;
    stroke: url(#bugu-ray-gradient);
    stroke-width: 1.8px;
    stroke-linecap: round;
    stroke-dasharray: 8, 8;
    animation: laser-flow 1.2s linear infinite;
    filter: drop-shadow(0 0 4px rgba(245, 158, 11, 0.9));
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
