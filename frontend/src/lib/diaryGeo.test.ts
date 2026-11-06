import { describe, it, expect } from 'vitest';
import { resolveTimeWeatherEmblem, isFutureHour, clampToPresentHour } from './diaryGeo';

describe('diaryGeo - Time Weather Emblem', () => {
    it('should resolve dawn correctly', () => {
        const time = new Date('2023-01-01T06:30:00');
        const res = resolveTimeWeatherEmblem(time, { condition: 'Clear' });
        expect(res.diurnalPhase).toBe('dawn');
        expect(res.weatherType).toBe('clear');
        expect(res.descriptor).toBe('破晓初晴');
    });

    it('should resolve midnight storm correctly', () => {
        const time = new Date('2023-01-01T02:00:00');
        const res = resolveTimeWeatherEmblem(time, { condition: 'thunderstorm' });
        expect(res.diurnalPhase).toBe('midnight');
        expect(res.weatherType).toBe('storm');
        expect(res.descriptor).toBe('深宵雷暴');
    });

    it('should handle missing weather gracefully', () => {
        const time = new Date('2023-01-01T14:00:00');
        const res = resolveTimeWeatherEmblem(time);
        expect(res.diurnalPhase).toBe('noon');
        expect(res.weatherType).toBe('unknown');
        expect(res.descriptor).toBe('正午');
    });
});

describe('diaryGeo - Future Guard', () => {
    it('should identify future hours', () => {
        const now = new Date('2023-01-01T12:30:00');
        const target = new Date('2023-01-01T13:15:00'); // 13:00 vs 12:00
        expect(isFutureHour(target, now)).toBe(true);
    });

    it('should not identify present hour as future', () => {
        const now = new Date('2023-01-01T12:30:00');
        const target = new Date('2023-01-01T12:45:00'); // 12:00 vs 12:00
        expect(isFutureHour(target, now)).toBe(false);
        expect(isFutureHour(target, now)).toBe(false);
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
        expect(res.wasFuture).toBe(true);
        expect(res.clamped.getTime()).toBe(now.getTime());
    });

    it('should not clamp past time', () => {
        const now = new Date('2023-01-01T12:30:00');
        const target = new Date('2023-01-01T10:45:00');
        const res = clampToPresentHour(target, now);
        expect(res.wasFuture).toBe(false);
        expect(res.clamped.getTime()).toBe(target.getTime());
    });
});
