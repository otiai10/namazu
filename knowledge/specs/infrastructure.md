# インフラストラクチャ

## GCP リソース

| リソース | 設定 |
|----------|------|
| リージョン | us-west1 (無料枠対象) |
| Compute Engine | e2-micro (無料枠) |
| OS | Container-Optimized OS (COS) |
| Firestore | Native mode |
| Container Registry | Artifact Registry |

## Pulumi 構成

```
infra/
├── main.go           # Pulumi エントリーポイント
├── Pulumi.yaml       # プロジェクト設定
├── Pulumi.stg.yaml   # ステージング環境
├── Pulumi.prod.yaml  # 本番環境
└── go.mod
```

### 環境設定

| 設定キー | stg | prod |
|----------|-----|------|
| `gcp:project` | namazu-live | namazu-live |
| `gcp:region` | us-west1 | us-west1 |
| `namazu-infra:environment` | stg | prod |
| `namazu-infra:machineType` | e2-micro | e2-micro |
| `namazu-infra:domain` | stg.namazu.live | namazu.live |

## デプロイフロー

### 初回セットアップ

```bash
# Pulumi スタック作成
cd infra
pulumi stack init stg
pulumi config set gcp:project namazu-live
pulumi config set gcp:region us-west1
pulumi config set namazu-infra:environment stg
pulumi config set namazu-infra:domain stg.namazu.live

# インフラ構築
pulumi up
```

### 日常のデプロイ

```bash
# ビルド、プッシュを一括実行
make ship

# または個別に
make build    # Docker イメージビルド (linux/amd64)
make push     # Artifact Registry にプッシュ
```

### インフラ変更時

```bash
# プレビュー
pulumi -C infra preview

# 適用
pulumi -C infra up
```

## メタデータベースの設定

GCE インスタンスのメタデータで設定を管理:

| メタデータキー | 説明 |
|----------------|------|
| `namazu-image` | Docker イメージ URL |
| `namazu-source-type` | データソースタイプ (p2pquake) |
| `namazu-source-endpoint` | WebSocket エンドポイント |
| `namazu-api-addr` | API リッスンアドレス |
| `namazu-store-project-id` | Firestore プロジェクト ID |
| `namazu-store-database` | Firestore データベース名 |
| `namazu-domain` | HTTPS ドメイン |

## HTTPS 構成

Caddy を使用した自動 HTTPS:

- Let's Encrypt による自動証明書取得
- HTTP → HTTPS 自動リダイレクト
- Docker ネットワークで namazu コンテナと通信

```bash
# 起動スクリプト内での Caddy 起動
docker run -d \
  --name caddy \
  --restart=always \
  --network namazu-net \
  -p 80:80 \
  -p 443:443 \
  -v /home/chronos/caddy_data:/data \
  caddy caddy reverse-proxy --from ${DOMAIN} --to namazu:8080
```

## IAM ロール

サービスアカウントに付与されるロール:

| ロール | 用途 |
|--------|------|
| `roles/artifactregistry.reader` | Docker イメージ取得 |
| `roles/datastore.user` | Firestore 読み書き |
| `roles/logging.logWriter` | Cloud Logging 書き込み |

## トラブルシューティング

### ログ確認

```bash
# シリアルポート出力
gcloud compute instances get-serial-port-output namazu-stg-instance --zone=us-west1-b

# コンテナログ（SSHしてから）
docker logs namazu
```

### SSH 接続

```bash
# IAP 経由で SSH
gcloud compute ssh namazu-stg-instance --zone=us-west1-b --tunnel-through-iap
```

### インスタンス再起動

```bash
gcloud compute instances reset namazu-stg-instance --zone=us-west1-b
```
