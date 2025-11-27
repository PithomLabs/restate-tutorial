# Concepts: Security in Restate Applications

> **Understand security patterns for distributed systems**

## 🎯 What You'll Learn

- Authentication and authorization patterns
- Securing service-to-service communication
- Handling sensitive data
- Common security vulnerabilities
- Security best practices

---

## 🔐 Authentication vs Authorization

### Authentication (AuthN)
**"Who are you?"** - Verifying identity

```go
// Verify user identity
func authenticate(token string) (*User, error) {
    claims, err := validateJWT(token)
    if err != nil {
        return nil, fmt.Errorf("invalid token")
    }
    
    user := getUserByID(claims.UserID)
    return user, nil
}
```

### Authorization (AuthZ)
**"What can you do?"** - Verifying permissions

```go
// Check if user can perform action
func authorize(user *User, action string, resource string) error {
    if !user.HasPermission(action, resource) {
        return fmt.Errorf("unauthorized: %s on %s", action, resource)
    }
    return nil
}
```

---

## 🎫 Authentication Patterns

### Pattern 1: API Keys

Simple authentication for service-to-service calls:

```go
const validAPIKey = "sk_live_abc123xyz789"

func (SecureService) ProcessRequest(
    ctx restate.ServiceContext,
    req SecureRequest,
) (Response, error) {
    // Validate API key
    if req.APIKey != validAPIKey {
        return Response{}, restate.TerminalError(
            fmt.Errorf("invalid API key"), 401)
    }
    
    // Process request
    return processSecurely(ctx, req)
}
```

**Best Practices:**
- Store keys in environment variables
- Rotate keys regularly
- Use different keys per environment
- Never log API keys

### Pattern 2: JWT (JSON Web Tokens)

Stateless authentication with signed tokens:

```go
import "github.com/golang-jwt/jwt/v5"

type Claims struct {
    UserID string   `json:"userId"`
    Role   string   `json:"role"`
    jwt.RegisteredClaims
}

func validateJWT(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(
        tokenString,
        &Claims{},
        func(token *jwt.Token) (interface{}, error) {
            // Validate signing method
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method")
            }
            return []byte(os.Getenv("JWT_SECRET")), nil
        },
    )
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, fmt.Errorf("invalid token")
}

func (UserService) GetProfile(
    ctx restate.ObjectContext,
    req ProfileRequest,
) (Profile, error) {
    // Validate JWT
    claims, err := validateJWT(req.Token)
    if err != nil {
        return Profile{}, restate.TerminalError(err, 401)
    }
    
    // Check authorization: users can only access their own profile
    userID := restate.Key(ctx)
    if claims.UserID != userID {
        return Profile{}, restate.TerminalError(
            fmt.Errorf("forbidden"), 403)
    }
    
    // Return profile
    profile, _ := restate.Get[Profile](ctx, "profile")
    return profile, nil
}
```

---

## 🛡️ Authorization Patterns

### Role-Based Access Control (RBAC)

```go
type Role string

const (
    RoleUser  Role = "user"
    RoleAdmin Role = "admin"
    RoleMod   Role = "moderator"
)

type Permission struct {
    Resource string
    Action   string
}

var rolePermissions = map[Role][]Permission{
    RoleUser: {
        {Resource: "profile", Action: "read"},
        {Resource: "profile", Action: "update"},
    },
    RoleAdmin: {
        {Resource: "*", Action: "*"}, // All permissions
    },
    RoleMod: {
        {Resource: "posts", Action: "delete"},
        {Resource: "users", Action: "suspend"},
    },
}

func hasPermission(role Role, resource, action string) bool {
    permissions := rolePermissions[role]
    for _, perm := range permissions {
        if (perm.Resource == "*" || perm.Resource == resource) &&
           (perm.Action == "*" || perm.Action == action) {
            return true
        }
    }
    return false
}

func (PostService) DeletePost(
    ctx restate.ObjectContext,
    req DeleteRequest,
) error {
    // Check permission
    if !hasPermission(req.UserRole, "posts", "delete") {
        return restate.TerminalError(
            fmt.Errorf("forbidden: insufficient permissions"), 403)
    }
    
    // Delete post
    restate.Clear(ctx, "post")
    return nil
}
```

### Attribute-Based Access Control (ABAC)

```go
func canAccessOrder(user User, order Order) bool {
    // User can access their own orders
    if user.ID == order.CustomerID {
        return true
    }
    
    // Admins can access all orders
    if user.Role == "admin" {
        return true
    }
    
    // Support can access orders in their region
    if user.Role == "support" && user.Region == order.Region {
        return true
    }
    
    return false
}
```

---

## 🔒 Securing Sensitive Data

### Encryption at Rest

```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
)

func encrypt(plaintext string, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(ciphertext string, key []byte) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := gcm.NonceSize()
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    return string(plaintext), err
}

// Usage
func (UserService) StoreSSN(
    ctx restate.ObjectContext,
    ssn string,
) error {
    key := []byte(os.Getenv("ENCRYPTION_KEY")) // 32 bytes for AES-256
    
    // Encrypt before storing
    encrypted, err := encrypt(ssn, key)
    if err != nil {
        return err
    }
    
    restate.Set(ctx, "ssn_encrypted", encrypted)
    return nil
}

func (UserService) GetSSN(
    ctx restate.ObjectContext,
) (string, error) {
    key := []byte(os.Getenv("ENCRYPTION_KEY"))
    
    encrypted, err := restate.Get[string](ctx, "ssn_encrypted")
    if err != nil {
        return "", err
    }
    
    // Decrypt when reading
    return decrypt(encrypted, key)
}
```

### Never Log Sensitive Data

