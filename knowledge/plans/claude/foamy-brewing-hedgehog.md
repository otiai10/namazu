# namazu - 地震速報 Webhook 中継サーバー

## プロジェクト概要

P2P地震情報 WebSocket API から地震情報を受信し、登録された Webhook に通知する中間サーバー。

```
[P2P地震情報 API] --WebSocket--> [namazu] --HTTP POST--> [登録済み Webhook]
```

## 技術スタック

| カテゴリ | 選択 |
|---------|------|
| 言語 | Go (標準ライブラリ net/http) |
| データベース | Google Datastore |
| 認証 | Google Identity Platform (OAuth) |
| ホスティング | GCP Compute Engine (Docker) |
| 監視 | Cloud Logging / Monitoring |
| IaC | Pulumi (Go) |

## インフラストラクチャ

### GCP リソース
| リソース | 設定 |
|----------|------|
| リージョン | us-west1 (無料枠対象) |
| Compute Engine | e2-micro (無料枠) |
| Datastore | Firestore in Datastore mode |
| Container Registry | Artifact Registry |

### Pulumi 構成
```
infra/
├── main.go           # Pulumi エントリーポイント
├── Pulumi.yaml       # プロジェクト設定
├── Pulumi.dev.yaml   # 開発環境
├── Pulumi.prod.yaml  # 本番環境
└── go.mod
```

---

## データソース

### P2P地震情報 API
- **本番**: `wss://api.p2pquake.net/v2/ws`
- **サンドボックス**: `wss://api-realtime-sandbox.p2pquake.net/v2/ws`
- 10分で強制切断 → 再接続ロジック必須
- 震度値: 10(震度1), 20(2), 30(3), 40(4), 45(5弱), 50(5強), 55(6弱), 60(6強), 70(7)
- 重複配信あり → `id` で重複排除

---

## フェーズ分け

### Phase 1: CLI ツール（コア機能）
**目標**: 動く最小限のプロトタイプ

#### 機能
- WebSocket で P2P地震情報 API に接続
- 10分ごとの再接続ロジック
- 設定ファイル (YAML) から Webhook URL を読み込み
- 地震情報受信時に Webhook へ POST
- HMAC-SHA256 署名付き

#### ディレクトリ構成
```
namazu/
├── cmd/
│   └── namazu/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go      # YAML設定読み込み
│   ├── p2pquake/
│   │   ├── client.go      # WebSocket クライアント
│   │   └── types.go       # データ型定義
│   ├── webhook/
│   │   ├── sender.go      # Webhook POST
│   │   └── signer.go      # HMAC 署名
│   └── app/
│       └── app.go         # アプリケーション制御
├── config.example.yaml
├── go.mod
└── go.sum
```

#### 設定ファイル例
```yaml
source:
  type: p2pquake
  endpoint: wss://api.p2pquake.net/v2/ws
  # endpoint: wss://api-realtime-sandbox.p2pquake.net/v2/ws  # サンドボックス

webhooks:
  - url: https://example.com/earthquake
    secret: your-secret-key
```

### Phase 2: REST API + Datastore
**目標**: Webhook の動的管理

#### 追加機能
- REST API で Webhook CRUD (認証なし)
- Google Datastore に Webhook 登録情報を永続化
- 地震情報履歴の保存

#### 追加ファイル
```
├── internal/
│   ├── api/
│   │   ├── server.go      # HTTP サーバー
│   │   ├── handler.go     # ハンドラー
│   │   └── middleware.go  # ミドルウェア
│   ├── store/
│   │   ├── datastore.go   # Datastore クライアント
│   │   ├── webhook.go     # Webhook リポジトリ
│   │   └── earthquake.go  # 地震履歴リポジトリ
│   └── model/
│       ├── webhook.go     # Webhook エンティティ
│       └── earthquake.go  # 地震情報エンティティ
```

#### API エンドポイント
```
POST   /api/subscriptions          # Subscription 登録
GET    /api/subscriptions          # Subscription 一覧
GET    /api/subscriptions/:id      # Subscription 詳細
PUT    /api/subscriptions/:id      # Subscription 更新
DELETE /api/subscriptions/:id      # Subscription 削除
GET    /api/events                 # 地震履歴一覧
```

### Phase 3: Google Identity Platform 認証
**目標**: ユーザー認証とマルチテナント（Account Linking 対応）

