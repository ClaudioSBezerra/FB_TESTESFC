---
phase: quick
plan: 260702-lp5
subsystem: database
tags: [postgres, migration, auth-hierarchy, coolify, deploy]

# Dependency graph
requires:
  - phase: quick-260702-k3u
    provides: deploy de produção no Coolify (docker-compose.prod.yml)
provides:
  - Migração idempotente que garante Ambiente/Grupo/Empresa MASTER e vincula o admin
  - Padrão MASTER documentado na skill global coolify-deploy-checklist
affects: [deploy-checklist, produção, próximo-produto-fbtax]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Migração NNN_ensure_master_link.sql: SELECT...INTO / IF NULL THEN INSERT...RETURNING com RAISE NOTICE, idempotente, roda em todo boot"

key-files:
  created:
    - backend/migrations/010_ensure_master_link.sql
  modified:
    - ~/.claude/skills/coolify-deploy-checklist/SKILL.md (fora do repo git, não commitado)

key-decisions:
  - "Migração portada quase literalmente de FB_APU04/backend/migrations/024_ensure_master_link.sql — schema 1:1 compatível"
  - "Padrão MASTER capturado como seção 1.7 da skill global, antes da seção 2 (verificação pós-deploy), para produtos futuros da família FBTax nascerem com essa migração"

patterns-established:
  - "Toda hierarquia Ambiente→Grupo→Empresa→user_environments dessa família de produtos deve ter uma migração ensure_master_link.sql para evitar estado 'logado sem tenant' no primeiro deploy com banco zerado"

requirements-completed: [QUICK-LP5]

# Metrics
duration: 8min
completed: 2026-07-02
---

# Quick Task 260702-lp5: Migração 010_ensure_master_link.sql Summary

**Migração idempotente que garante hierarquia MASTER (Ambiente→Grupo→Empresa) vinculada ao admin claudio_bezerra@hotmail.com, portada do FB_APU04, mais o padrão documentado na skill global de deploy para produtos futuros.**

## Performance

- **Duration:** 8min
- **Started:** 2026-07-02T18:33:00Z
- **Completed:** 2026-07-02T18:41:32Z
- **Tasks:** 2/2 completed
- **Files modified:** 2 (1 no repo git, 1 na skill global fora do repo)

## Accomplishments

- Criada `backend/migrations/010_ensure_master_link.sql`, port quase literal de `024_ensure_master_link.sql` do FB_APU04, adaptado à data e mantendo todos os nomes/strings ('MASTER', 'Ambiente Principal de Administração', 'Grupo Corporativo', 'MASTER Corporation', role 'admin', email do admin) e a lógica idempotente com `RAISE NOTICE`.
- Documentado o padrão MASTER como seção `## 1.7.` na skill global `coolify-deploy-checklist`, posicionada corretamente antes da seção `## 2. Depois do PRIMEIRO deploy`, referenciando tanto o padrão canônico do FB_APU04 quanto a aplicação concreta no FB_TESTESFC.

## Task Commits

Each task was committed atomically:

1. **Task 1: Criar migração 010_ensure_master_link.sql** - `0ef828a` (feat)
2. **Task 2: Documentar padrão MASTER na skill global coolify-deploy-checklist** - N/A (arquivo fora do repositório git do FB_TESTESFC, não commitado por design)

_Nota: a Task 2 edita `~/.claude/skills/coolify-deploy-checklist/SKILL.md`, que vive fora deste repositório — intencionalmente não versionado neste commit conforme escopo da tarefa._

## Files Created/Modified

- `backend/migrations/010_ensure_master_link.sql` - Migração idempotente `DO $$ ... END $$;` que garante Ambiente/Grupo/Empresa MASTER e vincula claudio_bezerra@hotmail.com via `user_environments` (role admin). Roda sozinha no próximo boot do backend, local ou produção.
- `~/.claude/skills/coolify-deploy-checklist/SKILL.md` - Nova seção 1.7 documentando o padrão de migração `ensure_master_link` como item do checklist de novo produto da família FBTax.

## Verification

- `backend/migrations/010_ensure_master_link.sql`: bloco `DO $$ ... END $$;` balanceado (1 abertura, 1 fechamento) confirmado via `awk`; 17 ocorrências de `MASTER` no arquivo (>= 4 esperadas); arquivo existe.
- `~/.claude/skills/coolify-deploy-checklist/SKILL.md`: seção `## 1.7.` existe, contém `ensure_master_link`, e está posicionada antes de `## 2. Depois do PRIMEIRO deploy` (confirmado via `awk` com números de linha).
- Nenhuma execução contra banco real necessária — validação apenas de sintaxe/estrutura via grep/awk, conforme escopo do plano.

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — migração segue o padrão já estabelecido em `001_auth_hierarchy.sql` e `004_seed_ferreira_costa.sql`, sem introduzir superfície nova (sem novos endpoints, sem novas colunas de trust boundary).

## Self-Check: PASSED

- FOUND: backend/migrations/010_ensure_master_link.sql
- FOUND: ~/.claude/skills/coolify-deploy-checklist/SKILL.md section 1.7
- FOUND: commit 0ef828a
