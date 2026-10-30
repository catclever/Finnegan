<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CardAuraGlow Component
  Unified Border Beam Engine:
  - Strict outer-edge alignment matching Aura (inset: 1.5px, no outer gap)
  - Ink wash (飞白) exclusively features card-surface scanning light sweep
  - Bugu & Bluesky strictly maintain clean single-segment trailing beams with zero card bleed
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

  interface BeamConfig {
    type: 'aura' | 'bsky' | 'bugu' | 'ink';
    gradient: string;
    duration: string;
    delay: number;
    surfaceScan: boolean;
  }

  $: animDelay = -((card?.id ? card.id.charCodeAt(card.id.length - 1) : 0) % 5);
  $: hash = card?.id ? card.id.split('').reduce((a: number, b: string) => a + b.charCodeAt(0), 0) : 0;
  $: auraDelay = -((hash % 40) / 10);

  // 1. Single-segment Ice Blue tail for Bluesky (冰蓝单段彗尾: 290°~360°)
  const BSKY_GRADIENT = `conic-gradient(
    from 0deg,
    transparent 0deg,
    transparent 290deg,
    rgba(56, 189, 248, 0) 295deg,
    rgba(56, 189, 248, 0.25) 325deg,
    rgba(125, 211, 252, 0.75) 350deg,
    rgba(224, 242, 254, 1) 360deg
  )`;

  // 2. Single-segment Amber Gold tail for Bugu (琥珀金单段彗尾: 290°~360°)
  const BUGU_GRADIENT = `conic-gradient(
    from 0deg,
    transparent 0deg,
    transparent 290deg,
    rgba(245, 158, 11, 0) 295deg,
    rgba(245, 158, 11, 0.25) 325deg,
    rgba(251, 191, 36, 0.75) 350deg,
    rgba(254, 240, 138, 1) 360deg
  )`;

  // 3. Double-segment Ink Wash for recent VIP (水墨双段飞白: 135°~185° + 220°~355°)
  const INK_GRADIENT = `conic-gradient(
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
  )`;

  $: activeBeam = ((): BeamConfig | null => {
    if (hasAura && auraStage < 3) {
      return {
        type: 'aura',
        gradient: conicGradient,
        duration: auraStage === 2 ? '9s' : '4s',
        delay: auraDelay,
        surfaceScan: false
      };
    }
    if (isBskyEcho && !isFocused) {
      return {
        type: 'bsky',
        gradient: BSKY_GRADIENT,
        duration: '4.6s',
        delay: animDelay,
        surfaceScan: false
      };
    }
    if (isBuguEcho && !isFocused) {
      return {
        type: 'bugu',
        gradient: BUGU_GRADIENT,
        duration: '4.6s',
        delay: animDelay,
        surfaceScan: false
      };
    }
    if (isRecentVIP && !isFocused) {
      return {
        type: 'ink',
        gradient: INK_GRADIENT,
        duration: '11.4s',
        delay: animDelay,
        surfaceScan: true
      };
    }
    return null;
  })();
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
    class:type-bsky={activeBeam.type === 'bsky'}
    class:type-bugu={activeBeam.type === 'bugu'}
    class:type-aura={activeBeam.type === 'aura'}
    class:type-ink={activeBeam.type === 'ink'}
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

  /* 统一卡片内遮罩层：位于最外边界内侧 1.5px，对齐 Aura，留出 1.5px 纯净外缘光轨 */
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

  /* 1. 水墨飞白 (Ink) 专属：微透漫射，独享卡面水墨流光扫掠质感 */
  .beam-mask.surface-scan {
    background: rgba(255, 255, 255, 0.91);
  }

  .beam-mask.surface-scan.temporary {
    background: linear-gradient(135deg, rgba(226, 236, 248, 0.88) 0%, rgba(210, 222, 238, 0.80) 100%);
  }

  /* 2. 布谷、Bluesky 与 Aura：不透明底色彻底隔断，严禁光线漏入卡面，仅保留最外边缘单段彗尾 */
  .beam-mask.type-bsky {
    background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  }

  .beam-mask.type-bugu {
    background: linear-gradient(135deg, #eef5fc 0%, #e1ebf7 100%);
  }

  .beam-mask.type-aura {
    background: rgba(255, 255, 255, 0.98);
  }

  .beam-mask.type-aura.temporary {
    background: linear-gradient(135deg, #e2ecf8 0%, #d2def0 100%);
  }
</style>
