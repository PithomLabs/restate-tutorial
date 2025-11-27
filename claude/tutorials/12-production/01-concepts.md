# Concepts: Production & Deployment

> **Deploy and operate Restate applications at scale**

## 🎯 What You'll Learn

- Production deployment patterns
- High availability configuration
- Disaster recovery strategies
- Performance optimization
- Operational best practices

---

## 🚀 Deployment Patterns

### Pattern 1: Docker Deployment

Simplest deployment option:

```yaml
# docker-compose.yml
version: '3.8'
services:
  restate:
    image: restatedev/restate:latest
    ports:
      - "8080:8080"   # Ingress
      - "9070:9070"   # Admin API
      - "9091:9091"   # Metrics
    environment:
      - RESTATE_OBSERVABILITY__LOG__FORMAT=json
      - RESTATE_OBSERVABILITY__LOG__LEVEL=info
    volumes:
      - restate-data:/target
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9070/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  my-service:
    build: .
    ports:
      - "9090:9090"
    environment:
      - RESTATE_URL=http://restate:8080
    depends_on:
      restate:
        condition: service_healthy
    restart: unless-stopped

volumes:
  restate-data:
```

### Pattern 2: Kubernetes Deployment

Production-grade orchestration:

```yaml
# restate-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: restate
  labels:
    app: restate
spec:
  replicas: 3
  selector:
    matchLabels:
      app: restate
  template:
    metadata:
      labels:
        app: restate
    spec:
      containers:
      - name: restate
        image: restatedev/restate:latest
        ports:
        - containerPort: 8080
          name: ingress
        - containerPort: 9070
          name: admin
        - containerPort: 9091
          name: metrics
        env:
        - name: RESTATE_OBSERVABILITY__LOG__FORMAT
          value: "json"
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 9070
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 9070
          initialDelaySeconds: 10
          periodSeconds: 5
        volumeMounts:
        - name: data
          mountPath: /target
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: restate-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: restate
spec:
  selector:
    app: restate
  ports:
  - name: ingress
    port: 8080
    targetPort: 8080
  - name: admin
    port: 9070
    targetPort: 9070
  - name: metrics
    port: 9091
    targetPort: 9091
  type: LoadBalancer
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: restate-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
```

---

## 🏗️ High Availability

### Load Balancing

```yaml
# nginx.conf
upstream restate {
    least_conn;
    server restate-1:8080;
    server restate-2:8080;
    server restate-3:8080;
}

server {
    listen 80;
    
    location / {
        proxy_pass http://restate;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Health Checks

```go
// Service health check
func HealthCheck(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status": "healthy",
        "version": version,
        "uptime": time.Since(startTime).Seconds(),
        "checks": map[string]bool{
            "database": checkDatabase(),
            "cache": checkCache(),
        },
    }
    
    allHealthy := true
    for _, check := range health["checks"].(map[string]bool) {
        if !check {
            allHealthy = false
        }
    }
    
    if !allHealthy {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(health)
}
```

### Graceful Shutdown

```go
func main() {
    server := createServer()
    
    // Start server in goroutine
    go func() {
        if err := server.Start(context.Background(), ":9090"); err != nil {
            log.Fatal(err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down gracefully...")
    
    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }
    
    log.Println("Server exited")
}
```

---

## 💾 Backup and Recovery

### Backup Strategy

```bash
#!/bin/bash
# backup-restate.sh

BACKUP_DIR="/backups/restate"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/restate-$TIMESTAMP.tar.gz"

# Stop writes (optional, for consistent backup)
# Restate handles this internally

# Backup Restate data directory
tar -czf $BACKUP_FILE /var/restate/data

# Upload to S3
aws s3 cp $BACKUP_FILE s3://my-backups/restate/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "restate-*.tar.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_FILE"
```

### Restore Procedure

```bash
#!/bin/bash
# restore-restate.sh

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file>"
    exit 1
fi

# Stop Restate
systemctl stop restate

# Restore data
tar -xzf $BACKUP_FILE -C /

# Start Restate
systemctl start restate

echo "Restore completed"
```

---

## ⚡ Performance Optimization

### Restate Configuration

```bash
# Increase worker threads
RESTATE_WORKER__NUM_THREADS=8

# Tune concurrent invocations
RESTATE_WORKER__INVOKER__CONCURRENT_INVOCATIONS_LIMIT=1000

# Increase timer capacity
RESTATE_WORKER__TIMERS__NUM_TIMERS_IN_MEMORY_LIMIT=10000

# Configure memory limits
RESTATE_WORKER__INVOKER__INACTIVITY_TIMEOUT=1m
RESTATE_WORKER__INVOKER__ABORT_TIMEOUT=1m
```

### Service Optimization

```go
// Connection pooling
var db *sql.DB

func initDatabase() {
    db, _ = sql.Open("postgres", dsn)
    
    // Configure pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(10 * time.Minute)
}

// Caching
var cache = make(map[string]interface{})
var cacheMutex sync.RWMutex

func getFromCache(key string) (interface{}, bool) {
    cacheMutex.RLock()
    defer cacheMutex.RUnlock()
    val, ok := cache[key]
    return val, ok
}

// Batching
func processBatch(items []Item) error {
    const batchSize = 100
    
    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }
        
        batch := items[i:end]
        if err := processBatchInternal(batch); err != nil {
            return err
        }
    }
    
    return nil
}
```

---

## 📊 Monitoring Production

### Key Metrics

```promql
# Request rate
rate(restate_invocations_total[5m])

