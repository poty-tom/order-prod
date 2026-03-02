"use client";

import { useEffect, useState } from "react";
import { checkout, CheckoutResult, formatYen, getCart, Cart } from "../../lib/api";

const DEFAULT_USER = "user-1";

export default function CheckoutPage() {
  const [userId, setUserId] = useState(DEFAULT_USER);
  const [cart, setCart] = useState<Cart | null>(null);
  const [result, setResult] = useState<CheckoutResult | null>(null);
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

  async function confirm() {
    try {
      setError("");
      setResult(await checkout(userId));
      await load(userId);
    } catch (e) {
      setError(String(e));
      setResult(null);
    }
  }

  return (
    <section>
      <div className="card">
        <h2>購入ページ</h2>
        <p>ユーザーID: {userId}</p>
        <p className="small">ここで確定するまで注文は成立しません。</p>
      </div>

      <div className="card">
        <h3>注文確認</h3>
        {cart?.items.length ? (
          <>
            {cart.items.map((item) => (
              <p key={item.productId}>
                {item.name} x {item.quantity} = {formatYen(item.subtotalPriceCents)}
              </p>
            ))}
            <p>
              合計: <strong>{formatYen(cart.totalCents)}</strong>
            </p>
            <button onClick={() => void confirm()}>購入確定</button>
          </>
        ) : (
          <p>カートは空です。</p>
        )}

        {error ? <p className="error">{error}</p> : null}

        {result ? (
          <p>
            注文ID {result.orderId} / 金額 {formatYen(result.totalCents)} / 状態 {result.status}
          </p>
        ) : null}
      </div>
    </section>
  );
}
