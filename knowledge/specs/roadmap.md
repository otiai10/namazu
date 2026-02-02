# 開発ロードマップ

## Phase 概要

| Phase | 目標 | 状態 |
|-------|------|------|
| Phase 1 | CLI ツール（コア機能） | ✅ 完了 |
| Phase 2 | REST API + Firestore | ✅ 完了 |
| Phase 3 | Google Identity Platform 認証 | ✅ 完了 |
| Phase 4 | フィルタ・リトライ・制限 | 🔄 進行中 |
| Phase 5 | Web UI・課金モデル | 🔄 進行中 |

---

## Phase 1: CLI ツール（コア機能）

**目標**: 動く最小限のプロトタイプ

### 機能
- [x] WebSocket で P2P地震情報 API に接続
- [x] 10分ごとの再接続ロジック
- [x] 設定ファイル (YAML) から Webhook URL を読み込み
- [x] 地震情報受信時に Webhook へ POST
- [x] HMAC-SHA256 署名付き

---

## Phase 2: REST API + Firestore

**目標**: Webhook の動的管理

### 機能
- [x] REST API で Subscription CRUD
- [x] Google Firestore に Subscription 永続化
- [x] 地震情報履歴の保存

---

## Phase 3: Google Identity Platform 認証

**目標**: ユーザー認証とマルチテナント

### 設計決定
- **ADR-001**: User (1) → Subscription (N) の関係を採用
- **Provider 埋め込み**: LinkedProvider を User ドキュメント内に配列として埋め込み
- **Account Linking**: 1ユーザーが複数の Identity Provider をリンク可能

### 機能
- [x] Firebase Auth による JWT 検証
- [x] ユーザープロファイル管理 (`/api/me`)
- [x] Subscription の所有権チェック
- [x] Account Linking 対応

---

## Phase 4: フィルタ・リトライ・制限

**目標**: 本格運用に向けた機能拡張

### Phase 4.1: フィルタ適用
- [ ] `FilterConfig.Matches()` 実装
- [ ] 震度フィルタ（MinScale）
- [ ] 地域フィルタ（Prefectures）

### Phase 4.2: Webhook リトライ
- [x] 指数バックオフによるリトライ
- [x] リトライ可能エラーの判定
- [x] `RetryingSender` デコレータ

### Phase 4.3: サブスクリプション数制限
- [x] クォータチェッカー実装
- [x] プランごとの制限値（Free: 1, Pro: 12）
- [x] API での制限チェック

---

## Phase 5: Web UI・課金モデル

**目標**: 一般公開サービス化

### 技術決定
| 項目 | 選択 |
|------|------|
| ホスティング | SPA + 同一サーバー（embed.FS） |
| フロントエンド | React + Vite + TypeScript + TanStack Router |
| 課金 | Stripe |

### 機能
- [x] React SPA 基盤構築
- [ ] Subscription 管理 UI
- [ ] Stripe Checkout 連携
- [ ] カスタマーポータル
- [ ] ダッシュボード（配信履歴、統計）

### ページ構成
| ページ | パス | 説明 |
|--------|------|------|
| ダッシュボード | `/` | 概要、最近のイベント |
| Subscriptions | `/subscriptions` | Webhook 一覧・作成・編集・削除 |
| Billing | `/billing` | プラン管理、Stripe Checkout |
| Settings | `/settings` | プロファイル、連携プロバイダー |

---

## 今後の拡張案

- 複数データソース対応（気象庁 API など）
- Push 通知（FCM）
- Slack/Discord ボット
- 公開 API（サードパーティ向け）
