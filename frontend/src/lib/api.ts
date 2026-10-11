import type { Adventurer, CreateQuestInput, ErrorResponse, Quest, QuestPage, QuestStatus } from "./types";

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

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
