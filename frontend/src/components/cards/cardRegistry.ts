/**
 * Finnegan Cosmic Stage - Frontend Card Registry Center
 * Task: card_type_registry_and_babel_decoupling_091 / host_freeze_frontend_ui_sandbox_and_omnibar_174
 *
 * Layer 3 Autonomous UI component registry:
 * - Built-in mappings:
 *   - thought  -> ThoughtCardBody (Default native core)
 *   - note     -> NoteCardBody (Native core)
 *   - chat_log -> ChatLogCardBody (Native core)
 *   - babel    -> BabelCardBody (Generic shared shape)
 * - Safe fallback to ThoughtCardBody if component does not exist, is null, or is empty.
 * - Dynamic framework-agnostic ExternalCardHost registration & live hot-reload via cardRegistryVersion.
 */

import { writable } from 'svelte/store';
import ThoughtCardBody from './ThoughtCardBody.svelte';
import NoteCardBody from './NoteCardBody.svelte';
import ChatLogCardBody from './ChatLogCardBody.svelte';
import BabelCardBody from './BabelCardBody.svelte';
import DiaryCardBody from './DiaryCardBody.svelte';
import BookCardBody from './BookCardBody.svelte';
import ExternalCardHost from './ExternalCardHost.svelte';
import {
  CardRegistry,
  appendCacheBust,
  buildCardTypeDefinitionFromSpec,
  reconcileExternalCardTypes,
} from '../../lib/cardRegistry';
import type { CardTypeDefinition, ModuleManifest } from '../../types/card';

export interface ExternalCardRendererEntry {
  type: string;
  moduleName?: string;
  customElementTag?: string;
  adapter?: any;
  uiEntry?: string;
}

export const cardRegistryVersion = writable(0);

// Initialize component registry with ThoughtCardBody as the default component
const globalCardRegistry = new CardRegistry<any>(ThoughtCardBody);
const externalRegisteredTypes = new Set<string>();
const externalDefinitionTypes = new Set<string>();
const externalRendererMap = new Map<string, ExternalCardRendererEntry>();

function bumpRegistryVersion(): void {
  cardRegistryVersion.update(v => v + 1);
}

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
globalCardRegistry.register('book', BookCardBody, {
  type: 'book',
  name: '读书笔记',
  description: '藏书票书摘：原文引文与个人批注',
  defaultTitle: 'Reading Note',
  icon: '📖',
  prefix: ':book',
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
 * Retrieves the external renderer descriptor for a given card type, if any.
 */
export function getExternalCardRenderer(type?: string | null): ExternalCardRendererEntry | undefined {
  if (!type || typeof type !== 'string') return undefined;
  return externalRendererMap.get(type.trim());
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
  bumpRegistryVersion();
}

/**
 * Unregisters a card component dynamically.
 */
export function unregisterCardComponent(type: string): void {
  if (!type || typeof type !== 'string') return;
  const cleanType = type.trim();
  externalRegisteredTypes.delete(cleanType);
  externalDefinitionTypes.delete(cleanType);
  externalRendererMap.delete(cleanType);
  globalCardRegistry.unregister(cleanType);
  bumpRegistryVersion();
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
 */
export function syncModularCardComponents(manifests?: ModuleManifest[] | null): void {
  if (!manifests) return;

  // Always register diary unless externally overridden
  if (!externalRegisteredTypes.has('diary')) {
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

  // Always register built-in book unless externally overridden
  if (!externalRegisteredTypes.has('book')) {
    globalCardRegistry.register('book', BookCardBody, {
      type: 'book',
      name: '读书笔记',
      description: '藏书票书摘：原文引文与个人批注',
      defaultTitle: 'Reading Note',
      icon: '📖',
      prefix: ':book',
      selectable: true,
    });
  }

  bumpRegistryVersion();
}

/**
 * Dynamically imports and registers external plugin card UI bundles declared in ModuleManifests.
 * Unregisters removed external card types so deleted plugins immediately fall back to ThoughtCardBody.
 */
export async function loadExternalPluginCardComponents(
  manifests?: ModuleManifest[] | null,
  cacheBust?: number
): Promise<void> {
  const { removedUITypes, activeUISpecs } = reconcileExternalCardTypes(
    globalCardRegistry,
    manifests,
    externalRegisteredTypes,
    externalDefinitionTypes
  );

  for (const removed of removedUITypes) {
    externalRendererMap.delete(removed);
  }

  if (removedUITypes.some(t => t === 'diary' || t === 'book')) {
    syncModularCardComponents(manifests);
  }

  for (const { manifest: m, spec } of activeUISpecs) {
    try {
      let exported: any = undefined;
      if (spec.uiEntry) {
        const urlWithCacheBust = appendCacheBust(spec.uiEntry, cacheBust);
        const mod = await import(/* @vite-ignore */ urlWithCacheBust);
        exported =
          mod?.default ??
          mod?.CardComponent ??
          mod?.[spec.type] ??
          (typeof mod?.mount === 'function' ? { mount: mod.mount } : undefined);
      }

      const hasMountAdapter =
        typeof exported === 'function' ||
        (exported && typeof exported.mount === 'function');
      const hasCustomElement = Boolean(spec.customElementTag);

      if (hasCustomElement || hasMountAdapter) {
        const existingDef = globalCardRegistry.hasDefinition(spec.type)
          ? globalCardRegistry.getDefinition(spec.type)
          : undefined;
        const mergedDef = buildCardTypeDefinitionFromSpec(spec, m, existingDef);
        externalRendererMap.set(spec.type, {
          type: spec.type,
          moduleName: m.name,
          customElementTag: spec.customElementTag,
          adapter: hasMountAdapter ? exported : undefined,
          uiEntry: spec.uiEntry,
        });
        externalRegisteredTypes.add(spec.type);
        globalCardRegistry.register(spec.type, ExternalCardHost, mergedDef);
      }
    } catch (err) {
      console.warn(`[CardRegistry] Failed to load external UI bundle for card type "${spec.type}" from ${spec.uiEntry}:`, err);
    }
  }

  bumpRegistryVersion();
}

export { globalCardRegistry };
export type { CardTypeDefinition } from '../../types/card';
export * from '../../lib/cardRegistry';
