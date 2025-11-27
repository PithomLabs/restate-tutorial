package main

import (
	"fmt"
	"math/rand"
	"time"
)

// WeatherData represents weather information from a single source
type WeatherData struct {
	Source      string  `json:"source"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
}

// MockWeatherAPI1 simulates calling OpenWeather API
// Demonstrates an external side effect that needs restate.Run
func MockWeatherAPI1(city string) (WeatherData, error) {
	// Simulate network delay
	time.Sleep(100 * time.Millisecond)

	// Simulate occasional failures (10% chance)
	if rand.Float64() < 0.1 {
		return WeatherData{}, fmt.Errorf("API1 temporarily unavailable")
	}

	return WeatherData{
		Source:      "OpenWeather",
		Temperature: 20.0 + rand.Float64()*10,
		Condition:   "Partly Cloudy",
		Humidity:    60 + rand.Intn(20),
	}, nil
}

// MockWeatherAPI2 simulates calling WeatherBit API
func MockWeatherAPI2(city string) (WeatherData, error) {
	time.Sleep(150 * time.Millisecond)

	if rand.Float64() < 0.1 {
		return WeatherData{}, fmt.Errorf("API2 temporarily unavailable")
	}

	return WeatherData{
		Source:      "WeatherBit",
		Temperature: 18.0 + rand.Float64()*12,
		Condition:   "Clear",
		Humidity:    55 + rand.Intn(25),
	}, nil
}

// MockWeatherAPI3 simulates calling Weatherstack API
func MockWeatherAPI3(city string) (WeatherData, error) {
	time.Sleep(120 * time.Millisecond)

	if rand.Float64() < 0.1 {
		return WeatherData{}, fmt.Errorf("API3 temporarily unavailable")
	}

	return WeatherData{
		Source:      "Weatherstack",
		Temperature: 19.0 + rand.Float64()*11,
		Condition:   "Sunny",
		Humidity:    58 + rand.Intn(22),
	}, nil
}
