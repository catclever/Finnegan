<script lang="ts">
import { onMount } from 'svelte';
  import { spring } from 'svelte/motion';
  import { CaptureThought } from '../wailsjs/go/main/App.js';

  interface Card {
    id: number;
    title: string;
    content: string;
    x: number;
    y: number;
    rotation: number;
    z: number;
  }

  // Generate some random cards with collision spacing
  const generateCards = (count: number): Card[] => {
    const cards: Card[] = [];
    const minDistance = 160; // Minimum distance between card centers
    
    for (let i = 0; i < count; i++) {
      let x = 0, y = 0;
      let valid = false;
      
      // Try to find a spot that doesn't overlap too much with existing cards
      for (let attempt = 0; attempt < 100; attempt++) {
        x = (Math.random() - 0.5) * 1100; // Wider horizontal spread
        y = (Math.random() - 0.5) * 650;
        valid = true;
        
        for (const c of cards) {
          const dist = Math.sqrt((x - c.x) ** 2 + (y - c.y) ** 2);
          if (dist < minDistance) {
            valid = false;
            break;
          }
        }
        if (valid) break;
      }

      cards.push({
        id: i,
        title: `Thought ${i + 1}`,
        content: `Stream of consciousness piece number ${i + 1}.`,
        x,
        y,
        rotation: (Math.random() - 0.5) * 60,
        z: Math.random() * 100
      });
    }
    return cards;
  };

  let cards = generateCards(15).sort((a, b) => b.z - a.z);
  let focusedCardId: number | null = null;
  let hoveredCardId: number | null = null;
  let omniText = "";
  let capturePending = false;
  
  async function handleOmnibarKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      const text = omniText.trim();
      if (!text || capturePending) return;
      capturePending = true;
      try {
        await CaptureThought(text);
        omniText = "";
      } catch (err: any) {
        console.error(err);
        const oldText = text;
        omniText = "Error: " + err;
        setTimeout(() => { if (omniText.startsWith("Error:")) omniText = oldText }, 2000);
      } finally {
        capturePending = false;
      }
    }
  }

  // B+C: Mouse Parallax & Depth Pulsing State
  let time = 0;
  
  onMount(() => {
    let frame: number;
    const loop = () => {
      time = performance.now() * 0.001; // seconds
      frame = requestAnimationFrame(loop);
    };
    loop();
    return () => cancelAnimationFrame(frame);
  });

  // Global Tilt Spring (for the entire cards area)
  let globalTilt = spring({ rx: 0, ry: 0 }, { stiffness: 0.04, damping: 0.25 });

  function handleMouseMove(e: MouseEvent) {
    const mX = (e.clientX / window.innerWidth) * 2 - 1;
    const mY = (e.clientY / window.innerHeight) * 2 - 1;
    
    if (focusedCardId === null) {
      // Tilt up to 15 degrees based on mouse position
      // Mouse Right -> Rotate Y negative (right side comes forward)
      // Mouse Down -> Rotate X positive (bottom side comes forward)
      globalTilt.set({ rx: mY * 15, ry: -mX * 15 });
    } else {
      globalTilt.set({ rx: 0, ry: 0 });
    }
  }

  // Single spring animation store for individual cards
  let cardSprings = spring(cards.map(c => ({
    x: c.x,
    y: c.y,
    rotation: c.rotation,
    opacity: 0.95 - (c.z * 0.0055),
    scale: 1 - (c.z * 0.005),
    blur: 0,
    saturation: 1 - (c.z * 0.007),
    textOpacity: 1 - (c.z * 0.008),
    rotX: 0,
    rotY: 0,
    z3d: 0
  })), { stiffness: 0.05, damping: 0.3 });

  // React to focus changes and time (breathing)
  $: {
    const currentTime = time;
    const gTilt = $globalTilt; // React to global tilt to make individual cards follow slightly
    
    cardSprings.set(cards.map(c => {
      // Depth Pulsing (C)
      const pulse = Math.sin(currentTime * 1.2 + c.id * 2.1) * 8;
      
      if (focusedCardId === null) {
        // Scattering state
        const isHovered = hoveredCardId === c.id;
        // When hovered, pull it to the front (Z=0) so it lights up and clears up.
        const effectiveZ = isHovered ? 0 : Math.max(0, Math.min(100, c.z + pulse));
        
        // Individual 3D tracking
        // Hovered card slightly faces the user directly, ignoring global tilt for itself
        const individualRotX = isHovered ? 0 : gTilt.rx * 0.3; 
        const individualRotY = isHovered ? 0 : gTilt.ry * 0.3;
        
        // Let true 3D translateZ handle the magnification instead of fake scaling
        const targetScale = 1 - (effectiveZ * 0.005);

        return {
          x: c.x,
          y: c.y,
          rotation: isHovered ? 0 : c.rotation + (Math.sin(currentTime * 0.8 + c.id) * 2),
          opacity: isHovered ? 1 : 0.95 - (effectiveZ * 0.0055),
          scale: targetScale,
          blur: 0,
          saturation: 1 - (effectiveZ * 0.007),
          textOpacity: 1 - (effectiveZ * 0.008),
          rotX: individualRotX,
          rotY: individualRotY,
          z3d: isHovered ? 120 : 0
        };
      } else if (focusedCardId === c.id) {
        // Focused state (locked, no parallax/pulse)
        return {
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
        };
      } else {
        // Unfocused state (blurred and faded out)
        return {
          x: c.x * 1.5, // slightly push outwards
          y: c.y * 1.5,
          rotation: c.rotation,
          opacity: 0.1,
          scale: 0.8,
          blur: 10,
          saturation: 0.1,
          textOpacity: 0.1,
          rotX: 0,
          rotY: 0,
          z3d: 0
        };
      }
    }));
  }

  function toggleFocus(id: number) {
    if (focusedCardId === id) {
      focusedCardId = null;
    } else {
      focusedCardId = id;
    }
  }
  
  function handleBgClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      focusedCardId = null;
    }
  }
