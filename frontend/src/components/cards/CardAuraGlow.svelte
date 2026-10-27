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
  export let isRecentVIP: boolean = false;
  export let isTemporary: boolean = false;

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
{:else if isRecentVIP && !isFocused}
  <!-- Ink Wash Traveling Light Beam (水墨流转飞白光弧) -->
  <div class="ink-beam-container">
    <div 
      class="ink-beam-spinner" 
      style="animation-delay: {animDelay}s;"
    ></div>
  </div>
  <div class="ink-beam-mask" class:temporary={isTemporary}></div>
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

  /* Ink Wash Traveling Light Beam (水墨流转飞白光弧) */
  .ink-beam-container {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    overflow: hidden;
    pointer-events: none;
    z-index: 0;
  }

  .ink-beam-spinner {
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    transform-origin: center center;
    background: conic-gradient(
      from 0deg,
      transparent 0deg,
      transparent 135deg,
      rgba(100, 116, 139, 0.18) 155deg,
      rgba(51, 65, 85, 0.60) 175deg,
      transparent 185deg,
      transparent 220deg,
      rgba(71, 85, 105, 0.25) 245deg,
      rgba(51, 65, 85, 0.65) 280deg,
      rgba(30, 41, 59, 0.90) 315deg,
      rgba(15, 23, 42, 1.0) 345deg,
      rgba(0, 0, 0, 1.0) 355deg,
      transparent 360deg
    );
    animation: ink-beam-spin 3.8s linear infinite;
    pointer-events: none;
  }

  @keyframes ink-beam-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .ink-beam-mask {
    position: absolute;
    inset: 2px;
    border-radius: 14px;
    background: rgba(255, 255, 255, 0.92);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    pointer-events: none;
    z-index: 0;
  }

  .ink-beam-mask.temporary {
    background: linear-gradient(135deg, rgba(226, 236, 248, 0.88) 0%, rgba(210, 222, 238, 0.80) 100%);
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
