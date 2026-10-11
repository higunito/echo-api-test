"use client";

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
