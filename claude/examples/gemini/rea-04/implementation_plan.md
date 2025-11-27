# Rewrite Programs Using Rea Framework

Convert the existing Restate SDK programs to use the `github.com/pithomlabs/rea` framework, which provides a higher-level abstraction over the Restate SDK.

## User Review Required

> [!IMPORTANT]
> **Rea Framework Usage**: The rea framework provides builder patterns and configuration helpers over the raw Restate SDK. The rewrite will maintain identical business logic while using rea's cleaner API.

**Questions for Clarification:**
1. Should the ingress layer use rea's helpers or continue using standard `restatedev/sdk-go/ingress`?
2. Are there specific rea patterns you want emphasized (e.g., configuration builders, service definitions)?
3. Should I preserve all the comments and documentation from the original files?

## Proposed Changes

### Setup

#### [NEW] [go.mod](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/gemini/rea/ingress/go.mod)
Copy from `gemini/go.mod` - Already has rea dependency

#### [NEW] [go.sum](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/gemini/rea/ingress/go.sum)
Copy from parent - Lock file for dependencies

#### [NEW] [go.mod](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/gemini/rea/services/go.mod)
Copy from `gemini/go.mod` - Already has rea dependency

#### [NEW] [go.sum](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/gemini/rea/services/go.sum)
Copy from parent - Lock file for dependencies

---

### Ingress Layer

#### [NEW] [ingress.go](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/gemini/rea/ingress/ingress.go)

**Changes from original**:
- Import `github.com/pithomlabs/rea` if it provides ingress helpers
- Use rea's configuration patterns if available
- Otherwise maintain standard ingress client usage
- Keep Chi router and middleware patterns identical
- Preserve authentication and security logic

**Key Differences**:
- May use rea's client builders if available
- Configuration might use rea's option patterns
- Business logic remains unchanged

---

### Services Layer  

#### [NEW] [svcs.go](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/gemini/rea/services/svcs.go)

**Changes from original**:

**1. Service Definitions**
- Replace `restate.Reflect()` with rea's service builders:
  - `rea.NewService()` for stateless services
  - `rea.NewObject()` for virtual objects  
  - `rea.NewWorkflow()` for workflows

**2. Handler Registration**
- Use rea's fluent API for handler registration
- Configuration options via rea's option pattern
- Security and retention settings via rea builders

**3. Context Operations**
- State management: Same `restate.Get/Set` API (rea wraps it)
- Service calls: Same `restate.Service/Object/Workflow` API
- Awakeables: Same `restate.Awakeable` API
- Promises: Same `restate.Promise` API
- Sleep: Same `restate.Sleep` API

**4. Server Setup**
- Replace `server.NewRestate().Bind()` with rea's server builder
- Use rea's configuration for security, retention, etc.

**Example Transformation**:
```go
// Before (raw Restate SDK)
server.NewRestate().
    Bind(restate.Reflect(UserSession{})).
    Start(ctx, ":9080")

// After (rea framework)
rea.NewServer().
    Register(rea.NewObject("UserSession").
        Handler("AddItem", UserSession{}.AddItem).
        Handler("Checkout", UserSession{}.Checkout)).
    Start(ctx, ":9080")
```

## Verification Plan

### Manual Compilation

**Step 1: Compile Ingress**
```bash
cd /home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/gemini/rea/ingress
go mod tidy
go build -o ingress ingress.go
```

**Step 2: Compile Services**
```bash
cd /home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/gemini/rea/services
go mod tidy
go build -o svcs svcs.go
```

**Expected Result**: Both programs compile without errors

### Functional Verification (Optional - User's Choice)

If user wants to test functionality:

1. Start Restate server (Docker)
2. Run `svcs` binary
3. Register with Restate
4. Run `ingress` binary  
5. Test HTTP endpoints with curl

**Note**: User will handle compilation manually per instructions
