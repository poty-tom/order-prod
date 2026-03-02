"use client";

import { FormEvent, useState } from "react";
import { createProduct } from "../../lib/api";

type ProductForm = {
  name: string;
  description: string;
  priceCents: number;
  stock: number;
  salePercent: number;
  saleStart: string;
  saleEnd: string;
};

const initialForm: ProductForm = {
  name: "",
  description: "",
  priceCents: 10000,
  stock: 10,
  salePercent: 0,
  saleStart: "",
  saleEnd: "",
};

function toRfc3339(localDateTime: string): string {
  return new Date(localDateTime).toISOString();
}

export default function AdminPage() {
  const [form, setForm] = useState<ProductForm>(initialForm);
  const [error, setError] = useState("");
  const [result, setResult] = useState("");

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    try {
      setError("");
      setResult("");

      const payload: {
        name: string;
        description: string;
        priceCents: number;
        stock: number;
        salePercent: number;
        saleStart?: string;
        saleEnd?: string;
      } = {
        name: form.name,
        description: form.description,
        priceCents: Number(form.priceCents),
        stock: Number(form.stock),
        salePercent: Number(form.salePercent),
      };

      if (payload.salePercent > 0) {
        payload.saleStart = toRfc3339(form.saleStart);
        payload.saleEnd = toRfc3339(form.saleEnd);
      }

      const res = await createProduct(payload);
      setResult(`商品を作成しました (id=${res.id})`);
      setForm(initialForm);
    } catch (err) {
      setError(String(err));
    }
  }

  return (
    <section>
      <div className="card">
        <h2>管理画面</h2>
        <p>商品追加とセール期間設定を行います。</p>
      </div>

      <form className="card" onSubmit={submit}>
        <label>
          商品名
          <input
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
          />
        </label>

        <label>
          商品説明
          <textarea
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
          />
        </label>

        <label>
          価格 (cents)
          <input
            type="number"
            min={1}
            value={form.priceCents}
            onChange={(e) => setForm({ ...form, priceCents: Number(e.target.value) })}
            required
          />
        </label>

        <label>
          在庫
          <input
            type="number"
            min={0}
            value={form.stock}
            onChange={(e) => setForm({ ...form, stock: Number(e.target.value) })}
            required
          />
        </label>

        <label>
          セール割引率 (%)
          <input
            type="number"
            min={0}
            max={90}
            value={form.salePercent}
            onChange={(e) => setForm({ ...form, salePercent: Number(e.target.value) })}
          />
        </label>

        {form.salePercent > 0 ? (
          <>
            <label>
              セール開始日時
              <input
                type="datetime-local"
                value={form.saleStart}
                onChange={(e) => setForm({ ...form, saleStart: e.target.value })}
                required
              />
            </label>
            <label>
              セール終了日時
              <input
                type="datetime-local"
                value={form.saleEnd}
                onChange={(e) => setForm({ ...form, saleEnd: e.target.value })}
                required
              />
            </label>
          </>
        ) : null}

        <div className="row">
          <button type="submit">商品を追加</button>
        </div>

        {result ? <p>{result}</p> : null}
        {error ? <p className="error">{error}</p> : null}
      </form>
    </section>
  );
}
