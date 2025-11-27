# Validation: Testing Security

> **Verify your secure user management system**

## 🎯 Validation Goals

- ✅ Verify authentication works
- ✅ Confirm authorization enforcement
- ✅ Test input validation
- ✅ Validate password security
- ✅ Check unauthorized access prevention

## 🧪 Test Scenarios

### Scenario 1: User Registration → Login → Get Profile

**Test:** Complete authentication flow

```bash
# 1. Register
curl -X POST http://localhost:8080/UserService/alice/Register \
  -d '{"email":"alice@example.com","password":"SecurePass123"}'

# 2. Login
RESPONSE=$(curl -X POST http://localhost:8080/UserService/alice/Login \
  -d '{"email":"alice@example.com","password":"SecurePass123"}')

TOKEN=$(echo $RESPONSE | jq -r '.token')

# 3. Get Profile
curl -X POST http://localhost:8080/UserService/alice/GetProfile \
  -d "{\"token\":\"$TOKEN\"}"
```

**Expected:** All steps succeed, profile returned

**Validation:**
- [ ] Registration creates user
- [ ] Login returns valid JWT
- [ ] Profile accessible with token
- [ ] Password not in response

### Scenario 2: Invalid Login Credentials

**Test:** Wrong password

```bash
curl -X POST http://localhost:8080/UserService/alice/Login \
  -d '{"email":"alice@example.com","password":"WrongPassword"}'
```

**Expected:** 401 Unauthorized  
**Validation:**
- [ ] Returns 401 status
- [ ] Error message doesn't leak info

### Scenario 3: Unauthorized Access

**Test:** User tries to access another user's profile

```bash
# Alice's token
curl -X POST http://localhost:8080/UserService/bob/GetProfile \
  -d "{\"token\":\"$ALICE_TOKEN\"}"
```

**Expected:** 403 Forbidden  
**Validation:**
- [ ] Access denied
- [ ] Audit log created
- [ ] No data leaked

### Scenario 4: Input Validation

**Test:** Invalid email and password

```bash
# Invalid email
curl -X POST http://localhost:8080/UserService/user1/Register \
  -d '{"email":"not-an-email","password":"password123"}'

# Weak password
curl -X POST http://localhost:8080/UserService/user2/Register \
  -d '{"email":"user@example.com","password":"weak"}'
```

**Expected:** 400 Bad Request for both  
**Validation:**
- [ ] Email validation works
- [ ] Password requirements enforced
- [ ] Clear error messages
- [ ] Terminal errors returned

### Scenario 5: Role-Based Access Control

**Test:** Non-admin tries to promote user

```bash
# Regular user token
curl -X POST http://localhost:8080/UserService/bob/PromoteToAdmin \
  -d "{\"token\":\"$USER_TOKEN\"}"
```

**Expected:** 403 Forbidden  
**Validation:**
- [ ] Non-admin blocked
- [ ] Only admins can promote
- [ ] Audit log shows attempt

## ✅ Validation Checklist

### Authentication
- [ ] User registration works
- [ ] Passwords hashed (bcrypt)
- [ ] JWT tokens generated
- [ ] Token validation works
- [ ] Invalid credentials rejected

### Authorization
- [ ] Users can access own data only
- [ ] RBAC enforced
- [ ] Admin operations protected
- [ ] Unauthorized access blocked
- [ ] Audit logs created

### Security
- [ ] Passwords never logged
- [ ] Passwords never in responses
- [ ] Input validation active
- [ ] Secure defaults used
- [ ] Error messages safe

## 🎓 Success Criteria

Pass when:
- ✅ All test scenarios pass
- ✅ Authentication works correctly
- ✅ Authorization enforced
- ✅ No security vulnerabilities
- ✅ Audit trail complete

## 🚀 Next Steps

👉 **Continue to [Exercises](./04-exercises.md)**
