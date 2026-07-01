---
phase: 02-import-pipeline-fiscal-execution
plan: 02
subsystem: fiscal-execution
tags: [go-ora, oracle, plsql, reflection, react-query, tanstack-query]

# Dependency graph
requires:
  - phase: 02-import-pipeline-fiscal-execution
    provides: "nfe_saidas / nfe_saidas_itens (Postgres schema — valores esperados do XML), tela ConsultaNFeSaidas.tsx"
provides:
  - "fiscal_execution_items (Postgres schema — resultado calculado pelo pacote fiscal, por item)"
  - "services/oracle_fiscal.go — gerador de bloco PL/SQL anônimo + CallFiscalPackage"
  - "handlers/fiscal_group_lookup.go — lookupGrupoFiscal (prod/PRODB) + resolveCodEmpresa"
  - "POST /api/fiscal-execution/run"
  - "Badge de status por item + botão 'Executar cálculo fiscal' em ConsultaNFeSaidas.tsx"
affects:
  - "Fase 3 (comparação visual esperado-vs-calculado) — lê fiscal_execution_items.full_result/colunas dedicadas como o valor CALCULADO"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Bloco PL/SQL anônimo com OUT escalares 'achatados', gerado via reflection a partir de uma única tabela de metadados (fiscalOutFields) — nenhum valor concatenado no SQL, só sql.Named/sql.Out"
    - "Isolamento de erro por item em lote Oracle: semáforo chan struct{} cap 5 + defer recover() + timeout de contexto 15s/item, commit Postgres por item (nunca transação única do lote)"
    - "cod_empresa (PRODB) resolvido por mapa estático Go (raiz do CNPJ do emitente → int), sem tabela nova"

key-files:
  created:
    - backend/migrations/008_fiscal_execution_items.sql
    - backend/migrations/009_nfe_saidas_itens_desconto.sql
    - backend/services/oracle_fiscal.go
    - backend/handlers/fiscal_group_lookup.go
    - backend/handlers/fiscal_execution.go
  modified:
    - backend/handlers/nfe_saidas.go
    - backend/main.go
    - frontend/src/pages/ConsultaNFeSaidas.tsx

key-decisions:
  - "cod_empresa resolvido pela raiz do CNPJ do emitente (8 primeiros dígitos), mapa estático — só a raiz da filial Recife/PE (10230480, fonte: exemplo do script de teste do pacote fiscal) está confirmada nesta execução; Garanhuns/PE (cod_empresa=1) fica pendente do checkpoint humano com Oracle real (sem XML real disponível neste ambiente para confirmar)"
  - "Vários dos 23 parâmetros de calcula_imposto_produto não têm fonte de dado persistida hoje (indIEDest, CRT/Simples Nacional do emitente, código de 'centro fiscal') — usados defaults conservadores documentados em fiscal_execution.go, a validar/refinar no checkpoint humano e na Fase 3"
  - "v_desc adicionado a nfe_saidas_itens via migração aditiva 009 (Rule 2) — o pacote fiscal exige desconto por item para a base de ICMS ficar correta; o Plano 02-01 parseava vDesc do XML mas não persistia"
  - "pDespesas fixado em 0 por item — NF-e não carrega despesas acessórias no nível de item (só vOutro no cabeçalho); overhead de rateio proporcional não implementado nesta fase"

requirements-completed: [ERP-01, ERP-02, ERP-03, FIS-01, FIS-02, FIS-03]

# Metrics
duration: ~45min
completed: 2026-07-01
---

# Phase 02 Plan 02: Fiscal Execution Summary

Pipeline de execução fiscal item a item — lookup de grupo fiscal em prod/PRODB, chamada de PKG_FISCAL_FCTAX.calcula_imposto_produto via bloco PL/SQL anônimo gerado por reflection a partir de uma única tabela de metadados de ~88 campos, persistência isolada por item em fiscal_execution_items e badges de status na tela de consulta. **Checkpoint humano pendente — não verificado nesta execução por falta de acesso a uma instância Oracle real (prod/PRODB/FCCORP_BKP).**

## Performance

- **Duration:** ~45 min
- **Tasks:** 3 de 4 (checkpoint humano bloqueante pendente)
- **Files modified:** 8 (5 criados, 3 modificados)

## Accomplishments

