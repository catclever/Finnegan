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
  export let isBskyEcho: boolean = false;
  export let isFocused: boolean = false;
  export let isRecentVIP: boolean = false;
  export let isTemporary: boolean = false;
  export let isStopped: boolean = false;

  $: animDelay = -((card?.id ? card.id.charCodeAt(card.id.length - 1) : 0) % 5);
  $: hash = card?.id ? card.id.split('').reduce((a: number, b: string) => a + b.charCodeAt(0), 0) : 0;
  $: auraDelay = -((hash % 40) / 10);
</script>

{#if hasAura && auraStage < 3}
  <div class="aura-spinner-container">
    <div 
      class="aura-spinner" 
      class:slow={auraStage === 2}
      class:stopped={isStopped}
      style="background: {conicGradient}; animation-delay: {auraDelay}s;"
    ></div>
  </div>
  <div class="aura-inner-mask" class:focused={isFocused}></div>
{:else if isBskyEcho && !isFocused}
  <!-- Bluesky Echo Traveling Border Beam (Ice Blue) -->
  <div class="bsky-beam-container">
    <div 
      class="bsky-beam-spinner" 
      class:stopped={isStopped}
      style="animation-delay: {animDelay}s;"
    ></div>
  </div>
  <div class="bsky-beam-mask"></div>
{:else if isBuguEcho && !isFocused}
  <!-- Bugu Echo Traveling Border Beam (Scattered State) -->
  <div class="bugu-beam-container">
    <div 
      class="bugu-beam-spinner" 
      class:stopped={isStopped}
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
  /* Bluesky Echo Traveling Border Beam (Ice Blue) */
  .bsky-beam-container {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    overflow: hidden;
    pointer-events: none;
    z-index: 0;
  }

  .bsky-beam-spinner {
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
      rgba(56, 189, 248, 0) 300deg,
      rgba(56, 189, 248, 0.25) 320deg,
      rgba(125, 211, 252, 0.75) 345deg,
      rgba(186, 230, 253, 1) 360deg
    );
    animation: bsky-beam-spin 4.6s linear infinite;
    pointer-events: none;
  }

  @keyframes bsky-beam-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .bsky-beam-mask {
    position: absolute;
    inset: 1px;
    border-radius: 15px;
    background: linear-gradient(135deg, rgba(240, 249, 255, 0.95) 0%, rgba(224, 242, 254, 0.90) 100%);
    pointer-events: none;
    z-index: 0;
  }

  /* Stopped State: Freeze all traveling beams & aura in stillness */
  .bsky-beam-spinner.stopped,
  .bugu-beam-spinner.stopped,
  .aura-spinner.stopped {
    animation-play-state: paused !important;
  }

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
      rgba(148, 163, 184, 0.20) 155deg,
      rgba(100, 116, 139, 0.45) 175deg,
      transparent 185deg,
      transparent 220deg,
      rgba(148, 163, 184, 0.22) 245deg,
      rgba(100, 116, 139, 0.52) 280deg,
      rgba(71, 85, 105, 0.72) 315deg,
      rgba(71, 85, 105, 0.88) 345deg,
      rgba(51, 65, 85, 0.88) 355deg,
      transparent 360deg
    );
    animation: ink-beam-spin 11.4s linear infinite;
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
