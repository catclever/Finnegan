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
import { CardRegistry } from '../../lib/cardRegistry';
import type { CardTypeDefinition } from '../../types/card';

// Initialize component registry with ThoughtCardBody as the default component
const globalCardRegistry = new CardRegistry<any>(ThoughtCardBody);

// Register built-in card components
globalCardRegistry.register('thought', ThoughtCardBody);
globalCardRegistry.register('note', NoteCardBody);
globalCardRegistry.register('chat_log', ChatLogCardBody);
globalCardRegistry.register('babel', BabelCardBody);

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

export { globalCardRegistry };
export type { CardTypeDefinition } from '../../types/card';
export * from '../../lib/cardRegistry';
