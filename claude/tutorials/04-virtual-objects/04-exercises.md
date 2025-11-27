# Exercises: Practice Virtual Objects

> **Master stateful services with hands-on exercises**

## 🎯 Objectives

Practice:
- Building Virtual Objects with state
- Exclusive vs concurrent handlers
- State transitions and validation
- Cross-object interactions
- Real-world stateful patterns

## 📚 Exercise Levels

- 🟢 **Beginner** - Basic state management
- 🟡 **Intermediate** - State transitions
- 🔴 **Advanced** - Complex state machines

---

## Exercise 1: User Wallet 🟢

**Goal:** Build a Virtual Object for managing user wallet balances

### Requirements

1. Create `Wallet` Virtual Object with handlers:
   - `Deposit(amount)` - Add funds (exclusive)
   - `Withdraw(amount)` - Remove funds with validation (exclusive)
   - `GetBalance()` - View current balance (concurrent)
   - `GetTransactionHistory()` - View last 10 transactions (concurrent)

2. Track transaction history with timestamps

3. Prevent negative balances

### Starter Code

```go
type Wallet struct{}

type Transaction struct {
    Type      string    `json:"type"` // "deposit" or "withdraw"
    Amount    float64   `json:"amount"`
    Balance   float64   `json:"balance"` // Balance after transaction
    Timestamp time.Time `json:"timestamp"`
}

type WalletState struct {
    Balance      float64       `json:"balance"`
    Transactions []Transaction `json:"transactions"`
}

func (Wallet) Deposit(
    ctx restate.ObjectContext,
    amount float64,
) error {
    if amount <= 0 {
        return restate.TerminalError(fmt.Errorf("amount must be positive"), 400)
    }
    
    // TODO: Get current state
    state, _ := restate.Get[WalletState](ctx, "wallet")
    
    // TODO: Update balance
    state.Balance += amount
    
    // TODO: Add transaction to history
    tx := Transaction{
        Type:      "deposit",
        Amount:    amount,
        Balance:   state.Balance,
        Timestamp: time.Now(),
    }
    state.Transactions = append(state.Transactions, tx)
    
    // Keep only last 10 transactions
    if len(state.Transactions) > 10 {
        state.Transactions = state.Transactions[len(state.Transactions)-10:]
    }
    
    // TODO: Save state
    restate.Set(ctx, "wallet", state)
    
    return nil
}

func (Wallet) Withdraw(
    ctx restate.ObjectContext,
    amount float64,
) error {
    // TODO: Validate amount
    // TODO: Get state
    // TODO: Check sufficient balance
    // TODO: Update balance
    // TODO: Add transaction
    // TODO: Save state
}

func (Wallet) GetBalance(
    ctx restate.ObjectSharedContext,
    _ restate.Void,
) (float64, error) {
    // TODO: Get and return balance
}

func (Wallet) GetTransactionHistory(
    ctx restate.ObjectSharedContext,
    _ restate.Void,
) ([]Transaction, error) {
    // TODO: Get and return transactions
}
```

### Test

```bash
# Deposit
curl -X POST http://localhost:9080/Wallet/user123/Deposit \
  -H 'Content-Type: application/json' \
  -d '100.00'

# Check balance
curl -X POST http://localhost:9080/Wallet/user123/GetBalance \
  -H 'Content-Type: application/json' \
  -d 'null'

# Withdraw
curl -X POST http://localhost:9080/Wallet/user123/Withdraw \
  -H 'Content-Type: application/json' \
  -d '30.00'

# View history
curl -X POST http://localhost:9080/Wallet/user123/GetTransactionHistory \
  -H 'Content-Type: application/json' \
  -d 'null'
```

---

## Exercise 2: Inventory Management 🟡

**Goal:** Track product inventory with reservations

### Requirements

1. Create `Inventory` Virtual Object (keyed by product ID)
2. Support:
   - `Restock(quantity)` - Add inventory
   - `Reserve(orderId, quantity)` - Reserve for order
   - `ConfirmReservation(orderId)` - Commit reservation
   - `CancelReservation(orderId)` - Return to available
   - `GetAvailability()` - View available quantity

3. State should track:
   - Total quantity
   - Available quantity
   - Reserved quantities per order

### Starter Code

