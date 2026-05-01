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
        title="Shell error"
        description="Something broke in the frontend shell. Reset and inspect the current surface state."
        meta={<Button onClick={reset}>Retry shell</Button>}
      />
    </div>
  );
}
