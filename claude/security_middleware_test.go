package framework_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/restatedev/examples/rea2/claude"
)

// Test 1: SecurityMiddleware with valid signature (strict mode)
func TestSecurityMiddleware_ValidSignature_Strict(t *testing.T) {
	// Generate key pair
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)

	// Configure validator
	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeStrict
	config.SigningKeys = []ed25519.PublicKey{publicKey}
	config.RequireHTTPS = false // Disable HTTPS for test

	validator := NewSecurityValidator(config, slog.Default())

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware
	securedHandler := SecurityMiddleware(validator)(handler)

	// Create request with valid signature
	req := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	signRequest(req, privateKey)

	// Execute request
	rec := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec, req)

	// Assert success
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "success" {
		t.Errorf("Expected body 'success', got '%s'", body)
	}
}

// Test 2: SecurityMiddleware with invalid signature (strict mode)
func TestSecurityMiddleware_InvalidSignature_Strict(t *testing.T) {
	// Generate key pair
	publicKey, _, _ := ed25519.GenerateKey(nil)
	_, wrongPrivateKey, _ := ed25519.GenerateKey(nil) // Different key

	// Configure validator
	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeStrict
	config.SigningKeys = []ed25519.PublicKey{publicKey}
	config.RequireHTTPS = false

	validator := NewSecurityValidator(config, slog.Default())

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	securedHandler := SecurityMiddleware(validator)(handler)

	// Create request with WRONG signature
	req := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	signRequest(req, wrongPrivateKey)

	// Execute request
	rec := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec, req)

	// Assert rejection
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

// Test 3: SecurityMiddleware with missing signature (strict mode)
func TestSecurityMiddleware_MissingSignature_Strict(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(nil)

	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeStrict
	config.SigningKeys = []ed25519.PublicKey{publicKey}
	config.RequireHTTPS = false

	validator := NewSecurityValidator(config, slog.Default())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	securedHandler := SecurityMiddleware(validator)(handler)

	// Create request WITHOUT signature
	req := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))

	rec := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec, req)

	// Assert rejection
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

// Test 4: SecurityMiddleware with invalid signature (permissive mode)
func TestSecurityMiddleware_InvalidSignature_Permissive(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(nil)
	_, wrongPrivateKey, _ := ed25519.GenerateKey(nil)

	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModePermissive // Permissive mode
	config.SigningKeys = []ed25519.PublicKey{publicKey}
	config.RequireHTTPS = false

	validator := NewSecurityValidator(config, slog.Default())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	securedHandler := SecurityMiddleware(validator)(handler)

	req := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	signRequest(req, wrongPrivateKey) // Wrong signature

	rec := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec, req)

	// In permissive mode, should still succeed (but log warning)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 in permissive mode, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "success" {
		t.Errorf("Expected body 'success', got '%s'", body)
	}
}

// Test 5: SecurityMiddleware with disabled validation
func TestSecurityMiddleware_Disabled(t *testing.T) {
	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeDisabled // Disabled

	validator := NewSecurityValidator(config, slog.Default())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	securedHandler := SecurityMiddleware(validator)(handler)

	// Request without any signature
	req := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))

	rec := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec, req)

	// Should succeed without validation
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 with disabled validation, got %d", rec.Code)
	}
}

// Test 6: HTTPS requirement enforcement
func TestSecurityMiddleware_HTTPSRequired(t *testing.T) {
	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeStrict
	config.RequireHTTPS = true
	config.EnableRequestValidation = false // Disable signature check for this test

	validator := NewSecurityValidator(config, slog.Default())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	securedHandler := SecurityMiddleware(validator)(handler)

	// Request over HTTP (not HTTPS)
	req := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	req.TLS = nil // No TLS

	rec := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec, req)

	// Should reject with Forbidden
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}
}

