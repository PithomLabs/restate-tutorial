# Module 00: Prerequisites & Setup

> **Setting up your development environment for Restate development**

## 📋 What You'll Need

### Required Software

| Software | Minimum Version | Purpose | Installation |
|----------|----------------|---------|--------------|
| **Go** | 1.21+ | Primary language | [Download](https://go.dev/dl/) |
| **Restate Server** | Latest | Durable execution runtime | [Download](https://github.com/restatedev/restate/releases) |
| **Git** | 2.0+ | Version control | [Download](https://git-scm.com/) |
| **curl** | Any | Testing HTTP endpoints | Usually pre-installed |

### Recommended Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| **VS Code** | Code editor | [Download](https://code.visualstudio.com/) |
| **Go Extension** | Go language support | [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=golang.go) |
| **Postman** or **Insomnia** | API testing | [Postman](https://www.postman.com/) / [Insomnia](https://insomnia.rest/) |
| **jq** | JSON processing | `brew install jq` or [Download](https://stedolan.github.io/jq/) |

## 🔧 Installation Steps

### 1. Install Go

```bash
# Verify Go installation
go version

# Should output: go version go1.21.x or higher
```

If not installed:
- **macOS**: `brew install go`
- **Linux**: Download from [golang.org](https://go.dev/dl/)
- **Windows**: Download installer from [golang.org](https://go.dev/dl/)

### 2. Set Up Go Environment

```bash
# Verify GOPATH
echo $GOPATH

# If empty, add to your shell profile (~/.bashrc, ~/.zshrc, etc.):
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

### 3. Install Restate Server

You mentioned you already have Restate server downloaded. Let's verify:

```bash
# Check if restate binary is available
restate --version

# If not in PATH, note the location where you downloaded it
```

**If you need to download Restate:**

```bash
# macOS/Linux - Download latest release
curl -LO https://github.com/restatedev/restate/releases/latest/download/restate-server-linux-x64.tar.gz

# Extract
tar -xzf restate-server-linux-x64.tar.gz

# Make executable and move to PATH
chmod +x restate-server
sudo mv restate-server /usr/local/bin/restate

# Verify
restate --version
```

### 4. Create Workspace Directory

```bash
# Create a dedicated workspace for tutorials
mkdir -p ~/restate-tutorials
cd ~/restate-tutorials

# Initialize Go module
go mod init restate-tutorials

# Install Restate Go SDK
go get github.com/restatedev/sdk-go
```

### 5. Verify Installation

Create a test file to verify everything works:

```bash
# Create test file
cat > test.go << 'EOF'
package main

import (
    "context"
    "fmt"
    "log"
    
    restate "github.com/restatedev/sdk-go"
    "github.com/restatedev/sdk-go/server"
)

type HelloService struct{}

func (HelloService) SayHello(ctx restate.Context, name string) (string, error) {
    return fmt.Sprintf("Hello, %s!", name), nil
}

func main() {
    if err := server.NewRestate().
        Bind(restate.Reflect(HelloService{})).
        Start(context.Background(), ":9080"); err != nil {
        log.Fatal(err)
    }
}
EOF

# Test compilation
go mod tidy
go build -o test-service test.go

# Clean up
rm test.go test-service
```

If this succeeds, you're ready to go! 🎉

## 🚀 Starting Restate Server

### Start the Server

Open a terminal and start Restate:

```bash
# Start Restate server (keep this running)
restate-server
```

You should see output like:

```
INFO Restate server started
INFO Listening on http://0.0.0.0:8080 (admin API)
INFO Listening on http://0.0.0.0:9080 (ingress)
```

**Key Ports:**
- **8080** - Admin API (for registering services)
- **9080** - Ingress (for calling services)

> **💡 Tip:** Keep the Restate server running in a dedicated terminal window throughout the tutorials

### Verify Server is Running

In a new terminal:

```bash
# Check server health
curl http://localhost:8080/health

# Should return: {"status":"ready"}
```

## 📚 Understanding the Components

### Restate Architecture

```
┌─────────────────────────────────────────────────────┐
│                 Your Application                     │
│  ┌──────────────┐  ┌──────────────┐                │
│  │   Service A  │  │   Service B  │                │
│  │   (Port      │  │   (Port      │                │
│  │    9090)     │  │    9091)     │                │
│  └──────┬───────┘  └──────┬───────┘                │
└─────────┼──────────────────┼──────────────────────────┘
          │                  │
          │   Register       │
          └────────┬─────────┘
                   ▼
         ┌─────────────────────┐
         │  Restate Server     │
         │                     │
         │  Admin API: 8080    │◄──── Register services
         │  Ingress:   9080    │◄──── Call services
         │                     │
         │  Journal & State    │
         └─────────────────────┘
```

### Development Workflow

1. **Write** your service in Go
2. **Start** your service (e.g., on port 9090)
3. **Register** service with Restate
4. **Call** service via Restate ingress
5. **Observe** durability and retries in action

## 🧪 Quick Smoke Test

Let's run a complete end-to-end test:

### 1. Start Restate Server

```bash
# Terminal 1
restate-server
```

### 2. Create and Run Test Service

```bash
# Terminal 2
cd ~/restate-tutorials

# Create minimal service
cat > smoke-test.go << 'EOF'
package main

import (
    "context"
    "fmt"
    "log"
    
    restate "github.com/restatedev/sdk-go"
    "github.com/restatedev/sdk-go/server"
)

type TestService struct{}

func (TestService) Echo(ctx restate.Context, msg string) (string, error) {
    ctx.Log().Info("Received message", "msg", msg)
    return fmt.Sprintf("Echo: %s", msg), nil
}

func main() {
    fmt.Println("Starting test service on :9090...")
    if err := server.NewRestate().
        Bind(restate.Reflect(TestService{})).
        Start(context.Background(), ":9090"); err != nil {
        log.Fatal(err)
    }
}
EOF

# Run the service
go run smoke-test.go
```

### 3. Register the Service

```bash
# Terminal 3 - Register with Restate
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

You should see a success response with service details.

### 4. Call the Service

```bash
# Call via Restate ingress
curl -X POST http://localhost:9080/TestService/Echo \
  -H 'Content-Type: application/json' \
  -d '"Hello Restate!"'

# Should return: "Echo: Hello Restate!"
```

### 5. Clean Up

```bash
# Stop the test service (Ctrl+C in Terminal 2)
# Remove test file
rm smoke-test.go
```

## ✅ Checklist

Before proceeding to Module 1, ensure you have:

- [ ] Go 1.21+ installed and verified
- [ ] Restate server installed and can start
- [ ] Created `~/restate-tutorials` workspace
- [ ] Installed Restate Go SDK
- [ ] Successfully ran smoke test
- [ ] Understand basic workflow (write → register → call)

## 🐛 Troubleshooting

### Port Already in Use

```bash
# If 8080 or 9080 is in use, find the process
lsof -i :8080
lsof -i :9080

# Kill if necessary
kill -9 <PID>
```

### Go Module Issues

```bash
# Clean and reinitialize
rm go.mod go.sum
go mod init restate-tutorials
go mod tidy
```

### Restate Server Won't Start

```bash
# Check if another instance is running
ps aux | grep restate

# Check logs for errors
restate-server --log-filter=debug
```

### Cannot Register Service

**Error:** `connection refused`
- Ensure your service is running on the correct port
- Verify the port is not blocked by firewall

**Error:** `service already registered`
- Delete the deployment: `curl -X DELETE http://localhost:8080/deployments/<id>`

## 📖 Additional Setup (Optional)

### Docker (Optional)

If you prefer Docker for Restate:

```bash
docker run -d --name restate \
  -p 8080:8080 -p 9080:9080 \
  ghcr.io/restatedev/restate:latest
```

### IDE Configuration

**VS Code settings.json:**

```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.formatTool": "goimports",
  "[go]": {
    "editor.formatOnSave": true
  }
}
```

## 🎯 Next Steps

You're all set! Continue to:

👉 **[Module 1: Hello Durable World](../01-foundation/README.md)**

---

**Having issues?** Check the [Troubleshooting Guide](../appendix/troubleshooting.md) or ask in the Restate community!
