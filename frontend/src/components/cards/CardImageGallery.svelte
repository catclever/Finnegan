<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CardImageGallery Subcomponent
  Task: thought_images_focus_gallery_030
  
  Focused Mode Image Gallery (Option A: Invariant Focus Natural Scroll & Horizontal Swipe).
  Features:
  - Width is 100% constrained to container
  - Height is unconstrained (height: auto) so screenshots render in natural aspect ratio
    with zero cropping
  - For multiple images: horizontal scroll-snap swipe gallery (flex, overflow-x: auto,
    scroll-snap-type: x mandatory)
  - Desktop prev/next navigation buttons with chevron icons & hover states
  - Clickable indicator dots at bottom center tracking active slide
  ======================================================================================
-->
<script lang="ts">
  export let images: string[] = [];

  let trackEl: HTMLElement | null = null;
  let currentIndex: number = 0;

  $: if (currentIndex >= images.length) {
    currentIndex = Math.max(0, images.length - 1);
  }

  function handleScroll() {
    if (!trackEl) return;
    const width = trackEl.clientWidth;
    if (width <= 0) return;
    const scrollLeft = trackEl.scrollLeft;
    const newIndex = Math.round(scrollLeft / width);
    if (newIndex !== currentIndex && newIndex >= 0 && newIndex < images.length) {
      currentIndex = newIndex;
    }
  }

  function scrollToSlide(index: number) {
    if (!trackEl) return;
    const targetIndex = Math.max(0, Math.min(index, images.length - 1));
    currentIndex = targetIndex;
    const width = trackEl.clientWidth;
    trackEl.scrollTo({
      left: targetIndex * width,
      behavior: 'smooth'
    });
  }

  function prevSlide() {
    if (currentIndex > 0) {
      scrollToSlide(currentIndex - 1);
    }
  }

  function nextSlide() {
    if (currentIndex < images.length - 1) {
      scrollToSlide(currentIndex + 1);
    }
  }

  function handleImageError(e: Event) {
    const el = e.currentTarget as HTMLElement | null;
    if (el) el.style.display = 'none';
  }
</script>

{#if images && images.length > 0}
  <div class="card-image-gallery" class:multi-images={images.length > 1}>
    {#if images.length === 1}
      <!-- Single image mode: natural aspect ratio, 100% width, zero cropping -->
      <div class="single-image-wrapper">
        <img
          src={images[0]}
          alt="Thought attachment"
          class="gallery-image"
          on:error={handleImageError}
        />
      </div>
    {:else}
      <!-- Multi-image mode: horizontal scroll-snap swipe gallery -->
      <div class="gallery-wrapper">
        <div
          class="gallery-track"
          bind:this={trackEl}
          on:scroll={handleScroll}
        >
          {#each images as img, i (img + i)}
            <div class="gallery-slide" role="group" aria-label="Image {i + 1} of {images.length}">
              <img
                src={img}
                alt="Thought attachment {i + 1}"
                class="gallery-image"
                on:error={handleImageError}
              />
            </div>
          {/each}
        </div>

        <!-- Desktop Prev Button -->
        <button
          type="button"
          class="gallery-nav-btn prev-btn"
          class:hidden={currentIndex === 0}
          on:click|stopPropagation={prevSlide}
          aria-label="Previous image"
        >
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="15 18 9 12 15 6" />
          </svg>
        </button>

        <!-- Desktop Next Button -->
        <button
          type="button"
          class="gallery-nav-btn next-btn"
          class:hidden={currentIndex === images.length - 1}
          on:click|stopPropagation={nextSlide}
          aria-label="Next image"
        >
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="9 18 15 12 9 6" />
          </svg>
        </button>

        <!-- Indicator dots -->
        <div class="indicator-dots" role="tablist" aria-label="Image gallery indicators">
          {#each images as _, i}
            <button
              type="button"
              class="indicator-dot"
              class:active={currentIndex === i}
              on:click|stopPropagation={() => scrollToSlide(i)}
              aria-label="View image {i + 1} of {images.length}"
            />
          {/each}
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  .card-image-gallery {
    width: 100%;
    max-width: 100%;
    box-sizing: border-box;
    margin-bottom: 14px;
    flex-shrink: 0;
    position: relative;
    user-select: none;
  }

  .single-image-wrapper {
    width: 100%;
    height: auto;
    display: flex;
    justify-content: center;
    align-items: flex-start;
  }

  .gallery-wrapper {
    position: relative;
    width: 100%;
    height: auto;
  }

  .gallery-track {
    display: flex;
    flex-direction: row;
    align-items: flex-start;
    width: 100%;
    height: auto;
    overflow-x: auto;
    overflow-y: hidden;
    scroll-snap-type: x mandatory;
    scroll-behavior: smooth;
    -webkit-overflow-scrolling: touch;
    overscroll-behavior-x: contain;
    scrollbar-width: none;
    -ms-overflow-style: none;
    border-radius: 14px;
  }

  .gallery-track::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  .gallery-slide {
    flex: 0 0 100%;
    width: 100%;
    min-width: 100%;
    max-width: 100%;
    box-sizing: border-box;
    scroll-snap-align: start;
    scroll-snap-stop: always;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    height: auto;
  }

  .gallery-image {
    width: 100%;
    height: auto;
    max-width: 100%;
    display: block;
    object-fit: contain;
    border-radius: 14px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08), 0 0 0 1px rgba(0, 0, 0, 0.04);
    background: rgba(0, 0, 0, 0.02);
    user-select: none;
    -webkit-user-drag: none;
  }

  /* Desktop Prev/Next Navigation Buttons */
  .gallery-nav-btn {
    position: absolute;
    top: clamp(40px, 50%, 180px);
    transform: translateY(-50%);
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.9);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border: 1px solid rgba(0, 0, 0, 0.08);
    box-shadow: 0 3px 12px rgba(0, 0, 0, 0.15);
    color: rgba(30, 30, 30, 0.88);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10;
    transition: opacity 0.22s ease, transform 0.22s ease, background-color 0.2s ease, box-shadow 0.2s ease;
    padding: 0;
  }

  .gallery-nav-btn.prev-btn {
    left: 10px;
  }

  .gallery-nav-btn.next-btn {
    right: 10px;
  }

  .gallery-nav-btn:hover:not(.hidden) {
    background: #ffffff;
    color: #000000;
    transform: translateY(-50%) scale(1.1);
    box-shadow: 0 5px 16px rgba(0, 0, 0, 0.22);
  }

  .gallery-nav-btn:active:not(.hidden) {
    transform: translateY(-50%) scale(0.95);
  }

  .gallery-nav-btn.hidden {
    opacity: 0;
    pointer-events: none;
    transform: translateY(-50%) scale(0.8);
  }

  /* Indicator Dots */
  .indicator-dots {
    position: absolute;
    bottom: 10px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    background: rgba(0, 0, 0, 0.35);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    border-radius: 12px;
    z-index: 10;
    pointer-events: auto;
  }

  .indicator-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.55);
    border: none;
    padding: 0;
    cursor: pointer;
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .indicator-dot:hover {
    background: rgba(255, 255, 255, 0.85);
  }

  .indicator-dot.active {
    width: 14px;
    border-radius: 3px;
    background: #ffffff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.45);
  }
</style>
