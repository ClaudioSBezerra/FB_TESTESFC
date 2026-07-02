---
phase: quick-260702-k3u
plan: 01
subsystem: deploy
tags: [coolify, docker-compose, github-actions, ci-cd, production]
dependency-graph:
  requires: []
  provides: [docker-compose.prod.yml, deploy-production-workflow]
  affects: [production-deployment]
tech-stack:
  added: []
  patterns: ["Traefik labels for Coolify-managed reverse proxy", "network isolation via dedicated bridge network + external coolify network"]
key-files:
  created:
    - docker-compose.prod.yml
    - .github/workflows/deploy-production.yml
  modified: []
decisions:
  - "db and api isolated to fb_net only; web is the sole service bridging fb_net and the external coolify network"
  - "COOKIE_SECURE fixed to true in production (not env-overridable) since HTTPS is mandatory"
  - "No ports: blocks on any service in production — Traefik routes via the coolify network, Postgres/API never touch the host"
  - "No GHCR build/push job in the workflow — Coolify builds directly from source"
metrics:
  duration: 10min
  completed: 2026-07-02
---

# Phase quick-260702-k3u Plan 01: Configurar deploy de produção no Coolify Summary

Criado o stack de produção (`docker-compose.prod.yml`) e o workflow de CI (`.github/workflows/deploy-production.yml`) para publicar o FB_TESTESFC em https://testesfc.fbtax.cloud via Coolify, com isolamento de rede correto entre `fb_net` (privada) e `coolify` (externa/Traefik).

## What Was Built

**Task 1 — `docker-compose.prod.yml`:**
- Três serviços: `api`, `web`, `db` (sem stack de monitoramento, decisão do usuário).
- `api`: build do `backend/Dockerfile` existente, `container_name: fb_testesfc-api` (mantido exatamente igual ao valor hardcoded no upstream do `frontend/nginx.conf`), sem `ports:` publicadas (`expose: 8085` apenas), `DATABASE_URL` apontando para `db` interno ao stack, `COOKIE_SECURE=true` fixo, healthcheck via `wget` (a imagem alpine do backend não tem `curl`) contra `/api/health` (confirmado em `backend/main.go:367`). Rede: só `fb_net`, com alias `testesfc-api`.
- `web`: build do `frontend/Dockerfile` existente, `container_name: fb_testesfc-web`, sem `ports:` (`expose: 80`), único serviço nas duas redes (`fb_net` + `coolify`), com labels Traefik completos (`traefik.enable`, `docker.network=coolify`, router `testesfc` com `Host(\`testesfc.fbtax.cloud\`)`, `entrypoints=https`, `tls=true`, `tls.certresolver=letsencrypt`, `loadbalancer.server.port=80`).
- `db`: `postgres:15-alpine`, `container_name: fb_testesfc-db`, sem `ports:`, volume dedicado `postgres_data_testesfc_prod`, healthcheck `pg_isready`. Rede: **apenas `fb_net`** — nunca `coolify` (mitigação T-k3u-01, evita colisão de DNS entre stacks no Traefik).
- Redes de topo: `fb_net` (bridge interno) e `coolify` (`external: true`, gerenciada pelo Coolify).

**Task 2 — `.github/workflows/deploy-production.yml`:**
- Trigger: `push` em `main` + `workflow_dispatch` (disparo manual para outras branches).
- Job único `deploy-coolify`, condicionado a `github.ref == 'refs/heads/main'`.
- Um step que dispara o webhook do Coolify via `curl` com os flags de timeout/retry exigidos pelo plano (`--connect-timeout 10 --max-time 30 --retry 2 --retry-delay 5`), mitigando instabilidade conhecida de firewall na porta 8000 do Coolify.
- Credenciais via `${{ secrets.COOLIFY_WEBHOOK_URL }}` e `${{ secrets.COOLIFY_TOKEN }}` — nada hardcoded no arquivo.
- Sem job de build/push para GHCR (Coolify builda direto do source; job de push seria peso morto, conforme já observado no FB_APU04).

## Verification

Todos os checks automatizados do plano passaram:
- `docker compose -f docker-compose.prod.yml config` valida sem erro.
- `HAS_EXTERNAL`, `API_NAME_OK`, `HOST_OK` confirmados; `DB_ON_COOLIFY_FAIL` NÃO disparou.
- Confirmação manual de topologia de rede via `docker compose config` parseado: `api -> [fb_net]`, `db -> [fb_net]`, `web -> [coolify, fb_net]`.
- Workflow YAML válido (`python3 -c "import yaml..."`), `WEBHOOK_OK`, `CURL_FLAGS_OK`, `DISPATCH_OK` confirmados.

## Deviations from Plan

None - plan executed exactly as written.

## Key Decisions

- Isolamento de rede: apenas `web` na rede `coolify`; `api` e `db` restritos a `fb_net` — defesa em profundidade contra exposição indevida de Postgres/API ao Traefik e colisão de DNS entre stacks Coolify.
- `COOKIE_SECURE=true` fixo (não parametrizável via env) em produção, já que HTTPS é obrigatório.
- Nenhum serviço publica portas ao host em produção — todo tráfego externo passa pelo Traefik via rede `coolify`.
- Workflow de deploy simplificado: sem job de build/push de imagem para GHCR, pois o Coolify builda direto do código-fonte.

## Known Stubs

None.

## Threat Flags

None — toda a superfície de rede/deploy criada está coberta pelo `<threat_model>` do plano (T-k3u-01 a T-k3u-05), todas com disposição `mitigate` implementada.

## Self-Check: PASSED

- FOUND: docker-compose.prod.yml
- FOUND: .github/workflows/deploy-production.yml
- FOUND commit fe696f5 (docker-compose.prod.yml)
- FOUND commit 590ad48 (deploy-production.yml)

## Next Steps (Operator)

Estes passos exigem ação humana fora do escopo deste quick task (não automatizáveis pelo executor):
1. Configurar os secrets `COOLIFY_WEBHOOK_URL` e `COOLIFY_TOKEN` no repositório GitHub (Settings → Secrets and variables → Actions).
2. Configurar as variáveis de ambiente de produção (`DB_USER`, `DB_PASSWORD`, `DB_NAME`, `SMTP_*`, `JWT_SECRET`, `ENCRYPTION_KEY`, `ALLOWED_ORIGINS`, `APP_URL`) no painel do Coolify para o projeto FB_TESTESFC.
3. Confirmar no Coolify que o projeto está configurado para usar `docker-compose.prod.yml` como compose file de produção.
4. Fazer o primeiro push em `main` (ou disparar `workflow_dispatch`) para validar o redeploy automático.
