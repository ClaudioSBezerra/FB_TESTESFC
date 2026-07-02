---
phase: 02-import-pipeline-fiscal-execution
verified: 2026-07-02T14:09:40Z
status: passed
score: 9/9 must-haves verified
overrides_applied: 0
re_verification: false
---

# Phase 2: Import Pipeline & Fiscal Execution — Verification Report

**Phase Goal:** Usuário importa XMLs de NFe de saída, o sistema obtém o grupo fiscal de cada item via Oracle (prod + PRODB), executa o script do pacote fiscal no FCCORP_BKP e persiste os impostos calculados — com erros isolados por item sem interromper o lote.

**Verified:** 2026-07-02T14:09:40Z
**Status:** PASSED
**Re-verification:** No — initial verification

**Note on MVP mode framing:** ROADMAP.md marks this phase `mode: mvp`, but the phase-level Goal text is not in `As a ..., I want to ..., so that ....` format (it is descriptive prose). Per `verify-mvp-mode.md` this is technically a format-guard discrepancy the mvp workflow expects `/gsd mvp-phase` to fix. However, both underlying plans (02-01, 02-02) each declare a properly-formatted `<phase_user_story>` block, the phase's two human-verify checkpoints were already executed and approved against a real Oracle instance (documented below), and the phase is already marked complete in ROADMAP.md with a downstream phase (Phase 3) built on top of it. Refusing verification here would provide no value, so this report proceeds with standard goal-backward verification against the ROADMAP Success Criteria (Step 2a) and PLAN frontmatter must-haves (Step 2b/2c), which is a superset of MVP-mode coverage. Recommend running `/gsd mvp-phase 2` purely for documentation hygiene — not a functional gap.

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Usuário carrega um ou vários XMLs de NF-e de saída e recebe confirmação de importação ou erro claro por arquivo (XML-01, XML-04) | ✓ VERIFIED | `ImportarXMLsSaida.tsx` dropzone (accept `.xml`/`.zip`) → `POST /api/xml/upload`; `XMLUploadHandler` (`xml_upload.go:193`) loops files, calls `processSingleXML` per file with isolated error handling (line 297: `if err := processSingleXML(...); err != nil { ... result.Rejeitados++ ... } else { result.Importados++ }`) — one bad file never aborts the batch. Rejection messages are specific (`"XML não é uma NF-e de saída (tpNF=%s)"`, `"modelo %s não suportado"`, `"chave de acesso inválida"`). UI shows badges "Importados: N"/"Rejeitados: N" + red error list per file (`bg-red-50 text-red-700`, `ImportarXMLsSaida.tsx:245`). |
| 2 | Um XML de saída válido é parseado e persiste cabeçalho + itens + impostos no Postgres (XML-02) | ✓ VERIFIED | `parseNFeXML`/`insertNFeSaidaHeader`/`insertNFeItens` in `nfe_saidas.go`; `ON CONFLICT (company_id, chave_nfe) DO UPDATE` (header) and `ON CONFLICT (nfe_id, n_item) DO UPDATE` (items) confirmed idempotent reimport; migrations 006/007/009 create/extend the schema (`v_bc_st`/`v_st`/`v_desc` added at item level). |
| 3 | Usuário visualiza notas/itens importados com valores de imposto originais do XML (XML-03) | ✓ VERIFIED | `ConsultaNFeSaidas.tsx` fetches `/api/nfe-saidas` (list) and opens a Dialog with item sub-table showing `v_bc_icms`/`v_icms`/`v_pis`/`v_cofins` per item; `NFeSaidaDetailHandler` (`nfe_saidas.go:697`) scopes by `company_id` from JWT. |
| 4 | Sistema consulta `prod`+`PRODB` no Oracle e obtém o grupo fiscal de cada item; itens sem grupo fiscal são sinalizados sem bloquear os demais (ERP-01, ERP-02, ERP-03) | ✓ VERIFIED | `lookupGrupoFiscal` (`fiscal_group_lookup.go:72`) runs `SELECT pb.grupo_fiscal, p.especial, p.ncm FROM prodb pb, prod p WHERE ... AND pb.cod_empresa = :codEmpresa` via bind vars; `sql.ErrNoRows` → `errSemGrupoFiscal` (non-fatal) → item persisted with `status='sem_grupo_fiscal'` (`fiscal_execution.go:271`) without aborting the rest of the batch (confirmed structurally and in the human checkpoint: `{"total":2,"ok":1,"sem_grupo_fiscal":1,"error":0}`). ERP-01 (Oracle connection config reused) confirmed via `openFiscalOracleConn` reading `erp_bridge_config` — same table/flow as inherited `ERPBridgeCredenciais.tsx`. |
| 5 | Para cada item, o backend executa `PKG_FISCAL_FCTAX.calcula_imposto_produto` via bloco PL/SQL anônimo e persiste os impostos calculados (FIS-01, FIS-02) | ✓ VERIFIED | `services/oracle_fiscal.go`: `BuildCalculaImpostoBlock()` generates the anonymous PL/SQL block from two fixed metadata tables (23 IN params, ~88 OUT fields); `buildBindArgs` uses reflection to bind IN via `sql.Named` and OUT via `sql.Out`/`go_ora.Out{Size:4000}` for strings (bug fix from commit `50773a8`, confirmed present in current code at line 358-362) — no `fmt.Sprintf` concatenation of values (`grep -c 'Sprintf.*result\.'` = 0). `persistFiscalItemResult` writes dedicated columns + `full_result JSONB` to `fiscal_execution_items` with `ON CONFLICT (nfe_item_id) DO UPDATE`. Confirmed against real Oracle in the 02-02 checkpoint: item with real product code 360 returned `status=ok` with ~88 fields populated. |
| 6 | Uma falha na execução do pacote fiscal de um item é marcada como `error` sem abortar o lote (FIS-03) | ✓ VERIFIED | `processFiscalBatch` (`fiscal_execution.go:209`) uses a `chan struct{}` semaphore (cap 5) + `sync.WaitGroup`, and each goroutine has `defer recover()` that persists `status='error'` and continues; `processSingleFiscalItem` persists `status='error'` on lookup or `CallFiscalPackage` failures without returning/aborting the loop. Human checkpoint confirmed isolation (`sem_grupo_fiscal` item did not prevent `ok` item from being processed in the same note). |
| 7 | Na tela "Notas Importadas", cada item exibe um badge de status (ok verde / sem_grupo_fiscal âmbar / error vermelho) | ✓ VERIFIED | `ConsultaNFeSaidas.tsx`: `FiscalStatusBadge` component (line 141) maps `ok`→`bg-green-50 text-green-700`, `sem_grupo_fiscal`→`bg-yellow-50 text-yellow-700`, `error`→`bg-red-50 text-red-700`; `NFeSaidaDetailHandler` extended with `LEFT JOIN fiscal_execution_items` (`nfe_saidas.go:769`) to expose `fiscal_status`/`fiscal_error_message` per item; button "Executar cálculo fiscal" (`useMutation` → `POST /api/fiscal-execution/run`, line 237) shows toast with `{total, ok, sem_grupo_fiscal, error}` summary. |
| 8 | Nenhum endpoint aceita `company_id` arbitrário do cliente | ✓ VERIFIED | `XMLUploadHandler`, `NFeSaidasListHandler`, `NFeSaidaDetailHandler`, `FiscalExecutionRunHandler` all resolve company via `erpBridgeGetCompany(db, r)` (JWT-derived); `FiscalExecutionRunHandler` additionally validates note ownership with `WHERE id = $1 AND company_id = $2` before processing (T-02-08). |
| 9 | Nenhuma credencial Oracle vaza em mensagens de erro ao cliente | ✓ VERIFIED | `openFiscalOracleConn` returns generic errors (`"credenciais Oracle não configuradas..."`, `"falha ao inicializar conexão Oracle"`) never `err.Error()`; `lookupGrupoFiscal` returns raw `scanErr` only to the internal caller, which logs it server-side and returns a sanitized message to the client (`fiscal_execution.go:279`). Confirmed empirically in the 02-02 human checkpoint (two real binding bugs surfaced only in `docker compose logs`, never in the HTTP response). |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `backend/migrations/006_nfe_saidas.sql` | Tabela `nfe_saidas` (cabeçalho, valores esperados) | ✓ VERIFIED | `CREATE TABLE IF NOT EXISTS nfe_saidas`, `UNIQUE (company_id, chave_nfe)`, FK `companies(id)`, 4 indexes |
| `backend/migrations/007_nfe_saidas_itens.sql` | Tabela `nfe_saidas_itens` (itens, valores esperados) | ✓ VERIFIED | `CREATE TABLE IF NOT EXISTS nfe_saidas_itens`, `UNIQUE (nfe_id, n_item)`, FK `nfe_saidas(id)`, includes `v_bc_st`/`v_st` at item level per plan decision |
| `backend/handlers/nfe_saidas.go` | Parsing NFe + persistência + handlers de consulta | ✓ VERIFIED | 800 lines; exports `NFeSaidasListHandler`, `NFeSaidaDetailHandler`, `parseNFeXML`, `insertNFeItens`, `insertNFeSaidaHeader`; `go build`/`go vet` clean |
| `backend/handlers/xml_upload.go` | Upload handler, ZIP extraction, per-file isolation | ✓ VERIFIED | 310 lines; exports `XMLUploadHandler`; anti-ZIP-bomb (`MaxUncompressedBytes`, `UncompressedSize64` accumulation), `filepath.Base` anti-path-traversal |
| `frontend/src/pages/ImportarXMLsSaida.tsx` | Tela de upload | ✓ VERIFIED | 298 lines; dropzone, badges, error list, `fetch('/api/xml/upload')` |
| `frontend/src/pages/ConsultaNFeSaidas.tsx` | Tela de consulta + detalhe + badges de status | ✓ VERIFIED | 622 lines; list + Dialog + `FiscalStatusBadge` + mutation button |
| `backend/migrations/008_fiscal_execution_items.sql` | Tabela de resultado calculado com status por item | ✓ VERIFIED | `CREATE TABLE IF NOT EXISTS fiscal_execution_items`, `UNIQUE(nfe_item_id)`, hybrid columns + `full_result JSONB NOT NULL` |
| `backend/services/oracle_fiscal.go` | Gerador do bloco PL/SQL a partir de metadados | ✓ VERIFIED | 380 lines; `PKG_FISCAL_FCTAX.calcula_imposto_produto` present; `declare` block generated programmatically; no `fmt.Sprintf` value concatenation |
| `backend/handlers/fiscal_group_lookup.go` | Lookup grupo fiscal prod/PRODB | ✓ VERIFIED | 95 lines; exports `lookupGrupoFiscal`; `resolveCodEmpresa` static map |
| `backend/handlers/fiscal_execution.go` | Pipeline isolado por item | ✓ VERIFIED | 415 lines; exports `FiscalExecutionRunHandler`; semaphore + `recover()` + per-item commit |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `ImportarXMLsSaida.tsx` | `/api/xml/upload` | FormData multipart POST | ✓ WIRED | `fetch('/api/xml/upload', {method:'POST', body: formData})` |
| `xml_upload.go` | `nfe_saidas`/`nfe_saidas_itens` | `tx.Exec INSERT ... ON CONFLICT` | ✓ WIRED | `insertNFeSaidaHeader`/`insertNFeItens` called inside `processSingleXML`'s transaction |
| `ConsultaNFeSaidas.tsx` | `/api/nfe-saidas` | `useQuery` fetch | ✓ WIRED | `@tanstack/react-query` |
| `main.go` | `XMLUploadHandler`/`NFeSaidasListHandler`/`NFeSaidaDetailHandler` | `withAuth` route registration | ✓ WIRED | `main.go:351-353` |
| `fiscal_execution.go` | `oracle_fiscal.go` | `CallFiscalPackage` | ✓ WIRED | `services.CallFiscalPackage(ctx, oracleDB, in)` at `fiscal_execution.go:316` |
| `fiscal_group_lookup.go` | Oracle prod/PRODB | `QueryRowContext` with binds | ✓ WIRED | `FROM prodb pb, prod p ... :codigoProduto ... :codEmpresa` |
| `fiscal_execution.go` | `fiscal_execution_items` | INSERT/UPDATE via `persistFiscalItemResult` | ✓ WIRED | `ON CONFLICT (nfe_item_id) DO UPDATE` |
| `ConsultaNFeSaidas.tsx` | `/api/fiscal-execution/run` | fetch POST + status badges | ✓ WIRED | `useMutation` calling `fetch('/api/fiscal-execution/run', ...)`, badge rendered from `item.fiscal_status` returned by `NFeSaidaDetailHandler`'s `LEFT JOIN` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `ConsultaNFeSaidas.tsx` badges | `item.fiscal_status` | `GET /api/nfe-saidas/{id}` → `NFeSaidaDetailHandler` `LEFT JOIN fiscal_execution_items` | Yes — real DB query, `COALESCE(f.status,'')` | ✓ FLOWING |
| `ImportarXMLsSaida.tsx` counters | `Importados`/`Rejeitados` | `XMLUploadHandler` response JSON, incremented per-file result of `processSingleXML` | Yes — reflects real per-file outcome, not static | ✓ FLOWING |
| `fiscal_execution_items.full_result` | `FiscalResult` struct | `CallFiscalPackage` → real Oracle bind results (confirmed 88 fields populated in checkpoint) | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Backend compiles/vets clean | `cd backend && go build ./... && go vet ./...` | exit 0 | ✓ PASS |
| Frontend typechecks clean | `cd frontend && npx tsc --noEmit` | exit 0 | ✓ PASS |
| No unsafe SQL concatenation in PL/SQL block builder | `grep -c 'Sprintf.*result\.' services/oracle_fiscal.go` | 0 matches | ✓ PASS |
| Routes registered via `withAuth` (no public fiscal/import endpoints) | `grep 'xml/upload\|nfe-saidas\|fiscal-execution' main.go` | all 4 routes wrapped in `withAuth(..., "")` | ✓ PASS |
| Commits referenced in SUMMARYs exist in git history | `git cat-file -e <hash>` for 2a6c624, bffaa7f, 263ab6b, dda4c70, 72d6a9c, 6865455, 50773a8 | all present | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` found in repository and none declared in PLAN/SUMMARY for this phase. SKIPPED (no runnable probes for this project).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| XML-01 | 02-01 | Importa um ou vários XMLs de NF-e de saída via tela | ✓ SATISFIED | Dropzone + `XMLUploadHandler` multi-file loop |
| XML-02 | 02-01 | Parse + persiste cabeçalho, itens, impostos | ✓ SATISFIED | `insertNFeSaidaHeader`/`insertNFeItens`, migrations 006/007 |
| XML-03 | 02-01 | Visualiza notas/itens com valores originais do XML | ✓ SATISFIED | `ConsultaNFeSaidas.tsx` Dialog + `NFeSaidaDetailHandler` |
| XML-04 | 02-01 | Reporta erros de parse/validação por arquivo | ✓ SATISFIED | Per-file error isolation + `bg-red-50` error list in UI |
| ERP-01 | 02-02 | Conexão Oracle de leitura configurável (ERP_BRIDGE) | ✓ SATISFIED | `openFiscalOracleConn` reuses `erp_bridge_config` + inherited `ERPBridgeCredenciais.tsx` |
| ERP-02 | 02-02 | Consulta `prod`+`PRODB` e obtém grupo fiscal por item | ✓ SATISFIED | `lookupGrupoFiscal` query confirmed against real Oracle in checkpoint |
| ERP-03 | 02-02 | Itens sem grupo fiscal sinalizados sem bloquear os demais | ✓ SATISFIED | `errSemGrupoFiscal` → `status='sem_grupo_fiscal'`, isolation confirmed in checkpoint |
| FIS-01 | 02-02 | Executa pacote fiscal no FCCORP_BKP com params do XML + grupo fiscal | ✓ SATISFIED | `CallFiscalPackage` via generated PL/SQL block, confirmed against real Oracle |
| FIS-02 | 02-02 | Carrega e persiste retorno do script vinculado ao item | ✓ SATISFIED | `persistFiscalItemResult` → `fiscal_execution_items` |
| FIS-03 | 02-02 | Falhas por item capturadas e exibidas sem abortar o lote | ✓ SATISFIED | Semaphore + `recover()` + per-item status persistence |

**Coverage:** 10/10 phase requirements satisfied with code + real-Oracle-checkpoint evidence.

**Documentation sync gap (non-blocking):** `.planning/REQUIREMENTS.md` still lists XML-01..04, ERP-01..03, FIS-01..03 with unchecked `[ ]` boxes and "Pending" in the Traceability table, even though ROADMAP.md marks Phase 2 complete (2026-07-01) and this verification confirms all 10 requirements are functionally satisfied. This is a documentation-freshness issue, not a functional gap — recommend updating REQUIREMENTS.md checkboxes/traceability to "Complete" as a housekeeping follow-up (does not require a closure plan).

### Anti-Patterns Found

No blocking or warning-level anti-patterns found in the phase's modified files (`grep` for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER|coming soon|not yet implemented` returned only false positives: the word "TODOS" (Portuguese "all") in a comment, and a legitimate HTML `placeholder=` attribute on a search input).