#### 設計決定
- **ADR-001**: User (1) → Subscription (N) の関係を採用
- **Provider 埋め込み**: LinkedProvider を User ドキュメント内に配列として埋め込み（サブコレクションではない）
- **Account Linking**: 1ユーザーが複数の Identity Provider をリンク可能

#### データモデル

```go
// User は認証済みユーザー（Firestore に保存）
type User struct {
    ID          string           `firestore:"-"`
    UID         string           `firestore:"uid"`         // Identity Platform UID
    Email       string           `firestore:"email"`
    DisplayName string           `firestore:"displayName"`
    PictureURL  string           `firestore:"pictureUrl,omitempty"`
    Plan        string           `firestore:"plan"`        // "free" | "pro"
    Providers   []LinkedProvider `firestore:"providers"`   // Account Linking
    CreatedAt   time.Time        `firestore:"createdAt"`
    UpdatedAt   time.Time        `firestore:"updatedAt"`
    LastLoginAt time.Time        `firestore:"lastLoginAt"`
}

// LinkedProvider はリンクされた認証プロバイダー
type LinkedProvider struct {
    ProviderID  string    `firestore:"providerId"`  // "google.com", "apple.com", "password"
    Subject     string    `firestore:"subject"`     // OIDC sub claim（プロバイダーにおけるユーザー識別子）
    Email       string    `firestore:"email,omitempty"`
    DisplayName string    `firestore:"displayName,omitempty"`
    LinkedAt    time.Time `firestore:"linkedAt"`
}
```

#### 追加ファイル
```
internal/
├── user/
│   ├── user.go              # User, LinkedProvider モデル
│   ├── repository.go        # UserRepository インターフェース
│   ├── firestore.go         # Firestore 実装
│   └── firestore_test.go
├── auth/
│   ├── auth.go              # Claims, TokenVerifier インターフェース
│   ├── firebase_auth.go     # Firebase Admin SDK による検証
│   ├── context.go           # UserContext ヘルパー
│   ├── middleware.go        # 認証ミドルウェア
│   └── middleware_test.go
├── api/
│   ├── me_handler.go        # /api/me エンドポイント
│   └── me_handler_test.go
```

#### 変更ファイル
```
internal/
├── subscription/
│   ├── subscription.go      # UserID フィールド追加
│   └── firestore.go         # ListByUserID メソッド追加
├── api/
│   ├── handler.go           # 所有権チェック追加
│   └── router.go            # 認証ミドルウェア適用
├── config/
│   └── config.go            # AuthConfig 追加
cmd/
└── namazu/
    └── main.go              # 認証コンポーネント初期化
```

#### API エンドポイント
```
# Public（認証不要）
GET    /health                    # ヘルスチェック
GET    /api/events                # 地震履歴一覧

# Protected（認証必須）
GET    /api/me                    # 現在のユーザープロファイル
PUT    /api/me                    # プロファイル更新
GET    /api/me/providers          # リンク済みプロバイダー一覧
POST   /api/subscriptions         # Subscription 作成（UserID 自動設定）
GET    /api/subscriptions         # 自分の Subscription 一覧
GET    /api/subscriptions/:id     # Subscription 詳細（所有権チェック）
PUT    /api/subscriptions/:id     # Subscription 更新（所有権チェック）
DELETE /api/subscriptions/:id     # Subscription 削除（所有権チェック）
```

#### API パス設計方針
- **ユーザー向け API**: `/api/...` - 一般ユーザーがアクセス
- **Admin API（将来）**: `/admin-api/...` - 管理者専用エンドポイント

#### 依存関係追加
```go
require (
    firebase.google.com/go/v4 v4.14.0
)
```

#### 環境変数
```
NAMAZU_AUTH_ENABLED=true
NAMAZU_AUTH_PROJECT_ID=namazu-live
NAMAZU_AUTH_CREDENTIALS=path/to/serviceaccount.json  # ローカル開発のみ
```

#### 実装ステップ

**Step 1: User ドメイン層**
1. `internal/user/user.go` - User, LinkedProvider モデル（イミュータブル設計）
2. `internal/user/repository.go` - UserRepository インターフェース
3. `internal/user/firestore.go` - Firestore 実装
4. ユニットテスト作成

**Step 2: 認証層**
1. Firebase Admin SDK 依存追加
2. `internal/auth/auth.go` - Claims, TokenVerifier
3. `internal/auth/firebase_auth.go` - JWT 検証
4. `internal/auth/context.go` - コンテキストヘルパー
5. `internal/auth/middleware.go` - 認証ミドルウェア
6. ユニットテスト作成

