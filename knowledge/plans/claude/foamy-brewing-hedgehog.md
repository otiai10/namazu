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
POST   /api/webhooks          # Webhook 登録
GET    /api/webhooks          # Webhook 一覧
GET    /api/webhooks/:id      # Webhook 詳細
PUT    /api/webhooks/:id      # Webhook 更新
DELETE /api/webhooks/:id      # Webhook 削除
GET    /api/earthquakes       # 地震履歴一覧
```

### Phase 3: Google OAuth 認証
**目標**: ユーザー認証とマルチテナント

#### 追加機能
- Google Identity Platform による認証
- ユーザーごとの Webhook 管理
- セッション管理

### Phase 4: フィルタ・リトライ・制限
**目標**: 本格運用に向けた機能拡張

#### 追加機能
- Webhook ごとのフィルタ条件（最小震度閾値など）
- 失敗時のリトライ（指数バックオフ、設定可能）
- ユーザーごとの制限（Webhook 数上限、レートリミット）

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
```go
type User struct {
    ID            string    `datastore:"-"`
    GoogleID      string    `datastore:"googleId"`
    Email         string    `datastore:"email"`
    DisplayName   string    `datastore:"displayName"`
    PictureURL    string    `datastore:"pictureUrl,omitempty"`
    Plan          string    `datastore:"plan"`     // "free" | "pro"
    WebhookLimit  int       `datastore:"webhookLimit"`
    CreatedAt     time.Time `datastore:"createdAt"`
    UpdatedAt     time.Time `datastore:"updatedAt"`
    LastLoginAt   time.Time `datastore:"lastLoginAt"`
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