- `services/oracle_fiscal.go`: os 23 parâmetros de entrada e ~88 campos de saída de `PKG_FISCAL_FCTAX.calcula_imposto_produto` estão centralizados em duas tabelas de metadados (`fiscalInParams`/`fiscalOutFields`); `BuildCalculaImpostoBlock()` gera a string do bloco PL/SQL anônimo e `buildBindArgs` usa reflection sobre essas mesmas tabelas para montar tanto os binds `sql.Named` (IN) quanto os destinos `sql.Out` (OUT) em um único `*FiscalResult` — nenhum valor é concatenado na string SQL (T-02-02).
- `handlers/fiscal_group_lookup.go`: `lookupGrupoFiscal` executa a query confirmada em `prod`/`PRODB` com binds `:codigoProduto`/`:codEmpresa`, traduzindo `sql.ErrNoRows` em `errSemGrupoFiscal` (não fatal). `resolveCodEmpresa` deriva o `cod_empresa` da raiz do CNPJ do emitente via mapa estático — falha explicitamente (nunca adivinha) quando a raiz não está mapeada.
- `handlers/fiscal_execution.go`: `FiscalExecutionRunHandler` valida a nota contra a company do JWT, abre uma conexão Oracle dedicada (`SetMaxOpenConns(5)`), e processa os itens com semáforo de concorrência (cap 5) + `defer recover()` + timeout de 15s por item. Cada item persiste seu próprio status (`ok`/`error`/`sem_grupo_fiscal`) em `fiscal_execution_items` — uma falha em um item nunca aborta os demais. Retorna `{total, ok, sem_grupo_fiscal, error}`.
- `migrations/008_fiscal_execution_items.sql`: tabela híbrida (colunas dedicadas para os campos usados na Fase 3 + `full_result JSONB` para o retorno completo), `UNIQUE(nfe_item_id)`.
- `ConsultaNFeSaidas.tsx`: botão "Executar cálculo fiscal" no Dialog de detalhe (mutation `POST /api/fiscal-execution/run`, toast com o resumo), nova coluna "Status" na sub-tabela de itens com badge de 3 estados (verde/âmbar/vermelho) + tooltip, `NFeSaidaDetailHandler` estendido com `LEFT JOIN fiscal_execution_items` para expor o status por item.

## Task Commits

Each task was committed atomically:

1. **Task 1: Migração 008 + oracle_fiscal.go** - `dda4c70` (feat)
2. **Task 2: fiscal_group_lookup.go + fiscal_execution.go + rota** - `72d6a9c` (feat)
3. **Task 3: Badges de status + botão "Executar cálculo fiscal"** - `6865455` (feat)
4. **Checkpoint: Verificação do pipeline de execução fiscal** - **APROVADO** contra Oracle real (ver "Next Phase Readiness")
5. **Fix pós-checkpoint: binds OUT string + tipos IdRegraCalculo\*** - `50773a8` (fix)

**Plan metadata:** (este commit) `docs: complete plan`

## Files Created/Modified

- `backend/migrations/008_fiscal_execution_items.sql` - Tabela de resultado calculado, modelo híbrido (colunas dedicadas + JSONB), status por item
- `backend/migrations/009_nfe_saidas_itens_desconto.sql` - Adiciona `v_desc` a `nfe_saidas_itens` (Rule 2 — desconto por item ausente, necessário para o pacote fiscal)
- `backend/services/oracle_fiscal.go` - Metadados dos 23 params IN + ~88 campos OUT, gerador do bloco PL/SQL, `CallFiscalPackage`
- `backend/handlers/fiscal_group_lookup.go` - `lookupGrupoFiscal` (prod/PRODB) + `resolveCodEmpresa` (mapa estático por raiz de CNPJ)
- `backend/handlers/fiscal_execution.go` - `FiscalExecutionRunHandler`, pipeline isolado por item, persistência em `fiscal_execution_items`
- `backend/handlers/nfe_saidas.go` - `insertNFeItens` agora persiste `v_desc`; `NFeSaidaDetailHandler` com `LEFT JOIN fiscal_execution_items`
- `backend/main.go` - Rota `POST /api/fiscal-execution/run` via `withAuth`
- `frontend/src/pages/ConsultaNFeSaidas.tsx` - Botão de execução, badge de status por item, tooltip

## Decisions Made

