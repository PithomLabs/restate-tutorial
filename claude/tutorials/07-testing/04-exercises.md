# Exercises: Testing Practice

> **Hands-on exercises to master testing Restate services**

## 🎯 Learning Goals

These exercises will help you:
- Write comprehensive unit tests
- Test error scenarios effectively
- Mock external dependencies
- Write integration tests
- Achieve high test coverage

## 📚 Exercise Structure

Each exercise includes:
- **Difficulty:** ⭐ Beginner, ⭐⭐ Intermediate, ⭐⭐⭐ Advanced
- **Objective:** What to build
- **Hints:** Guidance if stuck
- **Solution:** Reference implementation in `/solutions`

---

## Exercise 1: Test Password Update ⭐

### Objective

Add a `UpdatePassword` handler to `UserService` and write comprehensive tests for it.

### Requirements

1. **Handler:** `UpdatePassword(ctx, req) error`
   - Input: `{oldPassword, newPassword}`
   - Verify old password matches stored hash
   - Update to new password hash
   - Return error if old password is wrong

2. **Tests:**
   - Test successful password update
   - Test wrong old password (should fail)
   - Test user doesn't exist (should fail)

### Hints

<details>
<summary>Click to reveal hints</summary>

1. Add to `types.go`:
   ```go
   type UpdatePasswordRequest struct {
       OldPassword string `json:"oldPassword"`
       NewPassword string `json:"newPassword"`
   }
   ```

2. Handler skeleton:
   ```go
   func (UserService) UpdatePassword(
       ctx restate.ObjectContext,
       req UpdatePasswordRequest,
   ) error {
       // Get profile
       // Hash old password and compare
       // If match, hash new password and save
       // Return terminal error if wrong password
   }
   ```

3. Test setup:
   ```go
   func TestUserService_UpdatePassword_Success(t *testing.T) {
       ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
       service := UserService{}
       
       // Pre-populate with known password
       profile := UserProfile{
           UserID: "user-123",
           PasswordHash: hashPassword("oldpass"),
       }
       restate.Set(ctx, stateKeyProfile, profile)
       
       // Test update
       req := UpdatePasswordRequest{
           OldPassword: "oldpass",
           NewPassword: "newpass",
       }
       err := service.UpdatePassword(ctx, req)
       require.NoError(t, err)
       
       // Verify password updated
       updatedProfile, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
       assert.Equal(t, hashPassword("newpass"), updatedProfile.PasswordHash)
   }
   ```
</details>

### Success Criteria

- ✅ Handler implemented correctly
- ✅ 3+ tests written
- ✅ All tests pass
- ✅ Wrong password returns terminal error

---

## Exercise 2: Test Account Deletion ⭐⭐

### Objective

Add a `DeleteAccount` handler that removes all user state and write tests for it.

### Requirements

1. **Handler:** `DeleteAccount(ctx, req) error`
   - Input: `{password}` - requires password confirmation
   - Verify password matches
   - Clear all state (`restate.ClearAll`)
   - Return error if password wrong

2. **Tests:**
   - Test successful deletion (all state cleared)
   - Test wrong password (state unchanged)
   - Test deletion with no existing user

### Hints

<details>
<summary>Click to reveal hints</summary>

1. Handler:
   ```go
   func (UserService) DeleteAccount(
       ctx restate.ObjectContext,
       password string,
   ) error {
       profile, err := restate.Get[UserProfile](ctx, stateKeyProfile)
       if err != nil {
           return err
       }
       
       if hashPassword(password) != profile.PasswordHash {
           return restate.TerminalError(
               fmt.Errorf("invalid password"), 403)
       }
       
       restate.ClearAll(ctx)
       return nil
   }
   ```

2. Verify deletion:
   ```go
   // After deletion
   keys, _ := restate.Keys(ctx)
   assert.Empty(t, keys) // No state left
   ```
</details>

### Success Criteria

- ✅ `DeleteAccount` handler works
- ✅ Tests verify all state is cleared
- ✅ Password verification required
- ✅ Terminal error on wrong password

---

## Exercise 3: Test Email Change with Verification ⭐⭐

### Objective

Implement and test email change requiring verification of both old and new email addresses.

### Requirements

1. **Handlers:**
   - `RequestEmailChange(ctx, newEmail, password)` - initiate change
   - `ConfirmEmailChange(ctx, token)` - complete change

2. **Workflow:**
   - Store pending email in state
   - Generate verification token
   - Verify token matches before updating
   - Send emails to both old and new addresses

3. **Tests:**
   - Test full email change flow
   - Test token expiration (hint: store timestamp)
   - Test wrong token
   - Test unauthorized change (wrong password)

### Hints

<details>
<summary>Click to reveal hints</summary>

1. Add state keys:
   ```go
   const (
       stateKeyPendingEmail = "pending_email"
       stateKeyEmailChangeToken = "email_change_token"
       stateKeyTokenTimestamp = "token_timestamp"
   )
   ```

