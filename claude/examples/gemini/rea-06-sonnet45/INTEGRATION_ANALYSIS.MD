# REA-05 Integration Analysis

**Analysis Date:** 2025-11-27  
**Subject:** Cross-folder dependency and coupling analysis  
**Purpose:** Determine integration tightness before refactoring

---

## Executive Summary

The **rea-05** project demonstrates **ZERO COUPLING** between the main application folders (services/ingress) and the supporting folders (config/models/observability). Each folder operates as an **isolated module** with duplicated types and no cross-folder imports.

**Current Architecture:** ❌ **LOOSELY COUPLED** (Duplicated Code Pattern)  
**Recommendation:** ✅ **CONSOLIDATION NEEDED** (Establish proper module boundaries)

---

## 1. Import Dependency Analysis

### 1.1 Services Folder (`services/svcs.go`)

**External Imports:**
```go
import (
    "context"
    "fmt"
    "os"
    "time"
    
    "github.com/pithomlabs/rea"              // ✅ Framework
    restate "github.com/restatedev/sdk-go"   // ✅ SDK
    "github.com/restatedev/sdk-go/server"    // ✅ SDK Server
)
```

**Internal Imports:**
```
NONE - No imports from config/models/observability folders
```

**Type Definitions (Lines 14-32):**
```go
// Data structures defined locally for now (shared models will be in models/ package)
// TODO: Move to models/types.go when import path is resolved
type Order struct {
    OrderID         string
    UserID          string
    Items           string
    AmountCents     int
    ShippingAddress string
}

type ShipmentRequest struct {
    OrderID string
    Address string
}

type PaymentReceipt struct {
    TransactionID string
    Success       bool
}
```

**Assessment:** ⚠️ **Types duplicated instead of imported from `models/types.go`**

---

### 1.2 Ingress Folder (`ingress/ingress.go`)

**External Imports:**
```go
import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "log/slog"
    "net/http"
    "strings"
    
    "github.com/go-chi/chi/v5"                      // HTTP router
    "github.com/go-chi/chi/v5/middleware"          // Middleware
    "github.com/pithomlabs/rea"                     // ✅ Framework
    restateingress "github.com/restatedev/sdk-go/ingress" // ✅ SDK Ingress
)
```

**Internal Imports:**
```
NONE - No imports from config/models/observability folders
```

**Type Definitions (Lines 29-35):**
```go
// Data structures matching services
type Order struct {
    OrderID     string
    UserID      string
    Items       string
    AmountCents int
}
```

**Embedded Middleware (Lines 273-405):**
```go
// IdempotencyValidationMiddleware validates Idempotency-Key headers
type IdempotencyValidationMiddleware struct {
    policy  rea.FrameworkPolicy
    metrics *rea.MetricsCollector
    hooks   *rea.ObservabilityHooks
    logger  *slog.Logger
}
```

**Assessment:** ⚠️ **Duplicated Order type and embedded observability code**

---

## 2. Folder Purpose Analysis

### 2.1 Config Folder

**Files:**
- `idempotency.go` - Idempotency configuration loader

**Purpose:** Load environment-based configuration settings

**Current Usage:** ❌ **NOT USED** by services or ingress

**Contents:**
```go
type IdempotencyConfig struct {
    Strict bool
    Logger *slog.Logger
}

func LoadIdempotencyConfig() *IdempotencyConfig {
    // Loads from environment
}
```

**Assessment:** This is **standalone utility code** not integrated anywhere.

---

### 2.2 Models Folder

**Files:**
- `types.go` - Shared data structures

**Purpose:** Centralized data model definitions

**Current Usage:** ❌ **NOT USED** - Types duplicated in services and ingress

