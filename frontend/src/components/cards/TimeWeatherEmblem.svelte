<script lang="ts">
  import type { DiaryWeather } from '../../types/card';
  import { resolveTimeWeatherEmblem } from '../../lib/diaryGeo';

  export let timestamp: string | number | Date = new Date();
  export let weather: DiaryWeather | null = null;
  export let size: 'sm' | 'md' | 'lg' = 'md';
  export let showTooltip: boolean = true;

  $: emblemData = resolveTimeWeatherEmblem(timestamp, weather);

  $: formattedHour = new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  $: tempString = weather && weather.tempC !== undefined ? `${weather.tempC}°C` : '';
  $: tooltipText = `${formattedHour} · ${emblemData.descriptor}${tempString ? ` · ${tempString}` : ''}`;

</script>

<div class="time-weather-emblem {size === 'sm' ? 'size-sm' : size === 'lg' ? 'size-lg' : 'size-md'}" title={showTooltip ? tooltipText : undefined}>
  <svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
    <defs>
      <!-- Common Gradients -->
      <linearGradient id="sky-dawn" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#ffb347"/>
        <stop offset="100%" stop-color="#ffcc33"/>
      </linearGradient>
      <linearGradient id="sky-morning" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#87CEEB"/>
        <stop offset="100%" stop-color="#E0F6FF"/>
      </linearGradient>
      <linearGradient id="sky-noon" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#4facfe"/>
        <stop offset="100%" stop-color="#00f2fe"/>
      </linearGradient>
      <linearGradient id="sky-dusk" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#ff7b00"/>
        <stop offset="100%" stop-color="#ffb347"/>
      </linearGradient>
      <linearGradient id="sky-evening" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#2c3e50"/>
        <stop offset="100%" stop-color="#3498db"/>
      </linearGradient>
      <linearGradient id="sky-midnight" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#141E30"/>
        <stop offset="100%" stop-color="#243B55"/>
      </linearGradient>

      <!-- Sun/Moon Gradients -->
      <radialGradient id="sun-glow">
        <stop offset="0%" stop-color="#FFF5EE"/>
        <stop offset="50%" stop-color="#FFD700"/>
        <stop offset="100%" stop-color="#FFA500" stop-opacity="0"/>
      </radialGradient>
      <radialGradient id="moon-glow">
        <stop offset="0%" stop-color="#FFFFFF"/>
        <stop offset="70%" stop-color="#F0F8FF"/>
        <stop offset="100%" stop-color="#B0C4DE" stop-opacity="0"/>
      </radialGradient>

      <!-- Cloud Filter -->
      <filter id="cloud-blur" x="-20%" y="-20%" width="140%" height="140%">
        <feGaussianBlur stdDeviation="2"/>
      </filter>
    </defs>

    <!-- Background Layer (Diurnal Phase) -->
    <circle cx="50" cy="50" r="48" fill="url(#sky-{emblemData.diurnalPhase})" stroke="#c0a060" stroke-width="2" />

    <!-- Orbital Position Arc (subtle) -->
    <path d="M 10 50 A 40 40 0 0 1 90 50" fill="none" stroke="rgba(255,255,255,0.2)" stroke-width="1" stroke-dasharray="2,2"/>

    <!-- Celestial Body -->
    {#if ['dawn', 'morning', 'noon', 'dusk'].includes(emblemData.diurnalPhase)}
      <!-- Sun Position logic mapping loosely to arc -->
      {@const sunCY = emblemData.diurnalPhase === 'noon' ? 25 : (emblemData.diurnalPhase === 'dawn' || emblemData.diurnalPhase === 'dusk' ? 65 : 40)}
      {@const sunCX = emblemData.diurnalPhase === 'dawn' ? 25 : (emblemData.diurnalPhase === 'dusk' ? 75 : 50)}
      <circle cx={sunCX} cy={sunCY} r="15" fill="url(#sun-glow)" />
      <circle cx={sunCX} cy={sunCY} r="8" fill="#FFD700" />
    {:else}
      <!-- Moon -->
      <path d="M 60 30 A 15 15 0 1 0 70 50 A 12 12 0 1 1 60 30" fill="url(#moon-glow)" />
      <!-- Stars for midnight/evening -->
      {#if emblemData.weatherType === 'clear' || emblemData.weatherType === 'unknown'}
        <circle cx="20" cy="30" r="1" fill="#FFF" opacity="0.8"/>
        <circle cx="35" cy="20" r="1.5" fill="#FFF" opacity="0.6"/>
        <circle cx="80" cy="40" r="1" fill="#FFF" opacity="0.7"/>
        <circle cx="50" cy="15" r="1" fill="#FFF" opacity="0.9"/>
      {/if}
    {/if}

    <!-- Weather Layer -->
    {#if emblemData.weatherType === 'cloudy' || emblemData.weatherType === 'rain' || emblemData.weatherType === 'storm' || emblemData.weatherType === 'snow'}
      <!-- Base Cloud -->
      <g fill="rgba(255,255,255,0.7)" filter="url(#cloud-blur)">
        <circle cx="35" cy="55" r="15" />
        <circle cx="50" cy="45" r="20" />
        <circle cx="65" cy="55" r="15" />
        <rect x="35" y="50" width="30" height="20" rx="10" />
      </g>
      <!-- Darker cloud for rain/storm -->
      {#if emblemData.weatherType === 'rain' || emblemData.weatherType === 'storm'}
        <g fill="rgba(100,110,120,0.6)" filter="url(#cloud-blur)">
          <circle cx="35" cy="55" r="12" />
          <circle cx="50" cy="45" r="18" />
          <circle cx="65" cy="55" r="12" />
        </g>
      {/if}
    {/if}

    {#if emblemData.weatherType === 'rain' || emblemData.weatherType === 'storm'}
      <g stroke="#87CEEB" stroke-width="2" stroke-linecap="round">
        <line x1="35" y1="70" x2="30" y2="80" />
        <line x1="50" y1="75" x2="45" y2="85" />
        <line x1="65" y1="70" x2="60" y2="80" />
      </g>
    {/if}

    {#if emblemData.weatherType === 'storm'}
      <path d="M 50 60 L 45 75 L 52 75 L 48 90 L 60 70 L 53 70 Z" fill="#FFD700" />
    {/if}

    {#if emblemData.weatherType === 'snow'}
      <g fill="#FFF">
        <circle cx="35" cy="75" r="2" />
        <circle cx="50" cy="80" r="2.5" />
        <circle cx="65" cy="75" r="2" />
        <circle cx="42" cy="70" r="1.5" />
        <circle cx="58" cy="70" r="1.5" />
      </g>
    {/if}

    {#if emblemData.weatherType === 'fog'}
      <g stroke="rgba(255,255,255,0.6)" stroke-width="4" stroke-linecap="round">
        <line x1="20" y1="60" x2="80" y2="60" />
        <line x1="30" y1="70" x2="70" y2="70" />
        <line x1="40" y1="80" x2="60" y2="80" />
      </g>
    {/if}

    {#if emblemData.weatherType === 'wind'}
      <g stroke="rgba(255,255,255,0.7)" stroke-width="2" fill="none">
        <path d="M 20 60 Q 40 50 60 60 T 80 50" />
        <path d="M 30 75 Q 50 65 70 75" />
      </g>
    {/if}

  </svg>
</div>

<style>
  .time-weather-emblem {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
    transition: transform 0.2s ease;
  }
  .size-sm {
    width: 24px;
    height: 24px;
  }
  .size-md {
    width: 40px;
    height: 40px;
  }
  .size-lg {
    width: 64px;
    height: 64px;
  }
  svg {
    width: 100%;
    height: 100%;
  }
  .time-weather-emblem:hover {
    transform: scale(1.05);
  }
</style>