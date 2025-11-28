Perfect — I'll generate **PROMPTS.MD (Option C)** as a **single, comprehensive file**, containing:

1. **Developer Prompt Library**
    
2. **LLM Internal Behavior Rules (AGENTS.md–style)**
    
3. **Scenario-Based Prompt Packs & Code-Generation Templates**
    
4. **Debugging Prompts, Architecture Prompts, Determinism Enforcement Prompts**
    
5. **Prompts specific to Restate Go SDK + pithomlabs/rea framework**
    
6. **Prompts for CI, refactoring, testing, diagrams, saga creation, concurrency audits**
    
7. **Anti-pattern detection prompts**
    
8. **LLM-to-LLM meta prompts (for multi-agent pipelines)**
    

---

# ✅ **Your PROMPTS.MD**

_(You can drop this into your repo as-is)_

---

# **PROMPTS.MD**

**Prompt Engineering & Context Engineering Guide**

### for Restate Go SDK + pithomlabs/rea Framework

### Developer Prompts + LLM Behavior Rules + Scenario Prompt Packs

---

# ------------------------------

# **PART 1 — DEVELOPER PROMPT LIBRARY**

## (“How humans should talk to LLMs to get high-quality Restate Go + REA answers.”)

# ------------------------------

Below is a curated set of powerful prompts optimized for:

- Deterministic Restate handler design
    
- Safe concurrency with durable primitives
    
- Correct usage of `Run`, `RunAsync`, `Wait`, Awakeables & Durable Promises
    
- Saga orchestration
    
- Guardrail checks
    
- Ingress client usage
    
- REA framework primitives (RunDo, SafeRun, deterministic helpers, policy, state helpers)
    

Every prompt below is tested against the DO/DON’Ts from:

- **DOS_DONTS_MEGA.MD**
    
- **DOS_DONTS_REA.MD**
    

---

## 🧱 **1. Prompt: Generate a Full Restate Service / Object / Workflow**

```
Generate a full Restate Go SDK service/object/workflow using best practices.

Requirements:
- Follow ALL DO/DON’Ts from DOS_DONTS_MEGA.MD and DOS_DONTS_REA.MD.
- Use REA helpers (RunDo, SafeRun, deterministic helpers, WorkflowTimer, FanOut, etc.)
- Wrap all external or nondeterministic logic in restate.Run or framework.RunDo.
- No goroutines, no channels, no time.Now.
- All concurrency through RequestFuture, RunAsyncFuture, Wait, WaitFirst.
- Workflow must have exactly one Run handler.
- Virtual Objects must enforce blocking-safe operations (no sleeps, no awakeables in ObjectContext).
- Use ctx.Log() only.
- Ensure idempotency, state safety, and deterministic UUID/time usage.
- Add comments explaining durable execution boundaries (Control Plane vs Data Plane).
```

---

## 🏗️ **2. Prompt: Generate a Saga Workflow (with compensation)**

```
Create a full Saga Workflow in Go using Restate + REA framework.

Specifications:
- Use defer s.CompensateIfNeeded(&err)
- Use SafeStep or WithCompensation pattern BEFORE the primary action.
- Every compensation MUST be idempotent.
- Use deterministic UUID/time helpers.
- Use RequestFuture or RunAsync for parallel tasks.
- Insert durable timers using restate.After or WorkflowTimer.
- NO nondeterministic operations inside handler bodies.
- Add comments describing each durable step and compensation.
```

---

## 🔍 **3. Prompt: Analyze Code for Determinism & Guardrail Violations**

```
Analyze the following Go code for determinism and DO/DON'T violations.

Tasks:
- Identify ALL nondeterministic operations.
- Identify all misuses of ctx.Run and RunContext.
- Identify dangerous patterns in Virtual Objects (sleep, awakeable, deadlocks).
- Identify blocking calls inside futures or goroutines.
- Identify incorrect state usage (shared vs exclusive contexts).
- Identify incorrect use of ingress clients inside handlers.
- Identify unsafe idempotency key usage.
- Suggest exact fixes using REA helper functions.
```

---

## 🧭 **4. Prompt: Convert Imperative Logic into Durable Flow**

