The user wants me to fix the idempotency bug I identified in the evaluation. Specifically:

Replace time.Now() with restate.UUID(ctx) or deterministic timestamp from run context
Add validation against non-deterministic ID usage
Looking at the code in framework.go, line 437-440:

go
// GenerateIdempotencyKey creates a deterministic key for external calls.
func (cp *ControlPlaneService) GenerateIdempotencyKey(suffix string) string {
    return fmt.Sprintf("%s:%s:%d", cp.idempotencyPrefix, suffix, time.Now().UnixNano())
}
The issue is that time.Now() is non-deterministic. According to the best practices:

Should use restate.UUID(ctx) for deterministic random values
Or wrap time.Now() in restate.Run to capture it deterministically
However, I notice that ControlPlaneService doesn't have a context field. It's created with NewControlPlaneService(ctx restate.WorkflowContext, ...) but doesn't store the context.

Looking at the structure:

Line 370-374: ControlPlaneService struct has name, saga, idempotencyPrefix
Line 376-383: NewControlPlaneService creates it but doesn't store ctx
The saga field stores the context (line 182: wctx restate.WorkflowContext)
So I have a few options:

Add a context field to ControlPlaneService
Use the saga's context (cp.saga.wctx)
Change GenerateIdempotencyKey to take a context parameter
Use restate.Run with the saga's context
Option 3 seems cleanest - make GenerateIdempotencyKey accept a context so it can generate deterministic IDs. But we also need to validate usage.

Let me also check where this function is used... it's at line 437-440, but I don't see it being called anywhere in the framework.go file itself. So it's meant to be used by consumers of the framework.

I'll modify the method to use a context-based UUID generation, ensuring deterministic behavior across workflow executions. The validation will prevent non-deterministic ID patterns from being introduced accidentally.

Fixing Idempotency Key Generation Bug
Starting to fix the critical idempotency bug identified in the evaluation where 
GenerateIdempotencyKey()
 uses time.Now() instead of deterministic values.

Added # Validation Against Non-Deterministic ID Usage - Integration Guide (VALIDATION_GUIDE.MD)
