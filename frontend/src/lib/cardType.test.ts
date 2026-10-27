import test from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { parseCardTypePrefix } from './cardType.ts';

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
