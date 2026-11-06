<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import type { DiaryLocation, DiaryWeather } from '../../../src/types/card';
  import TimeWeatherEmblem from '../cards/TimeWeatherEmblem.svelte';
  import {
    checkGeoPermissionForDiary,
    resolveDiaryGeoContext,
    clampToPresentHour,
    createNamedPlace
  } from '../../../src/lib/diaryGeo';

  const dispatch = createEventDispatcher();

  export let diaryTime: Date = new Date();
  export let diaryLocation: DiaryLocation | undefined = undefined;
  export let diaryWeather: DiaryWeather | undefined = undefined;

  let hasGeoPermission = false;
  let isCheckingPermission = true;

  let nearbyPlaces: DiaryLocation[] = [];
  let nearbyLandmarks: { name: string; lat: number; lng: number }[] = [];
  let isResolvingContext = false;

  let showLocationSelector = false;
  let customPlaceName = '';

  // Truncate to hour
  $: {
    if (diaryTime) {
      const { clamped, wasFuture } = clampToPresentHour(diaryTime);
      if (wasFuture) {
        diaryTime = clamped;
        dispatch('timeChange', diaryTime);
      }
    }
  }

  onMount(async () => {
    isCheckingPermission = true;
    hasGeoPermission = await checkGeoPermissionForDiary();
    isCheckingPermission = false;

    if (hasGeoPermission && !diaryLocation) {
      await updateGeoContext();
    }
  });

  async function updateGeoContext() {
    isResolvingContext = true;
    // Attempt to get current location
    if ('geolocation' in navigator) {
      navigator.geolocation.getCurrentPosition(
        async (position) => {
          const lat = position.coords.latitude;
          const lng = position.coords.longitude;

          const context = await resolveDiaryGeoContext(lat, lng, diaryTime.getTime());
          nearbyPlaces = context.places || [];
          nearbyLandmarks = context.landmarks || [];
          if (!diaryWeather && context.weather) {
            diaryWeather = context.weather;
            dispatch('weatherChange', diaryWeather);
          }
          if (!diaryLocation && (nearbyPlaces.length > 0 || nearbyLandmarks.length > 0)) {
              diaryLocation = nearbyPlaces[0] || { name: nearbyLandmarks[0].name, lat, lng };
              dispatch('locationChange', diaryLocation);
          } else if (!diaryLocation) {
             diaryLocation = { lat, lng };
             dispatch('locationChange', diaryLocation);
          }
          isResolvingContext = false;
        },
        (error) => {
          console.error("Geolocation error:", error);
          isResolvingContext = false;
        },
        { timeout: 5000 }
      );
    } else {
      isResolvingContext = false;
    }
  }

  function handleHourChange(delta: number) {
    const newTime = new Date(diaryTime);
    newTime.setHours(newTime.getHours() + delta);

    const { clamped, wasFuture } = clampToPresentHour(newTime);
    if (!wasFuture) {
      diaryTime = clamped;
      dispatch('timeChange', diaryTime);
      if (hasGeoPermission && diaryLocation?.lat && diaryLocation?.lng) {
          updateGeoContext(); // Re-fetch weather for new hour
      }
    }
  }

  async function handleSaveCustomPlace() {
      if (customPlaceName.trim() && diaryLocation?.lat && diaryLocation?.lng) {
          const place = await createNamedPlace(customPlaceName, diaryLocation.lat, diaryLocation.lng);
          diaryLocation = place;
          dispatch('locationChange', diaryLocation);
          showLocationSelector = false;
          customPlaceName = '';
      }
  }

  function selectPlace(place: DiaryLocation) {
      diaryLocation = place;
      dispatch('locationChange', diaryLocation);
      showLocationSelector = false;
  }

  function selectLandmark(landmark: { name: string; lat: number; lng: number }) {
      diaryLocation = { name: landmark.name, lat: landmark.lat, lng: landmark.lng };
      dispatch('locationChange', diaryLocation);
      showLocationSelector = false;
  }

</script>

