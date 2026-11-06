package geo

import (
	"math"
)

const (
	earthRadiusM = 6371000.0
	ee           = 0.00669342162296594323
	a            = 6378245.0
)

// HaversineDistanceMeters calculates the spherical distance between two coordinates in meters
func HaversineDistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180.0
	lng1Rad := lng1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	lng2Rad := lng2 * math.Pi / 180.0

	dLat := lat2Rad - lat1Rad
	dLng := lng2Rad - lng1Rad

	val := math.Pow(math.Sin(dLat/2), 2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Pow(math.Sin(dLng/2), 2)
	c := 2 * math.Atan2(math.Sqrt(val), math.Sqrt(1-val))
	return earthRadiusM * c
}

// IsInChina returns true if the coordinate is broadly within China (used for GCJ-02 logic)
func IsInChina(lat, lng float64) bool {
	return lng >= 72.004 && lng <= 137.8347 && lat >= 0.8293 && lat <= 55.8271
}

// WGS84ToGCJ02 converts WGS84 coordinates to GCJ-02
func WGS84ToGCJ02(lat, lng float64) (float64, float64) {
	if !IsInChina(lat, lng) {
		return lat, lng
	}
	dLat := transformLat(lng-105.0, lat-35.0)
	dLng := transformLng(lng-105.0, lat-35.0)
	radLat := lat / 180.0 * math.Pi
	magic := math.Sin(radLat)
	magic = 1 - ee*magic*magic
	sqrtMagic := math.Sqrt(magic)
	dLat = (dLat * 180.0) / ((a * (1 - ee)) / (magic * sqrtMagic) * math.Pi)
	dLng = (dLng * 180.0) / (a / sqrtMagic * math.Cos(radLat) * math.Pi)
	mgLat := lat + dLat
	mgLng := lng + dLng
	return mgLat, mgLng
}

// GCJ02ToWGS84 converts GCJ-02 coordinates to WGS84
func GCJ02ToWGS84(lat, lng float64) (float64, float64) {
	if !IsInChina(lat, lng) {
		return lat, lng
	}
	wgsLat, wgsLng := lat, lng
	for i := 0; i < 5; i++ {
		mgLat, mgLng := WGS84ToGCJ02(wgsLat, wgsLng)
		dLat := lat - mgLat
		dLng := lng - mgLng
		wgsLat += dLat
		wgsLng += dLng
		if math.Abs(dLat) < 1e-6 && math.Abs(dLng) < 1e-6 {
			break
		}
	}
	return wgsLat, wgsLng
}

func transformLat(x, y float64) float64 {
	ret := -100.0 + 2.0*x + 3.0*y + 0.2*y*y + 0.1*x*y + 0.2*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(y*math.Pi) + 40.0*math.Sin(y/3.0*math.Pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(y/12.0*math.Pi) + 320*math.Sin(y*math.Pi/30.0)) * 2.0 / 3.0
	return ret
}

func transformLng(x, y float64) float64 {
	ret := 300.0 + x + 2.0*y + 0.1*x*x + 0.1*x*y + 0.1*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(x*math.Pi) + 40.0*math.Sin(x/3.0*math.Pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(x/12.0*math.Pi) + 300.0*math.Sin(x/30.0*math.Pi)) * 2.0 / 3.0
	return ret
}
