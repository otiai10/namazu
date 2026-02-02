# P2P地震情報 API 仕様

## 概要

P2P地震情報は、地震情報をリアルタイムで配信する WebSocket API を提供している。

## エンドポイント

| 環境 | URL |
|------|-----|
| 本番 | `wss://api.p2pquake.net/v2/ws` |
| サンドボックス | `wss://api-realtime-sandbox.p2pquake.net/v2/ws` |

## 接続仕様

- **強制切断**: 10分で強制切断されるため、再接続ロジックが必須
- **重複配信**: 同じイベントが複数回配信される場合があるため、`id` で重複排除が必要
- **対象イベント**: コード 551 (JMAQuake) をフィルタリング

## 震度コード

| コード | 震度 |
|--------|------|
| 10 | 震度1 |
| 20 | 震度2 |
| 30 | 震度3 |
| 40 | 震度4 |
| 45 | 震度5弱 |
| 50 | 震度5強 |
| 55 | 震度6弱 |
| 60 | 震度6強 |
| 70 | 震度7 |

## 実装のポイント

### 再接続ロジック

```go
// 10分タイマー + エラー時リトライ
for {
    err := client.Connect(ctx)
    if err != nil {
        log.Printf("Connection error: %v, reconnecting...", err)
        time.Sleep(5 * time.Second)
        continue
    }

    select {
    case <-ctx.Done():
        return
    case <-time.After(9 * time.Minute):
        // 10分切断前に自主的に再接続
        client.Close()
    }
}
```

### 重複排除

```go
var seenIDs = make(map[string]time.Time)
var mu sync.Mutex

func isDuplicate(id string) bool {
    mu.Lock()
    defer mu.Unlock()

    if _, exists := seenIDs[id]; exists {
        return true
    }

    seenIDs[id] = time.Now()

    // 古いエントリを削除（1時間以上前）
    for k, t := range seenIDs {
        if time.Since(t) > time.Hour {
            delete(seenIDs, k)
        }
    }

    return false
}
```

## 使用ライブラリ

- `github.com/gorilla/websocket` - WebSocket クライアント

## 参考資料

- [P2P地震情報 開発者向け](https://www.p2pquake.net/develop/)
- [P2P地震情報 JSON API v2 仕様](https://www.p2pquake.net/develop/json_api_v2/)
- [GitHub: epsp-specifications](https://github.com/p2pquake/epsp-specifications)
