# Hands-On: Building a Weather Aggregation Service

> **Learn to handle side effects by building a real-world API aggregation service**

## 🎯 What We're Building

A **Weather Aggregation Service** that:
- Fetches weather data from multiple mock APIs
- Uses `restate.Run` for durable external calls
- Handles partial failures gracefully
- Aggregates results into a unified response

## 📋 Prerequisites

- ✅ Completed [Module 01](../01-foundation/README.md)
- ✅ Understanding of `restate.Run` from [concepts](./01-concepts.md)
- ✅ Restate server running

## 🚀 Step-by-Step Tutorial

### Step 1: Project Setup

```bash
# Create project directory
mkdir -p ~/restate-tutorials/module02
cd ~/restate-tutorials/module02

# Initialize Go module
go mod init module02

# Install dependencies
go get github.com/restatedev/sdk-go
```

### Step 2: Create Mock Weather APIs

First, let's simulate external weather APIs.

Create `weather_apis.go`:

```go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

// Simulates calling an external weather API
// In production, this would be http.Get(...) or similar

type WeatherData struct {
	Source      string  `json:"source"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
}

// MockWeatherAPI1 simulates OpenWeather API
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

// MockWeatherAPI2 simulates WeatherBit API
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

// MockWeatherAPI3 simulates Weatherstack API
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
```

**🎓 Learning Point:** These mock APIs simulate real-world behavior:
- Network latency (sleep)
- Occasional failures (10% error rate)
- Non-deterministic results (random values)

### Step 3: Create the Weather Service (Wrong Way First!)

Let's first see what NOT to do.

Create `service_bad.go`:

```go
package main

import (
	"fmt"
	restate "github.com/restatedev/sdk-go"
)

type BadWeatherService struct{}

type WeatherRequest struct {
	City string `json:"city"`
}

type AggregatedWeather struct {
	City           string        `json:"city"`
	Sources        []WeatherData `json:"sources"`
	AverageTemp    float64       `json:"averageTemp"`
	SuccessfulAPIs int           `json:"successfulAPIs"`
}

// ❌ WRONG APPROACH - Side effects not wrapped
func (BadWeatherService) GetWeather(
	ctx restate.Context,
	req WeatherRequest,
) (AggregatedWeather, error) {
	ctx.Log().Info("Fetching weather", "city", req.City)
	
	var sources []WeatherData
	
	// ❌ PROBLEM: These API calls are NOT journaled!
	// If the handler crashes and retries, these execute again!
	
	data1, err := MockWeatherAPI1(req.City)
	if err == nil {
		sources = append(sources, data1)
	}
	
	data2, err := MockWeatherAPI2(req.City)
	if err == nil {
		sources = append(sources, data2)
	}
	
	data3, err := MockWeatherAPI3(req.City)
	if err == nil {
		sources = append(sources, data3)
	}
	
	if len(sources) == 0 {
		return AggregatedWeather{}, fmt.Errorf("all APIs failed")
	}
	
	// Calculate average
	var totalTemp float64
	for _, s := range sources {
		totalTemp += s.Temperature
	}
	avgTemp := totalTemp / float64(len(sources))
	
	return AggregatedWeather{
		City:           req.City,
		Sources:        sources,
		AverageTemp:    avgTemp,
		SuccessfulAPIs: len(sources),
	}, nil
}
```

**⚠️ Problems with This Approach:**
1. API calls execute on every retry
2. Results not journaled
3. Wastes API quota
4. Inconsistent results on replay

### Step 4: Create the Correct Service

Now let's do it properly using `restate.Run`.

Create `service.go`:

```go
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

// ✅ CORRECT APPROACH - Side effects wrapped in restate.Run
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
	
	// ✅ Fetch from API 1 - wrapped in restate.Run
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
	
	// ✅ Fetch from API 2 - wrapped in restate.Run
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
	
	// ✅ Fetch from API 3 - wrapped in restate.Run
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
```

**🎓 Key Learning Points:**

1. **Each API call wrapped in `restate.Run`**
   ```go
   data, err := restate.Run(ctx, func(rc restate.RunContext) (WeatherData, error) {
       return MockWeatherAPI1(city)
   })
   ```

2. **Logging outside Run blocks**
   ```go
   ctx.Log().Info("Fetching from API 1") // Before
   data, err := restate.Run(ctx, ...) // Side effect
   ctx.Log().Info("API 1 succeeded") // After
   ```

3. **Graceful partial failures**
   - If API 1 fails, still try API 2 and 3
   - Accept result if at least one succeeds
   - Fail and retry if all fail

4. **Deterministic aggregation**
   - Average calculation happens outside Run
   - Pure computation, no side effects

### Step 5: Create Main Entry Point

Create `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

