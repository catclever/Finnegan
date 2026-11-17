<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - ExternalCardHost Component
  Framework-agnostic sandbox host for external plugin card UIs:
  1. Mount Adapter Object / Factory: { mount(container, props, host) } or (container, props, host) => ({ update?, destroy? })
  2. Web Component / Custom Element: customElementTag (e.g. "finnegan-curio-card")
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { ExecuteModuleAction } from '../../../wailsjs/go/main/App.js';
  import {
    cardRegistryVersion,
    getCardTypeDefinition,
    getExternalCardRenderer,
    type ExternalCardRendererEntry,
  } from './cardRegistry';
  import ThoughtCardBody from './ThoughtCardBody.svelte';
  import type { CosmicCard } from '../../types/card';

  export let card: CosmicCard;
  export let allCards: CosmicCard[] = [];
  export let z: number | undefined = undefined;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let displayTime: string = '';
  export let thoughtTime: string = '';
  export let babelTime: string = '';
  export let activeConstellationTags: Map<string, string> = new Map();
  export let activeConstellationTag: string | null = null;

  const dispatch = createEventDispatcher<{
    tagClick: { tag: string; cardId: string };
  }>();

  let containerEl: HTMLDivElement | null = null;
  let isMounted = false;
  let mountedEntry: ExternalCardRendererEntry | undefined = undefined;
  let adapterInstance: { update?: (props: any) => void; destroy?: () => void } | null = null;
  let customEl: any = null;

  $: resolvedDisplayTime = displayTime || thoughtTime || babelTime || '';
  $: rendererEntry = ($cardRegistryVersion >= 0) ? getExternalCardRenderer(card?.type) : undefined;
  $: hostModuleName =
    rendererEntry?.moduleName ||
    (($cardRegistryVersion >= 0) ? getCardTypeDefinition(card?.type)?.moduleName : undefined) ||
    card?.origin ||
    card?.type ||
    '';

  const host = {
    openURL: (urlOrPath: string) => {
      if (urlOrPath) {
        BrowserOpenURL(urlOrPath);
      }
    },
    emitTagClick: (tag: string) => {
      if (tag) {
        dispatch('tagClick', { tag, cardId: card?.id });
      }
    },
    executeAction: (actionId: string, payload?: Record<string, any>) => {
      const targetModule = hostModuleName || card?.origin || card?.type || '';
      return ExecuteModuleAction(targetModule, actionId, payload || {});
    },
    resolveAssetUrl: (img: string): string => {
      if (!img || typeof img !== 'string') return '';
      if (
        img.startsWith('http://') ||
        img.startsWith('https://') ||
        img.startsWith('data:') ||
        img.startsWith('/api/assets/')
      ) {
        return img;
      }
      const cleanName = img.startsWith('/') ? img.slice(1) : img;
      return `/api/assets/${cleanName}`;
    },
  };

  function buildCurrentProps() {
    return {
      card,
      allCards,
      z,
      isFocused,
      isHovered,
      displayTime: resolvedDisplayTime,
      thoughtTime: thoughtTime || resolvedDisplayTime,
      babelTime: babelTime || resolvedDisplayTime,
      activeConstellationTags,
      activeConstellationTag,
      host,
    };
  }

  function handleCustomTagClick(e: Event) {
    const detail = (e as CustomEvent)?.detail;
    if (typeof detail === 'string' && detail) {
      dispatch('tagClick', { tag: detail, cardId: card?.id });
    } else if (detail && typeof detail.tag === 'string' && detail.tag) {
      dispatch('tagClick', { tag: detail.tag, cardId: detail.cardId || card?.id });
    }
  }

  function syncCustomElementProps(el: any) {
    if (!el) return;
    el.card = card;
    el.allCards = allCards;
    el.z = z;
    el.isFocused = isFocused;
    el.isHovered = isHovered;
    el.displayTime = resolvedDisplayTime;
    el.thoughtTime = thoughtTime || resolvedDisplayTime;
    el.babelTime = babelTime || resolvedDisplayTime;
    el.activeConstellationTags = activeConstellationTags;
    el.activeConstellationTag = activeConstellationTag;
    el.host = host;
  }

  function cleanupRenderer() {
    if (adapterInstance) {
      try {
        adapterInstance.destroy?.();
      } catch (err) {
        console.warn('[ExternalCardHost] adapter destroy error:', err);
      }
      adapterInstance = null;
    }
    if (customEl) {
      try {
        customEl.removeEventListener('tagClick', handleCustomTagClick);
        customEl.remove();
      } catch (err) {
        console.warn('[ExternalCardHost] custom element cleanup error:', err);
      }
      customEl = null;
    }
    if (containerEl) {
      containerEl.innerHTML = '';
    }
    mountedEntry = undefined;
  }

  function mountCurrentRenderer() {
    if (!isMounted || !containerEl) return;
    cleanupRenderer();
    mountedEntry = rendererEntry;
    if (!rendererEntry) return;

    if (rendererEntry.customElementTag) {
      const el = document.createElement(rendererEntry.customElementTag) as any;
      syncCustomElementProps(el);
      el.addEventListener('tagClick', handleCustomTagClick);
      containerEl.appendChild(el);
      customEl = el;
      return;
    }

    const adapter = rendererEntry.adapter;
    const props = buildCurrentProps();
    if (typeof adapter === 'function') {
      const res = adapter(containerEl, props, host);
      adapterInstance = res && typeof res === 'object' ? res : null;
    } else if (adapter && typeof adapter.mount === 'function') {
      const res = adapter.mount(containerEl, props, host);
      adapterInstance = res && typeof res === 'object' ? res : null;
    }
  }

  $: hasValidRenderer = Boolean(
    rendererEntry &&
    (rendererEntry.customElementTag ||
      typeof rendererEntry.adapter === 'function' ||
      typeof rendererEntry.adapter?.mount === 'function')
  );

  $: if (isMounted && containerEl && hasValidRenderer) {
    void card;
    void allCards;
    void z;
    void isFocused;
    void isHovered;
    void resolvedDisplayTime;
    void activeConstellationTags;
    void activeConstellationTag;

    if (rendererEntry !== mountedEntry) {
      mountCurrentRenderer();
    } else if (customEl) {
      syncCustomElementProps(customEl);
    } else if (adapterInstance && typeof adapterInstance.update === 'function') {
      adapterInstance.update(buildCurrentProps());
    }
  } else if (isMounted && !hasValidRenderer && (adapterInstance || customEl || mountedEntry)) {
    cleanupRenderer();
  }

  onMount(() => {
    isMounted = true;
    return () => {
      isMounted = false;
      cleanupRenderer();
    };
  });

  onDestroy(() => {
    isMounted = false;
    cleanupRenderer();
  });
</script>

{#if !hasValidRenderer}
  <ThoughtCardBody
    {card}
    {z}
    {isFocused}
    {isHovered}
    {thoughtTime}
    {babelTime}
    {displayTime}
    {activeConstellationTags}
    {activeConstellationTag}
    on:tagClick
  />
{:else}
  <div class="external-card-mount" bind:this={containerEl}></div>
{/if}

<style>
  .external-card-mount {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
</style>
