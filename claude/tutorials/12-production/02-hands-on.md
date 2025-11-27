# Hands-On: Production Deployment

> **Deploy a Restate application to production**

## 🎯 What We're Deploying

Production-ready order processing system with:
- Docker containerization
- Kubernetes orchestration  
- Health checks
- Monitoring
- Graceful shutdown

## 📝 Implementation

### Step 1: Dockerfile

```dockerfile
# Multi-stage build for smaller images
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /order-service

# Runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /order-service .

EXPOSE 9090
CMD ["./order-service"]
```

### Step 2: Docker Compose

```yaml
version: '3.8'
services:
  restate:
    image: restatedev/restate:latest
    ports:
      - "8080:8080"
      - "9070:9070"
      - "9091:9091"
    environment:
      - RESTATE_OBSERVABILITY__LOG__FORMAT=json
    volumes:
      - restate-data:/target
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9070/health"]
      interval: 10s
      timeout: 5s
      retries: 3
    restart: unless-stopped

  order-service:
    build: .
    ports:
      - "9090:9090"
    environment:
      - LOG_LEVEL=info
    depends_on:
      restate:
        condition: service_healthy
    restart: unless-stopped

volumes:
  restate-data:
```

### Step 3: Kubernetes Manifests

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: order-service
  template:
    metadata:
      labels:
        app: order-service
    spec:
      containers:
      - name: order-service
        image: order-service:latest
        ports:
        - containerPort: 9090
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: order-service
spec:
  selector:
    app: order-service
  ports:
  - port: 9090
    targetPort: 9090
```

### Step 4: Service with Health Checks

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    restate "github.com/restatedev/sdk-go"
    "github.com/restatedev/sdk-go/server"
)

var startTime = time.Now()

func main() {
    // Health check endpoint
    http.HandleFunc("/health", healthCheck)
    go http.ListenAndServe(":8081", nil)
    
    // Restate server
    restateServer := server.NewRestate()
    restateServer.Bind(restate.Reflect(OrderService{}))
    
    // Start server in goroutine
    go func() {
        if err := restateServer.Start(context.Background(), ":9090"); err != nil {
            log.Fatal(err)
        }
    }()
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down gracefully...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Cleanup
    log.Println("Server exited")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status": "healthy",
        "uptime": time.Since(startTime).Seconds(),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}
```

## 🚀 Deployment Steps

### Docker Deployment

```bash
# Build
docker-compose build

# Start
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

### Kubernetes Deployment

```bash
# Apply manifests
kubectl apply -f k8s/

# Check deployment
kubectl get pods
kubectl get services

# View logs
kubectl logs -f deployment/order-service

# Scale
kubectl scale deployment order-service --replicas=5

# Update
kubectl set image deployment/order-service \
  order-service=order-service:v2

# Rollback
kubectl rollout undo deployment/order-service
```

## 🎓 What You Learned

1. **Containerization** - Docker best practices
2. **Orchestration** - Kubernetes deployment
3. **Health Checks** - Liveness and readiness probes
4. **Graceful Shutdown** - Clean termination
5. **Production Patterns** - HA, monitoring, scaling

## 🚀 Next Steps

👉 **Continue to [Validation](./03-validation.md)**
