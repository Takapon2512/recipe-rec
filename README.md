# recipe-recommend

## 環境構築

### 前提

- Docker Desktop (Docker Engine 24+ / Docker Compose v2)
- Go 1.26 系 (ローカルでビルド・補完を行う場合のみ)

ホストで使用するポート: `8080` (API) / `3307` (MySQL) / `8081` (phpMyAdmin)。

### 構成

| サービス     | コンテナ名         | ポート (host→container) | 用途                          |
| ------------ | ------------------ | ----------------------- | ----------------------------- |
| `api`        | `app-api`          | `8080 → 8080`           | Go + Gin (air でホットリロード) |
| `mysql`      | `app-mysql`        | `3307 → 3306`           | MySQL 8.0                     |
| `phpmyadmin` | `app-phpmyadmin`   | `8081 → 80`             | DB GUI                        |

### ディレクトリ

```
backend/
├── Dockerfile              # api コンテナ (golang:1.26-alpine + air)
├── docker-compose.yaml     # api / mysql / phpmyadmin
├── .air.toml               # ホットリロード設定 (entry: ./cmd/api)
├── .env                    # APP_PORT など
├── cmd/api/main.go         # Gin エントリポイント
├── go.mod / go.sum
└── docker/mysql/
    ├── initdb/             # 初回起動時に実行される .sql / .sh
    └── conf.d/             # my.cnf 上書き用
```

### 起動

```sh
cd backend
docker compose up -d --build
```

起動確認:

```sh
curl http://localhost:8080/health
# => {"status":"ok"}
```

その他:

- API ログ: `docker compose logs -f api`
- phpMyAdmin: <http://localhost:8081> (user: `root` / pass: `root_password`)
- MySQL 直接接続: `mysql -h 127.0.0.1 -P 3307 -u app -papp_password app_db`

### 停止 / クリーンアップ

```sh
docker compose down            # コンテナ停止
docker compose down -v         # ボリュームごと削除 (DB 初期化)
```

### 開発のヒント

- `backend/` 配下の `.go` ファイルを編集すると air が自動で再ビルドします。
- 依存追加は `docker compose exec api go get <pkg>` または ホスト側で `go get` 後にコンテナを再起動。
- DB の初期スキーマは `backend/docker/mysql/initdb/` に `.sql` を置くと初回起動時に流れます (既存ボリュームがあれば `down -v` 後に再起動)。

### 接続情報 (開発用)

`docker-compose.yaml` で API コンテナに渡している環境変数:

| 変数          | 値              |
| ------------- | --------------- |
| `DB_HOST`     | `mysql`         |
| `DB_PORT`     | `3306`          |
| `DB_USER`     | `app`           |
| `DB_PASSWORD` | `app_password`  |
| `DB_NAME`     | `app_db`        |
| `APP_PORT`    | `8080`          |
| `TZ`          | `Asia/Tokyo`    |