**Step 3: Subscription 更新**
1. UserID フィールド追加
2. ListByUserID メソッド追加
3. テスト更新

**Step 4: API 更新**
1. `internal/api/me_handler.go` 作成（/api/me エンドポイント）
2. 既存ハンドラーに所有権チェック追加
3. ルーターに認証ミドルウェア適用
4. インテグレーションテスト作成

**Step 5: 設定・統合**
1. AuthConfig 追加
2. main.go 更新
3. .env.example 更新

#### 検証方法
1. Firebase Emulator で JWT 発行テスト
2. 認証なしリクエスト → 401 確認
3. 認証ありリクエスト → ユーザー作成・取得確認
4. 他ユーザーの Subscription 操作 → 403 確認
5. Account Linking のシミュレーション

### Phase 4: フィルタ・リトライ・制限
**目標**: 本格運用に向けた機能拡張

#### 追加機能
- Webhook ごとのフィルタ条件（最小震度閾値など）
- 失敗時のリトライ（指数バックオフ、設定可能）
- ユーザーごとの制限（Webhook 数上限、レートリミット）

---

## Phase 4.1 詳細実装計画: フィルタ適用

### 概要
Subscription ごとに設定されたフィルタ条件に基づいて、イベント配信をフィルタリングする。

### 現状
- `FilterConfig` は既に定義済み（`internal/subscription/subscription.go`）
- `EventRecord` に `Severity` と `AffectedAreas` が存在
- `handleEvent()` でフィルタ適用ロジックが**未実装**

### 変更ファイル

#### 1. `internal/subscription/filter.go` (新規)
フィルタマッチングロジックを実装。

```go
package subscription

import "github.com/ayanel/namazu/internal/source"

// MatchesFilter checks if an event matches the subscription's filter criteria.
// Returns true if:
//   - Filter is nil (no filter = match all)
//   - Event passes all configured filter conditions
func (f *FilterConfig) Matches(event source.Event) bool {
    if f == nil {
        return true // No filter = match all
    }

    // Check MinScale
    if f.MinScale > 0 && event.GetSeverity() < f.MinScale {
        return false
    }

    // Check Prefectures
    if len(f.Prefectures) > 0 {
        if !matchesPrefectures(f.Prefectures, event.GetAffectedAreas()) {
            return false
        }
    }

    return true
}

// matchesPrefectures checks if any affected area matches the filter prefectures
func matchesPrefectures(filterPrefectures, affectedAreas []string) bool {
    for _, area := range affectedAreas {
        for _, pref := range filterPrefectures {
            if area == pref || strings.HasPrefix(area, pref) {
                return true
            }
        }
    }
    return false
}
```

#### 2. `internal/app/app.go` (修正)
`handleEvent()` にフィルタ適用ロジックを追加。

```go
// 変更箇所: targets 生成部分
targets := make([]webhook.Target, 0, len(subscriptions))
for _, sub := range subscriptions {
    if sub.Delivery.Type == "webhook" {
        // ★ フィルタチェックを追加
        if sub.Filter != nil && !sub.Filter.Matches(event) {
            log.Printf("Subscription [%s]: filtered out (MinScale=%d, Prefectures=%v)",
                sub.Name, sub.Filter.MinScale, sub.Filter.Prefectures)
            continue
        }
        targets = append(targets, webhook.Target{
            URL:    sub.Delivery.URL,
            Secret: sub.Delivery.Secret,
            Name:   sub.Name,
        })
    }
}
```

#### 3. `internal/subscription/filter_test.go` (新規)
フィルタロジックのユニットテスト。

### テストケース
1. フィルタなし → 全イベント通過
2. MinScale=40 → 震度4未満は除外
3. Prefectures=["東京都"] → 東京都以外は除外
4. MinScale + Prefectures 複合条件
5. 空の AffectedAreas → Prefectures フィルタ通過しない

### 検証方法
```bash
# ユニットテスト
go test ./internal/subscription/... -v -run TestFilter

# 統合テスト（サンドボックス環境）
source .env.test && go run ./cmd/namazu/
# → 震度10のイベントが MinScale=40 の Subscription に配信されないことを確認
```

### 実装ステップ
1. `internal/subscription/filter.go` 作成
2. `internal/subscription/filter_test.go` 作成
3. `internal/app/app.go` の `handleEvent()` 修正
4. `internal/app/app_test.go` にフィルタテスト追加
5. 統合テスト実施

---

### Phase 5: Web UI・課金モデル
**目標**: 一般公開サービス化

