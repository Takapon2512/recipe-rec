# ============================================================
# 開発用 Dockerfile (Next.js)
# 本番は Vercel ホスティング想定 (infrastructure.md 4.1)
# このファイルはローカル開発専用
# ============================================================
FROM node:24-alpine

# 開発に必要なツール
#   git    : 依存解決・Next.js が一部で使用
#   tzdata : Asia/Tokyo タイムゾーン
RUN apk add --no-cache git tzdata \
    && cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime \
    && echo "Asia/Tokyo" > /etc/timezone

WORKDIR /app

# 依存解決を先に行ってキャッシュ効率を上げる
# (package.json がまだ無い初期段階ではスキップされる)
COPY package.jso[n] package-lock.jso[n] pnpm-lock.yam[l] yarn.loc[k] ./
RUN if [ -f package-lock.json ]; then npm ci; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable && pnpm install --frozen-lockfile; \
    elif [ -f yarn.lock ]; then corepack enable && yarn install --frozen-lockfile; \
    else echo "no lockfile yet — skipping install"; fi

# ソースコードはボリュームマウントで同期するため、ここでは COPY しない
# (compose の volumes で /app に bind mount される想定)

# Next.js 開発サーバのポート
EXPOSE 3000

# デフォルトは Next.js 開発サーバ (HMR 有効)
# Docker 内からホストにバインドするため 0.0.0.0 を指定
CMD ["npm", "run", "dev", "--", "-H", "0.0.0.0", "-p", "3000"]
