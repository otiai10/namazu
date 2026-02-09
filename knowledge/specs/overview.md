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
| データベース | Google Firestore |
| 認証 | Google Identity Platform (OAuth) |
| ホスティング | GCP Compute Engine (Docker + Container-Optimized OS) |
| リバースプロキシ | Caddy (自動 HTTPS / Let's Encrypt) |
| 監視 | Cloud Logging / Monitoring |
| IaC | Pulumi (Go) |
| フロントエンド | React + Vite + TypeScript + TanStack Router |
| 課金 | Stripe |

## アーキテクチャ

### システム全体像

```
┌─────────────────────────────────────────────────────────┐
│                    Go Server (namazu)                   │
│  ┌─────────────────┐   ┌────────────────────────────┐  │
│  │   Static Files  │   │      REST API              │  │
│  │   (embed.FS)    │   │  /api/subscriptions        │  │
│  │   React SPA     │   │  /api/me                   │  │
│  │                 │   │  /api/billing/...          │  │
│  │   /index.html   │   │  /api/webhooks/stripe      │  │
│  └─────────────────┘   └────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
            │                         │
            │                         ▼
            │               ┌─────────────────┐
            │               │     Stripe      │
            │               │  (Checkout +    │
            │               │   Webhooks)     │
            │               └─────────────────┘
            ▼
┌─────────────────────────────────────────────────────────┐
│                    Caddy (HTTPS)                        │
│  - 自動 Let's Encrypt 証明書                             │
│  - stg.namazu.live / namazu.live                        │
└─────────────────────────────────────────────────────────┘
```

### ディレクトリ構成

```
namazu/
├── backend/
│   ├── cmd/namazu/
│   │   ├── main.go           # エントリーポイント
│   │   └── static/           # ビルド済みフロントエンド (embed)
│   └── internal/
│       ├── api/              # REST API ハンドラー
│       ├── app/              # アプリケーション制御
│       ├── auth/             # 認証ミドルウェア
│       ├── billing/          # Stripe 連携
│       ├── config/           # 設定管理
│       ├── delivery/webhook/ # Webhook 配信
│       ├── quota/            # クォータ管理
│       ├── source/           # データソース抽象化
│       ├── store/            # Firestore リポジトリ
│       ├── subscription/     # サブスクリプション管理
│       └── user/             # ユーザー管理
├── frontend/             # フロントエンド (React)
├── infra/                # Pulumi IaC
├── scripts/              # ユーティリティスクリプト
├── knowledge/            # ドキュメント
├── Dockerfile
├── Makefile
└── go.mod
```

## セキュリティ考慮事項

- [ ] Webhook URL の SSRF 対策（プライベート IP ブロック）
- [ ] シークレットの安全な保管
- [ ] HTTPS のみ許可
- [ ] レートリミット実装

## 参考資料

- [P2P地震情報 開発者向け](https://www.p2pquake.net/develop/)
- [P2P地震情報 JSON API v2 仕様](https://www.p2pquake.net/develop/json_api_v2/)
- [GitHub: epsp-specifications](https://github.com/p2pquake/epsp-specifications)
- [Pulumi GCP Provider](https://www.pulumi.com/registry/packages/gcp/)
