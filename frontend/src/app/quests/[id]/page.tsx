import { Suspense } from "react";

import { QuestDetail } from "./quest-detail";

export default function QuestDetailPage() {
  return (
    <Suspense fallback={<p className="text-muted-foreground">読み込み中…</p>}>
      <QuestDetail />
    </Suspense>
  );
}
