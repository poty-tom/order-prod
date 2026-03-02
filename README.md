# Simple Shopping Service (Next.js + TypeScript + Go + MySQL + RabbitMQ)

## 構成
- `web/`: Next.js(TypeScript) のフロントエンド
  - ユーザー画面: 商品一覧、カート、購入ページ
  - 管理画面: 商品追加 + セール期間/割引設定
- `backend/`: Go API
  - 商品作成/一覧
  - カート追加/取得/削除
  - チェックアウト（在庫競合チェック + 注文確定）
  - 注文確定イベントを RabbitMQ に publish
- `mysql`: 在庫、カート、注文を永続化
- `rabbitmq`: `orders.confirmed` キューで注文イベント受信

## 起動
```bash
docker compose up --build
```

## 画面
- ユーザー画面: `http://localhost:3000/`
- カート: `http://localhost:3000/cart`
- 購入ページ: `http://localhost:3000/checkout`
- 管理画面: `http://localhost:3000/admin`
- RabbitMQ管理画面: `http://localhost:15672` (`guest` / `guest`)

## 主要仕様の対応
- 管理画面で商品を追加すると出品される
- 商品ごとにセール期間 (`saleStart`/`saleEnd`) と割引率 (`salePercent`) を設定可能
- ユーザーは商品をカートに追加し、購入ページで確定するまで注文は未確定
- チェックアウト時にトランザクション内で在庫を再確認し、在庫不足なら `409 Conflict` を返して注文失敗
- 注文確定後は在庫減算、カート削除、RabbitMQへ注文イベント送信

## APIエンドポイント
- `GET /products`
- `POST /admin/products`
- `GET /cart?userId=...`
- `POST /cart/items`
- `DELETE /cart/items`
- `POST /checkout`

