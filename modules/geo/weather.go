package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrFutureTimeNotAllowed = errors.New("cannot fetch weather for future time")

type WeatherSnapshot struct {
	Condition      string  `json:"condition" yaml:"condition"`
	WMOCode        int     `json:"wmoCode" yaml:"wmo_code"`
	Icon           string  `json:"icon" yaml:"icon"`
	TempC          float64 `json:"tempC" yaml:"temp_c"`
	Humidity       int     `json:"humidity,omitempty" yaml:"humidity,omitempty"`
	WindSpeedKmh   float64 `json:"windSpeedKmh,omitempty" yaml:"wind_speed_kmh,omitempty"`
	ObservedHour   string  `json:"observedHour" yaml:"observed_hour"`
	IsDay          bool    `json:"isDay" yaml:"is_day"`
}

func mapWMOCodeToCondition(code int, isDay bool) (string, string) {
	switch {
	case code == 0:
		if isDay {
			return "晴", "sun"
		}
		return "晴", "moon"
	case code == 1 || code == 2 || code == 3:
		if isDay {
			return "多云", "cloud-sun"
		}
		return "多云", "cloud-moon"
	case code >= 45 && code <= 48:
		return "雾", "smog"
	case code >= 51 && code <= 55, code >= 56 && code <= 57:
		return "毛毛雨", "cloud-rain"
	case code >= 61 && code <= 65, code >= 66 && code <= 67:
		return "雨", "cloud-showers-heavy"
	case code >= 71 && code <= 75, code == 77:
		return "雪", "snowflake"
	case code >= 80 && code <= 82:
		return "阵雨", "cloud-showers-water"
	case code >= 85 && code <= 86:
		return "阵雪", "snowflake"
	case code >= 95 && code <= 99:
		return "雷阵雨", "bolt"
	default:
		return "未知", "cloud"
	}
}

func GetWeatherAtHour(ctx context.Context, lat, lng float64, targetTime time.Time) (*WeatherSnapshot, error) {
	now := time.Now().Truncate(time.Hour)
	tgt := targetTime.Truncate(time.Hour)

	if tgt.After(now) {
		return nil, ErrFutureTimeNotAllowed
	}

	daysAgo := int(now.Sub(tgt).Hours() / 24)

	var url string
	if daysAgo > 90 {
		startDate := tgt.Format("2006-01-02")
		endDate := tgt.Add(24 * time.Hour).Format("2006-01-02")
		url = fmt.Sprintf("https://archive-api.open-meteo.com/v1/archive?latitude=%f&longitude=%f&start_date=%s&end_date=%s&hourly=temperature_2m,weathercode,relativehumidity_2m,windspeed_10m,is_day&timezone=UTC", lat, lng, startDate, endDate)
	} else {
		url = fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&hourly=temperature_2m,weathercode,relativehumidity_2m,windspeed_10m,is_day&past_days=%d&forecast_days=1&timezone=UTC", lat, lng, daysAgo+1)
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Hourly struct {
			Time               []string  `json:"time"`
			Temperature2m      []float64 `json:"temperature_2m"`
			Weathercode        []int     `json:"weathercode"`
			Relativehumidity2m []int     `json:"relativehumidity_2m"`
			Windspeed10m       []float64 `json:"windspeed_10m"`
			IsDay              []int     `json:"is_day"`
		} `json:"hourly"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	targetStr := tgt.UTC().Format("2006-01-02T15:00")

	for i, tStr := range data.Hourly.Time {
		if tStr == targetStr {
			wmo := data.Hourly.Weathercode[i]
			isDay := data.Hourly.IsDay[i] == 1
			cond, icon := mapWMOCodeToCondition(wmo, isDay)

			return &WeatherSnapshot{
				Condition:    cond,
				WMOCode:      wmo,
				Icon:         icon,
				TempC:        data.Hourly.Temperature2m[i],
				Humidity:     data.Hourly.Relativehumidity2m[i],
				WindSpeedKmh: data.Hourly.Windspeed10m[i],
				ObservedHour: tgt.Format(time.RFC3339),
				IsDay:        isDay,
			}, nil
		}
	}

	return nil, fmt.Errorf("weather data not found for the requested hour")
}
