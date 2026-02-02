# REST API 仕様

## エンドポイント一覧

### Public（認証不要）

| メソッド | パス | 説明 |
|----------|------|------|
| GET | `/health` | ヘルスチェック |
| GET | `/api/events` | 地震履歴一覧 |

### Protected（認証必須）

| メソッド | パス | 説明 |
|----------|------|------|
| GET | `/api/me` | 現在のユーザープロファイル |
| PUT | `/api/me` | プロファイル更新 |
| GET | `/api/me/providers` | リンク済み認証プロバイダー一覧 |
| POST | `/api/subscriptions` | Subscription 作成 |
| GET | `/api/subscriptions` | 自分の Subscription 一覧 |
| GET | `/api/subscriptions/:id` | Subscription 詳細 |
| PUT | `/api/subscriptions/:id` | Subscription 更新 |
| DELETE | `/api/subscriptions/:id` | Subscription 削除 |

### Billing API（認証必須）

| メソッド | パス | 説明 |
|----------|------|------|
| POST | `/api/billing/create-checkout-session` | Stripe Checkout セッション作成 |
| GET | `/api/billing/portal-session` | カスタマーポータルセッション取得 |
| GET | `/api/billing/status` | 現在のプラン状態取得 |

### Webhook（署名検証）

| メソッド | パス | 説明 |
|----------|------|------|
| POST | `/api/webhooks/stripe` | Stripe イベント受信 |

## API パス設計方針

- **ユーザー向け API**: `/api/...` - 一般ユーザーがアクセス
- **Admin API（将来）**: `/admin-api/...` - 管理者専用エンドポイント

## 認証

### Firebase Authentication

- Bearer トークンを `Authorization` ヘッダーで送信
- Firebase Admin SDK で JWT を検証

```
Authorization: Bearer <Firebase ID Token>
```

### テストモード

`--test-mode` フラグで認証をバイパス（E2E テスト用）

## Webhook 署名

配信される Webhook には HMAC-SHA256 署名が付与される:

```
X-Signature-256: sha256=<HMAC-SHA256 of body>
```

検証方法:
```go
mac := hmac.New(sha256.New, []byte(secret))
mac.Write(body)
expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
```

## 環境変数

```bash
# 認証
NAMAZU_AUTH_ENABLED=true
NAMAZU_AUTH_PROJECT_ID=namazu-live
NAMAZU_AUTH_CREDENTIALS=path/to/serviceaccount.json  # ローカル開発のみ

# Stripe
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_ID=price_...
STRIPE_SUCCESS_URL=https://namazu.live/billing/success
STRIPE_CANCEL_URL=https://namazu.live/billing
```
