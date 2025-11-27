# Comparison: Flyt vs Rea for Workflow Automation

**Analysis Date:** 2025-11-23  
**Frameworks Compared:**
- [Flyt](https://github.com/mark3labs/flyt) - mark3labs/flyt
- [Rea](https://github.com/pithomlabs/rea) - pithomlabs/rea

---

## 🎯 Core Purpose & Philosophy

### Flyt (mark3labs/flyt)
- **Focus**: Minimalist, general-purpose workflow orchestration
- **Architecture**: Node-based directed graph with action-driven routing
- **Dependencies**: Zero external dependencies
- **Use Case**: In-process workflow automation for Go applications
- **State Model**: In-memory shared store (non-persistent)

### Rea (pithomlabs/rea)  
- **Focus**: Distributed durable execution with Restate SDK
- **Architecture**: Control Plane/Data Plane separation with type-safe abstractions
- **Dependencies**: Built on top of Restate Go SDK
- **Use Case**: Resilient distributed systems with automatic durability
- **State Model**: Durable, replicated state with automatic recovery

---

## 🏗️ Architecture Comparison

| Aspect | Flyt | Rea |
|--------|------|-----|
| **Execution Model** | In-process, synchronous nodes | Distributed, durable async/await |
| **State Persistence** | Ephemeral (SharedStore) | Durable (Restate journaling) |
| **Failure Recovery** | Retry logic per node | Automatic replay from last checkpoint |
| **Workflow Definition** | Node graphs with actions | Service/Object/Workflow handlers |
| **Concurrency** | Worker pools, batch processing | Futures, Wait/WaitFirst patterns |
| **Error Handling** | Per-node retries + fallbacks | Terminal vs retryable errors + Sagas |

---

## ⚙️ Workflow Automation Features

### Flyt Strengths

1. **Simple Node Model** - 3-phase execution (Prep → Exec → Post)
2. **Batch Processing** - First-class support with `BatchNode` and configurable concurrency
3. **Action-Based Routing** - Dynamic workflow paths based on results
4. **Nested Flows** - Flows can be composed as reusable nodes
5. **Custom Node Types** - Easy to extend with interfaces (RetryableNode, FallbackNode)
6. **Worker Pools** - Built-in concurrent processing
7. **Lightweight** - No infrastructure required

**Example:**
```go
node := flyt.NewNode(
    flyt.WithExecFunc(func(ctx context.Context, prepResult flyt.Result) (flyt.Result, error) {
        return flyt.R(callAPI()), nil
    }),
    flyt.WithMaxRetries(3),
    flyt.WithWait(time.Second),
)
```

### Rea Strengths

1. **Durable Saga Framework** - Distributed transactions with automatic compensation
2. **Workflow Retention Policies** - State lifecycle management (WorkflowConfig)
3. **Promise-Based Coordination** - Human-in-the-loop workflows
4. **Type-Safe State Management** - Runtime-enforced read/write permissions
5. **Anti-Pattern Protection** - Prevents common distributed system mistakes
6. **Automatic Recovery** - Workflows resume after failures/restarts
7. **Idempotency Management** - Framework-level deduplication

**Example:**
```go
saga := framework.NewSaga(ctx, "payment-flow", nil)
saga.Register("charge_card", refundCard) // Compensation registered before action
saga.Add("charge_card", chargeData, false)
defer saga.CompensateIfNeeded(&err) // Automatic rollback on failure
```

---

## 🔄 Workflow Patterns Comparison

### Sequential Execution
- **Flyt**: Connect nodes with `flow.Connect(nodeA, "action", nodeB)`
- **Rea**: Call services sequentially via `client.Call(ctx, input)`

### Conditional Branching
- **Flyt**: Return different actions from Post phase → routes to different nodes
- **Rea**: Standard Go if/else with durable state checkpoints

### Parallel Execution
- **Flyt**: `BatchNode` with concurrency control
- **Rea**: `RequestFuture()` + `restate.Wait()` for fan-out/fan-in

### Error Recovery
- **Flyt**: Node-level retries + ExecFallback for degraded functionality
- **Rea**: Saga compensations for distributed rollback

### Long-Running Workflows
- **Flyt**: Limited to process lifetime
- **Rea**: **Superior** - Workflows can run for days/months with durable timers and promises

---

## 💡 Key Differentiators

| Feature | Flyt | Rea |
|---------|------|-----|
| **Durability** | ❌ No | ✅ Yes (automatic journaling) |
| **Distributed** | ❌ Single process | ✅ Yes (Restate cluster) |
| **State Retention** | ❌ Memory-only | ✅ Configurable retention policies |
| **Human-in-Loop** | ⚠️ Requires external coordination | ✅ Built-in via Promises/Awakeables |
| **Compensation** | ❌ Manual | ✅ Automatic Saga framework |
| **Learning Curve** | ✅ Low (simple API) | ⚠️ Medium (Restate concepts) |
| **Infrastructure** | ✅ None | ⚠️ Requires Restate runtime |
| **Batch Processing** | ✅ First-class support | ⚠️ Via futures pattern |

---

## 🎯 When to Choose Each

### Choose Flyt if you need:
- ✅ Simple in-process workflow orchestration
- ✅ Zero infrastructure dependencies
- ✅ Batch processing with controlled concurrency
- ✅ Quick prototyping or small applications
- ✅ Full control over execution environment
- ❌ **Don't** need durability or distributed execution

### Choose Rea if you need:
- ✅ Distributed, fault-tolerant workflows
- ✅ Long-running processes (hours/days/months)
- ✅ Automatic recovery from failures
- ✅ Complex compensation logic (Sagas)
- ✅ Human-in-the-loop workflows
- ✅ Strong consistency guarantees
- ✅ Event-driven microservices architecture
- ❌ **Don't** mind running Restate infrastructure

---

## 🚀 Hybrid Approach

You could potentially use **both** frameworks together:
- **Rea** for durable orchestration and distributed coordination
- **Flyt** as a data plane execution engine within Rea services

**Example:**
```go
// Rea controls distributed workflow
func (OrderWorkflow) Run(ctx restate.WorkflowContext, order Order) error {
    // Use RunDo to wrap Flyt execution
    result, err := framework.RunDo(ctx, func(rc restate.RunContext) (Result, error) {
        // Flyt handles complex batch processing
        flow := createOrderProcessingFlow()
        shared := flyt.NewSharedStore()
        shared.Set("order", order)
        return flyt.Run(context.Background(), flow, shared)
    })
    // Continue with Rea's durable orchestration
}
```

---

## 📊 Summary Table

| Criterion | Flyt | Rea |
|-----------|------|-----|
| **Simplicity** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Durability** | ⭐ | ⭐⭐⭐⭐⭐ |
| **Distributed** | ⭐ | ⭐⭐⭐⭐⭐ |
| **Batch Processing** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Error Recovery** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Type Safety** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Setup Complexity** | ⭐⭐⭐⭐⭐ | ⭐⭐ |
| **Long-Running** | ⭐ | ⭐⭐⭐⭐⭐ |

---

## 🎓 Conclusion

**Flyt** and **Rea** solve different problems in the workflow automation space:

### Flyt
Excellent for **synchronous, in-process workflow automation** with a clean, minimal API. It's perfect for applications that need structured flow control without distributed system complexity. The framework shines in:
- Batch processing scenarios
- Simple multi-step workflows
- Applications where durability isn't critical
- Rapid prototyping and development

### Rea
Designed for **resilient distributed systems** where workflows must survive failures, run across multiple services, and maintain strong consistency guarantees. It brings enterprise-grade durability to Go applications with:
- Automatic failure recovery
- Distributed transaction management via Sagas
- Long-running business processes
- Human-in-the-loop workflows
- Strong type safety and anti-pattern protection

### Rea's Unique Value Proposition

The **Rea framework has significant advantages** for:
1. Building fault-tolerant microservices
2. Implementing complex distributed transactions (Sagas)
3. Managing long-running business processes
4. Providing automatic recovery without operational overhead
5. Preventing common distributed system anti-patterns
6. Making Restate SDK accessible to developers

---

## 📚 Additional Resources

### Flyt
- Repository: https://github.com/mark3labs/flyt
- Website: http://go-flyt.dev/
- Cookbook Examples: Agent, Chat, LLM Streaming, MCP integrations

### Rea
- Repository: https://github.com/pithomlabs/rea
- Categories: 12 comprehensive feature categories
- Documentation: Complete guides for Sagas, Concurrency, Security, and more
- Built on: Restate Go SDK

---

## 🤔 Recommendation

**For Restate Tutorial Development**: **Rea is clearly the superior choice** as it:
- Provides the abstractions that make Restate accessible to developers
- Prevents common anti-patterns through framework-level guardrails
- Offers production-ready patterns (Sagas, type-safe state, idempotency)
- Demonstrates best practices for distributed durable execution
- Has comprehensive documentation and guides

**For Simple Workflows**: Flyt is an excellent choice when durability and distribution aren't requirements.

**For Maximum Power**: Consider using both - Rea for coordination, Flyt for complex local processing within durable steps.
