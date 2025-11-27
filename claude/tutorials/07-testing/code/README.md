# Module 07: Testing - Code

> **Complete working code for the User Registration Service with comprehensive tests**

## 📁 Files

- **`types.go`** - Data structures (RegisterRequest, UserProfile, etc.)
- **`user_service.go`** - UserService virtual object with state management
- **`user_service_test.go`** - Comprehensive unit tests
- **`email_service.go`** - Email verification service
- **`main.go`** - Server setup and registration
- **`go.mod`** - Go module dependencies

## 🚀 Quick Start

### 1. Install Dependencies

```bash
go mod download
```

### 2. Run Tests

```bash
# Run all tests
go test -v

# Run with coverage
go test -cover

# Generate coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 3. Start the Service

```bash
# Terminal 1: Start Restate server (Docker)
docker run --name restate_dev --rm -p 8080:8080 -p 9090:9090 -p 9091:9091 \
  --add-host=host.docker.internal:host-gateway \
  docker.io/restatedev/restate:latest

# Terminal 2: Start user service
go run .

# Terminal 3: Register service with Restate
curl -X POST http://localhost:9090/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

## 🧪 Test Scenarios

The test suite covers:

1. **Successful user registration** - New user creation
2. **Duplicate registration** - Existing user detection
3. **Email verification** - Token validation
4. **Invalid token** - Error handling
5. **Profile retrieval** - State reading
6. **Password hashing** - Table-driven tests
7. **Deterministic UUIDs** - Seeded randomness

## 📊 Expected Test Output

```
=== RUN   TestUserService_Register_Success
--- PASS: TestUserService_Register_Success (0.00s)
=== RUN   TestUserService_Register_AlreadyExists
--- PASS: TestUserService_Register_AlreadyExists (0.00s)
=== RUN   TestUserService_VerifyEmail_Success
--- PASS: TestUserService_VerifyEmail_Success (0.00s)
=== RUN   TestUserService_VerifyEmail_InvalidToken
--- PASS: TestUserService_VerifyEmail_InvalidToken (0.00s)
=== RUN   TestUserService_GetProfile
--- PASS: TestUserService_GetProfile (0.00s)
=== RUN   TestHashPassword
=== RUN   TestHashPassword/short_password
=== RUN   TestHashPassword/long_password
=== RUN   TestHashPassword/special_chars
--- PASS: TestHashPassword (0.00s)
=== RUN   TestGenerateToken_Deterministic
--- PASS: TestGenerateToken_Deterministic (0.00s)
PASS
coverage: 87.5% of statements
ok      module07    0.123s
```

## 🔑 Key Testing Patterns

### Mock Context Setup
```go
ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
```

### Pre-populating State
```go
profile := UserProfile{UserID: "user-123", Verified: true}
restate.Set(ctx, stateKeyProfile, profile)
```

### Verifying State Changes
```go
profile, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
assert.True(t, profile.Verified)
```

### Deterministic Testing
```go
ctx := restate.NewMockObjectContext(restate.WithRandomSeed(42))
// UUIDs will be deterministic with same seed
```

## 🎯 Learning Objectives

- ✅ Writing unit tests with mock contexts
- ✅ Testing stateful virtual objects
- ✅ Mocking external dependencies
- ✅ Table-driven test patterns
- ✅ Test coverage and quality metrics
- ✅ Deterministic test execution

## 📚 Related Files

- [Concepts](../01-concepts.md) - Testing theory
- [Hands-On Guide](../02-hands-on.md) - Step-by-step tutorial
- [Validation](../03-validation.md) - Integration testing
- [Exercises](../04-exercises.md) - Practice problems

---

**Module 07** | [Previous: Module 06 - Sagas](../../06-sagas/README.md) | [Next: Module 08 - External Integration](../../08-external-integration/README.md)
