// バックエンド API を呼ぶ関数群。画面のコンポーネントはここを経由して API を使う。
//
// パスは "/api/..." の相対パス。ブラウザは Next.js（:3000）に送り、
// next.config.ts の rewrites がバックエンド（:8080）へ転送する。CORS の設定が要らなくなる（research R12）。
import type { Adventurer, CreateQuestInput, ErrorResponse, Quest, QuestPage, QuestStatus } from "./types";

/** API がエラーを返したときに throw する。code で分岐し、message をそのまま画面に出す。 */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

/**
 * fetch の共通処理。<T> はジェネリクスで、呼び出し側が期待するレスポンスの型を指定する。
 * 失敗時はバックエンドのエラー形式 {"error":{code,message}} を ApiError に変換して throw する。
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      ...init,
      headers: { "Content-Type": "application/json", ...init?.headers },
    });
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "サーバーに接続できませんでした");
  }
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ErrorResponse | null;
    throw new ApiError(
      res.status,
      body?.error.code ?? "UNKNOWN_ERROR",
      body?.error.message ?? `エラーが発生しました（${res.status}）`,
    );
  }
  return (await res.json()) as T;
}

export async function listAdventurers(): Promise<Adventurer[]> {
  const body = await request<{ items: Adventurer[] }>("/api/adventurers");
  return body.items;
}

export function createQuest(input: CreateQuestInput): Promise<Quest> {
  return request<Quest>("/api/quests", { method: "POST", body: JSON.stringify(input) });
}

export type ListQuestsParams = {
  status?: QuestStatus;
  excludeExpired?: boolean;
  page?: number;
};

export function listQuests(params: ListQuestsParams): Promise<QuestPage> {
  const q = new URLSearchParams();
  if (params.status) q.set("status", params.status);
  if (params.excludeExpired) q.set("exclude_expired", "true");
  if (params.page) q.set("page", String(params.page));
  const qs = q.toString();
  return request<QuestPage>(`/api/quests${qs ? `?${qs}` : ""}`);
}

export function getQuest(id: string): Promise<Quest> {
  return request<Quest>(`/api/quests/${encodeURIComponent(id)}`);
}

export function acceptQuest(id: string, adventurerId: string): Promise<Quest> {
  return request<Quest>(`/api/quests/${encodeURIComponent(id)}/accept`, {
    method: "POST",
    body: JSON.stringify({ adventurer_id: adventurerId }),
  });
}

export function completeQuest(id: string, adventurerId: string): Promise<Quest> {
  return request<Quest>(`/api/quests/${encodeURIComponent(id)}/complete`, {
    method: "POST",
    body: JSON.stringify({ adventurer_id: adventurerId }),
  });
}
