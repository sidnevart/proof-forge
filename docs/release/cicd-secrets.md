# CI/CD секреты ProofForge

## GitHub Actions secrets
Нужно добавить в GitHub repository secrets:

- `DEPLOY_HOST`
- `DEPLOY_PORT`
- `DEPLOY_USER`
- `DEPLOY_PATH`
- `DEPLOY_SSH_PRIVATE_KEY`

## Рекомендуемые значения
- `DEPLOY_HOST=80.74.25.43`
- `DEPLOY_PORT=22`
- `DEPLOY_USER=proofforge-deploy`
- `DEPLOY_PATH=/opt/proofforge-prod`

## Что не нужно класть в GitHub secrets
Server runtime secrets лучше хранить только в серверном:

`/opt/proofforge-prod/.env.prod`

Туда входят:
- database password
- MinIO credentials
- SMTP credentials
- OpenAI key
- optional Telegram credentials

## GHCR
Для первой итерации отдельный `GHCR_TOKEN` не обязателен.

Pipeline использует:
- `${{ github.actor }}`
- `${{ secrets.GITHUB_TOKEN }}`

Этого достаточно, чтобы:
- push-ить образы в `ghcr.io`
- временно логинить сервер на pull во время deploy workflow
