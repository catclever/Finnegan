<!--
  ======================================================================================
  BskyButterflyBeacon.svelte - Bluesky Echo Fluttering Butterflies 3D Beacon
  Task: atproto_butterfly_fluttering_search_frontend_128
  Renders ethereal 3D fluttering butterflies around the search target card.
  ======================================================================================
-->
<script lang="ts">
  export let x: number = 0;
  export let y: number = 0;
  export let status: 'scanning' | 'linked' | 'completed' | 'fading' = 'scanning';
  export let count: number = 3;

  $: isStopped = status === 'completed' || status === 'fading';

  interface ButterflySpec {
    id: number;
    trackIndex: number;
    size: number;
    orbitSpeed: number;
    orbitDelay: number;
    flapSpeed: number;
    wingHue: string;
    dustDelay: number;
  }

  // Pre-configured butterfly archetypes for natural variety
  const specsPool: ButterflySpec[] = [
    { id: 0, trackIndex: 0, size: 28, orbitSpeed: 4.8, orbitDelay: 0.0, flapSpeed: 0.22, wingHue: '#38bdf8', dustDelay: 0.0 },
    { id: 1, trackIndex: 1, size: 23, orbitSpeed: 5.6, orbitDelay: 1.2, flapSpeed: 0.20, wingHue: '#0ea5e9', dustDelay: 0.4 },
    { id: 2, trackIndex: 2, size: 26, orbitSpeed: 5.1, orbitDelay: 2.3, flapSpeed: 0.24, wingHue: '#7dd3fc', dustDelay: 0.8 },
    { id: 3, trackIndex: 3, size: 21, orbitSpeed: 4.4, orbitDelay: 0.7, flapSpeed: 0.19, wingHue: '#0284c7', dustDelay: 1.1 },
    { id: 4, trackIndex: 4, size: 25, orbitSpeed: 6.0, orbitDelay: 3.1, flapSpeed: 0.23, wingHue: '#38bdf8', dustDelay: 1.5 },
    { id: 5, trackIndex: 5, size: 22, orbitSpeed: 5.3, orbitDelay: 1.8, flapSpeed: 0.21, wingHue: '#a5f3fc', dustDelay: 0.6 },
    { id: 6, trackIndex: 1, size: 24, orbitSpeed: 4.9, orbitDelay: 2.8, flapSpeed: 0.25, wingHue: '#0ea5e9', dustDelay: 1.3 },
    { id: 7, trackIndex: 3, size: 20, orbitSpeed: 5.8, orbitDelay: 0.4, flapSpeed: 0.18, wingHue: '#7dd3fc', dustDelay: 0.9 },
  ];

  // Dynamically slice butterflies based on count prop
  $: activeButterflies = specsPool.slice(0, Math.min(Math.max(count, 1), specsPool.length));
</script>

<div 
  class="bsky-beacon-wrapper" 
  class:fading={status === 'fading'}
  class:stopped={isStopped}
  style="transform: translate(calc(-50% + {x}px), calc(-50% + {y}px));"
