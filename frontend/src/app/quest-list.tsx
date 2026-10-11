"use client";

// 一覧画面の本体。page.tsx から <Suspense> の中で使われる。
// useSearchParams（URL のクエリを読むフック）を使うコンポーネントは、Next.js 16 の
// cacheComponents では <Suspense> で囲む必要がある（囲まないとビルド時の事前描画でエラーになる）。

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { useActor } from "@/components/actor-context";
import { NativeSelect } from "@/components/native-select";
import { QuestStatusBadge } from "@/components/quest-status-badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ApiError, listQuests } from "@/lib/api";
import { formatDateTime, formatReward } from "@/lib/format";
import type { QuestPage, QuestStatus } from "@/lib/types";

const statuses: QuestStatus[] = ["open", "accepted", "completed"];

function parseStatus(v: string | null): QuestStatus | undefined {
  return statuses.find((s) => s === v);
}

function parsePage(v: string | null): number {
  const n = Number(v);
  return Number.isInteger(n) && n >= 1 ? n : 1;
}

/** 一覧（FR-006, FR-006a, FR-007b）。絞り込みとページは URL のクエリで保持する。 */
export function QuestList() {
  const { role } = useActor();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const status = parseStatus(searchParams.get("status"));
  const excludeExpired = searchParams.get("exclude_expired") === "true";
  const page = parsePage(searchParams.get("page"));

  const [data, setData] = useState<QuestPage | null>(null);
  const [error, setError] = useState<string | null>(null);

  // URL のクエリ（status / exclude_expired / page）が変わるたびに API を呼び直す。
  // cancelled フラグは、古いリクエストの結果が後から届いて新しい表示を上書きするのを防ぐため。
  useEffect(() => {
    let cancelled = false;
    listQuests({ status, excludeExpired, page })
      .then((d) => {
        if (!cancelled) {
          setData(d);
          setError(null);
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof ApiError ? e.message : "一覧を取得できませんでした");
      });
    return () => {
      cancelled = true;
    };
  }, [status, excludeExpired, page]);

  // 絞り込みやページ送りは state ではなく URL を書き換える。URL が変わると上の useEffect が再実行される。
  // URL に状態を持たせると、再読み込みやブラウザの「戻る」でも同じ表示に戻れる（FR-006a）。
  function navigate(next: { status?: QuestStatus; excludeExpired?: boolean; page?: number }) {
    const q = new URLSearchParams();
    if (next.status) q.set("status", next.status);
    if (next.excludeExpired) q.set("exclude_expired", "true");
    if (next.page && next.page > 1) q.set("page", String(next.page));
    const qs = q.toString();
    router.push(qs ? `${pathname}?${qs}` : pathname);
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-4">
        <label className="flex items-center gap-2 text-sm">
          状態
          <NativeSelect
            value={status ?? ""}
            onChange={(e) => navigate({ status: parseStatus(e.target.value), excludeExpired, page: 1 })}
          >
            <option value="">すべて</option>
            <option value="open">募集中</option>
            <option value="accepted">受注済</option>
            <option value="completed">完了</option>
          </NativeSelect>
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={excludeExpired}
            onChange={(e) => navigate({ status, excludeExpired: e.target.checked, page: 1 })}
          />
          期限切れを除く
        </label>
        {role === "guild" && (
          <Link href="/quests/new" className={buttonVariants({ className: "ml-auto" })}>
            クエストを登録
          </Link>
        )}
      </div>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {data && data.total_count === 0 && <p className="text-muted-foreground">クエストがありません</p>}

      {data && data.total_count > 0 && (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>タイトル</TableHead>
                <TableHead className="text-right">報酬</TableHead>
                <TableHead>期限</TableHead>
                <TableHead>状態</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.items.map((q) => (
                <TableRow key={q.id}>
                  <TableCell>
                    <Link href={`/quests/${q.id}`} className="font-medium underline-offset-4 hover:underline">
                      {q.title}
                    </Link>
                  </TableCell>
                  <TableCell className="text-right">{formatReward(q.reward)}</TableCell>
                  <TableCell>{formatDateTime(q.deadline)}</TableCell>
                  <TableCell>
                    <QuestStatusBadge quest={q} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {data.items.length === 0 && <p className="text-muted-foreground">このページにはクエストがありません</p>}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">
              全 {data.total_count} 件 ・ {data.page} / {data.total_pages} ページ
            </span>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => navigate({ status, excludeExpired, page: page - 1 })}
              >
                前へ
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= data.total_pages}
                onClick={() => navigate({ status, excludeExpired, page: page + 1 })}
              >
                次へ
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
