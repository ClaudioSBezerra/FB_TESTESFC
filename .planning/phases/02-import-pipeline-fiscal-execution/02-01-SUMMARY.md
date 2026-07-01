---
phase: 02-import-pipeline-fiscal-execution
plan: 01
subsystem: import-pipeline
tags: [xml-upload, nfe-saidas, postgres, react-dropzone, react-query]
requires: []
provides:
  - "nfe_saidas / nfe_saidas_itens (Postgres schema — valores esperados do XML)"
  - "POST /api/xml/upload"
  - "GET /api/nfe-saidas"
  - "GET /api/nfe-saidas/{id}"
  - "Tabs 'Importar XMLs' / 'Notas Importadas'"
affects:
  - "Fase 2 Plano 02 (execução fiscal) — lê nfe_saidas_itens como fonte de itens a recalcular"
  - "Fase 3 (comparação visual) — lê nfe_saidas/nfe_saidas_itens como gabarito (esperado)"
tech-stack:
  added:
    - "golang.org/x/text v0.34.0 (charmap/transform — parsing de encoding legado Windows-1252/ISO-8859-1)"
  patterns:
    - "Cópia seletiva de parsing NFe do FB_APU04 (structs SEFAZ inalteradas)"
    - "ON CONFLICT (company_id, chave_nfe) / (nfe_id, n_item) DO UPDATE — reimportação idempotente"
    - "Escopo por company_id sempre via erpBridgeGetCompany/GetEffectiveCompanyID — nunca aceito do cliente"
key-files:
  created:
    - backend/migrations/006_nfe_saidas.sql
    - backend/migrations/007_nfe_saidas_itens.sql
    - backend/handlers/nfe_saidas.go
    - backend/handlers/xml_upload.go
    - frontend/src/pages/ImportarXMLsSaida.tsx
    - frontend/src/pages/ConsultaNFeSaidas.tsx
  modified:
    - backend/go.mod
    - backend/go.sum
    - backend/main.go
    - frontend/src/lib/navigation.ts
    - frontend/src/App.tsx
decisions:
  - "Upload aceita apenas .xml e .zip (não .rar) — rardecode não estava no go.mod e não havia auditoria confirmada no RESEARCH.md para essa dependência nesta fase; reduz superfície de supply-chain sem perder cobertura funcional do XML-01"
  - "Upload síncrono (200 imediato com contadores importados/rejeitados) em vez do fluxo batch assíncrono com polling do FB_APU04 — plano não pediu tabela xml_upload_batches; simplifica para o volume esperado da Ferreira Costa"
  - "v_bc_st/v_st adicionadas a nível de item em nfe_saidas_itens (não existiam no schema original do FB_APU04) — necessário para a Fase 3 comparar ICMS-ST item a item"
metrics:
  duration: "~35min"
  completed: 2026-07-01
---

# Phase 02 Plan 01: Import Pipeline de XMLs de Saída Summary

Fatia vertical completa de importação de NF-e de saída: dropzone de upload (.xml/.zip) parseia e persiste cabeçalho + itens + impostos esperados no Postgres com idempotência via ON CONFLICT, e uma tela de consulta lista as notas com Dialog de detalhe item a item.

## What Was Built