</script>

<svelte:window on:keydown={(e) => e.key === 'Escape' && (focusedCardId = null)} on:mousemove={handleMouseMove} />

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="container" on:click={handleBgClick}>
  <!-- Omnibar -->
  <div class="omnibar-container" class:faded={focusedCardId !== null}>
    <input 
      type="text" 
      class="omnibar" 
      placeholder="Capture a thought..." 
      bind:value={omniText}
      on:keydown={handleOmnibarKeydown}
      disabled={capturePending}
    />
  </div>

  <!-- Cards container -->
  <div class="cards-wrapper" on:click={handleBgClick}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="cards-area" style="transform: rotateX({$globalTilt.rx}deg) rotateY({$globalTilt.ry}deg); transform-style: preserve-3d;">
      {#each cards as card, i (card.id)}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div 
          class="card"
          class:focused={focusedCardId === card.id}
          on:click|stopPropagation={() => toggleFocus(card.id)}
          on:mouseenter={() => hoveredCardId = card.id}
          on:mouseleave={() => hoveredCardId = null}
          style="
            transform: translate(calc(-50% + {$cardSprings[i].x}px), calc(-50% + {$cardSprings[i].y}px)) 
                       translateZ({$cardSprings[i].z3d}px)
                       rotate({$cardSprings[i].rotation}deg) 
                       rotateX({$cardSprings[i].rotX}deg)
                       rotateY({$cardSprings[i].rotY}deg)
                       scale({$cardSprings[i].scale});
            opacity: {$cardSprings[i].opacity};
            filter: blur({$cardSprings[i].blur}px) saturate({$cardSprings[i].saturation});
            z-index: {focusedCardId === card.id ? 100 : (hoveredCardId === card.id ? 99 : Math.round(100 - card.z))};
            --text-opacity: {$cardSprings[i].textOpacity};
          "
        >
          <h3>{card.title}</h3>
          <p>{card.content}</p>
        </div>
      {/each}
    </div>
  </div>
</div>

<style>
  /* Reset background for translucent macOS window */
  :global(body), :global(html), :global(#app) {
    background: transparent !important;
    background-color: transparent !important;
    margin: 0;
    padding: 0;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
  }

  .container {
    width: 100vw;
    height: 100vh;
    position: relative;
    display: flex;
    justify-content: center;
    align-items: center;
    background: transparent;
  }

  .omnibar-container {
    position: absolute;
    z-index: 9999;
    transition: opacity 0.5s ease, filter 0.5s ease;
  }

  .omnibar-container.faded {
    opacity: 0;
    filter: blur(10px);
    pointer-events: none;
  }

  .omnibar {
    width: 500px;
    padding: 16px 28px;
    font-size: 1.25rem;
    border-radius: 40px;
    border: 2px solid rgba(255, 255, 255, 0.9);
    background: rgba(255, 255, 255, 0.85);
    color: #333;
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    outline: none;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05);
    transition: background 0.3s ease, border 0.3s ease, box-shadow 0.3s ease, transform 0.2s ease;
    text-align: center;
  }

  .omnibar:focus {
    background: rgba(255, 255, 255, 0.95);
    border: 2px solid rgba(255, 255, 255, 1);
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2), 0 0 0 1px rgba(0, 0, 0, 0.05);
    transform: scale(1.02);
  }

  .omnibar::placeholder {
    color: rgba(51, 51, 51, 0.6);
  }

  .cards-wrapper {
    position: absolute;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    perspective: 1200px;
    mask-image: radial-gradient(ellipse at center, black 40%, transparent 100%);
    -webkit-mask-image: radial-gradient(ellipse at center, black 40%, transparent 100%);
    pointer-events: none;
  }

  .cards-area {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    pointer-events: auto;
  }

  .card {
    position: absolute;
    top: 0;
    left: 0;
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
  }

  .card.focused {
    background: #ffffff;
    box-shadow: 0 24px 60px rgba(0, 0, 0, 0.3);
    cursor: default;
    border: none;
  }

  h3 {
    margin: 0 0 12px 0;
    font-size: 1.3rem;
  }

  p {
    margin: 0;
    font-size: 1rem;
    color: rgba(85, 85, 85, var(--text-opacity, 1));
    line-height: 1.4;
  }
</style>
