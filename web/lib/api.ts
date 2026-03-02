export type Product = {
  id: number;
  name: string;
  description: string;
  priceCents: number;
  stock: number;
  salePercent: number;
  saleStart?: string;
  saleEnd?: string;
  currentPriceCents: number;
  onSale: boolean;
};

export type CartItem = {
  productId: number;
  name: string;
  quantity: number;
  priceCents: number;
  discountPercent: number;
  currentPriceCents: number;
  subtotalPriceCents: number;
};

export type Cart = {
  userId: string;
  items: CartItem[];
  totalCents: number;
};

export type CheckoutResult = {
  orderId: number;
  userId: string;
  totalCents: number;
  status: string;
};

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options?.headers || {}),
    },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(await res.text());
  }

  return (await res.json()) as T;
}

export function listProducts() {
  return request<Product[]>("/products");
}

export function getCart(userId: string) {
  return request<Cart>(`/cart?userId=${encodeURIComponent(userId)}`);
}

export function addToCart(userId: string, productId: number, quantity: number) {
  return request<Cart>("/cart/items", {
    method: "POST",
    body: JSON.stringify({ userId, productId, quantity }),
  });
}

export function removeFromCart(userId: string, productId: number) {
  return request<Cart>("/cart/items", {
    method: "DELETE",
    body: JSON.stringify({ userId, productId }),
  });
}

export function checkout(userId: string) {
  return request<CheckoutResult>("/checkout", {
    method: "POST",
    body: JSON.stringify({ userId }),
  });
}

export function createProduct(payload: {
  name: string;
  description: string;
  priceCents: number;
  stock: number;
  salePercent: number;
  saleStart?: string;
  saleEnd?: string;
}) {
  return request<{ id: number }>("/admin/products", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function formatYen(cents: number) {
  return `JPY ${(cents / 100).toLocaleString("ja-JP", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`;
}