```go
type Inventory struct{}

type InventoryState struct {
    TotalQuantity     int                `json:"totalQuantity"`
    AvailableQuantity int                `json:"availableQuantity"`
    Reservations      map[string]int `json:"reservations"` // orderID -> quantity
}

type Availability struct {
    ProductID  string `json:"productId"`
    Available  int    `json:"available"`
    Reserved   int    `json:"reserved"`
    Total      int    `json:"total"`
}

func (Inventory) Restock(
    ctx restate.ObjectContext,
    quantity int,
) error {
    productID := restate.Key(ctx)
    
    state, _ := restate.Get[InventoryState](ctx, "inventory")
    
    // TODO: Add to total and available
    
    restate.Set(ctx, "inventory", state)
    return nil
}

func (Inventory) Reserve(
    ctx restate.ObjectContext,
    req struct {
        OrderID  string `json:"orderId"`
        Quantity int    `json:"quantity"`
    },
) error {
    // TODO: Check available quantity
    // TODO: Move from available to reserved
    // TODO: Track reservation by order ID
}

func (Inventory) ConfirmReservation(
    ctx restate.ObjectContext,
    orderID string,
) error {
    // TODO: Remove from total (items shipped)
    // TODO: Remove reservation record
}

func (Inventory) CancelReservation(
    ctx restate.ObjectContext,
    orderID string,
) error {
    // TODO: Move from reserved back to available
    // TODO: Remove reservation record
}

func (Inventory) GetAvailability(
    ctx restate.ObjectSharedContext,
    _ restate.Void,
) (Availability, error) {
    // TODO: Return availability info
}
```

### Test Flow

```bash
# Initial stock
curl -X POST http://localhost:9080/Inventory/WIDGET-001/Restock \
  -H 'Content-Type: application/json' \
  -d '100'

# Reserve for order
curl -X POST http://localhost:9080/Inventory/WIDGET-001/Reserve \
  -H 'Content-Type: application/json' \
  -d '{"orderId": "ORD-123", "quantity": 10}'

# Check availability (should show 90 available, 10 reserved)
curl -X POST http://localhost:9080/Inventory/WIDGET-001/GetAvailability \
  -H 'Content-Type: application/json' \
  -d 'null'

# Confirm reservation (items shipped)
curl -X POST http://localhost:9080/Inventory/WIDGET-001/ConfirmReservation \
  -H 'Content-Type: application/json' \
  -d '"ORD-123"'
```

---

## Exercise 3: Order State Machine 🟡

**Goal:** Implement order lifecycle with state transitions

### Requirements

1. Create `Order` Virtual Object with statuses:
   - `pending` → `confirmed` → `shipped` → `delivered`
   - `pending` → `cancelled`
   
2. Handlers:
   - `CreateOrder(items)` - Initialize order
   - `ConfirmPayment()` - pending → confirmed
   - `Ship(trackingNumber)` - confirmed → shipped
   - `Deliver()` - shipped → delivered
   - `Cancel()` - pending → cancelled
   - `GetStatus()` - View current status

