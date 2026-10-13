<script lang="ts">
import { onMount } from 'svelte';
import UICard from './UICard.svelte';
  import { spring } from 'svelte/motion';
  import { CaptureThought, GetTempCards, LoadLocalCards } from '../wailsjs/go/main/App.js';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';

  interface Card {
    id: string;
    type?: string;
    title: string;
    content: string;
    originalText?: string;
    translatedText?: string;
    sourceUrl?: string;
    x: number;
    y: number;
    rotation: number;
    z: number;
  }

  let cards: Card[] = [];
  let focusedCardId: string | null = null;
  let hoveredCardId: string | null = null;
  let omniText = "";
  let capturePending = false;
  
  async function handleOmnibarKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault(); // prevent new line
      const text = omniText.trim();
      if (!text || capturePending) return;
      capturePending = true;
      try {
        await CaptureThought(text);
        omniText = "";
        // Reset height
        if (e.target instanceof HTMLTextAreaElement) {
          e.target.style.height = 'auto';
        }
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

  function handleInput(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    target.style.height = 'auto';
    target.style.height = target.scrollHeight + 'px';
  }

  // B+C: Mouse Parallax & Depth Pulsing State
  let time = 0;
  
  onMount(() => {
    const loadCards = async () => {
      const localCards = await LoadLocalCards();
      const tempCards = await GetTempCards();
      
      let allCards = [];
      if (localCards && localCards.length > 0) {
        allCards = [...allCards, ...localCards];
      }
      if (tempCards && tempCards.length > 0) {
        allCards = [...allCards, ...tempCards];
      }
      
      if (allCards.length > 0) {
        console.log("Loaded allCards:", allCards);
        const mapped = allCards.map(c => mapToUICard(c));
        console.log("Mapped cards:", mapped);
        cards = mapped.sort((a, b) => b.z - a.z);
      } else {
        console.log("No cards found in local or temp storage.");
      }
    };
    loadCards();

    EventsOn("thought:captured", (data) => {
      const newCard = mapToUICard({
        id: data.DocID,
        type: 'thought',
        title: 'Thought',
        content: data.Text
      });
      cards = [...cards, newCard].sort((a, b) => b.z - a.z);
    });

    EventsOn("bugu:card_spawned", (c) => {
      const newCard = mapToUICard(c);
      cards = [...cards, newCard].sort((a, b) => b.z - a.z);
    });

    let frame: number;
    const loop = () => {
      time = performance.now() * 0.001; // seconds
      frame = requestAnimationFrame(loop);
    };
    loop();
    return () => cancelAnimationFrame(frame);
  });

  function mapToUICard(c: any): Card {
    return {
      id: c.id || c.ID,
      type: c.type || c.Type,
      title: c.title || c.Title || 'Card',
      content: c.content || c.Content || '',
      originalText: c.originalText || c.OriginalText || '',
      translatedText: c.translatedText || c.TranslatedText || '',
      sourceUrl: c.sourceUrl || c.SourceURL || '',
      x: (Math.random() - 0.5) * 1100,
      y: (Math.random() - 0.5) * 650,
      rotation: (Math.random() - 0.5) * 60,
      z: Math.random() * 100
    };
  }

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
  

  // React to focus changes and time (breathing)
  
  function toggleFocus(id: string) {
    if (focusedCardId === id) {
      focusedCardId = null;
    } else {
      focusedCardId = id;
    }
  }

  function toggleLanguage(cardId: string) {
    const idx = cards.findIndex(c => c.id === cardId);
    if (idx !== -1) {
      cards[idx].showingOriginal = !cards[idx].showingOriginal;
      cards = [...cards];
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
    <textarea 
      class="omnibar" 
      placeholder="Capture a thought..." 
      bind:value={omniText}
      on:keydown={handleOmnibarKeydown}
      on:input={handleInput}
      disabled={capturePending}
      rows="1"
    ></textarea>
  </div>

  <!-- Cards container -->
  <div class="cards-wrapper" on:click={handleBgClick}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="scene" style="transform: rotateX({$globalTilt.rx}deg) rotateY({$globalTilt.ry}deg); transform-style: preserve-3d;">
      {#each cards as card (card.id)}
        <UICard 
          {card}
          {time}
          globalTilt={$globalTilt}
          isFocused={focusedCardId === card.id}
          isHovered={hoveredCardId === card.id}
          onToggleFocus={() => toggleFocus(card.id)}
          onHover={() => hoveredCardId = card.id}
          onLeave={() => hoveredCardId = null}
        />
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
    border-radius: 32px;
    border: 2px solid rgba(255, 255, 255, 0.9);
    background: rgba(255, 255, 255, 0.85);
    color: #333;
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    outline: none;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05);
    transition: background 0.3s ease, border 0.3s ease, box-shadow 0.3s ease, transform 0.2s ease;
    text-align: left;
    resize: none;
    min-height: 62px;
    max-height: 250px;
    overflow-y: auto;
    line-height: 1.5;
    box-sizing: border-box;
    font-family: inherit;
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

  /* Hide scrollbar for omnibar entirely */
  .omnibar::-webkit-scrollbar {
    display: none;
  }
  .omnibar {
    -ms-overflow-style: none;  /* IE and Edge */
    scrollbar-width: none;  /* Firefox */
  }

  .cards-wrapper {
    position: absolute;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    perspective: 1000px;
    pointer-events: none; /* Let clicks pass through empty space */
  }

  .scene {
    pointer-events: auto; /* Catch clicks on cards */
  }

  .scene {
    width: 100%;
    height: 100%;
    position: relative;
    transform-style: preserve-3d;
    transition: transform 0.1s ease-out;
  }
</style>

