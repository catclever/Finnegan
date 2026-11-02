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
  await t.test('provides core types and curio by default', () => {
    const opts = getAvailableCardTypeOptions();
    const types = opts.map(o => o.type);
    assert.ok(types.includes('thought'));
    assert.ok(types.includes('note'));
    assert.ok(types.includes('chat_log'));
    assert.ok(types.includes('curio'));
  });

  await t.test('includes curio when module manifest includes curio', () => {
    const manifests: any[] = [
      { name: 'curio', cardTypes: [{ type: 'curio' }] }
    ];
    const opts = getAvailableCardTypeOptions(manifests);
    assert.ok(opts.some(o => o.type === 'curio'));
  });

  await t.test('excludes curio when module manifests do not include curio', () => {
    const manifests: any[] = [
      { name: 'bugu', cardTypes: [] }
    ];
    const opts = getAvailableCardTypeOptions(manifests);
    assert.equal(opts.some(o => o.type === 'curio'), false);
  });

  await t.test('filters options by query correctly', () => {
    const opts = getAvailableCardTypeOptions();
    const filteredCu = filterCardTypeOptions(opts, 'cu');
    assert.equal(filteredCu.length, 1);
    assert.equal(filteredCu[0].type, 'curio');

    const filteredNote = filterCardTypeOptions(opts, '备忘');
    assert.equal(filteredNote.length, 1);
    assert.equal(filteredNote[0].type, 'note');
  });
});
