

# PROMPTS.MD: Effective Prompts for Restate Go SDK and Pithomlabs/REA Framework

This document provides a comprehensive collection of prompts for developers and LLMs to build resilient distributed applications using Restate Go SDK and Pithomlabs/REA Framework. These prompts are designed to ensure best practices, proper implementation patterns, and effective context engineering.

## 1. Service Selection and Design Patterns

### Prompts for Service Type Selection

```
"Given a use case for [describe your use case], which Restate service type (Basic Service, Virtual Object, or Workflow) would be most appropriate and why?"

"Design a Restate service architecture for a [specific domain] application. For each component, specify whether it should be a Basic Service, Virtual Object, or Workflow, and explain your reasoning."

"Compare and contrast the three Restate service types in terms of state management, concurrency model, and ideal use cases."

"Explain when to use a Virtual Object versus a Basic Service for [specific scenario], including the trade-offs of each approach."

"Design a workflow for a [multi-step business process] that requires human approval steps. Which Restate service type would you use and how would you structure the handlers?"
```

### Prompts for Service Architecture

```
"Design a microservice architecture using Restate for [specific business domain]. Identify the key services, their responsibilities, and the communication patterns between them."

"How would you implement a Saga pattern in Restate for [specific distributed transaction scenario]? Provide a detailed design of the compensating actions."

"Design an event-driven architecture using Restate for [specific use case], explaining how events flow between services and how state is managed."

"Create a design for a stateful entity management system using Restate Virtual Objects for [specific domain like e-commerce, IoT, etc.]."

"Explain how to structure a long-running business process in Restate that needs to survive service restarts, platform migrations, and extended pauses."
```

## 2. Deterministic Execution and Data Plane Separation

### Prompts for Deterministic Execution

```
"Explain the concept of deterministic execution in Restate and why it's critical for building resilient applications."

"Identify the non-deterministic operations in this [code snippet] and show how to properly wrap them using Restate's deterministic execution primitives."

"Write a Go function that performs [external operation] in a deterministic way using Restate's ctx.Run."

"Explain the Control Plane vs Data Plane separation in Restate and provide examples of operations that belong to each plane."

"How would you refactor this [non-deterministic code] to work correctly with Restate's durable execution model?"
```

### Prompts for Data Plane Operations

```
"Show how to properly implement a database operation in Restate using the rea framework's SafeRun or RunDo helpers."

"Write a complete example of making an HTTP call to an external API in a Restate handler, ensuring deterministic execution."

"Explain how to handle time-sensitive operations in Restate, including how to get the current time deterministically."

"Show how to generate a UUID for idempotency purposes in Restate, explaining why standard UUID generation won't work."

"Write a function that reads from a file system in a deterministic way using Restate's data plane isolation."
```

## 3. State Management

### Prompts for State Management Patterns

```
"Design a state management strategy for a [specific domain entity] using Restate Virtual Objects, including state schema, access patterns, and consistency guarantees."

"Explain how to implement a state machine using Restate, including state transitions, persistence, and error handling."

"Show how to implement optimistic concurrency control for state updates in Restate."

"Design a caching strategy using Restate's built-in state management versus external caching solutions."

"Explain how to handle large state objects in Restate, including strategies for partitioning and lazy loading."
```

### Prompts for State Access Patterns

```
"Write code examples showing the correct way to read and modify state in a Restate Virtual Object using the rea framework."

"Explain the difference between exclusive and shared handlers in Virtual Objects and when to use each."

"Show how to implement a read-only query endpoint for a Virtual Object state that can execute concurrently with write operations."

"Design a state migration strategy for evolving the state schema of a long-running Virtual Object."

"Explain how to clear state in Restate and when this is necessary, with code examples."
```

## 4. Communication Patterns

### Prompts for Internal Communication

