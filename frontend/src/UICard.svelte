<script lang="ts">
  import { spring } from 'svelte/motion';
  import { onMount, onDestroy } from 'svelte';

  export let card: any;
  export let time: number = 0;
  export let globalTilt: { rx: number, ry: number } = { rx: 0, ry: 0 };
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let onToggleFocus: () => void;
  export let onHover: () => void;
  export let onLeave: () => void;

  let animSpring = spring({
    x: card.x,
    y: card.y,
    rotation: card.rotation,
    opacity: 1,
    scale: 1,
    blur: 0,
    saturation: 1,
    textOpacity: 1,
    rotX: 0,
    rotY: 0,
    z3d: 0
  }, { stiffness: 0.06, damping: 0.35 });

  $: {
    const hashId = card.id.length > 0 ? card.id.charCodeAt(0) : 0;
    // Breathing pulse along Z-axis in pixels
    const pulseZ = Math.sin(time * 1.2 + hashId * 2.1) * 8;
    
    if (isFocused) {
      animSpring.set({
        x: 0,
        y: 0,
        rotation: 0,
        opacity: 1,
        scale: 1,
        blur: 0,
        saturation: 1,
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 160
      });
    } else if (isHovered) {
      // Pull towards center to stay in viewport
      const pullX = card.x > 260 ? 260 : (card.x < -260 ? -260 : card.x);
      const pullY = card.y > 170 ? 170 : (card.y < -170 ? -170 : card.y);
      animSpring.set({
        x: pullX,
        y: pullY,
        rotation: 0,
        opacity: 1,
        scale: 1.05,
        blur: 0,
        saturation: 1,
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 120
      });
    } else {
      // Background space: keep z3d >= 0 so WebKit hit-testing works flawlessly
      const effectiveZ = Math.max(0, Math.min(100, card.z + pulseZ));
      const dist = Math.sqrt(card.x * card.x + card.y * card.y);
      const edgeFade = Math.max(0, Math.min(1, 1 - (dist - 420) / 250));
      
      const anotherFocused = !isFocused && document.querySelector('.card.focused');
      
      if (anotherFocused) {
        animSpring.set({
          x: card.x * 1.4,
          y: card.y * 1.4,
          rotation: card.rotation,
          opacity: 0.08,
          scale: 0.7,
          blur: 10,
          saturation: 0.1,
          textOpacity: 0.1,
          rotX: 0,
          rotY: 0,
          z3d: 0
        });
      } else {
        // At the exact same depth/distance, card size is strictly uniform
        const uniformScale = Math.max(0.55, 1 - (card.z * 0.0045));
        animSpring.set({
          x: card.x,
          y: card.y,
          rotation: card.rotation + (Math.sin(time * 0.8 + hashId) * 1.5),
          opacity: (0.95 - (card.z * 0.005)) * edgeFade,
          scale: uniformScale,
          blur: card.z > 70 ? (card.z - 70) * 0.04 : 0,
          saturation: 1 - (card.z * 0.006),
          textOpacity: 1 - (card.z * 0.006),
          rotX: globalTilt.rx * 0.25,
          rotY: globalTilt.ry * 0.25,
          z3d: (100 - card.z) * 0.4
        });
      }
    }
  }

  $: hostname = card.sourceUrl ? (new URL(card.sourceUrl).hostname || 'Source') : '';
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div 
  class="card"
  class:focused={isFocused}
  class:hovered={isHovered && !isFocused}
  on:click={(e) => { e.stopPropagation(); onToggleFocus(); }}
  on:mouseenter={onHover}
  on:mouseleave={onLeave}
  style="
    transform: translate(calc(-50% + {$animSpring.x}px), calc(-50% + {$animSpring.y}px)) 
               translateZ({$animSpring.z3d}px)
               rotate({isFocused ? 0 : $animSpring.rotation}deg) 
               rotateX({isFocused ? 0 : $animSpring.rotX}deg) 
               rotateY({isFocused ? 0 : $animSpring.rotY}deg) 
               scale({$animSpring.scale});
    opacity: {$animSpring.opacity};
    filter: blur({$animSpring.blur}px) saturate({$animSpring.saturation});
    z-index: {isFocused ? 1000 : (isHovered ? 999 : Math.round(100 - card.z))};
    --text-opacity: {$animSpring.textOpacity};
  "
