# Security Middleware Implementation Guide

> **HTTP middleware for enforcing SecurityConfig validation on Restate ingress handlers**

## Overview

The `SecurityMiddleware` provides **production-ready HTTP middleware** that enforces the `SecurityConfig` validation rules for incoming Restate requests. This implements DOS guidance for "validate request identity" and "require HTTPS" for public endpoints.

### What It Does

✅ **Request Signature Verification**: Validates Ed25519 signatures from Restate servers  
✅ **HTTPS Enforcement**: Rejects HTTP requests when HTTPS is required  
✅ **Origin Validation**: Restricts requests to allowed Restate server origins  
✅ **Configurable Modes**: Strict, permissive, or disabled validation  
✅ **Structured Logging**: Comprehensive logging of security events  

## Implementation

### Core Middleware

```go
func SecurityMiddleware(validator *SecurityValidator) func(http.Handler) http.Handler
```

**Returns**: A middleware wrapper that can be applied to any `http.Handler`

**Behavior by Mode**:

| Mode | Invalid Request | Valid Request |
|------|----------------|---------------|
| `SecurityModeStrict` | Reject with HTTP 401/403 | Pass through |
| `SecurityModePermissive` | Log warning, pass through | Pass through |
| `SecurityModeDisabled` | Pass through (no validation) | Pass through |

### Convenience Wrappers

```go
// Secure a single handler function
func SecureHandlerFunc(validator *SecurityValidator, handler http.HandlerFunc) http.HandlerFunc

// Secure an entire ServeMux
func SecureServer(validator *SecurityValidator, mux *http.ServeMux) http.Handler
```

## Quick Start

### 1. Basic Setup

```go
package main

import (
    "log"
    "log/slog"
    "net/http"
    
    . "github.com/restatedev/examples/rea2/claude"
)

func main() {
    // 1. Configure security
    config := DefaultSecurityConfig()
    config.ValidationMode = SecurityModeStrict
    
    // 2. Load signing keys (from environment or config)
    keyBase64 := os.Getenv("RESTATE_PUBLIC_KEY")
    publicKey, err := ParseSigningKey(keyBase64)
    if err != nil {
        log.Fatalf("Invalid signing key: %v", err)
    }
    config.SigningKeys = []ed25519.PublicKey{publicKey}
    
    // 3. Create validator
    validator := NewSecurityValidator(config, slog.Default())
    
    // 4. Wrap your handler
    restateHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Your Restate handler logic
        w.WriteHeader(http.StatusOK)
    })
    
    securedHandler := SecurityMiddleware(validator)(restateHandler)
    
    // 5. Start server
    http.Handle("/restate", securedHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### 2. Production Configuration

```go
func setupSecureServer() {
    config := SecurityConfig{
        EnableRequestValidation: true,
        RequireHTTPS:            true,
        RestrictPublicAccess:    true,
        ValidationMode:          SecurityModeStrict,
        AllowedOrigins: []string{
            "restate-prod-1.example.com",
            "restate-prod-2.example.com",
        },
    }
    
    // Load multiple signing keys for key rotation
    keyStrings := []string{
        os.Getenv("RESTATE_PRIMARY_KEY"),
        os.Getenv("RESTATE_SECONDARY_KEY"),
    }
    keys, err := ParseSigningKeys(keyStrings)
    if err != nil {
        log.Fatalf("Failed to parse keys: %v", err)
    }
    config.SigningKeys = keys
    
    validator := NewSecurityValidator(config, slog.Default())
    
    // Secure all Restate endpoints
    mux := http.NewServeMux()
    mux.HandleFunc("/restate", restateHandler)
    mux.HandleFunc("/restate/discovery", discoveryHandler)
    
    securedMux := SecureServer(validator, mux)
    
    slog.Info("Starting secure server",
        "validation_mode", config.ValidationMode,
        "https_required", config.RequireHTTPS,
        "num_signing_keys", len(config.SigningKeys))
    
    log.Fatal(http.ListenAndServe(":8080", securedMux))
}
```

## Validation Modes

### Strict Mode (Production)

**Use when**: Running in production or CI environments

```go
config.ValidationMode = SecurityModeStrict
```

**Behavior**:
- ❌ **Rejects** invalid requests with HTTP 401/403
- ✅ Passes valid requests to handler
- 📝 Logs all validation failures at ERROR level
- 📝 Logs successful validations at DEBUG level

**Status Codes**:
- `401 Unauthorized`: Missing or invalid signature
- `403 Forbidden`: HTTPS required or origin not allowed

**Example Response**:
```
HTTP/1.1 401 Unauthorized
Content-Type: text/plain

