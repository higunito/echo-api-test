"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { useActor } from "@/components/actor-context";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ApiError, createQuest } from "@/lib/api";
import { jstLocalToISO } from "@/lib/format";

type FormState = {
  title: string;
  description: string;
  reward: string;
  deadline: string;
};

const initialForm: FormState = { title: "", description: "", reward: "", deadline: "" };

export default function NewQuestPage() {
  const { role } = useActor();
  const router = useRouter();
  const [form, setForm] = useState<FormState>(initialForm);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (role !== "guild") {
    return <p className="text-muted-foreground">クエストを登録するには、役割を「ギルド職員」にしてください。</p>;
  }

  const set = (key: keyof FormState) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm((f) => ({ ...f, [key]: e.target.value }));

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const reward = Number(form.reward);
    if (form.reward.trim() === "" || !Number.isInteger(reward)) {
      setError("報酬は 0 以上の整数で入力してください");
      return;
    }
    setSubmitting(true);
    try {
      const quest = await createQuest({
        title: form.title,
        description: form.description,
        reward,
        deadline: form.deadline ? jstLocalToISO(form.deadline) : null,
      });
      router.push(`/quests/${quest.id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "登録に失敗しました");
      setSubmitting(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>クエストを登録</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="title">タイトル（必須・100 文字以内）</Label>
            <Input id="title" value={form.title} onChange={set("title")} maxLength={100} required />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="description">説明（2000 文字以内）</Label>
            <Textarea id="description" value={form.description} onChange={set("description")} maxLength={2000} rows={4} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="reward">報酬（ゴールド）</Label>
            <Input id="reward" type="number" min={0} step={1} value={form.reward} onChange={set("reward")} required />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="deadline">期限（任意・日本時間）</Label>
            <Input id="deadline" type="datetime-local" value={form.deadline} onChange={set("deadline")} />
          </div>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <div>
            <Button type="submit" disabled={submitting}>
              {submitting ? "登録中…" : "登録する"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