- **Migrações 006/007**: `nfe_saidas` (cabeçalho, valores esperados do XML — ICMSTot + IBSCBSTot da Reforma) e `nfe_saidas_itens` (itens, incluindo `v_bc_st`/`v_st` por item — lacuna do FB_APU04 resolvida para permitir comparação de ICMS-ST item a item na Fase 3).
- **`handlers/nfe_saidas.go`**: structs de parsing NFe (nfeProc/infNFe/det/prod/detImposto/ICMS/PIS/COFINS/IPI/IBS/CBS) copiadas do FB_APU04 sem alteração (mapeamento SEFAZ é padrão de mercado); helpers `toDecimal`/`toNullDecimal`/`toNullSmallInt`; `parseNFeXML` com suporte a encoding legado (Windows-1252/ISO-8859-1) e wrapper `nfeProc` ausente; `insertNFeSaidaHeader` e `insertNFeItens` com `ON CONFLICT ... DO UPDATE` para reimportação idempotente; `NFeSaidasListHandler` (GET lista escopada por company) e `NFeSaidaDetailHandler` (GET cabeçalho + itens, valida ownership por company_id).
- **`handlers/xml_upload.go`**: `XMLUploadHandler` (POST) aceita um ou mais `.xml` ou um `.zip`, com mitigações anti-ZIP-bomb (`UncompressedSize64` acumulado, limite 8GB) e anti-path-traversal (`filepath.Base`), isolamento de erro por arquivo (um XML inválido não aborta o lote), validação de `tpNF=1` (rejeita entradas com mensagem clara) e modelo 55/65.
- **Rotas em `main.go`**: `POST /api/xml/upload`, `GET /api/nfe-saidas`, `GET /api/nfe-saidas/{id}` — todas via `withAuth(handler, "")`, nenhuma pública.
- **`ImportarXMLsSaida.tsx`**: dropzone `react-dropzone` (aceita `.xml`/`.zip`, 2GB), badges verde/vermelho de importados/rejeitados, lista de erros por arquivo, histórico de uploads da sessão. Sem campo de competência (fora de escopo do validador fiscal).
- **`ConsultaNFeSaidas.tsx`**: lista de notas via `@tanstack/react-query`, filtros client-side (cliente, intervalo de datas), cards totalizadores, Dialog de detalhe (`Secao`/`Linha`/`LinhaBRL`) com sub-tabela de itens mostrando valores esperados (base ICMS, vICMS, vPIS, vCOFINS) por item.
- **Navegação**: tabs "Importar XMLs" e "Notas Importadas" adicionadas em `modules.config.tabs`, e rotas React registradas em `App.tsx` dentro do bloco protegido (qualquer usuário autenticado, sem `AdminRoute`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Upload restrito a .xml/.zip (sem .rar)**
- **Found during:** Task 2 (leitura de `xml_upload.go` do FB_APU04, que usa `github.com/nwaples/rardecode/v2`)
- **Issue:** o pacote `rardecode/v2` não está no `go.mod` do FB_TESTESFC; adicioná-lo seria uma nova dependência de supply-chain sem cobertura de auditoria explícita no RESEARCH.md desta fase (apenas ZIP foi confirmado como mitigado).
- **Fix:** removido o branch `.rar` do handler de upload e do dropzone do frontend; mantido apenas `.xml`/`.zip`, cobrindo o requisito XML-01 sem introduzir dependência não auditada.
- **Files modified:** `backend/handlers/xml_upload.go`, `frontend/src/pages/ImportarXMLsSaida.tsx`
- **Commit:** bffaa7f, 263ab6b

**2. [Rule 1 - Bug] gofmt em nfe_saidas.go/main.go**
- **Found during:** Task 2, verificação pré-commit
- **Issue:** alinhamento de imports/colunas fora do padrão `gofmt`.
- **Fix:** `gofmt -w` aplicado; build/vet confirmados limpos após.
- **Files modified:** `backend/handlers/nfe_saidas.go`, `backend/main.go`
- **Commit:** bffaa7f

## Known Stubs

Nenhum stub — a fatia é funcional fim-a-fim (upload → parse → persistência → consulta) sem dados mockados. A ausência de coluna/badge de status de execução fiscal na sub-tabela de itens é intencional (fora de escopo deste plano — reservado para o Plano 02 desta fase).

## Threat Flags

Nenhuma superfície nova além do que já estava mapeado no `<threat_model>` do plano (T-02-01 a T-02-05, T-02-SC). Redução de superfície: `.rar` não foi implementado, então a extração RAR não existe nesta versão (menor superfície do que o threat model previa auditar).

## Self-Check: PASSED

- FOUND: backend/migrations/006_nfe_saidas.sql
- FOUND: backend/migrations/007_nfe_saidas_itens.sql
- FOUND: backend/handlers/nfe_saidas.go
- FOUND: backend/handlers/xml_upload.go
- FOUND: frontend/src/pages/ImportarXMLsSaida.tsx
- FOUND: frontend/src/pages/ConsultaNFeSaidas.tsx
- FOUND commit 2a6c624 (Task 1 — migrações + go.mod)
- FOUND commit bffaa7f (Task 2 — handlers + rotas)
- FOUND commit 263ab6b (Task 3 — telas + navegação)
- `cd backend && go build ./... && go vet ./...` — exit 0
- `cd frontend && npx tsc --noEmit` — exit 0 (sem erros)

## Next Steps

Checkpoint humano pendente (`checkpoint:human-verify`, gate="blocking"): subir `docker compose up --build --force-recreate` e validar upload→persistência→consulta com um XML real de NF-e de saída da Ferreira Costa, incluindo teste de rejeição de XML de entrada e idempotência de reimportação.