#### 追加機能
- Webhook 管理用 Web UI
- 無料/有料プラン
- ダッシュボード（配信履歴、統計）

---

## Phase 1 詳細実装計画

### 1. プロジェクト初期化
```bash
go mod init github.com/ayanel/namazu
```

### 2. 主要コンポーネント

#### P2P地震情報クライアント (`internal/p2pquake/`)
- `gorilla/websocket` を使用
- 再接続ロジック（10分タイマー + エラー時リトライ）
- `id` による重複排除
- コード 551 (JMAQuake) のフィルタリング

#### Webhook 送信 (`internal/webhook/`)
- HMAC-SHA256 署名を `X-Signature-256` ヘッダーに付与
- Content-Type: application/json
- タイムアウト設定

#### 設定管理 (`internal/config/`)
- YAML ファイル読み込み
- 環境変数によるオーバーライド

### 3. テスト計画
- サンドボックス環境を使った統合テスト
- 再接続ロジックのユニットテスト
- HMAC 署名のユニットテスト

---

## 検証方法

### Phase 1 検証
1. サンドボックス WebSocket に接続
2. テスト用 Webhook サーバー (ngrok + local server) を準備
3. 地震情報が正しく POST されることを確認
4. HMAC 署名の検証
5. 再接続が正しく動作することを確認（10分以上稼働）

---

## セキュリティ考慮事項

- [ ] Webhook URL の SSRF 対策（プライベート IP ブロック）
- [ ] シークレットの安全な保管
- [ ] HTTPS のみ許可
- [ ] レートリミット実装（Phase 4）

---

---

## データモデル（抽象化スキーマ）

### 設計方針
- **Event**: 地震・津波・気象警報など全イベントの抽象基底
- **Source**: P2P地震情報・気象庁APIなどデータソースの抽象化
- **Filter**: イベントタイプに応じたフィルタ条件の抽象化

### User（Phase 3 以降）

> **設計決定**: Account Linking 対応のため、LinkedProvider を埋め込み配列として保持。
> `GoogleID` ではなく `UID` (Identity Platform 共通ID) + `Providers` 配列を使用。
> 詳細は ADR-001 および Phase 3 セクション参照。

```go
type User struct {
    ID          string           `firestore:"-"`
    UID         string           `firestore:"uid"`           // Identity Platform UID（全プロバイダー共通）
    Email       string           `firestore:"email"`
    DisplayName string           `firestore:"displayName"`
    PictureURL  string           `firestore:"pictureUrl,omitempty"`
    Plan        string           `firestore:"plan"`          // "free" | "pro"
    Providers   []LinkedProvider `firestore:"providers"`     // Account Linking
    CreatedAt   time.Time        `firestore:"createdAt"`
    UpdatedAt   time.Time        `firestore:"updatedAt"`
    LastLoginAt time.Time        `firestore:"lastLoginAt"`
}

type LinkedProvider struct {
    ProviderID  string    `firestore:"providerId"`  // "google.com", "apple.com", "password"
    Subject     string    `firestore:"subject"`     // OIDC sub claim
    Email       string    `firestore:"email,omitempty"`
    DisplayName string    `firestore:"displayName,omitempty"`
    LinkedAt    time.Time `firestore:"linkedAt"`
}
```

### Event（抽象基底）
```go
// Event は全イベントタイプの共通インターフェース
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

// EventType は対応するイベント種別
type EventType string

const (
    EventTypeEarthquake EventType = "earthquake"
    EventTypeTsunami    EventType = "tsunami"
    EventTypeWeather    EventType = "weather"    // 将来拡張
    EventTypeVolcano    EventType = "volcano"    // 将来拡張
)
```

### EventRecord（Datastore 保存用）
```go
type EventRecord struct {
    ID            string    `datastore:"-"`
    Type          string    `datastore:"type"`          // EventType
    Source        string    `datastore:"source"`        // データソース識別子
    Severity      int       `datastore:"severity"`      // 正規化された重大度
    AffectedAreas []string  `datastore:"affectedAreas"` // 影響地域
    OccurredAt    time.Time `datastore:"occurredAt"`
    ReceivedAt    time.Time `datastore:"receivedAt"`
    RawJSON       string    `datastore:"rawJson,noindex"`

    // イベント固有データ（JSON）
    Details       string    `datastore:"details,noindex"`
}
```

