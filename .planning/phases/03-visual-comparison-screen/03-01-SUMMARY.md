---
phase: 03-visual-comparison-screen
plan: 01
subsystem: ui
tags: [go, postgres, react, react-query, tailwind, shadcn]

# Dependency graph
requires:
  - phase: 02-import-pipeline-fiscal-execution
    provides: nfe_saidas / nfe_saidas_itens (esperado, do XML) e fiscal_execution_items (calculado, pacote fiscal) já persistidos e populáveis via /api/fiscal-execution/run
provides:
  - "GET /api/fiscal-comparison — endpoint autenticado, escopado por company_id, com JOIN esperado x calculado por item"
  - "Página ComparacaoFiscal.tsx navegável em /importacoes/comparacao-fiscal"
  - "Funções puras exportadas pairDiff/isDivergente/itemBucket — contrato reutilizável pelo Plano 03-02 (Dialog de detalhe)"
affects: [03-02-visual-comparison-detail, verificação-fase-3]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Handler Go de leitura com JOIN (esperado x calculado) escopado por company_id via erpBridgeGetCompany — mesmo padrão de NFeSaidaDetailHandler"
    - "Colunas calculadas nullable via sql.NullFloat64 → *float64 (JSON null quando item não processado)"
    - "Classificação de item em 3 baldes (ok/divergente/nao_calculado) como funções puras testáveis, exportadas da página"

key-files:
  created:
    - backend/handlers/fiscal_comparison.go
    - frontend/src/pages/ComparacaoFiscal.tsx
  modified:
    - backend/main.go
    - frontend/src/lib/navigation.ts
    - frontend/src/App.tsx

key-decisions:
  - "Divergência = qualquer diferença != 0 nos 4 pares de VALOR (ICMS, ICMS-ST, PIS, COFINS), sem tolerância de arredondamento (D-06)"
  - "Item com fiscal_status != 'ok' é classificado como 'Não calculado', nunca como divergente (D-10)"
  - "Lista principal mostra o par de VALOR de cada imposto (não a base) nas 3 colunas Esperado/Calculado/Diferença; bases ficam reservadas para o Dialog de detalhe do Plano 03-02"

patterns-established:
  - "pairDiff(esperado, calculado) / isDivergente(item) / itemBucket(item) exportados de ComparacaoFiscal.tsx para reuso no Dialog de detalhe (03-02) sem duplicar lógica de comparação"

requirements-completed: [CMP-01, CMP-02]

# Metrics
duration: 20min
completed: 2026-07-02
---

# Phase 3 Plan 1: Visual Comparison Screen (List) Summary

**Endpoint GET /api/fiscal-comparison + página "Comparação Fiscal" listando item a item, com 4 impostos x 3 colunas (Esperado/Calculado/Diferença) e destaque vermelho para divergências.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-07-02T13:00:00Z (aprox.)
- **Completed:** 2026-07-02T13:32:00Z
- **Tasks:** 2/2 completed
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments
- Endpoint `/api/fiscal-comparison` faz o JOIN `nfe_saidas_itens` (esperado) x `nfe_saidas` x `fiscal_execution_items` (calculado), escopado por `company_id` resolvido via JWT (nunca aceito do cliente), com `LIMIT 2000` como guarda de payload.
- Página React "Comparação Fiscal" navegável, listando todos os itens de todas as notas da empresa com 4 blocos de imposto (ICMS, ICMS-ST, PIS, COFINS) x 3 colunas (Esperado, Calculado, Diferença), badge de 3 estados (OK/Divergente/Não calculado) e toggle "só divergentes" + cards de resumo.
- Lógica de comparação (`pairDiff`, `isDivergente`, `itemBucket`) exportada como funções puras, prontas para reuso no Dialog de detalhe do Plano 03-02.

## Task Commits

Each task was committed atomically:

1. **Task 1: Endpoint GET /api/fiscal-comparison (JOIN esperado x calculado)** - `4e294dd` (feat)
2. **Task 2: Página ComparacaoFiscal.tsx + wiring de navegação/rota** - `5cc91bd` (feat)

## Files Created/Modified
- `backend/handlers/fiscal_comparison.go` - `FiscalComparisonListHandler`; JOIN esperado x calculado escopado por company_id, colunas calculadas como `*float64` (null quando não processado)
- `backend/main.go` - registra `/api/fiscal-comparison` via `withAuth(handlers.FiscalComparisonListHandler, "")`
- `frontend/src/pages/ComparacaoFiscal.tsx` - página de lista com tabela de 4 impostos x 3 colunas, badge de divergência, filtro e cards de resumo
- `frontend/src/lib/navigation.ts` - aba "Comparação Fiscal" adicionada ao módulo `config`
- `frontend/src/App.tsx` - import + rota protegida `/importacoes/comparacao-fiscal`

## Decisions Made
- Nenhuma decisão nova além das já registradas em `03-CONTEXT.md` (D-01 a D-10) — plano executado seguindo essas decisões ao pé da letra.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- O endpoint e a página cobrem CMP-01/CMP-02 no nível da lista. O Plano 03-02 pode agora construir o Dialog de detalhe (D-03/D-07: seção "Só calculado" com `full_result`) reaproveitando `pairDiff`/`isDivergente`/`itemBucket` já exportados desta página, evitando duplicar a lógica de comparação.
- Verificação funcional plena depende de dados reais na base (itens de `nfe_saidas_itens` com correspondência em `fiscal_execution_items`); a Fase 2 já validou o pipeline contra Oracle real, então dados de teste devem estar disponíveis para o checkpoint humano de fim de fase.
- Nenhum bloqueio técnico identificado para o Plano 03-02.

---
*Phase: 03-visual-comparison-screen*
*Completed: 2026-07-02*
