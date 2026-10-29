/**
 * ATProto / Bluesky Module Types for Finnegan Frontend
 */

export interface BskyBeacon {
  sourceCardId?: string | null;
  status: 'scanning' | 'linked' | 'completed' | 'fading';
  butterflyCount: number;
}
