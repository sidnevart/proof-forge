"use client";

import { Button } from "@/components/core/button";
import { StatePanel } from "@/components/core/state-panel";

export default function Error({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="page-shell" style={{ paddingTop: 24, paddingBottom: 64 }}>
      <StatePanel
        tone="error"
        title="Ошибка интерфейса"
        description="Не удалось собрать текущий экран. Перезапустите интерфейс и повторите действие."
        meta={<Button onClick={reset}>Перезапустить экран</Button>}
      />
    </div>
  );
}
