export const STATUS_LABELS = {
  active: "Активно",
  approved: "Подтверждено",
  pending: "Ожидает",
  changes_requested: "Нужна доработка",
  rejected: "Отклонено",
} as const;

export type UIStatus = keyof typeof STATUS_LABELS;

export const TONE_LABELS = {
  loading: "Загрузка",
  error: "Ошибка",
  empty: "Пусто",
  success: "Готово",
  pending: "Нужно действие",
} as const;

export function formatDateLabel(value: string, options?: Intl.DateTimeFormatOptions): string {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
    ...options,
  }).format(new Date(value));
}