```
"Design a communication pattern between Restate services for [specific scenario], including error handling, retries, and idempotency."

"Show how to implement a fan-out/fan-in pattern in Restate using the rea framework's concurrency helpers."

"Explain how to implement request-response communication between Virtual Objects without causing deadlocks."

"Write code showing how to call another Restate service from within a handler, including proper error handling."

"Design a publish-subscribe pattern using Restate services, explaining how to ensure message delivery and ordering."
```

### Prompts for External Communication

```
"Show how to integrate with an external system using Restate's Awakeable pattern, including both the waiting and resolving sides."

"Design a webhook handler using Restate that ensures exactly-once processing of incoming events."

"Explain how to securely expose a Restate service to external systems, including authentication and authorization."

"Write code showing how to call an external API from a Restate service, including proper error handling and retries."

"Design a pattern for handling long-running external operations in Restate, including how to handle timeouts and failures."
```

## 5. Awakeables and Promises

### Prompts for Awakeables

```
"Explain the difference between Awakeables and Promises in Restate, with specific use cases for each."

"Design a payment processing workflow using Restate Awakeables to handle callbacks from external payment gateways."

"Write a complete example of using Awakeables in a Restate service, including both creating and resolving the Awakeable."

"Show how to implement a human approval workflow using Awakeables, including how to handle timeouts and rejections."

"Explain how to test code that uses Awakeables, including strategies for mocking the external resolution."
```

### Prompts for Promises

```
"Design a multi-step workflow using Restate Promises to coordinate between different handlers."

"Write code showing how to use Promises in a Restate Workflow to wait for signals from other parts of the system."

"Explain how to implement a timeout for a Promise in Restate, including how to handle the timeout case."

"Show how to use Promises to implement a fan-in pattern in a Restate Workflow."

"Design a pattern for implementing a durable timer using Promises in Restate."
```

## 6. Error Handling and Sagas

### Prompts for Error Handling

```
"Explain the difference between transient errors and terminal errors in Restate, with examples of each."

"Design an error handling strategy for a Restate service that integrates with multiple external systems."

"Show how to implement proper error propagation in a Restate Workflow, including how to handle errors in sub-processes."

"Write code showing how to use TerminalError in Restate to stop retries for business logic failures."

"Explain how to implement a circuit breaker pattern using Restate's error handling mechanisms."
```

### Prompts for Sagas

```
"Design a Saga pattern for a [specific distributed transaction] using Restate, including all compensating actions."

"Write a complete example of implementing a Saga in Restate using Go's defer statement and the rea framework."

"Explain how to ensure that compensating actions in a Saga are idempotent, with code examples."

"Show how to handle partial failures in a Saga implemented with Restate, including how to determine which compensations to run."

"Design a pattern for monitoring and debugging Sagas in Restate, including how to track the state of each step."
```

## 7. Concurrency and Flow Control

### Prompts for Concurrency

```
"Explain how to implement parallel execution in Restate while maintaining deterministic behavior."

"Design a pattern for implementing a worker pool using Restate, including how to distribute work and collect results."

"Show how to implement a race condition in Restate using WaitFirst, including how to handle the winner and losers."

"Write code showing how to implement a map-reduce pattern using Restate's concurrency primitives."

"Explain how to implement backpressure in a Restate service that processes messages from a high-throughput source."
```

### Prompts for Flow Control

```
"Design a flow control pattern for a Restate service that needs to process messages at a specific rate."

"Show how to implement a priority queue using Restate, including how to prioritize different types of work."

"Write code showing how to implement a conditional workflow in Restate based on external state."

"Explain how to implement a loop with a delay in Restate that survives service restarts."

"Design a pattern for implementing a workflow that can be paused and resumed by external signals."
```

## 8. Security and Deployment

### Prompts for Security

```
"Design a security strategy for a Restate application that includes authentication, authorization, and encryption."

"Show how to implement secure communication between Restate services, including how to propagate identity."

"Explain how to secure a Restate service that is exposed to the public internet, including best practices for API security."

"Write code showing how to implement an authentication middleware for a Restate service using the rea framework."

"Design a pattern for implementing fine-grained access control in a Restate application."
```