signature verification failed with all configured keys
```

### Permissive Mode (Development)

**Use when**: Developing locally or in staging

```go
config.ValidationMode = SecurityModePermissive
```

**Behavior**:
- ✅ **Allows** all requests through
- ⚠️ Logs validation failures at WARN level
- 📝 Logs successful validations at DEBUG level
- 🔍 Useful for debugging signature issues

**Example Log**:
```json
{
  "level": "WARN",
  "msg": "security: request validation failed (permissive mode)",
  "error": "signature verification failed with all configured keys",
  "origin": "restate-dev.example.com",
  "path": "/restate/invoke",
  "method": "POST"
}
```

### Disabled Mode (Testing Only)

**Use when**: Testing without Restate server integration

```go
config.ValidationMode = SecurityModeDisabled
```

**Behavior**:
- ✅ **Skips** all validation
- ⚠️ Logs warning that validation is disabled
- ❌ **NOT recommended** for development or production

## Validation Rules

### 1. Signature Verification

**What it checks**: Ed25519 signature on request

**Enabled when**: `EnableRequestValidation = true` and `SigningKeys` configured

**Signature Format**:
```
X-Restate-Signature: <base64-encoded-signature>
```

**Signed Message Format**:
```
METHOD PATH?QUERY
BODY
```

**Example**:
```go
config.EnableRequestValidation = true
config.SigningKeys = []ed25519.PublicKey{publicKey}
```

### 2. HTTPS Enforcement

**What it checks**: Request protocol is HTTPS

**Enabled when**: `RequireHTTPS = true`

**Checks**:
1. `req.TLS != nil` (direct TLS connection)
2. `X-Forwarded-Proto: https` (behind reverse proxy)

**Example**:
```go
config.RequireHTTPS = true
```

### 3. Origin Validation

**What it checks**: `X-Restate-Server` header matches allowed origins

**Enabled when**: `AllowedOrigins` is non-empty

**Example**:
```go
config.AllowedOrigins = []string{
    "restate-prod-1.example.com",
    "restate-prod-2.example.com",
}
```

## Key Management

### Loading Keys from Environment

```go
// Single key
keyBase64 := os.Getenv("RESTATE_PUBLIC_KEY")
publicKey, err := ParseSigningKey(keyBase64)
if err != nil {
    log.Fatalf("Invalid key: %v", err)
}
config.SigningKeys = []ed25519.PublicKey{publicKey}
```

### Multiple Keys for Rotation

```go
// Support key rotation by accepting multiple keys
primaryKey, _ := ParseSigningKey(os.Getenv("RESTATE_PRIMARY_KEY"))
secondaryKey, _ := ParseSigningKey(os.Getenv("RESTATE_SECONDARY_KEY"))

config.SigningKeys = []ed25519.PublicKey{
    primaryKey,    // Current key
    secondaryKey,  // Previous key (during rotation)
}
```

**Key Rotation Process**:
1. Add new key to `SigningKeys` list
2. Update Restate server to use new key
3. Wait for all in-flight requests to complete
4. Remove old key from list

### Key Format

**Ed25519 Public Key** (32 bytes):
```
Base64-encoded: "MC4CAQAwBQYDK2VwBCIEINTjo7oa/..." (44 characters)
```

**Generate keys** (for testing):
```go
publicKey, privateKey, _ := ed25519.GenerateKey(nil)
publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)
```

## Middleware Patterns

### Pattern 1: Single Endpoint

```go
http.Handle("/restate", SecurityMiddleware(validator)(restateHandler))
```

### Pattern 2: Multiple Endpoints with Same Security

```go
mux := http.NewServeMux()
mux.HandleFunc("/restate", handler1)
mux.HandleFunc("/restate/awakeables", handler2)

