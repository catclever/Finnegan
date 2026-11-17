import test from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import {
  parseCardTypePrefix,
  detectCardTypeTrigger,
  getAvailableCardTypeOptions,
  filterCardTypeOptions,
} from './cardType.ts';

test('Card Type Prefix Parser (:cardType syntax)', async (t) => {
  await t.test('recognizes :note with trailing space and empty content', () => {
    const res = parseCardTypePrefix(':note ');
    assert.deepEqual(res, {
      hasTypePrefix: true,
      type: 'note',
      content: '',
    });
  });

  await t.test('recognizes :note with trailing space and single-line content', () => {
    const res = parseCardTypePrefix(':note docker run -d redis:alpine');
    assert.deepEqual(res, {
      hasTypePrefix: true,
      type: 'note',
      content: 'docker run -d redis:alpine',
    });
  });

  await t.test('recognizes :curio with trailing space and note content', () => {
    const res = parseCardTypePrefix(':curio #包豪斯 Braun ET66 计算器');
    assert.deepEqual(res, {
      hasTypePrefix: true,
      type: 'curio',
      content: '#包豪斯 Braun ET66 计算器',
    });
  });

  await t.test('recognizes :note with multiline content and # title', () => {
    const raw = ':note # Redis启动命令\ndocker run -d -p 6379:6379 redis:alpine';
    const res = parseCardTypePrefix(raw);
    assert.deepEqual(res, {
      hasTypePrefix: true,
      type: 'note',
      content: '# Redis启动命令\ndocker run -d -p 6379:6379 redis:alpine',
    });
  });

  await t.test('rejects :note without trailing space', () => {
    const res = parseCardTypePrefix(':note');
    assert.deepEqual(res, {
      hasTypePrefix: false,
      type: null,
      content: ':note',
    });
  });

  await t.test('rejects :note with leading space', () => {
    const res = parseCardTypePrefix(' :note hello');
    assert.deepEqual(res, {
      hasTypePrefix: false,
      type: null,
      content: ' :note hello',
    });
  });

  await t.test('supports future extensible card types (e.g. :todo, :memo)', () => {
    const resTodo = parseCardTypePrefix(':todo write documentation');
    assert.deepEqual(resTodo, {
      hasTypePrefix: true,
      type: 'todo',
      content: 'write documentation',
    });

    const resMemo = parseCardTypePrefix(':memo meeting at 3pm');
    assert.deepEqual(resMemo, {
      hasTypePrefix: true,
      type: 'memo',
      content: 'meeting at 3pm',
    });
  });

  await t.test('does not misidentify ordinary text containing colons', () => {
    const res1 = parseCardTypePrefix('Time: 12:00 PM');
    assert.deepEqual(res1, {
      hasTypePrefix: false,
      type: null,
      content: 'Time: 12:00 PM',
    });

    const res2 = parseCardTypePrefix('This is a note: not a type prefix');
    assert.deepEqual(res2, {
      hasTypePrefix: false,
      type: null,
      content: 'This is a note: not a type prefix',
    });
  });
});

test('Card Type Dropdown Trigger Detection', async (t) => {
  await t.test('triggers on bare ":"', () => {
    const res = detectCardTypeTrigger(':');
    assert.deepEqual(res, { isTriggered: true, query: '' });
  });

  await t.test('triggers on partial type ":c" or ":cur"', () => {
    assert.deepEqual(detectCardTypeTrigger(':c'), { isTriggered: true, query: 'c' });
    assert.deepEqual(detectCardTypeTrigger(':cur'), { isTriggered: true, query: 'cur' });
    assert.deepEqual(detectCardTypeTrigger(':note'), { isTriggered: true, query: 'note' });
  });

  await t.test('does not trigger when trailing space exists (completed prefix)', () => {
    assert.deepEqual(detectCardTypeTrigger(':note '), { isTriggered: false, query: '' });
    assert.deepEqual(detectCardTypeTrigger(':curio test'), { isTriggered: false, query: '' });
  });

  await t.test('does not trigger on plain text without leading colon', () => {
    assert.deepEqual(detectCardTypeTrigger('hello:world'), { isTriggered: false, query: '' });
    assert.deepEqual(detectCardTypeTrigger(''), { isTriggered: false, query: '' });
  });
});

