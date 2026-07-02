---
phase: quick-260702-ju0
plan: 01
subsystem: ui
tags: [react, login, frontend]

# Dependency graph
requires: []
provides:
  - Tela de login com bullet único refletindo o propósito real do FB_TESTESFC (não mais os 5 itens herdados do FB_APU04 sobre EFD/SPED/dashboards)
affects: [login-page]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - frontend/src/pages/Login.tsx

key-decisions:
  - "Escopo restrito exatamente ao array FEATURES conforme instrução explícita do plano ('não alterar nenhuma outra parte do arquivo') — footer 'EFD ICMS/IPI v{apiVersion}' (linha 135) não foi tocado por estar fora do escopo da task, apesar de também ser um resquício herdado do FB_APU04"

patterns-established: []

requirements-completed: [QUICK-ju0]

# Metrics
duration: 5min
completed: 2026-07-02
---

# Quick Task 260702-ju0: Trocar bullets da tela de login Summary

**Array `FEATURES` da tela de login substituído por um único item que reflete o propósito real do FB_TESTESFC (validação de recálculo fiscal), removendo os 5 bullets herdados do FB_APU04 sobre EFD/SPED/dashboards.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-07-02T17:14:00Z
- **Completed:** 2026-07-02T17:19:02Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- `FEATURES` agora contém exatamente um item: "Ler base atual de saídas e fazer cálculos para reforma tributária"
- Removidos os 5 textos herdados sobre EFD ICMS, ERP Bridge/PIS/COFINS/IPI, dashboards, rastreabilidade EFD e SPED layout 020
- `tsc --noEmit` não reporta novos erros em `Login.tsx`

## Task Commits

Each task was committed atomically:

1. **Task 1: Substituir array FEATURES por item único** - `790bebf` (feat)

## Files Created/Modified
- `frontend/src/pages/Login.tsx` - Array `FEATURES` reduzido de 5 itens (EFD/SPED/dashboards herdados do FB_APU04) para 1 item refletindo o propósito do FB_TESTESFC

## Decisions Made
- Mantido escopo estrito ao array `FEATURES` conforme instrução explícita da task ("não alterar nenhuma outra parte do arquivo"). Ver Issues Encountered para o footer de versão que também menciona "EFD".

## Deviations from Plan

None - plan executado exatamente como escrito. O array `FEATURES` foi substituído sem tocar em mais nada do arquivo (constante, imports, lógica de renderização intactos).

## Issues Encountered

O verify automatizado do plano usa `grep -qE 'EFD|SPED|dashboards|Rastreabilidade'` sobre o arquivo inteiro, que também casa com um footer pré-existente e fora do escopo da task, não relacionado ao array `FEATURES`: `EFD ICMS/IPI v{apiVersion}` (linha 135, rodapé com a versão do backend). Este texto já existia antes desta task e não faz parte dos "5 itens" que a task pede para remover — é um label de versão herdado do FB_APU04, fora do escopo desta quick task cuja instrução explícita foi "não alterar nenhuma outra parte do arquivo". Não foi corrigido aqui. Registrado como item potencial para uma futura quick task, caso o usuário queira renomear esse rótulo de versão também.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Tela de login pronta com mensagem correta sobre o propósito do FB_TESTESFC.
- Possível follow-up (fora de escopo): renomear o rótulo de versão do rodapé "EFD ICMS/IPI v{apiVersion}" para refletir o nome do FB_TESTESFC, se desejado.

---
*Phase: quick-260702-ju0*
*Completed: 2026-07-02*

## Self-Check: PASSED

- FOUND: frontend/src/pages/Login.tsx
- FOUND: 790bebf (Task 1 commit)
- FOUND: .planning/quick/260702-ju0-na-tela-de-login-trocar-a-lista-de-bulle/260702-ju0-SUMMARY.md