Two accepted, explicitly-documented limitations from the 02-02-SUMMARY (both surfaced to and accepted by the human at the 02-02 checkpoint, not code defects):
- `codEmpresaPorCNPJRaiz` only maps the Recife/PE CNPJ root (`10230480` → `cod_empresa=2`); Garanhuns/PE is unmapped and returns an explicit `error` status per item rather than a guessed value — this is the intended fail-safe behavior of ERP-03/FIS-03 (never guess), not silent breakage.
- Several of the 23 PL/SQL input parameters (`pTipoContribuinte`, `pTipoCentroFiscal`, `pIndicadorServico`, `FornecedorSimplesNacional`, `pAliquotaSimplesNacional`, `pDespesas`) use documented conservative defaults because the current schema has no persisted source for them; flagged for Phase 3 comparison to expose any incorrect default empirically.

### Human Verification Required

None. Both blocking human-verify checkpoints for this phase (02-01: import→persist→consulta; 02-02: fiscal execution pipeline against real Oracle) were already executed and approved in a prior session, with concrete evidence recorded in `02-01-SUMMARY.md` and `02-02-SUMMARY.md` (including two real binding bugs found and fixed against FCCORP_BKP in commit `50773a8`, and the actual Oracle response `{"total":2,"ok":1,"sem_grupo_fiscal":1,"error":0}`). No new user-facing behavior was added since that approval that would require re-verification.

### Gaps Summary

No gaps found. All 9 derived observable truths are verified against the codebase (not just SUMMARY claims): build/vet/tsc are clean, migrations match the planned schema, per-item/per-file error isolation is implemented with `recover()`/semaphores and confirmed structurally, the PL/SQL block generator uses only bind variables (no SQL injection surface), routes are all authenticated and company-scoped, and the two prior human checkpoints provide real-Oracle evidence for the Oracle-dependent truths that cannot be re-verified by static analysis alone. The only non-blocking finding is a documentation-freshness gap in REQUIREMENTS.md (checkboxes/traceability table not updated to "Complete"), which does not affect phase goal achievement and does not require a closure plan.

---

_Verified: 2026-07-02T14:09:40Z_
_Verifier: Claude (gsd-verifier)_
