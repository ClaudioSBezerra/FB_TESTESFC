# Phase 3: Visual Comparison Screen - Context

**Gathered:** 2026-07-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Uma tela nova e dedicada ("Comparação Fiscal") mostra, item a item, os impostos esperados (do XML, `nfe_saidas_itens`) versus os calculados pelo pacote fiscal (`fiscal_execution_items`), com divergências destacadas visualmente, filtro para ver só os itens divergentes, e um resumo de totais. Cobre CMP-01, CMP-02, CMP-03, CMP-04.

**Fora desta fase:** qualquer alteração no pipeline de importação (Fase 2) ou de execução fiscal (Fase 2) — esta fase só lê e apresenta dados já persistidos. Exportação CSV/Excel (AUTO-02, v2), suíte automatizada de testes (AUTO-01, v2), filtro granular por imposto específico (deferido), conceito de "lote de importação" como entidade (deferido).

</domain>

<decisions>
## Implementation Decisions

### Navegação / localização da tela
- **D-01:** Nova aba/página dedicada **"Comparação Fiscal"** na navegação — não expande o Dialog existente de "Notas Importadas". Justificativa do usuário: uma tela própria permite listar/filtrar itens de múltiplas notas de uma vez, o que o filtro (CMP-03) e o resumo (CMP-04) exigem.
- **D-02:** A tela abre direto numa **lista de itens** (todas as notas juntas, com coluna de nota/cliente), não um fluxo de "escolher a nota primeiro". Máxima flexibilidade para análise rápida.
- **D-03:** Clicar num item abre um **Dialog/modal de detalhe** (mesmo padrão já usado em `ConsultaNFeSaidas.tsx`) — não uma página própria por item.

### Colunas e agrupamento de impostos
- **D-04:** A lista principal mostra por padrão **só os pares 1:1 disponíveis**: Base/Valor ICMS, Base/Valor ICMS-ST, Base/Valor PIS, Base/Valor COFINS. Esses são os únicos campos que o XML também traz para comparar esperado vs. calculado item a item.
- **D-05:** Cada imposto ocupa **3 colunas separadas**: Esperado | Calculado | Diferença (não uma célula compacta "esperado → calculado").
- **D-06:** Critério de divergência (CMP-02) = **qualquer diferença ≠ 0** — sem tolerância de arredondamento no v1. É um validador fiscal; até 1 centavo pode importar. Pode ser revisitado se gerar ruído demais na prática (ver Deferred).
- **D-07:** O Dialog de detalhe tem uma **seção "Só calculado"** para os campos do pacote fiscal sem par no XML (DIFAL, FCP, IBS, CBS, e os demais dos ~90 campos de `fiscal_execution_items.full_result`) — só o valor calculado, sem coluna esperado/diferença, útil para auditoria/debug do pacote.

### Filtro e resumo
- **D-08:** O filtro "só divergentes" (CMP-03) é um **toggle simples**: item aparece se qualquer um dos 4 pares (ICMS/ICMS-ST/PIS/COFINS) tiver diferença ≠ 0. Não há filtro granular por imposto específico no v1 (ver Deferred).
- **D-09:** O resumo (CMP-04) aparece em duas granularidades: **cards globais no topo da tela** (respeitando o filtro/período atual) + **resumo por nota** dentro do Dialog de detalhe. Não existe conceito de "lote de importação" como entidade no schema hoje — decisão explícita de não introduzir essa entidade só para o resumo (evita reconstruir complexidade fora do escopo do projeto).
- **D-10:** Itens ainda sem cálculo fiscal concluído (`status` = `pending`/`sem_grupo_fiscal`/`error` em `fiscal_execution_items`) entram como uma **categoria própria "Não calculado"** — terceiro balde separado de OK/Divergente no resumo, e **não aparecem** no filtro "só divergentes" (não há o que comparar ainda).

### Claude's Discretion
- Larguras/responsividade exata da tabela (ela fica larga: 4 impostos × 3 colunas + colunas de identificação).
- Cores/ícones exatos de divergência — seguir o padrão já estabelecido em `FiscalStatusBadge` (`ConsultaNFeSaidas.tsx`: verde/âmbar/vermelho, `bg-X-50/text-X-700/border-X-200`).
- Paginação/virtualização da lista de itens, se o volume justificar.
- Mapeamento exato dos nomes de campo em `full_result` (JSONB) para labels amigáveis na seção "Só calculado" do Dialog.
- Confirmar durante research se DIFAL/FCP realmente não têm nenhum valor esperado equivalente em `nfe_saidas_itens` (schema atual não tem essas colunas por item) — se existir uma fonte no XML não capturada na Fase 2, é um achado a reportar, não a resolver nesta fase.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Schema de dados (esperado vs. calculado)
- `backend/migrations/007_nfe_saidas_itens.sql` — schema dos valores ESPERADOS por item (gabarito do XML)
- `backend/migrations/009_nfe_saidas_itens_desconto.sql` — coluna `v_desc` adicional a `nfe_saidas_itens`
- `backend/migrations/008_fiscal_execution_items.sql` — schema dos valores CALCULADOS por item (colunas dedicadas + `full_result` JSONB com os ~88 campos)