securedMux := SecureServer(validator, mux)
http.ListenAndServe(":8080", securedMux)
```

### Pattern 3: Different Security Levels

```go
// Strict for production
strictValidator := NewSecurityValidator(strictConfig, logger)

// Permissive for development
permissiveValidator := NewSecurityValidator(permissiveConfig, logger)

http.Handle("/restate", SecurityMiddleware(strictValidator)(prodHandler))
http.Handle("/restate/dev", SecurityMiddleware(permissiveValidator)(devHandler))
```

### Pattern 4: Middleware Chain

```go
handler := loggingMiddleware(
    metricsMiddleware(
        SecurityMiddleware(validator)(
            restateHandler,
        ),
    ),
)
```

**Order matters**: Security middleware should be placed **after** logging/metrics but **before** the actual handler.

## Testing

### Unit Tests

```go
func TestSecurityMiddleware_ValidSignature(t *testing.T) {
    publicKey, privateKey, _ := ed25519.GenerateKey(nil)
    
    config := DefaultSecurityConfig()
    config.ValidationMode = SecurityModeStrict
    config.SigningKeys = []ed25519.PublicKey{publicKey}
    config.RequireHTTPS = false
    
    validator := NewSecurityValidator(config, slog.Default())
    
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    
    securedHandler := SecurityMiddleware(validator)(handler)
    
    // Create signed request
    req := httptest.NewRequest("POST", "/restate", strings.NewReader("body"))
    signRequest(req, privateKey)
    
    rec := httptest.NewRecorder()
    securedHandler.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", rec.Code)
    }
}
```

See [`security_middleware_test.go`](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/security_middleware_test.go) for comprehensive test suite covering:
- ✅ Valid signatures
- ❌ Invalid signatures
- ❌ Missing signatures
- 🔄 Multiple signing keys
- 🔒 HTTPS enforcement
- 🌐 Origin validation
- 📊 All validation modes

### Integration Tests

```bash
# Start server with strict validation
RESTATE_PUBLIC_KEY="MC4CAQ..." go run main.go

# Test with valid signature (should succeed)
curl -X POST http://localhost:8080/restate \
  -H "X-Restate-Signature: validSignature" \
  -d '{"test": "data"}'

# Test without signature (should fail with 401)
curl -X POST http://localhost:8080/restate \
  -d '{"test": "data"}'
```

## Logging

### Log Levels

| Event | Mode | Level | Example |
|-------|------|-------|---------|
| Validation disabled | All | WARN | `validation disabled - accepting request without verification` |
| Invalid request | Strict | ERROR | `rejecting invalid request (strict mode)` |
| Invalid request | Permissive | WARN | `request validation failed (permissive mode)` |
| Valid request | All | DEBUG | `request validated successfully` |

### Log Fields

All security logs include structured fields:

```json
{
  "level": "ERROR",
  "msg": "security: rejecting invalid request (strict mode)",
  "error": "signature verification failed with all configured keys",
  "origin": "restate-unknown.example.com",
  "path": "/restate/invoke",
  "method": "POST",
  "remote_addr": "192.168.1.100:54321"
}
```

### Monitoring Alerts

Recommended alerts for production:

```yaml
# High rate of validation failures
- alert: HighSecurityFailureRate
  expr: rate(security_validation_failures_total[5m]) > 0.1
  
# Unexpected origins
- alert: UnknownOrigin
  expr: security_validation_failures{reason="origin_not_allowed"} > 0

# Missing signatures in production
- alert: MissingSignatures
  expr: security_validation_failures{reason="missing_signature"} > 10
