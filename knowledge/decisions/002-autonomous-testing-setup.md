# ADR-002: 自律的テスト環境の構築

> **IMPORTANT: このドキュメントは常に最新の状態に保つこと**
>
> このドキュメントは Claude Code が自律的にテストを実行するための基盤を記述しています。
> テスト環境やスクリプトに変更があった場合は、必ずこのドキュメントも更新してください。
>
> - テストケースの追加・変更 → 「テストケース」セクションを更新
> - 新しいフラグやオプション → 「使用方法」セクションを更新
> - 関連ファイルの追加 → 「関連ファイル」セクションを更新

## ステータス

採用（Accepted）

## コンテキスト

Claude Code が自律的に開発・テスト・動作確認を行うためには、以下の障壁があった：

1. **認証トークンの必要性**: API テストのたびにユーザーの手が必要
2. **実 Firestore への永続化**: テストデータが溜まり、クリーンアップが面倒
3. **サービスアカウント認証**: 認証情報がないと動かない

## 決定

以下の機能を実装し、Claude Code が自律的にテストできる環境を整備する：

### 1. `--test-mode` フラグ

```bash
go run ./cmd/namazu/ --test-mode
```

- 認証を無効化して起動
- 警告ログを出力（本番使用禁止の明示）
- API エンドポイントに認証なしでアクセス可能

### 2. Firebase Emulator 自動検出

```go
// FIRESTORE_EMULATOR_HOST があればエミュレータを使用
emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
if emulatorHost != "" {
    log.Printf("🔧 Using Firestore Emulator at %s", emulatorHost)
}
```

- Go SDK が自動的にエミュレータに接続
- 認証情報不要
- ローカルで完結、状態リセット容易

### 3. E2E テストスクリプト

```bash
./scripts/e2e-test.sh
```

- 自動モード検出（Emulator 優先、なければ Real Firestore）
- サーバー起動 → API テスト → クリーンアップを自動化
- 8つのテストケースを実行

## テストケース

| テスト | 説明 |
|--------|------|
| Health check | `/health` が `ok` を返す |
| Create subscription (no filter) | フィルタなしサブスクリプション作成 |
| Create subscription (MinScale) | MinScale フィルタ付き作成 |
| Create subscription (Prefecture) | Prefecture フィルタ付き作成 |
| List subscriptions | 一覧取得 |
| Get subscription by ID | ID指定で取得 |
| Update subscription | 更新 |
| Delete subscription | 削除 |

## 使用方法

```bash
# 自動検出（Emulator 優先）
./scripts/e2e-test.sh

# 強制的に Real Firestore 使用
./scripts/e2e-test.sh real

# 強制的に Emulator 使用（要 firebase CLI）
./scripts/e2e-test.sh emulator
```

### Firebase Emulator のセットアップ（推奨）

```bash
# Firebase CLI インストール
npm install -g firebase-tools

# Emulator 付きでテスト
./scripts/e2e-test.sh emulator
```

## 結果

Claude Code が以下を自律的に実行可能になった：

1. サーバー起動（`--test-mode`）
2. API テスト実行
3. 動作確認
4. クリーンアップ

ユーザーの介入なしで開発・テストサイクルを回せる。

## Hook 統合

`.claude/settings.json` にて、`git commit` 前に自動実行されるよう設定済み：

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash(git commit*)",
        "hooks": [
          { "type": "command", "command": "./scripts/test.sh" },
          { "type": "command", "command": "./scripts/e2e-test.sh real" }
        ]
      }
    ]
  }
}
```

これにより、Claude Code がコミットを作成する前に：
1. ユニットテスト (`test.sh`)
2. E2E テスト (`e2e-test.sh`)

が自動実行され、品質を担保する。

## 関連ファイル

- `cmd/namazu/main.go` - `--test-mode` フラグ追加
- `internal/store/firestore.go` - Emulator 検出ログ追加
- `scripts/e2e-test.sh` - E2E テストスクリプト
- `cmd/dummysubscriber/main.go` - PORT 環境変数対応
- `.claude/settings.json` - Hook 設定
