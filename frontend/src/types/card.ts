/**
 * Finnegan Cosmic Stage - Card & Search Types
 * Task: frontend_logic_refactor_011
 */

export interface AuraTrace {
  searchId: string;
  color: string;
  hitTime: number;
  duration?: number;
}

export interface CardResurgenceState {
  summonTime: number;
  durationMs: number;
  initialZ: number;
  targetZ: number;
  searchId: string;
}

export type CardType = 'thought' | 'babel' | 'chat_log' | 'local' | 'note' | 'curio' | string;

export interface CardTypeDefinition {
  type: string;
  name: string;
  description?: string;
  isDefault?: boolean;
  defaultTitle?: string;
  icon?: string;
}

export interface DialogueTurn {
  speaker: string;
  lines: string[];
}

export interface CosmicCard {
  id: string;
  type?: CardType;
  title: string;
  content: string;
  originalText?: string;
  translatedText?: string;
  sourceUrl?: string;
  publishedAt?: string;
  authorName?: string;
  authorAvatar?: string;
  createdAt?: string | number;
  participants?: string[];
  startedAt?: string;
  endedAt?: string;
  echoOf?: string;
  matchedAnchor?: string;
  echoReason?: string;
  x: number;
  y: number;
  rotation: number;
  z: number;
  images?: string[];
  keywords?: Record<string, string[]>;
  tags?: string[];
  isEnriching?: boolean;     // True when card title/metadata is being derived by LLM
  filePath?: string;

  // Lifecycle & Decay properties
  spawnTime: number;         // When entered active stage (ms timestamp)
  decayDuration: number;     // Half-life in stage (ms)
  isFresh: boolean;          // True for new arrivals (100% optical clarity)
  isVIP: boolean;            // Latest manual user input flag
  isFadingOut?: boolean;     // True when departing stage
  isTemporary?: boolean;     // Volatile temporary card flag
  origin: 'manual' | 'bugu' | 'local' | 'atproto';
  replyCount?: number;
  repostCount?: number;
  likeCount?: number;
  macroExpiresAt?: number;   // In-memory 24-hr TTL
  elevatedAt?: number;       // Universal gravitational elevation timestamp (ms)

  // Aura Search properties
  isSearchResult?: boolean;
  auraTraces?: AuraTrace[];
  resurgence?: CardResurgenceState;

  brand?: string;
  platform?: string;
  price?: string;
  intent?: "wish" | "someday" | "archive" | "acquired";
  availability?: "deadstock" | "available" | "delisted";
  snapshotPath?: string;
  screenshotPath?: string;
}

export interface SearchSession {
  isActive: boolean;
  isConfirmed?: boolean;
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
  requireKeywords?: boolean;
  excludedTypes?: string[];
  allowedTypes?: string[];
  autoStageToViewport?: boolean;
}

export type CandidateTier = 'current_screen' | 'other_screens' | 'vault';

export interface CandidateCardItem {
  card: CosmicCard;
  tier: CandidateTier;
  screenIndex?: number;
}

export interface CommandDef {
  id: string;
  name: string;
  trigger: string;
  description: string;
  params?: ParamSpec[];
  namespace?: string;
  isPrimary?: boolean;
  subCommand?: string;
}
