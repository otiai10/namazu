# Config Package

YAML configuration loader for Namazu earthquake monitoring system.

## Usage

```go
package main

import (
    "log"
    "github.com/otiai10/namazu/internal/config"
)

func main() {
    cfg, err := config.Load("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Source: %s at %s", cfg.Source.Type, cfg.Source.Endpoint)
    log.Printf("Webhooks: %d configured", len(cfg.Webhooks))
}
```

## Configuration Schema

```yaml
source:
  type: p2pquake
  endpoint: wss://api-realtime-sandbox.p2pquake.net/v2/ws

webhooks:
  - url: https://example.com/webhook1
    secret: your-secret-key
    name: optional-webhook-name
  - url: https://example.com/webhook2
    secret: another-secret
```

## Environment Variables

Environment variables override YAML configuration:

- `NAMAZU_SOURCE_ENDPOINT` - Overrides `source.endpoint`

## Validation Rules

The configuration is automatically validated when loaded:

1. Source type must be "p2pquake"
2. Source endpoint is required
3. At least one webhook must be configured
4. Each webhook requires:
   - `url` (required)
   - `secret` (required)
   - `name` (optional)

## Test Coverage

Current coverage: **96%**

Run tests:
```bash
go test ./internal/config/... -v -cover
```

## Error Handling

All errors are wrapped with descriptive messages:

```go
cfg, err := config.Load("nonexistent.yaml")
// Error: failed to read config file: open nonexistent.yaml: no such file or directory

cfg, err := config.Load("invalid.yaml")
// Error: failed to parse YAML: yaml: ...

cfg, err := config.Load("missing-webhooks.yaml")
// Error: invalid configuration: at least one webhook is required
```