// Test 7: Origin validation
func TestSecurityMiddleware_OriginValidation(t *testing.T) {
	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeStrict
	config.AllowedOrigins = []string{"restate-prod.example.com"}
	config.EnableRequestValidation = false
	config.RequireHTTPS = false

	validator := NewSecurityValidator(config, slog.Default())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	securedHandler := SecurityMiddleware(validator)(handler)

	// Test with disallowed origin
	req := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	req.Header.Set("X-Restate-Server", "restate-dev.example.com")

	rec := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec, req)

	// Should reject
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for disallowed origin, got %d", rec.Code)
	}

	// Test with allowed origin
	req2 := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	req2.Header.Set("X-Restate-Server", "restate-prod.example.com")

	rec2 := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec2, req2)

	// Should succeed
	if rec2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for allowed origin, got %d", rec2.Code)
	}
}

// Test 8: Multiple signing keys
func TestSecurityMiddleware_MultipleKeys(t *testing.T) {
	pub1, priv1, _ := ed25519.GenerateKey(nil)
	pub2, priv2, _ := ed25519.GenerateKey(nil)

	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeStrict
	config.SigningKeys = []ed25519.PublicKey{pub1, pub2} // Multiple keys
	config.RequireHTTPS = false

	validator := NewSecurityValidator(config, slog.Default())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	securedHandler := SecurityMiddleware(validator)(handler)

	// Test with first key
	req1 := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	signRequest(req1, priv1)

	rec1 := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("Expected success with first key, got %d", rec1.Code)
	}

	// Test with second key
	req2 := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	signRequest(req2, priv2)

	rec2 := httptest.NewRecorder()
	securedHandler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("Expected success with second key, got %d", rec2.Code)
	}
}

// Test 9: SecureHandlerFunc convenience wrapper
func TestSecureHandlerFunc(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)

	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeStrict
	config.SigningKeys = []ed25519.PublicKey{publicKey}
	config.RequireHTTPS = false

	validator := NewSecurityValidator(config, slog.Default())

	// Use convenience wrapper
	securedFunc := SecureHandlerFunc(validator, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	req := httptest.NewRequest("POST", "/restate/invoke", strings.NewReader("test body"))
	signRequest(req, privateKey)

	rec := httptest.NewRecorder()
	securedFunc(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// Test 10: SecureServer with ServeMux
func TestSecureServer(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)

	config := DefaultSecurityConfig()
	config.ValidationMode = SecurityModeStrict
	config.SigningKeys = []ed25519.PublicKey{publicKey}
	config.RequireHTTPS = false

	validator := NewSecurityValidator(config, slog.Default())

	// Create mux with multiple endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoint1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("endpoint1"))
	})
	mux.HandleFunc("/endpoint2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("endpoint2"))
	})

	// Secure entire mux
	securedMux := SecureServer(validator, mux)

	// Test endpoint1 with valid signature
	req1 := httptest.NewRequest("POST", "/endpoint1", strings.NewReader("body"))
	signRequest(req1, privateKey)

	rec1 := httptest.NewRecorder()
	securedMux.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("Expected status 200 for endpoint1, got %d", rec1.Code)
	}

	// Test endpoint2 with valid signature
	req2 := httptest.NewRequest("POST", "/endpoint2", strings.NewReader("body"))
	signRequest(req2, privateKey)

	rec2 := httptest.NewRecorder()
	securedMux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for endpoint2, got %d", rec2.Code)
	}

	// Test endpoint1 without signature - should fail
	req3 := httptest.NewRequest("POST", "/endpoint1", strings.NewReader("body"))

	rec3 := httptest.NewRecorder()
	securedMux.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 without signature, got %d", rec3.Code)
	}
}

// Helper function to sign a request (mimics Restate signature)
func signRequest(req *http.Request, privateKey ed25519.PrivateKey) {
	// Read body
	bodyBytes, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// Construct signed message
	var builder strings.Builder
	builder.WriteString(req.Method)
	builder.WriteString(" ")
	builder.WriteString(req.URL.Path)
	if req.URL.RawQuery != "" {
		builder.WriteString("?")
		builder.WriteString(req.URL.RawQuery)
	}
	builder.WriteString("\n")
	builder.Write(bodyBytes)

	message := []byte(builder.String())

	// Sign
	signature := ed25519.Sign(privateKey, message)

	// Add header
	req.Header.Set("X-Restate-Signature", base64.StdEncoding.EncodeToString(signature))
}
