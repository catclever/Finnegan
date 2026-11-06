/**
 * Finnegan Cosmic Stage - Diary Scroll Rendering Engine
 * Task: frontend_diary_parchment_scroll_and_date_picker_162
 */

export interface DiaryCard {
  id: string;
  type: string;
  createdAt?: string | number; // timestamp
  weather?: {
    observedHour?: string | number; // timestamp or ISO
    condition?: string; // 'sun', 'moon', 'stars', 'clouds', 'rain', 'snow', 'fog'
  };
  location?: {
    name?: string;
  };
  content?: string;
  filePath?: string;
  images?: string[];
  tags?: string[];
  [key: string]: any;
}

/**
 * Extracts the local calendar date YYYY-MM-DD from the card.
 */
export function getDiaryDateKey(card: DiaryCard): string {
  let ts = Date.now();
  if (card.createdAt) {
    ts = typeof card.createdAt === 'string' ? new Date(card.createdAt).getTime() : card.createdAt;
  } else if (card.weather?.observedHour) {
    ts = typeof card.weather.observedHour === 'string'
      ? new Date(card.weather.observedHour).getTime()
      : card.weather.observedHour;
  }

  const d = new Date(ts);
  if (isNaN(d.getTime())) return new Date().toISOString().split('T')[0];

  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Filters all cards, getting diary cards matching target date, deduplicated and sorted.
 * strict chronological ascending order (正序，从清晨到深夜: earliest createdAt first)
 */
export function getSameDayDiaryCards(targetDateKey: string, allCards: DiaryCard[]): DiaryCard[] {
  const matching = allCards.filter(
    (c) => c.type === 'diary' && getDiaryDateKey(c) === targetDateKey
  );

  // Deduplicate by card.id
  const dedupMap = new Map<string, DiaryCard>();
  for (const c of matching) {
    if (c.id && !dedupMap.has(c.id)) {
      dedupMap.set(c.id, c);
    }
  }

  const deduped = Array.from(dedupMap.values());

  // Sort chronological ascending
  deduped.sort((a, b) => {
    let tsA = 0;
    if (a.createdAt) tsA = typeof a.createdAt === 'string' ? new Date(a.createdAt).getTime() : a.createdAt;

    let tsB = 0;
    if (b.createdAt) tsB = typeof b.createdAt === 'string' ? new Date(b.createdAt).getTime() : b.createdAt;

    return tsA - tsB;
  });

  return deduped;
}

/**
 * Returns a map of YYYY-MM-DD -> entryCount for all dates that have diary cards.
 */
export function getAvailableDiaryDateMap(allCards: DiaryCard[]): Map<string, number> {
  const map = new Map<string, number>();
  for (const c of allCards) {
    if (c.type === 'diary') {
      const key = getDiaryDateKey(c);
      map.set(key, (map.get(key) || 0) + 1);
    }
  }
  return map;
}

/**
 * Generates an SVG descriptor based on time and weather.
 * Used for the DiaryCardBody Time-Weather Celestial SVG Icon.
 */
export function computeCelestialWeatherGlyph(timestamp: number | string | undefined, weatherCondition: string | undefined): string {
  let ts = Date.now();
  if (timestamp) {
    ts = typeof timestamp === 'string' ? new Date(timestamp).getTime() : timestamp;
  }

  const d = new Date(ts);
  const hour = d.getHours();

  let timePeriod = 'midnight';
  if (hour >= 5 && hour < 8) timePeriod = 'dawn';
  else if (hour >= 8 && hour < 12) timePeriod = 'morning';
  else if (hour >= 12 && hour < 17) timePeriod = 'noon';
  else if (hour >= 17 && hour < 19) timePeriod = 'dusk';
  else if (hour >= 19 && hour < 23) timePeriod = 'evening';

  const condition = weatherCondition?.toLowerCase() || 'sun';

  // Mapping conditions to SVG paths/emoji for simplicity in generating self-contained icons
  // A robust implementation could return actual SVG string
  let icon = '☀️'; // Default sun

  if (condition.includes('cloud')) icon = '☁️';
  else if (condition.includes('rain')) icon = '🌧️';
  else if (condition.includes('snow')) icon = '❄️';
  else if (condition.includes('fog')) icon = '🌫️';
  else if (timePeriod === 'evening' || timePeriod === 'midnight' || timePeriod === 'dawn') {
      if (condition.includes('moon')) icon = '🌙';
      else if (condition.includes('star')) icon = '✨';
      else icon = '🌙';
  }

  // To build an SVG, we just return a self-contained SVG string wrapping the emoji/text
  // or a basic shape to comply with "SVG descriptor generator" requirement.
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="24" height="24">
    <text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" font-size="20">${icon}</text>
  </svg>`;

  return svg;
}
