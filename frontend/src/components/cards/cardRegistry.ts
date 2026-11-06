/**
 * Finnegan Cosmic Stage - Frontend Card Registry Center
 * Task: card_type_registry_and_babel_decoupling_091
 *
 * Layer 3 Autonomous UI component registry:
 * - Built-in mappings:
 *   - thought  -> ThoughtCardBody (Default native core)
 *   - note     -> NoteCardBody (Native core)
 *   - chat_log -> ChatLogCardBody (Native core)
 *   - babel    -> BabelCardBody (Generic shared shape)
 * - Safe fallback to ThoughtCardBody if component does not exist, is null, or is empty.
 * - Dynamic registration API for future modular extensions.
 */

import ThoughtCardBody from './ThoughtCardBody.svelte';
import NoteCardBody from './NoteCardBody.svelte';
import ChatLogCardBody from './ChatLogCardBody.svelte';
import BabelCardBody from './BabelCardBody.svelte';
import CurioCardBody from './CurioCardBody.svelte';
import DiaryCardBody from './DiaryCardBody.svelte';
import { CardRegistry } from '../../lib/cardRegistry';
import type { CardTypeDefinition, ModuleManifest } from '../../types/card';

// Initialize component registry with ThoughtCardBody as the default component
const globalCardRegistry = new CardRegistry<any>(ThoughtCardBody);

// Register built-in card components
globalCardRegistry.register('thought', ThoughtCardBody);
globalCardRegistry.register('note', NoteCardBody);
globalCardRegistry.register('chat_log', ChatLogCardBody);
globalCardRegistry.register('babel', BabelCardBody);
globalCardRegistry.register('diary', DiaryCardBody, {
  type: 'diary',
  name: '羊皮卷日记',
  description: '羊皮卷手记',
  defaultTitle: 'Diary',
  icon: '📜',
  prefix: ':diary',
  selectable: true,
});
globalCardRegistry.register('curio', CurioCardBody, {
  type: 'curio',
  name: '奇物标本',
  description: '模块驱动：商品快照与恋物藏品标本盒',
  defaultTitle: 'Curio Specimen',
  icon: '🏺',
  prefix: ':curio',
  selectable: true,
});

/**
 * Retrieves the Svelte component for a given card type.
 * If type does not exist, is empty, or is unknown, safely falls back to ThoughtCardBody.
 */
export function getCardComponent(type?: string | null): any {
  return globalCardRegistry.get(type) || ThoughtCardBody;
}

/**
 * Registers a new card component dynamically (e.g., for future module extensions like curio).
 */
export function registerCardComponent(
  type: string,
  component: any,
  meta?: Partial<CardTypeDefinition>
): void {
  globalCardRegistry.register(type, component, meta);
}

/**
 * Checks whether a card component is registered.
 */
export function hasCardComponent(type?: string | null): boolean {
  return globalCardRegistry.has(type);
}

/**
 * Gets metadata definition for a card type.
 */
export function getCardTypeDefinition(type?: string | null): CardTypeDefinition {
  return globalCardRegistry.getDefinition(type);
}

/**
 * Returns all registered card type definitions.
 */
export function getAllCardTypeDefinitions(): CardTypeDefinition[] {
  return globalCardRegistry.getAllDefinitions();
}

/**
 * Returns only the card type definitions that are selectable in OmniBar.
 */
export function getSelectableCardTypeDefinitions(): CardTypeDefinition[] {
  return globalCardRegistry.getSelectableDefinitions();
}

/**
 * Synchronizes modular card components based on active module manifests.
 * If curio module is present, register CurioCardBody.
 * If curio module is absent/disabled, unregister curio so it gracefully falls back to ThoughtCardBody!
 */
export function syncModularCardComponents(manifests?: ModuleManifest[] | null): void {
  if (!manifests) return;
  const manifestList = Array.isArray(manifests) ? manifests : [];
  const hasCurio = manifestList.some(
    m => m?.name === 'curio' || (m?.cardTypes && m.cardTypes.some(ct => ct?.type === 'curio'))
  );

  if (hasCurio) {
    globalCardRegistry.register('curio', CurioCardBody, {
      type: 'curio',
      name: '奇物标本',
      description: '模块驱动：商品快照与恋物藏品标本盒',
      defaultTitle: 'Curio Specimen',
      icon: '🏺',
      prefix: ':curio',
      selectable: true,
    });
  } else {
    globalCardRegistry.unregister('curio');
  }

  // Always register diary
  globalCardRegistry.register('diary', DiaryCardBody, {
    type: 'diary',
    name: '羊皮卷日记',
    description: '羊皮卷手记',
    defaultTitle: 'Diary',
    icon: '📜',
    prefix: ':diary',
    selectable: true,
  });
}

/**
 * Dynamically imports and registers external plugin card UI bundles declared in ModuleManifests.
 */
export async function loadExternalPluginCardComponents(manifests?: ModuleManifest[] | null): Promise<void> {
  if (!Array.isArray(manifests)) return;
  for (const m of manifests) {
    if (!m?.cardTypes || !Array.isArray(m.cardTypes)) continue;
    for (const spec of m.cardTypes) {
      if (!spec?.type || !spec.uiEntry) continue;
      try {
        const mod = await import(/* @vite-ignore */ spec.uiEntry);
        const comp = mod?.default || mod?.CardComponent || mod?.[spec.type];
        if (comp) {
          registerCardComponent(spec.type, comp, {
            type: spec.type,
            name: m.name || spec.type,
            description: m.description,
          });
        }
      } catch (err) {
        console.warn(`[CardRegistry] Failed to load external UI bundle for card type "${spec.type}" from ${spec.uiEntry}:`, err);
      }
    }
  }
}

export { globalCardRegistry };
export type { CardTypeDefinition } from '../../types/card';
export * from '../../lib/cardRegistry';
