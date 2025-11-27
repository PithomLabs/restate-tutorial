# Comparative Code Examples: Standard Restate Go SDK vs. Custom Framework

This document provides brief, concise code comparisons between the standard Restate Go SDK and the custom framework from [`framework.go`](evals/rea2/claude/framework.go). Each category shows the standard implementation first, followed by the framework's version, highlighting reductions in boilerplate for better productivity and readability.

## Ingress Client
### Standard Restate Go SDK
```go
// Basic ingress client setup for external calls
client := restateingress.NewClient("http://localhost:8080")
result, err := restateingress.Service[string, string](client, "MyService", "MyHandler").Request(context.Background(), "Hello")
if err != nil {
    return err
}
```

### Custom Framework
```go
// Framework-wrapped ingress client with built-in options
client := NewIngressClient("http://localhost:8080") // Assumes framework abstraction
result, err := client.Call[string, string]("MyService", "MyHandler", "Hello", WithIdempotency("key"))
if err != nil {
    return err
}
```

## Service Invocation
### Standard Restate Go SDK
```go
// Invoking a service handler
result, err := restate.Service[string](ctx, "MyService", "MyHandler").Request("Input")
if err != nil {
    return err
}
```

### Custom Framework
```go
// Framework service client for invocation
svcClient := NewServiceClient[string, string]("MyService", "MyHandler")
result, err := svcClient.Call(ctx, "Input") // Includes automatic context handling
if err != nil {
    return err
}
```

## Restate Context
### Standard Restate Go SDK
```go
// Using context for state access
value, err := restate.Get[string](ctx, "key")
if err != nil {
    return err
}
restate.Set(ctx, "key", "value")
```

### Custom Framework
```go
// Framework state accessor with context guards
state := NewState[string](ctx, "key")
value, err := state.Get()
if err != nil {
    return err
}
err = state.Set("value") // Validates context internally
if err != nil {
    return err
}
```

## State Management
### Standard Restate Go SDK
```go
// Managing state in a handler
items, err := restate.Get[[]string](ctx, "items")
if err != nil {
    return err
}
items = append(items, "newItem")
restate.Set(ctx, "items", items)
```

### Custom Framework
```go
// Framework state management with type safety
state := NewState[[]string](ctx, "items")
items, err := state.Get()
if err != nil {
    return err
}
items = append(items, "newItem")
err = state.Set(items) // Handles validation and sets
if err != nil {
    return err
}
```

## Interservice Communication
### Standard Restate Go SDK
```go
// Sending a message
restate.ServiceSend(ctx, "MyService", "MyHandler").Send("Message", restate.WithIdempotencyKey("key"))
```

### Custom Framework
```go
// Framework send with options
svcClient := NewServiceClient[any, any]("MyService", "MyHandler")
svcClient.Send(ctx, "Message", CallOption{IdempotencyKey: "key"})
```

## Futures and Async Primitives
### Standard Restate Go SDK
```go
// Waiting for futures
fut1 := restate.Service[string](ctx, "Svc1", "Handler").RequestFuture("Input")
fut2 := restate.Service[string](ctx, "Svc2", "Handler").RequestFuture("Input")
results, err := restate.Wait(ctx, fut1, fut2)
if err != nil {
    return err
}
```

### Custom Framework
```go
// Framework async wait wrapper
futures := []restate.Future{fut1, fut2}
results, err := GatherFutures(ctx, futures...) // Framework utility for waiting
if err != nil {
    return err
}
```

## Run Operation
### Standard Restate Go SDK
```go
// Wrapping a side effect
result, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
    return callExternalAPI(), nil
})
if err != nil {
    return err
}
```

### Custom Framework
```go
// Framework run with error handling
result, err := RunWithFramework(ctx, callExternalAPI) // Wraps and handles retries
if err != nil {
    return err
}
```

## Restate Server
### Standard Restate Go SDK
```go
// Starting the server
server.NewRestate().Bind(restate.Reflect(MyService{})).Start(context.Background(), ":8080")
```

### Custom Framework
```go
// Framework server setup with bindings
server := NewFrameworkServer().BindServices(MyService{})
server.Start(":8080") // Includes auto-configurations
```

## Error Handling
### Standard Restate Go SDK
```go
// Handling errors
if err != nil {
    return restate.TerminalError(err, 500)
}
```

### Custom Framework
```go
// Framework error wrapper
if err != nil {
    return HandleError(err) // Wraps and converts to terminal error
}
```

## Ensuring Idempotency
### Standard Restate Go SDK
```go
// Idempotent call
restate.ServiceSend(ctx, "Svc", "Handler").Send("Data", restate.WithIdempotencyKey("unique-key"))
```

### Custom Framework
```go
// Framework idempotent send
svcClient.Send(ctx, "Data", CallOption{IdempotencyKey: "unique-key"})
```

This approach reduces boilerplate by abstracting common patterns, improving readability and productivity.