- **cod_empresa por raiz de CNPJ:** ver `key-decisions` no frontmatter — apenas a filial Recife/PE está mapeada nesta execução (fonte: exemplo do script de teste do pacote fiscal). Notas de outras filiais retornam status `error` explícito por item até o mapa ser completado.
- **Defaults documentados para parâmetros sem fonte de dado:** `pTipoContribuinte`, `pTipoCentroFiscal`, `pIndicadorServico`, `FornecedorSimplesNacional`, `pAliquotaSimplesNacional`, `pDespesas` não têm coluna persistida hoje — usados valores conservadores com comentário explícito em `fiscal_execution.go`, para revisão no checkpoint humano e na Fase 3 (a comparação esperado-vs-calculado vai expor rapidamente qualquer default incorreto).
- **v_desc por item (migração 009):** adicionado porque sua ausência quebraria a correção estrutural do FIS-01/FIS-02 (base de ICMS calculada sistematicamente errada sem o desconto do item).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] `v_desc` (desconto por item) ausente em `nfe_saidas_itens`**
- **Found during:** Task 2, ao mapear os campos de `FiscalInput` (`pDesconto`) a partir dos dados já persistidos pelo Plano 02-01
- **Issue:** o Plano 02-01 parseia `vDesc` do XML (`prod.VDesc`) mas nunca o persiste — sem esse valor, `pDesconto` seria sempre 0, produzindo uma base de cálculo de ICMS estruturalmente incorreta em toda comparação da Fase 3
- **Fix:** migração aditiva `009_nfe_saidas_itens_desconto.sql` (`ALTER TABLE ... ADD COLUMN IF NOT EXISTS v_desc`); `insertNFeItens` atualizado para persistir `toDecimal(d.Prod.VDesc)`
- **Files modified:** `backend/migrations/009_nfe_saidas_itens_desconto.sql`, `backend/handlers/nfe_saidas.go`
- **Verification:** `go build ./... && go vet ./...` limpo; coluna aditiva com `DEFAULT 0`, não quebra dados já importados
- **Committed in:** `dda4c70` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical — Rule 2)
**Impact on plan:** Necessário para a correção estrutural do valor calculado que a Fase 3 vai comparar. Sem escopo além do estritamente exigido pelo pipeline desta fase.

## Issues Encountered

- **Ausência de acesso a Oracle real e a um XML real da Ferreira Costa** neste ambiente de execução: impediu (a) testar `CallFiscalPackage`/`lookupGrupoFiscal` contra `prod`/`PRODB`/`FCCORP_BKP` de verdade, e (b) confirmar a raiz de CNPJ da filial Garanhuns/PE para completar `codEmpresaPorCNPJRaiz`. Ambos ficam documentados como pendências explícitas do checkpoint humano (ver abaixo) — o código está pronto para ser testado assim que o ambiente com Oracle estiver disponível.
- **Mapeamento de negócio de alguns dos 23 parâmetros de entrada** (`pTipoContribuinte`, `pTipoCentroFiscal`, `FornecedorSimplesNacional`) não tem fonte de dado persistida no schema atual — resolvido com defaults documentados (ver Decisions), não um bloqueio de código, mas uma limitação de precisão que só pode ser validada contra o pacote fiscal real.

## Known Stubs

Nenhum stub de UI (nenhum dado mockado renderizado). Limitações de dados de entrada do pacote fiscal (defaults para parâmetros sem fonte persistida, mapa `cod_empresa` incompleto) estão documentadas acima em "Decisions Made"/"Issues Encountered" — não bloqueiam a compilação nem a funcionalidade de isolamento de erro por item, mas podem produzir resultados calculados incorretos para filiais/cenários não cobertos pelos defaults, a ser corrigido durante o checkpoint humano e a Fase 3.

## Threat Flags

Nenhuma superfície nova além do `<threat_model>` do plano (T-02-02, T-02-06, T-02-07, T-02-08, T-02-09, T-02-SC) — todas com mitigação implementada:
- T-02-02: bloco PL/SQL 100% estático/gerado de metadados fixos, valores só via bind (verificado por grep no verify do Task 1).
- T-02-06: nenhum `err.Error()` bruto do Oracle é propagado ao cliente (`openFiscalOracleConn`, `lookupGrupoFiscal`, `CallFiscalPackage` — mensagens sanitizadas nos handlers).
- T-02-07: `SetMaxOpenConns(5)` + semáforo cap 5 + timeout 15s/item.
- T-02-08: `nfe_id` validado contra `company_id` do JWT antes de qualquer trabalho; nenhum handler aceita `company_id` do cliente.
- T-02-09: nenhuma tarefa executa INSERT/UPDATE/DELETE contra tabelas Oracle (`prod`/`PRODB`/`FCCORP_BKP`) — apenas SELECT e chamada de função.

## User Setup Required

None - nenhuma configuração de serviço externo nova (reaproveita `erp_bridge_config` da Fase 1).

## Self-Check: PASSED

