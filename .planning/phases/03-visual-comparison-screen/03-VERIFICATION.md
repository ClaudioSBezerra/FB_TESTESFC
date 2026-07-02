---
phase: 03-visual-comparison-screen
verified: 2026-07-02T14:15:22Z
status: passed
score: 4/4 must-haves verified
overrides_applied: 0
re_verification: false
---

# Phase 3: Visual Comparison Screen — Verification Report

**Phase Goal:** Usuário visualiza, item a item e imposto a imposto, o valor esperado (do XML) versus o calculado (pelo pacote fiscal), com divergências destacadas e filtros para análise rápida.

**Verified:** 2026-07-02T14:15:22Z
**Status:** PASSED
**Re-verification:** No — initial verification

**Note on `mode: mvp`:** ROADMAP.md marks this phase `mode: mvp`, but the phase goal is not in the `"As a X, I want to Y, so that Z."` user-story format required for MVP-mode UAT framing (`references/verify-mvp-mode.md`). This is a project-wide convention (Phases 1-3 all carry `mode: mvp` with non-user-story goals), and Phase 1's own verification (`01-VERIFICATION.md`) proceeded with standard goal-backward methodology without applying the MVP framing or flagging a discrepancy. Following that established precedent, this report applies the standard (non-MVP) goal-backward methodology.

