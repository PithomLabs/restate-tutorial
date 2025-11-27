// ingress_chi.go
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/time/rate"
)

// ---------- Configuration via environment ----------
var (
	// RESTATE_GATEWAY_HTTP should be like "https://restate-gateway.example" (no trailing slash)
	ReStateGatewayBase = getenv("RESTATE_GATEWAY_HTTP", "http://localhost:2223")

	// optional token to authenticate to the Restate gateway
	ReStateGatewayToken = os.Getenv("RESTATE_GATEWAY_TOKEN")

	// JWT secret for incoming requests (HMAC). For real deployments, use JWKS or RSA public keys.
	JWTSecret = getenv("INGRESS_JWT_SECRET", "replace-with-secure-secret")

	// Rate limiter config (requests per second) and burst
	RateRPS   = getenvInt("INGRESS_RATE_RPS", 5)
	RateBurst = getenvInt("INGRESS_RATE_BURST", 10)
)

// ---------- JSON Schemas (inline) ----------
const schemaApprovalStart = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["amount", "description"],
  "properties": {
    "amount": { "type": "integer", "minimum": 0 },
    "description": { "type": "string" }
  },
  "additionalProperties": false
}`

const schemaApprovalCallback = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["awakeable_id", "approved"],
  "properties": {
    "awakeable_id": { "type": "string", "minLength": 1 },
    "approved": { "type": "boolean" }
  },
  "additionalProperties": false
}`