- FOUND: backend/migrations/008_fiscal_execution_items.sql
- FOUND: backend/migrations/009_nfe_saidas_itens_desconto.sql
- FOUND: backend/services/oracle_fiscal.go
- FOUND: backend/handlers/fiscal_group_lookup.go
- FOUND: backend/handlers/fiscal_execution.go
- FOUND commit dda4c70 (Task 1 — migrações + oracle_fiscal.go)
- FOUND commit 72d6a9c (Task 2 — handlers + rota)
- FOUND commit 6865455 (Task 3 — frontend + LEFT JOIN)
- `cd backend && go build ./... && go vet ./...` — exit 0
- `cd frontend && npx tsc --noEmit` — exit 0 (sem erros)

## Next Phase Readiness

**CHECKPOINT HUMANO APROVADO em 2026-07-01**, com Oracle real (Ferreira Costa, DSN `10.131.1.118:1521/FCCORP`, usuário `fcosta`) e um XML sintético estruturalmente válido (sem XML real de venda disponível, mas com CNPJ real da filial Recife/PE — `10230480001536` — e produto real com grupo fiscal configurado — código `360`):

1. ✅ "Testar Conexão" (Configurações → Credenciais ERP) retornou sucesso contra o Oracle real.
2. ✅ `docker compose up --build --force-recreate` — migrações 008 e 009 executadas limpo.
3. ✅ "Executar cálculo fiscal" rodado via API contra a nota de teste — resposta `{"total":2,"ok":1,"sem_grupo_fiscal":1,"error":0}`.
4. ✅ Item com produto real (360) retornou `status=ok` com os ~88 campos do pacote fiscal persistidos em `fiscal_execution_items.full_result` (bloco clássico + Reforma Tributária/IBS/CBS); item com produto fictício (PROD002) retornou `sem_grupo_fiscal` com mensagem clara — **isolamento por item confirmado**, nenhum erro abortou o lote.
5. **Dois bugs de binding encontrados e corrigidos nesta sessão** (só apareciam contra o Oracle real, ver commit `50773a8`):
   - `sql.Out` genérico do `database/sql` passa `size=0` ao driver go-ora para binds OUT de string → `MaxLen=0` → `ORA-06502 buffer too small`. Corrigido com `go_ora.Out{Dest, Size: 4000}` (tipo nativo do driver) para todos os campos string do resultado.
   - `IdRegraCalculoIcms/PisCofins/Ipi/Ibs/Cbs` foram assumidos como `NUMBER` (float64) pelo nome do campo — são `VARCHAR2` no objeto real (ex: `"IVA_N_FC01PEPE1SNVRJNE6811810030002IC61"`) → `ORA-06502 character to number conversion error`. Corrigidos para `string`.
6. `cod_empresa` da filial Recife/PE (`10230480` → `2`) **confirmado correto** contra o Oracle real.
7. Nenhuma mensagem de erro exposta ao cliente vazou DSN/usuário/senha Oracle durante os testes (inclusive nos dois bugs acima, cujo erro completo só apareceu em `docker compose logs`, nunca na resposta HTTP).

**Pendências que sobrevivem ao checkpoint (não bloqueiam Fase 3, mas precisam de acompanhamento):**
- `codEmpresaPorCNPJRaiz` (`backend/handlers/fiscal_group_lookup.go`): só Recife/PE está mapeada. Garanhuns/PE (`cod_empresa=1`) continua sem confirmação — notas dessa filial retornam erro explícito por item até ser adicionada.
- Defaults de `pTipoContribuinte`/`pTipoCentroFiscal`/`pIndicadorServico`/`FornecedorSimplesNacional`/`pAliquotaSimplesNacional` em `fiscal_execution.go` ainda não foram validados contra um caso real de cada variação (Simples Nacional, prestação de serviço, etc.) — o teste realizado cobriu apenas o caminho "normal" (não-Simples, não-serviço). A comparação da Fase 3 vai expor rapidamente qualquer default incorreto quando dados reais divergentes aparecerem.
- Teste foi feito com XML sintético (não uma NF-e real de venda) — os valores de entrada (preço, desconto) são fictícios; o pacote aceitou e calculou normalmente, mas vale reimportar com uma nota real assim que disponível.

A Fase 2 está pronta para a Fase 3 (tela de comparação item a item, esperado-vs-calculado), que consumirá `fiscal_execution_items` como a fonte do valor calculado.

---
*Phase: 02-import-pipeline-fiscal-execution*
*Completed: 2026-07-01 (checkpoint humano aprovado contra Oracle real)*
