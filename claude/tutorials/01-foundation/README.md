# Module 01: Hello Durable World - Your First Restate Service

> **Build your first durable service and understand the fundamentals of durable execution**

## 🎯 Learning Objectives

By the end of this module, you will:

- ✅ Understand what durable execution means and why it matters
- ✅ Differentiate between the three service types (Service, Virtual Object, Workflow)
- ✅ Create and deploy a Basic Service
- ✅ Make durable service calls with automatic retries
- ✅ Observe journaling and replay in action
- ✅ Handle errors properly (Terminal vs Retriable)

## 📚 Module Contents

| File | Description | Time |
|------|-------------|------|
| **[01-concepts.md](./01-concepts.md)** | Core concepts and theory | 15 min |
| **[02-hands-on.md](./02-hands-on.md)** | Step-by-step tutorial | 30 min |
| **[03-validation.md](./03-validation.md)** | Testing and verification | 10 min |
| **[04-exercises.md](./04-exercises.md)** | Practice exercises | 15 min |

## 🎓 Prerequisites

- Completed [Module 00: Prerequisites & Setup](../00-prerequisites/README.md)
- Restate server running on ports 8080/9080
- Go development environment ready

## 🏗️ What You'll Build

**Project: Durable Greeting Service**

A simple but powerful greeting service that demonstrates:
- Automatic journaling of operations
- Retry on failures
- Request-response communication
- Error handling patterns

```
Input: {"name": "Alice", "shouldFail": false}
         ↓
    [Greeting Service]
         ↓
Output: "Hello, Alice! You're awesome!"
```

## 🗺️ Learning Path

```
1. Concepts (Theory)
   ↓
   - What is durable execution?
   - Service types overview
   - Context and journaling
   
2. Hands-On (Practice)
   ↓
   - Create Basic Service
   - Implement greeting logic
   - Add error handling
   - Deploy and test
   
3. Validation (Verify)
   ↓
   - Test retry behavior
   - Observe journaling
   - Verify error handling
   
4. Exercises (Reinforce)
   ↓
   - Extend the service
   - Add new features
   - Handle edge cases
```

## 🚀 Quick Start

If you're ready to dive in:

1. **Read:** [01-concepts.md](./01-concepts.md) for theory
2. **Code:** Follow [02-hands-on.md](./02-hands-on.md) step-by-step
3. **Test:** Verify with [03-validation.md](./03-validation.md)
4. **Practice:** Complete [04-exercises.md](./04-exercises.md)

## 💡 Key Takeaways

After this module, you'll understand:

> **Durable Execution** means your code runs to completion, even if:
> - Your service crashes mid-execution
> - Network requests fail temporarily
> - External services are temporarily unavailable
>
> Restate automatically handles retries and maintains execution state.

## 📁 Code Structure

```
01-foundation/
├── README.md           ← You are here
├── 01-concepts.md      ← Theory
├── 02-hands-on.md      ← Tutorial
├── 03-validation.md    ← Testing
├── 04-exercises.md     ← Practice
├── code/
│   ├── main.go         ← Complete service
│   ├── service.go      ← Service logic
│   └── go.mod          ← Dependencies
└── solutions/
    ├── exercise1.go    ← Exercise solutions
    ├── exercise2.go
    └── exercise3.go
```

## 🎯 Success Criteria

You've mastered this module when you can:

- [ ] Explain what durable execution means
- [ ] Create a Basic Service from scratch
- [ ] Register and call services via Restate
- [ ] Differentiate Terminal vs Retriable errors
- [ ] Observe and understand journaling
- [ ] Complete all exercises independently

## ⏱️ Time Commitment

- **Minimum:** 30 minutes (concepts + hands-on)
- **Recommended:** 70 minutes (all materials + exercises)
- **Mastery:** 2 hours (with experimentation)

## 🔗 Next Module

After completing this module, continue to:

👉 **[Module 2: Resilient Stateless APIs](../02-side-effects/README.md)**

---

**Ready to start?** Begin with [Concepts](./01-concepts.md)!