### Prompts for Deployment

```
"Design a deployment strategy for a Restate application that includes versioning, blue-green deployments, and rollback procedures."

"Explain how to configure Restate for optimal performance in a Kubernetes environment, including resource allocation and networking."

"Show how to implement observability for a Restate application, including logging, metrics, and tracing."

"Write a deployment configuration for a Restate application that includes health checks, autoscaling, and disaster recovery."

"Design a testing strategy for a Restate application that includes unit tests, integration tests, and chaos engineering."
```

## 9. Code Generation and Refactoring

### Prompts for Code Generation

```
"Generate a complete Restate service for [specific domain] using the rea framework, including all necessary handlers and state management."

"Create a template for a Restate Workflow that includes error handling, logging, and state persistence."

"Generate code for a Virtual Object that implements [specific interface], including proper state management and concurrency control."

"Create a REST API wrapper for a Restate service that includes proper HTTP status codes, error handling, and documentation."

"Generate a set of unit tests for a Restate service that covers all happy paths and error scenarios."
```

### Prompts for Refactoring

```
"Refactor this [existing code] to use Restate's deterministic execution model, explaining each change."

"Convert this [existing state management code] to use Restate Virtual Objects, ensuring the same behavior with better resilience."

"Migrate this [existing workflow] to use Restate, explaining how to handle long-running operations and failures."

"Refactor this [existing microservice communication] to use Restate's internal client, ensuring proper error handling and retries."

"Convert this [existing saga implementation] to use Restate's primitives, ensuring better observability and reliability."
```

## 10. Advanced Patterns and Use Cases

### Prompts for Advanced Patterns

```
"Design a CQRS pattern using Restate, explaining how to separate command and query responsibilities."

"Implement an event sourcing pattern using Restate, including how to store and replay events."

"Design a distributed locking mechanism using Restate Virtual Objects, explaining how to handle lock contention and failures."

"Implement a distributed cache invalidation pattern using Restate, ensuring consistency across multiple services."

"Design a rate limiting service using Restate, explaining how to handle distributed state and high throughput."
```

### Prompts for Specific Use Cases

```
"Design a complete e-commerce checkout process using Restate, including inventory management, payment processing, and order fulfillment."

"Implement a real-time collaboration system using Restate, explaining how to handle concurrent updates and conflict resolution."

"Design a IoT data processing pipeline using Restate, explaining how to handle high-volume data streams and device state."

"Implement a multi-tenant SaaS application using Restate, explaining how to isolate tenant data and handle resource allocation."

"Design a financial trading system using Restate, explaining how to ensure transaction consistency and handle market data."
```

## 11. Testing and Debugging

### Prompts for Testing

```
"Design a comprehensive testing strategy for a Restate application, including unit tests, integration tests, and end-to-end tests."

"Show how to write unit tests for Restate handlers that involve external dependencies, including mocking strategies."

"Explain how to test a Saga implementation in Restate, including how to verify compensating actions."

"Write code showing how to test a Restate Workflow that includes timers, delays, and external signals."

"Design a chaos engineering experiment for a Restate application, explaining how to inject failures and verify resilience."
```

### Prompts for Debugging

```
"Explain how to debug a Restate application that is experiencing non-deterministic behavior, including tools and techniques."

"Show how to add observability to a Restate service for debugging purposes, including logging, metrics, and tracing."

"Design a pattern for debugging long-running Workflows in Restate, including how to inspect state and execution history."

"Explain how to troubleshoot common issues in Restate, such as deadlocks, state corruption, and performance problems."

"Write a debugging guide for developers new to Restate, including common pitfalls and how to avoid them."
```

## 12. Performance Optimization

### Prompts for Performance

