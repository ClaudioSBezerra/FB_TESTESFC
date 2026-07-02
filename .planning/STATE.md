---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: Awaiting next milestone
stopped_at: "Fase 03.1 completa (checkpoint click-through admin+não-admin aprovado). v1.0-MILESTONE-AUDIT.md re-rodado: status passed. Próximo: /gsd:complete-milestone 1.0."
last_updated: "2026-07-02T17:06:43.599Z"
last_activity: 2026-07-02 — Milestone v1.0 completed and archived
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

Phase: Milestone v1.0 complete
Plan: —
Status: Awaiting next milestone
Last activity: 2026-07-02 - Completed quick task 260702-mwi: Aumentar limite de upload de XML/ZIP de 2GB para 5GB

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
- [Quick 260702-lp5]: Migração `010_ensure_master_link.sql` portada quase literalmente de `024_ensure_master_link.sql` do FB_APU04, garantindo hierarquia MASTER e vínculo do admin logo após deploy com banco zerado; padrão capturado na skill global `coolify-deploy-checklist` (seção 1.7) para produtos futuros da família FBTax

### Roadmap Evolution

- Phase 03.1 inserted after Phase 3 (URGENT) — Fechar gap: navegação quebrada (achado pelo audit do milestone v1.0 — nenhuma tela de negócio alcançável clicando, loop de redirect para usuários não-admin). Ver `.planning/v1.0-MILESTONE-AUDIT.md`.

### Pending Todos

None yet.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260702-jt1 | Trocar texto do badge da tela de login: "Simulador da Reforma Tributária - SPED" → "Simulador do pacote fiscal - FCTAX" | 2026-07-02 | 0aa3418 | [260702-jt1-na-tela-inicial-do-login-trocar-o-texto-](./quick/260702-jt1-na-tela-inicial-do-login-trocar-o-texto-/) |
| 260702-ju0 | Substituir os 5 bullets herdados do FB_APU04 (EFD/SPED) na tela de login por "Ler base atual de saídas e fazer cálculos para reforma tributária" | 2026-07-02 | 790bebf | [260702-ju0-na-tela-de-login-trocar-a-lista-de-bulle](./quick/260702-ju0-na-tela-de-login-trocar-a-lista-de-bulle/) |
| 260702-k3u | Configurar deploy de produção no Coolify/Hostinger (docker-compose.prod.yml + workflow de CI) para https://testesfc.fbtax.cloud | 2026-07-02 | 590ad48 | [260702-k3u-configurar-deploy-de-produ-o-no-coolify-](./quick/260702-k3u-configurar-deploy-de-produ-o-no-coolify-/) |
| 260702-le0 | Fix produção (503): nginx.conf usava container_name fb_testesfc-api (ignorado pelo Coolify) em vez do alias de rede testesfc-api | 2026-07-02 | b22df36 | [260702-le0-corrigir-bug-de-deploy-em-produ-o-fronte](./quick/260702-le0-corrigir-bug-de-deploy-em-produ-o-fronte/) |
| 260702-lp5 | Migração idempotente 010_ensure_master_link.sql (garante hierarquia MASTER + vínculo do admin) + padrão documentado na skill global coolify-deploy-checklist | 2026-07-02 | 0ef828a | [260702-lp5-criar-migra-o-010-ensure-master-link-sql](./quick/260702-lp5-criar-migra-o-010-ensure-master-link-sql/) |
| 260702-mwi | Aumentar limite de upload de XML/ZIP de 2GB para 5GB (nginx+backend+frontend) + corrigir bug real: nginx estava travado em 512M | 2026-07-02 | 4bbfe81 | [260702-mwi-aumentar-limite-de-upload-de-xml-zip-de-](./quick/260702-mwi-aumentar-limite-de-upload-de-xml-zip-de-/) |

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

Last session: 2026-07-02T18:41:32.000Z
Stopped at: Completed quick task 260702-lp5: Migração 010_ensure_master_link.sql criada e padrão MASTER documentado na skill global coolify-deploy-checklist. Próximo: /gsd:complete-milestone 1.0 (ainda pendente).
Resume file: .planning/v1.0-MILESTONE-AUDIT.md

## Operator Next Steps

- Start the next milestone with /gsd:new-milestone