>
  {#if card.type === 'babel'}
    <div class="babel-header">
      {#if card.sourceUrl}
        <a href={card.sourceUrl} target="_blank" class="source-link" on:click|stopPropagation>
          <span class="domain-icon">🌐</span> {hostname}
        </a>
      {/if}
    </div>
    <div class="babel-content twinkle-container" class:focused={isFocused}>
      <div class="twinkle-text lang-trans">
        <p class="translated">{card.translatedText}</p>
      </div>
      <div class="twinkle-text lang-orig">
        <p class="original">{card.originalText}</p>
      </div>
    </div>
  {:else}
    <div class="thought-content">
      <p>{card.content}</p>
    </div>
  {/if}
</div>

<style>
  .card {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 200px;
    height: 190px;
    padding: 18px 20px;
    pointer-events: auto; /* Catch clicks and mouse events reliably */
    background: rgba(255, 255, 255, 0.7);
    border-radius: 16px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.04);
    color: rgba(51, 51, 51, var(--text-opacity, 1));
    cursor: pointer;
    transform-origin: center center;
    user-select: none;
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.5);
    overflow: hidden;
  }

  .babel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  
  .source-link {
    font-size: 0.75rem;
    color: rgba(0, 0, 0, 0.5);
    text-decoration: none;
    display: flex;
    align-items: center;
    gap: 5px;
    background: rgba(255, 255, 255, 0.6);
    padding: 3px 8px;
    border-radius: 20px;
    transition: all 0.2s;
  }
  
  .source-link:hover {
    background: rgba(255, 255, 255, 0.9);
    color: rgba(0, 0, 0, 0.8);
  }

  /* Twinkling effect */
  .twinkle-container {
    position: relative;
    width: 100%;
    height: 95px;
  }
  
  .twinkle-text {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
  
  .lang-trans {
    animation: twinkle-a 8s infinite cubic-bezier(0.4, 0, 0.2, 1);
  }
  
  .lang-orig {
    animation: twinkle-b 8s infinite cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* Stop animation completely when focused */
  .twinkle-container.focused .lang-trans,
  .twinkle-container.focused .lang-orig {
    animation: none !important;
    position: relative;
    opacity: 1;
    filter: none;
    height: auto;
    margin-bottom: 10px;
    display: block;
  }
  
  .twinkle-container.focused {
    height: auto;
    overflow-y: auto;
    max-height: 400px;
  }
  
  @keyframes twinkle-a {
    0%, 35% { opacity: 1; filter: blur(0px); z-index: 2; }
    45%, 85% { opacity: 0; filter: blur(8px); z-index: 1; }
    100% { opacity: 1; filter: blur(0px); z-index: 2; }
  }
  
  @keyframes twinkle-b {
    0%, 35% { opacity: 0; filter: blur(8px); z-index: 1; }
    45%, 85% { opacity: 1; filter: blur(0px); z-index: 2; }
    100% { opacity: 0; filter: blur(8px); z-index: 1; }
  }

  .card.hovered::after {
    content: '';
    position: absolute;
    top: -250px;
    left: -250px;
    right: -250px;
    bottom: -250px;
    z-index: -1;
  }

  .card.focused {
    width: 420px;
    height: 420px;
    max-width: 88vw;
    max-height: 88vh;
    padding: 36px;
    border-radius: 28px;
    background: rgba(255, 255, 255, 0.96);
    box-shadow: 0 40px 100px rgba(0, 0, 0, 0.22), 0 0 0 1px rgba(255, 255, 255, 0.9);
    cursor: default;
    overflow-y: auto;
  }

  .card.focused .thought-content {
    justify-content: center;
    height: 100%;
    overflow-y: auto;
  }

  .card.focused .thought-content p {
    font-size: 1.15rem;
    line-height: 1.75;
    display: block;
    -webkit-line-clamp: unset;
    overflow: visible;
  }

  .thought-content {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: center; /* 垂直居中 */
    height: 100%;
    width: 100%;
    overflow: hidden;
  }

  .thought-content p {
    margin: 0;
    font-size: 0.825rem;
    line-height: 1.55;
    color: rgba(40, 40, 40, var(--text-opacity, 0.95));
    font-weight: 450;
    letter-spacing: 0.01em;
    display: -webkit-box;
    -webkit-line-clamp: 5; /* 适度增加到5行以适应正方形高度 */
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    word-break: break-word;
    width: 100%;
  }

  p {
    margin: 0;
    font-size: 0.825rem;
    line-height: 1.5;
    white-space: pre-wrap;
  }
  
  .original {
    font-size: 0.775rem;
    color: rgba(0,0,0,0.6);
    font-style: italic;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  
  .translated {
    font-size: 0.825rem;
    font-weight: 500;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  
  /* Global sleek scrollbar for cards and other elements */
  ::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }
  ::-webkit-scrollbar-track {
    background: transparent;
    margin: 8px 0; 
  }
  ::-webkit-scrollbar-thumb {
    background: rgba(0, 0, 0, 0.15);
    border-radius: 10px;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: rgba(0, 0, 0, 0.3);
  }
</style>
