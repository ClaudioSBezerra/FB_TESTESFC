---
phase: quick-260702-le0
plan: 01
subsystem: infra
tags: [nginx, docker-compose, coolify, deploy, production-fix]

# Dependency graph
requires:
  - phase: quick-260702-k3u
    provides: docker-compose.prod.yml com alias de rede testesfc-api já configurado (referência)
provides:
  - nginx.conf com upstream resolvendo via alias de rede estável (testesfc-api), independente de renomeação de container pelo Coolify
  - docker-compose.yml (dev) com o mesmo alias de rede, mantendo o dev local funcional após a mudança no nginx.conf
affects: [deploy-producao, frontend-nginx, docker-compose-dev]

# Tech tracking
tech-stack:
  added: []
  patterns: ["upstream nginx via alias de rede docker (não container_name) — padrão já usado no FB_APU04"]

key-files:
  created: []
  modified:
    - frontend/nginx.conf
    - docker-compose.yml

key-decisions:
  - "Upstream do nginx.conf trocado de container_name (fb_testesfc-api) para alias de rede (testesfc-api), pois o Coolify ignora container_name e renomeia containers em produção"
  - "docker-compose.yml (dev) recebeu o alias de rede de forma aditiva (mantendo container_name) para não quebrar nenhuma outra referência local ao hostname fb_testesfc-api"

patterns-established: []

requirements-completed: [BUGFIX-503]

# Metrics
duration: 1min
completed: 2026-07-02
---

# Quick Task 260702-le0: Corrigir bug de deploy em produção (503) Summary

**Upstream do nginx.conf trocado de container_name (`fb_testesfc-api`, ignorado pelo Coolify) para alias de rede estável (`testesfc-api`), eliminando o crash loop que causava 503 em produção.**

## Performance

- **Duration:** ~1 min
- **Started:** 2026-07-02T18:25:46Z
- **Completed:** 2026-07-02T18:26:27Z
- **Tasks:** 2/2 completed
- **Files modified:** 2

## Accomplishments
- `frontend/nginx.conf` agora aponta o upstream `backend` para `testesfc-api:8085` (alias de rede), resolvendo tanto em produção (Coolify) quanto em dev local.
- `docker-compose.yml` (dev) passou a expor o serviço `api` sob o alias de rede `testesfc-api`, de forma aditiva ao `container_name` existente, mantendo o ambiente local funcional.
- `docker-compose.prod.yml` permanece inalterado — já estava correto e serviu de referência.

## Task Commits

Each task was committed atomically:

1. **Task 1: Corrigir upstream do nginx.conf para usar o alias de rede** - `d43060e` (fix)
2. **Task 2: Adicionar alias de rede testesfc-api ao serviço api do docker-compose.yml (dev)** - `b22df36` (fix)

_Note: sem tarefas TDD nesta plan — mudança de configuração pura._

## Files Created/Modified
- `frontend/nginx.conf` - upstream `backend` agora resolve `testesfc-api:8085` em vez de `fb_testesfc-api:8085`
- `docker-compose.yml` - serviço `api` migrado de `networks: [fb_testesfc_net]` (lista simples) para forma expandida com `aliases: [testesfc-api]`, mantendo `container_name: fb_testesfc-api`

## Decisions Made
- Alias de rede escolhido em vez de qualquer alternativa (ex: variável de ambiente para hostname) porque replica exatamente o padrão já comprovado em produção pelo projeto irmão FB_APU04 (`server apu04-api:8084;`), minimizando risco na correção urgente.
- Mudança no `docker-compose.yml` (dev) feita de forma estritamente aditiva — não removeu `container_name` nem alterou outros serviços — para não introduzir risco em ambiente local que não está com bug.

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — mudança restrita a configuração de rede interna entre containers, sem novo endpoint, caminho de auth ou acesso a dados.

## Verification Results
- `grep -c 'server testesfc-api:8085;' frontend/nginx.conf` → `1` (OK)
- `grep -q 'fb_testesfc-api' frontend/nginx.conf` → não encontrado (OK)
- `docker compose -f docker-compose.yml config` → válido, sem erro de sintaxe (OK)
- `grep -q 'container_name: fb_testesfc-api' docker-compose.yml` → presente (OK)
- `git diff --name-only HEAD~2 HEAD` → apenas `frontend/nginx.conf` e `docker-compose.yml`; `docker-compose.prod.yml` não aparece (OK)

## Next Steps
- Teste funcional em produção (deploy via Coolify) fica a cargo do usuário, fora do escopo desta correção — confirmar que https://testesfc.fbtax.cloud volta a responder 200 após o próximo deploy.

## Self-Check: PASSED

- FOUND: frontend/nginx.conf
- FOUND: docker-compose.yml
- FOUND: d43060e (Task 1 commit)
- FOUND: b22df36 (Task 2 commit)
