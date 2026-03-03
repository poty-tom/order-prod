# Web Service (Next.js / TypeScript)

## アーキテクチャ構成

Next.js App Router ベースの単一フロントエンドです。UIは以下の4画面で構成され、すべて `backend` API を呼び出します。

- `/` : ユーザー商品一覧
- `/cart` : カート確認・削除
- `/checkout` : 購入確認・確定
- `/admin` : 管理者の商品追加（セール設定含む）

主要構成:
- `app/` : 画面コンポーネント
- `lib/api.ts` : APIクライアントと型定義

データフロー:
1. 画面から `lib/api.ts` を呼び出し
2. `NEXT_PUBLIC_API_URL` 経由で Go API を実行
3. レスポンスを画面表示

## 技術スタック

- Next.js `14.2.16` (App Router)
- React `18.3.1`
- TypeScript `5.6.3`
- CSS: `app/globals.css`

## インタフェース定義

フロントの外部インタフェースは `backend` の HTTP API です。

### 利用API一覧
- `GET /products`
- `POST /admin/products`
- `GET /cart?userId=...`
- `POST /cart/items`
- `DELETE /cart/items`
- `POST /checkout`

### フロント内部型（`lib/api.ts`）

- `Product`
  - `id`, `name`, `description`, `priceCents`, `stock`, `salePercent`, `saleStart`, `saleEnd`, `currentPriceCents`, `onSale`
- `CartItem`
  - `productId`, `name`, `quantity`, `priceCents`, `discountPercent`, `currentPriceCents`, `subtotalPriceCents`
- `Cart`
  - `userId`, `items`, `totalCents`
- `CheckoutResult`
  - `orderId`, `userId`, `totalCents`, `status`

### 画面仕様

1. ユーザー画面 (`/`)
- 商品一覧を取得して表示
- 商品をカートに追加
- `localStorage.userId` を利用してユーザー切替

2. カート画面 (`/cart`)
- ユーザーのカートを表示
- 商品削除
- 購入ページへの導線

3. 購入ページ (`/checkout`)
- 現在カート内容の確認
- 購入確定 (`POST /checkout`)
- 在庫不足時は API エラー（`409`）表示

4. 管理画面 (`/admin`)
- 商品登録
- `salePercent > 0` の場合、セール開始/終了日時を入力
- `datetime-local` を RFC3339 に変換して送信

## 起動方法

### A. Docker Compose（推奨）
リポジトリルートで実行:

```bash
docker compose up --build web
```

アクセス:
- `http://localhost:3000`

### B. ローカル起動

```bash
cd web
npm install
npm run dev
```

環境変数:
- `NEXT_PUBLIC_API_URL` (default: `http://localhost:8080`)

## 開発手順

1. 依存インストール
```bash
cd web
npm install
```

2. 開発サーバー起動
```bash
npm run dev
```

3. 型・本番ビルド確認
```bash
npm run build
```

4. 画面確認
- 管理画面で商品作成
- ユーザー画面でカート追加
- 購入ページで確定
- 別ユーザーIDで同一商品を購入し、在庫競合ケースを再現