# P99 latency
histogram_quantile(0.99, 
    rate(restate_invocation_duration_seconds_bucket[5m]))

# Error rate
rate(restate_invocations_total{status="error"}[5m]) /
rate(restate_invocations_total[5m])

# Active invocations
restate_invocations_active

# Queue depth
restate_queue_depth
```

### Alerts

```yaml
# prometheus-alerts.yml
groups:
- name: restate
  rules:
  - alert: HighErrorRate
    expr: |
      rate(restate_invocations_total{status="error"}[5m]) /
      rate(restate_invocations_total[5m]) > 0.05
    for: 5m
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value | humanizePercentage }}"

  - alert: HighLatency
    expr: |
      histogram_quantile(0.99,
        rate(restate_invocation_duration_seconds_bucket[5m])) > 5
    for: 10m
    annotations:
      summary: "High P99 latency"
      description: "P99 latency is {{ $value }}s"

  - alert: ServiceDown
    expr: up{job="restate"} == 0
    for: 1m
    annotations:
      summary: "Restate service is down"
```

---

## 🚨 Incident Response

### Incident Checklist

1. **Detect** - Monitoring alerts trigger
2. **Assess** - Determine severity and impact
3. **Respond** - Mitigate immediate impact
4. **Communicate** - Update stakeholders
5. **Resolve** - Fix root cause
6. **Review** - Post-mortem analysis

### Common Issues

**High Latency:**
```bash
# Check resource usage
kubectl top pods

# Check active invocations
curl http://restate:9070/invocations | jq '.[] | select(.status=="running")'

# Check for slow queries
# Review traces in Jaeger

# Scale if needed
kubectl scale deployment restate --replicas=5
```

**Service Unavailable:**
```bash
# Check pod status
kubectl get pods

# Check logs
kubectl logs -f restate-xxx

# Check health endpoint
curl http://restate:9070/health

# Restart if needed
kubectl rollout restart deployment/restate
```

**Data Corruption:**
```bash
# Stop writes
kubectl scale deployment restate --replicas=0

# Restore from backup
./restore-restate.sh /backups/restate-20241122.tar.gz

# Verify data
# Run consistency checks

# Resume service
kubectl scale deployment restate --replicas=3
```

---

## 🔒 Production Checklist

### Before Deployment

- [ ] All tests passing (unit, integration, E2E)
- [ ] Security review completed
- [ ] Performance testing done
- [ ] Monitoring configured
- [ ] Alerting rules set
- [ ] Documentation updated
- [ ] Rollback plan ready
- [ ] Disaster recovery tested
- [ ] Load testing completed
- [ ] Dependencies updated

### Configuration

- [ ] Environment variables set
- [ ] Secrets managed securely (not in code)
- [ ] Resource limits configured
- [ ] Network policies defined
- [ ] Backup schedule configured
- [ ] Log retention set
- [ ] Metrics exported
- [ ] Health checks enabled

### Post-Deployment

- [ ] Monitor dashboards
- [ ] Check error rates
- [ ] Verify functionality
- [ ] Review logs
- [ ] Test rollback procedure
- [ ] Update runbooks
- [ ] Notify stakeholders

---

## ✅ Best Practices

### Infrastructure as Code

```hcl
# terraform/main.tf
resource "kubernetes_deployment" "restate" {
  metadata {
    name = "restate"
  }
  
  spec {
    replicas = 3
    
    template {
      spec {
        container {
          name  = "restate"
          image = "restatedev/restate:latest"
          
          resources {
            requests = {
              cpu    = "500m"
              memory = "1Gi"
            }
            limits = {
              cpu    = "1000m"
              memory = "2Gi"
            }
          }
        }
      }
    }
  }
}
```

### GitOps Workflow

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Run tests
      run: go test ./...
    
    - name: Build image
      run: docker build -t myservice:${{ github.sha }} .
    
    - name: Push image
      run: docker push myservice:${{ github.sha }}
    
    - name: Deploy to k8s
      run: |
        kubectl set image deployment/myservice \
          myservice=myservice:${{ github.sha }}
        kubectl rollout status deployment/myservice
```

---

## 🚀 Next Steps

You now understand production deployment!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

Deploy a production-ready Restate application!

---

**Questions?** Review this document or check the [module README](./README.md).
