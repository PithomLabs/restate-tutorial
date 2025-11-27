# Module 11: Security

> **Secure your Restate applications**

## 🎯 Learning Objectives

By completing this module, you will:
- ✅ Implement authentication and authorization
- ✅ Secure service-to-service communication
- ✅ Handle sensitive data safely
- ✅ Implement security best practices
- ✅ Audit and compliance logging
- ✅ Protect against common vulnerabilities

## 📚 Module Content

### 1. Concepts (~20 min)
- Authentication vs authorization
- JWT and API key patterns
- Service-to-service security
- Data encryption
- Security best practices

### 2. Hands-On (~30 min)
- Implement JWT authentication
- Add authorization checks
- Secure inter-service calls
- Encrypt sensitive data
- Add audit logging

### 3. Best Practices (~15 min)
- Principle of least privilege
- Secret management
- Security headers
- Rate limiting
- Security testing

## 🎯 Key Concepts

### Authentication Patterns

**1. API Keys**
```go
func AuthMiddleware(apiKey string) bool {
    validKeys := []string{"key1", "key2"}
    return contains(validKeys, apiKey)
}
```

**2. JWT Tokens**
```go
func ValidateJWT(token string) (*Claims, error) {
    parsed, err := jwt.Parse(token, keyFunc)
    if err != nil {
        return nil, err
    }
    return parsed.Claims.(*Claims), nil
}
```

### Authorization

**Role-Based Access Control (RBAC)**
```go
func (OrderService) CancelOrder(
    ctx restate.ObjectContext,
    req CancelRequest,
) error {
    // Check if user has permission
    if !hasRole(req.UserID, "admin") {
        return restate.TerminalError(
            fmt.Errorf("unauthorized"), 403)
    }
    
    // Proceed with cancellation
}
```

## 🔐 Security Best Practices

### 1. Validate All Inputs

```go
// ✅ GOOD - Validate and sanitize
func (UserService) CreateUser(
    ctx restate.ObjectContext,
    req CreateUserRequest,
) error {
    if !isValidEmail(req.Email) {
        return restate.TerminalError(
            fmt.Errorf("invalid email"), 400)
    }
    
    if len(req.Password) < 12 {
        return restate.TerminalError(
            fmt.Errorf("password too short"), 400)
    }
    
    // Continue...
}
```

### 2. Never Log Sensitive Data

```go
// ❌ BAD - Logs password!
ctx.Log().Info("User login",
    "email", email,
    "password", password) // NEVER!

// ✅ GOOD - No sensitive data
ctx.Log().Info("User login",
    "email", email)
```

### 3. Use Secrets Management

```go
// ✅ GOOD - From environment/secrets manager
apiKey := os.Getenv("STRIPE_API_KEY")

// ❌ BAD - Hardcoded secret
apiKey := "sk_live_abc123..." // NEVER!
```

### 4. Encrypt Sensitive State

```go
// ✅ GOOD - Encrypt before storing
encryptedSSN := encrypt(user.SSN, key)
restate.Set(ctx, "ssn", encryptedSSN)

// When reading
encryptedSSN, _ := restate.Get[string](ctx, "ssn")
ssn := decrypt(encryptedSSN, key)
```

## ⚠️ Common Vulnerabilities

### 1. Injection Attacks

```go
// ❌ BAD - SQL injection risk
query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userID)

// ✅ GOOD - Parameterized query
query := "SELECT * FROM users WHERE id = ?"
db.Query(query, userID)
```

### 2. Insecure Deserialization

```go
// ❌ BAD - Unsafe deserialization
var data UntrustedData
json.Unmarshal(input, &data)

// ✅ GOOD - Validate after deserialization
var data UntrustedData
if err := json.Unmarshal(input, &data); err != nil {
    return err
}
if !validate(data) {
    return fmt.Errorf("invalid data")
}
```

### 3. Missing Access Control

```go
// ❌ BAD - No authorization check
func DeleteUser(userID string) {
    // Anyone can delete anyone!
}

// ✅ GOOD - Check permissions
func DeleteUser(actorID, targetID string) error {
    if actorID != targetID && !isAdmin(actorID) {
        return fmt.Errorf("unauthorized")
    }
    // Proceed...
}
```

## 🛡️ Security Checklist

- [ ] All inputs validated
- [ ] Authentication implemented
- [ ] Authorization enforced
- [ ] Secrets in environment variables
- [ ] Sensitive data encrypted
- [ ] Audit logging enabled
- [ ] Rate limiting configured
- [ ] Security headers set
- [ ] Dependencies updated
- [ ] Security tests written

## 🎓 Learning Path

**Current Module:** Security  
**Previous:** [Observability](../10-observability/README.md)  
**Next:** [Module 12 - Production](../12-production/README.md)

---

**Secure your applications!** 🔒