3. Validate state transitions (can't ship before confirming)

### Starter Code

```go
type Order struct{}

type OrderStatus string

const (
    StatusPending   OrderStatus = "pending"
    StatusConfirmed OrderStatus = "confirmed"
    StatusShipped   OrderStatus = "shipped"
    StatusDelivered OrderStatus = "delivered"
    StatusCancelled OrderStatus = "cancelled"
)

type OrderState struct {
    OrderID        string      `json:"orderId"`
    Status         OrderStatus `json:"status"`
    Items          []OrderItem `json:"items"`
    TrackingNumber string      `json:"trackingNumber,omitempty"`
    CreatedAt      time.Time   `json:"createdAt"`
    UpdatedAt      time.Time   `json:"updatedAt"`
}

type OrderItem struct {
    ProductID string  `json:"productId"`
    Quantity  int     `json:"quantity"`
    Price     float64 `json:"price"`
}

func (Order) CreateOrder(
    ctx restate.ObjectContext,
    items []OrderItem,
) error {
    orderID := restate.Key(ctx)
    
    state := OrderState{
        OrderID:   orderID,
        Status:    StatusPending,
        Items:     items,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    restate.Set(ctx, "order", state)
    return nil
}

func (Order) ConfirmPayment(
    ctx restate.ObjectContext,
    _ restate.Void,
) error {
    state, _ := restate.Get[OrderState](ctx, "order")
    
    // Validate transition
    if state.Status != StatusPending {
        return restate.TerminalError(
            fmt.Errorf("can only confirm pending orders, current: %s", state.Status),
            400,
        )
    }
    
    // TODO: Update status to confirmed
    // TODO: Update timestamp
    // TODO: Save state
}

func (Order) Ship(
    ctx restate.ObjectContext,
    trackingNumber string,
) error {
    // TODO: Validate status is confirmed
    // TODO: Update to shipped
    // TODO: Save tracking number
}

func (Order) Deliver(
    ctx restate.ObjectContext,
    _ restate.Void,
) error {
    // TODO: Validate status is shipped
    // TODO: Update to delivered
}

func (Order) Cancel(
    ctx restate.ObjectContext,
    _ restate.Void,
) error {
    // TODO: Can only cancel if pending
    // TODO: Update to cancelled
}

func (Order) GetStatus(
    ctx restate.ObjectSharedContext,
    _ restate.Void,
) (OrderState, error) {
    // TODO: Return current state
}
```

### Test State Machine

```bash
# Create order
curl -X POST http://localhost:9080/Order/ORD-123/CreateOrder \
  -H 'Content-Type: application/json' \
  -d '[{"productId": "WIDGET", "quantity": 2, "price": 50.00}]'

# Confirm payment
curl -X POST http://localhost:9080/Order/ORD-123/ConfirmPayment \
  -H 'Content-Type: application/json' \
  -d 'null'

# Ship
curl -X POST http://localhost:9080/Order/ORD-123/Ship \
  -H 'Content-Type: application/json' \
  -d '"TRACK-123456"'

# Try to cancel (should fail - already shipped)
curl -X POST http://localhost:9080/Order/ORD-123/Cancel \
  -H 'Content-Type: application/json' \
  -d 'null'

# Deliver
curl -X POST http://localhost:9080/Order/ORD-123/Deliver \
  -H 'Content-Type: application/json' \
  -d 'null'

# Check final status
curl -X POST http://localhost:9080/Order/ORD-123/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null'
```

---

## Exercise 4: Multi-Tenant Counter 🟢

**Goal:** Global counter with per-tenant isolation

### Requirements

1. Create `Counter` Virtual Object (keyed by tenant ID)
2. Handlers:
   - `Increment()` - Add 1
   - `IncrementBy(amount)` - Add amount
   - `Decrement()` - Subtract 1
   - `Reset()` - Set to 0
   - `GetValue()` - View current value
   - `GetStats()` - Total increments/decrements

### Starter Code

```go
type Counter struct{}

type CounterState struct {
    Value            int       `json:"value"`
    TotalIncrements  int       `json:"totalIncrements"`
    TotalDecrements  int       `json:"totalDecrements"`
    LastUpdated      time.Time `json:"lastUpdated"`
}

type CounterStats struct {
    Value       int       `json:"value"`
    Increments  int       `json:"increments"`
    Decrements  int       `json:"decrements"`
    LastUpdated time.Time `json:"lastUpdated"`
}

func (Counter) Increment(
    ctx restate.ObjectContext,
    _ restate.Void,
) (int, error) {
    state, _ := restate.Get[CounterState](ctx, "counter")
    
    // TODO: Increment value
    // TODO: Track stats
    // TODO: Save state
    // TODO: Return new value
}

// TODO: Implement IncrementBy, Decrement, Reset, GetValue, GetStats
```

---

## Exercise 5: Session Manager 🔴

**Goal:** Manage user sessions with expiration

### Requirements

1. Create `Session` Virtual Object (keyed by session ID)
2. Features:
   - `Create(userId, ttlMinutes)` - Start session
   - `Refresh()` - Extend expiration
   - `AddData(key, value)` - Store session data
   - `GetData(key)` - Retrieve data
   - `Invalidate()` - End session
   - `IsValid()` - Check if expired

3. Auto-cleanup after TTL expires using `restate.Sleep`

### Starter Code

```go
type Session struct{}

type SessionState struct {
    UserID     string                 `json:"userId"`
    Data       map[string]interface{} `json:"data"`
    CreatedAt  time.Time              `json:"createdAt"`
    ExpiresAt  time.Time              `json:"expiresAt"`
    LastAccess time.Time              `json:"lastAccess"`
}

func (Session) Create(
    ctx restate.ObjectContext,
    req struct {
        UserID     string `json:"userId"`
        TTLMinutes int    `json:"ttlMinutes"`
    },
) error {
    sessionID := restate.Key(ctx)
    
    expiresAt := time.Now().Add(time.Duration(req.TTLMinutes) * time.Minute)
    
    state := SessionState{
        UserID:     req.UserID,
        Data:       make(map[string]interface{}),
        CreatedAt:  time.Now(),
        ExpiresAt:  expiresAt,
        LastAccess: time.Now(),
    }
    
    restate.Set(ctx, "session", state)
    
    // TODO: Schedule auto-cleanup
    // Use restate.Sleep and restate.Send to self
    
    return nil
}

func (Session) Refresh(
    ctx restate.ObjectContext,
    ttlMinutes int,
) error {
    // TODO: Get state
    // TODO: Check if expired
    // TODO: Extend expiration
    // TODO: Save state
}

func (Session) IsValid(
    ctx restate.ObjectSharedContext,
    _ restate.Void,
) (bool, error) {
    state, _ := restate.Get[SessionState](ctx, "session")
    
    // Check if expired
    return time.Now().Before(state.ExpiresAt), nil
}

// TODO: Implement AddData, GetData, Invalidate
```

---

## Exercise 6: Leaderboard 🔴

**Goal:** Per-game leaderboard with top scores

### Requirements

1. Create `Leaderboard` Virtual Object (keyed by game ID)
2. Handlers:
   - `SubmitScore(player, score)`
   - `GetTopN(n)` - Get top N players
   - `GetPlayerRank(player)` - Get specific player's rank
   - `GetPlayerScore(player)` - Get specific player's score
3. Keep top 100 scores in sorted order

### Starter Code

```go
type Leaderboard struct{}

type ScoreEntry struct {
    Player    string    `json:"player"`
    Score     int       `json:"score"`
    Timestamp time.Time `json:"timestamp"`
}

type LeaderboardState struct {
    Scores []ScoreEntry `json:"scores"` // Sorted descending
}

func (Leaderboard) SubmitScore(
    ctx restate.ObjectContext,
    req struct {
        Player string `json:"player"`
        Score  int    `json:"score"`
    },
) (int, error) {
    // TODO: Get state
    // TODO: Find player's current entry
    // TODO: Update if new score is higher
    // TODO: Re-sort scores
    // TODO: Keep top 100 only
    // TODO: Return player's new rank
}

func (Leaderboard) GetTopN(
    ctx restate.ObjectSharedContext,
    n int,
) ([]ScoreEntry, error) {
    // TODO: Get state
    // TODO: Return first n entries
}

func (Leaderboard) GetPlayerRank(
    ctx restate.ObjectSharedContext,
    player string,
) (int, error) {
    // TODO: Find player in sorted list
    // TODO: Return position (1-indexed)
}
```

---

## ✅ Exercise Checklist

- [ ] Exercise 1: User Wallet (Beginner)
- [ ] Exercise 2: Inventory Management (Intermediate)
- [ ] Exercise 3: Order State Machine (Intermediate)
- [ ] Exercise 4: Multi-Tenant Counter (Beginner)
- [ ] Exercise 5: Session Manager (Advanced)
- [ ] Exercise 6: Leaderboard (Advanced)

## 📁 Solutions

Complete solutions available in [solutions/](./solutions/):

- `exercise1_wallet.go`
- `exercise2_inventory.go`
- `exercise3_order_state.go`
- `exercise4_counter.go`
- `exercise5_session.go`
- `exercise6_leaderboard.go`

## 🎯 Next Module

Congratulations! You've mastered Virtual Objects.

You now understand:
- ✅ Stateful services with durable state
- ✅ Key-based addressing and isolation
- ✅ Exclusive vs concurrent handlers
- ✅ State transitions and validation
- ✅ Real-world stateful patterns

Ready to learn about long-running orchestrations?

👉 **Continue to [Module 5: Workflows](../05-workflows/README.md)**

Learn to build durable workflows with human-in-the-loop and async await patterns!

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