```
Convert the following business flow into a deterministic Restate Workflow.

Output:
- Durable Run handler
- Partition state if needed
- Calls expressed as futures (RequestFuture / RunAsync)
- All I/O wrapped in ctx.Run or framework.RunDo
- Deterministic branching only
- Durable timers instead of time.Sleep
- Use durable signals (Awakeable / Promise) appropriately
```

---

## 🪢 **5. Prompt: Debug Concurrency / Deadlock Issues**

```
Diagnose this code for deadlock and concurrency issues.

Checklist:
- Are Virtual Objects making cross-calls?
- Any exclusive handler waiting on a future?
- Any Awakeable.Result() in ObjectContext?
- Any goroutines calling Response() or Result()?
- Any request-response cycles A→B→A?
- Any workflow using Awakeables instead of Durable Promises?
- Suggest corrections using framework.FanOut / FanOutFail / RacePromiseWithTimeout.
```

---

## 📡 **6. Prompt: External Signal / Awakeable Modeling**

```
Model this external callback flow using Awakeables (Services/Objects) or Durable Promises (Workflows).

Decide:
- Should external system use Awakeable?
- Should internal coordination use Durable Promise?
- Show full Go code for Wait and Resolve.
- Follow REA framework rules and guardrails.
- Avoid blocking exclusive handlers.
```

---

## 🔐 **7. Prompt: Security Audit**

```
Perform a security audit of this service.

Look for:
- Missing L1 signature validation
- Missing L2 identity propagation
- Misuse of IngressClient inside handlers
- Absent idempotency key validation
- Non-durable external calls
- Leaking internal services publicly
```

---

## 🧪 **8. Prompt: Generate CI Tests for Durable Code**

```
Write Go tests for this Restate handler.

Requirements:
- Test determinism
- Test idempotency
- Test compensation correctness
- Test durable timers behavior
- Test concurrency with fake futures
- Provide test harness code
```

---

## 🧨 **9. Prompt: Migrate Imperative Microservice → Restate**

```
Transform the following REST microservice into:
- Restate Service
- Virtual Object (if stateful)
- Workflow (if multi-step)
Include:
- Deterministic flows
- Run blocks
- Durable timers
- REA helpers (deterministic time, UUID, state wrappers)
```

---

## 🧯 **10. Prompt: Generate Stress-Resilient Version**

```
Rewrite this handler to be fully resilient to retries, crashes, replay, and idempotency failures.

Must:
- Have deterministic branching
- Isolate all side effects in Run blocks
- Use durable timers
- Avoid any native concurrency
```

---

# ------------------------------

# **PART 2 — LLM INTERNAL RULES (AGENTS.md style)**

## (“How LLMs MUST behave when generating code.”)

# ------------------------------

Copy this section into:

- `.cursor/rules/`
    
- `.claude/`
    
- `AGENTS.md`
    

---

# **LLM RULES FOR RESTATE + REA CODE GENERATION**

### ✅ **Required Behaviors**

1. **Always generate deterministic, replay-safe logic.**
    
2. **Always separate Control Plane (ctx operations) vs Data Plane (Run blocks).**
    
3. **Always wrap nondeterministic work in `restate.Run` or `framework.RunDo`.**
    
4. **Use NO native concurrency:**
    
    - ❌ no `go func()`
        
    - ❌ no channels
        
    - ❌ no `select {}`  
        Use only **RequestFuture / RunAsync / Wait / WaitFirst**.
        
5. **Workflows:**
    
    - Must have **exactly one Run handler**.
        
    - Must use **Durable Promises** for internal waiting.
        
    - Must not use Awakeables for internal events.
        
6. **Virtual Objects:**
    
    - Exclusive handlers (**ObjectContext**) must not block on:
        
        - ❌ Sleep
            
        - ❌ Awakeables
            
        - ❌ Awaiting futures
            
    - All external calls must leave object quickly; use delayed messages instead.
        
7. **Logging:**
    
    - Use only `ctx.Log()`.
        
    - No fmt printing, log/slog globals, console prints.
        
8. **Deterministic Helpers:**
    
    - UUIDs: `framework.NewDeterministicHelpers(ctx).UUID()`
        
    - Time: `framework.NewTime(ctx).Now()`
        
9. **State Rules:**
    
    - Shared contexts (ObjectSharedContext, WorkflowSharedContext) = read-only.
        
    - Exclusive contexts (ObjectContext, WorkflowContext) = mutation allowed.
        
