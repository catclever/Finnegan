<!--
  ======================================================================================
  SynapseRayNetwork.svelte - Forwarder for backwards compatibility
  Delegates directly to the universal CosmicRayNetwork component.
  ======================================================================================
-->
<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import CosmicRayNetwork from './CosmicRayNetwork.svelte';

  export let cards: CosmicCard[] = [];
  export let hoveredCardId: string | null = null;
  export let activeConstellationTag: string | null = null;
  export let scrollExpansion: number = 0;
  export let focusedCardId: string | null = null;
</script>

<CosmicRayNetwork 
  {cards} 
  {hoveredCardId} 
  {activeConstellationTag} 
  {scrollExpansion} 
  {focusedCardId} 
/>
