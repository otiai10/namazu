# ADR-001: User と Subscription の関係

## ステータス

採用 (2025-01-30)

## コンテキスト

Phase 3 で User 認証を導入するにあたり、User と Subscription のデータモデル関係を決定する必要がある。

### 検討した選択肢

**Option A: User has many Subscriptions (1:N)、Subscription has one Delivery (1:1)**

```
User (1) ───> Subscription (N)
                    │
                    ├──> Delivery (1)
                    └──> Filter (1)
```

**Option B: User has many Subscriptions (1:N)、Subscription has many Deliveries (1:N)**

```
User (1) ───> Subscription (N)
                    │
                    ├──> Deliveries (N)
                    └──> Filter (1)
```

## 決定

**Option A を採用する。**

## 理由

1. **フィルタの柔軟性**: ユーザーが配信先ごとに異なるフィルタ条件を設定したい場合がある
   - 本番サーバー: 震度5以上、東京都のみ
   - 開発サーバー: 全件通知（テスト用）
   - Slack: 震度6以上（重大な地震のみ）

2. **シンプルさ**: 1 Subscription = 1 Delivery の関係は理解しやすい

3. **個別管理**: 配信先ごとに有効/無効を切り替えやすい

4. **課金モデル**: Subscription 単位での課金が明確

## 結果

### データモデル

```go
type User struct {
    ID          string
    Email       string
    // ...
}

type Subscription struct {
    ID       string
    UserID   string         // User への外部キー
    Name     string
    Delivery DeliveryConfig // 1:1
    Filter   *FilterConfig  // 1:1 (optional)
}

type DeliveryConfig struct {
    Type   string // "webhook" | "slack" | "email"
    URL    string
    Secret string
}

type FilterConfig struct {
    MinScale    int
    Prefectures []string
}
```

### ユースケース例

```yaml
# User: 田中さん の Subscriptions
- name: "本番サーバー"
  delivery:
    type: webhook
    url: https://prod.example.com
  filter:
    min_scale: 50
    prefectures: ["東京都"]

- name: "開発サーバー"
  delivery:
    type: webhook
    url: https://dev.example.com
  # filter なし = 全件通知

- name: "Slack通知"
  delivery:
    type: slack
    url: https://hooks.slack.com/...
  filter:
    min_scale: 60
```

## 関連

- Phase 3: Google OAuth 認証
- Phase 4: フィルタ機能の本格実装
