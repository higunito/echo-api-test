import { Suspense } from "react";

import { QuestList } from "./quest-list";

export default function Home() {
  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-xl font-semibold">クエスト一覧</h1>
      <Suspense fallback={<p className="text-muted-foreground">読み込み中…</p>}>
        <QuestList />
      </Suspense>
    </div>
  );
}
