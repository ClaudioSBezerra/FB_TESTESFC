---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: verified
stopped_at: "Milestone v1.0 audit re-rodado após Fase 03.1: status passed. Pronto para /gsd:complete-milestone."
last_updated: "2026-07-02T17:15:00.000Z"
last_activity: 2026-07-02
progress:
  total_phases: 4
  completed_phases: 4
  total_plans: 13
  completed_plans: 13
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-30)

**Core value:** Tela que compara, item a item e imposto a imposto, o valor esperado (do XML real) vs. o calculado pelo pacote fiscal (script no FCCORP_BKP), destacando divergências.
**Current focus:** Milestone v1.0 completo e auditado — pronto para /gsd:complete-milestone

## Current Position

Phase: 03.1 (fechar-gap-navega-o-quebrada-ligar-navega-o-clic-vel-s-telas) — COMPLETE
Plan: 3 of 3 — checkpoint humano aprovado (click-through admin + não-admin)
Status: BLOCKER-1 e BLOCKER-2 do audit v1.0 fechados. v1.0-MILESTONE-AUDIT.md status: passed. Milestone pronto pra fechar.
Last activity: 2026-07-02

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**

- Total plans completed: 0
- Average duration: —
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*
| Phase 01-foundation-inherited-stack P01-02 | 30min | 2 tasks | 78 files |
| Phase 01-foundation-inherited-stack P01-03 | 12min | 2 tasks | 6 files |
| Phase 01-foundation-inherited-stack P01-04 | 25min | 2 tasks | 6 files |
| Phase 03 P01 | 20min | 2 tasks | 5 files |
| Phase 03 P02 | 20min | 2 tasks | 3 files |
| Phase 03.1 P01 | 8min | 2 tasks | 2 files |
| Phase 03.1 P02 | 5min | 2 tasks | 3 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Cópia seletiva do FB_APU04 (não clonar tudo) — projeto enxuto, só o necessário para o fluxo de teste fiscal
- FCCORP_BKP e prod/PRODB na mesma instância Oracle — uma única conexão/credencial resolve os dois acessos
- XML = esperado (gabarito), script = valor testado
- Entregável v1 = tela de comparação visual (não suíte Go)
- v1 só Ferreira Costa — simplificar multi-tenant herdado
- [Phase ?]: Token de acesso em memória React (não localStorage) + refresh token httpOnly SameSite=Strict — proteção XSS/CSRF no Walking Skeleton
- [Phase ?]: 5 telas de gestão (GestaoAmbiente, Managers, AdminUsers, ERPBridgeConfig, ERPBridgeCredenciais) antecipadas do 01-05 para o 01-02 — escopo do 01-05 reduzido para menus/navegação e botão Testar Conexão
- [Phase ?]: environment.go adaptado sem cnpj/cnae_secundario/municipio: schema 001 não tem essas colunas
- [Phase 03]: Comparação Fiscal: divergência = qualquer diferença != 0 nos 4 pares (ICMS/ICMS-ST/PIS/COFINS), sem tolerância de arredondamento — Validador fiscal — até 1 centavo pode importar (D-06 do 03-CONTEXT.md)
- [Phase 03]: Item com fiscal_status != 'ok' é classificado como 'Não calculado', nunca como divergente — Evita falso positivo quando o cálculo ainda não foi concluído (D-10)
- [Phase ?]: [Phase 03-02]: Cards de resumo global recalculados para derivar de displayItems (pós-filtro), corrigindo comportamento herdado do 03-01 que contava sobre a lista bruta
- [Phase ?]: [Phase 03-02]: Mapeamento curado de full_result (IBS UF/Município, CBS, alíquotas) com rótulos amigáveis + fallback genérico chave-valor para os demais ~70 campos
- [Phase 03.1]: config.tabs mantido com exatamente 5 entradas admin/config; as 3 abas de negócio (Importar XMLs, Notas Importadas, Comparação Fiscal) migraram para chaves de módulo próprias
- [Phase 03.1]: getActiveModule fallback retorna 'comparacao' em vez de 'config', alinhado à nova página de destino pós-login (D-07/D-08) a ser ligada no plano 02
- [Phase 03.1]: Configurações icon retargeted from AdminRoute (/config/erp-bridge) to ProtectedRoute (/config/ambiente) so no role bounces to '/'
- [Phase 03.1]: Root route '/' and post-login redirect both point to /importacoes/comparacao-fiscal (ProtectedRoute), ending the non-admin redirect loop (BLOCKER-2)

### Roadmap Evolution

- Phase 03.1 inserted after Phase 3 (URGENT) — Fechar gap: navegação quebrada (achado pelo audit do milestone v1.0 — nenhuma tela de negócio alcançável clicando, loop de redirect para usuários não-admin). Ver `.planning/v1.0-MILESTONE-AUDIT.md`.

### Pending Todos

None yet.

### Blockers/Concerns

- **`codEmpresaPorCNPJRaiz` incompleto**: só a raiz de CNPJ de Recife/PE (`10230480` → `cod_empresa=2`) está confirmada e mapeada em `backend/handlers/fiscal_group_lookup.go`. Garanhuns/PE (`cod_empresa=1`) ainda não tem raiz de CNPJ confirmada — notas dessa filial retornam erro explícito por item até ser adicionada. Não bloqueia a Fase 3, mas deve ser completado antes de usar o validador com notas reais de todas as filiais.
- **Defaults de parâmetros do pacote fiscal não totalmente validados**: `pTipoContribuinte`, `pTipoCentroFiscal`, `pIndicadorServico`, `FornecedorSimplesNacional`, `pAliquotaSimplesNacional` em `backend/handlers/fiscal_execution.go` usam defaults conservadores (só o caminho "normal" foi testado contra Oracle real). A comparação da Fase 3 vai expor rapidamente qualquer default incorreto quando aparecerem casos reais divergentes (Simples Nacional, prestação de serviço, etc.).

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Automation | AUTO-01: Suíte de testes Go automatizada | v2 | Roadmap creation |
| Automation | AUTO-02: Exportação CSV/Excel | v2 | Roadmap creation |
| Multi-tenant | MTN-01: Suporte a múltiplas empresas | v2 | Roadmap creation |
| Quick task | 260701-oaa: busca automática de XMLs NFe (diretamente do ERP?) — pasta vazia sem conteúdo, artefato órfão de sessão anterior | acknowledged | Milestone v1.0 close (2026-07-02) |

## Session Continuity

Last session: 2026-07-02T17:15:00.000Z
Stopped at: Fase 03.1 completa (checkpoint click-through admin+não-admin aprovado). v1.0-MILESTONE-AUDIT.md re-rodado: status passed. Próximo: /gsd:complete-milestone 1.0.
Resume file: .planning/v1.0-MILESTONE-AUDIT.md
