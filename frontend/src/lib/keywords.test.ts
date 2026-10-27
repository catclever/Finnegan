import { describe, it } from 'node:test';
import assert from 'node:assert/strict';

describe('Finnegan Keywords & Manual Hashtag Unification', () => {
  describe('ThoughtCardBody manual keyword filtering', () => {
    function getManualKeywords(card: any): string[] {
      if (!card) return [];
      const list: string[] = [];
      const seen = new Set<string>();

      const kwSource = card.keywords || card.Keywords;
      if (kwSource && typeof kwSource === 'object') {
        for (const values of Object.values(kwSource)) {
          if (Array.isArray(values)) {
            for (const item of values) {
              if (typeof item === 'string' && item.endsWith('*')) {
                const clean = item.replace(/\*+$/, '').trim();
                if (clean && !seen.has(clean)) {
                  seen.add(clean);
                  list.push(clean);
                }
              }
            }
          }
        }
      }
      return list;
    }

    it('only returns keywords that have "*" suffix and renders them without "*"', () => {
      const card = {
        keywords: {
          object: ['WebKitGTK 4.1*', 'Tauri v2'],
          mood: ['松弛感*'],
          motif: ['体验新玩具'],
        },
      };

      const manual = getManualKeywords(card);
      assert.deepEqual(manual, ['WebKitGTK 4.1', '松弛感']);
    });

    it('returns empty array when no keywords have "*" suffix', () => {
      const card = {
        keywords: {
          object: ['Tauri v2'],
          motif: ['体验新玩具'],
        },
      };

      const manual = getManualKeywords(card);
      assert.deepEqual(manual, []);
    });

    it('deduplicates manual keywords', () => {
      const card = {
        keywords: {
          object: ['WebKitGTK*', 'WebKitGTK*'],
          mood: ['松弛感*'],
        },
      };

      const manual = getManualKeywords(card);
      assert.deepEqual(manual, ['WebKitGTK', '松弛感']);
    });
  });

  describe('OmniBar autocomplete keyword normalization', () => {
    function aggregateAllAvailableTags(cards: any[], localVaultPool: any[]): string[] {
      const set = new Set<string>();
      for (const c of [...cards, ...localVaultPool]) {
        if (c.tags && Array.isArray(c.tags)) {
          for (const t of c.tags) {
            if (typeof t === 'string') {
              const clean = t.replace(/\*+$/, '').trim();
              if (clean) set.add(clean);
            }
          }
        }
        if (c.keywords && typeof c.keywords === 'object') {
          for (const list of Object.values(c.keywords)) {
            if (Array.isArray(list)) {
              for (const kw of list) {
                if (typeof kw === 'string') {
                  const clean = (kw as string).replace(/\*+$/, '').trim();
                  if (clean) set.add(clean);
                }
              }
            }
          }
        }
      }
      return Array.from(set).sort((a, b) => a.localeCompare(b));
    }

    it('aggregates all keywords and tags normalized without "*" markers', () => {
      const cards = [
        {
          tags: ['legacy-tag', 'tagged*'],
          keywords: {
            object: ['WebKitGTK 4.1*'],
            mood: ['松弛感*'],
          },
        },
      ];
      const localVaultPool = [
        {
          keywords: {
            object: ['WebKitGTK 4.1', 'Tauri v2'],
            concept: ['伪提效*'],
          },
        },
      ];

      const tags = aggregateAllAvailableTags(cards, localVaultPool);
      // 'WebKitGTK 4.1*' and 'WebKitGTK 4.1' normalize to 'WebKitGTK 4.1' and are deduplicated
      assert.ok(tags.includes('WebKitGTK 4.1'));
      assert.ok(!tags.includes('WebKitGTK 4.1*'));
      assert.ok(tags.includes('松弛感'));
      assert.ok(!tags.includes('松弛感*'));
      assert.ok(tags.includes('Tauri v2'));
      assert.ok(tags.includes('伪提效'));
      assert.ok(tags.includes('legacy-tag'));
      assert.ok(tags.includes('tagged'));
      assert.ok(!tags.includes('tagged*'));
    });
  });
});