10. **Ingress Clients:**
    
    - NEVER used inside durable handlers.
        
    - Only used outside Restate runtime.
        
11. **Sagas:**
    
    - Compensations must be idempotent.
        
    - Must be registered before the action.
        
12. **Security:**
    
    - Prefer L1 signature validation and L2 identity propagation headers.
        
13. **Never introduce non-determinism:**
    
    - No `time.Now()`
        
    - No random libraries
        
    - No iteration over Go maps when performing ctx operations
        

---

# **LLM MUST DETECT AND FLAG THE FOLLOWING ANTI-PATTERNS:**

❌ Using Awakeables in Virtual Object exclusive handlers  
❌ Request-response cycles leading to deadlocks  
❌ Side effects outside Run blocks  
❌ Native Go concurrency  
❌ Missing idempotency  
❌ time.Sleep or time.After  
❌ External calls inside handler body  
❌ Logging with fmt.Printf  
❌ Non-idempotent compensations  
❌ Changing state in shared context

---

# ------------------------------

# **PART 3 — SCENARIO-BASED PROMPT PACKS**

# ------------------------------

Below are specialized pre-built prompts.

---

## ⭐ **1. “Generate a Multi-Step Order Processing Workflow”**

```
Create a durable order processing workflow using Restate + REA.

Steps:
- Reserve inventory (service call)
- Charge payment (RunDo external)
- Send email (RunDo external)
- Ship order (service/object call)
- Add durable timers for reminders
- Add compensation for payment and inventory
Follow all DO/DON’Ts.
```

---

## ⭐ **2. “Fan-Out + Gather” Parallel Execution Prompt**

```
Generate a parallel flow using:
- RequestFuture for calls
- RunAsync for computations
- Wait or WaitFirst for coordination
- No goroutines or channels
- Use REA.FanOutFail for aggregated error handling
```

---

## ⭐ **3. “External Approval Flow using Awakeables”**

```
Design a service where:
- Durable handler waits for external approval via Awakeable
- External system hits ingress API to resolve
- Add timeout path via restate.After
- Must avoid blocking ObjectContext
```

---

## ⭐ **4. “Durable Timeout Race”**

```
Model this flow:
- Wait for an external signal
- But timeout after 15 minutes
Use:
- Durable Promise OR Awakeable
- RacePromiseWithTimeout if using Workflow
```

---

## ⭐ **5. “Object with Deterministic State Machine”**

```
Design a Virtual Object with:
- Deterministic state transitions
- No sleeps in exclusive handlers
- All external calls moved to orchestrating service/workflow
- Clear state once session ends
```

---

## ⭐ **6. “Internal AUDIT: analyze determinism, concurrency, state safety”**

```
Audit this code and produce:
- Determinism violations
- Context misuse
- Side-effect placement errors
- Concurrency violations
- State misuse (shared vs exclusive)
- Deadlock risks
- Idempotency risks
Provide exact fixes.
```

---

## ⭐ **7. “Generate Diagrams from Restate Logic”**

```
Generate:
- Sequence diagram for durable operations
- Diagram showing control-plane vs data-plane
- Fan-out concurrency diagram
- Saga/Compensation flow diagram
```

---

# ------------------------------

# **PART 4 — META-PROMPTS (FOR MULTI-AGENT USE)**

# ------------------------------

## 🤖→🤖 “Refactor my code using Restate + REA standards”

```
You are the secondary agent specializing in enforcement of DOS_DON’Ts.

Refactor this code to:
- Use deterministic helpers
- Move all side effects to Run blocks
- Use durable futures
- Remove nondeterministic constructs
- Apply REA framework helpers
- Add full comments
```

---

## 🤖→🤖 “Design + Validate Loop”

```
Primary agent: generate Restate logic.
Secondary agent: validate every line using DOS_DON’Ts, REA rules, and LLM AGENTS.md.

Run one iteration automatically:
- LLM A proposes design
- LLM B audits determinism, concurrency, state, safety
```

---

# ------------------------------

# **END OF PROMPTS.MD**

# ------------------------------

If you want, I can also generate:

✅ README section describing how to use PROMPTS.MD  
✅ A version embedded in AGENTS rules format  
✅ A CLI “prompt runner” (e.g., prompt packs in JSON)  
✅ A Mermaid diagram summarizing prompt categories

Just tell me!