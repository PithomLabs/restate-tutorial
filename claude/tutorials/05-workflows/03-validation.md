# Validation: Testing Workflows and Durable Promises

> **Verify promise resolution, timeout handling, and long-running workflows**

## 🎯 Objectives

Verify that:
- ✅ Workflows execute asynchronously
- ✅ Durable promises work correctly
- ✅ External handlers can resolve promises
- ✅ Timeouts trigger as expected
- ✅ Status queries work while workflow is pending
- ✅ Multiple workflows run independently

## 📋 Pre-Validation Checklist

- [ ] Restate server running (ports 8080/9080)
- [ ] Approval service running (port 9090)
- [ ] Service registered with Restate
- [ ] `curl` and `jq` available

## 🧪 Test Suite

### Test 1: Basic Workflow Execution

**Purpose:** Verify workflow starts and completes

```bash
# Start workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/test-doc-001/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test-doc-001",
    "title": "Test Document",
    "content": "Testing basic workflow",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'

# Check initial status
curl -s -X POST http://localhost:9080/ApprovalWorkflow/test-doc-001/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{documentId, status}'
```

**Expected:**
```json
{
  "documentId": "test-doc-001",
  "status": "pending"
}
```

**Validation:**
- ✅ Workflow started
- ✅ Initial status is "pending"

---

### Test 2: Promise Resolution (Approval)

**Purpose:** Verify external handler can resolve promise

```bash
# Start workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/test-doc-002/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test-doc-002",
    "title": "Document for Approval",
    "content": "Please approve this",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'

# Wait a moment
sleep 1

# Approve it
curl -X POST http://localhost:9080/ApprovalWorkflow/test-doc-002/Approve \
  -H 'Content-Type: application/json' \
  -d '{
    "approved": true,
    "approver": "bob",
    "comments": "Looks good!"
  }'

# Check final status
sleep 1
curl -s -X POST http://localhost:9080/ApprovalWorkflow/test-doc-002/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{status, approver, completedAt}'
```

**Expected:**
```json
{
  "status": "approved",
  "approver": "bob",
  "completedAt": "2024-01-15T10:01:00Z"
}
```

**Validation:**
- ✅ Promise resolved successfully
- ✅ Workflow resumed and completed
- ✅ Status changed to "approved"
- ✅ Approver recorded

---

### Test 3: Promise Resolution (Rejection)

**Purpose:** Verify rejection flow works

```bash
# Start workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/test-doc-003/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test-doc-003",
    "title": "Document for Rejection",
    "content": "This should be rejected",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'

# Reject it
curl -X POST http://localhost:9080/ApprovalWorkflow/test-doc-003/Reject \
  -H 'Content-Type: application/json' \
  -d '{
    "approved": false,
    "approver": "bob",
    "comments": "Needs more work"
  }'

# Check status
sleep 1
curl -s -X POST http://localhost:9080/ApprovalWorkflow/test-doc-003/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{status, approver}'
```

**Expected:**
```json
{
  "status": "rejected",
  "approver": "bob"
}
```

**Validation:**
- ✅ Rejection handled correctly
- ✅ Status changed to "rejected"

---

### Test 4: Multiple Independent Workflows

**Purpose:** Verify workflows don't interfere with each other

```bash
# Start workflow 1
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-A/Run \
  -H 'Content-Type: application/json' \
  -d '{"id": "doc-A", "title": "Doc A", "content": "A", "author": "alice", "submittedAt": "2024-01-15T10:00:00Z"}'

# Start workflow 2
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-B/Run \
  -H 'Content-Type: application/json' \
  -d '{"id": "doc-B", "title": "Doc B", "content": "B", "author": "bob", "submittedAt": "2024-01-15T10:00:00Z"}'

# Approve only doc-A
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-A/Approve \
  -H 'Content-Type: application/json' \
  -d '{"approved": true, "approver": "manager", "comments": "OK"}'

sleep 1

# Check doc-A status
echo "Doc A:"
curl -s -X POST http://localhost:9080/ApprovalWorkflow/doc-A/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{status}'

# Check doc-B status (should still be pending)
echo "Doc B:"
curl -s -X POST http://localhost:9080/ApprovalWorkflow/doc-B/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{status}'
```

**Expected:**
```
Doc A:
{
  "status": "approved"
}
Doc B:
{
  "status": "pending"
}
```

**Validation:**
- ✅ Workflows are isolated by key
- ✅ Approving doc-A doesn't affect doc-B
- ✅ Each has independent promise

---

### Test 5: Idempotent Promise Resolution

**Purpose:** Verify promise can only be resolved once

```bash
# Start workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/test-idem/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test-idem",
    "title": "Idempotency Test",
    "content": "Test",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'

# Approve it
curl -X POST http://localhost:9080/ApprovalWorkflow/test-idem/Approve \
  -H 'Content-Type: application/json' \
  -d '{"approved": true, "approver": "bob", "comments": "First approval"}'

sleep 1

# Try to approve again (should be idempotent or error)
curl -X POST http://localhost:9080/ApprovalWorkflow/test-idem/Approve \
  -H 'Content-Type: application/json' \
  -d '{"approved": true, "approver": "charlie", "comments": "Second approval"}'

# Check status - should still show bob as approver
curl -s -X POST http://localhost:9080/ApprovalWorkflow/test-idem/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{status, approver}'
```

**Expected:** First approval wins

**Validation:**
- ✅ Promise can only be resolved once
- ✅ Second resolution has no effect

---

### Test 6: Query Status While Pending

**Purpose:** Verify status queries work during workflow execution

