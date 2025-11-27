# Validation: Integration Testing

> **Test your service with real Restate server and verify production behavior**

## 🎯 Learning Objectives

By the end of this validation, you will:
- ✅ Run integration tests with real Restate server
- ✅ Test actual HTTP endpoints
- ✅ Verify state persistence across restarts
- ✅ Test failure scenarios and recovery
- ✅ Measure test coverage and quality

## 📋 Prerequisites

- ✅ Completed [hands-on tutorial](./02-hands-on.md)
- ✅ Docker installed and running
- ✅ All unit tests passing

## 🧪 Validation Steps

### Step 1: Verify Unit Tests Pass

First, ensure all unit tests are working:

```bash
cd ~/restate-tutorials/module07/code
go test -v
```

**Expected Output:**
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
--- PASS: TestHashPassword (0.00s)
=== RUN   TestGenerateToken_Deterministic
--- PASS: TestGenerateToken_Deterministic (0.00s)
PASS
```

✅ **Success Criteria:** All tests pass, no errors

### Step 2: Check Test Coverage

```bash
go test -cover
```

**Expected Output:**
```
PASS
coverage: 87.5% of statements
ok      module07    0.123s
```

Generate detailed coverage report:

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

This opens a browser showing which lines are covered by tests.

✅ **Success Criteria:** 
- Coverage > 80%
- All critical paths tested (register, verify, getProfile)

### Step 3: Start Restate Server

```bash
docker run --name restate_dev --rm \
  -p 8080:8080 -p 9090:9090 -p 9091:9091 \
  --add-host=host.docker.internal:host-gateway \
  docker.io/restatedev/restate:latest
```

**Expected Output:**
```
INFO  [restate_server] Restate Server starting...
INFO  [restate_server] Admin API listening on 0.0.0.0:9070
INFO  [restate_server] Ingress listening on 0.0.0.0:8080
```

✅ **Success Criteria:** Restate server running on ports 8080 (ingress) and 9070 (admin)

### Step 4: Start User Service

In a new terminal:

```bash
cd ~/restate-tutorials/module07/code
go run .
```

**Expected Output:**
```
🧪 Starting User Registration Service on :9090...

📝 Virtual Object: UserService
Handlers:
  Exclusive (modify state):
    - Register
    - VerifyEmail

  Concurrent (read-only):
    - GetProfile

📧 Service: EmailService
Handlers:
    - SendVerificationEmail

✓ Ready to accept requests
```

✅ **Success Criteria:** Service running on port 9090, no errors

### Step 5: Register Service with Restate

In a third terminal:

```bash
curl -X POST http://localhost:9070/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

**Expected Output:**
```json
{
  "id": "dp_...",
  "deployment": {
    "min_protocol_version": 1,
    "max_protocol_version": 2,
    "services": [
      {
        "name": "UserService",
        "ty": "VIRTUAL_OBJECT",
        "handlers": [
          {"name": "Register", "ty": "EXCLUSIVE"},
          {"name": "VerifyEmail", "ty": "EXCLUSIVE"},
          {"name": "GetProfile", "ty": "SHARED"}
        ]
      },
      {
        "name": "EmailService",
        "ty": "SERVICE",
        "handlers": [
          {"name": "SendVerificationEmail"}
        ]
      }
    ]
  }
}
```

✅ **Success Criteria:** Both UserService and EmailService registered successfully

## 🔬 Integration Tests

### Test 1: Register New User

```bash
curl -X POST http://localhost:8080/UserService/user-alice/Register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "username": "alice",
    "password": "secret123"
  }'
```

**Expected Response:**
```json
{
  "userId": "user-alice",
  "status": "created"
}
```

✅ **Success Criteria:** 
- HTTP 200 OK
- Status is "created"
- UserID matches key

### Test 2: Verify Duplicate Registration

```bash
# Try registering same user again
curl -X POST http://localhost:8080/UserService/user-alice/Register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "username": "alice",
    "password": "secret123"
  }'
```

**Expected Response:**
```json
{
  "userId": "user-alice",
  "status": "already_exists"
}
```

✅ **Success Criteria:**
- HTTP 200 OK (not an error)
- Status is "already_exists"

