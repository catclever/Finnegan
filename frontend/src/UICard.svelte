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
  }, { stiffness: 0.05, damping: 0.3 });

  $: {
    const hashId = card.id.length > 0 ? card.id.charCodeAt(0) : 0;
    const pulse = Math.sin(time * 1.2 + hashId * 2.1) * 8;
    
    if (isFocused) {
      animSpring.set({
        x: 0,
        y: 0,
        rotation: 0,
        opacity: 1,
        scale: 1.5,
        blur: 0,
        saturation: 1,
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 200
      });
    } else if (isHovered) {
      // Pull heavily towards center to avoid clipping at edges
      const pullX = card.x > 300 ? 300 : (card.x < -300 ? -300 : card.x);
      const pullY = card.y > 200 ? 200 : (card.y < -200 ? -200 : card.y);
      animSpring.set({
        x: pullX,
        y: pullY,
        rotation: 0,
        opacity: 1,
        scale: 1.1,
        blur: 0,
        saturation: 1,
        textOpacity: 1,
        rotX: 0,
        rotY: 0,
        z3d: 120
      });
    } else {
      // Background state
      const effectiveZ = Math.max(0, Math.min(100, card.z + pulse));
      // Calculate distance from center to fade out edges
      const dist = Math.sqrt(card.x * card.x + card.y * card.y);
      const edgeFade = Math.max(0, Math.min(1, 1 - (dist - 400) / 250));
      
      // Determine if another card is focused
      const anotherFocused = !isFocused && document.querySelector('.card.focused');
      
      if (anotherFocused) {
        animSpring.set({
          x: card.x * 1.5,
          y: card.y * 1.5,
          rotation: card.rotation,
          opacity: 0.1,
          scale: 0.8,
          blur: 10,
          saturation: 0.1,
          textOpacity: 0.1,
          rotX: 0,
          rotY: 0,
          z3d: 0
        });
      } else {
        animSpring.set({
          x: card.x,
          y: card.y,
          rotation: card.rotation + (Math.sin(time * 0.8 + hashId) * 2),
          opacity: (0.95 - (effectiveZ * 0.0055)) * edgeFade,
          scale: 1 - (effectiveZ * 0.005),
          blur: 0,
          saturation: 1 - (effectiveZ * 0.007),
          textOpacity: 1 - (effectiveZ * 0.008),
          rotX: globalTilt.rx * 0.3,
          rotY: globalTilt.ry * 0.3,
          z3d: effectiveZ
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
               rotate({$animSpring.rotation}deg) 
               rotateX({$animSpring.rotX}deg)
               rotateY({$animSpring.rotY}deg)
               scale({$animSpring.scale});
    opacity: {$animSpring.opacity};
    filter: blur({$animSpring.blur}px) saturate({$animSpring.saturation});
    z-index: {isFocused ? 100 : (isHovered ? 99 : Math.round(100 - card.z))};
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
    width: 260px;
    height: 180px;
    padding: 24px;
    background: rgba(255, 255, 255, 0.7);
    border-radius: 16px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.05);
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
    margin-bottom: 12px;
  }
  
  .source-link {
    font-size: 0.8rem;
    color: rgba(0, 0, 0, 0.5);
    text-decoration: none;
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(255, 255, 255, 0.6);
    padding: 4px 10px;
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
    height: 100px;
  }
  
  .twinkle-text {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
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
    width: 500px;
    height: auto;
    max-height: 600px;
    background: rgba(255, 255, 255, 0.95);
    box-shadow: 0 40px 100px rgba(0, 0, 0, 0.3), 0 0 0 1px rgba(255, 255, 255, 1);
    cursor: default;
    overflow-y: auto;
  }

  .thought-content {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    width: 100%;
  }

  .thought-content p {
    margin: 0;
    font-size: 1.15rem;
    line-height: 1.6;
    white-space: pre-wrap;
    color: rgba(0, 0, 0, 0.85);
    font-weight: 500;
  }

  p {
    margin: 0;
    font-size: 0.95rem;
    line-height: 1.5;
    white-space: pre-wrap;
  }
  
  .original {
    font-size: 0.85rem;
    color: rgba(0,0,0,0.6);
    font-style: italic;
  }
  
  .translated {
    font-size: 0.95rem;
    font-weight: 500;
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
