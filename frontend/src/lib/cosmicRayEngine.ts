/**
 * Finnegan Cosmic Stage - Cosmic Ray Engine & Staggered Cascade Assembly
 * Layer 2 Pure Topological / Ray Calculation Engine (Zero DOM Mutation)
 */

import type { CosmicCard } from '../types/card';
import { computeAnimatedBeamPath, type BeamCoordinates } from './beamMath';
import { cardHasTag, cardHasKeyword, getCardTags } from './tagNetwork';
import { isClusteredChildCard } from './stageCoords';

export interface MotherChildRayDef {
  id: string;
  source: CosmicCard;
  target: CosmicCard;
  origin: string;
  matchedAnchor?: string;
  echoReason?: string;
}

export interface SynapseRayDef {
  id: string;
  source: CosmicCard;
  target: CosmicCard;
  tag: string;
}

export interface CosmicRay {
  instanceId: number;
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

export interface ActiveRayInstance {
  instanceId: number;
  rayId: string;
  sourceId: string;
  targetId: string;
}

/**
 * Builds mother-child ray definitions between mother cards and their echo child cards.
 * Excludes physically clustered child cards under mother cards.
 */
export function buildMotherChildRayDefs(cards: CosmicCard[] = []): MotherChildRayDef[] {
  const list: MotherChildRayDef[] = [];

  for (const child of cards) {
    if ((child.origin === 'bugu' || child.origin === 'atproto') && child.echoOf && !child.isFadingOut) {
      if (isClusteredChildCard(child, cards)) continue;
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
}

/**
 * Builds synapse ray definitions connecting a hovered card with other stage cards sharing manual tags/keywords.
 */
export function buildSynapseRayDefs(
  cards: CosmicCard[] = [],
  hoveredCardId: string | null = null,
  focusedCardId: string | null = null
): SynapseRayDef[] {
  if (focusedCardId !== null || !hoveredCardId) return [];

  const hoveredCard = cards.find(c => c.id === hoveredCardId && !c.isFadingOut);
  if (!hoveredCard) return [];

  const hoveredTags = getCardTags(hoveredCard);
  if (hoveredTags.length === 0) return [];

  const list: SynapseRayDef[] = [];
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
}

export interface AssembleRenderedRaysParams {
  cards: CosmicCard[];
  focusedCardId: string | null;
  hoveredCardId: string | null;
  motherChildRayDefs: MotherChildRayDef[];
  synapseRayDefs: SynapseRayDef[];
  activeInstance: ActiveRayInstance | null;
}

/**
 * Assembles all active and visible cosmic rays with docking curves and cascading stagger indices.
 * Rules:
 * A. Mother-Child Rays: ONLY when hovering a child card (points strictly to its mother).
 * B. Mother-Child Rays: Ambient Decaying Pulse (strictly 1 single pulse at a time).
 * C. Synapse Rays: Cascading Ice Blue rays while a card is hovered.
 */
export function assembleRenderedRays(params: AssembleRenderedRaysParams): CosmicRay[] {
  const {
    cards,
    focusedCardId,
    hoveredCardId,
    motherChildRayDefs,
    synapseRayDefs,
    activeInstance,
  } = params;

  // Hide all rays during full modal focus
  if (focusedCardId !== null) return [];

  const rays: CosmicRay[] = [];
  let cascadeIndex = 0;

  // A. Mother-Child Rays: ONLY when hovering a child card (points strictly to its mother).
  const permanentMotherRayMap = new Set<string>();
  if (hoveredCardId) {
    const hoveredChild = cards.find(c => c.id === hoveredCardId && Boolean(c.echoOf) && !c.isFadingOut);
    if (hoveredChild && hoveredChild.echoOf) {
      const mother = cards.find(c => c.id === hoveredChild.echoOf && !c.isFadingOut);
      if (mother) {
        const matchDef = motherChildRayDefs.find(d => d.source.id === mother.id && d.target.id === hoveredChild.id);
        if (matchDef) {
          permanentMotherRayMap.add(matchDef.id);
          const coords = computeAnimatedBeamPath(mother, hoveredChild);
          rays.push({
            instanceId: 0,
            id: `perm:${matchDef.id}`,
            type: 'mother_child',
            origin: matchDef.origin,
            source: mother,
            target: hoveredChild,
            badgeText: matchDef.matchedAnchor,
            badgeTitle: matchDef.echoReason || matchDef.matchedAnchor,
            coords,
            staggerIndex: 0,
            isPermanent: true,
          });
        }
      }
    }
  }

  // B. Mother-Child Rays: Ambient Decaying Pulse (Strictly 1 Single Pulse)
  if (activeInstance && !permanentMotherRayMap.has(activeInstance.rayId)) {
    const def = motherChildRayDefs.find(r => r.id === activeInstance.rayId);
    if (def) {
      const source = cards.find(c => c.id === activeInstance.sourceId && !c.isFadingOut);
      const target = cards.find(c => c.id === activeInstance.targetId && !c.isFadingOut);
      if (source && target) {
        const coords = computeAnimatedBeamPath(source, target);
        rays.push({
          instanceId: activeInstance.instanceId,
          id: `inst:${activeInstance.instanceId}:${activeInstance.rayId}`,
          type: 'mother_child',
          origin: def.origin,
          source,
          target,
          badgeText: def.matchedAnchor,
          badgeTitle: def.echoReason || def.matchedAnchor,
          coords,
          staggerIndex: 0,
          isPermanent: false,
        });
      }
    }
  }

  // C. Synapse Rays (Cascading Ice Blue)
  for (const syn of synapseRayDefs) {
    const coords = computeAnimatedBeamPath(syn.source, syn.target);
    rays.push({
      instanceId: 0,
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
}
