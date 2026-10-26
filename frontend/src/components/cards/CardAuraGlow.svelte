<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CardAuraGlow Component
  Modularized Aura spinning halo and Bugu traveling border beam effect layer.
  ======================================================================================
-->
<script lang="ts">
  export let card: any;
  export let hasAura: boolean = false;
  export let auraStage: number = 0;
  export let conicGradient: string = '';
  export let isBuguEcho: boolean = false;
  export let isFocused: boolean = false;

  $: animDelay = -((card?.id ? card.id.charCodeAt(card.id.length - 1) : 0) % 5);
</script>

{#if hasAura && auraStage < 3}
  <div class="aura-spinner-container">
    <div 
      class="aura-spinner" 
      class:slow={auraStage === 2}
      style="background: {conicGradient};"
    ></div>
  </div>
  <div class="aura-inner-mask" class:focused={isFocused}></div>
{:else if isBuguEcho && !isFocused}
  <!-- Bugu Echo Traveling Border Beam (Scattered State) -->
  <div class="bugu-beam-container">
    <div 
      class="bugu-beam-spinner" 
      style="animation-delay: {animDelay}s;"
    ></div>
  </div>
  <div class="bugu-beam-mask"></div>
{/if}

<style>
  /* Bugu Echo Traveling Border Beam (Scattered State) */
  .bugu-beam-container {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    overflow: hidden;
    pointer-events: none;
    z-index: 0;
  }

  .bugu-beam-spinner {
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    transform-origin: center center;
    background: conic-gradient(
      from 0deg,
      transparent 0deg,
      transparent 295deg,
      rgba(245, 158, 11, 0) 300deg,
      rgba(245, 158, 11, 0.2) 320deg,
      rgba(245, 158, 11, 0.7) 345deg,
      rgba(251, 191, 36, 1) 360deg
    );
    animation: bugu-beam-spin 4.6s linear infinite;
    pointer-events: none;
  }

  @keyframes bugu-beam-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .bugu-beam-mask {
    position: absolute;
    inset: 1px;
    border-radius: 15px;
    background: linear-gradient(135deg, rgba(230, 239, 250, 0.94) 0%, rgba(216, 228, 244, 0.90) 100%);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    pointer-events: none;
    z-index: 0;
  }

  /* Flowing Light Aura Animation Layers */
  .aura-spinner-container {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    overflow: hidden;
    pointer-events: none;
    z-index: 0;
  }

  .aura-spinner {
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    transform-origin: center center;
    animation: aura-spin 4s linear infinite;
    pointer-events: none;
  }

  .aura-spinner.slow {
    animation: aura-spin 9s linear infinite;
  }

  @keyframes aura-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .aura-inner-mask {
    position: absolute;
    inset: 2px;
    border-radius: 14px;
    background: rgba(255, 255, 255, 0.90);
    pointer-events: none;
    z-index: 0;
  }

  .aura-inner-mask.focused {
    border-radius: 26px;
    background: rgba(255, 255, 255, 0.96);
  }
</style>
