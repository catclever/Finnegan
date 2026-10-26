/**
 * Finnegan Cosmic Stage - Unified Card Time Formatting Engine
 * Task: unified_card_time_engine_061
 *
 * Establishes consistent, elegant, and predictable temporal hierarchy rules
 * across all card types (Thought, Babel, ChatLog).
 */

/**
 * Pads a number with leading zeros to 2 digits.
 */
function pad2(n: number): string {
  return String(n).padStart(2, '0');
}

/**
 * Robustly parses a date from various input formats:
 * - ISO date strings
 * - Unix millisecond timestamps (numbers or numeric strings)
 * - Date objects
 * - null / undefined / empty string gracefully handled -> returns null
 */
export function parseDate(input?: string | number | Date | null): Date | null {
  if (input === null || input === undefined || input === '') {
    return null;
  }

  if (input instanceof Date) {
    return isNaN(input.getTime()) ? null : new Date(input.getTime());
  }

  if (typeof input === 'number') {
    if (isNaN(input) || !isFinite(input)) {
      return null;
    }
    const d = new Date(input);
    return isNaN(d.getTime()) ? null : d;
  }

  if (typeof input === 'string') {
    const trimmed = input.trim();
    if (!trimmed) {
      return null;
    }

    // Pure numeric string representing unix timestamp
    if (/^\d+$/.test(trimmed)) {
      const n = Number(trimmed);
      if (!isNaN(n) && isFinite(n)) {
        const d = new Date(n);
        if (!isNaN(d.getTime())) {
          return d;
        }
      }
    }

    const d = new Date(trimmed);
    return isNaN(d.getTime()) ? null : d;
  }

  return null;
}

/**
 * Maps hour of day (0-23) to intuitive atmospheric time period:
 * - 清晨: 5点 ~ 8点 (05:00 - 07:59)
 * - 上午: 8点 ~ 12点 (08:00 - 11:59)
 * - 中午: 12点 ~ 15点 (12:00 - 14:59)
 * - 下午: 15点 ~ 19点 (15:00 - 18:59)
 * - 晚上: 19点 ~ 23点 (19:00 - 22:59)
 * - 深夜: 23点 ~ 3点 (23:00 - 02:59)
 * - 凌晨: 3点 ~ 5点 (03:00 - 04:59)
 */
export function getTimePeriod(hour: number): string {
  if (hour >= 5 && hour < 8) return '清晨';
  if (hour >= 8 && hour < 12) return '上午';
  if (hour >= 12 && hour < 15) return '中午';
  if (hour >= 15 && hour < 19) return '下午';
  if (hour >= 19 && hour < 23) return '晚上';
  if (hour >= 3 && hour < 5) return '凌晨';
  return '深夜'; // 23点 ~ 3点
}

/**
 * Formats a single point in time using the Finnegan temporal hierarchy:
 * - < 5 minutes: '刚刚'
 * - < 24 hours: '今天 HH:mm'
 * - < 48 hours: '昨天 HH:mm'
 * - < 30 days: '{days} 天前 {period}' (清晨/上午/中午/下午/晚上/深夜/凌晨)
 * - 30 days to 1 year: 'MM/DD HH:mm' (padded 2 digits)
 * - > 1 year (different year):
 *   * Easter egg: If month and day match today, return '{years} 年前的今天'
 *   * Otherwise: 'YYYY/MM/DD'
 */
