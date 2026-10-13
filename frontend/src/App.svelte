<script lang="ts">
  import { onMount } from 'svelte';
  import UICard from './UICard.svelte';
  import { spring } from 'svelte/motion';
  import { CaptureThought, GetTempCards, LoadLocalCards } from '../wailsjs/go/main/App.js';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';

  interface CosmicCard {
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
    
    // Lifecycle & Decay properties
    spawnTime: number;         // When entered active stage (ms timestamp)
    decayDuration: number;     // Half-life in stage (ms)
    isFresh: boolean;          // True for new arrivals (100% optical clarity)
    isVIP: boolean;            // Latest manual user input flag
    isFadingOut?: boolean;     // True when departing stage
    origin: 'manual' | 'bugu' | 'local';
    macroExpiresAt?: number;   // In-memory 24-hr TTL
  }

  // --- Cosmic Stage Configuration ---
  const BASE_CAPACITY = 10;                     // Base single-screen card count (8~12)
  const MACRO_TTL_MS = 24 * 3600 * 1000;        // 24 hours macro TTL for in-memory temporary cards
  const BASE_STAGE_DECAY_MS = 3.5 * 3600 * 1000;// 3.5 hours base stage half-life

  let memoryPool: CosmicCard[] = [];            // In-memory transient cards (Bugu / Radar)
  let localVaultPool: CosmicCard[] = [];        // Local Markdown cards (Scattering + Waving)
  let cards: CosmicCard[] = [];                 // Currently rendered active cards on stage
  let latestManualCardId: string | null = null; // Single exclusive VIP thought

  let focusedCardId: string | null = null;
  let hoveredCardId: string | null = null;
  let omniText = "";
  let capturePending = false;
  let time = 0;

  // --- Gravitational Tide (Y-Axis Expansion) ---
  let scrollExpansion = spring(0, { stiffness: 0.05, damping: 0.3 }); // 0 = single screen, 1 = expanded
  let tideTimer: any = null;

  $: targetCapacity = Math.round(BASE_CAPACITY * (1 + $scrollExpansion * 0.6));

  function resetTideTimer() {
    if (tideTimer) clearTimeout(tideTimer);
    tideTimer = setTimeout(() => {
      // 45s inactivity -> Smooth gravitational collapse
      scrollExpansion.set(0);
      setTimeout(() => {
        // Gently dismiss cards that ended up far below in expanded space
        cards.forEach(c => {
          if (c.y > 400 && !c.isVIP && !c.isFadingOut) {
            c.isFadingOut = true;
            setTimeout(() => {
              cards = cards.filter(item => item.id !== c.id);
            }, 2500);
          }
        });
      }, 1000);
    }, 45000);
  }

  function handleWheel(e: WheelEvent) {
    if (focusedCardId !== null) return; // Allow reading inside focused card

    if (e.deltaY > 0) {
      // Scroll down -> extend Y space
      const current = $scrollExpansion;
      scrollExpansion.set(Math.min(1.0, current + e.deltaY * 0.0012));
      resetTideTimer();
    } else if (e.deltaY < 0) {
      // Scroll up -> retract
      const current = $scrollExpansion;
      scrollExpansion.set(Math.max(0, current + e.deltaY * 0.0012));
      resetTideTimer();
    }
  }

  // --- Dynamic Half-Life Calculation ---
  function calculateDecayDuration(origin: 'manual' | 'bugu' | 'local', z: number, isVIP: boolean): number {
    if (origin === 'manual' && isVIP) {
      return 12 * 3600 * 1000; // VIP protected half-life
    }

    let typeBase = BASE_STAGE_DECAY_MS;
    if (origin === 'local') {
      typeBase = 2 * 3600 * 1000; // Local memories stay ~2 hours
    }

    // Depth factor (closer = faster decay, farther = slower decay)
    const depthFactor = 0.6 + (z / 100) * 0.9;

    // Queue pressure: if memory pool has unshown cards waiting
    const activeIds = new Set(cards.map(c => c.id));
    const restingCount = memoryPool.filter(mc => !activeIds.has(mc.id)).length;
    const pressureFactor = Math.max(0.4, 1 / (1 + restingCount * 0.12));

    // Random jitter ±20%
    const jitter = 0.8 + Math.random() * 0.4;

    return Math.round(typeBase * depthFactor * pressureFactor * jitter);
  }

  function getNextCandidate(): CosmicCard | null {
    const activeIds = new Set(cards.map(c => c.id));
    const now = Date.now();

    // Priority 1: In-memory temporary cards (Bugu) within 24h TTL
    const validMemory = memoryPool.filter(c => !activeIds.has(c.id) && (!c.macroExpiresAt || c.macroExpiresAt > now));
    if (validMemory.length > 0) {
      const idx = Math.floor(Math.random() * validMemory.length);
      return validMemory[idx];
    }

    // Priority 2: Local persistent vault cards (Scattering / Waving)
    const validLocal = localVaultPool.filter(c => !activeIds.has(c.id));
    if (validLocal.length > 0) {
      const idx = Math.floor(Math.random() * validLocal.length);
      return validLocal[idx];
    }

    return null;
  }

  function stageCandidate(candidate: CosmicCard, isNewlyCreated = false): CosmicCard {
    const z = Math.random() * 100;
    const expansion = $scrollExpansion;
    const yRange = 600 + expansion * 800;
    const yCenter = expansion * 300;
    const y = (Math.random() - 0.5) * yRange + yCenter;
    const x = (Math.random() - 0.5) * 1100;
    const rotation = (Math.random() - 0.5) * 50;

    const spawnTime = Date.now();
    const decayDuration = calculateDecayDuration(candidate.origin, z, candidate.isVIP);

    return {
      ...candidate,
      x,
      y,
      rotation,
      z,
      spawnTime,
      decayDuration,
      isFresh: isNewlyCreated,
      isFadingOut: false
    };
  }

  async function handleOmnibarKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      const text = omniText.trim();
      if (!text || capturePending) return;
      capturePending = true;
      try {
        await CaptureThought(text);
        omniText = "";
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

  onMount(() => {
    // 1. Initial Load of Local & In-Memory Cards
    const loadCards = async () => {
      const localCards = await LoadLocalCards();
      const tempCards = await GetTempCards();
      
      if (tempCards && tempCards.length > 0) {
        memoryPool = tempCards.map(c => ({
          id: c.id || c.ID,
          type: 'babel',
          title: c.title || c.Title || 'Babel',
          content: c.content || c.Content || '',
          originalText: c.originalText || c.OriginalText || '',
          translatedText: c.translatedText || c.TranslatedText || '',
          sourceUrl: c.sourceUrl || c.SourceURL || '',
          x: 0, y: 0, rotation: 0, z: 50,
          spawnTime: 0,
          decayDuration: 0,
          isFresh: false,
          isVIP: false,
          origin: 'bugu',
          macroExpiresAt: Date.now() + MACRO_TTL_MS
        }));
      }

      if (localCards && localCards.length > 0) {
        localVaultPool = localCards.map(c => ({
          id: c.id || c.ID,
          type: c.type || c.Type || 'thought',
          title: c.title || c.Title || 'Thought',
          content: c.content || c.Content || '',
          originalText: c.originalText || c.OriginalText || '',
          translatedText: c.translatedText || c.TranslatedText || '',
          sourceUrl: c.sourceUrl || c.SourceURL || '',
          x: 0, y: 0, rotation: 0, z: 50,
          spawnTime: 0,
          decayDuration: 0,
          isFresh: false,
          isVIP: false,
          origin: 'local'
        }));
      }

      // Populate initial stage up to target capacity
      let initialStage: CosmicCard[] = [];
      while (initialStage.length < BASE_CAPACITY) {
        const candidate = getNextCandidate();
        if (!candidate) break;
        const staged = stageCandidate(candidate, false);
        initialStage.push(staged);
      }
      cards = initialStage.sort((a, b) => b.z - a.z);
    };
    loadCards();

    // 2. Real-time Event: User Thought Captured
    EventsOn("thought:captured", (data) => {
      const cardId = data.DocID || 'thought_' + Date.now();
      
      // Demote previous VIP
      if (latestManualCardId) {
        const prevVIP = cards.find(c => c.id === latestManualCardId);
        if (prevVIP) {
          prevVIP.isVIP = false;
          prevVIP.decayDuration = BASE_STAGE_DECAY_MS;
        }
      }
      latestManualCardId = cardId;

      const newCard: CosmicCard = {
        id: cardId,
        type: 'thought',
        title: 'Thought',
        content: data.Text,
        x: (Math.random() - 0.5) * 600,
        y: (Math.random() - 0.5) * 300,
        rotation: (Math.random() - 0.5) * 20,
        z: Math.random() * 40 + 10,
        spawnTime: Date.now(),
        decayDuration: 12 * 3600 * 1000,
        isFresh: true,
        isVIP: true,
        origin: 'manual'
      };

      // Add to persistent vault cache
      localVaultPool = [newCard, ...localVaultPool];

      // If active stage is full, evict the card with the shortest remaining lifetime
      if (cards.filter(c => !c.isFadingOut).length >= targetCapacity) {
        const evictable = cards
          .filter(c => !c.isVIP && !c.isFadingOut)
          .sort((a, b) => (a.spawnTime + a.decayDuration) - (b.spawnTime + b.decayDuration));
        if (evictable.length > 0) {
          evictable[0].isFadingOut = true;
          setTimeout(() => {
            cards = cards.filter(c => c.id !== evictable[0].id);
          }, 2500);
        }
      }

      cards = [...cards, newCard];
    });

    // 3. Real-time Event: Bugu Babel Card Spawned
    EventsOn("bugu:card_spawned", (c: any) => {
      const cardId = c.id || c.ID || 'babel_' + Date.now();
      const newCard: CosmicCard = {
        id: cardId,
        type: 'babel',
        title: c.title || c.Title || 'Babel',
        content: c.content || c.Content || '',
        originalText: c.originalText || c.OriginalText || '',
        translatedText: c.translatedText || c.TranslatedText || '',
        sourceUrl: c.sourceUrl || c.SourceURL || '',
        x: 0, y: 0, rotation: 0, z: 50,
        spawnTime: 0,
        decayDuration: 0,
        isFresh: true,
        isVIP: false,
        origin: 'bugu',
        macroExpiresAt: Date.now() + MACRO_TTL_MS
      };

      // Push into in-memory pool
      memoryPool = [newCard, ...memoryPool.filter(m => m.id !== cardId)];

      // If stage has vacancy, immediately stage it
      if (cards.filter(c => !c.isFadingOut).length < targetCapacity) {
        const staged = stageCandidate(newCard, true);
        cards = [...cards, staged];
      }
    });

    // 4. Metabolic Lifecycle Heartbeat (Runs every 2s)
    const metabolicInterval = setInterval(() => {
      const now = Date.now();

      // Clean expired in-memory cards (24-hr TTL)
      memoryPool = memoryPool.filter(c => !c.macroExpiresAt || c.macroExpiresAt > now);

      // Check freshness expiration (first 8 minutes)
      cards.forEach(c => {
        if (c.isFresh && now - c.spawnTime > 8 * 60 * 1000) {
          c.isFresh = false;
        }
      });

      // Decay expired stage cards
      let needsUpdate = false;
      cards.forEach(c => {
        if (!c.isVIP && !c.isFadingOut && (now - c.spawnTime > c.decayDuration)) {
          c.isFadingOut = true;
          needsUpdate = true;
          setTimeout(() => {
            cards = cards.filter(card => card.id !== c.id);
          }, 2500);
        }
      });

      // Replenish stage vacancies
      const activeCount = cards.filter(c => !c.isFadingOut).length;
      if (activeCount < targetCapacity) {
        const candidate = getNextCandidate();
        if (candidate) {
          const staged = stageCandidate(candidate, false);
          cards = [...cards, staged];
          needsUpdate = true;
        }
      }

      if (needsUpdate) {
        cards = [...cards];
      }
    }, 2000);

    // 5. 3D Render Animation Loop
    let frame: number;
    const loop = () => {
      time = performance.now() * 0.001;
      frame = requestAnimationFrame(loop);
    };
    loop();

    return () => {
      cancelAnimationFrame(frame);
      clearInterval(metabolicInterval);
      if (tideTimer) clearTimeout(tideTimer);
    };
  });

  // Global Tilt Spring
  let globalTilt = spring({ rx: 0, ry: 0 }, { stiffness: 0.04, damping: 0.25 });

  function handleMouseMove(e: MouseEvent) {
    resetTideTimer();
    const mX = (e.clientX / window.innerWidth) * 2 - 1;
    const mY = (e.clientY / window.innerHeight) * 2 - 1;
    
    if (focusedCardId === null) {
      globalTilt.set({ rx: mY * 15, ry: -mX * 15 });
    } else {
      globalTilt.set({ rx: 0, ry: 0 });
    }
  }

  function toggleFocus(id: string) {
    resetTideTimer();
    if (focusedCardId === id) {
      focusedCardId = null;
    } else {
      focusedCardId = id;
      globalTilt.set({ rx: 0, ry: 0 }, { hard: true });
    }
  }

  function handleBgClick(e: MouseEvent) {
    resetTideTimer();
    if (e.target === e.currentTarget) {
      focusedCardId = null;
    }
  }
</script>

<svelte:window 
  on:keydown={(e) => e.key === 'Escape' && (focusedCardId = null)} 
  on:mousemove={handleMouseMove} 
  on:wheel={handleWheel}
/>

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
  <div class="cards-wrapper" class:elevated={hoveredCardId !== null || focusedCardId !== null} on:click={handleBgClick}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div 
      class="scene" 
      style="
        transform: rotateX({$globalTilt.rx}deg) rotateY({$globalTilt.ry}deg) translateY({-$scrollExpansion * 300}px); 
        transform-style: preserve-3d;
      "
    >
      {#each cards as card (card.id)}
        <UICard 
          {card}
          {time}
          globalTilt={$globalTilt}
          isFocused={focusedCardId === card.id}
          isHovered={hoveredCardId === card.id}
          onToggleFocus={() => toggleFocus(card.id)}
          onHover={() => { hoveredCardId = card.id; resetTideTimer(); }}
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
    z-index: 10;
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
    z-index: 5; /* Background cards sit behind omnibar */
  }

  .cards-wrapper.elevated {
    z-index: 20; /* Hovered or focused cards jump above omnibar */
  }

  .scene {
    pointer-events: none; /* Pass through empty space to background */
    width: 100%;
    height: 100%;
    position: relative;
    transform-style: preserve-3d;
    transition: transform 0.1s ease-out;
  }
</style>

