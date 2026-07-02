---
phase: 03-visual-comparison-screen
plan: 02
subsystem: ui
tags: [go, postgres, react, react-query, tailwind, shadcn]

# Dependency graph
requires:
  - phase: 03-visual-comparison-screen
    provides: "GET /api/fiscal-comparison (lista) e pairDiff/isDivergente/itemBucket exportados de ComparacaoFiscal.tsx (Plano 03-01)"
provides:
  - "GET /api/fiscal-comparison/{id} — endpoint de detalhe autenticado, escopado por company_id, com full_result + DIFAL/FCP"
  - "Dialog de detalhe do item em ComparacaoFiscal.tsx: comparação Base+Valor, seção 'Só calculado' e resumo por nota"
  - "Cards de resumo global que respeitam o toggle 'só divergentes' (derivam de displayItems, não da lista bruta)"
affects: [verificação-fase-3]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Handler Go de detalhe (QueryRow único) escopado por company_id — mesmo padrão de NFeSaidaDetailHandler, reaproveitando erpBridgeGetCompany e jsonErr"
    - "full_result (JSONB) escaneado como []byte e reemitido como json.RawMessage — o frontend decodifica e mapeia rótulos amigáveis para um subconjunto curado, com fallback genérico chave→valor para os demais campos"
    - "Dialog de detalhe reaproveita Secao/Linha/LinhaBRL (padrão ConsultaNFeSaidas.tsx) + novo helper LinhaComparativa (3 colunas Esperado/Calculado/Diferença)"

key-files:
  created: []
  modified:
    - backend/handlers/fiscal_comparison.go
    - backend/main.go
    - frontend/src/pages/ComparacaoFiscal.tsx

key-decisions:
  - "Curated mapping de full_result (IBS UF/Município, CBS, alíquotas, MVA, partilha destino) com rótulos amigáveis; demais ~70 campos renderizados genericamente (chave → valor) numa seção separada 'Demais campos do pacote fiscal' — evita esconder dado de auditoria sem exigir mapeamento manual de todos os ~88 campos (discretion do plano)"
  - "Cards de resumo global recalculados para derivar de displayItems (pós-filtro), não da lista bruta — antes do Plano 03-02 os cards já existiam mas contavam sobre items sem filtro; corrigido para respeitar 'só divergentes' conforme D-09"
  - "Resumo por nota no Dialog usa os itens já carregados na lista (items, sem filtro) agrupados por nfe_id do item selecionado — nenhuma chamada de rede adicional"

patterns-established:
  - "LinhaComparativa(label, esperado, calculado) — variante de Linha com 3 colunas e destaque vermelho quando pairDiff != 0, reutilizável em futuras seções de comparação"

requirements-completed: [CMP-03, CMP-04]

# Metrics
duration: 20min
completed: 2026-07-02
---

# Phase 3 Plan 2: Visual Comparison Screen (Detail) Summary

**Endpoint de detalhe GET /api/fiscal-comparison/{id} + Dialog no item da tabela com as 8 comparações Base/Valor, seção "Só calculado" (DIFAL/FCP/full_result) e resumo da nota; cards de resumo global agora respeitam o toggle "só divergentes".**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2/2 completed
- **Files modified:** 3 (0 created, 3 modified)

## Accomplishments
- `FiscalComparisonDetailHandler` faz `QueryRow` sobre `nfe_saidas_itens` JOIN `nfe_saidas` LEFT JOIN `fiscal_execution_items`, escopado por `WHERE i.id = $1 AND i.company_id = $2` (IDOR-safe — item de outra empresa retorna 404, não 200 com dado vazado). Retorna os 8 pares esperado/calculado, DIFAL/FCP/percentual_difal e `full_result` como `json.RawMessage`.
- Rota `/api/fiscal-comparison/{id}` registrada em `main.go` com `withAuth`, ao lado da rota de lista do Plano 03-01.
- `ComparacaoFiscal.tsx` ganhou `DetalheItem`: Dialog acionado por clique na linha, com seção "Comparação — Esperado vs. Calculado" (8 `LinhaComparativa`), seção "Só calculado (sem par no XML)" com DIFAL/FCP + campos curados de `full_result` (IBS UF/Município, CBS, alíquotas, MVA) + fallback genérico para os demais campos, e "Resumo da nota" com contagem OK/Divergente/Não calculado dos itens da mesma NF-e.
- Cards de resumo global corrigidos para derivar de `displayItems` (pós-filtro) em vez da lista bruta — agora refletem corretamente o toggle "só divergentes" (CMP-04/D-09).
- Item com `fiscal_status !== 'ok'` exibe aviso "Item ainda sem cálculo fiscal — nada a comparar." em vez das seções de valores, evitando comparação sem sentido.

## Task Commits

Each task was committed atomically:

1. **Task 1: Endpoint de detalhe GET /api/fiscal-comparison/{id}** - `a3318ec` (feat)
2. **Task 2: Toggle/cards/Dialog de detalhe no frontend** - `9b5f20c` (feat)

## Files Created/Modified
- `backend/handlers/fiscal_comparison.go` - `FiscalComparisonDetailHandler`; QueryRow único com full_result + DIFAL/FCP, escopado por company_id
- `backend/main.go` - registra `/api/fiscal-comparison/` (trailing slash) via `withAuth(handlers.FiscalComparisonDetailHandler, "")`
- `frontend/src/pages/ComparacaoFiscal.tsx` - `DetalheItem` (Dialog), `LinhaComparativa`, `Secao/Linha/LinhaBRL`, `FULL_RESULT_LABELS`, linhas da tabela clicáveis, cards recalculados sobre `displayItems`

## Decisions Made
- Mapeamento curado de `full_result` (ver key-decisions acima) — subconjunto de auditoria de maior valor com rótulos amigáveis, resto genérico.
- Cards de resumo global agora usam `displayItems` (correção de comportamento vs. o que o Plano 03-01 havia deixado, que contava sobre `items` sem filtro).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Cards de resumo global não respeitavam o filtro "só divergentes"**
- **Found during:** Task 2 (leitura do código herdado do Plano 03-01 antes de estender)
- **Issue:** `countOK`/`countDivergente`/`countNaoCalculado` e o card "Total itens" eram calculados sobre `items` (lista bruta), não sobre `displayItems` (pós-filtro) — violava a truth do 03-02-PLAN.md "Cards no topo... respeitando o filtro atual" (CMP-04/D-09).
- **Fix:** Recalculados os três contadores e o total para derivar de `displayItems`.
- **Files modified:** `frontend/src/pages/ComparacaoFiscal.tsx`
- **Commit:** `9b5f20c`

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CMP-01 a CMP-04 estão cobertos: lista com 4 pares Esperado/Calculado/Diferença (03-01), filtro "só divergentes" (03-02), cards de resumo global + resumo por nota (03-02), e Dialog de detalhe com a seção "Só calculado" (DIFAL/FCP/full_result) (03-02).
- Verificação funcional completa (visual) fica para o checkpoint humano do Plano 03-03, conforme já definido no 03-02-PLAN.md.
- Nenhum bloqueio técnico identificado para o Plano 03-03.

---
*Phase: 03-visual-comparison-screen*
*Completed: 2026-07-02*

## Self-Check: PASSED

- FOUND: backend/handlers/fiscal_comparison.go
- FOUND: backend/main.go
- FOUND: frontend/src/pages/ComparacaoFiscal.tsx
- FOUND: a3318ec (Task 1 commit)
- FOUND: 9b5f20c (Task 2 commit)
- FOUND: func FiscalComparisonDetailHandler
- FOUND: function DetalheItem