>
  <!-- Ambient Center Luminous Core (Soft Ethereal Azure Glow) -->
  <div class="ambient-azure-core" class:stopped={isStopped}></div>

  <!-- Concentric Ice-Blue Sonar Pulse Rings -->
  <div class="bsky-sonar-ring ring-1" class:stopped={isStopped}></div>
  <div class="bsky-sonar-ring ring-2" class:stopped={isStopped}></div>

  <!-- Luminous Ice-Blue Halo Ring & Rotating Shimmer -->
  <div class="bsky-scanning-halo" class:stopped={isStopped}></div>

  {#each activeButterflies as b (b.id)}
    <div 
      class="butterfly-orbit track-{b.trackIndex}"
      style="
        --orbit-speed: {b.orbitSpeed}s;
        --orbit-delay: -{b.orbitDelay}s;
        --size: {b.size}px;
      "
    >
      <!-- Emerge & Scatter Transform Wrapper -->
      <div class="butterfly-flight">
        <!-- Glowing Stardust Particle Trail -->
        <div 
          class="stardust-trail" 
          style="--dust-delay: {b.dustDelay}s; --glow-color: {b.wingHue};"
        ></div>

        <!-- 3D Fluttering Butterfly SVG -->
        <svg 
          class="butterfly-svg" 
          viewBox="-24 -24 48 48" 
          width="{b.size}" 
          height="{b.size}"
          style="--flap-speed: {b.flapSpeed}s;"
        >
          <defs>
            <linearGradient id="forewing-grad-{b.id}" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#bae6fd" stop-opacity="0.95" />
              <stop offset="55%" stop-color={b.wingHue} stop-opacity="0.85" />
              <stop offset="100%" stop-color="#0284c7" stop-opacity="0.65" />
            </linearGradient>
            <linearGradient id="hindwing-grad-{b.id}" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color={b.wingHue} stop-opacity="0.90" />
              <stop offset="100%" stop-color="#0369a1" stop-opacity="0.60" />
            </linearGradient>
          </defs>

          <!-- Thorax & Head -->
          <ellipse cx="0" cy="0" rx="1.4" ry="5.5" fill="#e0f2fe" />
          <circle cx="0" cy="-6" r="1.5" fill="#f0f9ff" />

          <!-- Antennae -->
          <path 
            d="M -0.5 -7 Q -3 -11 -6 -12.5 M 0.5 -7 Q 3 -11 6 -12.5" 
            stroke="#bae6fd" 
            stroke-width="0.85" 
            stroke-linecap="round" 
            fill="none" 
            opacity="0.85" 
          />

          <!-- Left Wings Pair -->
          <g class="wing-group left-wing">
            <!-- Left Forewing -->
            <path 
              d="M -0.5 -3 C -6 -14 -18 -16 -19 -8 C -20 -3 -14 2 -0.5 1 Z" 
              fill="url(#forewing-grad-{b.id})" 
            />
            <!-- Left Hindwing -->
            <path 
              d="M -0.5 0.5 C -7 3 -15 7 -13 13.5 C -10 18 -4 14.5 -0.5 3.5 Z" 
              fill="url(#hindwing-grad-{b.id})" 
            />
          </g>

          <!-- Right Wings Pair -->
          <g class="wing-group right-wing">
            <!-- Right Forewing -->
            <path 
              d="M 0.5 -3 C 6 -14 18 -16 19 -8 C 20 -3 14 2 0.5 1 Z" 
              fill="url(#forewing-grad-{b.id})" 
            />
            <!-- Right Hindwing -->
            <path 
              d="M 0.5 0.5 C 7 3 15 7 13 13.5 C 10 18 4 14.5 0.5 3.5 Z" 
              fill="url(#hindwing-grad-{b.id})" 
            />
          </g>
        </svg>
      </div>
    </div>
  {/each}
</div>

<style>
  .bsky-beacon-wrapper {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 380px;
    height: 320px;
    pointer-events: none;
    z-index: 12;
    transform-style: preserve-3d;
    transition: opacity 2s ease-out;
  }

  .bsky-beacon-wrapper.fading {
    opacity: 0;
  }

  /* Soft Ambient Azure Breath */
  .ambient-azure-core {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 220px;
    height: 180px;
    transform: translate(-50%, -50%) translateZ(-5px);
    border-radius: 50%;
    background: radial-gradient(
      ellipse at center,
      rgba(56, 189, 248, 0.16) 0%,
      rgba(14, 165, 233, 0.08) 45%,
      transparent 70%
    );
    animation: azure-pulse 4s ease-in-out infinite alternate;
    filter: blur(8px);
    pointer-events: none;
  }

  .ambient-azure-core.stopped,
  .bsky-beacon-wrapper.stopped .ambient-azure-core {
    animation-play-state: paused !important;
  }

  @keyframes azure-pulse {
    0% { transform: translate(-50%, -50%) translateZ(-5px) scale(0.85); opacity: 0.6; }
    100% { transform: translate(-50%, -50%) translateZ(-5px) scale(1.25); opacity: 1; }
  }

  /* Ice-Blue Scanning Halo Ring */
  .bsky-scanning-halo {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 290px;
    height: 240px;
    border-radius: 28px;
    border: 1.5px solid rgba(56, 189, 248, 0.65);
    box-shadow: 
      0 0 24px rgba(56, 189, 248, 0.45),
      inset 0 0 20px rgba(14, 165, 233, 0.25);
    transform: translate(-50%, -50%) translateZ(4px);
    pointer-events: none;
    animation: bsky-halo-spin 6s linear infinite;
  }

  .bsky-scanning-halo.stopped,
  .bsky-beacon-wrapper.stopped .bsky-scanning-halo {
    animation-play-state: paused !important;
  }

  @keyframes bsky-halo-spin {
    0% {
      transform: translate(-50%, -50%) translateZ(4px) rotate(0deg);
      border-color: rgba(56, 189, 248, 0.65);
      box-shadow: 0 0 24px rgba(56, 189, 248, 0.45), inset 0 0 20px rgba(14, 165, 233, 0.25);
    }
    50% {
      transform: translate(-50%, -50%) translateZ(4px) rotate(180deg);
      border-color: rgba(125, 211, 252, 0.85);
      box-shadow: 0 0 32px rgba(56, 189, 248, 0.60), inset 0 0 24px rgba(56, 189, 248, 0.35);
    }
    100% {
      transform: translate(-50%, -50%) translateZ(4px) rotate(360deg);
      border-color: rgba(56, 189, 248, 0.65);
      box-shadow: 0 0 24px rgba(56, 189, 248, 0.45), inset 0 0 20px rgba(14, 165, 233, 0.25);
    }
  }

  /* Concentric Ice-Blue Sonar Pulse Rings */
  .bsky-sonar-ring {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 250px;
    height: 210px;
    border-radius: 26px;
    border: 1.5px solid rgba(56, 189, 248, 0.50);
    box-shadow: 0 0 18px rgba(56, 189, 248, 0.35), inset 0 0 16px rgba(14, 165, 233, 0.15);
    animation: bsky-sonar-expand 4.2s cubic-bezier(0.2, 0.8, 0.2, 1) infinite;
    pointer-events: none;
    transform: translate(-50%, -50%);
  }

  .bsky-sonar-ring.ring-1 {
    animation-delay: 0s;
  }
  .bsky-sonar-ring.ring-2 {
    animation-delay: 2.1s;
  }

  .bsky-sonar-ring.stopped,
  .bsky-beacon-wrapper.stopped .bsky-sonar-ring {
    animation-play-state: paused !important;
  }

  @keyframes bsky-sonar-expand {
    0% {
      transform: translate(-50%, -50%) scale(0.85);
      opacity: 0.85;
      border-color: rgba(56, 189, 248, 0.70);
    }
    50% {
      opacity: 0.45;
    }
    100% {
      transform: translate(-50%, -50%) scale(1.6);
      opacity: 0;
      border-color: rgba(56, 189, 248, 0);
    }
  }

  /* Individual Orbit Track */
  .butterfly-orbit {
    position: absolute;
    top: 50%;
    left: 50%;
    transform-style: preserve-3d;
    pointer-events: none;
  }

  .butterfly-flight {
    position: relative;
    transform-style: preserve-3d;
    animation: butterfly-emerge 0.75s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }

  .bsky-beacon-wrapper.fading .butterfly-flight {
    animation: butterfly-scatter 2.2s cubic-bezier(0.4, 0, 0.2, 1) forwards !important;
  }

  @keyframes butterfly-emerge {
    from {
      opacity: 0;
      transform: scale(0.2) translateZ(0);
    }
    to {
      opacity: 1;
      transform: scale(1) translateZ(0);
    }
  }

  @keyframes butterfly-scatter {
    0% {
      opacity: 1;
      transform: scale(1) translateY(0) translateZ(0);
    }
    100% {
      opacity: 0;
      transform: scale(0.35) translateY(-220px) translateZ(90px) rotate(45deg);
    }
  }

  /* Stardust Phosphor Trail */
  .stardust-trail {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--glow-color, #38bdf8);
    transform: translate(-50%, -50%);
    filter: blur(1.5px) drop-shadow(0 0 6px var(--glow-color, #38bdf8));
    animation: stardust-twinkle 1.8s ease-in-out infinite alternate;
    opacity: 0.65;
  }

  @keyframes stardust-twinkle {
    0% { transform: translate(-50%, -50%) scale(0.6); opacity: 0.3; }
    100% { transform: translate(-50%, -50%) scale(1.6); opacity: 0.85; }
  }

  /* 3D Butterfly SVG & Wings */
  .butterfly-svg {
    display: block;
    transform-style: preserve-3d;
    filter: drop-shadow(0 0 7px rgba(56, 189, 248, 0.75)) drop-shadow(0 0 14px rgba(2, 132, 199, 0.45));
  }

  .wing-group {
    transform-style: preserve-3d;
  }

  .wing-group.left-wing {
    transform-origin: 0px 0px;
    animation: flap-left var(--flap-speed, 0.22s) ease-in-out infinite alternate;
  }

  .wing-group.right-wing {
    transform-origin: 0px 0px;
    animation: flap-right var(--flap-speed, 0.22s) ease-in-out infinite alternate;
  }

  @keyframes flap-left {
    0% {
      transform: rotateY(0deg) skewY(0deg) scaleX(1);
    }
    100% {
      transform: rotateY(70deg) skewY(-10deg) scaleX(0.24);
    }
  }

  @keyframes flap-right {
    0% {
      transform: rotateY(0deg) skewY(0deg) scaleX(1);
    }
    100% {
      transform: rotateY(-70deg) skewY(10deg) scaleX(0.24);
    }
  }

  /* =========================================================================
     6 Distinct Organic 3D Lissajous & Orbit Tracks Around the Card
     Cards are roughly 320px x 260px; orbits gracefully weave around bounds
     ========================================================================= */

  /* Track 0: Clockwise Elliptical Perimeter Sweep with High Dip */
  .track-0 {
    animation: orbit-track-0 var(--orbit-speed, 4.8s) linear infinite;
    animation-delay: var(--orbit-delay, 0s);
  }
  @keyframes orbit-track-0 {
    0% {
      transform: translate(calc(-50% + 185px), calc(-50% - 75px)) rotate(18deg) translateZ(28px);
    }
    25% {
      transform: translate(calc(-50% + 40px), calc(-50% - 155px)) rotate(-28deg) translateZ(8px);
    }
    50% {
      transform: translate(calc(-50% - 190px), calc(-50% - 40px)) rotate(-105deg) translateZ(-16px);
    }
    75% {
      transform: translate(calc(-50% - 70px), calc(-50% + 145px)) rotate(-190deg) translateZ(14px);
    }
    100% {
      transform: translate(calc(-50% + 185px), calc(-50% - 75px)) rotate(18deg) translateZ(28px);
    }
  }

  /* Track 1: Counter-Clockwise Infinity 8 Crossing Center Front & Back */
  .track-1 {
    animation: orbit-track-1 var(--orbit-speed, 5.6s) ease-in-out infinite;
    animation-delay: var(--orbit-delay, 0s);
  }
  @keyframes orbit-track-1 {
    0% {
      transform: translate(calc(-50% - 175px), calc(-50% + 80px)) rotate(-25deg) translateZ(22px);
    }
    25% {
      transform: translate(calc(-50% - 20px), calc(-50% - 30px)) rotate(15deg) translateZ(36px);
    }
    50% {
      transform: translate(calc(-50% + 175px), calc(-50% - 130px)) rotate(75deg) translateZ(-12px);
    }
    75% {
      transform: translate(calc(-50% + 30px), calc(-50% + 140px)) rotate(165deg) translateZ(10px);
    }
    100% {
      transform: translate(calc(-50% - 175px), calc(-50% + 80px)) rotate(-25deg) translateZ(22px);
    }
  }

  /* Track 2: Vertical Arc Over Card Header with Gentle Hover */
  .track-2 {
    animation: orbit-track-2 var(--orbit-speed, 5.1s) cubic-bezier(0.42, 0, 0.58, 1) infinite;
    animation-delay: var(--orbit-delay, 0s);
  }
  @keyframes orbit-track-2 {
    0% {
      transform: translate(calc(-50% + 110px), calc(-50% - 165px)) rotate(-45deg) translateZ(24px);
    }
    35% {
      transform: translate(calc(-50% - 130px), calc(-50% - 145px)) rotate(-135deg) translateZ(32px);
    }
    65% {
      transform: translate(calc(-50% - 170px), calc(-50% + 60px)) rotate(-220deg) translateZ(-10px);
    }
    85% {
      transform: translate(calc(-50% + 60px), calc(-50% + 130px)) rotate(-310deg) translateZ(12px);
    }
    100% {
      transform: translate(calc(-50% + 110px), calc(-50% - 165px)) rotate(-405deg) translateZ(24px);
    }
  }

  /* Track 3: Fast Diagonal Skim Across Top-Right and Bottom-Left */
  .track-3 {
    animation: orbit-track-3 var(--orbit-speed, 4.4s) ease-in-out infinite;
    animation-delay: var(--orbit-delay, 0s);
  }
  @keyframes orbit-track-3 {
    0% {
      transform: translate(calc(-50% + 195px), calc(-50% - 140px)) rotate(35deg) translateZ(34px);
    }
    40% {
      transform: translate(calc(-50% - 60px), calc(-50% + 40px)) rotate(115deg) translateZ(18px);
    }
    60% {
      transform: translate(calc(-50% - 180px), calc(-50% + 150px)) rotate(145deg) translateZ(-8px);
    }
    85% {
      transform: translate(calc(-50% + 40px), calc(-50% + 30px)) rotate(225deg) translateZ(12px);
    }
    100% {
      transform: translate(calc(-50% + 195px), calc(-50% - 140px)) rotate(395deg) translateZ(34px);
    }
  }

  /* Track 4: Hovering Spiral Orbit around Top-Left Card Corner */
  .track-4 {
    animation: orbit-track-4 var(--orbit-speed, 6.0s) cubic-bezier(0.36, 0, 0.66, 1) infinite;
    animation-delay: var(--orbit-delay, 0s);
  }
  @keyframes orbit-track-4 {
    0% {
      transform: translate(calc(-50% - 145px), calc(-50% - 150px)) rotate(-15deg) translateZ(30px);
    }
    25% {
      transform: translate(calc(-50% - 195px), calc(-50% - 70px)) rotate(-80deg) translateZ(15px);
    }
    50% {
      transform: translate(calc(-50% - 130px), calc(-50% + 20px)) rotate(-160deg) translateZ(-14px);
    }
    75% {
      transform: translate(calc(-50% - 70px), calc(-50% - 80px)) rotate(-250deg) translateZ(18px);
    }
    100% {
      transform: translate(calc(-50% - 145px), calc(-50% - 150px)) rotate(-375deg) translateZ(30px);
    }
  }

  /* Track 5: Low Gliding Sweep along Card Bottom */
  .track-5 {
    animation: orbit-track-5 var(--orbit-speed, 5.3s) ease-in-out infinite;
    animation-delay: var(--orbit-delay, 0s);
  }
  @keyframes orbit-track-5 {
    0% {
      transform: translate(calc(-50% - 160px), calc(-50% + 135px)) rotate(10deg) translateZ(16px);
    }
    30% {
      transform: translate(calc(-50% + 15px), calc(-50% + 155px)) rotate(-15deg) translateZ(26px);
    }
    60% {
      transform: translate(calc(-50% + 185px), calc(-50% + 115px)) rotate(-80deg) translateZ(8px);
    }
    85% {
      transform: translate(calc(-50% + 90px), calc(-50% + 40px)) rotate(-150deg) translateZ(-10px);
    }
    100% {
      transform: translate(calc(-50% - 160px), calc(-50% + 135px)) rotate(-350deg) translateZ(16px);
    }
  }
</style>
