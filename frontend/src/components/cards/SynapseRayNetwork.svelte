<!--
  ======================================================================================
  SynapseRayNetwork.svelte - Ice Blue Synapse Ray Network & Tag Association Constellation
  Task: tag_association_display_network_103
  ======================================================================================
-->
<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import { cardHasTag, getCardTags } from '../../lib/tagNetwork';

  export let cards: CosmicCard[] = [];
  export let hoveredCardId: string | null = null;
  export let activeConstellationTag: string | null = null;
  export let scrollExpansion: number = 0;
  export let focusedCardId: string | null = null;
  $: void scrollExpansion;

  interface SynapseRay {
    id: string;
    source: CosmicCard;
    target: CosmicCard;
    tag: string;
    path: string;
    x1: number;
    y1: number;
    x2: number;
    y2: number;
    midX: number;
    midY: number;
  }

  function getCardScale(card: CosmicCard): number {
    const zNorm = Math.max(0, Math.min(1, (card.z || 0) / 100));
    return 1.08 - (zNorm * 0.44);
  }

  const DOCKING_INSET_PX = 2;

  function computeSynapseBeamPath(
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
        midY: source.y,
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
      if (rawDx >= 0) {
        startX = source.x + sourceHalfW;
        endX = target.x - targetHalfW;
      } else {
        startX = source.x - sourceHalfW;
        endX = target.x + targetHalfW;
      }
      const maxSagY = Math.min(24, Math.abs(rawDy) * 0.2);
      startY = source.y + (rawDy > 0 ? maxSagY : -maxSagY);
      endY = target.y + (rawDy > 0 ? -maxSagY : maxSagY);

      const dx = endX - startX;
      const curvature = Math.min(180, Math.max(40, Math.abs(dx) * 0.45));

      c1x = startX + (dx >= 0 ? curvature : -curvature);
      c1y = startY;
      c2x = endX - (dx >= 0 ? curvature : -curvature);
      c2y = endY;
    } else {
      if (rawDy >= 0) {
        startY = source.y + sourceHalfH;
        endY = target.y - targetHalfH;
      } else {
        startY = source.y - sourceHalfH;
        endY = target.y + targetHalfH;
      }
      const maxSagX = Math.min(24, Math.abs(rawDx) * 0.2);
      startX = source.x + (rawDx > 0 ? maxSagX : -maxSagX);
      endX = target.x + (rawDx > 0 ? -maxSagX : maxSagX);

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

  $: renderedRays = (() => {
    // Hide rays during full modal focus
    if (focusedCardId !== null) return [];

    const rayMap = new Map<string, { source: CosmicCard; target: CosmicCard; tag: string }>();

    // Trigger B: activeConstellationTag is set: connects all pairs of cards on stage possessing that tag
    if (activeConstellationTag) {
      const tag = activeConstellationTag;
      const matchingCards = cards.filter(c => !c.isFadingOut && cardHasTag(c, tag));
      for (let i = 0; i < matchingCards.length; i++) {
        for (let j = i + 1; j < matchingCards.length; j++) {
          const c1 = matchingCards[i];
          const c2 = matchingCards[j];
          const key = [c1.id, c2.id].sort().join('--');
          rayMap.set(key, { source: c1, target: c2, tag });
        }
      }
    }

    // Trigger A: hoveredCardId is set: connects hovered card with all other cards on stage sharing any of its tags
    if (hoveredCardId) {
      const hoveredCard = cards.find(c => c.id === hoveredCardId && !c.isFadingOut);
      if (hoveredCard) {
        const hoveredTags = getCardTags(hoveredCard);
        if (hoveredTags.length > 0) {
          for (const other of cards) {
            if (other.id !== hoveredCard.id && !other.isFadingOut) {
              const sharedTag = hoveredTags.find(tag => cardHasTag(other, tag));
              if (sharedTag) {
                const key = [hoveredCard.id, other.id].sort().join('--');
                if (!rayMap.has(key)) {
                  rayMap.set(key, { source: hoveredCard, target: other, tag: sharedTag });
                }
              }
            }
          }
        }
      }
    }

    const result: SynapseRay[] = [];
    for (const [key, item] of rayMap.entries()) {
      const beam = computeSynapseBeamPath(item.source, item.target);
      result.push({
        id: key,
        source: item.source,
        target: item.target,
        tag: item.tag,
        path: beam.path,
        x1: beam.x1,
        y1: beam.y1,
        x2: beam.x2,
        y2: beam.y2,
        midX: beam.midX,
        midY: beam.midY,
      });
    }

    return result;
  })();
</script>

<div class="synapse-ray-network">
  {#if renderedRays.length > 0}
    <!-- Midpoint Floating Anchor Badges -->
    <div class="synapse-flat-html">
      {#each renderedRays as ray (ray.id)}
        <div 
          class="synapse-badge-container"
          style="transform: translate(calc({ray.midX}px - 50%), calc({ray.midY}px - 50%)) translateZ(8px);"
        >
          <div class="synapse-anchor-badge">
            #{ray.tag}
          </div>
        </div>
      {/each}
    </div>

    <!-- Undirected Ethereal Ice Blue SVG Bezier Synapse Beams -->
    <svg class="synapse-flat-svg">
      <defs>
        <!-- Soft glowing radial gradient for aperture docking nodes in Ice Blue (#38bdf8) -->
        <radialGradient id="synapse-aperture-glow" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#38bdf8" stop-opacity="1" />
          <stop offset="35%" stop-color="#38bdf8" stop-opacity="0.65" />
          <stop offset="70%" stop-color="#38bdf8" stop-opacity="0.2" />
          <stop offset="100%" stop-color="#38bdf8" stop-opacity="0" />
        </radialGradient>
      </defs>

      {#each renderedRays as ray (ray.id)}
        <g class="synapse-ray-instance">
          <!-- Docking Node 1 Aperture Bloom -->
          <g class="aperture-group" transform="translate({ray.x1}, {ray.y1})">
            <circle cx="0" cy="0" r="16" fill="url(#synapse-aperture-glow)" class="synapse-glow-ring" />
            <circle cx="0" cy="0" r="2.5" fill="#38bdf8" class="synapse-core-dot" />
          </g>

          <!-- Ambient Glow Track -->
          <path
            d={ray.path}
            class="synapse-beam-glow"
          />

          <!-- Crisp Ethereal Ice Blue Core Track (Undirected) -->
          <path
            d={ray.path}
            class="synapse-beam-core"
          />

          <!-- Docking Node 2 Aperture Bloom -->
          <g class="aperture-group" transform="translate({ray.x2}, {ray.y2})">
            <circle cx="0" cy="0" r="16" fill="url(#synapse-aperture-glow)" class="synapse-glow-ring" />
            <circle cx="0" cy="0" r="2.5" fill="#38bdf8" class="synapse-core-dot" />
          </g>
        </g>
      {/each}
    </svg>
  {/if}
</div>

<style>
  /* Requirement 4: Renders beneath cards at z-index: -1, transform: translateZ(-5px), pointer-events: none */
  .synapse-ray-network {
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

  .synapse-flat-svg {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    overflow: visible;
    pointer-events: none;
  }

  .synapse-ray-instance {
    pointer-events: none;
  }

  .aperture-group {
    pointer-events: none;
  }

  .synapse-glow-ring {
    transform-origin: 0 0;
    filter: drop-shadow(0 0 6px rgba(56, 189, 248, 0.85));
  }

  .synapse-core-dot {
    transform-origin: 0 0;
    filter: drop-shadow(0 0 3px #38bdf8);
  }

  /* Ambient glowing ray background */
  .synapse-beam-glow {
    fill: none;
    stroke: rgba(56, 189, 248, 0.32);
    stroke-width: 3.5px;
    filter: drop-shadow(0 0 6px rgba(56, 189, 248, 0.65));
    transition: opacity 0.3s ease;
  }

  /* Crisp ethereal Ice Blue core (#38bdf8) */
  .synapse-beam-core {
    fill: none;
    stroke: #38bdf8;
    stroke-width: 1.2px;
    stroke-linecap: round;
    filter: drop-shadow(0 0 2px rgba(56, 189, 248, 0.9));
    transition: opacity 0.3s ease;
  }

  .synapse-flat-html {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    overflow: visible;
    pointer-events: none;
  }

  .synapse-badge-container {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none;
    display: flex;
    align-items: center;
    justify-content: center;
    animation: badge-fade-in 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  }

  /* Floating Anchor Badge: 10px typography, glass aesthetic in ethereal Ice Blue */
  .synapse-anchor-badge {
    font-size: 10px;
    font-weight: 500;
    color: #0284c7;
    background: rgba(255, 255, 255, 0.88);
    border: 1px solid rgba(56, 189, 248, 0.45);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    padding: 1px 7px;
    border-radius: 6px;
    box-shadow: 0 2px 8px rgba(56, 189, 248, 0.25), 0 0 1px rgba(0, 0, 0, 0.08);
    letter-spacing: 0.02em;
    user-select: none;
    white-space: nowrap;
    pointer-events: none;
  }

  @keyframes badge-fade-in {
    from {
      opacity: 0;
      transform: scale(0.85);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
