"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Cart, formatYen, getCart, removeFromCart } from "../../lib/api";

const DEFAULT_USER = "user-1";

export default function CartPage() {
  const [userId, setUserId] = useState(DEFAULT_USER);
  const [cart, setCart] = useState<Cart | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const stored = window.localStorage.getItem("userId") || DEFAULT_USER;
    setUserId(stored);
    void load(stored);
  }, []);

  async function load(id: string) {
    try {
      setError("");
      setCart(await getCart(id));
    } catch (e) {
      setError(String(e));
    }
  }

  async function remove(productId: number) {
    try {
      setError("");
      setCart(await removeFromCart(userId, productId));
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <section>
      <div className="card">
        <h2>カート</h2>
        <p>ユーザーID: {userId}</p>
        <button onClick={() => void load(userId)}>再読込</button>
        {error ? <p className="error">{error}</p> : null}
      </div>

      <div className="card">
        {cart?.items.length ? (
          <>
            {cart.items.map((item) => (
              <div className="row" key={item.productId}>
                <strong>{item.name}</strong>
                <span>{item.quantity}個</span>
                <span>{formatYen(item.currentPriceCents)} / 個</span>
                <span>小計: {formatYen(item.subtotalPriceCents)}</span>
                <button className="danger" onClick={() => void remove(item.productId)}>
                  削除
                </button>
              </div>
            ))}
            <hr />
            <p>
              合計: <strong>{formatYen(cart.totalCents)}</strong>
            </p>
            <Link href="/checkout">購入ページへ</Link>
          </>
        ) : (
          <p>カートは空です。</p>
        )}
      </div>
    </section>
  );
}