```go
// ❌ BAD - Logs sensitive data
ctx.Log().Info("User login",
    "email", email,
    "password", password,        // NEVER!
    "creditCard", creditCard,    // NEVER!
    "ssn", ssn)                  // NEVER!

// ✅ GOOD - No sensitive data
ctx.Log().Info("User login",
    "userId", userID,
    "email", email,
    "loginTime", time.Now())
```

---

## ⚠️ Common Security Vulnerabilities

### 1. Injection Attacks

```go
// ❌ BAD - SQL injection vulnerability
func getUser(username string) {
    query := fmt.Sprintf("SELECT * FROM users WHERE username='%s'", username)
    // If username = "admin' OR '1'='1"
    // Query becomes: SELECT * FROM users WHERE username='admin' OR '1'='1'
}

// ✅ GOOD - Parameterized query
func getUser(username string) {
    query := "SELECT * FROM users WHERE username = ?"
    db.Query(query, username)
}
```

### 2. Insecure Deserialization

```go
// ❌ BAD - No validation
var data UntrustedInput
json.Unmarshal(input, &data)
// Process without validation

// ✅ GOOD - Validate after deserialization
var data UntrustedInput
if err := json.Unmarshal(input, &data); err != nil {
    return restate.TerminalError(err, 400)
}

// Validate
if !isValid(data) {
    return restate.TerminalError(
        fmt.Errorf("invalid input"), 400)
}
```

### 3. Missing Access Control

```go
// ❌ BAD - No authorization check
func (OrderService) CancelOrder(
    ctx restate.ObjectContext,
    req CancelRequest,
) error {
    // Anyone can cancel any order!
    restate.Set(ctx, "status", "cancelled")
}

// ✅ GOOD - Check authorization
func (OrderService) CancelOrder(
    ctx restate.ObjectContext,
    req CancelRequest,
) error {
    order, _ := restate.Get[Order](ctx, "order")
    
    // Only order owner or admin can cancel
    if req.UserID != order.CustomerID && req.UserRole != "admin" {
        return restate.TerminalError(
            fmt.Errorf("unauthorized"), 403)
    }
    
    restate.Set(ctx, "status", "cancelled")
    return nil
}
```

### 4. Exposure of Sensitive Information

```go
// ❌ BAD - Leaks implementation details
func (Service) Handler() error {
    err := db.Query("SELECT * FROM secret_table")
    return err  // Returns: "table secret_table does not exist"
}

// ✅ GOOD - Generic error message
func (Service) Handler() error {
    err := db.Query("SELECT * FROM secret_table")
    if err != nil {
        ctx.Log().Error("Database error", "error", err)
        return restate.TerminalError(
            fmt.Errorf("internal server error"), 500)
    }
}
```

---

## ✅ Security Best Practices

### 1. Input Validation

```go
func validateEmail(email string) error {
    if len(email) == 0 {
        return fmt.Errorf("email required")
    }
    if len(email) > 255 {
        return fmt.Errorf("email too long")
    }
    if !strings.Contains(email, "@") {
        return fmt.Errorf("invalid email format")
    }
    return nil
}

func validatePassword(password string) error {
    if len(password) < 12 {
        return fmt.Errorf("password must be at least 12 characters")
    }
    // Add more complexity requirements
    return nil
}
```

### 2. Least Privilege Principle

```go
// Give minimal permissions needed
func (DataService) ReadData(req Request) (Data, error) {
    // User can only read their own data
    if req.UserID != req.DataOwnerID {
        return Data{}, restate.TerminalError(
            fmt.Errorf("forbidden"), 403)
    }
    return getData(req.DataOwnerID)
}
```

### 3. Secure Defaults

```go
// ✅ GOOD - Secure by default
type SecurityConfig struct {
    RequireAuth       bool     `default:"true"`
    RequireHTTPS      bool     `default:"true"`
    MaxLoginAttempts  int      `default:"3"`
    SessionTimeout    duration `default:"15m"`
}
```

### 4. Defense in Depth

Multiple layers of security:

```go
func (SecureService) ProcessPayment(
    ctx restate.ObjectContext,
    req PaymentRequest,
) error {
    // Layer 1: Authentication
    if !isAuthenticated(req.Token) {
        return restate.TerminalError(fmt.Errorf("unauthenticated"), 401)
    }
    
    // Layer 2: Authorization
    if !canProcessPayment(req.UserID, req.Amount) {
        return restate.TerminalError(fmt.Errorf("unauthorized"), 403)
    }
    
    // Layer 3: Input validation
    if err := validatePaymentRequest(req); err != nil {
        return restate.TerminalError(err, 400)
    }
    
    // Layer 4: Rate limiting
    if isRateLimited(req.UserID) {
        return restate.TerminalError(fmt.Errorf("rate limited"), 429)
    }
    
    // Process payment
    return processPayment(ctx, req)
}
```

---

## 🔑 Secret Management

### Use Environment Variables

```go
// ✅ GOOD - From environment
apiKey := os.Getenv("API_KEY")
dbPassword := os.Getenv("DB_PASSWORD")
jwtSecret := os.Getenv("JWT_SECRET")

// ❌ BAD - Hardcoded
const apiKey = "sk_live_abc123..."  // NEVER!
```

### Secrets Manager Integration

```go
import "github.com/aws/aws-sdk-go/service/secretsmanager"

func getSecret(secretName string) (string, error) {
    svc := secretsmanager.New(session.New())
    result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretName),
    })
    if err != nil {
        return "", err
    }
    return *result.SecretString, nil
}
```

---

## 🚀 Next Steps

You now understand security patterns!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

Build a secure user management system!

---

**Questions?** Review this document or check the [module README](./README.md).
