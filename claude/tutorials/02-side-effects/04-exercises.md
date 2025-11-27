# Exercises: Practice Side Effects and `restate.Run`

> **Hands-on exercises to master durable side effects**

## 🎯 Objectives

Practice:
- Wrapping different types of side effects
- Handling errors in Run blocks
- Building resilient services with external dependencies
- Implementing retry strategies

## 📚 Exercise Levels

- 🟢 **Beginner** - Direct application
- 🟡 **Intermediate** - Combining concepts
- 🔴 **Advanced** - Complex scenarios

---

## Exercise 1: Add Temperature Conversion 🟢

**Goal:** Add a handler that converts temperature units using `restate.Run`

### Requirements

1. Create a new handler `GetWeatherInFahrenheit`
2. Call the existing `GetWeather` (using `restate.Service`)
3. Convert Celsius to Fahrenheit in a Run block (simulate external conversion API)

### Starter Code

```go
// Add to service.go

// Mock external temperature conversion API
func convertCelsiusToFahrenheit(celsius float64) (float64, error) {
    time.Sleep(50 * time.Millisecond) // Simulate API delay
    
    if rand.Float64() < 0.05 { // 5% failure
        return 0, fmt.Errorf("conversion API unavailable")
    }
    
    return celsius*9/5 + 32, nil
}

func (WeatherService) GetWeatherInFahrenheit(
    ctx restate.Context,
    req WeatherRequest,
) (AggregatedWeather, error) {
    // TODO: Call GetWeather
    celsiusData, err := restate.Service[AggregatedWeather](
        ctx, "WeatherService", "GetWeather",
    ).Request(req)
    if err != nil {
        return AggregatedWeather{}, err
    }
    
    // TODO: Convert temperature using restate.Run
    fahrenheit, err := restate.Run(ctx, func(rc restate.RunContext) (float64, error) {
        return convertCelsiusToFahrenheit(celsiusData.AverageTemp)
    })
    if err != nil {
        return AggregatedWeather{}, err
    }
    
    // TODO: Update the result with Fahrenheit temperature
    celsiusData.AverageTemp = fahrenheit
    return celsiusData, nil
}
```

### Test

```bash
curl -X POST http://localhost:9080/WeatherService/GetWeatherInFahrenheit \
  -H 'Content-Type: application/json' \
  -d '{"city": "London"}'

# Should return temperature in Fahrenheit
```

---

## Exercise 2: Add Caching with TTL 🟡

**Goal:** Implement a simple cache to avoid calling APIs too frequently

### Requirements

1. Use a global map to cache results (key: city, value: result + timestamp)
2. Check cache before calling APIs
3. If cache hit and < 5 minutes old, return cached data
4. Otherwise, fetch fresh data and update cache
5. Use `restate.Run` to wrap cache operations

### Starter Code

```go
import "sync"

var (
    weatherCache = make(map[string]CachedWeather)
    cacheMutex   sync.RWMutex
)

type CachedWeather struct {
    Data      AggregatedWeather
    Timestamp time.Time
}

func (WeatherService) GetWeatherCached(
    ctx restate.Context,
    req WeatherRequest,
) (AggregatedWeather, error) {
    // TODO: Check cache in restate.Run
    cached, err := restate.Run(ctx, func(rc restate.RunContext) (CachedWeather, error) {
        cacheMutex.RLock()
        defer cacheMutex.RUnlock()
        
        if data, ok := weatherCache[req.City]; ok {
            // Check if still valid (< 5 minutes)
            if time.Since(data.Timestamp) < 5*time.Minute {
                return data, nil
            }
        }
        
        return CachedWeather{}, fmt.Errorf("cache miss")
    })
    
    if err == nil {
        ctx.Log().Info("Cache hit", "city", req.City)
        return cached.Data, nil
    }
    
    // Cache miss - fetch fresh data
    ctx.Log().Info("Cache miss, fetching fresh data", "city", req.City)
    
    // TODO: Call GetWeather to fetch fresh data
    freshData, err := restate.Service[AggregatedWeather](
        ctx, "WeatherService", "GetWeather",
    ).Request(req)
    if err != nil {
        return AggregatedWeather{}, err
    }
    
    // TODO: Update cache in restate.Run
    _, err = restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
        cacheMutex.Lock()
        defer cacheMutex.Unlock()
        
        weatherCache[req.City] = CachedWeather{
            Data:      freshData,
            Timestamp: time.Now(),
        }
        return true, nil
    })
    
    return freshData, nil
}
```

