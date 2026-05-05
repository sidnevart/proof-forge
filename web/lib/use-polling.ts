"use client";

import { useCallback, useEffect, useRef } from "react";

export function usePolling(
  callback: () => void | Promise<void>,
  intervalMs: number,
  { enabled = true, visibilityAware = true }: { enabled?: boolean; visibilityAware?: boolean } = {},
) {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const stop = useCallback(() => {
    if (timerRef.current !== null) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const start = useCallback(() => {
    stop();
    timerRef.current = setInterval(() => {
      void callbackRef.current();
    }, intervalMs);
  }, [intervalMs, stop]);

  useEffect(() => {
    if (!enabled) {
      stop();
      return;
    }

    start();

    if (!visibilityAware) {
      return stop;
    }

    const handleVisibility = () => {
      if (document.visibilityState === "visible") {
        void callbackRef.current();
        start();
      } else {
        stop();
      }
    };

    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      stop();
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [enabled, visibilityAware, start, stop]);
}
