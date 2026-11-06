import type { DiaryWeather } from '../types/card';
import { ExecuteModuleAction } from '../../wailsjs/go/main/App.js';

export interface TimeWeatherEmblemData {
  descriptor: string;
  diurnalPhase: 'dawn' | 'morning' | 'noon' | 'dusk' | 'evening' | 'midnight';
  weatherType: 'clear' | 'cloudy' | 'rain' | 'storm' | 'snow' | 'fog' | 'wind' | 'unknown';
}

function getDiurnalPhase(hour: number): TimeWeatherEmblemData['diurnalPhase'] {
  if (hour >= 5 && hour < 7) return 'dawn';
  if (hour >= 7 && hour < 11) return 'morning';
  if (hour >= 11 && hour < 16) return 'noon';
  if (hour >= 16 && hour < 19) return 'dusk';
  if (hour >= 19 && hour < 23) return 'evening';
  return 'midnight'; // 23:00 - 04:59
}

function getWeatherType(weather?: DiaryWeather | null): TimeWeatherEmblemData['weatherType'] {
  if (!weather || !weather.condition) return 'unknown';
  const condition = weather.condition.toLowerCase();

  if (condition.includes('storm') || condition.includes('thunder') || condition.includes('雷')) return 'storm';
  if (condition.includes('rain') || condition.includes('drizzle') || condition.includes('shower') || condition.includes('雨')) return 'rain';
  if (condition.includes('snow') || condition.includes('ice') || condition.includes('sleet') || condition.includes('雪')) return 'snow';
  if (condition.includes('fog') || condition.includes('mist') || condition.includes('haze') || condition.includes('雾') || condition.includes('霾')) return 'fog';
  if (condition.includes('wind') || condition.includes('breeze') || condition.includes('gale') || condition.includes('风')) return 'wind';
  if (condition.includes('cloud') || condition.includes('overcast') || condition.includes('云') || condition.includes('阴')) return 'cloudy';
  if (condition.includes('clear') || condition.includes('sunny') || condition.includes('晴')) return 'clear';

  return 'unknown';
}

function generateDescriptor(phase: TimeWeatherEmblemData['diurnalPhase'], weatherType: TimeWeatherEmblemData['weatherType']): string {
  const descriptors: Record<string, Record<string, string>> = {
    dawn: { clear: '破晓初晴', cloudy: '破晓微云', rain: '晨雨破晓', storm: '破晓风暴', snow: '破晓飞雪', fog: '破晓晨雾', wind: '破晓清风', unknown: '破晓' },
    morning: { clear: '晨光晴朗', cloudy: '晨光微云', rain: '晨光细雨', storm: '晨光雷暴', snow: '晨光飞雪', fog: '晨光微雾', wind: '晨光清风', unknown: '晨光' },
    noon: { clear: '正午晴朗', cloudy: '正午多云', rain: '正午阵雨', storm: '正午雷暴', snow: '正午飞雪', fog: '正午迷雾', wind: '正午清风', unknown: '正午' },
    dusk: { clear: '夕照晴空', cloudy: '暮云夕照', rain: '夕照微雨', storm: '夕照风暴', snow: '夕照飞雪', fog: '夕照暮雾', wind: '夕照晚风', unknown: '夕照' },
    evening: { clear: '初夜星明', cloudy: '初夜微云', rain: '初夜细雨', storm: '初夜雷暴', snow: '初夜飞雪', fog: '初夜夜雾', wind: '初夜晚风', unknown: '初夜' },
    midnight: { clear: '深宵星夜', cloudy: '深宵暗云', rain: '夜雨残月', storm: '深宵雷暴', snow: '深宵飞雪', fog: '深宵夜雾', wind: '深宵夜风', unknown: '深宵' }
  };

  return descriptors[phase]?.[weatherType] || '未知时象';
}

export function resolveTimeWeatherEmblem(timestampOrHour: string | number | Date, weather?: DiaryWeather | null): TimeWeatherEmblemData {
  let date: Date;
  if (timestampOrHour instanceof Date) {
    date = timestampOrHour;
  } else if (typeof timestampOrHour === 'number') {
    date = new Date(timestampOrHour);
  } else {
    date = new Date(timestampOrHour);
    if (isNaN(date.getTime())) {
      date = new Date();
    }
  }

  const hour = date.getHours();
  const diurnalPhase = getDiurnalPhase(hour);
  const weatherType = getWeatherType(weather);
  const descriptor = generateDescriptor(diurnalPhase, weatherType);

  return { descriptor, diurnalPhase, weatherType };
}

export function isFutureHour(target: Date, now: Date = new Date()): boolean {
  // Truncate both to the hour boundary for comparison
  const targetHour = new Date(target);
  targetHour.setMinutes(0, 0, 0);

  const nowHour = new Date(now);
  nowHour.setMinutes(0, 0, 0);

  return targetHour.getTime() > nowHour.getTime();
}

export function clampToPresentHour(date: Date, now: Date = new Date()): { clamped: Date; wasFuture: boolean } {
  if (isFutureHour(date, now)) {
    const clamped = new Date(now);
    // keep to the top of the hour to match behavior, or maybe keep the exact minute.
    // Usually hour pickers just align to hours anyway.
    return { clamped, wasFuture: true };
  }
  return { clamped: new Date(date), wasFuture: false };
}

export async function checkGeoPermissionForDiary(): Promise<boolean> {
  try {
    if (typeof window === 'undefined' || !(window as any).go || !(window as any).go.main || !(window as any).go.main.App || !(window as any).go.main.App.ExecuteModuleAction) {
      console.warn('Wails backend binding not available for geo module. Returning true for mock.');
      return true;
    }
    const result = await ExecuteModuleAction('geo', 'check_permission', { module: 'diary', context: 'diary' });
    return Boolean(result?.allowed ?? result?.granted);
  } catch (error) {
    console.error('Error checking geo permission:', error);
    return false;
  }
}

export async function resolveDiaryGeoContext(lat: number, lng: number, timestamp: number | string | Date): Promise<any> {
  try {
    if (typeof window === 'undefined' || !(window as any).go || !(window as any).go.main || !(window as any).go.main.App || !(window as any).go.main.App.ExecuteModuleAction) {
      console.warn('Wails backend binding not available for geo module. Returning mock geo context.');
      return { places: [], landmarks: [], weather: null };
    }
    const isoTimestamp = timestamp instanceof Date
      ? timestamp.toISOString()
      : (typeof timestamp === 'number' ? new Date(timestamp).toISOString() : timestamp);
    const result = await ExecuteModuleAction('geo', 'resolve_context', {
      module: 'diary',
      lat,
      lng,
      timestamp: isoTimestamp,
    });
    return result || { places: [], landmarks: [], weather: null };
  } catch (error) {
    console.error('Error resolving diary geo context:', error);
    return { places: [], landmarks: [], weather: null };
  }
}

export async function createNamedPlace(name: string, lat: number, lng: number, landmark?: string): Promise<any> {
  try {
    if (typeof window === 'undefined' || !(window as any).go || !(window as any).go.main || !(window as any).go.main.App || !(window as any).go.main.App.ExecuteModuleAction) {
      console.warn('Wails backend binding not available for geo module. Returning mock place.');
      return { id: 'mock-place-id', name, lat, lng, landmark };
    }
    const result = await ExecuteModuleAction('geo', 'create_place', { name, lat, lng, landmark: landmark || '' });
    return result || { id: 'mock-place-id', name, lat, lng, landmark };
  } catch (error) {
    console.error('Error creating named place:', error);
    return { id: 'mock-place-id', name, lat, lng, landmark };
  }
}