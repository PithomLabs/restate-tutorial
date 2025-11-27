# Exercises: Security Practice

> **Practice building secure Restate applications**

## Exercise 1: API Key Authentication ⭐
Add API key authentication to a service.

**Requirements:**
- Store API keys in environment variables
- Validate on every request
- Return 401 for invalid keys
- Rate limit by API key

## Exercise 2: Multi-Factor Authentication ⭐⭐
Implement 2FA for user login.

**Requirements:**
- Generate TOTP codes
- Verify codes on login
- Store backup codes
- Enforce for sensitive operations

## Exercise 3: Data Encryption ⭐⭐
Encrypt sensitive user data at rest.

**Requirements:**
- Encrypt SSN, credit cards
- Use AES-256
- Manage encryption keys securely
- Decrypt only when needed

## Exercise 4: Audit Logging ⭐⭐
Comprehensive audit trail for security events.

**Requirements:**
- Log all auth attempts
- Log data access
- Log permission changes
- Queryable audit log

## Exercise 5: OAuth 2.0 Integration ⭐⭐⭐
Implement OAuth 2.0 provider integration.

**Requirements:**
- Authorization code flow
- Token refresh
- Scope-based permissions
- Secure token storage

## Exercise 6: Security Testing ⭐⭐
Write security tests for vulnerabilities.

**Requirements:**
- SQL injection tests
- XSS prevention tests
- CSRF protection tests
- Rate limiting tests

## 📚 Resources
- [Concepts](./01-concepts.md)
- [Hands-On](./02-hands-on.md)
- [Validation](./03-validation.md)

**Secure coding!** 🔒
