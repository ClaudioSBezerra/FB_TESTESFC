---
phase: quick-260702-nb4
plan: 01
subsystem: infra
tags: [docker-compose, godebug, multipart, go, upload]

requires: []
provides:
  - "Backend Go aceita upload multipart com mais de 1000 partes (arquivos) em dev e produção"
affects: [importacao-xml, upload-xml-zip]

tech-stack:
  added: []
  patterns:
    - "GODEBUG=multipartmaxparts=<N> como forma de sobrepor limites hardcoded da stdlib do Go sem alterar código"

key-files:
  created: []
  modified:
    - docker-compose.yml
    - docker-compose.prod.yml

key-decisions:
  - "GODEBUG=multipartmaxparts=100000 aplicado via variável de ambiente no serviço api (dev e prod) — resolve o limite hardcoded de 1000 partes em mime/multipart/formdata.go sem tocar em código Go"

patterns-established:
  - "Limites hardcoded da stdlib do Go relacionados a GODEBUG devem ser sobrepostos via variável de ambiente no compose, nunca via patch de código"

requirements-completed: [BUGFIX-multipart-parts]

duration: 5min
completed: 2026-07-02
---

# Quick Task 260702-nb4: Aumentar limite de partes multipart do Go Summary

**Adicionado `GODEBUG=multipartmaxparts=100000` ao serviço `api` em docker-compose.yml e docker-compose.prod.yml, removendo o limite hardcoded de 1000 partes multipart da stdlib do Go que bloqueava uploads de milhares de XMLs.**

## Performance

- **Duration:** 5min
- **Started:** 2026-07-02T19:46:00Z
- **Completed:** 2026-07-02T19:51:11Z
- **Tasks:** 1 completed
- **Files modified:** 2

## Accomplishments
- Upload de ~5.000 XMLs (ou qualquer lote com mais de 1000 arquivos) deixa de ser recusado pela stdlib do Go antes mesmo de chegar na validação da aplicação
- Correção aplicada de forma idêntica em dev (docker-compose.yml) e produção (docker-compose.prod.yml), sem alterar código Go

## Task Commits

Each task was committed atomically:

1. **Task 1: Adicionar GODEBUG=multipartmaxparts=100000 ao serviço api em dev e prod** - `31e73b7` (fix)

**Plan metadata:** (commit separado pelo orquestrador)

## Files Created/Modified
- `docker-compose.yml` - Adicionada `GODEBUG=multipartmaxparts=100000` ao bloco `environment:` do serviço `api` (dev)
- `docker-compose.prod.yml` - Adicionada `GODEBUG=multipartmaxparts=100000` ao bloco `environment:` do serviço `api` (prod)

## Decisions Made
- Nenhuma decisão nova além da já definida no plano: usar a variável de ambiente `GODEBUG=multipartmaxparts=100000` em vez de qualquer alteração de código Go, pois o limite é lido em runtime via `godebug.New("multipartmaxparts").Value()`.

## Deviations from Plan

None - plan executado exatamente como escrito.

## Verification Results

- `grep -c 'GODEBUG=multipartmaxparts=100000' docker-compose.yml` → `1`
- `grep -c 'GODEBUG=multipartmaxparts=100000' docker-compose.prod.yml` → `1`
- `docker compose config -q` → OK (exit 0)
- `docker compose -f docker-compose.prod.yml config -q` → OK (exit 0)
- Nenhuma outra linha dos arquivos compose foi alterada (diff mostra apenas 2 inserções, 0 remoções)

## Known Stubs

None.

## Threat Flags

None — variável de ambiente controla apenas um limite de parsing da stdlib, sem novo endpoint, caminho de auth, ou schema.

## Issues Encountered
None.

## Next Steps
- Após deploy (dev via `docker compose up --build --force-recreate api` / prod via Coolify redeploy), validar upload real de um lote com mais de 1000 XMLs para confirmar que o erro de "too many parts" não ocorre mais.

## Self-Check

- FOUND: docker-compose.yml (contém `GODEBUG=multipartmaxparts=100000`)
- FOUND: docker-compose.prod.yml (contém `GODEBUG=multipartmaxparts=100000`)
- FOUND: commit 31e73b7

## Self-Check: PASSED
