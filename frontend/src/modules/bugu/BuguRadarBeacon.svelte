<!--
  ======================================================================================
  BuguRadarBeacon.svelte - Bugu Radar Emitting Waves & Beacon
  Task: bugu_ray_network_and_beacon
  ======================================================================================
-->
<script lang="ts">
  export let x: number = 0;
  export let y: number = 0;
  export let status: 'scanning' | 'linked' | 'fading' = 'scanning';
</script>

<div 
  class="bugu-beacon-wrapper" 
  class:fading={status === 'fading'}
  style="transform: translate(calc(-50% + {x}px), calc(-50% + {y}px));"
>
  <!-- Concentric expanding sonar pulse rings -->
  <div class="beacon-ring ring-1"></div>
  <div class="beacon-ring ring-2"></div>
  <div class="beacon-ring ring-3"></div>

  <!-- Rotating Radar Sweep Cone -->
  <div class="radar-sweep"></div>
</div>

<style>
  .bugu-beacon-wrapper {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 240px;
    height: 240px;
    pointer-events: none;
    z-index: 2;
    transition: opacity 2.5s ease-out;
  }

  .bugu-beacon-wrapper.fading {
    opacity: 0;
  }

  /* Expanding Sonar Rings */
  .beacon-ring {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    border-radius: 50%;
    border: 1.5px solid rgba(245, 158, 11, 0.45);
    box-shadow: 0 0 16px rgba(245, 158, 11, 0.3), inset 0 0 16px rgba(245, 158, 11, 0.15);
    animation: beacon-expand 4.5s infinite cubic-bezier(0.2, 0.8, 0.2, 1);
    opacity: 0;
  }

  .ring-1 {
    animation-delay: 0s;
  }
  .ring-2 {
    animation-delay: 1.5s;
  }
  .ring-3 {
    animation-delay: 3s;
  }

  @keyframes beacon-expand {
    0% {
      transform: scale(0.65);
      opacity: 0.85;
      border-color: rgba(245, 158, 11, 0.7);
    }
    50% {
      opacity: 0.5;
    }
    100% {
      transform: scale(2.6);
      opacity: 0;
      border-color: rgba(245, 158, 11, 0);
    }
  }

  /* Radar sweep beam */
  .radar-sweep {
    position: absolute;
    top: -20px;
    left: -20px;
    width: 280px;
    height: 280px;
    border-radius: 50%;
    background: conic-gradient(
      from 0deg,
      transparent 0deg,
      rgba(245, 158, 11, 0.18) 50deg,
      rgba(245, 158, 11, 0.35) 60deg,
      transparent 61deg
    );
    animation: radar-rotate 4s linear infinite;
    mix-blend-mode: screen;
  }

  @keyframes radar-rotate {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }
</style>