```bash
# Start workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/test-pending/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test-pending",
    "title": "Pending Test",
    "content": "Test",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'

# Query status multiple times while pending
for i in {1..3}; do
  echo "Query $i:"
  curl -s -X POST http://localhost:9080/ApprovalWorkflow/test-pending/GetStatus \
    -H 'Content-Type: application/json' \
    -d 'null' | jq '{status}'
  sleep 1
done
```

**Expected:** All queries return "pending"

**Validation:**
- ✅ Can query status while workflow is waiting
- ✅ Status reflects current state

---

### Test 7: Workflow Invocation ID

**Purpose:** Verify workflow invocation tracking

```bash
# Start workflow with idempotency key
curl -s -X POST http://localhost:9080/ApprovalWorkflow/test-inv/Run \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: test-workflow-123' \
  -d '{
    "id": "test-inv",
    "title": "Invocation Test",
    "content": "Test",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'

# List invocations for this workflow
curl -s 'http://localhost:8080/invocations?target_service=ApprovalWorkflow&target_key=test-inv' | \
  jq '.invocations[] | {id, status, target_handler}'
```

**Expected:** Shows Run invocation

**Validation:**
- ✅ Workflow invocation tracked
- ✅ Can query invocation status

---

### Test 8: Journal Inspection

**Purpose:** Examine workflow journal entries

```bash
# Start and complete a workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/test-journal/Run \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: journal-test' \
  -d '{
    "id": "test-journal",
    "title": "Journal Test",
    "content": "Test",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'

sleep 1

curl -X POST http://localhost:9080/ApprovalWorkflow/test-journal/Approve \
  -H 'Content-Type: application/json' \
  -d '{"approved": true, "approver": "bob", "comments": "OK"}'

sleep 2

# Get invocation ID
INV_ID=$(curl -s 'http://localhost:8080/invocations?target_service=ApprovalWorkflow&target_key=test-journal&target_handler=Run' | \
  jq -r '.invocations[0].id')

echo "Invocation ID: $INV_ID"

# View journal
curl -s "http://localhost:8080/invocations/$INV_ID/journal" | \
  jq '.entries[] | {index, type, name}'
```

**Expected Journal Entries:**
- `SetState` - Initial status
- `Run` - Send notification
- `Sleep` / `CompleteAwakeable` - Promise/timeout race
- `SetState` - Final status
- `Output` - Result

**Validation:**
- ✅ Journal captures all operations
- ✅ Promise operations visible

---

### Test 9: Async Workflow Execution

**Purpose:** Verify workflow runs asynchronously

```bash
# Start workflow and measure response time
echo "Starting workflow..."
time curl -X POST http://localhost:9080/ApprovalWorkflow/test-async/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test-async",
    "title": "Async Test",
    "content": "Test",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'
```

**Expected:** Returns immediately (< 1 second)

**Validation:**
- ✅ Workflow start is non-blocking
- ✅ Response received quickly
- ✅ Workflow continues in background

---

### Test 10: Workflow State Persistence

**Purpose:** Verify state survives service restart

```bash
# Start workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/test-persist/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test-persist",
    "title": "Persistence Test",
    "content": "Test",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'

# Check status
curl -s -X POST http://localhost:9080/ApprovalWorkflow/test-persist/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null' > /tmp/status_before.json

# Restart service
echo "Restart your approval service now, then press Enter..."
read

# Check status after restart
curl -s -X POST http://localhost:9080/ApprovalWorkflow/test-persist/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null' > /tmp/status_after.json

# Compare
diff /tmp/status_before.json /tmp/status_after.json && \
  echo "✅ State survived restart!"
```

**Validation:**
- ✅ Workflow state persists across restarts
- ✅ Can still approve after restart

---

## 📊 Test Results Summary

| Test | Purpose | Expected | Pass/Fail |
|------|---------|----------|-----------|
| Basic Execution | Start workflow | Status "pending" | |
| Approval | Promise resolution | Status "approved" | |
| Rejection | Promise with rejection | Status "rejected" | |
| Independence | Multiple workflows | Isolated execution | |
| Idempotency | Duplicate resolution | First wins | |
| Status Query | Read while pending | Current status | |
| Invocation Tracking | View invocation | Tracked in Restate | |
| Journal | Inspect operations | All ops recorded | |
| Async | Non-blocking start | Fast response | |
| Persistence | Survive restart | State preserved | |

## ✅ Validation Checklist

- [ ] ✅ Workflows start asynchronously
- [ ] ✅ Promises created successfully
- [ ] ✅ External handlers resolve promises
- [ ] ✅ Approval flow works
- [ ] ✅ Rejection flow works
- [ ] ✅ Multiple workflows isolated
- [ ] ✅ Status queryable while pending
- [ ] ✅ Promise resolution is idempotent
- [ ] ✅ Journal tracks operations
- [ ] ✅ State persists across restarts

## 🎓 What You Learned

1. **Async Workflow Execution** - Workflows run in background
2. **Durable Promises** - Survive failures while waiting
3. **External Resolution** - External code can resume workflows
4. **State Isolation** - Each workflow ID is independent
5. **Queryable State** - Can check status anytime

## 🐛 Troubleshooting

### Promise Not Resolving

Check:
1. Using correct workflow ID in URL
2. Promise name matches exactly
3. Type matches (ApprovalDecision)

### Workflow Not Starting

Ensure:
1. Service registered
2. Using POST method
3. Correct endpoint format

### Status Always Pending

Verify:
1. Called Approve/Reject handler
2. Used correct workflow ID
3. No errors in service logs

## 🎯 Next Steps

Excellent! Your workflow implementation is working correctly.

Practice with more complex scenarios:

👉 **Continue to [Exercises](./04-exercises.md)**

---

**Questions?** Review [concepts](./01-concepts.md) or [hands-on](./02-hands-on.md)!
