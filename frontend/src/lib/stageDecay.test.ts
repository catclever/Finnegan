import test from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { toCosmicCard } from './stageDecay.ts';
// @ts-ignore
import { mergeCapturedThought, updateCardInPools } from './cardCapture.ts';

test('CosmicCard.meta preservation across stageDecay and cardCapture', async (t) => {
  await t.test('toCosmicCard preserves camelCase meta and PascalCase Meta', () => {
    const camel = toCosmicCard(
      {
        id: 'card_1',
        type: 'taohuo',
        title: 'Vintage Synth',
        content: 'Found on Mercari',
        meta: { price: '¥1200', seller: 'retro_shop', condition: 'mint' },
      },
      'taohuo',
      'local'
    );
    assert.deepEqual(camel.meta, { price: '¥1200', seller: 'retro_shop', condition: 'mint' });

    const pascal = toCosmicCard(
      {
        ID: 'card_2',
        Type: 'taohuo',
        Title: 'Field Recorder',
        Content: 'Sony PCM',
        Meta: { brand: 'Sony', tier: 1 },
      },
      'taohuo',
      'manual'
    );
    assert.deepEqual(pascal.meta, { brand: 'Sony', tier: 1 });
  });

  await t.test('mergeCapturedThought preserves Meta for newly captured module cards', () => {
    const res = mergeCapturedThought({
      data: {
        DocID: 'mod_001',
        Type: 'taohuo',
        Title: 'Braun ET66',
        Text: 'Classic calculator',
        Meta: { designer: 'Dieter Rams', year: 1987 },
      },
      cards: [],
      memoryPool: [],
      localVaultPool: [],
      scrollExpansion: 0,
      latestManualCardId: null,
    });

    const staged = res.cards.find((c: any) => c.id === 'mod_001');
    assert.ok(staged);
    assert.equal(staged.type, 'taohuo');
    assert.deepEqual(staged.meta, { designer: 'Dieter Rams', year: 1987 });
  });

  await t.test('updateCardInPools preserves updated meta across stage, localVault, and memory pools', () => {
    const initial = toCosmicCard(
      {
        id: 'mod_002',
        type: 'taohuo',
        title: 'Initial',
        content: 'Body',
        filePath: '/tmp/mod_002.md',
        meta: { status: 'draft' },
      },
      'taohuo',
      'local'
    );

    const updated = updateCardInPools(
      {
        id: 'mod_002',
        type: 'taohuo',
        title: 'Updated',
        content: 'Updated Body',
        filePath: '/tmp/mod_002.md',
        meta: { status: 'archived', score: 9.5 },
      },
      {
        cards: [initial],
        memoryPool: [initial],
        localVaultPool: [initial],
      }
    );

    assert.deepEqual(updated.cards[0].meta, { status: 'archived', score: 9.5 });
    assert.deepEqual(updated.memoryPool[0].meta, { status: 'archived', score: 9.5 });
    assert.deepEqual(updated.localVaultPool[0].meta, { status: 'archived', score: 9.5 });
  });
});
