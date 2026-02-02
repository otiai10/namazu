# Namazu Infrastructure

Pulumi を使用した namazu のインフラストラクチャ定義。

## 前提条件

- [Pulumi CLI](https://www.pulumi.com/docs/install/)
- [Go 1.24+](https://golang.org/dl/)
- [Google Cloud SDK](https://cloud.google.com/sdk/docs/install)
- GCP プロジェクト（Firestore, Compute Engine API 有効化済み）

## セットアップ

### 1. GCP 認証

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

### 2. Pulumi ログイン

```bash
# Pulumi Cloud を使用する場合
pulumi login

# ローカルファイルを使用する場合
pulumi login --local
```

### 3. スタックの作成

```bash
cd infra

# ステージング環境
pulumi stack init stg
pulumi config set gcp:project YOUR_STG_PROJECT_ID

# 本番環境
pulumi stack init prod
pulumi config set gcp:project YOUR_PROD_PROJECT_ID
pulumi config set namazu-infra:domain namazu.example.com
```

## デプロイ

### プレビュー

```bash
pulumi preview
```

### デプロイ実行

```bash
pulumi up
```

### 出力値の確認

```bash
pulumi stack output
```

## リソース構成

| リソース | 説明 |
|----------|------|
| Artifact Registry | Docker イメージリポジトリ |
| Firestore | データベース |
| Service Account | アプリケーション用サービスアカウント |
| VPC Network | プライベートネットワーク |
| Firewall Rules | HTTP/HTTPS, IAP SSH, ヘルスチェック |
| Static IP | 固定外部 IP アドレス |
| Cloud Router + NAT | アウトバウンド接続用 |
| Compute Engine | アプリケーションサーバー (e2-micro) |

## 環境変数

デプロイ後、インスタンスに設定される環境変数:

| 変数 | stg | prod |
|------|-----|------|
| NAMAZU_SOURCE_TYPE | p2pquake | p2pquake |
| NAMAZU_SOURCE_ENDPOINT | wss://api-realtime-sandbox.p2pquake.net/v2/ws | wss://api.p2pquake.net/v2/ws |
| NAMAZU_API_ADDR | :8080 | :8080 |
| NAMAZU_STORE_PROJECT_ID | (GCP Project ID) | (GCP Project ID) |

追加の環境変数（Firebase Auth, Stripe 等）は、インスタンスにSSHして設定:

```bash
# IAP経由でSSH
gcloud compute ssh namazu-stg-instance --zone=us-west1-b --tunnel-through-iap

# 環境変数を追加してコンテナを再起動
docker stop namazu
docker rm namazu
docker run -d \
  --name namazu \
  --restart=always \
  -p 8080:8080 \
  -e NAMAZU_SOURCE_TYPE=p2pquake \
  -e NAMAZU_SOURCE_ENDPOINT=wss://api.p2pquake.net/v2/ws \
  -e NAMAZU_API_ADDR=:8080 \
  -e NAMAZU_STORE_PROJECT_ID=YOUR_PROJECT \
  -e NAMAZU_AUTH_ENABLED=true \
  -e NAMAZU_AUTH_PROJECT_ID=YOUR_PROJECT \
  -e STRIPE_SECRET_KEY=sk_xxx \
  -e STRIPE_WEBHOOK_SECRET=whsec_xxx \
  -e STRIPE_PRICE_ID=price_xxx \
  us-west1-docker.pkg.dev/YOUR_PROJECT/namazu/namazu:latest
```

## GitHub Actions シークレット

CI/CD に必要なシークレット:

| シークレット | 説明 |
|-------------|------|
| GCP_CREDENTIALS | GCP サービスアカウント JSON |
| GCP_PROJECT_ID | GCP プロジェクト ID |
| PULUMI_ACCESS_TOKEN | Pulumi アクセストークン |

## トラブルシューティング

### インスタンスのログ確認

```bash
# シリアルポートログ
gcloud compute instances get-serial-port-output namazu-stg-instance --zone=us-west1-b

# コンテナログ（SSHしてから）
docker logs namazu
```

### インスタンスの再起動

```bash
gcloud compute instances reset namazu-stg-instance --zone=us-west1-b
```

### リソースの削除

```bash
pulumi destroy
```
