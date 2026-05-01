# CI/CD

## GitHub Actions

`.github/workflows/build-and-push.yml` — сборка Docker образов и пуш в GHCR при каждом push в `main`.

### Образы

| Сервис | Dockerfile | GHCR путь |
|--------|-----------|-----------|
| API | `backend/Dockerfile` (`TARGET=api`) | `ghcr.io/{OWNER}/proof-forge-api` |
| Worker | `backend/Dockerfile` (`TARGET=worker`) | `ghcr.io/{OWNER}/proof-forge-worker` |
| Web | `web/Dockerfile` | `ghcr.io/{OWNER}/proof-forge-web` |

### Теги

- `{git-sha}` — конкретный коммит
- `latest` — последний из `main`

### Ручной запуск

```bash
gh workflow run build-and-push.yml --ref main
```

## Деплой

См. `infra/docker/README.md` и `infra/scripts/server/`.
