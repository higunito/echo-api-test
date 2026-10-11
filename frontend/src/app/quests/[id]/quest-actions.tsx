"use client";

import { useState } from "react";

import { useActor } from "@/components/actor-context";
import { Button } from "@/components/ui/button";
import { ApiError, acceptQuest, completeQuest } from "@/lib/api";
import type { Quest } from "@/lib/types";

type Props = {
  quest: Quest;
  onChange: (q: Quest) => void;
};

/** 冒険者として操作しているときの受注（US3）・完了報告（US4）ボタン。 */
export function QuestActions({ quest, onChange }: Props) {
  const { role, adventurerId } = useActor();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (role !== "adventurer") return null;
  if (!adventurerId) {
    return (
      <p className="text-sm text-muted-foreground">受注・完了報告するには、ヘッダで操作する冒険者を選んでください。</p>
    );
  }

  const canAccept = quest.status === "open" && !quest.is_expired;
  const canComplete = quest.status === "accepted" && quest.assignee?.id === adventurerId;
  if (!canAccept && !canComplete) return null;

  async function run(action: (id: string, adventurerId: string) => Promise<Quest>) {
    setPending(true);
    setError(null);
    try {
      onChange(await action(quest.id, adventurerId!));
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "操作に失敗しました");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <div>
        {canAccept && (
          <Button onClick={() => run(acceptQuest)} disabled={pending}>
            {pending ? "処理中…" : "受注する"}
          </Button>
        )}
        {canComplete && (
          <Button onClick={() => run(completeQuest)} disabled={pending}>
            {pending ? "処理中…" : "完了報告する"}
          </Button>
        )}
      </div>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}
