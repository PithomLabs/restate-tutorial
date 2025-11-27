# Comprehensive Explanation of framework.go

`framework.go` is a Go package that provides a high-level abstraction layer for the Restate Go SDK. It enforces best practices for building Restate services, including state management, saga patterns for distributed transactions, and type-safe operations. The code is organized into sections, each focusing on specific functionalities. Below is a detailed breakdown:

## Section 1: Core Types and Service Classification
This section defines fundamental types:
- `ServiceType`: An enumeration to distinguish between "control_plane" (orchestration, like sagas) and "data_plane" (business logic, state management).
- `Void`: A type alias for operations that return no data.
- `CallOption`: A struct for configuring inter-service calls, including idempotency keys and delays.

These types help in categorizing and structuring services in a Restate application.

## Section 2: External vs Internal Signaling
This handles coordination mechanisms:
- `WaitForExternalSignal`: Creates a durable awakeable for external events (e.g., waiting for user input).
- `ResolveExternalSignal` and `RejectExternalSignal`: Functions to complete or fail an external signal.
- `GetInternalSignal`: Retrieves a durable promise for internal workflow coordination.

These functions enable asynchronous communication, ensuring durability in distributed systems.

## Section 3: Type-Safe Durable State Management
This section provides safe access to Restate's key-value store:
- `State[T]`: A generic struct for getting, setting, or clearing state values, with runtime checks to ensure operations are performed in appropriate contexts (e.g., exclusive contexts for writes).
- Functions like `NewState`, `Get`, `Set`, `Clear`, and `ClearAll` enforce type safety and prevent errors by validating the context.

This promotes reliable state management while avoiding common pitfalls in concurrent environments.

## Section 4: Durable Saga Framework
This implements sagas for handling distributed transactions with compensation:
- `SagaCompensationFunc`, `SagaEntry`, and `SagaConfig`: Define compensation logic, entry structures, and retry configurations.
- `SagaFramework`: Manages sagas, including registering handlers, adding steps, and compensating if errors occur.
- Key methods include `Add` (for adding compensation steps), `CompensateIfNeeded` (executes compensations on failure), and utilities for dead-letter queue (DLQ) handling.

Sagas ensure that if a transaction fails, compensating actions can rollback changes, maintaining consistency.

## Section 5: Control Plane Service
This orchestrates workflows:
- `ControlPlaneService`: A struct for managing sagas and idempotency in orchestration.
- Methods like `Orchestrate`, `RegisterCompensation`, and `AddCompensationStep` wrap functions in saga contexts.
- Additional utilities include awaiting human approval (commented out) and generating idempotency keys.

It's designed for higher-level coordination, such as workflows involving multiple services.

## Section 6: Data Plane Service
This handles pure business logic:
- `DataPlaneService[I, O]`: A generic struct for stateless services.
- `NewStatelessService` and `Execute`: Run operations in a durable context, ensuring they are executed safely.

This focuses on the core logic without orchestration overhead.

## Section 7: Type-Safe Service Clients
This enables inter-service communication:
- `ServiceClient[I, O]`: A struct for type-safe calls.
- Methods like `Call` (request-response) and `Send` (fire-and-forget) handle interactions with options for idempotency and delays.

It abstracts away low-level details for reliable service invocation.

## Section 8: Concurrency Utilities
This provides tools for handling concurrent operations:
- `Race` and `Gather`: Functions to manage races between futures and gather results from multiple futures.

These utilities help in scenarios requiring parallel execution.

## Section 9: Validation Guards
This includes safeguards:
- `GuardRunContext`: Prevents misuse of contexts inside `restate.Run`.
- `ValidateServiceDefinition` and error wrappers: Ensure services adhere to rules.

These prevent common errors and enforce best practices.

## Section 10: Utility Functions
Internal helpers like `computeBackoff`, `deterministicStepID`, and `canonicalJSON` support the above sections, handling tasks like retries and data normalization.

Overall, `framework.go` simplifies building robust, durable services with Restate by abstracting complexities, ensuring type safety, and promoting error handling through sagas and state management.