test('Card Type Options & Filtering Engine', async (t) => {
  await t.test('provides built-in selectable types (note, book, diary) by default without hardcoded curio, excluding thought and chat_log', () => {
    const opts = getAvailableCardTypeOptions();
    const types = opts.map(o => o.type);
    assert.ok(types.includes('note'));
    assert.ok(types.includes('book'));
    assert.ok(types.includes('diary'));
    assert.equal(types.includes('curio'), false);
    assert.equal(types.includes('thought'), false);
    assert.equal(types.includes('chat_log'), false);
    assert.equal(types.includes('babel'), false);
  });

  await t.test('discovers curio dynamically when external curio manifest is present', () => {
    const manifests: any[] = [
      {
        name: 'curio',
        description: '商品网页快照与恋物藏品 (Curio 外部插件驱动)',
        themeColor: '#10b981',
        isExternal: true,
        cardTypes: [
          {
            type: 'curio',
            name: '奇物标本',
            icon: '🏺',
            prefix: ':curio',
            submitLabel: '标本入库',
            badgeColor: '#10b981',
            selectable: true,
            uiEntry: '/api/plugins/curio/ui.js',
          },
        ],
      },
    ];
    const opts = getAvailableCardTypeOptions(manifests);
    const curio = opts.find(o => o.type === 'curio');
    assert.ok(curio);
    assert.equal(curio.prefix, ':curio');
    assert.equal(curio.label, '奇物标本');
    assert.equal(curio.submitLabel, '标本入库');
    assert.equal(curio.badgeColor, '#10b981');
    assert.equal(curio.isExternal, true);
  });

  await t.test('excludes curio when module manifests do not include curio', () => {
    const manifests: any[] = [
      { name: 'bugu', cardTypes: [] }
    ];
    const opts = getAvailableCardTypeOptions(manifests);
    assert.equal(opts.some(o => o.type === 'curio'), false);
    assert.deepEqual(opts.map(o => o.type), ['note', 'book']);
  });

  await t.test('note and book are always present as built-in selectable types with empty/null manifests', () => {
    for (const m of [[], null, undefined] as any[]) {
      const types = getAvailableCardTypeOptions(m).map(o => o.type);
      assert.ok(types.includes('note'));
      assert.ok(types.includes('book'));
      assert.equal(types.includes('curio'), false);
      assert.equal(types.includes('thought'), false);
      assert.equal(types.includes('chat_log'), false);
    }
  });

  await t.test('supports custom registry definitions directly with selectable filtering', () => {
    const customDefs: any[] = [
      { type: 'thought', name: '灵感', selectable: false },
      { type: 'note', name: '长文', selectable: true, prefix: ':note', icon: '📝' },
      { type: 'chat_log', name: '对话', selectable: false },
      { type: 'curio', name: '奇物', selectable: true, prefix: ':curio', icon: '🏺' }
    ];
    const customOpts = getAvailableCardTypeOptions(null, customDefs);
    assert.equal(customOpts.length, 2);
    assert.deepEqual(customOpts.map(o => o.type), ['note', 'curio']);
  });

  await t.test('filters options by query correctly', () => {
    const opts = getAvailableCardTypeOptions([
      {
        name: 'curio',
        isExternal: true,
        cardTypes: [{ type: 'curio', name: '奇物标本', prefix: ':curio', selectable: true }],
      },
    ]);
    const filteredCu = filterCardTypeOptions(opts, 'cu');
    assert.equal(filteredCu.length, 1);
    assert.equal(filteredCu[0].type, 'curio');

    const filteredNote = filterCardTypeOptions(opts, '长文');
    assert.equal(filteredNote.length, 1);
    assert.equal(filteredNote[0].type, 'note');
  });

  await t.test('dynamically discovers external CardTypeSpec :prefix options with display metadata and deduplication', () => {
    const manifests: any[] = [
      {
        name: 'taohuo',
        description: '淘货发现',
        themeColor: '#ec4899',
        isExternal: true,
        cardTypes: [
          {
            type: 'taohuo',
            name: '淘货',
            description: '好物与二手情报',
            icon: '🛍️',
            prefix: ':taohuo',
            submitLabel: '淘货入库',
            badgeColor: '#db2777',
            selectable: true,
          },
          {
            type: 'taohuo',
            name: 'Duplicate Taohuo',
            prefix: ':taohuo',
            selectable: true,
          },
          {
            type: 'internal_hidden',
            name: 'Hidden Type',
            selectable: false,
          },
        ],
      },
      {
        name: 'recipe',
        description: '食谱记录',
        themeColor: '#10b981',
        isExternal: true,
        cardTypes: [
          {
            type: 'recipe',
            prefix: 'recipe',
          },
        ],
      },
      {
        name: 'diary',
        cardTypes: [{ type: 'diary' }],
      },
    ];

    const opts = getAvailableCardTypeOptions(manifests);
    const taohuo = opts.find(o => o.type === 'taohuo');
    assert.ok(taohuo);
    assert.equal(taohuo.prefix, ':taohuo');
    assert.equal(taohuo.label, '淘货');
    assert.equal(taohuo.description, '好物与二手情报');
    assert.equal(taohuo.icon, '🛍️');
    assert.equal(taohuo.submitLabel, '淘货入库');
    assert.equal(taohuo.badgeColor, '#db2777');
    assert.equal(taohuo.isExternal, true);
    assert.equal(taohuo.moduleName, 'taohuo');

    assert.equal(opts.filter(o => o.type === 'taohuo').length, 1);
    assert.equal(opts.some(o => o.type === 'internal_hidden'), false);

    const recipe = opts.find(o => o.type === 'recipe');
    assert.ok(recipe);
    assert.equal(recipe.prefix, ':recipe');
    assert.equal(recipe.label, 'recipe');
    assert.equal(recipe.description, '食谱记录');
    assert.equal(recipe.icon, '🧩');
    assert.equal(recipe.submitLabel, '入库');
    assert.equal(recipe.badgeColor, '#10b981');
    assert.equal(recipe.moduleName, 'recipe');

    const diary = opts.find(o => o.type === 'diary');
    assert.ok(diary);
    assert.equal(diary.label, '羊皮卷日记');
    assert.equal(diary.icon, '📜');
    assert.equal(diary.prefix, ':diary');
  });
});