**Contents:**
```go
type Order struct {
    OrderID         string `json:"order_id"`
    UserID          string `json:"user_id"`
    Items           string `json:"items"`
    AmountCents     int    `json:"amount_cents"`
    ShippingAddress string `json:"shipping_address"`
}

type ShipmentRequest struct {
    OrderID string `json:"order_id"`
    Address string `json:"address"`
}

type PaymentReceipt struct {
    TransactionID string `json:"transaction_id"`
    Success       bool   `json:"success"`
}
```

**Assessment:** These exact types are **duplicated** in:
- `services/svcs.go` (lines 16-32)
- `ingress/ingress.go` (lines 29-35, partial)

---

### 2.3 Observability Folder

**Files:**
- `instrumentation.go` - Handler wrappers and metrics

**Purpose:** Observability instrumentation utilities

**Current Usage:** ❌ **NOT USED** - Ingress embeds its own middleware

**Contents:**
```go
type IdempotencyInstrumentedHandler[I, O any] struct {
    handler func(restate.Context, I) (O, error)
    metrics *rea.MetricsCollector
    hooks   *rea.ObservabilityHooks
    tracing *rea.TracingContext
}

type IdempotencyMetricsCollector struct {
    baseCollector *rea.MetricsCollector
}
```

**Assessment:** Ingress has its own `IdempotencyValidationMiddleware` embedded instead of using this.

---

## 3. Integration Patterns

### 3.1 Current Pattern: **EMBEDDED DUPLICATION**