### EarthquakeDetails（地震固有データ）
```go
type EarthquakeDetails struct {
    MaxScale    int        `json:"maxScale"`    // 最大震度
    Hypocenter  Hypocenter `json:"hypocenter"`
    Tsunami     string     `json:"tsunami"`
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

### Source（データソース抽象化）
```go
// Source はデータソースの共通インターフェース
type Source interface {
    ID() string                          // "p2pquake" | "jma" | ...
    Connect(ctx context.Context) error
    Events() <-chan Event
    Close() error
}

// SourceConfig はソースごとの設定
type SourceConfig struct {
    Type     string            `yaml:"type"`     // "p2pquake" | "jma"
    Endpoint string            `yaml:"endpoint"`
    Options  map[string]string `yaml:"options"`
}
```

### Webhook
```go
type Webhook struct {
    ID          string    `datastore:"-"`
    UserID      string    `datastore:"userId"`
    URL         string    `datastore:"url"`
    Secret      string    `datastore:"secret"`
    Name        string    `datastore:"name"`
    Description string    `datastore:"description,omitempty"`
    Enabled     bool      `datastore:"enabled"`

    // 購読するイベントタイプ
    EventTypes  []string  `datastore:"eventTypes"` // ["earthquake", "tsunami"]

    // フィルタ設定（JSON）- イベントタイプごとに異なる条件
    Filters     string    `datastore:"filters,noindex"`

    // リトライ設定
    Retry       *RetryConfig `datastore:"retry,omitempty"`

    // 統計
    SuccessCount int64     `datastore:"successCount"`
    FailureCount int64     `datastore:"failureCount"`
    LastSentAt   time.Time `datastore:"lastSentAt,omitempty"`

    CreatedAt    time.Time `datastore:"createdAt"`
    UpdatedAt    time.Time `datastore:"updatedAt"`
}

type RetryConfig struct {
    Enabled    bool `datastore:"enabled"`
    MaxRetries int  `datastore:"maxRetries"`
    InitialMs  int  `datastore:"initialMs"`
}
```

### WebhookFilters（イベントタイプ別フィルタ）
```go
// WebhookFilters は Webhook.Filters に JSON として保存
type WebhookFilters struct {
    Earthquake *EarthquakeFilter `json:"earthquake,omitempty"`
    Tsunami    *TsunamiFilter    `json:"tsunami,omitempty"`
    Weather    *WeatherFilter    `json:"weather,omitempty"`
}

type EarthquakeFilter struct {
    MinSeverity  int      `json:"minSeverity"`  // 最小重大度 (0-100)
    MinScale     int      `json:"minScale"`     // 最小震度 (10-70)
    Prefectures  []string `json:"prefectures"`  // 対象都道府県
}

type TsunamiFilter struct {
    MinSeverity int      `json:"minSeverity"`
    Prefectures []string `json:"prefectures"`
}

type WeatherFilter struct {
    MinSeverity int      `json:"minSeverity"`
    Types       []string `json:"types"`  // ["rain", "storm", "snow"]
    Prefectures []string `json:"prefectures"`
}
```

### DeliveryLog（配信履歴）
```go
type DeliveryLog struct {
    ID           string    `datastore:"-"`
    WebhookID    string    `datastore:"webhookId"`
    EventID      string    `datastore:"eventId"`      // EventRecord.ID
    EventType    string    `datastore:"eventType"`
    Status       string    `datastore:"status"`
    StatusCode   int       `datastore:"statusCode"`
    RetryCount   int       `datastore:"retryCount"`
    ErrorMessage string    `datastore:"errorMessage,omitempty"`
    ResponseTime int64     `datastore:"responseTime"`
    CreatedAt    time.Time `datastore:"createdAt"`
}
```

### 震度 → Severity 変換
```go
// 震度を 0-100 の重大度に正規化
func ScaleToSeverity(scale int) int {
    // 10(震度1)=10, 20(震度2)=20, ..., 70(震度7)=100
    switch scale {
    case 10: return 10
    case 20: return 20
    case 30: return 30
    case 40: return 40
    case 45: return 50  // 5弱
    case 50: return 60  // 5強
    case 55: return 70  // 6弱
    case 60: return 80  // 6強
    case 70: return 100 // 7
    default: return 0
    }
}
```

---

## 参考資料

- [P2P地震情報 開発者向け](https://www.p2pquake.net/develop/)
- [P2P地震情報 JSON API v2 仕様](https://www.p2pquake.net/develop/json_api_v2/)
- [GitHub: epsp-specifications](https://github.com/p2pquake/epsp-specifications)
- [Pulumi GCP Provider](https://www.pulumi.com/registry/packages/gcp/)
