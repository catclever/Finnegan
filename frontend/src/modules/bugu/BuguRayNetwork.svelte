<!--
  ======================================================================================
  BuguRayNetwork.svelte - Forwarder for backwards compatibility
  Delegates directly to the universal CosmicRayNetwork component.
  ======================================================================================
-->
<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import type { BuguBeacon } from './types';
  import CosmicRayNetwork from '../../components/cards/CosmicRayNetwork.svelte';

  export let cards: CosmicCard[] = [];
  export let beacon: BuguBeacon | null = null;
  export let scrollExpansion: number = 0;
  export let hoveredMotherId: string | null = null;
  export let session: any = null;
</script>

<CosmicRayNetwork 
  {cards} 
  {beacon} 
  {session} 
  {scrollExpansion} 
  {hoveredMotherId} 
/>