**Characteristics:**
- ✅ Each folder is **self-contained** and can run independently
- ❌ **Code duplication** across folders (Order type appears 3x)
- ❌ **No dependency management** - types drift over time
- ❌ Violates **DRY principle** (Don't Repeat Yourself)

**Diagram:**
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  services/  │     │  ingress/   │     │   models/   │
│             │     │             │     │             │
│  Order ❌   │     │  Order ❌   │     │  Order ✅   │
│  ShipReq ❌ │     │             │     │  ShipReq ✅ │
│  PayReceipt │     │             │     │  PayReceipt │
└─────────────┘     └─────────────┘     └─────────────┘
      ↓                   ↓                  ↑
      NO IMPORTS          NO IMPORTS         NOT USED
```

---

### 3.2 Recommended Pattern: **CENTRALIZED MODELS**

**Characteristics:**
- ✅ **Single source of truth** for data models
- ✅ **Consistency** across all consumers
- ✅ **Easier refactoring** - change once, affects all
- ⚠️ Requires **Go module structure** setup

**Diagram:**
```
┌─────────────┐     ┌─────────────┐
│  services/  │     │  ingress/   │
│             │     │             │
│  import ────┼─────┼─────→       │
└─────────────┘     └─────────────┘
                           ↓
                    ┌─────────────┐
                    │   models/   │
                    │             │
                    │  Order ✅   │
                    │  ShipReq ✅ │
                    │  PayReceipt │
                    └─────────────┘
```

---

## 4. Coupling Assessment

### 4.1 Tight vs. Loose Coupling

| Folder Pair | Import Direction | Coupling Level | Evidence |
|-------------|-----------------|----------------|----------|
| services → config | **NONE** | ❌ **No Coupling** | No imports found |
| services → models | **NONE** | ❌ **No Coupling** | Types duplicated |
| services → observability | **NONE** | ❌ **No Coupling** | No imports found |
| ingress → config | **NONE** | ❌ **No Coupling** | No imports found |
| ingress → models | **NONE** | ❌ **No Coupling** | Types duplicated |
| ingress → observability | **NONE** | ❌ **No Coupling** | Middleware embedded |

**Verdict:** ❌ The folders are **NOT INTEGRATED** - they are completely decoupled.

---

### 4.2 Framework Dependency (Only External Integration)

Both services and ingress **DO** import the `rea` framework:

**Services:**
```go
import "github.com/pithomlabs/rea"

// Usage:
cfg := rea.RunConfig{...}
rea.RunWithRetry(ctx, cfg, ...)
rea.ClearAll(ctx)
```

**Ingress:**
```go
import "github.com/pithomlabs/rea"

// Usage:
metrics := rea.NewMetricsCollector()
hooks := rea.DefaultObservabilityHooks(logger)
policy := rea.GetFrameworkPolicy()
rea.HandleGuardrailViolation(...)
rea.ValidateIdempotencyKey(...)
```

**Assessment:** ✅ Both correctly depend on external `rea` framework, but **ignore internal modules**.

---

## 5. Code Duplication Map

### 5.1 Duplicated Order Type

**Location 1: `models/types.go`** *(Original, unused)*
```go
type Order struct {
    OrderID         string `json:"order_id"`
    UserID          string `json:"user_id"` // L2: Authenticated user identity
    Items           string `json:"items"`
    AmountCents     int    `json:"amount_cents"`
    ShippingAddress string `json:"shipping_address"`
}
```

**Location 2: `services/svcs.go`** *(Duplicate, active)*
```go
type Order struct {
    OrderID         string
    UserID          string
    Items           string
    AmountCents     int
    ShippingAddress string
}
```

**Location 3: `ingress/ingress.go`** *(Duplicate, partial)*
```go
type Order struct {
    OrderID     string
    UserID      string
    Items       string
    AmountCents int
}
```

**Issues:**
- ⚠️ **Missing `ShippingAddress`** in ingress version
- ⚠️ **Missing JSON tags** in services version
- ⚠️ **Documentation inconsistency** (L2 comment only in models)

---

### 5.2 Duplicated Supporting Types

| Type | models/ | services/ | ingress/ | Status |
|------|---------|-----------|----------|--------|
| `Order` | ✅ | ✅ | ✅ (partial) | 3 copies |
| `ShipmentRequest` | ✅ | ✅ | ❌ | 2 copies |
| `PaymentReceipt` | ✅ | ✅ | ❌ | 2 copies |

---

## 6. Module Structure Issues

### 6.1 Go Module Configuration

Each subfolder has its **own go.mod**:

```
rea-05/
├── config/go.mod          ⚠️ Separate module
├── ingress/go.mod         ⚠️ Separate module
├── models/                ❌ No go.mod (should be in parent)
├── observability/go.mod   ⚠️ Separate module
└── services/go.mod        ⚠️ Separate module
```

**Problem:** This structure **prevents cross-folder imports** without complex replace directives.

**Solution:** Single root-level `go.mod` with internal package structure:
```
rea-05/
├── go.mod                 ✅ Single module root
├── config/
├── ingress/
├── models/
├── observability/
└── services/
```

---

## 7. Import Path Analysis (If Fixed)

### 7.1 Expected Import Pattern

**If using shared models from `rea-05/models`:**

```go
// In services/svcs.go:
import (
    "rea-05/models"  // ✅ Local package import
)

func (UserSession) Checkout(ctx restate.ObjectContext, orderID string) (bool, error) {
    orderPayload := models.Order{  // ✅ Use shared type
        OrderID:     orderID,
        UserID:      userID,
        Items:       "item-1,item-2",
        AmountCents: 10000,
    }
}
```

**If using shared observability from `rea-05/observability`:**

```go
// In ingress/ingress.go:
import (
    "rea-05/observability"  // ✅ Local package import
)

func main() {
    validator := observability.NewIdempotencyValidationMiddleware(
        logger, metrics, hooks,
    )
    r.Use(validator.Middleware)
}
```

---

## 8. Recommendations

### 8.1 High Priority: Consolidate Module Structure

**Action:** Create single root `go.mod` for rea-05

**Steps:**
1. Remove individual `go.mod` files from subfolders
2. Create `rea-05/go.mod` with module name
3. Update import paths in all files

**Benefit:** Enables cross-folder imports

---

### 8.2 High Priority: Eliminate Type Duplication

**Action:** Import `models.Order` instead of redefining

**Changes Required:**

**In `services/svcs.go`:**
```go
import (
    "rea-05/models"  // Add import
)

// Remove lines 16-32 (duplicate types)

// Update references:
orderPayload := models.Order{...}
```

**In `ingress/ingress.go`:**
```go
import (
    "rea-05/models"  // Add import
)

// Remove lines 29-35 (duplicate Order)

// Update references:
var order models.Order
```

**Benefit:** Single source of truth, consistent JSON tags

---

### 8.3 Medium Priority: Use Shared Observability

**Action:** Import `observability.IdempotencyValidationMiddleware`

**Changes Required:**

**In `ingress/ingress.go`:**
```go
import (
    "rea-05/observability"  // Add import
)

// Remove lines 273-405 (embedded middleware)

// Replace with:
validator := observability.NewIdempotencyValidationMiddleware(...)
```

**Benefit:** DRY principle, reusable across multiple ingress points

---

### 8.4 Low Priority: Use Config Module

**Action:** Call `config.LoadIdempotencyConfig()` in main

**Changes Required:**

**In `ingress/ingress.go` or `services/svcs.go`:**
```go
import "rea-05/config"

func main() {
    cfg := config.LoadIdempotencyConfig()
    if cfg.Strict {
        rea.SetFrameworkPolicy(rea.PolicyStrict)
    }
    // ... rest of main
}
```

**Benefit:** Centralized configuration management

---

## 9. Tightness vs. Modularity Tradeoffs

### 9.1 Current State: **ZERO COUPLING**

**Pros:**
- ✅ Each folder can be copied independently
- ✅ No risk of circular dependencies
- ✅ Easy to understand in isolation

**Cons:**
- ❌ Massive code duplication
- ❌ Types can diverge (already happening with `ShippingAddress`)
- ❌ Difficult to maintain consistency
- ❌ Violates framework best practices

---

### 9.2 Recommended State: **CONTROLLED COUPLING**

**Pros:**
- ✅ DRY principle enforced
- ✅ Consistent types across codebase
- ✅ Easier refactoring and maintenance
- ✅ Better framework integration

**Cons:**
- ⚠️ Requires proper Go module structure
- ⚠️ Folders now depend on each other
- ⚠️ Can't copy folders independently

**Mitigation:** Use **layered architecture**:
```
models/      ← Foundation (no dependencies)
    ↑
config/      ← Configuration (depends on models)
    ↑
observability/ ← Utilities (depends on models)
    ↑
services/    ← Business logic (depends on all)
ingress/     ← API layer (depends on all)
```

---

## 10. Conclusion

### Current Integration Status

**services/ and ingress/ folders:**
- ❌ **DO NOT** reference config/
- ❌ **DO NOT** reference models/
- ❌ **DO NOT** reference observability/
- ✅ **DO** reference external `github.com/pithomlabs/rea` framework

**Verdict:** ❌ **NOT INTEGRATED** - Completely decoupled with duplicated code

---

### Required Changes for Integration

**Minimal (Just Fix Duplication):**
1. Create root `go.mod`
2. Import `models.Order` in services and ingress
3. Remove duplicate type definitions

**Complete (Full Integration):**
1. Create root `go.mod`
2. Import all types from `models/`
3. Import middleware from `observability/`
4. Import config loader from `config/`
5. Establish dependency hierarchy

---

### Integration Complexity

| Change | Complexity | Risk | Benefit |
|--------|-----------|------|---------|
| Root go.mod | **Low** | Low | High (enables imports) |
| Import models | **Low** | Low | High (DRY, consistency) |
| Import observability | **Medium** | Medium | Medium (shared utilities) |
| Import config | **Low** | Low | Low (nice-to-have) |

**Recommendation:** Start with **root go.mod + import models** (high value, low risk).

---

**Next Steps:** Await user approval to proceed with refactoring in rea-05 folder.

---

**Analyst:** Claude Sonnet 4.5  
**Analysis Type:** Dependency and Coupling Assessment  
**Methodology:** Static import analysis + code comparison
