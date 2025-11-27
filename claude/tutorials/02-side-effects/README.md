# Module 02: Resilient Stateless APIs - Side Effects and `restate.Run`

> **Master the art of durable side effects and build truly resilient stateless services**

## 🎯 Learning Objectives

By the end of this module, you will:

- ✅ Understand what side effects are and why they need special handling
- ✅ Master the `restate.Run` pattern for durable side effects
- ✅ Differentiate between deterministic and non-deterministic operations
- ✅ Implement retry strategies with exponential backoff
- ✅ Avoid common anti-patterns with context misuse
- ✅ Build a real-world API aggregation service

## 📚 Module Contents

| File | Description | Time |
|------|-------------|------|
| **[01-concepts.md](./01-concepts.md)** | Side effects and determinism | 20 min |
| **[02-hands-on.md](./02-hands-on.md)** | Build weather aggregation service | 45 min |
| **[03-validation.md](./03-validation.md)** | Testing side effects | 15 min |
| **[04-exercises.md](./04-exercises.md)** | Practice exercises | 20 min |

## 🎓 Prerequisites

- Completed [Module 01: Foundation](../01-foundation/README.md)
- Understanding of Basic Services and error handling
- Restate server running

## 🏗️ What You'll Build

**Project: Weather Aggregation Service**

A practical service that:
- Fetches weather data from multiple external APIs
- Aggregates results using `restate.Run`
- Handles partial failures gracefully
- Demonstrates retry with exponential backoff

```
Input: {"city": "London"}
         ↓
    [Fetch from API 1] ─┐
    [Fetch from API 2] ─┼─► [Aggregate Results]
    [Fetch from API 3] ─┘
         ↓
Output: Combined weather data
```

## 💡 Key Concepts Preview

### What is a Side Effect?

**Side Effect:** Any operation that:
- Interacts with the outside world
- Has non-deterministic results
- Cannot be safely replayed

Examples:
- 🌐 HTTP API calls
- 💾 Database queries
- 📧 Sending emails
- 🎲 Generating random numbers
- ⏰ Getting current time

### The Problem

```go
// ❌ WRONG - Not durable!
func (s *Service) Process(ctx restate.Context, input string) (string, error) {
    // This API call is lost on crash/retry!
    data := callExternalAPI(input)
    return process(data), nil
}
```

### The Solution

```go
// ✅ CORRECT - Durable with restate.Run!
func (s *Service) Process(ctx restate.Context, input string) (string, error) {
    // API call result is journaled
    data, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        return callExternalAPI(input), nil
    })
    if err != nil {
        return "", err
    }
    return process(data), nil
}
```

## 🎯 Learning Path

```
1. Concepts
   ↓
   - What are side effects?
   - Determinism requirements
   - restate.Run pattern
   - Anti-patterns to avoid
   
2. Hands-On
   ↓
   - Build weather service
   - Fetch from multiple APIs
   - Handle failures
   - Aggregate results
   
3. Validation
   ↓
   - Test side effect journaling
   - Verify retry behavior
   - Check determinism
   
4. Exercises
   ↓
   - Extend to more APIs
   - Add caching
   - Implement timeouts
```

## 📁 Module Structure

```
02-side-effects/
├── README.md           ← You are here
├── 01-concepts.md      ← Theory: side effects, Run pattern
├── 02-hands-on.md      ← Build weather aggregation service
├── 03-validation.md    ← Test journaling and retries
├── 04-exercises.md     ← Practice exercises
├── code/
│   ├── main.go
│   ├── service.go
│   ├── weather_apis.go ← Mock weather API calls
│   └── go.mod
└── solutions/
    └── *.go
```

## 🎓 Success Criteria

You've mastered this module when you can:

- [ ] Explain why side effects need `restate.Run`
- [ ] Wrap external calls properly
- [ ] Understand what can/cannot go in`Run` blocks
- [ ] Avoid context misuse anti-patterns
- [ ] Build services that aggregate external data
- [ ] Implement proper error handling for external calls

## ⏱️ Time Commitment

- **Minimum:** 45 minutes (concepts + hands-on)
- **Recommended:** 1.5 hours (all materials + exercises)
- **Mastery:** 2.5 hours (with experimentation)

## ⚠️ Common Pitfalls We'll Avoid

1. **Using `ctx` inside `restate.Run`** ❌
   ```go
   restate.Run(ctx, func(rc restate.RunContext) {
       ctx.Sleep(...) // WRONG! Use rc, not ctx
   })
   ```

2. **Not wrapping external calls** ❌
   ```go
   data := callAPI() // Lost on crash!
   ```

3. **Non-deterministic operations outside Run** ❌
   ```go
   if time.Now().Hour() > 12 { // Non-deterministic!
   ```

## 🔗 Next Module

After completing this module:

👉 **[Module 3: Concurrent Execution](../03-concurrency/README.md)**

Learn to call multiple services in parallel with fan-out/fan-in patterns!

---

**Ready to dive in?** Start with [Concepts](./01-concepts.md)!
