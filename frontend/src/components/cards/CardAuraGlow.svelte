<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CardAuraGlow Component
  Decoupled Universal Border Beam Presenter:
  - Consumes resolved preset from BorderBeamRegistry (zero hardcoded module styles)
  - Pure presenter: Single .beam-container + .beam-spinner + .beam-mask
  - 1.5px outer-boundary alignment matching Aura (zero outer gap)
  ======================================================================================
-->
<script lang="ts">
  import { resolveCardBorderBeam } from '../../lib/borderBeamRegistry';

  export let card: any;
  export let hasAura: boolean = false;
  export let auraStage: number = 0;
  export let conicGradient: string = '';
  // Legacy props kept for backwards compatibility with UICard caller
  export let isBuguEcho: boolean = false;
  export let isBskyEcho: boolean = false;
  export let isFocused: boolean = false;
  export let isRecentVIP: boolean = false;
  export let isTemporary: boolean = false;
  export let isStopped: boolean = false;

  $: void isBuguEcho;
  $: void isBskyEcho;

  $: activeBeam = resolveCardBorderBeam(card, {
    hasAura,
    auraStage,
    conicGradient,
    isFocused,
    isRecentVIP,
    isTemporary,
  });

  $: maskBg = activeBeam
    ? (isTemporary ? (activeBeam.maskTemporaryBackground || activeBeam.maskBackground || '') : (activeBeam.maskBackground || ''))
    : '';
</script>

{#if activeBeam}
  <div class="beam-container">
    <div 
      class="beam-spinner" 
      class:stopped={isStopped}
      style="
        background: {activeBeam.gradient};
        animation-duration: {activeBeam.duration};
        animation-delay: {activeBeam.delay}s;
      "
    ></div>
  </div>
  <div 
    class="beam-mask"
    class:surface-scan={activeBeam.surfaceScan}
    class:temporary={isTemporary}
    class:focused={isFocused}
    style="{maskBg ? `background: ${maskBg};` : ''}"
  ></div>
{/if}

<style>
  /* Unified Outer Container & Center-pivot Rotating Disk */
  .beam-container {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    overflow: hidden;
    pointer-events: none;
    z-index: 0;
  }

  .beam-spinner {
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    transform-origin: center center;
    animation: beam-spin 4.6s linear infinite;
    pointer-events: none;
  }

  .beam-spinner.stopped {
    animation-play-state: paused !important;
  }

  @keyframes beam-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  /* 统一内遮罩层：位于最外边界内侧 1.5px，对齐 Aura，留出 1.5px 纯净外缘光轨 */
  .beam-mask {
    position: absolute;
    inset: 1.5px;
    border-radius: 14.5px;
    pointer-events: none;
    z-index: 0;
  }

  .beam-mask.focused {
    border-radius: 26.5px;
  }
</style>
