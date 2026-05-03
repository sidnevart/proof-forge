"use client";

import { useEffect, useState } from "react";

export type AppMode = "all" | "owner" | "buddy";

const STORAGE_KEY = "proofforge.app_mode";

// useAppMode persists the user's preferred dashboard filter in localStorage
// so it survives reloads. Defaults to the provided fallback until hydration
// completes (avoids server/client mismatch).
export function useAppMode(fallback: AppMode = "all"): [AppMode, (next: AppMode) => void] {
  const [mode, setMode] = useState<AppMode>(fallback);

  useEffect(() => {
    try {
      const stored = window.localStorage.getItem(STORAGE_KEY);
      if (stored === "all" || stored === "owner" || stored === "buddy") {
        setMode(stored);
      }
    } catch {
      // localStorage unavailable (private mode, SSR) — keep fallback
    }
  }, []);

  function update(next: AppMode) {
    setMode(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // ignore
    }
  }

  return [mode, update];
}
