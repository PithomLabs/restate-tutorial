# Module 12: Production & Deployment

> **Deploy and operate Restate applications in production**

## 🎯 Learning Objectives

By completing this module, you will:
- ✅ Deploy Restate to production
- ✅ Configure for high availability
- ✅ Implement disaster recovery
- ✅ Optimize performance
- ✅ Monitor production systems
- ✅ Handle operational issues

## 📚 Module Content

### 1. Deployment Strategies (~25 min)
- Docker deployment
- Kubernetes deployment
- Cloud provider deployment (AWS, GCP, Azure)
- Configuration management
- Scaling strategies

### 2. Operations (~25 min)
- Health checks
- Backup and restore
- Disaster recovery
- Performance tuning
- Capacity planning

### 3. Best Practices (~20 min)
- Production checklist
- Monitoring and alerting
- Incident response
- Change management
- SLA management

## 🎯 Key Deployment Patterns

### Docker Deployment

```yaml
# docker-compose.yml
version: '3'
services:
  restate:
    image: restatedev/restate:latest
    ports:
      - "8080:8080"
      - "9070:9070"
    environment:
      - RESTATE_OBSERVABILITY__LOG__FORMAT=json
    volumes:
      - restate-data:/target
    restart: unless-stopped
  
  my-service:
    build: .
    ports:
      - "9090:9090"
    environment:
      - RESTATE_URL=http://restate:8080
    depends_on:
      - restate
```

### Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: restate
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
        - containerPort: 9070
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
```

## 🏗️ High Availability

### Restate Server HA

**Cluster Mode (Coming Soon)**
- Multiple Restate nodes
- Distributed state
- Automatic failover
- Load balancing

**Current Best Practices:**
- Deploy behind load balancer
- Use persistent storage
- Regular backups
- Monitoring and alerts

### Service HA

```go
// Health check endpoint
func HealthCheck(w http.ResponseWriter, r *http.Request) {
    status := map[string]string{
        "status": "healthy",
        "version": version,
        "uptime": uptime(),
    }
    json.NewEncoder(w).Encode(status)
}

// Register health check
http.HandleFunc("/health", HealthCheck)
```

## 📊 Production Checklist

### Before Deployment

- [ ] All tests passing
- [ ] Security review completed
- [ ] Performance testing done
- [ ] Monitoring configured
- [ ] Alerting rules set
- [ ] Documentation updated
- [ ] Rollback plan ready
- [ ] Disaster recovery tested

### Configuration

- [ ] Environment variables set
- [ ] Secrets managed securely
- [ ] Resource limits configured
- [ ] Network policies defined
- [ ] Backup schedule configured
- [ ] Log retention set
- [ ] Metrics exported

### Monitoring

- [ ] Health checks enabled
- [ ] Metrics dashboard created
- [ ] Log aggregation configured
- [ ] Error tracking set up
- [ ] Alerts configured
- [ ] On-call rotation defined

## 🔧 Performance Optimization

### Restate Configuration

```bash
# Increase parallelism
RESTATE_WORKER__INVOKER__CONCURRENT_INVOCATIONS_LIMIT=1000

# Tune timers
RESTATE_WORKER__TIMERS__NUM_TIMERS_IN_MEMORY_LIMIT=10000

# Logging
RESTATE_OBSERVABILITY__LOG__LEVEL=info
RESTATE_OBSERVABILITY__LOG__FORMAT=json
```

### Service Optimization

```go
// Use connection pooling
db := sql.Open("postgres", dsn)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5*time.Minute)

// Cache frequently accessed data
var cache = make(map[string]interface{})

// Batch operations
func ProcessBatch(items []Item) {
    // Process multiple items together
}
```

## 🚨 Incident Response

### Incident Checklist

1. **Detect** - Monitoring alerts
2. **Assess** - Determine severity
3. **Respond** - Mitigate impact
4. **Communicate** - Update stakeholders
5. **Resolve** - Fix root cause
6. **Review** - Post-mortem

### Common Issues

**High Latency**
- Check resource utilization
- Review slow queries
- Analyze traces
- Scale services

**Service Unavailable**
- Check health endpoints
- Review error logs
- Verify network connectivity
- Restart if needed

**Data Inconsistency**
- Review workflow state
- Check compensation logic
- Audit logs
- Run consistency checks

## 📈 Scaling Strategies

### Vertical Scaling
- Increase CPU/memory
- Faster storage
- Better network

### Horizontal Scaling
- More service instances
- Load balancing
- Distributed processing

### Auto-Scaling

```yaml
# HorizontalPodAutoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-service
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## 🎓 Success Criteria

Production-ready when:
- [x] Deployed with HA
- [x] Monitoring comprehensive
- [x] Backups automated
- [x] Alerts configured
- [x] Incidents handled smoothly
- [x] Performance optimized
- [x] Security hardened
- [x] Documentation complete

## 🎓 Learning Path

**Current Module:** Production & Deployment  
**Previous:** [Security](../11-security/README.md)  
**Series Complete!** 🎉

---

## 🎉 Congratulations!

You've completed the Restate Go Tutorial Series!

You now know how to:
- Build resilient distributed applications
- Implement durable workflows
- Coordinate microservices
- Handle failures gracefully
- Deploy to production

**Next Steps:**
- Build your own Restate applications
- Join the Restate community
- Contribute to the ecosystem
- Share your learnings

---

**Happy building with Restate!** 🚀