export function formatPointTime(
  date?: Date | string | number | null,
  baseDate?: Date | string | number | null
): string {
  const d = parseDate(date);
  if (!d) return '';

  const now = (baseDate ? parseDate(baseDate) : null) ?? new Date();
  const diffMs = now.getTime() - d.getTime();

  const MS_PER_MINUTE = 60 * 1000;
  const MS_PER_HOUR = 60 * MS_PER_MINUTE;
  const MS_PER_DAY = 24 * MS_PER_HOUR;
  const MS_PER_YEAR = 365 * MS_PER_DAY;

  // < 5 minutes (including negative drift)
  if (diffMs < 5 * MS_PER_MINUTE) {
    return '刚刚';
  }

  // < 24 hours: '今天 HH:mm'
  if (diffMs < 24 * MS_PER_HOUR) {
    return `今天 ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
  }

  // < 48 hours: '昨天 HH:mm'
  if (diffMs < 48 * MS_PER_HOUR) {
    return `昨天 ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
  }

  // < 30 days: '{days} 天前 {period}' (清晨/上午/中午/下午/晚上/深夜/凌晨)
  if (diffMs < 30 * MS_PER_DAY) {
    const days = Math.floor(diffMs / MS_PER_DAY);
    const period = getTimePeriod(d.getHours());
    return `${days} 天前 ${period}`;
  }

  // Easter egg: If in a different/previous year and month and day match today
  if (
    d.getFullYear() < now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate()
  ) {
    const years = now.getFullYear() - d.getFullYear();
    return `${years} 年前的今天`;
  }

  // 30 days to 1 year: 'MM/DD HH:mm' (padded 2 digits)
  if (diffMs < MS_PER_YEAR) {
    return `${pad2(d.getMonth() + 1)}/${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
  }

  // > 1 year (different year): 'YYYY/MM/DD'
  return `${d.getFullYear()}/${pad2(d.getMonth() + 1)}/${pad2(d.getDate())}`;
}

/**
 * Formats a start and end interval for ChatLog cards with proper zero padding
 * and handles same-day, cross-day, and cross-year cases.
 *
 * Examples:
 * - Same-day (current year): '05/12 10:00 – 11:30'
 * - Same-day (past year):    '2024/05/12 10:00 – 11:30'
 * - Cross-day (current year): '05/12 22:00 – 05/13 02:00'
 * - Cross-day (past year):   '2024/05/12 22:00 – 05/13 02:00'
 * - Cross-year:              '2025/12/31 23:00 – 2026/01/01 01:00'
 */
export function formatIntervalTime(
  start?: Date | string | number | null,
  end?: Date | string | number | null,
  baseDate?: Date | string | number | null
): string {
  const dStart = parseDate(start);
  const dEnd = parseDate(end);
  const now = (baseDate ? parseDate(baseDate) : null) ?? new Date();

  if (!dStart && !dEnd) {
    return '';
  }

  // Single date fallback
  if (!dStart || !dEnd) {
    const d = dStart || dEnd!;
    if (d.getFullYear() !== now.getFullYear()) {
      return `${d.getFullYear()}/${pad2(d.getMonth() + 1)}/${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
    }
    return `${pad2(d.getMonth() + 1)}/${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
  }

  let s = dStart;
  let e = dEnd;
  if (s.getTime() > e.getTime()) {
    [s, e] = [e, s];
  }

  const sameYear = s.getFullYear() === e.getFullYear();
  const sameDay = sameYear && s.getMonth() === e.getMonth() && s.getDate() === e.getDate();

  // Cross-year interval: YYYY/MM/DD HH:mm – YYYY/MM/DD HH:mm
  if (!sameYear) {
    return `${s.getFullYear()}/${pad2(s.getMonth() + 1)}/${pad2(s.getDate())} ${pad2(s.getHours())}:${pad2(s.getMinutes())} – ${e.getFullYear()}/${pad2(e.getMonth() + 1)}/${pad2(e.getDate())} ${pad2(e.getHours())}:${pad2(e.getMinutes())}`;
  }

  // Same-day interval
  if (sameDay) {
    if (s.getTime() === e.getTime()) {
      if (s.getFullYear() !== now.getFullYear()) {
        return `${s.getFullYear()}/${pad2(s.getMonth() + 1)}/${pad2(s.getDate())} ${pad2(s.getHours())}:${pad2(s.getMinutes())}`;
      }
      return `${pad2(s.getMonth() + 1)}/${pad2(s.getDate())} ${pad2(s.getHours())}:${pad2(s.getMinutes())}`;
    }

    if (s.getFullYear() !== now.getFullYear()) {
      return `${s.getFullYear()}/${pad2(s.getMonth() + 1)}/${pad2(s.getDate())} ${pad2(s.getHours())}:${pad2(s.getMinutes())} – ${pad2(e.getHours())}:${pad2(e.getMinutes())}`;
    }
    return `${pad2(s.getMonth() + 1)}/${pad2(s.getDate())} ${pad2(s.getHours())}:${pad2(s.getMinutes())} – ${pad2(e.getHours())}:${pad2(e.getMinutes())}`;
  }

  // Cross-day interval (same year)
  if (s.getFullYear() !== now.getFullYear()) {
    return `${s.getFullYear()}/${pad2(s.getMonth() + 1)}/${pad2(s.getDate())} ${pad2(s.getHours())}:${pad2(s.getMinutes())} – ${pad2(e.getMonth() + 1)}/${pad2(e.getDate())} ${pad2(e.getHours())}:${pad2(e.getMinutes())}`;
  }
  return `${pad2(s.getMonth() + 1)}/${pad2(s.getDate())} ${pad2(s.getHours())}:${pad2(s.getMinutes())} – ${pad2(e.getMonth() + 1)}/${pad2(e.getDate())} ${pad2(e.getHours())}:${pad2(e.getMinutes())}`;
}

/**
 * Returns the formatted display time for any card based on its card.type:
 * - 'chat_log': Formats interval using startedAt/endedAt if available, fallback to createdAt/spawnTime.
 * - 'babel': Formats publishedAt if available, fallback to createdAt/spawnTime.
 * - 'thought' / other: Formats createdAt/spawnTime.
 */
export function getCardDisplayTime(
  card: any,
  baseDate?: Date | string | number | null
): string {
  if (!card || typeof card !== 'object') {
    return '';
  }

  const type = card.type || 'thought';

  if (type === 'chat_log') {
    if (card.startedAt || card.endedAt) {
      const interval = formatIntervalTime(card.startedAt, card.endedAt, baseDate);
      if (interval) return interval;
    }
    const point = card.createdAt ?? card.spawnTime;
    return formatPointTime(point, baseDate);
  }

  if (type === 'babel') {
    const point = card.publishedAt ?? card.createdAt ?? card.spawnTime;
    const formatted = formatPointTime(point, baseDate);
    if (formatted) return formatted;
    if (typeof card.publishedAt === 'string' && card.publishedAt.trim()) {
      const s = card.publishedAt.trim();
      return s.length > 18 ? s.slice(0, 18) : s;
    }
    return '';
  }

  // Default / thought / local
  const point = card.createdAt ?? card.spawnTime;
  return formatPointTime(point, baseDate);
}
