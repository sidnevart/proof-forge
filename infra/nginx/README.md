# Nginx

`nginx.prod.high-port.conf` — production-like конфиг для isolated ProofForge
stack на shared VPS.

Особенности:
- не использует host ports `80/443`
- слушает внутри контейнера `80/443`, а compose публикует их на `18080/18443`
- редиректит HTTP на HTTPS high-port
- проксирует `/v1/*`, `/healthz`, `/readyz` в API
- проксирует `/` в Next.js web app

Сертификаты не коммитятся. Файлы `cert.pem` и `key.pem` создаются на сервере в
`nginx/certs/`.