2. Request handler:
   ```go
   func (UserService) RequestEmailChange(
       ctx restate.ObjectContext,
       req struct {
           NewEmail string
           Password string
       },
   ) error {
       // Verify password
       // Store new email as pending
       // Generate token
       // Send verification emails
   }
   ```

3. Test workflow:
   ```go
   func TestUserService_EmailChange_FullFlow(t *testing.T) {
       ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
       service := UserService{}
       
       // Setup existing user
       // Request email change
       // Get token from state
       // Confirm with token
       // Verify email updated
   }
   ```
</details>

### Success Criteria

- ✅ Two-step email change process
- ✅ Both handlers tested
- ✅ Token validation works
- ✅ Password required for change

---

## Exercise 4: Mock External Email Service ⭐⭐⭐

### Objective

Write a proper integration test that mocks the email service to verify it's called correctly.

### Requirements

1. **Create integration test** that:
   - Starts a real Restate server
   - Registers UserService
   - Makes HTTP requests
   - Verifies email service was called with correct parameters

2. **Mock Email Service:**
   - Track all calls (email, token pairs)
   - Return success/failure based on test scenario
   - Verify no duplicate emails sent

### Hints

<details>
<summary>Click to reveal hints</summary>

1. Integration test setup:
   ```go
   // integration_test.go
   func TestIntegration_Registration_SendsEmail(t *testing.T) {
       // This requires:
       // - Docker with Restate running
       // - HTTP client to call endpoints
       // - Way to capture email service calls
       
       if testing.Short() {
           t.Skip("Skipping integration test")
       }
       
       // Setup
       client := &http.Client{}
       baseURL := "http://localhost:8080"
       
       // Register user
       resp, err := client.Post(
           baseURL+"/UserService/user-test/Register",
           "application/json",
           strings.NewReader(`{
               "email": "test@example.com",
               "username": "test",
               "password": "pass123"
           }`),
       )
       
       require.NoError(t, err)
       assert.Equal(t, http.StatusOK, resp.StatusCode)
       
       // Verify email was sent
       // (This requires checking EmailService was called)
   }
   ```

2. Run integration tests:
   ```bash
   # Unit tests only (fast)
   go test -short
   
   # All tests including integration (slow)
   go test
   ```
</details>

### Success Criteria

- ✅ Integration test requires Docker
- ✅ Makes real HTTP requests
- ✅ Verifies email service called
- ✅ Can be skipped with `-short` flag

---

## Exercise 5: Table-Driven Error Scenarios ⭐⭐

### Objective

Write comprehensive table-driven tests covering all error cases in the UserService.

### Requirements

Create table-driven tests for:
1. **Invalid inputs** - empty email, short password, etc.
2. **State errors** - missing user, corrupted state
3. **Business logic errors** - unverified user trying protected action

### Example Structure

