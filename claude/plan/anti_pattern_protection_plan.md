# Anti-Pattern Protection Implementation Plan

## Problem Statement

The framework currently provides **runtime checks** but lacks **compile-time prevention** of critical anti-patterns in Restate applications. This creates opportunities for developers to introduce subtle bugs that only manifest during execution.

## Current Protection Gaps

| Anti-Pattern | Current Protection | Gap |
|--------------|-------------------|-----|
| Calling futures in goroutines | ❌ No prevention | Still possible, no linting |
| Using `ctx` inside `restate.Run` | ⚠️ Runtime panic | Can't detect at compile time |
| Sleeping in exclusive object handler | ❌ No prevention | Developer can block object |
| Deadlock-prone object calls | ⚠️ Logging & patterns | No static analysis |
| Non-deterministic iteration | ❌ No prevention | Maps/sets still risky |
| Changing code without versioning | ❌ Deployment concern | Outside framework scope |

## Proposed Solution: Multi-Layer Protection

### Layer 1: Static Analysis (Compile-Time)

Create custom Go analyzers using `golang.org/x/tools/go/analysis` to detect anti-patterns.

**Tools to Build:**
1. **`restatelint`** - Custom analyzer package
   - Detects `restate.Context` usage inside `restate.Run` closures
   - Flags goroutine creation with future handling
   - Warns on `time.Sleep` in object handler functions
   - Detects map iteration over non-deterministic sources
   - Checks for self-referential object calls

2. **`golangci-lint` Integration**
   - Plugin configuration for CI/CD pipelines
   - Pre-commit hooks setup

### Layer 2: Enhanced Type Safety (Compile-Time)

Use Go's type system to make dangerous patterns impossible.

**New Types:**
1. **`SafeRunContext`** - Wrapper that doesn't expose parent context
2. **`DeterministicMap[K, V]`** - Ordered map for iteration
3. **`ObjectCallGuard`** - Prevents self-calls at type level
4. **`NonBlockingHandler[I, O]`** - Handler type that forbids blocking

### Layer 3: Runtime Guards (Enhanced)

Improve runtime detection with better error messages and recovery.

**Enhancements:**
1. **Context Guard** - Detect nested context usage
2. **Goroutine Detector** - Track futures across goroutines
3. **Blocking Call Detector** - Warn on long-running operations
4. **Determinism Validator** - Check for non-deterministic patterns

### Layer 4: Documentation & Tooling

**Deliverables:**
1. `ANTI_PATTERNS.md` - Comprehensive guide with examples
2. `pre-commit` hook configuration
3. CI/CD integration examples
4. VS Code extension recommendations

## Implementation Plan

### Phase 1: Static Analyzer (Priority: HIGH)
**Files to Create:**
- `tools/restatelint/analyzer.go` - Main analyzer
- `tools/restatelint/checks/context_in_run.go` - Context check
- `tools/restatelint/checks/futures_in_goroutine.go` - Goroutine check
- `tools/restatelint/checks/sleep_in_handler.go` - Blocking check
- `tools/restatelint/checks/map_iteration.go` - Determinism check
- `tools/restatelint/cmd/restatelint/main.go` - CLI tool

**Checks to Implement:**
1. ✅ Context usage in `restate.Run()` closures
2. ✅ Future operations in goroutines
3. ✅ `time.Sleep()` in object/workflow handlers
4. ✅ Map range iteration patterns
5. ✅ Self-referential object calls

### Phase 2: Type-Safe Wrappers (Priority: MEDIUM)
**Files to Modify:**
- `framework.go` - Add new types

**New Types:**
```go
// SafeRunContext prevents parent context leakage
type SafeRunContext struct {
    rc restate.RunContext
}

// DeterministicMap ensures ordered iteration
type DeterministicMap[K comparable, V any] struct {
    data  map[K]V
    order []K
}

// NonBlockingHandler prevents blocking operations
type NonBlockingHandler[I, O any] interface {
    Handle(ctx ObjectContext, input I) (O, error)
}
```

### Phase 3: Enhanced Runtime Guards (Priority: MEDIUM)
**Files to Modify:**
- `framework.go` - Add detection utilities

**New Guards:**
```go
// DetectContextMisuse checks for nested context usage
func DetectContextMisuse(parentCtx, innerCtx Context) error

// GuardAgainstGoroutines prevents future operations in goroutines  
func GuardAgainstGoroutines(ctx Context, operation string) error

// WarnOnBlockingCall detects long-running operations
func WarnOnBlockingCall(ctx Context, duration time.Duration)
```

### Phase 4: Documentation (Priority: HIGH)
**Files to Create:**
- `ANTI_PATTERNS.md` - Comprehensive guide
- `tools/restatelint/README.md` - Linter documentation
- `.pre-commit-config.yaml` - Pre-commit hook setup
- `.github/workflows/restate-lint.yml` - CI/CD example

## Success Criteria

### Developer Experience
- [ ] Developers get immediate feedback in IDE
- [ ] Pre-commit hooks catch issues before push
- [ ] CI/CD fails on anti-pattern detection
- [ ] Clear error messages explain how to fix

### Coverage
- [ ] 90%+ of common anti-patterns detectable
- [ ] Zero false positives in normal usage
- [ ] Integration with existing Go tooling

### Performance
- [ ] Analysis completes in <5s for typical projects
- [ ] No runtime overhead from type wrappers
- [ ] Minimal CI/CD pipeline impact

## Timeline

- **Week 1**: Static analyzer core + context/goroutine checks
- **Week 2**: Complete all static checks + testing
- **Week 3**: Type-safe wrappers + runtime guards
- **Week 4**: Documentation + tooling integration

## Open Questions

1. **Versioning Detection**: Should we add git hook to enforce versioning strategy?
2. **Performance**: Should runtime guards be disabled in production?
3. **IDE Integration**: Build dedicated VS Code extension or rely on gopls?
4. **Opt-out**: Should developers be able to suppress warnings?

## Next Steps

1. ✅ Review and approve this plan
2. Create `tools/restatelint` analyzer package
3. Implement first check (context in Run)
4. Test on framework codebase
5. Iterate and expand coverage
