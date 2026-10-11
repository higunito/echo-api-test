"use client";

// "use client" はこのファイルがブラウザで動く「クライアントコンポーネント」であることを示す。
// useState / useEffect / localStorage はブラウザでしか使えないので、この宣言が必要。
// 何も書かないファイルは、サーバーで描画される「サーバーコンポーネント」になる（Next.js App Router の既定）。

import { createContext, useCallback, useContext, useEffect, useState } from "react";

export type Role = "guild" | "adventurer";

type Actor = {
  role: Role;
  adventurerId: string | null;
};

type ActorContextValue = Actor & {
  setRole: (role: Role) => void;
  setAdventurerId: (id: string | null) => void;
};

const STORAGE_KEY = "guild.actor";
const defaultActor: Actor = { role: "guild", adventurerId: null };

// React Context は、離れたコンポーネント間で値を共有する仕組み。
// layout.tsx で <ActorProvider> で全体を囲み、どの画面からも useActor() で役割と冒険者を読める。
const ActorContext = createContext<ActorContextValue | null>(null);

function load(): Actor {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultActor;
    const v = JSON.parse(raw) as Partial<Actor>;
    return {
      role: v.role === "adventurer" ? "adventurer" : "guild",
      adventurerId: typeof v.adventurerId === "string" ? v.adventurerId : null,
    };
  } catch {
    return defaultActor;
  }
}

function save(actor: Actor) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(actor));
  } catch {
    // 保存できなくても画面上の選択は有効なままにする
  }
}

/** 画面上で選んだ役割と冒険者を保持する（認証なし。spec Clarifications Q1）。 */
export function ActorProvider({ children }: { children: React.ReactNode }) {
  const [actor, setActor] = useState<Actor>(defaultActor);

  useEffect(() => {
    // localStorage はブラウザでしか読めないため、マウント後に復元する
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setActor(load());
  }, []);

  const update = useCallback((patch: Partial<Actor>) => {
    setActor((prev) => {
      const next = { ...prev, ...patch };
      save(next);
      return next;
    });
  }, []);

  const setRole = useCallback((role: Role) => update({ role }), [update]);
  const setAdventurerId = useCallback((adventurerId: string | null) => update({ adventurerId }), [update]);

  return (
    <ActorContext.Provider value={{ ...actor, setRole, setAdventurerId }}>{children}</ActorContext.Provider>
  );
}

export function useActor(): ActorContextValue {
  const ctx = useContext(ActorContext);
  if (!ctx) throw new Error("useActor must be used within ActorProvider");
  return ctx;
}
