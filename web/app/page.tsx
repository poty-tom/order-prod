"use client";

import { useEffect, useState } from "react";
import { addToCart, formatYen, listProducts, Product } from "../lib/api";

const DEFAULT_USER = "user-1";

export default function HomePage() {
  const [userId, setUserId] = useState(DEFAULT_USER);
  const [products, setProducts] = useState<Product[]>([]);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    const stored = window.localStorage.getItem("userId");
    if (stored) setUserId(stored);
    void loadProducts();
  }, []);

  async function loadProducts() {
    try {
      setError("");
      setProducts(await listProducts());
    } catch (e) {
      setError(String(e));
    }
  }

  function updateUser(next: string) {
    setUserId(next);
    window.localStorage.setItem("userId", next);
  }

  async function onAdd(productId: number) {
    try {
      setError("");
      setMessage("");
      await addToCart(userId, productId, 1);
      setMessage("カートに追加しました");
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <section>
      <div className="card">
        <h2>ユーザー画面</h2>
        <label>
          ユーザーID
          <input value={userId} onChange={(e) => updateUser(e.target.value)} />
        </label>
        <p className="small">別のユーザーIDを入れると、別ユーザーとして購入フローを試せます。</p>
        {message ? <p>{message}</p> : null}
        {error ? <p className="error">{error}</p> : null}
      </div>

      <div className="grid products">
        {products.map((p) => (
          <article className="card" key={p.id}>
            <h3>{p.name}</h3>
            <p>{p.description || "説明なし"}</p>
            <p>在庫: {p.stock}</p>
            <p>
              価格: {formatYen(p.currentPriceCents)}
              {p.onSale ? ` (通常 ${formatYen(p.priceCents)} / ${p.salePercent}%OFF)` : ""}
            </p>
            <button disabled={p.stock <= 0} onClick={() => void onAdd(p.id)}>
              カートに入れる
            </button>
          </article>
        ))}
      </div>
    </section>
  );
}