```
"Analyze this Restate service for performance bottlenecks and suggest optimizations."

"Design a performance testing strategy for a Restate application, including load testing and stress testing."

"Explain how to optimize state access in Restate for high-throughput scenarios."

"Show how to implement batching in Restate to improve performance for high-volume operations."

"Design a pattern for implementing caching in a Restate application, explaining how to maintain cache consistency."
```

### Prompts for Scalability

```
"Design a scaling strategy for a Restate application, including horizontal scaling, load balancing, and resource allocation."

"Explain how to scale a Restate application that uses Virtual Objects, including key distribution and partitioning strategies."

"Show how to implement auto-scaling for a Restate application running in Kubernetes."

"Design a pattern for implementing backpressure in a Restate application to prevent overload."

"Explain how to monitor and tune the performance of a Restate deployment, including key metrics and thresholds."
```

## 13. Migration and Integration

### Prompts for Migration

```
"Design a migration strategy for moving an existing application to Restate, including incremental migration and rollback procedures."

"Explain how to integrate Restate with existing systems, including data synchronization and API compatibility."

"Show how to implement a Strangler Fig pattern using Restate to gradually replace an existing system."

"Design a pattern for implementing a compatibility layer between an existing system and a new Restate implementation."

"Explain how to handle data migration when moving to Restate, including how to ensure data consistency and minimize downtime."
```

### Prompts for Integration

```
"Design an integration pattern for connecting Restate with external message brokers like Kafka or RabbitMQ."

"Show how to implement a database connector for Restate that ensures deterministic behavior."

"Explain how to integrate Restate with existing authentication and authorization systems."

"Design a pattern for implementing event-driven integration between Restate and external systems."

"Show how to implement a REST API gateway for a Restate application, including routing, authentication, and rate limiting."
```

## 14. Framework-Specific Prompts (REA)

### Prompts for REA Framework Usage

```
"Explain the key abstractions provided by the pithomlabs/rea framework and how they simplify Restate development."

"Show how to use the rea framework's SafeRun helper to wrap non-deterministic operations."

"Design a Virtual Object using the rea framework, including state management and handler implementation."

"Explain how to use the rea framework's concurrency helpers like FanOut and FanOutFail."

"Show how to implement a Saga using the rea framework's Saga helper, including compensating actions."
```

### Prompts for REA Best Practices

```
"Explain the policy enforcement mechanisms in the rea framework and how to configure them for different environments."

"Show how to use the rea framework's deterministic helpers for generating UUIDs and timestamps."

"Design a security strategy using the rea framework's security middleware."

"Explain how to use the rea framework's state access helpers to ensure type safety and proper context usage."

"Show how to implement a workflow using the rea framework's workflow helpers, including timers and signals."
```

## 15. LLM-Specific Prompts

### Prompts for LLM Code Generation

```
"Generate a complete Restate service in Go using the rea framework for [specific use case], ensuring all non-deterministic operations are properly wrapped."

"Refactor this [existing Go code] to work with Restate's deterministic execution model, explaining each change."

"Write a comprehensive test suite for this Restate service, covering all happy paths and error scenarios."

"Generate documentation for this Restate service, including API documentation, deployment instructions, and troubleshooting guide."

"Create a step-by-step tutorial for implementing [specific pattern] in Restate, with code examples and explanations."
```

### Prompts for LLM Analysis

```
"Analyze this Restate service for potential issues with determinism, concurrency, or error handling, and suggest improvements."

"Explain the execution flow of this Restate Workflow, including how state changes and how errors are handled."

"Compare this implementation of [pattern] with best practices for Restate, identifying any deviations and their implications."

"Review this Restate deployment configuration for security issues, performance bottlenecks, or scalability limitations."

"Explain how this Restate service would behave under failure conditions, including how it would recover and what data might be lost."
```

These prompts are designed to help developers and LLMs create robust, resilient distributed applications using Restate Go SDK and the Pithomlabs/REA Framework. They cover a wide range of scenarios from basic service implementation to advanced patterns and deployment strategies.