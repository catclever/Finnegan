import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import {
  parseDate,
  getTimePeriod,
  formatPointTime,
  formatIntervalTime,
  getCardDisplayTime,
  stageTime,
  refreshStageTime
} from './time.ts';

describe('Finnegan Unified Card Time Formatting Engine', () => {
  describe('parseDate', () => {
    it('gracefully returns null for null, undefined, empty string, and whitespace', () => {
      assert.equal(parseDate(null), null);
      assert.equal(parseDate(undefined), null);
      assert.equal(parseDate(''), null);
      assert.equal(parseDate('   '), null);
    });

    it('gracefully returns null for invalid numbers and NaN', () => {
      assert.equal(parseDate(NaN), null);
      assert.equal(parseDate(Infinity), null);
    });

    it('gracefully returns null for invalid date strings and invalid Date objects', () => {
      assert.equal(parseDate('not-a-valid-date'), null);
      assert.equal(parseDate('2026-99-99T99:99:99'), null);
      assert.equal(parseDate(new Date('invalid')), null);
    });

    it('parses Date objects', () => {
      const original = new Date(2026, 9, 20, 15, 30);
      const parsed = parseDate(original);
      assert.ok(parsed instanceof Date);
      assert.equal(parsed?.getTime(), original.getTime());
    });

    it('parses numeric millisecond unix timestamps', () => {
      const ms = 1729425600000;
      const parsed = parseDate(ms);
      assert.ok(parsed instanceof Date);
      assert.equal(parsed?.getTime(), ms);
    });

    it('parses string millisecond unix timestamps', () => {
      const ms = 1729425600000;
      const parsed = parseDate(String(ms));
      assert.ok(parsed instanceof Date);
      assert.equal(parsed?.getTime(), ms);
    });

    it('parses ISO 8601 strings', () => {
      const iso = '2026-10-20T12:00:00.000Z';
      const parsed = parseDate(iso);
      assert.ok(parsed instanceof Date);
      assert.equal(parsed?.toISOString(), iso);
    });
  });

  describe('getTimePeriod', () => {
    it('correctly maps 24 hours to intuitive time periods', () => {
      // 凌晨: 3点 ~ 5点 (03:00 - 04:59)
      assert.equal(getTimePeriod(3), '凌晨');
      assert.equal(getTimePeriod(4), '凌晨');

      // 清晨: 5点 ~ 8点 (05:00 - 07:59)
      assert.equal(getTimePeriod(5), '清晨');
      assert.equal(getTimePeriod(6), '清晨');
      assert.equal(getTimePeriod(7), '清晨');

      // 上午: 8点 ~ 12点 (08:00 - 11:59)
      assert.equal(getTimePeriod(8), '上午');
      assert.equal(getTimePeriod(9), '上午');
      assert.equal(getTimePeriod(11), '上午');

      // 中午: 12点 ~ 15点 (12:00 - 14:59)
      assert.equal(getTimePeriod(12), '中午');
      assert.equal(getTimePeriod(13), '中午');
      assert.equal(getTimePeriod(14), '中午');

      // 下午: 15点 ~ 19点 (15:00 - 18:59)
      assert.equal(getTimePeriod(15), '下午');
      assert.equal(getTimePeriod(16), '下午');
      assert.equal(getTimePeriod(18), '下午');

      // 晚上: 19点 ~ 23点 (19:00 - 22:59)
      assert.equal(getTimePeriod(19), '晚上');
      assert.equal(getTimePeriod(20), '晚上');
      assert.equal(getTimePeriod(22), '晚上');

      // 深夜: 23点 ~ 3点 (23:00 - 02:59)
      assert.equal(getTimePeriod(23), '深夜');
      assert.equal(getTimePeriod(0), '深夜');
      assert.equal(getTimePeriod(1), '深夜');
      assert.equal(getTimePeriod(2), '深夜');
    });
  });

  describe('formatPointTime', () => {
    // Reference base time: 2026-10-20 12:00:00 (local time)
    const base = new Date(2026, 9, 20, 12, 0, 0);

    it('returns empty string for null, undefined, or invalid dates', () => {
      assert.equal(formatPointTime(null, base), '');
      assert.equal(formatPointTime(undefined, base), '');
      assert.equal(formatPointTime('invalid-date', base), '');
    });

    it('< 5 minutes: returns "刚刚" (including slight future drift)', () => {
      // 0 seconds ago (exact match)
      assert.equal(formatPointTime(new Date(2026, 9, 20, 12, 0, 0), base), '刚刚');
      // 10 seconds in the future
      assert.equal(formatPointTime(new Date(2026, 9, 20, 12, 0, 10), base), '刚刚');
      // 1 minute ago
      assert.equal(formatPointTime(new Date(2026, 9, 20, 11, 59, 0), base), '刚刚');
      // 4 minutes 59 seconds ago
      assert.equal(formatPointTime(new Date(2026, 9, 20, 11, 55, 1), base), '刚刚');
    });

    it('same calendar day: returns "今天 HH:mm"', () => {
      // Exactly 5 minutes ago today
      assert.equal(formatPointTime(new Date(2026, 9, 20, 11, 55, 0), base), '今天 11:55');
      // 2 hours ago today
      assert.equal(formatPointTime(new Date(2026, 9, 20, 10, 0, 0), base), '今天 10:00');
      // Early morning today (11 hours ago)
      assert.equal(formatPointTime(new Date(2026, 9, 20, 1, 0, 0), base), '今天 01:00');
    });

    it('1 calendar day ago: returns "昨天 HH:mm"', () => {
      // Yesterday noon (24 hours ago)
      assert.equal(formatPointTime(new Date(2026, 9, 19, 12, 0, 0), base), '昨天 12:00');
      // Yesterday midnight (36 hours ago)
      assert.equal(formatPointTime(new Date(2026, 9, 19, 0, 0, 0), base), '昨天 00:00');
      // Yesterday night late (12 hours 1 minute ago)
      assert.equal(formatPointTime(new Date(2026, 9, 19, 23, 59, 0), base), '昨天 23:59');
      // Yesterday afternoon
      assert.equal(formatPointTime(new Date(2026, 9, 19, 15, 30, 0), base), '昨天 15:30');
    });

    it('2 calendar days ago: returns "前天 HH:mm"', () => {
      // The day before yesterday (Oct 18) at noon
      assert.equal(formatPointTime(new Date(2026, 9, 18, 12, 0, 0), base), '前天 12:00');
      // The day before yesterday afternoon (e.g. 15:00 - previously wrongly returned "昨天" under 48h rolling window)
      assert.equal(formatPointTime(new Date(2026, 9, 18, 15, 0, 0), base), '前天 15:00');
      // The day before yesterday night (e.g. 23:30)
      assert.equal(formatPointTime(new Date(2026, 9, 18, 23, 30, 0), base), '前天 23:30');
      // The day before yesterday early morning (e.g. 06:30)
      assert.equal(formatPointTime(new Date(2026, 9, 18, 6, 30, 0), base), '前天 06:30');
    });

    it('3 to 29 calendar days ago: returns "{days} 天前 {period}"', () => {
      // 3 days ago at 09:15 (上午)
      assert.equal(formatPointTime(new Date(2026, 9, 17, 9, 15, 0), base), '3 天前 上午');
      // 5 days ago at 16:20 (下午)
      assert.equal(formatPointTime(new Date(2026, 9, 15, 16, 20, 0), base), '5 天前 下午');
      // 5 days ago at 10:20 (上午)
      assert.equal(formatPointTime(new Date(2026, 9, 15, 10, 20, 0), base), '5 天前 上午');
      // 7 days ago at 20:45 (晚上)
      assert.equal(formatPointTime(new Date(2026, 9, 13, 20, 45, 0), base), '7 天前 晚上');
      // 10 days ago at 23:30 (深夜)
      assert.equal(formatPointTime(new Date(2026, 9, 10, 23, 30, 0), base), '10 天前 深夜');
      // 15 days ago at 04:00 (凌晨)
      assert.equal(formatPointTime(new Date(2026, 9, 5, 4, 0, 0), base), '15 天前 凌晨');
    });

    it('30 days to 1 year: returns "MM/DD HH:mm" with 2-digit zero padding', () => {
      // Exactly 30 days ago: 2026-09-20 12:00
      assert.equal(formatPointTime(new Date(2026, 8, 20, 12, 0, 0), base), '09/20 12:00');
      // 60 days ago: 2026-08-21 09:05
      assert.equal(formatPointTime(new Date(2026, 7, 21, 9, 5, 0), base), '08/21 09:05');
      // ~200 days ago (same year): 2026-04-03 14:20
      assert.equal(formatPointTime(new Date(2026, 3, 3, 14, 20, 0), base), '04/03 14:20');
      // 364 days ago (different day than today): 2025-10-21 12:00
      assert.equal(formatPointTime(new Date(2025, 9, 21, 12, 0, 0), base), '10/21 12:00');
    });

    it('> 1 year (different year): returns "YYYY/MM/DD" with zero padding', () => {
      // 366 days ago: 2025-10-19 12:00
      assert.equal(formatPointTime(new Date(2025, 9, 19, 12, 0, 0), base), '2025/10/19');
      // 500 days ago: 2025-06-07 09:00
      assert.equal(formatPointTime(new Date(2025, 5, 7, 9, 0, 0), base), '2025/06/07');
      // 3 years ago on a different day: 2023-01-05
      assert.equal(formatPointTime(new Date(2023, 0, 5, 10, 0, 0), base), '2023/01/05');
    });

    describe('Easter egg: "{years} 年前的今天"', () => {
      it('returns "1 年前的今天" when month and day match today 1 year ago', () => {
        // Exactly 1 year ago today
        const oneYearAgo = new Date(2025, 9, 20, 10, 0, 0);
        assert.equal(formatPointTime(oneYearAgo, base), '1 年前的今天');

        // Later in the evening 1 year ago today
        const oneYearAgoEvening = new Date(2025, 9, 20, 22, 30, 0);
        assert.equal(formatPointTime(oneYearAgoEvening, base), '1 年前的今天');
      });

      it('returns "2 年前的今天" for 2 years ago today', () => {
        const twoYearsAgo = new Date(2024, 9, 20, 14, 15, 0);
        assert.equal(formatPointTime(twoYearsAgo, base), '2 年前的今天');
      });

      it('returns "5 年前的今天" for 5 years ago today', () => {
        const fiveYearsAgo = new Date(2021, 9, 20, 8, 0, 0);
        assert.equal(formatPointTime(fiveYearsAgo, base), '5 年前的今天');
      });
    });
  });

  describe('formatIntervalTime', () => {
    const base = new Date(2026, 9, 20, 12, 0, 0);

    it('returns empty string if both start and end are missing or invalid', () => {
      assert.equal(formatIntervalTime(null, null, base), '');
      assert.equal(formatIntervalTime(undefined, undefined, base), '');
      assert.equal(formatIntervalTime('invalid', 'also-invalid', base), '');
    });

    it('formats single date properly if only one date is present', () => {
      // Current year single date
      assert.equal(
        formatIntervalTime(new Date(2026, 9, 20, 9, 5), null, base),
        '10/20 09:05'
      );
      assert.equal(
        formatIntervalTime(null, new Date(2026, 9, 20, 9, 5), base),
        '10/20 09:05'
      );
      // Different year single date
      assert.equal(
        formatIntervalTime(new Date(2024, 4, 3, 8, 4), null, base),
        '2024/05/03 08:04'
      );
    });

    it('handles same-day interval in current year: "MM/DD HH:mm – HH:mm"', () => {
      const start = new Date(2026, 9, 20, 9, 5);
      const end = new Date(2026, 9, 20, 11, 45);
      assert.equal(formatIntervalTime(start, end, base), '10/20 09:05 – 11:45');
    });

    it('handles same-day interval with zero-padded month, day, hours, and minutes', () => {
      const start = new Date(2026, 4, 3, 8, 4);
      const end = new Date(2026, 4, 3, 9, 8);
      assert.equal(formatIntervalTime(start, end, base), '05/03 08:04 – 09:08');
    });

    it('handles same-day interval in a past year: "YYYY/MM/DD HH:mm – HH:mm"', () => {
      const start = new Date(2024, 4, 3, 8, 4);
      const end = new Date(2024, 4, 3, 9, 8);
      assert.equal(formatIntervalTime(start, end, base), '2024/05/03 08:04 – 09:08');
    });

    it('handles cross-day interval in current year: "MM/DD HH:mm – MM/DD HH:mm"', () => {
      const start = new Date(2026, 9, 20, 23, 30);
      const end = new Date(2026, 9, 21, 1, 15);
      assert.equal(formatIntervalTime(start, end, base), '10/20 23:30 – 10/21 01:15');
    });

    it('handles cross-day interval in a past year: "YYYY/MM/DD HH:mm – MM/DD HH:mm"', () => {
      const start = new Date(2024, 9, 20, 23, 30);
      const end = new Date(2024, 9, 21, 1, 15);
      assert.equal(formatIntervalTime(start, end, base), '2024/10/20 23:30 – 10/21 01:15');
    });

    it('handles cross-year interval: "YYYY/MM/DD HH:mm – YYYY/MM/DD HH:mm"', () => {
      const start = new Date(2025, 11, 31, 23, 45);
      const end = new Date(2026, 0, 1, 0, 30);
      assert.equal(
        formatIntervalTime(start, end, base),
        '2025/12/31 23:45 – 2026/01/01 00:30'
      );
    });

    it('automatically reorders start and end if start > end', () => {
      const start = new Date(2026, 9, 20, 11, 45);
      const end = new Date(2026, 9, 20, 9, 5);
      assert.equal(formatIntervalTime(start, end, base), '10/20 09:05 – 11:45');
    });

    it('handles same timestamp for start and end gracefully', () => {
      const start = new Date(2026, 9, 20, 9, 5);
      const end = new Date(2026, 9, 20, 9, 5);
      assert.equal(formatIntervalTime(start, end, base), '10/20 09:05');
    });
  });

  describe('getCardDisplayTime', () => {
    const base = new Date(2026, 9, 20, 12, 0, 0);

    it('returns empty string for null or non-object card', () => {
      assert.equal(getCardDisplayTime(null, base), '');
      assert.equal(getCardDisplayTime(undefined, base), '');
      assert.equal(getCardDisplayTime('not-a-card' as any, base), '');
    });

    it('formats thought card using createdAt or spawnTime', () => {
      const card1 = {
        type: 'thought',
        createdAt: new Date(2026, 9, 20, 11, 58, 0)
      };
      assert.equal(getCardDisplayTime(card1, base), '刚刚');

      const card2 = {
        type: 'thought',
        spawnTime: new Date(2026, 9, 19, 12, 0, 0).getTime()
      };
      assert.equal(getCardDisplayTime(card2, base), '昨天 12:00');
    });

    it('formats babel card using publishedAt or fallback', () => {
      const babelCard = {
        type: 'babel',
        publishedAt: new Date(2026, 9, 20, 10, 0, 0).toISOString()
      };
      assert.equal(getCardDisplayTime(babelCard, base), '今天 10:00');

      const babelOld = {
        type: 'babel',
        publishedAt: new Date(2024, 2, 15, 10, 0, 0).toISOString()
      };
      assert.equal(getCardDisplayTime(babelOld, base), '2024/03/15');

      const babelRaw = {
        type: 'babel',
        publishedAt: 'NonDate string'
      };
      assert.equal(getCardDisplayTime(babelRaw, base), 'NonDate string');
    });

    it('formats chat_log card using startedAt and endedAt interval', () => {
      const chatCard = {
        type: 'chat_log',
        startedAt: '2026-10-20T09:00:00',
        endedAt: '2026-10-20T10:30:00'
      };
      // Chat interval same day
      const dStart = new Date(2026, 9, 20, 9, 0);
      const dEnd = new Date(2026, 9, 20, 10, 30);
      const expected = formatIntervalTime(dStart, dEnd, base);
      assert.equal(
        getCardDisplayTime(
          chatCard,
          base
        ),
        expected
      );
    });

    it('falls back to createdAt or spawnTime for chat_log cards without interval', () => {
      const chatCard = {
        type: 'chat_log',
        createdAt: new Date(2026, 9, 20, 11, 57, 0)
      };
      assert.equal(getCardDisplayTime(chatCard, base), '刚刚');
    });

    it('handles easter egg on cards from previous years', () => {
      const cardEasterEgg = {
        type: 'thought',
        createdAt: new Date(2025, 9, 20, 10, 0, 0)
      };
      assert.equal(getCardDisplayTime(cardEasterEgg, base), '1 年前的今天');
    });
  });

  describe('Focus-Driven Stage Time Store & Relative Time Transitions', () => {
    it('updates stageTime store when refreshStageTime is called', () => {
      let currentTime = 0;
      const unsubscribe = stageTime.subscribe((val) => {
        currentTime = val;
      });

      const targetTime = new Date(2026, 9, 20, 15, 30, 0).getTime();
      refreshStageTime(targetTime);
      assert.equal(currentTime, targetTime);

      const targetTime2 = new Date(2026, 9, 21, 9, 0, 0).getTime();
      refreshStageTime(targetTime2);
      assert.equal(currentTime, targetTime2);

      unsubscribe();
    });

    it('dynamically transitions card display time across focus intervals ("刚刚" -> "今天" -> "昨天" -> "前天" -> "3 天前")', () => {
      let currentStageTime = 0;
      const unsubscribe = stageTime.subscribe((val) => {
        currentStageTime = val;
      });

      const card = {
        id: 'test-card-1',
        type: 'thought',
        createdAt: new Date(2026, 9, 20, 10, 0, 0) // Oct 20, 10:00
      };

      // 1. Initial creation / focus at 10:02 (2 minutes after creation) -> "刚刚"
      refreshStageTime(new Date(2026, 9, 20, 10, 2, 0).getTime());
      assert.equal(getCardDisplayTime(card, currentStageTime), '刚刚');

      // 2. Window focus after 15 minutes (10:15) -> "今天 10:00"
      refreshStageTime(new Date(2026, 9, 20, 10, 15, 0).getTime());
      assert.equal(getCardDisplayTime(card, currentStageTime), '今天 10:00');

      // 3. Window focus next morning across midnight (Oct 21, 10:30, 24.5 hours later) -> "昨天 10:00"
      refreshStageTime(new Date(2026, 9, 21, 10, 30, 0).getTime());
      assert.equal(getCardDisplayTime(card, currentStageTime), '昨天 10:00');

      // 4. Window focus 2 calendar days later (Oct 22, 14:00) -> "前天 10:00"
      refreshStageTime(new Date(2026, 9, 22, 14, 0, 0).getTime());
      assert.equal(getCardDisplayTime(card, currentStageTime), '前天 10:00');

      // 5. Window focus 3 calendar days later (Oct 23, 11:00) -> "3 天前 上午"
      refreshStageTime(new Date(2026, 9, 23, 11, 0, 0).getTime());
      assert.equal(getCardDisplayTime(card, currentStageTime), '3 天前 上午');

      // 6. Window focus exactly 1 year later (Oct 20, 2027) -> "1 年前的今天"
      refreshStageTime(new Date(2027, 9, 20, 12, 0, 0).getTime());
      assert.equal(getCardDisplayTime(card, currentStageTime), '1 年前的今天');

      unsubscribe();
    });
  });
});
