# Hands-On: Building Your First Durable Service

> **Step-by-step guide to creating, deploying, and testing a Restate service**

## 🎯 What We're Building

A **Durable Greeting Service** that:
- Accepts a name and returns a personalized greeting
- Demonstrates retry behavior on failures
- Shows journaling in action
- Handles both success and error cases

## 📋 Prerequisites

- ✅ Completed [concepts](./01-concepts.md)
- ✅ Restate server running (`restate-server`)
- ✅ Terminal ready at `~/restate-tutorials/module01`

## 🚀 Step-by-Step Tutorial

### Step 1: Create Project Structure

```bash
# Create project directory
mkdir -p ~/restate-tutorials/module01
cd ~/restate-tutorials/module01

# Initialize Go module
go mod init module01

# Install Restate SDK
go get github.com/restatedev/sdk-go
```

### Step 2: Create the Service

Create `service.go`:

```go
package main

import (
    "fmt"
    restate "github.com/restatedev/sdk-go"
)

// GreetingService is a Basic Service (stateless)
type GreetingService struct{}

// GreetRequest defines our input structure
type GreetRequest struct {
    Name       string `json:"name"`
    ShouldFail bool   `json:"shouldFail"` // For testing retry behavior
}

// GreetResponse defines our output structure
type GreetResponse struct {
    Message   string `json:"message"`
    Timestamp string `json:"timestamp"`
}

// Greet is our handler function
func (GreetingService) Greet(
    ctx restate.Context,
    req GreetRequest,
) (GreetResponse, error) {
    // Log the request (won't duplicate on replay!)
    ctx.Log().Info("Processing greeting", 
        "name", req.Name,
        "shouldFail", req.ShouldFail)
    
    // Simulate a failure for testing
    if req.ShouldFail {
        // This will cause Restate to retry
        return GreetResponse{}, fmt.Errorf("simulated failure")
    }
    
    // Generate a deterministic UUID (same on replay)
    requestID := restate.UUID(ctx).String()
    
    // Create response
    response := GreetResponse{
        Message:   fmt.Sprintf("Hello, %s! You're awesome!", req.Name),
        Timestamp: requestID, // Using UUID as a unique identifier
    }
    
    ctx.Log().Info("Greeting generated successfully", "requestID", requestID)
    
    return response, nil
}
```

**🎓 Learning Points:**

1. **Service Structure**
   - `GreetingService struct{}` - Empty struct for Basic Service
   - Handler signature: `(ctx restate.Context, input T) (output T, error)`

2. **Context Logging**
   - `ctx.Log()` - Logs only once, not on replay
   - Structured logging with key-value pairs

3. **Deterministic Operations**
   - `restate.UUID(ctx)` - Same UUID on replay (deterministic)
   - Never use `time.Now()` or `rand.Float64()` directly!

### Step 3: Create the Main Entry Point

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
    // Create a new Restate server
    restateServer := server.NewRestate()
    
    // Register our service
    if err := restateServer.Bind(
        restate.Reflect(GreetingService{}),
    ); err != nil {
        log.Fatal("Failed to bind service:", err)
    }
    
    // Start the server on port 9090
    fmt.Println("🚀 Starting Greeting Service on :9090...")
    fmt.Println("📝 Service: GreetingService")
    fmt.Println("📌 Handlers: Greet")
    
    if err := restateServer.Start(context.Background(), ":9090"); err != nil {
        log.Fatal("Server error:", err)
    }
}
```

**🎓 Learning Points:**

1. **Server Creation**
   - `server.NewRestate()` - Creates Restate server
   - Binds to a specific port (9090)

2. **Service Registration**
   - `restate.Reflect()` - Discovers handlers automatically
   - Analyzes struct methods matching handler signature

3. **Service Discovery**
   - Restate uses reflection to find handlers
   - Exported methods with correct signature become handlers

### Step 4: Build and Run

```bash
# Ensure dependencies are downloaded
go mod tidy

# Build the service
go build -o greeting-service

# Run the service
./greeting-service
```

You should see:
```
🚀 Starting Greeting Service on :9090...
📝 Service: GreetingService
📌 Handlers: Greet
```

### Step 5: Register with Restate

In a **new terminal**:

```bash
# Register the service
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{
    "uri": "http://localhost:9090",
    "force": false
  }'
```

**Expected Response:**
```json
{
  "id": "dp_xxx...",
  "services": [
    {
      "name": "GreetingService",
      "handlers": [
        {
          "name": "Greet",
          "input": "...",
          "output": "..."
        }
      ]
    }
  ]
}
```

**🎓 What Just Happened?**

1. Restate called your service's discovery endpoint
2. It learned about `GreetingService` and its `Greet` handler
3. It stored this information in its registry
4. Now you can call the service through Restate!

### Step 6: Call the Service (Success Case)

```bash
# Call via Restate ingress
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Alice",
    "shouldFail": false
  }'
```

**Expected Response:**
```json
{
  "message": "Hello, Alice! You're awesome!",
  "timestamp": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
```

Check your service logs - you'll see:
```
INFO Processing greeting name=Alice shouldFail=false
INFO Greeting generated successfully requestID=f47ac10b-...
```

### Step 7: Test Retry Behavior

Now let's trigger a failure:

```bash
# Call with shouldFail=true
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Bob",
    "shouldFail": true
  }'
