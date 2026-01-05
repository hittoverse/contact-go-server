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
