# Module 01: Foundation - Complete Code

This directory contains the complete, working code for Module 1.

## Files

- **main.go** - Service entry point and server setup
- **service.go** - GreetingService implementation
- **go.mod** - Go module dependencies

## How to Run

### 1. Start Restate Server

```bash
# In a separate terminal
restate-server
```

### 2. Build and Run the Service

```bash
cd ~/restate-tutorials/module01
# Or copy these files to your working directory

# Install dependencies
go mod tidy

# Build
go build -o greeting-service

# Run
./greeting-service
```

You should see:
```
🚀 Starting Greeting Service on :9090...
📝 Service: GreetingService
📌 Handlers: Greet

Register with Restate:
  curl -X POST http://localhost:8080/deployments \
    -H 'Content-Type: application/json' \
    -d '{"uri": "http://localhost:9090"}'
```

### 3. Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

### 4. Test the Service

**Success case:**
```bash
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Alice",
    "shouldFail": false
  }'
```

**Terminal error (validation):**
```bash
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "",
    "shouldFail": false
  }'
```

**Retriable error (will retry):**
```bash
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Bob",
    "shouldFail": true
  }'
# Remember to cancel this invocation!
```

## Code Structure

### main.go

Entry point that:
- Creates Restate server
- Registers GreetingService
- Starts listening on port 9090

### service.go

Contains:
- `GreetingService` struct (Basic Service)
- `GreetRequest` and `GreetResponse` types
- `Greet` handler with:
  - Context logging
  - Input validation
  - Error handling (Terminal vs Retriable)
  - Deterministic UUID generation

## Key Learning Points

1. **Context Usage**
   ```go
   ctx.Log().Info("message", "key", value)  // Smart logging
   requestID := restate.UUID(ctx)            // Deterministic UUID
   ```

2. **Error Handling**
   ```go
   // Terminal error - no retry
   restate.TerminalError(err, httpCode)
   
   // Retriable error - auto retry
   return response, fmt.Errorf("transient error")
   ```

3. **Handler Signature**
   ```go
   func (ServiceStruct) HandlerName(
       ctx restate.Context,
       input InputType,
   ) (OutputType, error)
   ```

## Common Issues

### Port Already in Use

```bash
# Find and kill process on port 9090
lsof -i :9090
kill -9 <PID>
```

### Service Won't Register

```bash
# Check if service is running
curl http://localhost:9090

# Force re-registration
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090", "force": true}'
```

## Next Steps

After running this code successfully:

1. Modify the service to add new features
2. Try the exercises in `../04-exercises.md`
3. Experiment with different error scenarios
4. Move on to Module 2!

---

**Having issues?** Compare your code with these files or check the [troubleshooting guide](../../appendix/troubleshooting.md).
