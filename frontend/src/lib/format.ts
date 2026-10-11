import type { QuestStatus } from "./types";

const dateTimeFormat = new Intl.DateTimeFormat("ja-JP", {
  timeZone: "Asia/Tokyo",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
});

/** UTC の ISO 文字列を日本時間で表示する。null は「—」。 */
export function formatDateTime(iso: string | null): string {
  if (!iso) return "—";
  return dateTimeFormat.format(new Date(iso));
}

/**
 * `<input type="datetime-local">` の値（日本時間とみなす）を UTC の ISO 文字列にする。
 * 例: "2026-10-12T12:00" → "2026-10-12T03:00:00.000Z"。API とは UTC でやり取りする（research R12）。
 */
export function jstLocalToISO(local: string): string {
  return new Date(`${local}:00+09:00`).toISOString();
}

export const statusLabels: Record<QuestStatus, string> = {
  open: "募集中",
  accepted: "受注済",
  completed: "完了",
};

export function formatReward(reward: number): string {
  return `${reward.toLocaleString("ja-JP")} G`;
}