```

**What Happens:**

1. Your service returns an error
2. Restate automatically retries with exponential backoff
3. It keeps retrying until success (or you cancel)
4. Check your logs - you'll see multiple "Processing greeting" entries

**To stop the retries**, use the Restate CLI or kill the invocation:

```bash
# First, find the invocation ID from the service logs or admin API
curl http://localhost:8080/invocations | jq '.invocations[] | select(.target_service_name == "GreetingService")'

# Kill the invocation (replace <invocation-id>)
curl -X DELETE http://localhost:8080/invocations/<invocation-id>
```

### Step 8: Use Terminal Errors (Proper Error Handling)

Let's improve our service to handle failures properly.

Update `service.go`:

```go
func (GreetingService) Greet(
    ctx restate.Context,
    req GreetRequest,
) (GreetResponse, error) {
    ctx.Log().Info("Processing greeting", 
        "name", req.Name,
        "shouldFail", req.ShouldFail)
    
    // Validate input
    if req.Name == "" {
        // Terminal error - no retry needed
        return GreetResponse{}, restate.TerminalError(
            fmt.Errorf("name cannot be empty"),
            400, // HTTP status code
        )
    }
    
    // Simulate a transient failure
    if req.ShouldFail {
        // Regular error - Restate will retry
        return GreetResponse{}, fmt.Errorf("simulated transient failure")
    }
    
    requestID := restate.UUID(ctx).String()
    
    response := GreetResponse{
        Message:   fmt.Sprintf("Hello, %s! You're awesome!", req.Name),
        Timestamp: requestID,
    }
    
    ctx.Log().Info("Greeting generated successfully", "requestID", requestID)
    
    return response, nil
}
```

Rebuild and restart:

```bash
# Stop the service (Ctrl+C)
# Rebuild
go build -o greeting-service
# Restart
./greeting-service
```

Test with empty name (Terminal error - no retry):

```bash
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "",
    "shouldFail": false
  }'
```

**Response:**
```json
{
  "error": "name cannot be empty",
  "code": 400
}
```

Notice: **No retries!** The error is returned immediately.

## 🎓 Understanding What We Built

### The Flow

```
1. Client Call
   curl → http://localhost:9080/GreetingService/Greet
   ↓
2. Restate Ingress
   - Creates invocation ID
   - Starts journaling
   ↓
3. Your Service
   - Receives request via Restate
   - Processes with ctx
   - Returns response
   ↓
4. Restate Journals Result
   - Stores response in journal
   - Can replay from here if needed
   ↓
5. Response to Client
   - Returns result
```

### The Journal

Restate maintains an execution journal:

```
Invocation: inv_GreetingService_Greet_abc123
├─ Entry 1: Handler started
├─ Entry 2: Log: "Processing greeting"
├─ Entry 3: Generated UUID: f47ac10b-...
├─ Entry 4: Log: "Greeting generated"
└─ Entry 5: Handler completed with result
```

**On Retry:** Restate skips completed entries and resumes from where it left off.

### Error Handling Strategy

```
┌─────────────────────────────────────┐
│ Is this a permanent failure?        │
├─────────────────────────────────────┤
│ YES → restate.TerminalError()       │
│       - Invalid input               │
│       - Business logic failure      │
│       - Resource not found          │
│                                     │
│ NO  → regular error                 │
│       - Network timeout             │
│       - Service unavailable         │
│       - Database connection failed  │
└─────────────────────────────────────┘
```

## ✅ Verification Checklist

Test each scenario:

- [ ] **Success case**: Name provided, shouldFail=false
- [ ] **Retry case**: shouldFail=true (observe retries)
- [ ] **Terminal error**: Empty name (no retry)
- [ ] **Logs**: Check that logs don't duplicate on replay

## 🐛 Common Issues and Solutions

### Issue: "Service already registered"

**Solution:**
```bash
# List deployments
curl http://localhost:8080/deployments

# Delete old deployment
curl -X DELETE http://localhost:8080/deployments/<deployment-id>

# Re-register
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

### Issue: "Connection refused"

**Solutions:**
1. Ensure Restate server is running
2. Check your service is running on :9090
3. Verify no firewall blocking the ports

### Issue: Service changes not taking effect

**Solution:**
```bash
# For code changes during development, you can force update:
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090", "force": true}'
```

⚠️ **Important**: In production, use versioned deployments instead of `force: true`!

## 💡 Key Takeaways

1. **Context is Central** - Always use `ctx` for durable operations
2. **Journaling is Automatic** - Restate records every step
3. **Retry by Default** - Transient failures auto-retry
4. **Terminal Errors Matter** - Use for permanent failures
5. **Logging is Smart** - `ctx.Log()` deduplicates on replay

## 🎯 Next Steps

You've built your first durable service! Now let's verify it works correctly.

👉 **Continue to [Validation](./03-validation.md)**

---

**Stuck?** Check the [complete code](./code/) or ask in the community!
