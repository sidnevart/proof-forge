import { StatePanel } from "@/components/core/state-panel";

export default function Loading() {
  return (
    <div className="page-shell" style={{ paddingTop: 24, paddingBottom: 64 }}>
      <StatePanel
        tone="loading"
        title="Loading surfaces"
        description="Frontend shell собирает status layers и proof surfaces before paint."
      />
    </div>
  );
}