### Test

```bash
# First call - cache miss
curl -X POST http://localhost:9080/WeatherService/GetWeatherCached \
  -H 'Content-Type: application/json' \
  -d '{"city": "Paris"}'

# Second call within 5 min - cache hit
curl -X POST http://localhost:9080/WeatherService/GetWeatherCached \
  -H 'Content-Type: application/json' \
  -d '{"city": "Paris"}'

# Check logs - second call should show "Cache hit"
```

**⚠️ Note:** Global cache is NOT durable! For production, use Virtual Objects (Module 4).

---

## Exercise 3: Add Retry Logic with Backoff 🟡

**Goal:** Implement custom retry logic for API calls

### Requirements

1. Create a helper function that retries API calls with exponential backoff
2. Max 3 attempts
3. Backoff: 1s, 2s, 4s
4. Each attempt wrapped in `restate.Run`
5. Return error if all attempts fail

### Starter Code

```go
func fetchWeatherWithRetry(
    ctx restate.Context,
    city string,
    apiCall func(string) (WeatherData, error),
    apiName string,
) (WeatherData, error) {
    maxAttempts := 3
    var lastErr error
    
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        ctx.Log().Info("Attempting API call",
            "api", apiName,
            "attempt", attempt)
        
        // TODO: Wrap API call in restate.Run
        data, err := restate.Run(ctx, func(rc restate.RunContext) (WeatherData, error) {
            return apiCall(city)
        })
        
        if err == nil {
            ctx.Log().Info("API call succeeded",
                "api", apiName,
                "attempt", attempt)
            return data, nil
        }
        
        lastErr = err
        ctx.Log().Warn("API call failed",
            "api", apiName,
            "attempt", attempt,
            "error", err)
        
        // Backoff before retry (except on last attempt)
        if attempt < maxAttempts {
            backoff := time.Duration(attempt) * time.Second
            ctx.Log().Info("Backing off", "duration", backoff)
            restate.Sleep(ctx, backoff)
        }
    }
    
    return WeatherData{}, fmt.Errorf("all %d attempts failed: %w",
        maxAttempts, lastErr)
}

func (WeatherService) GetWeatherWithRetry(
    ctx restate.Context,
    req WeatherRequest,
) (AggregatedWeather, error) {
    var sources []WeatherData
    
    // TODO: Use fetchWeatherWithRetry for each API
    data1, err := fetchWeatherWithRetry(ctx, req.City, MockWeatherAPI1, "API1")
    if err == nil {
        sources = append(sources, data1)
    }
    
    // ... similar for API2 and API3
    
    // Rest of aggregation logic...
}
```

### Test

```bash
# This may retry if APIs fail
curl -X POST http://localhost:9080/WeatherService/GetWeatherWithRetry \
  -H 'Content-Type: application/json' \
  -d '{"city": "Berlin"}'

# Watch logs for retry attempts and backoff
```

---

## Exercise 4: Parallel API Calls 🔴