```

## Troubleshooting

### Error: "signature verification failed with all configured keys"

**Cause**: Signature doesn't match any configured public key

**Solutions**:
1. Verify public key matches Restate server's private key
2. Check key format (should be base64-encoded Ed25519 public key)
3. Ensure signature construction matches Restate's format
4. Try permissive mode to debug: `ValidationMode: SecurityModePermissive`

### Error: "HTTPS required but request received over HTTP"

**Cause**: `RequireHTTPS = true` but request came over HTTP

**Solutions**:
1. Configure TLS on your server
2. Set `X-Forwarded-Proto: https` if behind reverse proxy
3. For local dev, set `RequireHTTPS = false`

### Error: "origin not allowed"

**Cause**: `X-Restate-Server` header not in `AllowedOrigins` list

**Solutions**:
1. Add origin to `AllowedOrigins`:
   ```go
   config.AllowedOrigins = []string{"restate-server.example.com"}
   ```
2. Remove origin restrictions (not recommended for production)
3. Check `X-Restate-Server` header value in logs

### Request passes in permissive but fails in strict

**Debugging steps**:
1. Check logs for specific validation failure
2. Verify signature construction:
   ```
   METHOD PATH?QUERY
   BODY
   ```
3. Test with minimal config (disable HTTPS, origin checks)
4. Compare working vs failing requests

## Security Best Practices

### ✅ DO

1. **Use strict mode in production**:
   ```go
   config.ValidationMode = SecurityModeStrict
   ```

2. **Enable all validations**:
   ```go
   config.EnableRequestValidation = true
   config.RequireHTTPS = true
   config.AllowedOrigins = []string{...}
   ```

3. **Rotate keys regularly**:
   ```go
   config.SigningKeys = []ed25519.PublicKey{newKey, oldKey}
   ```

4. **Store keys securely** (environment variables, secrets manager, not in code)

5. **Monitor validation failures**:
   ```go
   // Log all failures for security monitoring
   ```

### ❌ DON'T

1. **Don't disable validation in production**:
   ```go
   // ❌ BAD
   config.ValidationMode = SecurityModeDisabled
   ```

2. **Don't commit keys to version control**:
   ```go
   // ❌ BAD
   config.SigningKeys = []ed25519.PublicKey{hardcodedKey}
   ```

3. **Don't skip HTTPS in production**:
   ```go
   // ❌ BAD for production
   config.RequireHTTPS = false
   ```

4. **Don't ignore validation errors**:
   ```go
   // ❌ BAD - at least log them
   if !result.Valid {
       // ignoring error
   }
   ```

## Performance Considerations

### Middleware Overhead

| Operation | Time | Impact |
|-----------|------|--------|
| Signature verification (Ed25519) | ~50-100μs | Low |
| Header checks | ~1μs | Negligible |
| Origin validation | ~1μs | Negligible |
| **Total overhead** | **~50-100μs** | **Minimal** |

### Optimization Tips

1. **Place security middleware early in chain** (fail fast on invalid requests)
2. **Use multiple keys sparingly** (linear search, but Ed25519 is fast)
3. **Cache validator instances** (don't create per-request)
4. **Disable validation for health checks**:
   ```go
   http.Handle("/health", healthHandler) // No middleware
   http.Handle("/restate", securedHandler) // With middleware
   ```

## Related Documentation

- [Security Configuration](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/FRAMEWORK_VS_SDK_SUMMARY.MD#security) - SecurityConfig reference
- [Framework Policy](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/FRAMEWORK_POLICY.MD) - Global validation control
- [Examples](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/security_middleware_example.go) - Complete usage examples
- [Tests](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/security_middleware_test.go) - Test suite

## Summary

| Aspect | Details |
|--------|---------|
| **Purpose** | Enforce SecurityConfig validation on HTTP ingress |
| **Validations** | Signature, HTTPS, Origin |
| **Modes** | Strict (reject), Permissive (warn), Disabled |
| **Performance** | ~50-100μs overhead per request |
| **Testing** | Comprehensive test suite included |
| **Production Ready** | ✅ Yes, use SecurityModeStrict |

The SecurityMiddleware provides a production-ready, configurable solution for securing Restate ingress endpoints with minimal performance overhead and comprehensive logging! 🔒