const schemaVerifyEmail = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["verified"],
  "properties": {
    "verified": { "type": "boolean" }
  },
  "additionalProperties": false
}`

// ---------- JSON schema loaders ----------
var (
	approvalStartLoader    = gojsonschema.NewStringLoader(schemaApprovalStart)
	approvalCallbackLoader = gojsonschema.NewStringLoader(schemaApprovalCallback)
	verifyEmailLoader      = gojsonschema.NewStringLoader(schemaVerifyEmail)
)

// ---------- Rate limiter per-IP ----------
type ipLimiter struct {
	limiter *rate.Limiter
	last    time.Time
}

var (
	limiterMap  sync.Map // map[string]*ipLimiter
	cleanupOnce sync.Once
)

func getLimiterForIP(ip string) *rate.Limiter {
	v, ok := limiterMap.Load(ip)
	if ok {
		return v.(*ipLimiter).limiter
	}
	lim := &ipLimiter{
		limiter: rate.NewLimiter(rate.Limit(RateRPS), RateBurst),
		last:    time.Now(),
	}
	limiterMap.Store(ip, lim)
	// spawn cleanup once
	cleanupOnce.Do(func() {
		go cleanupOldLimiters()
	})
	return lim.limiter
}

func cleanupOldLimiters() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		limiterMap.Range(func(key, val any) bool {
			il := val.(*ipLimiter)
			if now.Sub(il.last) > 30*time.Minute {
				limiterMap.Delete(key)
			}
			return true
		})
	}
}

// ---------- Helpers ----------
func getenv(key, d string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return d
}

func getenvInt(key string, d int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return d
}

func remoteIP(r *http.Request) string {
	// try X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// fallback to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func newTraceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- Tracing middleware ----------
func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the caller provided a trace header, forward. Otherwise generate one.
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = newTraceID()
		}
		// Set traceparent if not present (simple W3C-ish placeholder)
		if r.Header.Get("traceparent") == "" {
			r.Header.Set("traceparent", fmt.Sprintf("00-%s-%s-01", traceID, strings.Repeat("0", 32)))
		}
		// Always set X-Trace-Id
		r.Header.Set("X-Trace-Id", traceID)

		// Add to context for handlers
		ctx := context.WithValue(r.Context(), "trace_id", traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ---------- Rate limit middleware ----------
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := remoteIP(r)
		lim := getLimiterForIP(ip)
		if !lim.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		// update last
		if v, ok := limiterMap.Load(ip); ok {
			v.(*ipLimiter).last = time.Now()
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- Simple JWT auth middleware (HMAC) ----------
func jwtMiddleware(next http.Handler) http.Handler {
	secret := []byte(JWTSecret)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Expect: Authorization: Bearer <token>
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		parts := strings.Fields(auth)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]

		// Parse HMAC token. For production use RS256 with jwks.
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			// only HMAC allowed in this example
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// pass token claims to context if needed
		ctx := context.WithValue(r.Context(), "jwt_claims", token.Claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ---------- JSON schema validation helper ----------
func validateJSONSchema(schemaLoader gojsonschema.JSONLoader, body []byte) error {
	loader := gojsonschema.NewBytesLoader(body)
	result, err := gojsonschema.Validate(schemaLoader, loader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}
	if !result.Valid() {
		var sb strings.Builder
		for _, e := range result.Errors() {
			sb.WriteString(e.String())
			sb.WriteString("; ")
		}
		return fmt.Errorf("request does not conform to schema: %s", sb.String())
	}
	return nil
}

// ---------- Restate gateway invocation helper ----------
func callRestateFunction(ctx context.Context, service, handler string, workflowKey *string, body []byte) (int, []byte, error) {
	// function URL: {gateway}/functions/{service}/{handler}
	url := fmt.Sprintf("%s/functions/%s/%s", ReStateGatewayBase, service, handler)

	// If caller wants to run a workflow with a key, encode as query param ?workflow_key=...
	if workflowKey != nil && *workflowKey != "" {
		url = url + "?workflow_key=" + *workflowKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Propagate tracing headers
	if v := ctx.Value("trace_id"); v != nil {
		req.Header.Set("X-Trace-Id", fmt.Sprintf("%v", v))
	}
	// Propagate traceparent if present in incoming request context (try to read from ctx if available)
	if v := ctx.Value("traceparent"); v != nil {
		req.Header.Set("traceparent", fmt.Sprintf("%v", v))
	}

	// Attach gateway auth if configured
	if ReStateGatewayToken != "" {
		req.Header.Set("Authorization", "Bearer "+ReStateGatewayToken)
	}

	client := &http.Client{
		Timeout: 25 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, respBody, nil
}

// ---------- Handler implementations ----------

func handleApprovalStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	// validate schema
	if err := validateJSONSchema(approvalStartLoader, body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// call Restate function: ApprovalService.RequestApproval
	status, respBody, err := callRestateFunction(ctx, "ApprovalService", "RequestApproval", nil, body)
	if err != nil {
		http.Error(w, "gateway call failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	// passthrough status and body (attempt to forward JSON)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(respBody)
}

func handleApprovalCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	if err := validateJSONSchema(approvalCallbackLoader, body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// call Restate function: ApprovalService.HandleApprovalCallback
	status, respBody, err := callRestateFunction(ctx, "ApprovalService", "HandleApprovalCallback", nil, body)
	if err != nil {
		http.Error(w, "gateway call failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(respBody)
}

func handleOnboardingStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")
	// Build simple payload; adapt to your actual Run signature.
	payload := map[string]string{"user_id": userID}
	body, _ := json.Marshal(payload)

	// Call workflow Run with workflow_key set to userID
	status, respBody, err := callRestateFunction(ctx, "OnboardingWorkflow", "Run", &userID, body)
	if err != nil {
		http.Error(w, "gateway call failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(respBody)
}

func handleOnboardingVerify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	if err := validateJSONSchema(verifyEmailLoader, body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// This maps to a shared handler on the workflow instance (signal)
	//status, respBody, err := callRestateFunction(ctx, "OnboardingWorkflow", "VerifyEmail", &userID, body)
	status, respBody, err := callRestateFunction(ctx, "OrderWorkflow", "VerifyEmail", &userID, body)
	if err != nil {
		http.Error(w, "gateway call failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(respBody)
}

// ---------- Main ----------
func main() {
	// Basic validation
	if ReStateGatewayBase == "" {
		log.Fatal("RESTATE_GATEWAY_HTTP must be set")
	}

	r := chi.NewRouter()

	// Middleware stack:
	// 1) tracing
	// 2) rate limiting
	// 3) jwt auth
	// 4) JSON body size limit (e.g., 1MB)
	r.Use(tracingMiddleware)
	r.Use(rateLimitMiddleware)
	r.Use(jwtMiddleware)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.Post("/approval/start", handleApprovalStart)
	r.Post("/approval/callback", handleApprovalCallback)
	r.Post("/onboarding/start/{userID}", handleOnboardingStart)
	r.Post("/onboarding/verify/{userID}", handleOnboardingVerify)

	// Optional health
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	addr := getenv("INGRESS_ADDR", ":8080")
	log.Printf("Starting HTTP ingress on %s, gateway=%s", addr, ReStateGatewayBase)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("listen failed: %v", err)
	}
}
