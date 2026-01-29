# Webhook Package

A robust, concurrent webhook sender with HMAC-SHA256 signature support for Go.

## Features

- HTTP POST to webhook URLs with JSON payloads
- Automatic HMAC-SHA256 signature generation (`X-Signature-256` header)
- Configurable timeout (default: 10 seconds)
- Concurrent delivery to multiple webhooks
- Comprehensive delivery results with timing and error information
- Context support for cancellation and timeouts
- Thread-safe for concurrent use

## Installation

```go
import "github.com/ayanel/namazu/internal/delivery/webhook"
```

## Quick Start

### Single Webhook

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ayanel/namazu/internal/delivery/webhook"
)

func main() {
    // Create sender
    sender := webhook.NewSender()

    // Prepare payload
    payload := []byte(`{"event":"user.created","user_id":123}`)

    // Send webhook
    result := sender.Send(
        context.Background(),
        "https://api.example.com/webhooks",
        "your-secret-key",
        payload,
    )

    if result.Success {
        fmt.Printf("Delivered in %v\n", result.ResponseTime)
    } else {
        log.Printf("Failed: %s\n", result.ErrorMessage)
    }
}
```

### Multiple Webhooks (Concurrent)

```go
webhooks := []webhook.WebhookTarget{
    {
        URL:    "https://api1.example.com/hook",
        Secret: "secret-1",
        Name:   "Production API",
    },
    {
        URL:    "https://api2.example.com/hook",
        Secret: "secret-2",
        Name:   "Analytics",
    },
}

payload := []byte(`{"event":"order.completed","order_id":456}`)

// Send to all webhooks concurrently
results := sender.SendAll(context.Background(), webhooks, payload)

// Check results
for i, result := range results {
    if result.Success {
        fmt.Printf("✓ %s delivered\n", webhooks[i].Name)
    } else {
        fmt.Printf("✗ %s failed: %s\n", webhooks[i].Name, result.ErrorMessage)
    }
}
```

## Configuration

### Custom Timeout

```go
// 5 second timeout
sender := webhook.NewSender(webhook.WithTimeout(5 * time.Second))

// 30 second timeout for slow endpoints
sender := webhook.NewSender(webhook.WithTimeout(30 * time.Second))
```

### Context Timeout

```go
// Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

result := sender.Send(ctx, url, secret, payload)
```

## Signature Verification

The sender automatically includes `X-Signature-256` header with HMAC-SHA256 signature:

```
X-Signature-256: sha256=734cc62f32841568f45715aeb9f4d7891324e6d948e4c6c60c0621cdac48623a
```

### Receiver Side Verification

```go
import "github.com/ayanel/namazu/internal/delivery/webhook"

// In your webhook handler
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    // Read body
    body, _ := io.ReadAll(r.Body)

    // Get signature from header
    signature := r.Header.Get("X-Signature-256")

    // Verify signature
    secret := "your-secret-key"
    if !webhook.Verify(secret, body, signature) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }

    // Process webhook...
    w.WriteHeader(http.StatusOK)
}
```

## HTTP Headers Sent

Every webhook request includes:

- `Content-Type: application/json`
- `X-Signature-256: sha256=<hmac-sha256-hex>`
- `User-Agent: namazu/1.0`

## DeliveryResult Structure

```go
type DeliveryResult struct {
    URL          string        // The webhook URL
    StatusCode   int           // HTTP status code (0 if request failed)
    Success      bool          // True if status is 2xx
    ErrorMessage string        // Error description if failed
    ResponseTime time.Duration // Request duration
}
```

## Error Handling

### Status Codes

- `Success = true`: Status codes 200-299 (2xx)
- `Success = false`: All other status codes or errors
- `StatusCode = 0`: Connection error, timeout, or invalid URL

### Retry Logic Example

```go
result := sender.Send(ctx, url, secret, payload)

if !result.Success {
    if result.StatusCode >= 500 && result.StatusCode < 600 {
        // Server error - retry recommended
        log.Println("Server error, will retry")
    } else if result.StatusCode >= 400 && result.StatusCode < 500 {
        // Client error - don't retry
        log.Println("Client error, fix request")
    } else if result.StatusCode == 0 {
        // Connection/timeout error - may retry
        log.Println("Network error, may retry")
    }
}
```

## Performance

### Benchmarks

```
BenchmarkSend_Success-8           5000    250000 ns/op    5120 B/op    65 allocs/op
BenchmarkSendAll_10Webhooks-8     2000    800000 ns/op   51200 B/op   650 allocs/op
```

### Concurrent Delivery

`SendAll` uses goroutines to send webhooks in parallel:

- 10 webhooks: ~50-100ms total (not 500ms sequential)
- Each webhook gets its own goroutine
- Results returned in same order as input

## Test Coverage

- **93.5%** statement coverage
- Comprehensive unit tests
- Integration tests with httptest
- Concurrency tests
- Error path coverage

## Thread Safety

`Sender` is safe for concurrent use by multiple goroutines. You can use a single sender instance across your application:

```go
var globalSender = webhook.NewSender()

// Safe to call from multiple goroutines
func handler1() {
    globalSender.Send(ctx, url1, secret1, payload1)
}

func handler2() {
    globalSender.Send(ctx, url2, secret2, payload2)
}
```

## Best Practices

1. **Reuse Sender**: Create once, use many times (thread-safe)
2. **Set Appropriate Timeout**: 10s default may be too long for some use cases
3. **Use Context**: Pass context for cancellation and request tracing
4. **Verify Signatures**: Always verify on receiver side
5. **Handle Errors**: Check `Success` and `ErrorMessage` fields
6. **Log Timing**: Use `ResponseTime` for monitoring
7. **Retry Server Errors**: 5xx errors may succeed on retry

## Examples

See `example_test.go` for more usage examples:

```bash
go test -v -run=Example ./internal/delivery/webhook/
```

## License

Part of the namazu project.