### Código backend a reaproveitar/integrar
- `backend/handlers/nfe_saidas.go` — `NFeSaidaDetailHandler` (já faz `LEFT JOIN fiscal_execution_items`), padrão de queries/handlers escopados por `company_id`
- `backend/services/oracle_fiscal.go` — `FiscalResult` struct, fonte única de nomes/tipos dos ~88 campos calculados (necessário para mapear `full_result` na seção "Só calculado")
- `backend/handlers/fiscal_execution.go` — contrato dos status `ok`/`error`/`sem_grupo_fiscal`

### Código frontend a reaproveitar
- `frontend/src/pages/ConsultaNFeSaidas.tsx` — `FiscalStatusBadge` (padrão de cores verde/âmbar/vermelho já aprovado), helpers `Secao`/`Linha`/`LinhaBRL`, Dialog de detalhe existente
- `frontend/src/lib/navigation.ts` — onde registrar a nova aba "Comparação Fiscal"

### Design contract da fase anterior (padrões visuais já aprovados)
- `.planning/phases/02-import-pipeline-fiscal-execution/02-UI-SPEC.md` — badges, cores, componentes aprovados na Fase 2 (6/6 dimensões)

### Planejamento do projeto
- `.planning/REQUIREMENTS.md` — CMP-01, CMP-02, CMP-03, CMP-04 (requisitos desta fase)
- `.planning/ROADMAP.md` — Goal e success criteria da Fase 3
- `.planning/PROJECT.md` — Core value do projeto (a própria tela desta fase)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `FiscalStatusBadge` (`ConsultaNFeSaidas.tsx`) — badge de 3 cores (verde/âmbar/vermelho) já usado pro status de execução fiscal; mesmo padrão visual serve para "OK"/"Divergente"/"Não calculado" nesta fase.
- `Secao`/`Linha`/`LinhaBRL` — helpers de layout do Dialog de detalhe já estabelecidos; reaproveitar para o novo Dialog de detalhe item a item.
- `@tanstack/react-query` já configurado no projeto — usar para o data fetching da nova tela/lista.
- Componente `Dialog` (shadcn) já em uso — reaproveitar para o detalhe do item (D-03).

### Established Patterns
- Backend: handlers Go com queries sempre escopadas por `company_id` resolvido via JWT (nunca aceito do cliente) — `NFeSaidaDetailHandler` é o modelo mais próximo do que esta fase precisa (já faz o JOIN entre esperado e calculado).
- Frontend: páginas registradas em `navigation.ts` (tabs) + rota em `App.tsx` dentro do bloco protegido.

### Integration Points
- Nova rota GET precisa fazer JOIN entre `nfe_saidas_itens` (esperado) e `fiscal_execution_items` (calculado) por `nfe_item_id`, escopado por `company_id`.
- Nova aba em `navigation.ts` + rota React em `App.tsx`.

</code_context>

<specifics>
## Specific Ideas

- Usuário pediu explicitamente para "discutir o design da tela e carregar uma primeira versão" — prioridade é ter algo funcional e visível cedo (iterar rápido), não travar em polimento antes de validar o fluxo completo.
- A tela é o **core value do projeto** (ver PROJECT.md) — merece atenção redobrada em clareza visual das divergências, já que é o critério de sucesso do validador inteiro.

</specifics>

<deferred>
## Deferred Ideas

- **Filtro granular por imposto específico** (ex: "só ICMS divergente") — v1 usa toggle simples (D-08); pode virar v2 se o usuário sentir falta na prática.
- **Tolerância de arredondamento configurável** — v1 usa "qualquer diferença ≠ 0" (D-06); revisitar se gerar ruído (muitos falsos positivos de centavo).
- **Conceito de "lote de importação" como entidade** — não existe no schema hoje; resumo por lote foi descartado em favor de global + por nota (D-09).
- **Exportação CSV/Excel dos resultados** — já listado como AUTO-02 (v2) em REQUIREMENTS.md, não repetido aqui.

None além do acima — discussão permaneceu no escopo da fase.

</deferred>

---

*Phase: 3-Visual Comparison Screen*
*Context gathered: 2026-07-02*
