import { StatePanel } from "@/components/core/state-panel";

export default function Loading() {
  return (
    <div className="page-shell" style={{ paddingTop: 24, paddingBottom: 64 }}>
      <StatePanel
        tone="loading"
        title="Подготавливаем интерфейс"
        description="Собираем рабочие блоки и актуальное состояние по целям."
      />
    </div>
  );
}