**Goal:** Call all three weather APIs in parallel (we'll learn this better in Module 3!)

### Requirements

1. Use `RequestFuture` to call all APIs concurrently
2. Each wrapped in `restate.Run`
3. Wait for all to complete
4. Aggregate successful results

### Starter Code

```go
func (WeatherService) GetWeatherParallel(
    ctx restate.Context,
    req WeatherRequest,
) (AggregatedWeather, error) {
    ctx.Log().Info("Fetching weather in parallel", "city", req.City)
    
    // Create async tasks for each API
    future1 := restate.RunAsync(ctx, func(rc restate.RunContext) (WeatherData, error) {
        return MockWeatherAPI1(req.City)
    })
    
    future2 := restate.RunAsync(ctx, func(rc restate.RunContext) (WeatherData, error) {
        return MockWeatherAPI2(req.City)
    })
    
    future3 := restate.RunAsync(ctx, func(rc restate.RunContext) (WeatherData, error) {
        return MockWeatherAPI3(req.City)
    })
    
    // Wait for all to complete
    var sources []WeatherData
    
    // TODO: Use restate.Wait to collect results
    for fut, err := range restate.Wait(ctx, future1, future2, future3) {
        if err != nil {
            ctx.Log().Warn("API failed", "error", err)
            continue
        }
        
        switch fut {
        case future1:
            if data, err := future1.Result(); err == nil {
                sources = append(sources, data)
            }
        case future2:
            if data, err := future2.Result(); err == nil {
                sources = append(sources, data)
            }
        case future3:
            if data, err := future3.Result(); err == nil {
                sources = append(sources, data)
            }
        }
    }
    
    // Aggregate results...
    if len(sources) == 0 {
        return AggregatedWeather{}, fmt.Errorf("all APIs failed")
    }
    
    var totalTemp float64
    for _, s := range sources {
        totalTemp += s.Temperature
    }
    
    return AggregatedWeather{
        City:           req.City,
        Sources:        sources,
        AverageTemp:    totalTemp / float64(len(sources)),
        SuccessfulAPIs: len(sources),
    }, nil
}
```

**Note:** We'll cover concurrency patterns more in Module 3!

---

## Exercise 5: Database Integration 🔴

**Goal:** Store weather results in a mock database

### Requirements

1. Create mock database operations (read/write)
2. Wrap all DB operations in `restate.Run`
3. Store each weather fetch in the "database"
4. Add a handler to retrieve historical data

### Starter Code

```go
// Mock database
var weatherDB = make(map[string][]AggregatedWeather)
var dbMutex sync.RWMutex

// Mock DB write (simulate external DB)
func saveToDatabase(city string, data AggregatedWeather) error {
    time.Sleep(30 * time.Millisecond) // Simulate DB latency
    
    if rand.Float64() < 0.05 { // 5% failure
        return fmt.Errorf("database write failed")
    }
    
    dbMutex.Lock()
    defer dbMutex.Unlock()
    
    weatherDB[city] = append(weatherDB[city], data)
    return nil
}

// Mock DB read
func readFromDatabase(city string) ([]AggregatedWeather, error) {
    time.Sleep(20 * time.Millisecond)
    
    if rand.Float64() < 0.05 {
        return nil, fmt.Errorf("database read failed")
    }
    
    dbMutex.RLock()
    defer dbMutex.RUnlock()
    
    return weatherDB[city], nil
}

func (WeatherService) GetAndStoreWeather(
    ctx restate.Context,
    req WeatherRequest,
) (AggregatedWeather, error) {
    // Get weather data
    data, err := restate.Service[AggregatedWeather](
        ctx, "WeatherService", "GetWeather",
    ).Request(req)
    if err != nil {
        return AggregatedWeather{}, err
    }
    
    // TODO: Save to database using restate.Run
    _, err = restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
        return true, saveToDatabase(req.City, data)
    })
    if err != nil {
        ctx.Log().Error("Failed to save to DB", "error", err)
        // Continue anyway - don't fail the request
    }
    
    return data, nil
}

func (WeatherService) GetWeatherHistory(
    ctx restate.Context,
    req WeatherRequest,
) ([]AggregatedWeather, error) {
    // TODO: Read from database using restate.Run
    history, err := restate.Run(ctx, func(rc restate.RunContext) ([]AggregatedWeather, error) {
        return readFromDatabase(req.City)
    })
    
    return history, err
}
```

### Test

```bash
# Store weather data
curl -X POST http://localhost:9080/WeatherService/GetAndStoreWeather \
  -H 'Content-Type: application/json' \
  -d '{"city": "Tokyo"}'

# Call a few more times
# ...

# Retrieve history
curl -X POST http://localhost:9080/WeatherService/GetWeatherHistory \
  -H 'Content-Type: application/json' \
  -d '{"city": "Tokyo"}'

# Should return array of historical weather data
```

---

## ✅ Exercise Checklist

- [ ] Exercise 1: Temperature Conversion (Beginner)
- [ ] Exercise 2: Caching with TTL (Intermediate)
- [ ] Exercise 3: Retry with Backoff (Intermediate)
- [ ] Exercise 4: Parallel API Calls (Advanced)
- [ ] Exercise 5: Database Integration (Advanced)

## 📁 Solutions

Complete solutions are available in [solutions/](./solutions/):

- `exercise1.go` - Temperature conversion
- `exercise2.go` - Caching
- `exercise3.go` - Retry logic
- `exercise4.go` - Parallel calls
- `exercise5.go` - Database integration

## 🎯 Next Module

Congratulations! You've mastered side effects and `restate.Run`.

You now understand:
- ✅ Why side effects need special handling
- ✅ How to use `restate.Run` properly
- ✅ Deterministic vs non-deterministic operations
- ✅ Building resilient services with external dependencies

Ready for the next challenge?

👉 **Continue to [Module 3: Concurrent Execution](../03-concurrency/README.md)**

Learn to orchestrate multiple concurrent operations efficiently!

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
