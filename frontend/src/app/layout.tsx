import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import Link from "next/link";

import { ActorProvider } from "@/components/actor-context";
import { ActorSelector } from "@/components/actor-selector";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "冒険者ギルド",
  description: "クエストの登録・受注・完了報告",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="ja" className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}>
      <body className="flex min-h-full flex-col">
        <ActorProvider>
          <header className="border-b">
            <div className="mx-auto flex max-w-4xl flex-wrap items-center justify-between gap-3 px-4 py-3">
              <Link href="/" className="font-semibold">
                冒険者ギルド
              </Link>
              <ActorSelector />
            </div>
          </header>
          <main className="mx-auto w-full max-w-4xl flex-1 px-4 py-6">{children}</main>
        </ActorProvider>
      </body>
    </html>
  );
}
