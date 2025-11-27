package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

type WeatherService struct{}

type WeatherRequest struct {
	City string `json:"city"`
}

type AggregatedWeather struct {
	City           string        `json:"city"`
	Sources        []WeatherData `json:"sources"`
	AverageTemp    float64       `json:"averageTemp"`
	SuccessfulAPIs int           `json:"successfulAPIs"`
}

// GetWeather aggregates weather data from multiple APIs
// Demonstrates proper use of restate.Run for side effects
func (WeatherService) GetWeather(
	ctx restate.Context,
	req WeatherRequest,
) (AggregatedWeather, error) {
	ctx.Log().Info("Starting weather aggregation", "city", req.City)

	// Validate input
	if req.City == "" {
		return AggregatedWeather{}, restate.TerminalError(
			fmt.Errorf("city cannot be empty"),
			400,
		)
	}

	var sources []WeatherData

	// Fetch from API 1 - wrapped in restate.Run
	ctx.Log().Info("Fetching from API 1")
	data1, err := restate.Run(ctx, func(rc restate.RunContext) (WeatherData, error) {
		return MockWeatherAPI1(req.City)
	})

	if err != nil {
		ctx.Log().Warn("API 1 failed", "error", err)
	} else {
		sources = append(sources, data1)
		ctx.Log().Info("API 1 succeeded", "temp", data1.Temperature)
	}

	// Fetch from API 2 - wrapped in restate.Run
	ctx.Log().Info("Fetching from API 2")
	data2, err := restate.Run(ctx, func(rc restate.RunContext) (WeatherData, error) {
		return MockWeatherAPI2(req.City)
	})

	if err != nil {
		ctx.Log().Warn("API 2 failed", "error", err)
	} else {
		sources = append(sources, data2)
		ctx.Log().Info("API 2 succeeded", "temp", data2.Temperature)
	}

	// Fetch from API 3 - wrapped in restate.Run
	ctx.Log().Info("Fetching from API 3")
	data3, err := restate.Run(ctx, func(rc restate.RunContext) (WeatherData, error) {
		return MockWeatherAPI3(req.City)
	})

	if err != nil {
		ctx.Log().Warn("API 3 failed", "error", err)
	} else {
		sources = append(sources, data3)
		ctx.Log().Info("API 3 succeeded", "temp", data3.Temperature)
	}

	// Check if we got at least one result
	if len(sources) == 0 {
		return AggregatedWeather{}, fmt.Errorf("all weather APIs failed, will retry")
	}

	// Calculate average temperature (deterministic computation)
	var totalTemp float64
	for _, s := range sources {
		totalTemp += s.Temperature
	}
	avgTemp := totalTemp / float64(len(sources))

	ctx.Log().Info("Weather aggregation complete",
		"successfulAPIs", len(sources),
		"averageTemp", avgTemp)

	return AggregatedWeather{
		City:           req.City,
		Sources:        sources,
		AverageTemp:    avgTemp,
		SuccessfulAPIs: len(sources),
	}, nil
}
