import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Simple Shopping Service",
  description: "Next.js + Go + MySQL + RabbitMQ sample",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ja">
      <body>
        <header className="header">
          <h1>Simple Shopping Service</h1>
          <nav>
            <a href="/">ユーザー画面</a>
            <a href="/cart">カート</a>
            <a href="/checkout">購入ページ</a>
            <a href="/admin">管理画面</a>
          </nav>
        </header>
        <main className="container">{children}</main>
      </body>
    </html>
  );
}
