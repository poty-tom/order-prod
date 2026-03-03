# Backend Service (Go)

## アーキテクチャ構成

本サービスは Go の単一APIプロセスです。起動時に MySQL に接続し、必要なテーブルを自動作成します。

- Entry point: `cmd/server/main.go`
- 外部依存:
  - MySQL: 商品、カート、注文データの永続化
  - RabbitMQ: 注文確定イベント (`orders.confirmed`) の配信

リクエスト処理の流れ:
1. HTTP APIでリクエストを受ける
2. DBで商品・カート・注文を更新
3. チェックアウト成功時のみ RabbitMQ にイベントを publish

在庫整合性:
- `/checkout` は DB トランザクション (`Serializable`) で実行
- カート対象の商品行を `FOR UPDATE` でロックし、在庫不足時は `409 Conflict`

## 技術スタック

- Go `1.22`
- 標準ライブラリ (`net/http`)
- MySQL Driver: `github.com/go-sql-driver/mysql`
- RabbitMQ Client: `github.com/rabbitmq/amqp091-go`

## インタフェース定義

Base URL: `http://localhost:8080`

### 1. ヘルスチェック
- `GET /health`
- Response `200`

```json
{ "ok": true }
```

### 2. 商品一覧（ユーザー向け）
- `GET /products`
- Response `200`

```json
[
  {
    "id": 1,
    "name": "T-shirt",
    "description": "cotton",
    "priceCents": 250000,
    "stock": 10,
    "salePercent": 20,
    "saleStart": "2026-03-04T00:00:00Z",
    "saleEnd": "2026-03-05T00:00:00Z",
    "currentPriceCents": 200000,
    "onSale": true
  }
]
```

### 3. 商品登録（管理画面向け）
- `POST /admin/products`
- Request

```json
{
  "name": "T-shirt",
  "description": "cotton",
  "priceCents": 250000,
  "stock": 10,
  "salePercent": 20,
  "saleStart": "2026-03-04T00:00:00Z",
  "saleEnd": "2026-03-05T00:00:00Z"
}
```

- Response `201`

```json
{ "id": 1 }
```

備考:
- `salePercent > 0` の場合は `saleStart` / `saleEnd` 必須（RFC3339）

### 4. カート取得
- `GET /cart?userId={userId}`
- Response `200`

```json
{
  "userId": "user-1",
  "items": [
    {
      "productId": 1,
      "name": "T-shirt",
      "quantity": 2,
      "priceCents": 250000,
      "discountPercent": 20,
      "currentPriceCents": 200000,
      "subtotalPriceCents": 400000
    }
  ],
  "totalCents": 400000
}
```

### 5. カート追加
- `POST /cart/items`
- Request

```json
{
  "userId": "user-1",
  "productId": 1,
  "quantity": 1
}
```

- Response: 最新カート情報 (`200`)

### 6. カート削除
- `DELETE /cart/items`
- Request

```json
{
  "userId": "user-1",
  "productId": 1
}
```

- Response: 最新カート情報 (`200`)

### 7. 購入確定
- `POST /checkout`
- Request

```json
{ "userId": "user-1" }
```

- Response `200`

```json
{
  "orderId": 10,
  "userId": "user-1",
  "totalCents": 400000,
  "status": "CONFIRMED"
}
```

- Error
  - `400`: カート空など
  - `409`: 在庫不足（他ユーザー先行購入）

### RabbitMQ イベント
- Queue: `orders.confirmed`
- Payload

```json
{
  "orderId": 10,
  "userId": "user-1",
  "totalCents": 400000,
  "createdAt": "2026-03-04T01:23:45Z"
}
```

## 起動方法

### A. Docker Compose（推奨）
リポジトリルートで実行:

```bash
docker compose up --build api
```

### B. ローカル起動
前提:
- MySQL と RabbitMQ が起動済み

```bash
cd backend
go mod tidy
go run ./cmd/server
```

環境変数（省略時はデフォルト値を使用）:

- `PORT` (default: `8080`)
- `MYSQL_DSN` (default: `root:root@tcp(mysql:3306)/shop?parseTime=true`)
- `RABBITMQ_URL` (default: `amqp://guest:guest@rabbitmq:5672/`)

## 開発手順

1. 依存同期
```bash
cd backend
go mod tidy
```

2. フォーマット
```bash
gofmt -w ./cmd/server/main.go
```

3. ビルド確認
```bash
go build ./...
```

4. 動作確認（例）
```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/products
```

5. 競合在庫シナリオ確認
- 別ユーザーで同一商品をカートに追加
- 片方を先に `/checkout`
- 後続 `/checkout` が `409` になることを確認

