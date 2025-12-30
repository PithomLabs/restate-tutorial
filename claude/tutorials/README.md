# Restate Go Tutorial Series

> **Master distributed systems with durable execution in Go**

Build production-ready, resilient backend applications using [Restate](https://restate.dev) - the durable execution framework that makes distributed systems simple.

---

## 🎯 What You'll Learn

This comprehensive tutorial series teaches you how to build **fault-tolerant distributed systems** without the complexity of traditional approaches.

**By the end of this series, you'll be able to:**

✅ Build services that automatically recover from failures  
✅ Implement saga patterns for distributed transactions  
✅ Handle idempotency and exactly-once semantics  
✅ Integrate with external APIs reliably  
✅ Orchestrate complex microservices workflows  
✅ Deploy production-ready systems with observability  
✅ Secure your applications with industry best practices  

---

## 🚀 Why Restate?

Traditional distributed systems are hard:
- ❌ Manual failure recovery and retries
- ❌ Complex state management across services
- ❌ Race conditions and concurrency bugs
- ❌ Distributed transaction nightmares
- ❌ Idempotency is an afterthought

**Restate makes it simple:**
- ✅ Automatic failure recovery built-in
- ✅ Durable execution - code runs to completion
- ✅ Built-in state management (no external DB needed)
- ✅ Saga patterns in plain code
- ✅ Idempotency by default

---

## 👨‍💻 Who Is This For?

**Perfect if you:**
- Have 1-5 years of Go experience
- Want to learn distributed systems
- Need to build reliable microservices
- Are tired of manual error handling
- Want production-ready patterns

**Not required:**
- Deep distributed systems knowledge
- Previous Restate experience
- Kubernetes expertise (we'll cover it!)

---

## 📚 Tutorial Modules

### Foundation (Modules 1-3)

#### [Module 1: Getting Started](./01-getting-started/)
**Learn:** Restate fundamentals, setup, first service  
**Build:** Hello World service with durable execution  
**Time:** 1-2 hours  

#### [Module 2: Resilient Stateless APIs - Side Effects](./02-side-effects/)
- **Learn:** Understand what side effects are and why they need special handling
- **Build:** Build a real-world API aggregation service
- **Time:** 2-3 hours  

#### [Module 3: State Management](./04-virtual-objects/)
**Learn:** Virtual Objects, key-value state  
**Build:** Shopping cart service  
**Time:** 2-3 hours  

---

### Core Patterns (Modules 4-6)

#### [Module 4: Workflows - Long-Running Orchestration](./05-workflows/)
**Learn:** How workflows extend Virtual Objects with special capabilities for long-running orchestrations 
**Build:** Durable workflows with human-in-the-loop and async await patterns
**Time:** 2-3 hours  

#### [Module 5: Idempotency](./07-idempotency/)
**Learn:** Exactly-once semantics, idempotency keys  
**Build:** Payment deduplication system  
**Time:** 2 hours  

#### [Module 6: Sagas & Distributed Transactions](./06-sagas/)
**Learn:** Saga pattern, compensation, rollback  
**Build:** Travel booking with multi-step transactions  
**Time:** 3-4 hours  

---

### Integration (Modules 7-9)

#### [Module 7: Testing Restate Applications](./07-testing/)
**Learn:** Unit testing, mock dependencies, integration testing  
**Build:** Test-driven development (TDD) with Restate
**Time:** 3-4 hours  

#### [Module 8: External Integration](./08-external-integration/)
**Learn:** Third-party API calls, webhooks, awakeables  
**Build:** Stripe payment integration  
**Time:** 2-3 hours  

#### [Module 9: Microservices Orchestration](./09-microservices/)
**Learn:** Workflows, complex orchestration, parallel execution  
**Build:** E-commerce checkout workflow  
**Time:** 3-4 hours  

---

### Production (Modules 10-12)

#### [Module 10: Observability](./10-observability/)
**Learn:** Metrics, logging, tracing, dashboards  
**Build:** Full observability stack with Prometheus & Grafana  
**Time:** 3-4 hours  

#### [Module 11: Security](./11-security/)
**Learn:** Authentication, authorization, encryption  
**Build:** Secure user management system  
**Time:** 3-4 hours  

#### [Module 12: Production & Deployment](./12-production/)
**Learn:** Docker, Kubernetes, HA, disaster recovery  
**Build:** Production-ready deployment  
**Time:** 4-5 hours  

---

## ⏱️ Time Commitment

**Total:** ~35-45 hours for complete mastery  

**Recommended Pace:**
- **Intensive:** 1 module/day (2 weeks total)
- **Balanced:** 2 modules/week (6 weeks total)
- **Relaxed:** 1 module/week (3 months total)

---

## 🛠️ Prerequisites

### Required Software

```bash
# Go 1.22 or higher
go version  # Should be 1.22+

# Docker is optional (for running Restate locally) 
# You can download Restate binary at https://github.com/restatedev/restate/releases
docker --version

# Git
git --version
```

### Installation

```bash
# Install Restate SDK for Go
go get github.com/restatedev/sdk-go

# Run Restate server (local development)
docker run -d --name restate \
  -p 8080:8080 -p 9070:9070 \
  restatedev/restate:latest
```

### Optional (for later modules)

- **Module 7:** Kafka (Docker Compose provided)
- **Module 10:** Prometheus & Grafana (Docker Compose provided)
- **Module 12:** Kubernetes (minikube or kind)

---

## 🎓 How to Use This Tutorial

### 1. Module Structure

Each module contains:

```
XX-module-name/
├── README.md           # Overview & learning objectives
├── 01-concepts.md      # Theory & explanations
├── 02-hands-on.md      # Step-by-step code tutorial
├── 03-validation.md    # Test your implementation
└── 04-exercises.md     # Practice challenges
```

### 2. Recommended Workflow

**For each module:**

1. **Read** `README.md` - Understand what you'll learn (5 mins)
2. **Study** `01-concepts.md` - Learn the theory (15-30 mins)
3. **Build** `02-hands-on.md` - Write code step-by-step (1-2 hours)
4. **Validate** `03-validation.md` - Test your work (15-30 mins)
5. **Practice** `04-exercises.md` - Reinforce learning (1-2 hours)

### 3. Learning Paths

**🏃 Fast Track (Core Basics)**
Modules: 1, 2, 3, 4, 6  
*Get productive quickly with essential patterns*

**🎯 Full Stack Developer**
Modules: 1-9  
*Build complete applications end-to-end*

**🚀 Production Engineer**
All modules (1-12)  
*Deploy and maintain production systems*

**🔧 Integration Specialist**
Modules: 1-5, 7-8  
*Focus on event-driven and API integration*

---

## 💻 Code Examples

All code examples are **production-ready** and follow best practices:

- ✅ Proper error handling
- ✅ Comprehensive testing
- ✅ Clear documentation
- ✅ Idiomatic Go style
- ✅ Real-world patterns

**SDK Version:** `github.com/restatedev/sdk-go v0.13.1`  
**Go Version:** 1.22+  

---

## 📖 Additional Resources

### Official Restate Documentation
- [Restate Docs](https://docs.restate.dev)
- [Go SDK Reference](https://pkg.go.dev/github.com/restatedev/sdk-go)
- [Restate Examples](https://github.com/restatedev/examples)

### Community
- [Restate Discord](https://discord.gg/skW3AZ6uGd)
- [GitHub Discussions](https://github.com/restatedev/restate/discussions)
- [Twitter](https://twitter.com/restatedev)

### Learning More
- [Distributed Systems in Go](https://www.oreilly.com/library/view/distributed-services-with/9781680509557/)
- [The Saga Pattern](https://microservices.io/patterns/data/saga.html)
- [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)

---

## 🎯 Learning Objectives by Module

| Module | Key Concepts | Skills Gained |
|--------|-------------|---------------|
| **1** | Restate basics, handlers | Setup, first service |
| **2** | Journaling, replay | Understand durable execution |
| **3** | Virtual Objects, state | Build stateful services |
| **4** | RPC patterns, messaging | Service communication |
| **5** | Idempotency keys | Exactly-once processing |
| **6** | Saga pattern, compensation | Distributed transactions |
| **7** | Event sourcing, CQRS | Event-driven architecture |
| **8** | External APIs, webhooks | Third-party integration |
| **9** | Workflows, orchestration | Complex business logic |
| **10** | Metrics, tracing, logs | Production observability |
| **11** | Auth, encryption, security | Secure applications |
| **12** | K8s, HA, DR | Production deployment |

---

## 🚦 Getting Started Right Now

### Quick Start (5 minutes)

```bash
# 1. Clone or create project directory
mkdir restate-tutorial
cd restate-tutorial

# 2. Initialize Go module
go mod init restate-tutorial

# 3. Install Restate SDK
go get github.com/restatedev/sdk-go@v0.13.1

# 4. Start Restate server
docker run -d --name restate \
  -p 8080:8080 -p 9070:9070 \
  restatedev/restate:latest

# 5. Start with Module 1!
```

**→ [Begin with Module 1: Getting Started](./01-getting-started/)**

---

## 📊 Progress Tracking

Track your progress using [PROGRESS.md](./PROGRESS.md):

```markdown
- [x] Module 1: Getting Started
- [x] Module 2: Durable Execution
- [ ] Module 3: State Management
- [ ] Module 4: Service Communication
...
```

---

## 🎁 What's Included

### All 12 Modules
Complete, production-ready tutorial covering:
- Getting Started
- Durable Execution
- State Management
- Service Communication
- Idempotency
- Sagas & Transactions
- Event-Driven Architecture
- External Integration
- Orchestration
- Observability
- Security
- Production Deployment

**Status:** ✅ All modules complete!

---

## 🤝 Contributing

Found a bug? Have a suggestion?

- **Issues:** Report problems or improvements
- **Discussions:** Ask questions
- **Pull Requests:** Contributions welcome!

---

## 📝 License

**Tutorial Content:** MIT License  
**Code Examples:** MIT License  

Feel free to use, modify, and share with attribution.

---

## 🙏 Acknowledgments

**Built with:**
- [Restate](https://restate.dev) - Durable execution platform
- [Go](https://go.dev) - Programming language
- Community feedback and contributions

---

## 💡 Why This Tutorial Series?

**Comprehensive**
- 12 modules covering foundations to production
- 48 lesson files with extensive code examples
- Real-world patterns, not toy examples

**Practical**
- Learn by building actual applications
- Every module includes hands-on coding
- Validation tests ensure you understand

**Production-Ready**
- Best practices from day one
- Security, observability, deployment covered
- Code you can actually use in production

**Beginner-Friendly**
- Clear explanations of complex concepts
- Step-by-step instructions
- No assumed knowledge of distributed systems

---

## 🎉 Ready to Begin?

Stop wrestling with distributed systems complexity. Learn how to build reliable, fault-tolerant applications with durable execution.

**Your journey starts here:**

### [→ Module 1: Getting Started](./01-getting-started/)

---

**Happy Learning! 🚀**

---

## 📚 Additional Info

### Module Breakdown

**Total Modules:** 12  
**Estimated Time:** 35-45 hours  
**Difficulty:** Beginner to Advanced  
**SDK Version:** v0.13.1  

### Business Information

Interested in the business model behind this tutorial series?

**→ [View Business Plan](./bizplan/)** 

Learn how this was built as a sustainable digital product business using free tiers of SaaS tools.

---

*Last updated: 2024-11-22*  
*Tutorial Series Version: 1.0*  
*SDK Version: v0.13.1*

---

**Questions? Feedback? Let's connect!** 💬
