/**
 * Finnegan Cosmic Stage - Card & Search Types
 * Task: frontend_logic_refactor_011
 */

export interface AuraTrace {
  searchId: string;
  color: string;
  hitTime: number;
}

export interface CosmicCard {
  id: string;
  type?: string;
  title: string;
  content: string;
  originalText?: string;
  translatedText?: string;
  sourceUrl?: string;
  publishedAt?: string;
  authorName?: string;
  authorAvatar?: string;
  createdAt?: string | number;
  echoOf?: string;
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

  // Aura Search properties
  isSearchResult?: boolean;
  auraTraces?: AuraTrace[];
}

export interface SearchSession {
  isActive: boolean;
  searchId: string;
  stashedCards: CosmicCard[];
  rankedResults: CosmicCard[];
  color: string;
}

export interface ParamSpec {
  name: string;
  type: string;
  description: string;
  dataSource?: string;
}

export interface CommandDef {
  id: string;
  name: string;
  trigger: string;
  description: string;
  params?: ParamSpec[];
}