```go
func TestUserService_Register_ValidationErrors(t *testing.T) {
    tests := []struct {
        name        string
        req         RegisterRequest
        wantErr     bool
        errContains string
    }{
        {
            name: "empty email",
            req: RegisterRequest{
                Email:    "",
                Username: "alice",
                Password: "pass123",
            },
            wantErr:     true,
            errContains: "email required",
        },
        {
            name: "invalid email format",
            req: RegisterRequest{
                Email:    "not-an-email",
                Username: "alice",
                Password: "pass123",
            },
            wantErr:     true,
            errContains: "invalid email",
        },
        // Add more cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
            service := UserService{}
            
            _, err := service.Register(ctx, tt.req)
            
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errContains)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Hints

<details>
<summary>Click to reveal hints</summary>

1. Add validation to handlers:
   ```go
   // In Register handler
   if req.Email == "" {
       return RegisterResult{}, restate.TerminalError(
           fmt.Errorf("email required"), 400)
   }
   
   if !strings.Contains(req.Email, "@") {
       return RegisterResult{}, restate.TerminalError(
           fmt.Errorf("invalid email format"), 400)
   }
   
   if len(req.Password) < 8 {
       return RegisterResult{}, restate.TerminalError(
           fmt.Errorf("password must be at least 8 characters"), 400)
   }
   ```

2. Test error types:
   ```go
   // Verify it's a terminal error
   assert.True(t, restate.IsTerminalError(err))
   
   // Verify error code
   assert.Equal(t, 400, restate.ErrorCode(err))
   ```
</details>

### Success Criteria

- ✅ Table-driven test structure
- ✅ 5+ error scenarios covered
- ✅ All tests pass
- ✅ Clear test names describing each case

---

## Exercise 6: Test Concurrent Operations ⭐⭐⭐

### Objective

Write tests that verify exclusive and shared handlers behave correctly under concurrent access.

### Requirements

1. **Test exclusive handlers** execute sequentially
2. **Test shared handlers** can run concurrently
3. **Verify no race conditions** on state access

### Challenge

This is an integration test that requires:
- Multiple goroutines making requests
- Timing verification
- State consistency checks

### Hints

<details>
<summary>Click to reveal hints</summary>

1. Test skeleton:
   ```go
   func TestConcurrency_ExclusiveHandlers_Sequential(t *testing.T) {
       if testing.Short() {
           t.Skip("Skipping concurrency test")
       }
       
       // This test verifies exclusive handlers run sequentially
       // even when called concurrently
       
       var wg sync.WaitGroup
       results := make(chan RegisterResult, 10)
       
       // Launch 10 concurrent registrations for same key
       for i := 0; i < 10; i++ {
           wg.Add(1)
           go func(id int) {
               defer wg.Done()
               
               result, _ := callRegisterViaHTTP(
                   "user-test",
                   fmt.Sprintf("user%d@example.com", id),
               )
               results <- result
           }(i)
       }
       
       wg.Wait()
       close(results)
       
       // Verify: exactly one "created", rest "already_exists"
       createdCount := 0
       for result := range results {
           if result.Status == "created" {
               createdCount++
           }
       }
       
       assert.Equal(t, 1, createdCount, 
           "Only one registration should succeed")
   }
   ```

2. Concurrent reads:
   ```go
   func TestConcurrency_SharedHandlers_Concurrent(t *testing.T) {
       // Shared handlers should execute in parallel
       // Measure that 10 concurrent GetProfile calls
       // complete faster than 10 sequential calls would
       
       start := time.Now()
       
       var wg sync.WaitGroup
       for i := 0; i < 10; i++ {
           wg.Add(1)
           go func() {
               defer wg.Done()
               callGetProfileViaHTTP("user-alice")
           }()
       }
       wg.Wait()
       
       duration := time.Since(start)
       
       // If truly concurrent, should finish quickly
       // If sequential, would take much longer
       assert.Less(t, duration, 1*time.Second)
   }
   ```
</details>

### Success Criteria

- ✅ Concurrent exclusive calls don't cause races
- ✅ Only one registration succeeds for same key
- ✅ Shared handlers run concurrently
- ✅ State remains consistent

---

## 🎓 Bonus Challenges

### Bonus 1: Snapshot Testing ⭐⭐⭐

Implement snapshot testing for UserProfile serialization:
- Generate profile
- Serialize to JSON
- Compare against golden file
- Detect unexpected changes

### Bonus 2: Fuzzing ⭐⭐⭐

Use Go's fuzzing to test password hashing:
```go
func FuzzHashPassword(f *testing.F) {
    f.Add("password123")
    f.Fuzz(func(t *testing.T, password string) {
        hash := hashPassword(password)
        assert.Len(t, hash, 64)
    })
}
```

Run with: `go test -fuzz=FuzzHashPassword`

### Bonus 3: Benchmark Tests ⭐⭐

Write benchmarks for critical operations:
```go
func BenchmarkHashPassword(b *testing.B) {
    for i := 0; i < b.N; i++ {
        hashPassword("password123")
    }
}
```

Run with: `go test -bench=.`

---

## 📝 Exercise Solutions

Solutions are provided in the `solutions/` directory:

- `solutions/exercise1_password_update.go`
- `solutions/exercise2_account_deletion.go`
- `solutions/exercise3_email_change.go`
- `solutions/exercise4_integration_test.go`
- `solutions/exercise5_table_driven_test.go`
- `solutions/exercise6_concurrency_test.go`

**Try solving without looking first!** The solutions are for reference and learning.

---

## ✅ Completion Checklist

You've mastered testing when you can:

- [x] Write unit tests with mock contexts
- [x] Test state changes explicitly
- [x] Use table-driven tests effectively
- [x] Mock external dependencies
- [x] Write integration tests
- [x] Test error scenarios comprehensively
- [x] Verify concurrent behavior
- [x] Achieve >80% test coverage

---

## 🎯 Next Steps

Congratulations on completing Module 07! You now know how to:
- ✅ Write comprehensive unit tests
- ✅ Test stateful virtual objects
- ✅ Mock external services
- ✅ Run integration tests
- ✅ Achieve high test coverage

### Continue Learning

👉 **Next Module:** [Module 08 - External Integration](../../08-external-integration/README.md)

Learn how to integrate with external systems and APIs!

### Additional Practice

- Test the services from previous modules
- Add tests to your own Restate projects
- Contribute tests to open source projects
- Write custom test utilities for your team

---

## 💡 Testing Best Practices Summary

1. **Write tests first** (TDD) or immediately after implementation
2. **Test behavior, not implementation** - focus on inputs/outputs
3. **Keep tests fast** - use mocks for unit tests
4. **Make tests deterministic** - no random failures
5. **Test edge cases** - empty inputs, large inputs, boundary conditions
6. **Use descriptive names** - `TestFeature_Scenario_ExpectedBehavior`
7. **One assertion per test** (when practical)
8. **Integration tests supplement unit tests** - they don't replace them

---

**Questions?** Review [concepts](./01-concepts.md), [hands-on](./02-hands-on.md), or [validation](./03-validation.md).
