"use client";

import { useEffect, useState } from "react";

import { listAdventurers } from "@/lib/api";
import type { Adventurer } from "@/lib/types";
import { type Role, useActor } from "./actor-context";
import { NativeSelect } from "./native-select";

/** ヘッダの役割・冒険者セレクタ。 */
export function ActorSelector() {
  const { role, adventurerId, setRole, setAdventurerId } = useActor();
  const [adventurers, setAdventurers] = useState<Adventurer[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listAdventurers()
      .then(setAdventurers)
      .catch((e: Error) => setError(e.message));
  }, []);

  return (
    <div className="flex flex-wrap items-center gap-2 text-sm">
      <label className="flex items-center gap-1">
        <span className="text-muted-foreground">役割</span>
        <NativeSelect value={role} onChange={(e) => setRole(e.target.value as Role)} aria-label="役割">
          <option value="guild">ギルド職員</option>
          <option value="adventurer">冒険者</option>
        </NativeSelect>
      </label>
      {role === "adventurer" && (
        <label className="flex items-center gap-1">
          <span className="text-muted-foreground">冒険者</span>
          <NativeSelect
            value={adventurerId ?? ""}
            onChange={(e) => setAdventurerId(e.target.value || null)}
            aria-label="操作する冒険者"
          >
            <option value="">選択してください</option>
            {adventurers.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </NativeSelect>
        </label>
      )}
      {error && <span className="text-destructive">{error}</span>}
    </div>
  );
}
