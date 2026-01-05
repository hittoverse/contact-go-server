# Contact Server

TCP server for `nc contact.hitto.me 1337`

## Run locally

```bash
go run main.go
```

## Test

```bash
nc contact.hitto.me 1337
```

## Docker

```bash
docker build -t contact-server .
docker run -p 1337:1337 contact-server
```

## Cloudflare Tunnel

```bash
cloudflared tunnel route tcp 1337 contact.hitto.me:1337
```

---

## セキュリティ機能

本サーバーは外部公開を想定し、以下のセキュリティ対策を実装しています：

| 対策 | 設定値 | 説明 |
|------|--------|------|
| 同時接続数制限 | 100接続 | サーバーリソース枯渇を防止 |
| 入力サイズ制限 | 1KB/行 | 大量データ送信による攻撃を防止 |
| 接続タイムアウト | 1分 | 長時間占有を防止 |
| 読み込みタイムアウト | 30秒 | 無応答接続を切断 |
| IP単位レート制限 | 5接続/10秒 | DoS攻撃を緩和 |
| Graceful Shutdown | SIGINT/SIGTERM | アクティブ接続を待機して安全に終了 |

### 設定値の変更

`main.go` の定数を編集して調整可能です：

```go
const (
    maxConnections   = 100               // 最大同時接続数
    maxInputSize     = 1024              // 入力読み込み上限 (1KB)
    connTimeout      = 1 * time.Minute   // 接続タイムアウト
    readTimeout      = 30 * time.Second  // 読み込みタイムアウト
    rateLimitWindow  = 10 * time.Second  // レート制限の時間枠
    rateLimitMax     = 5                 // 時間枠内の最大接続数
)
```

---

## セットアップ手順

```bash
# 1. ディレクトリ作成
mkdir contact-server && cd contact-server

# 2. 上のファイルを作成

# 3. 動作確認
go run main.go

# 4. 別ターミナルでテスト
nc localhost 1337
```
