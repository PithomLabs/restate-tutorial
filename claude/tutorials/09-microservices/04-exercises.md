# Exercises: Microservices Orchestration Practice

> **Build your own orchestration systems**

## 🎯 Learning Objectives

Practice coordinating distributed microservices using orchestration patterns.

---

## Exercise 1: E-Commerce Order Fulfillment ⭐⭐

**Goal:** Build complete order fulfillment orchestrator

### Services Needed
- `InventoryService` - Check and reserve stock
- `PaymentService` - Process payment
- `WarehouseService` - Pick and pack
- `ShippingService` - Create label and ship
- `OrderOrchestrator` - Coordinate workflow

### Workflow
1. Check inventory availability
2. Reserve inventory
3. Charge customer
4. Create warehouse pick ticket
5. Create shipping label
6. Send tracking email

### Compensation
- Release inventory if payment fails
- Refund payment if shipping fails

### Success Criteria
- [ ] Complete workflow implemented
- [ ] Compensation logic works
- [ ] Idempotent operations
- [ ] All services coordinated

---

## Exercise 2: Bank Transfer Orchestration ⭐⭐⭐

**Goal:** Implement distributed money transfer

### Services
- `AccountService` - Debit/credit accounts
- `FraudService` - Check for fraud
- `ComplianceService` - Regulatory checks
- `TransferOrchestrator` - Coordinate transfer

### Workflow
1. Validate source account has funds
2. Run fraud check
3. Run compliance check
4. Debit source account
5. Credit destination account
6. Send confirmation

### Requirements
- Atomic transfer (all or nothing)
- Fraud detection blocks transfer
- Compensation if credit fails (refund debit)

### Success Criteria
- [ ] Atomic transfers
- [ ] Fraud blocking works
- [ ] Compensation implemented
- [ ] Audit trail maintained

---

## Exercise 3: Food Delivery Coordination ⭐⭐

**Goal:** Coordinate restaurant, driver, and customer

### Services
- `RestaurantService` - Accept/prepare orders
- `DriverService` - Assign and track drivers
- `PaymentService` - Process payment
- `DeliveryOrchestrator` - Main coordinator

### Workflow
1. Submit order to restaurant
2. Charge customer
3. Assign driver
4. Wait for restaurant ready (webhook/awakeable)
5. Notify driver to pickup
6. Track delivery
7. Complete order

### Requirements
- Wait for async restaurant confirmation
- Reassign driver if unavailable

### Success Criteria
- [ ] Async operations handled
- [ ] Driver assignment works
- [ ] Order tracking implemented
- [ ] Compensation for cancellations

---

## Exercise 4: CI/CD Pipeline Orchestration ⭐⭐⭐

**Goal:** Build deployment pipeline orchestrator

### Services
- `BuildService` - Compile code
- `TestService` - Run tests
- `SecurityService` - Security scan
- `DeployService` - Deploy to environment
- `PipelineOrchestrator` - Coordinate pipeline

### Workflow
1. Trigger build
2. Run unit tests (parallel with build)
3. Run security scan
4. Deploy to staging
5. Run integration tests
6. Deploy to production (manual approval)

### Requirements
- Parallel build + test
- Manual approval step (use awakeable)
- Rollback on test failure

### Success Criteria
- [ ] Parallel execution works
- [ ] Manual approval implemented
- [ ] Rollback on failure
- [ ] Pipeline status tracking

---

## Exercise 5: Multi-Tenant SaaS Provisioning ⭐⭐

**Goal:** Provision new tenant resources

### Services
- `DatabaseService` - Create tenant DB
- `StorageService` - Create S3 bucket
- `AuthService` - Create tenant auth
- `BillingService` - Setup billing
- `ProvisioningOrchestrator` - Coordinate

### Workflow
1. Create database schema
2. Setup S3 bucket
3. Configure authentication
4. Setup billing
5. Send welcome email

### Requirements
- All resources created atomically
- Cleanup if any step fails
- Tenant ID tracked

### Success Criteria
- [ ] All resources created
- [ ] Cleanup on failure
- [ ] Idempotent provisioning
- [ ] Status tracking

---

## 💡 General Tips

### Orchestration Best Practices

1. **Central coordinator**
   ```go
   type Orchestrator struct{}
   func (Orchestrator) Execute(ctx restate.WorkflowContext)
   ```

2. **Use futures for parallelism**
   ```go
   fut1 := service1.CallFuture()
   fut2 := service2.CallFuture()
   result1, _ := fut1.Response()
   result2, _ := fut2.Response()
   ```

3. **Implement compensation**
   ```go
   res1, _ := service1.Reserve()
   res2, err := service2.Reserve()
   if err != nil {
       service1.Cancel(res1.ID)  // Compensate
   }
   ```

4. **Track workflow state**
   ```go
   type WorkflowState struct {
       Status string
       Step   int
       Results map[string]interface{}
   }
   ```

## 📚 Resources

- [Concepts](./01-concepts.md)
- [Hands-On](./02-hands-on.md)
- [Validation](./03-validation.md)

---

**Good luck orchestrating!** 🚀