---

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Tela de comparação exibe cada item com colunas lado a lado para esperado (XML) e calculado (script) nas colunas fiscais: base ICMS, vlr ICMS, base ST, ICMS ST, base PIS/COFINS, vlr PIS/COFINS, DIFAL, FCP e demais retornados pelo script | ✓ VERIFIED | List table (`ComparacaoFiscal.tsx:549-608`) shows Esperado\|Calculado\|Diferença for VALOR of ICMS/ICMS-ST/PIS/COFINS (D-04). Dialog (`DetalheItem`, lines 373-393) adds the 4 BASE comparisons. "Só calculado" section (lines 395-408) shows DIFAL (`valor_icms_partilha_destino`), FCP (`valor_icms_pobreza`), `percentual_difal`, `grupo_fiscal_codigo`, curated `full_result` fields (IBS UF/Município, CBS, aliquots, MVA) plus a generic fallback section for the remaining ~88 `full_result` fields. Confirmed live via `GET /api/fiscal-comparison/{id}` — response includes 31 keys including `full_result` with 88 entries. |
| 2 | Itens com qualquer divergência são destacados visualmente (cor/ícone) e a diferença numérica é exibida em cada campo divergente | ✓ VERIFIED | `isDivergente()` (line 126) implements "any diff != 0 in any of the 4 value pairs, only when `fiscal_status==='ok'`" (D-06/D-10) as a pure, exported function. Table rows get `bg-red-50` when `itemBucket(item)==='divergente'` (line 582); `DiffCells` (line 198) renders the numeric delta in `text-red-700 font-bold` when `diff!=0`. Dialog's `LinhaComparativa` mirrors the same highlight. Live API check: item `7d44a2d3-...` has `esp_icms=18, calc_icms=0` (real R$18 divergence) — this is the exact scenario the human-verify checkpoint (03-03-SUMMARY.md) screenshotted and approved. |
| 3 | Usuário pode filtrar a visualização para exibir apenas os itens com ao menos uma divergência | ✓ VERIFIED | `somenteDivergentes` toggle (line 433) + `displayItems` `useMemo` (line 453-457) filters to `itemBucket(item)==='divergente'` only — "Não calculado" items are excluded by construction (`isDivergente` returns `false` when `fiscal_status !== 'ok'`). Table iterates `displayItems`, not raw `items` (line 578). |
| 4 | Tela apresenta um resumo com total de itens, itens sem divergência e itens com divergência — por nota e por lote importado | ✓ VERIFIED (scope narrowed by explicit user decision, D-09) | Top-of-page cards (lines 508-522) show Total/OK/Divergente/Não calculado derived from `displayItems` (respects current filter). Dialog's "Resumo da nota" (lines 358-365, `resumoNota` useMemo at 323-332) shows the same 4-way breakdown scoped to the item's `nfe_id`. "Por lote importado" was explicitly discussed and descoped during context-gathering (03-DISCUSSION-LOG.md lines 80-86: user chose "Global + por nota" over "Por lote de importação", because a "lote" entity doesn't exist in the schema and creating one was out of scope per PROJECT.md). REQUIREMENTS.md's own CMP-04 wording is "por nota **e/ou** por lote importado" (either/or), which this decision satisfies. |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `backend/handlers/fiscal_comparison.go` | `FiscalComparisonListHandler` + `FiscalComparisonDetailHandler`, JOIN esperado×calculado scoped by company_id | ✓ VERIFIED | 305 lines. Both handlers present, compile cleanly (`go build ./...` exit 0). List: `WHERE i.company_id = $1`, `LIMIT 2000`, nullable calc fields via `sql.NullFloat64` → `*float64`. Detail: `WHERE i.id = $1 AND i.company_id = $2` (IDOR-safe), `full_result` as `json.RawMessage`, `sql.ErrNoRows` → 404. |
| `frontend/src/pages/ComparacaoFiscal.tsx` | List page + toggle + summary cards + detail Dialog | ✓ VERIFIED | 624 lines. Exports `pairDiff`, `isDivergente`, `itemBucket` as pure functions; renders full table, toggle, cards, and `DetalheItem` Dialog. |
| `frontend/src/lib/navigation.ts` | "Comparação Fiscal" tab registered | ✓ VERIFIED | `{ label: 'Comparação Fiscal', path: '/importacoes/comparacao-fiscal' }` present (line 22). |
| `frontend/src/App.tsx` | Protected route wired | ✓ VERIFIED | `import ComparacaoFiscal` (line 15) + `<Route path="/importacoes/comparacao-fiscal" element={<ProtectedRoute><ComparacaoFiscal /></ProtectedRoute>} />` (line 66). |
| `backend/main.go` | Both routes registered with auth | ✓ VERIFIED | Lines 363-364: `/api/fiscal-comparison` and `/api/fiscal-comparison/` both via `withAuth(..., "")`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `ComparacaoFiscal.tsx` (list) | `GET /api/fiscal-comparison` | `fetch` in `useQuery` | ✓ WIRED | Live call confirmed: returned `total:6, items:[...]` with real Postgres data (test XML from Fase 2). |
| `ComparacaoFiscal.tsx` (`DetalheItem`) | `GET /api/fiscal-comparison/{id}` | `fetch` in `useQuery` | ✓ WIRED | Live call confirmed: returned 31-key JSON including `full_result` (88 entries), `valor_icms_partilha_destino`, `valor_icms_pobreza`. |
| `fiscal_comparison.go` (list) | `nfe_saidas_itens` + `nfe_saidas` + `fiscal_execution_items` | `JOIN` / `LEFT JOIN`, scoped by `company_id` | ✓ WIRED | Confirmed via schema (migrations 006/007/008) and live query results (calc_* fields null for `status=error`/`sem_grupo_fiscal`, populated for `status=ok`). |
| `fiscal_comparison.go` (detail) | `fiscal_execution_items.full_result` | `SELECT` scoped by `company_id` + `id` | ✓ WIRED | Live call returns 88-key `full_result` JSONB. |
| `main.go` | `handlers.FiscalComparisonListHandler` / `DetailHandler` | `http.HandleFunc` + `withAuth` | ✓ WIRED | Confirmed unauthenticated `GET /api/fiscal-comparison` → 401; authenticated → 200. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `ComparacaoFiscal.tsx` list table | `items` (from `useQuery`) | `GET /api/fiscal-comparison` → Postgres JOIN | Yes — live call returned 6 real items, one with `fiscal_status='ok'` and a genuine ICMS divergence (`esp_icms=18` vs `calc_icms=0`) | ✓ FLOWING |
| `ComparacaoFiscal.tsx` `DetalheItem` | `data` (from `useQuery`) | `GET /api/fiscal-comparison/{id}` → Postgres `QueryRow` incl. `full_result` | Yes — live call returned full 88-field `full_result`, DIFAL/FCP fields, `grupo_fiscal_codigo` | ✓ FLOWING |
| Summary cards | `countOK`/`countDivergente`/`countNaoCalculado` | `useMemo` over `displayItems` (client-derived from the same fetched `items`) | Yes — no separate network call; derives directly from real fetched data | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| List endpoint requires auth | `curl GET /api/fiscal-comparison` (no token) | HTTP 401 | ✓ PASS |
| List endpoint returns real comparison data | `curl GET /api/fiscal-comparison` (authenticated) | HTTP 200, `total:6`, items include one `fiscal_status:'ok'` with `esp_icms:18, calc_icms:0` (real divergence) and PIS/COFINS matching exactly | ✓ PASS |
| Detail endpoint returns full_result + DIFAL/FCP | `curl GET /api/fiscal-comparison/{ok-item-id}` | HTTP 200, 31 keys, `full_result` has 88 entries | ✓ PASS |
| Detail endpoint is IDOR-safe (404, not leak) | `curl GET /api/fiscal-comparison/00000000-0000-0000-0000-000000000000` | HTTP 404 | ✓ PASS |
| Backend compiles | `go -C backend build ./...` | exit 0 | ✓ PASS |
| Frontend builds/typechecks | `npm --prefix frontend run build` | exit 0 (only a chunk-size warning, non-blocking) | ✓ PASS |

All spot-checks re-run independently in this verification session (not sourced from SUMMARY.md claims) against the live `docker compose` stack left running from the 03-03 checkpoint.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CMP-01 | 03-01 | Lista item a item, esperado vs. calculado, para os principais impostos | ✓ SATISFIED | List table + Dialog Base/Valor comparisons (see Truth #1). |
| CMP-02 | 03-01 | Divergências destacadas visualmente com diferença numérica | ✓ SATISFIED | `isDivergente`/red highlighting/DiffCells (see Truth #2); confirmed with a real divergence in live data. |
| CMP-03 | 03-02 | Filtro para exibir só itens divergentes | ✓ SATISFIED | `somenteDivergentes` toggle + `displayItems` (see Truth #3); human checkpoint confirmed filtering 6→1 item. |
| CMP-04 | 03-02 | Resumo (total/OK/divergente) por nota e/ou por lote | ✓ SATISFIED | Global cards + per-nota summary in Dialog (see Truth #4); "por lote" explicitly descoped by user decision D-09, consistent with REQUIREMENTS.md's "e/ou" wording. |

No orphaned requirements — REQUIREMENTS.md maps only CMP-01..04 to Phase 3, and all four appear in the `requirements:` frontmatter of 03-01/03-02/03-03 plans.

### Anti-Patterns Found

None. Scanned `backend/handlers/fiscal_comparison.go`, `frontend/src/pages/ComparacaoFiscal.tsx`, `frontend/src/lib/navigation.ts`, `frontend/src/App.tsx`, `backend/main.go` for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"coming soon"/"not yet implemented" — no matches (the two `jsonErr(w, http.StatusMethodNotAllowed, "Método não permitido")` grep hits are false positives from the word "TODO"-adjacent pattern match on "Método", not actual debt markers). No stub returns (`return null`/`return {}`/empty handlers) found in the reviewed files.

### Human Verification Required

None outstanding. The mandatory human-verify checkpoint (03-03) was already executed and approved in a prior session within this same conversation, with concrete evidence: real `docker compose` stack, authenticated `curl` calls, and Playwright screenshots against real Oracle FCCORP_BKP-calculated data (documented in `03-03-SUMMARY.md`). This verification independently re-ran the same API calls (list + detail + auth + IDOR checks) against the still-running stack and confirmed matching, non-fabricated results — corroborating rather than merely trusting the SUMMARY's narrative.

### Gaps Summary

No gaps. All 4 ROADMAP success criteria, all 4 REQUIREMENTS.md items (CMP-01..04), and all must_haves declared in the 03-01/03-02/03-03 PLAN frontmatter are verified against actual running code and live data — not just SUMMARY.md claims. The one apparent scope reduction (resumo "por lote importado") is a documented, user-approved decision made during context-gathering (03-CONTEXT.md D-09, 03-DISCUSSION-LOG.md), not an executor shortcut, and is covered by REQUIREMENTS.md's own "e/ou" (either/or) phrasing for CMP-04.

---

_Verified: 2026-07-02T14:15:22Z_
_Verifier: Claude (gsd-verifier)_
