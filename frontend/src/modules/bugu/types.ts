/**
 * Bugu (不孤) Radar Session Interface
 */
export interface BuguSession {
  id: string;
  sourceCardId: string;
  targetCardIds: string[];
  status: 'scanning' | 'linked' | 'fading';
  startedAt: number;
}
