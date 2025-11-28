. Core Restate concepts needed for this use case 2. How this differs from my current approach 3. Step-by-step implementation guide 4. Common mistakes to avoid 5. How to test and deploy

**Learning Style:** [Hands-on examples | Conceptual explanations | Comparisons to familiar tech]

````

### For AI/LLM Agents

```markdown
You are an AI coding agent working on a Restate project. Before generating any code:

**Mandatory Checks:**
1. Verify service type (Basic Service | Virtual Object | Workflow)
2. Confirm all external I/O is wrapped in ctx.Run or framework.RunDo
3. Check context type matches handler signature
4. Validate no ctx usage inside Run blocks
5. Ensure error handling includes TerminalError where appropriate

**Code Generation Standards:**
- Use pithomlabs/rea framework utilities
- Include comprehensive comments
- Add logging via ctx.Log()
- Generate deterministic IDs via framework helpers
- Include both happy and error paths

**Validation:**
After generation, run mental checklist against DOS_DONTS_MEGA.MD and DOS_DONTS_REA.MD

Now proceed with: [Specific task]
````

### For DevOps Engineers

```markdown
I'm a DevOps engineer setting up Restate infrastructure. Help me:

**Current Stack:**
- Orchestration: [Kubernetes/ECS/Lambda]
- Monitoring: [Prometheus/Datadog/CloudWatch]
- Logging: [ELK/Splunk/CloudWatch Logs]

**Deployment Goal:**
[e.g., "Production-ready Restate cluster with HA"]

**Please provide:**
1. Infrastructure architecture diagram (describe textually)
2. Deployment manifests (K8s/Terraform)
3. Monitoring setup (metrics to track)
4. Logging configuration (structured JSON logs)
5. Backup and disaster recovery strategy
6. Security hardening checklist
7. Scaling guidelines

**Constraints:**
- Budget: [Development/Production]
- Scale: [Expected RPS, number of services]
- Compliance: [Any regulatory requirements]
```

### For QA Engineers

````markdown
I'm a QA engineer testing Restate services. Help me create:

**Test Goal:** [E.g., "Verify exactly-once semantics under failures"]

**Service Under Test:**
```go
[Paste handler code]
````

**Test Requirements:**

1. Functional correctness
2. Failure recovery (crash, network partition)
3. Concurrency safety (race conditions)
4. Idempotency verification
5. Performance under load

**Please generate:**

1. Test plan (scenarios to cover)
2. Test implementation (code)
3. Chaos scenarios (how to inject failures)
4. Assertions (what to verify)
5. Performance benchmarks (expected latency/throughput)

**Test Environment:** [Local docker-compose | Staging K8s cluster]

````

---

## Advanced Prompt Patterns

### Pattern: "Explain Code Transformation"

```markdown
Transform this code from [Source Pattern] to [Target Pattern], explaining each step:

**Source Code:**
```go
[Paste code]
````

**Transformation Goal:** [e.g., "Convert synchronous calls to concurrent with timeout"]

**Explain:**

1. Why this transformation improves the design
2. What Restate primitives enable this
3. What could go wrong without proper transformation
4. How to test the transformation

**Output:**

- Transformed code with inline comments explaining changes
- Before/After comparison
- Migration checklist

````

### Pattern: "Design Pattern Comparison"

```markdown
Compare these two implementation approaches for [Use Case]:

**Approach A:**
```go
[Code for Approach A]
````

**Approach B:**

```go
[Code for Approach B]
```

**Compare on:**

1. Determinism compliance
2. Concurrency characteristics
3. Failure recovery behavior
4. Performance implications
5. Code maintainability
6. Alignment with Restate best practices

**Recommend:** Which approach and why

````

### Pattern: "Anti-Pattern Detection"

```markdown
Analyze this code for Restate anti-patterns:

```go
[Paste code]
````

**Scan for:**

1. Determinism violations
2. Concurrency issues (goroutines with durable ops)
3. State management errors
4. Deadlock risks
5. Missing error handling
6. Performance bottlenecks

**For each anti-pattern found:**

- Explain why it's problematic
- Show correct implementation
- Estimate severity
- Reference documentation section

````

### Pattern: "Architecture Evolution"

```markdown
My Restate architecture needs to evolve:

**Current State:**
[Describe current services, handlers, state management]

**New Requirements:**
[Describe new features, scale requirements, compliance needs]

**Please:**
1. Analyze current architecture gaps
2. Propose evolution path (phased changes)
3. Identify backward compatibility concerns
4. Show migration strategy for stateful services
5. Estimate effort and risk for each phase

**Constraints:**
- Cannot have downtime
- Must support old clients during transition
- Budget: [Limited/Moderate/Ample]
````

---

## Conclusion

This prompt engineering guide provides a comprehensive framework for effectively communicating with LLMs when building Restate applications. The key principles are:

1. **Provide Rich Context:** Always include relevant documentation (DOS_DONTS_*.MD, AGENTS.MD)
2. **Be Specific:** Clearly state service type, requirements, and constraints
3. **Request Validation:** Ask LLMs to self-check against Restate principles
4. **Iterate:** Use progressive refinement for complex tasks
5. **Learn Patterns:** Recognize and reuse proven prompt patterns

By following these patterns, developers can leverage LLMs to accelerate Restate development while maintaining the determinism, durability, and consistency guarantees that make Restate powerful.

---

**Document Version:** 1.0  
**Last Updated:** 2025-11-28  
**Feedback:** Submit issues or improvements to the `pithomlabs/rea` repository