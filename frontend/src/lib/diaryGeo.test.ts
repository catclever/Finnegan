import { describe, it } from 'node:test';
import * as assert from 'node:assert';
import { resolveTimeWeatherEmblem, isFutureHour, clampToPresentHour } from './diaryGeo.ts';

describe('diaryGeo - Time Weather Emblem', () => {
    it('should resolve dawn correctly', () => {
        const time = new Date('2023-01-01T06:30:00');
        const res = resolveTimeWeatherEmblem(time, { condition: 'Clear' });
        assert.strictEqual(res.diurnalPhase, 'dawn');
        assert.strictEqual(res.weatherType, 'clear');
        assert.strictEqual(res.descriptor, '破晓初晴');
    });

    it('should resolve midnight storm correctly', () => {
        const time = new Date('2023-01-01T02:00:00');
        const res = resolveTimeWeatherEmblem(time, { condition: 'thunderstorm' });
        assert.strictEqual(res.diurnalPhase, 'midnight');
        assert.strictEqual(res.weatherType, 'storm');
        assert.strictEqual(res.descriptor, '深宵雷暴');
    });

    it('should handle missing weather gracefully', () => {
        const time = new Date('2023-01-01T14:00:00');
        const res = resolveTimeWeatherEmblem(time);
        assert.strictEqual(res.diurnalPhase, 'noon');
        assert.strictEqual(res.weatherType, 'unknown');
        assert.strictEqual(res.descriptor, '正午');
    });
});

describe('diaryGeo - Future Guard', () => {
    it('should identify future hours', () => {
        const now = new Date('2023-01-01T12:30:00');
        const target = new Date('2023-01-01T13:15:00'); // 13:00 vs 12:00
        assert.strictEqual(isFutureHour(target, now), true);
    });

    it('should not identify present hour as future', () => {
        const now = new Date('2023-01-01T12:30:00');
        const target = new Date('2023-01-01T12:45:00'); // 12:00 vs 12:00
        assert.strictEqual(isFutureHour(target, now), false);
    });

    it('should not identify past hour as future', () => {
        const now = new Date('2023-01-01T12:30:00');
        const target = new Date('2023-01-01T11:45:00'); // 11:00 vs 12:00
        assert.strictEqual(isFutureHour(target, now), false);
    });

    it('should clamp future time to present', () => {
        const now = new Date('2023-01-01T12:30:00');
        const target = new Date('2023-01-01T14:45:00');
        const res = clampToPresentHour(target, now);
        assert.strictEqual(res.wasFuture, true);
        assert.strictEqual(res.clamped.getTime(), now.getTime());
    });

    it('should not clamp past time', () => {
        const now = new Date('2023-01-01T12:30:00');
        const target = new Date('2023-01-01T10:45:00');
        const res = clampToPresentHour(target, now);
        assert.strictEqual(res.wasFuture, false);
        assert.strictEqual(res.clamped.getTime(), target.getTime());
    });
});
