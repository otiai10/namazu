# データモデル

## 設計方針

- **Event**: 地震・津波・気象警報など全イベントの抽象基底
- **Source**: P2P地震情報・気象庁APIなどデータソースの抽象化
- **Filter**: イベントタイプに応じたフィルタ条件の抽象化

## User

> **設計決定**: Account Linking 対応のため、LinkedProvider を埋め込み配列として保持。
> 詳細は ADR-001 参照。

```go
type User struct {
    ID          string           `firestore:"-"`
    UID         string           `firestore:"uid"`           // Identity Platform UID
    Email       string           `firestore:"email"`
    DisplayName string           `firestore:"displayName"`
    PictureURL  string           `firestore:"pictureUrl,omitempty"`
    Plan        string           `firestore:"plan"`          // "free" | "pro"
    Providers   []LinkedProvider `firestore:"providers"`     // Account Linking
    CreatedAt   time.Time        `firestore:"createdAt"`
    UpdatedAt   time.Time        `firestore:"updatedAt"`
    LastLoginAt time.Time        `firestore:"lastLoginAt"`

    // Stripe 連携
    StripeCustomerID     string    `firestore:"stripeCustomerId,omitempty"`
    SubscriptionID       string    `firestore:"subscriptionId,omitempty"`
    SubscriptionStatus   string    `firestore:"subscriptionStatus,omitempty"`
    SubscriptionEndsAt   time.Time `firestore:"subscriptionEndsAt,omitempty"`
}

type LinkedProvider struct {
    ProviderID  string    `firestore:"providerId"`  // "google.com", "apple.com", "password"
    Subject     string    `firestore:"subject"`     // OIDC sub claim
    Email       string    `firestore:"email,omitempty"`
    DisplayName string    `firestore:"displayName,omitempty"`
    LinkedAt    time.Time `firestore:"linkedAt"`
}
```

## Event（抽象基底）

```go
type Event interface {
    GetID() string
    GetType() EventType       // "earthquake" | "tsunami" | "weather" | ...
    GetSource() string        // "p2pquake" | "jma" | ...
    GetSeverity() int         // 0-100 の正規化された重大度
    GetAffectedAreas() []string
    GetOccurredAt() time.Time
    GetReceivedAt() time.Time
    GetRawJSON() string
}

type EventType string

const (
    EventTypeEarthquake EventType = "earthquake"
    EventTypeTsunami    EventType = "tsunami"
    EventTypeWeather    EventType = "weather"    // 将来拡張
    EventTypeVolcano    EventType = "volcano"    // 将来拡張
)
```

## EventRecord（Firestore 保存用）

```go
type EventRecord struct {
    ID            string    `firestore:"-"`
    Type          string    `firestore:"type"`
    Source        string    `firestore:"source"`
    Severity      int       `firestore:"severity"`
    AffectedAreas []string  `firestore:"affectedAreas"`
    OccurredAt    time.Time `firestore:"occurredAt"`
    ReceivedAt    time.Time `firestore:"receivedAt"`
    RawJSON       string    `firestore:"rawJson"`
    Details       string    `firestore:"details"`  // イベント固有データ（JSON）
}
```

## EarthquakeDetails（地震固有データ）

```go
type EarthquakeDetails struct {
    MaxScale    int         `json:"maxScale"`
    Hypocenter  Hypocenter  `json:"hypocenter"`
    Tsunami     string      `json:"tsunami"`
    Points      []PointInfo `json:"points"`
}

type Hypocenter struct {
    Name      string  `json:"name"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Depth     int     `json:"depth"`
    Magnitude float64 `json:"magnitude"`
}

type PointInfo struct {
    Prefecture string `json:"prefecture"`
    Name       string `json:"name"`
    Scale      int    `json:"scale"`
}
```

## Subscription

```go
type Subscription struct {
    ID        string          `firestore:"-"`
    UserID    string          `firestore:"userId"`
    Name      string          `firestore:"name"`
    Enabled   bool            `firestore:"enabled"`
    Filter    *FilterConfig   `firestore:"filter,omitempty"`
    Delivery  DeliveryConfig  `firestore:"delivery"`
    CreatedAt time.Time       `firestore:"createdAt"`
    UpdatedAt time.Time       `firestore:"updatedAt"`
}

type DeliveryConfig struct {
    Type     string       `firestore:"type"`     // "webhook" | "slack" | "discord" | "line" | "email"
    URL      string       `firestore:"url"`
    Secret   string       `firestore:"secret"`
    Retry    *RetryConfig `firestore:"retry,omitempty"`
    Template string       `firestore:"template,omitempty"` // Pro: カスタムペイロード
}

type FilterConfig struct {
    // 基本フィルタ（Free + Pro）
    MinScale    int      `firestore:"minScale,omitempty"`
    Prefectures []string `firestore:"prefectures,omitempty"`

    // 詳細フィルタ（Pro のみ）
    MinDepth     *int     `firestore:"minDepth,omitempty"`
    MaxDepth     *int     `firestore:"maxDepth,omitempty"`
    MinMagnitude *float64 `firestore:"minMagnitude,omitempty"`
    TsunamiOnly  bool     `firestore:"tsunamiOnly,omitempty"`
}

type RetryConfig struct {
    Enabled    bool `firestore:"enabled"`
    MaxRetries int  `firestore:"maxRetries"`  // Default: 3
    InitialMs  int  `firestore:"initialMs"`   // Default: 1000
    MaxMs      int  `firestore:"maxMs"`       // Default: 60000
}
```

## Source（データソース抽象化）

```go
type Source interface {
    ID() string                          // "p2pquake" | "jma" | ...
    Connect(ctx context.Context) error
    Events() <-chan Event
    Close() error
}

type SourceConfig struct {
    Type     string            `yaml:"type"`     // "p2pquake" | "jma"
    Endpoint string            `yaml:"endpoint"`
    Options  map[string]string `yaml:"options"`
}
```

## 震度 → Severity 変換

```go
func ScaleToSeverity(scale int) int {
    switch scale {
    case 10: return 10  // 震度1
    case 20: return 20  // 震度2
    case 30: return 30  // 震度3
    case 40: return 40  // 震度4
    case 45: return 50  // 震度5弱
    case 50: return 60  // 震度5強
    case 55: return 70  // 震度6弱
    case 60: return 80  // 震度6強
    case 70: return 100 // 震度7
    default: return 0
    }
}
```
