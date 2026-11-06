import { test } from 'node:test';
import * as assert from 'node:assert';
import {
  getDiaryDateKey,
  getSameDayDiaryCards,
  getAvailableDiaryDateMap,
  computeCelestialWeatherGlyph
} from './diaryScroll.ts';
import type { DiaryCard } from './diaryScroll.ts';

test('getDiaryDateKey extracts date correctly', () => {
  const card1: DiaryCard = {
    id: '1',
    type: 'diary',
    createdAt: '2026-11-06T10:00:00.000Z'
  };

  assert.strictEqual(getDiaryDateKey(card1), '2026-11-06');

  const card2: DiaryCard = {
    id: '2',
    type: 'diary',
    weather: {
      observedHour: '2026-12-25T14:30:00.000Z'
    }
  };

  assert.strictEqual(getDiaryDateKey(card2), '2026-12-25');
});

test('getSameDayDiaryCards filters, deduplicates and sorts correctly', () => {
  const allCards: DiaryCard[] = [
    {
      id: '1',
      type: 'diary',
      createdAt: '2026-11-06T15:00:00.000Z' // 3 PM
    },
    {
      id: '2',
      type: 'thought',
      createdAt: '2026-11-06T12:00:00.000Z' // Wrong type
    },
    {
      id: '3',
      type: 'diary',
      createdAt: '2026-11-06T08:00:00.000Z' // 8 AM
    },
    {
      id: '4',
      type: 'diary',
      createdAt: '2026-11-07T08:00:00.000Z' // Wrong day
    },
    {
      id: '3',
      type: 'diary',
      createdAt: '2026-11-06T08:00:00.000Z' // Duplicate
    }
  ];

  const result = getSameDayDiaryCards('2026-11-06', allCards);

  assert.strictEqual(result.length, 2);
  assert.strictEqual(result[0].id, '3'); // 8 AM first
  assert.strictEqual(result[1].id, '1'); // 3 PM second
});

test('getAvailableDiaryDateMap generates correct map', () => {
  const allCards: DiaryCard[] = [
    { id: '1', type: 'diary', createdAt: '2026-11-06T10:00:00.000Z' },
    { id: '2', type: 'diary', createdAt: '2026-11-06T15:00:00.000Z' },
    { id: '3', type: 'diary', createdAt: '2026-11-07T08:00:00.000Z' },
    { id: '4', type: 'thought', createdAt: '2026-11-08T08:00:00.000Z' }
  ];

  const map = getAvailableDiaryDateMap(allCards);

  assert.strictEqual(map.get('2026-11-06'), 2);
  assert.strictEqual(map.get('2026-11-07'), 1);
  assert.strictEqual(map.has('2026-11-08'), false);
});

test('computeCelestialWeatherGlyph generates SVG', () => {
  // Morning Sun
  const svg1 = computeCelestialWeatherGlyph('2026-11-06T09:00:00.000Z', 'sun');
  assert.ok(svg1.includes('<svg'));
  assert.ok(svg1.includes('☀️'));

  // Night Moon
  const svg2 = computeCelestialWeatherGlyph('2026-11-06T22:00:00.000Z', 'moon');
  assert.ok(svg2.includes('<svg'));
  assert.ok(svg2.includes('🌙'));

  // Rain
  const svg3 = computeCelestialWeatherGlyph('2026-11-06T14:00:00.000Z', 'rain');
  assert.ok(svg3.includes('<svg'));
  assert.ok(svg3.includes('🌧️'));
});