### Test 3: Get User Profile

```bash
curl -X POST http://localhost:8080/UserService/user-alice/GetProfile \
  -H 'Content-Type: application/json' \
  -d '{}'
```

**Expected Response:**
```json
{
  "userId": "user-alice",
  "email": "alice@example.com",
  "username": "alice",
  "passwordHash": "...",
  "createdAt": "2024-11-22T00:23:00Z",
  "verified": false
}
```

✅ **Success Criteria:**
- Profile returned successfully
- Email and username match
- Verified is false (not yet verified)
- Password is hashed (not plaintext)

### Test 4: Email Verification - Invalid Token

```bash
curl -X POST http://localhost:8080/UserService/user-alice/VerifyEmail \
  -H 'Content-Type: application/json' \
  -d '{
    "token": "wrong-token"
  }'
```

**Expected Response:**
```json
{
  "code": 400,
  "message": "invalid verification token"
}
```

✅ **Success Criteria:**
- HTTP 400 Bad Request
- Error message indicates invalid token
- This is a **terminal error** (won't retry)

### Test 5: Email Verification - Valid Token

First, check Restate logs to find the generated token:

```bash
# Look for log line with verification token
docker logs restate_dev 2>&1 | grep "verification_token"
```

Or query the state directly via Restate admin API:

```bash
curl http://localhost:9070/restate/invocation/UserService/user-alice/state/verification_token
```

Then verify with the correct token:

```bash
# Replace TOKEN_HERE with actual token from state
curl -X POST http://localhost:8080/UserService/user-alice/VerifyEmail \
  -H 'Content-Type: application/json' \
  -d '{
    "token": "TOKEN_HERE"
  }'
```

**Expected Response:**
```
(empty, HTTP 200 OK)
```

Verify the profile is updated:

```bash
curl -X POST http://localhost:8080/UserService/user-alice/GetProfile \
  -H 'Content-Type: application/json' \
  -d '{}'
```

**Expected Response:**
```json
{
  "userId": "user-alice",
  "email": "alice@example.com",
  "username": "alice",
  "verified": true
}
```

✅ **Success Criteria:**
- Verification succeeds (HTTP 200)
- Profile shows `verified: true`

## 🔄 State Persistence Test

### Test 6: Service Restart - State Persists

1. **Stop the Go service** (Ctrl+C in terminal running `go run .`)

2. **Start it again:**
   ```bash
   go run .
   ```

3. **Query user profile:**
   ```bash
   curl -X POST http://localhost:8080/UserService/user-alice/GetProfile \
     -H 'Content-Type: application/json' \
     -d '{}'
   ```

**Expected Response:**
```json
{
  "userId": "user-alice",
  "email": "alice@example.com",
  "username": "alice",
  "verified": true
}
```

✅ **Success Criteria:**
- State persists after service restart
- User is still verified
- No data loss

> **💡 Why this works:** Restate stores state durably, not in your service's memory. Your service is stateless!

## 🎯 Testing Multiple Users

Test state isolation by creating multiple users:

```bash
# Register user-bob
curl -X POST http://localhost:8080/UserService/user-bob/Register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "bob@example.com",
    "username": "bob",
    "password": "password456"
  }'

# Register user-charlie
curl -X POST http://localhost:8080/UserService/user-charlie/Register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "charlie@example.com",
    "username": "charlie",
    "password": "pass789"
  }'

# Get all three profiles
curl -X POST http://localhost:8080/UserService/user-alice/GetProfile \
  -H 'Content-Type: application/json' -d '{}'
curl -X POST http://localhost:8080/UserService/user-bob/GetProfile \
  -H 'Content-Type: application/json' -d '{}'
curl -X POST http://localhost:8080/UserService/user-charlie/GetProfile \
  -H 'Content-Type: application/json' -d '{}'
```

✅ **Success Criteria:**
- Each user has separate state
- No interference between users
- alice is verified, bob and charlie are not

## 📊 Performance Testing

### Test Concurrent Reads

Concurrent `GetProfile` calls should work in parallel (they're SHARED handlers):

```bash
# Run multiple concurrent requests
for i in {1..10}; do
  curl -X POST http://localhost:8080/UserService/user-alice/GetProfile \
    -H 'Content-Type: application/json' \
    -d '{}' &
done
wait
```

✅ **Success Criteria:**
- All requests succeed
- Fast response times (concurrent execution)

### Test Sequential Writes

Exclusive handlers (`Register`, `VerifyEmail`) execute sequentially per key:

```bash
# These will execute one at a time for user-alice
curl -X POST http://localhost:8080/UserService/user-alice/Register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","username":"alice","password":"pass1"}' &
curl -X POST http://localhost:8080/UserService/user-alice/Register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","username":"alice","password":"pass2"}' &
wait
```

✅ **Success Criteria:**
- Both requests complete
- No race conditions
- First succeeds with "already_exists"

## 🐛 Error Handling Tests

### Test Missing Required Fields

```bash
curl -X POST http://localhost:8080/UserService/user-test/Register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "test@example.com"
  }'
```

✅ **Success Criteria:** Service handles gracefully (may accept or reject based on validation)

### Test Empty User Key

```bash
curl -X POST http://localhost:8080/UserService//Register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "test@example.com",
    "username": "test",
    "password": "password"
  }'
```

✅ **Success Criteria:** Restate returns error (empty key not allowed)

## 📈 Observability

### View Invocation History

```bash
# Get invocations for user-alice
curl http://localhost:9070/restate/invocation/UserService/user-alice
```

This shows all invocations (Register, VerifyEmail, GetProfile) for this key.

### View State

```bash
# View all state keys
curl http://localhost:9070/restate/invocation/UserService/user-alice/state

# View specific state
curl http://localhost:9070/restate/invocation/UserService/user-alice/state/profile
curl http://localhost:9070/restate/invocation/UserService/user-alice/state/verification_token
```

✅ **Success Criteria:**
- Can inspect state via admin API
- State matches expectations

## ✅ Validation Checklist

Your implementation is complete when:

- [x] **Unit Tests**
  - All tests pass (`go test -v`)
  - Coverage > 80%
  - Tests are deterministic

- [x] **Integration Tests**
  - Service registers with Restate
  - User registration works
  - Duplicate detection works
  - Email verification works (valid/invalid tokens)
  - Profile retrieval works

- [x] **State Management**
  - State persists across service restarts
  - State is isolated per user key
  - No race conditions on exclusive handlers

- [x] **Error Handling**
  - Invalid tokens return terminal errors
  - Graceful handling of edge cases

- [x] **Observability**
  - Can view invocation history
  - Can inspect state via admin API

## 🎓 What You Learned

### Testing Patterns
- ✅ **Unit tests** with mock contexts for fast feedback
- ✅ **Integration tests** with real server for confidence
- ✅ **State verification** tests for correctness
- ✅ **Table-driven tests** for comprehensive coverage

### Restate Testing Features
- ✅ **Mock contexts** eliminate Restate server dependency
- ✅ **Deterministic UUIDs** with seeded randomness
- ✅ **State inspection** via admin API
- ✅ **Invocation history** for debugging

### Best Practices
- ✅ Test both success and failure paths
- ✅ Verify state changes explicitly
- ✅ Use table-driven tests for similar scenarios
- ✅ Integration tests verify real behavior
- ✅ Mock external dependencies (email service)

## 🚀 Next Steps

Congratulations! You've learned comprehensive testing for Restate services!

👉 **Continue to [Exercises](./04-exercises.md)**

Practice your testing skills with challenges!

---

## 🔍 Troubleshooting

### Tests fail with "connection refused"
- **Cause:** Restate server not running (integration tests only)
- **Fix:** Start Restate with Docker (unit tests don't need it)

### Coverage lower than expected
- **Cause:** Not all code paths tested
- **Fix:** Add tests for error cases, edge cases, and helpers

### State not persisting
- **Cause:** Using different Restate server or cleared state
- **Fix:** Keep same Restate container running

### Duplicate registration test fails
- **Cause:** State from previous test run
- **Fix:** Use unique keys for each test or restart Restate

---

**Questions?** Review [concepts](./01-concepts.md) or [hands-on guide](./02-hands-on.md).
