"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { QuestStatusBadge } from "@/components/quest-status-badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ApiError, getQuest } from "@/lib/api";
import { formatDateTime, formatReward } from "@/lib/format";
import type { Quest } from "@/lib/types";
import { QuestActions } from "./quest-actions";

/** 詳細（FR-007）。受注・完了報告は QuestActions が担う。 */
export function QuestDetail() {
  const { id } = useParams<{ id: string }>();
  const [quest, setQuest] = useState<Quest | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    getQuest(id)
      .then((q) => {
        setQuest(q);
        setError(null);
      })
      .catch((e: unknown) => {
        if (e instanceof ApiError && (e.status === 404 || e.status === 400)) {
          setNotFound(true);
        } else {
          setError(e instanceof ApiError ? e.message : "クエストを取得できませんでした");
        }
      });
  }, [id]);

  useEffect(load, [load]);

  if (notFound) {
    return (
      <div className="flex flex-col gap-2">
        <p>クエストが見つかりません</p>
        <Link href="/" className="text-sm underline underline-offset-4">
          一覧へ戻る
        </Link>
      </div>
    );
  }
  if (error) {
    return (
      <p role="alert" className="text-destructive">
        {error}
      </p>
    );
  }
  if (!quest) {
    return <p className="text-muted-foreground">読み込み中…</p>;
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle className="text-lg">{quest.title}</CardTitle>
          <QuestStatusBadge quest={quest} />
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <p className="whitespace-pre-wrap">{quest.description || "（説明なし）"}</p>
        <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1 text-sm">
          <dt className="text-muted-foreground">報酬</dt>
          <dd>{formatReward(quest.reward)}</dd>
          <dt className="text-muted-foreground">期限</dt>
          <dd>{formatDateTime(quest.deadline)}</dd>
          <dt className="text-muted-foreground">受注者</dt>
          <dd>{quest.assignee?.name ?? "—"}</dd>
          <dt className="text-muted-foreground">受注日時</dt>
          <dd>{formatDateTime(quest.accepted_at)}</dd>
          <dt className="text-muted-foreground">完了日時</dt>
          <dd>{formatDateTime(quest.completed_at)}</dd>
          <dt className="text-muted-foreground">登録日時</dt>
          <dd>{formatDateTime(quest.created_at)}</dd>
        </dl>
        <QuestActions quest={quest} onChange={setQuest} />
        <Link href="/" className="text-sm underline underline-offset-4">
          一覧へ戻る
        </Link>
      </CardContent>
    </Card>
  );
}
