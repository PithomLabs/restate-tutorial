# Exercises: Practice and Extend

> **Hands-on exercises to reinforce your learning and build confidence**

## 🎯 Objectives

Practice and extend your knowledge by:
- Building variations of the greeting service
- Implementing new handlers
- Handling different error scenarios
- Making service-to-service calls

## 📚 Exercise Levels

- 🟢 **Beginner** - Direct application of concepts
- 🟡 **Intermediate** - Requires combining concepts
- 🔴 **Advanced** - Requires research and problem-solving

---

## Exercise 1: Add a Counter Handler 🟢

**Goal:** Create a handler that counts how many greetings have been sent.

### Requirements

1. Add a new handler `GetGreetingCount` to `GreetingService`
2. Track the count using a **global variable** (we'll learn better ways in Module 4!)
3. Return the current count

### Hints

```go
// Add to service.go
var greetingCount int // Global variable (not recommended for production!)

func (GreetingService) GetGreetingCount(
    ctx restate.Context,
    _ restate.Void, // No input needed
) (int, error) {
    // TODO: Return the count
}

// Update Greet to increment the counter
func (GreetingService) Greet(...) {
    greetingCount++ // Increment on each call
    // ... rest of the code
}
```

### Test

```bash
# Call Greet a few times
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "shouldFail": false}'

curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Bob", "shouldFail": false}'

# Check the count
curl -X POST http://localhost:9080/GreetingService/GetGreetingCount \
  -H 'Content-Type: application/json' \
  -d 'null'

# Should return: 2
```

### ⚠️ Important Learning Point

**What's wrong with this approach?**

The global variable `greetingCount` will:
- Reset to 0 when the service restarts
- Be lost on crashes
- Not work correctly with multiple service instances

**Better Solution:** Use a Virtual Object with state (Module 4!)

---

## Exercise 2: Language Support 🟡

**Goal:** Support multiple languages for greetings.

### Requirements

1. Add a `language` field to `GreetRequest`
2. Return greetings in different languages:
   - `en`: "Hello, {name}!"
   - `es`: "¡Hola, {name}!"
   - `fr`: "Bonjour, {name}!"
   - `de`: "Hallo, {name}!"
3. Return a terminal error for unsupported languages

### Starter Code

```go
type GreetRequest struct {
    Name       string `json:"name"`
    Language   string `json:"language"`   // New field
    ShouldFail bool   `json:"shouldFail"`
}

func (GreetingService) Greet(
    ctx restate.Context,
    req GreetRequest,
) (GreetResponse, error) {
    // Validate
    if req.Name == "" {
        return GreetResponse{}, restate.TerminalError(
            fmt.Errorf("name cannot be empty"), 400)
    }
    
    // TODO: Implement language selection
    var greeting string
    switch req.Language {
    case "en":
        // TODO
    case "es":
        // TODO
    case "fr":
        // TODO
    case "de":
        // TODO
    default:
        // TODO: Return terminal error for unsupported language
    }
    
    // TODO: Build response with localized greeting
}
```

### Test

```bash
# English
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "language": "en", "shouldFail": false}'

# Spanish
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "language": "es", "shouldFail": false}'

# Unsupported (should fail immediately)
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "language": "zz", "shouldFail": false}'
```

---

## Exercise 3: Service-to-Service Calls 🟡

**Goal:** Create a second service that calls the GreetingService.

### Requirements

1. Create a new service `FormalGreetingService`
2. It should call `GreetingService.Greet` internally
3. Transform the result to be more formal: "Good day, {name}. {original greeting}"

### Starter Code

Create `formal_service.go`:

```go
package main

import (
    "fmt"
    restate "github.com/restatedev/sdk-go"
)

type FormalGreetingService struct{}

type FormalRequest struct {
    Name string `json:"name"`
}

type FormalResponse struct {
    FormalMessage string `json:"formalMessage"`
}

func (FormalGreetingService) FormalGreet(
    ctx restate.Context,
    req FormalRequest,
) (FormalResponse, error) {
    // TODO: Call GreetingService.Greet using ctx
    // Hint: Use restate.Service[OutputType](ctx, "ServiceName", "Handler").Request(input)
    
    greetReq := GreetRequest{
        Name:       req.Name,
        ShouldFail: false,
    }
    
    // Call GreetingService
    greetResp, err := restate.Service[GreetResponse](
        ctx, 
        "GreetingService", 
        "Greet",
    ).Request(greetReq)
    
    if err != nil {
        return FormalResponse{}, err
    }
    
    // Transform to formal
    formalMsg := fmt.Sprintf("Good day, %s. %s", 
        req.Name, 
        greetResp.Message)
    
    return FormalResponse{
        FormalMessage: formalMsg,
    }, nil
}
```

Update `main.go` to register both services:

```go
func main() {
    restateServer := server.NewRestate()
    
    // Register both services
    if err := restateServer.Bind(
        restate.Reflect(GreetingService{}),
    ); err != nil {
        log.Fatal(err)
    }
    
    if err := restateServer.Bind(
        restate.Reflect(FormalGreetingService{}),
    ); err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("🚀 Starting services on :9090...")
    if err := restateServer.Start(context.Background(), ":9090"); err != nil {
        log.Fatal(err)
    }
}
```

### Test

```bash
# Rebuild and restart service
go build -o greeting-service
./greeting-service

# Re-register (to discover new service)
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090", "force": true}'

# Call the formal service
curl -X POST http://localhost:9080/FormalGreetingService/FormalGreet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice"}'

# Should return something like:
# {"formalMessage": "Good day, Alice. Hello, Alice! You're awesome!"}
```

### 🎓 Learning Points

- Service-to-service calls use `restate.Service[T](ctx, ...).Request(...)`
- The call is **journaled** by Restate
- If `FormalGreet` retries, the call to `GreetingService` is **replayed from journal** (not executed again!)

---

## Exercise 4: Retry with Exponential Backoff 🔴

**Goal:** Observe and understand retry behavior in detail.

### Requirements

1. Create a handler that fails a specific number of times, then succeeds
2. Use a global counter to track attempts (not production-ready, but good for learning)
3. Log each attempt with timestamp
4. Observe the exponential backoff pattern

### Starter Code

```go
var attemptCount int

type RetryTestRequest struct {
    FailureCount int `json:"failureCount"` // Fail this many times, then succeed
}

func (GreetingService) RetryTest(
    ctx restate.Context,
    req RetryTestRequest,
) (string, error) {
    attemptCount++
    
    ctx.Log().Info("Retry attempt", 
        "attempt", attemptCount,
        "willFailFor", req.FailureCount)
    
    if attemptCount <= req.FailureCount {
        // Still failing
        return "", fmt.Errorf("attempt %d/%d failed", 
            attemptCount, req.FailureCount)
    }
    
    // Success!
    result := fmt.Sprintf("Succeeded after %d attempts", attemptCount)
    attemptCount = 0 // Reset for next test
    return result, nil
}
```

### Test and Observe

```bash
# This will fail 3 times, then succeed
curl -X POST http://localhost:9080/GreetingService/RetryTest \
  -H 'Content-Type: application/json' \
  -d '{"failureCount": 3}'

# Watch your service logs and note:
# - Attempt 1: immediate
# - Attempt 2: after ~1 second
# - Attempt 3: after ~2 seconds  
# - Attempt 4: after ~4 seconds (succeeds)
```

Record the timestamps and calculate the backoff intervals.

### Questions to Answer

1. What is the backoff pattern? (1s, 2s, 4s, 8s, ...)
2. What is the maximum backoff interval?
3. What happens if it fails indefinitely?

---

## Exercise 5: Input Validation 🟡

**Goal:** Implement comprehensive input validation.

### Requirements

Add validation for:
1. Name must be 2-50 characters
2. Name must not contain numbers or special characters (letters and spaces only)
3. If language is provided, it must be in the supported list
4. Return appropriate Terminal errors with descriptive messages and correct HTTP status codes

### Starter Code

```go
import "regexp"

var namePattern = regexp.MustCompile(`^[a-zA-Z\s]+$`)

func validateGreetRequest(req GreetRequest) error {
    // Validate name length
    if len(req.Name) < 2 || len(req.Name) > 50 {
        return restate.TerminalError(
            fmt.Errorf("name must be between 2 and 50 characters"),
            400,
        )
    }
    
    // Validate name pattern
    if !namePattern.MatchString(req.Name) {
        return restate.TerminalError(
            fmt.Errorf("name must contain only letters and spaces"),
            400,
        )
    }
    
    // Validate language if provided
    if req.Language != "" {
        validLanguages := []string{"en", "es", "fr", "de"}
        // TODO: Check if req.Language is in validLanguages
        // Return terminal error if not
    }
    
    return nil
}

func (GreetingService) Greet(...) {
    // Call validation first
    if err := validateGreetRequest(req); err != nil {
        return GreetResponse{}, err
    }
    
    // ... rest of handler
}
```

### Test Cases

```bash
# Valid
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "language": "en"}'

# Too short
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "A"}'

# Numbers
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice123"}'

# Invalid language
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "language": "xx"}'
```

All invalid cases should return immediately (no retry) with 400 status.

---

## 🏆 Bonus Challenge: Combine Everything 🔴

**Goal:** Build a complete, production-ready greeting service.

### Requirements

1. Multiple language support (Exercise 2)
2. Input validation (Exercise 5)
3. Service-to-service calls (Exercise 3)
4. Proper error handling
5. Comprehensive logging
6. Multiple handlers

### Additional Features

- Add a `GreetMany` handler that greets multiple people in parallel
- Add a `PersonalizedGreet` handler that includes time of day ("Good morning", "Good evening")
- Use `restate.Rand(ctx)` for random greetings

### Hints

```go
func (GreetingService) GreetMany(
    ctx restate.Context,
    names []string,
) ([]string, error) {
    // Use futures for parallel execution
    futures := make([]restate.ResponseFuture[GreetResponse], 0, len(names))
    
    for _, name := range names {
        fut := restate.Service[GreetResponse](
            ctx, "GreetingService", "Greet",
        ).RequestFuture(GreetRequest{Name: name})
        
        futures = append(futures, fut)
    }
    
    // Wait for all (you'll learn this in Module 3!)
    results := make([]string, 0, len(names))
    for fut, err := range restate.Wait(ctx, futures...) {
        if err != nil {
            return nil, err
        }
        resp, err := fut.(restate.ResponseFuture[GreetResponse]).Response()
        if err != nil {
            return nil, err
        }
        results = append(results, resp.Message)
    }
    
    return results, nil
}
```

---

## ✅ Exercise Checklist

Track your progress:

- [ ] Exercise 1: Counter Handler (Beginner)
- [ ] Exercise 2: Language Support (Intermediate)
- [ ] Exercise 3: Service-to-Service Calls (Intermediate)
- [ ] Exercise 4: Retry Observation (Advanced)
- [ ] Exercise 5: Input Validation (Intermediate)
- [ ] Bonus: Complete Production Service (Advanced)

## 📁 Solutions

Complete solutions are available in the [solutions/](./solutions/) directory:

- `exercise1.go` - Counter handler
- `exercise2.go` - Language support
- `exercise3.go` - Service-to-service calls
- `exercise4.go` - Retry testing
- `exercise5.go` - Input validation
- `bonus.go` - Complete service

**Don't peek until you've tried!** 😊

## 🎯 Next Module

Congratulations! You've completed Module 1 and should now understand:
- ✅ Durable execution fundamentals
- ✅ Basic Services and handlers
- ✅ Error handling (Terminal vs Retriable)
- ✅ Service-to-service communication
- ✅ Journaling and replay

Ready for the next challenge?

👉 **Continue to [Module 2: Resilient Stateless APIs](../02-side-effects/README.md)**

Learn about side effects, `restate.Run`, and building truly resilient services!

---

**Questions?** Review the [concepts](./01-concepts.md) or check the [complete code](./code/)!