func main() {
	restateServer := server.NewRestate()
	
	// Register the weather service
	if err := restateServer.Bind(
		restate.Reflect(WeatherService{}),
	); err != nil {
		log.Fatal("Failed to bind service:", err)
	}
	
	fmt.Println("🌤️  Starting Weather Aggregation Service on :9090...")
	fmt.Println("📝 Service: WeatherService")
	fmt.Println("📌 Handler: GetWeather")
	fmt.Println("")
	fmt.Println("Register with:")
	fmt.Println("  curl -X POST http://localhost:8080/deployments \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"uri\": \"http://localhost:9090\"}'")
	fmt.Println("")
	
	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 6: Build and Run

```bash
# Ensure dependencies are ready
go mod tidy

# Build
go build -o weather-service

# Run
./weather-service
```

### Step 7: Register with Restate

In a new terminal:

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

### Step 8: Test the Service

**Basic call:**
```bash
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -d '{"city": "London"}'
```

**Expected response:**
```json
{
  "city": "London",
  "sources": [
    {
      "source": "OpenWeather",
      "temperature": 24.5,
      "condition": "Partly Cloudy",
      "humidity": 65
    },
    {
      "source": "WeatherBit",
      "temperature": 23.8,
      "condition": "Clear",
      "humidity": 62
    },
    {
      "source": "Weatherstack",
      "temperature": 24.2,
      "condition": "Sunny",
      "humidity": 60
    }
  ],
  "averageTemp": 24.17,
  "successfulAPIs": 3
}
```

**Check the logs** in your service terminal:
```
INFO Starting weather aggregation city=London
INFO Fetching from API 1
INFO API 1 succeeded temp=24.5
INFO Fetching from API 2
INFO API 2 succeeded temp=23.8
INFO Fetching from API 3
INFO API 3 succeeded temp=24.2
INFO Weather aggregation complete successfulAPIs=3 averageTemp=24.17
```

### Step 9: Test Journaling (Advanced)

Let's verify that `restate.Run` actually journals the results.

**Call with idempotency key:**
```bash
# First call
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: test-replay-123' \
  -d '{"city": "Paris"}'

# Save the response (note the temperatures)

# Second call - same idempotency key
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: test-replay-123' \
  -d '{"city": "Paris"}'
```

**Observe:**
- Both responses are **identical** (same temperatures!)
- Service logs show API calls only on the **first** request
- Second request is served from journal (no API calls)

**Why?** Because:
1. `restate.Run` journaled the API results
2. Same idempotency key = same invocation
3. Restate replays from journal instead of re-executing

## 🎓 Understanding the Flow

### First Execution

```
Client Request → Restate → WeatherService
                    ↓
               Creates Journal:
               ┌─────────────────────────┐
               │ Entry 1: Run API1       │
               │   Result: {temp: 24.5}  │ ← Stored
               ├─────────────────────────┤
               │ Entry 2: Run API2       │
               │   Result: {temp: 23.8}  │ ← Stored
               ├─────────────────────────┤
               │ Entry 3: Run API3       │
               │   Result: {temp: 24.2}  │ ← Stored
               └─────────────────────────┘
                    ↓
               Return aggregated result
```

### On Replay (Same Invocation ID)

```
Client Request → Restate → Check Journal
                    ↓
               Journal Found:
               ┌─────────────────────────┐
               │ Entry 1: Replay         │
               │   Result: {temp: 24.5}  │ ← From journal
               ├─────────────────────────┤
               │ Entry 2: Replay         │
               │   Result: {temp: 23.8}  │ ← From journal
               ├─────────────────────────┤
               │ Entry 3: Replay         │
               │   Result: {temp: 24.2}  │ ← From journal
               └─────────────────────────┘
                    ↓
               Return same result (no API calls!)
```

## ✅ Verification Checklist

- [ ] Service starts successfully
- [ ] Can register with Restate
- [ ] GetWeather returns aggregated data
- [ ] Handles partial failures (some APIs fail)
- [ ] Service logs show all three API calls
- [ ] Idempotent calls return identical results
- [ ] Second call doesn't execute APIs (journal replay)

## 🐛 Common Issues

### All APIs Failing

If you see "all weather APIs failed" repeatedly:
- This is expected occasionally (10% failure rate per API)
- Restate will retry automatically
- Eventually, at least one API will succeed

### Different Results Each Call

If results differ on every call:
- Check you're using the same idempotency key
- Verify you're using `restate.Run` (not calling APIs directly)

## 💡 Key Takeaways

1. **Wrap Every Side Effect**
   - External API calls → `restate.Run`
   - Database queries → `restate.Run`
   - Any non-deterministic operation → `restate.Run`

2. **Logging Strategy**
   - Log before the Run block
   - Log after the Run block
   - Never log inside the Run block

3. **Partial Failures**
   - Design for resilience
   - Aggregate from successful sources
   - Only fail if all sources fail

4. **Journaling Benefits**
   - No duplicate API calls on retry
   - Consistent results for same invocation
   - Efficient replay from journal

## 🎯 Next Steps

Great! You've built a resilient service with durable side effects.

Now let's validate it thoroughly:

👉 **Continue to [Validation](./03-validation.md)**

---

**Questions?** Compare with the [complete code](./code/) or review [concepts](./01-concepts.md)!