{#if !isCheckingPermission && hasGeoPermission}
  <div class="diary-spatiotemporal-bar">
    <div class="time-weather-section">
      <button class="hour-btn" on:click={() => handleHourChange(-1)} title="Previous Hour">‹</button>

      <TimeWeatherEmblem timestamp={diaryTime} weather={diaryWeather} size="sm" />

      <div class="time-display">
        {diaryTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
      </div>

      <button class="hour-btn" on:click={() => handleHourChange(1)} title="Next Hour" disabled={clampToPresentHour(new Date(diaryTime.getTime() + 3600000)).wasFuture}>›</button>
    </div>

    <div class="location-section relative">
      <button class="location-btn" on:click={() => showLocationSelector = !showLocationSelector}>
        📍 {diaryLocation?.name || '位置未知'}
      </button>

      {#if showLocationSelector}
        <div class="location-selector-dropdown">
           <div class="custom-place-input">
              <input type="text" bind:value={customPlaceName} placeholder="命名此地..." on:keydown={(e) => e.key === 'Enter' && handleSaveCustomPlace()} />
              <button on:click={handleSaveCustomPlace}>保存</button>
           </div>

           {#if nearbyPlaces.length > 0}
             <div class="dropdown-section">
               <h4>已保存的地点</h4>
               <ul>
                 {#each nearbyPlaces as place}
                   <!-- svelte-ignore a11y-click-events-have-key-events -->
                   <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
                   <li on:click={() => selectPlace(place)}>{place.name}</li>
                 {/each}
               </ul>
             </div>
           {/if}

           {#if nearbyLandmarks.length > 0}
             <div class="dropdown-section">
               <h4>附近地标</h4>
               <ul>
                 {#each nearbyLandmarks as landmark}
                   <!-- svelte-ignore a11y-click-events-have-key-events -->
                   <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
                   <li on:click={() => selectLandmark(landmark)}>{landmark.name}</li>
                 {/each}
               </ul>
             </div>
           {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .diary-spatiotemporal-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background: var(--bg-hover, rgba(0, 0, 0, 0.05));
    border-radius: 8px;
    margin-bottom: 8px;
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
    color: var(--text-primary, #333);
    font-size: 0.9rem;
    backdrop-filter: blur(10px);
  }

  .time-weather-section {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .hour-btn {
    background: none;
    border: none;
    color: inherit;
    font-size: 1.2rem;
    cursor: pointer;
    padding: 0 4px;
    opacity: 0.7;
    transition: opacity 0.2s;
  }
  .hour-btn:hover:not(:disabled) {
    opacity: 1;
  }
  .hour-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .time-display {
    font-variant-numeric: tabular-nums;
  }

  .location-section {
    position: relative;
  }

  .location-btn {
    background: none;
    border: none;
    color: inherit;
    cursor: pointer;
    opacity: 0.8;
    transition: opacity 0.2s;
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .location-btn:hover {
    opacity: 1;
  }

  .location-selector-dropdown {
    position: absolute;
    bottom: 100%;
    right: 0;
    margin-bottom: 8px;
    background: var(--bg-dropdown, #fff);
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
    border-radius: 8px;
    padding: 8px;
    min-width: 200px;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
    z-index: 50;
  }

  .custom-place-input {
    display: flex;
    gap: 4px;
    margin-bottom: 8px;
  }
  .custom-place-input input {
    flex: 1;
    background: var(--bg-input, rgba(0, 0, 0, 0.05));
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
    color: inherit;
    padding: 4px 8px;
    border-radius: 4px;
  }
  .custom-place-input button {
    background: var(--bg-button, rgba(0, 0, 0, 0.05));
    border: none;
    color: inherit;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;
  }

  .dropdown-section h4 {
    margin: 8px 0 4px;
    font-size: 0.8rem;
    opacity: 0.6;
    text-transform: uppercase;
  }

  .dropdown-section ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .dropdown-section li {
    padding: 6px 8px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .dropdown-section li:hover {
    background: rgba(255, 255, 255, 0.1);
  }
</style>