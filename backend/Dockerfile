# ============================================================
# 開発用 Dockerfile (Go + air によるホットリロード)
# ローカル: go1.26.2 / コンテナ: 1.26-alpine (1.26系の最新パッチ)
# ============================================================
FROM golang:1.26-alpine

# 開発に必要なツール
#   git    : go get / go mod 用
#   curl   : ヘルスチェック・デバッグ用
#   tzdata : Asia/Tokyo タイムゾーン
#   make   : Makefile 利用想定
RUN apk add --no-cache git curl tzdata make \
    && cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime \
    && echo "Asia/Tokyo" > /etc/timezone

# ホットリロードツール (air)
# v1.62.0 は Go 1.26 系対応の安定版
RUN go install github.com/air-verse/air@v1.62.0

# 作業ディレクトリ
WORKDIR /app

# 依存解決を先に行ってキャッシュ効率を上げる
# (go.mod / go.sum がまだ無い初期段階ではスキップされる)
COPY go.mod go.su[m] ./
RUN go mod download || true

# ソースコードはボリュームマウントで同期するため、ここでは COPY しない
# (compose の volumes で /app に bind mount される想定)

# APIサーバのポート (内部)
EXPOSE 8080

# デフォルトは air によるホットリロード起動
# air の設定ファイル .air.toml はプロジェクトルートに配置する
CMD ["air", "-c", ".air.toml"]
