import { Badge } from "@/components/ui/badge";
import { statusLabels } from "@/lib/format";
import type { Quest } from "@/lib/types";

/** 状態バッジ。募集中で期限切れのときは「期限切れ」も並べる（FR-007a）。 */
export function QuestStatusBadge({ quest }: { quest: Pick<Quest, "status" | "is_expired"> }) {
  return (
    <span className="inline-flex gap-1">
      <Badge variant={quest.status === "open" ? "default" : quest.status === "accepted" ? "secondary" : "outline"}>
        {statusLabels[quest.status]}
      </Badge>
      {quest.is_expired && <Badge variant="destructive">期限切れ</Badge>}
    </span>
  );
}
