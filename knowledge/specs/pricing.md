# 課金モデル

## プラン比較

| 機能 | Free | Pro (¥500/月) |
|------|------|---------------|
| **サブスクリプション数** | 1 | 12 |
| **配信先** | Webhook のみ | Webhook, Slack, Discord, LINE, Email |
| **フィルタ** | 基本（震度、地域） | 詳細（震源深さ、マグニチュード等） |
| **カスタムペイロード** | ✗ | ✓ |
| **配信履歴閲覧** | ✗ | ✓ |
| **課金サイクル** | - | 月額のみ |

## 配信先タイプ（DeliveryType）

| タイプ | Free | Pro | 説明 |
|--------|------|-----|------|
| `webhook` | ✓ | ✓ | 汎用 HTTP POST |
| `slack` | ✗ | ✓ | Slack Incoming Webhook |
| `discord` | ✗ | ✓ | Discord Webhook |
| `line` | ✗ | ✓ | LINE Notify |
| `email` | ✗ | ✓ | メール通知 |

## Pro 限定機能

### 1. 詳細フィルタ

- 震源深さ（MinDepth, MaxDepth）
- マグニチュード（MinMagnitude）
- 津波警報有無

### 2. カスタムペイロード

- テンプレートで Webhook ボディをカスタマイズ
- Mustache / Go template 形式

### 3. 配信履歴

- 過去30日分の配信ログ閲覧
- 成功/失敗、レスポンスタイム

## クォータ制限

```go
var (
    FreePlanLimits = PlanLimits{MaxSubscriptions: 1}
    ProPlanLimits  = PlanLimits{MaxSubscriptions: 12}
)
```

## プラン機能検証ロジック

```go
func validatePlanFeatures(plan string, sub *Subscription) error {
    if plan == "free" {
        // 配信先チェック
        if sub.Delivery.Type != "webhook" {
            return ErrProFeatureRequired
        }
        // カスタムペイロードチェック
        if sub.Delivery.Template != "" {
            return ErrProFeatureRequired
        }
        // 詳細フィルタチェック
        if sub.Filter != nil && hasAdvancedFilters(sub.Filter) {
            return ErrProFeatureRequired
        }
    }
    return nil
}
```

## Stripe 統合フロー

```
1. ユーザーが「Pro にアップグレード」クリック
   ↓
2. POST /api/billing/create-checkout-session
   - Stripe Checkout セッション作成
   - success_url, cancel_url 設定
   ↓
3. フロントエンドが Stripe Checkout にリダイレクト
   ↓
4. 支払い完了後、Stripe が Webhook 送信
   POST /api/webhooks/stripe
   - checkout.session.completed イベント
   - User.Plan を "pro" に更新
   - User.StripeCustomerID, SubscriptionID 保存
   ↓
5. ユーザーがサービスに戻る（success_url）
   - Pro 機能が有効化
```

## 環境変数

```bash
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_ID=price_...              # Pro プランの Price ID
STRIPE_SUCCESS_URL=https://namazu.live/billing/success
STRIPE_CANCEL_URL=https://namazu.live/billing
```
