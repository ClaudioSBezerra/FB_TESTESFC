# FB_TESTESFC — Validador de Testes Unitários do Pacote Fiscal (Ferreira Costa)

## What This Is

Ferramenta de **validação fiscal** para a empresa **Ferreira Costa**. Importa XMLs completos de vendas (NFe de saída) e, para cada item vendido, recalcula os impostos chamando o pacote fiscal real (`PKG_FISCAL_FCTAX.calcula_imposto_produto`, no banco `FCCORP_BKP`) e compara o resultado contra os valores que vieram no próprio XML. O objetivo é **testar unitariamente o pacote fiscal** — confirmar que ele reproduz os impostos corretos da nota real. **Shipado como v1.0 MVP em 2026-07-02**, ponta a ponta (login → importar XML → executar cálculo fiscal → comparar), verificado contra Oracle real.

Herda do projeto irmão **FB_APU04** (cópia seletiva): autenticação/login, gestão de ambiente/grupo/empresa/usuário, conexão **ERP_BRIDGE (Oracle)** e o módulo de **importação de XMLs de saída**.

## Core Value

Dado um XML de venda real, a tela "Comparação Fiscal" mostra **item a item, imposto a imposto**, o valor esperado (do XML) vs. o calculado pelo pacote fiscal (script no FCCORP_BKP), **destacando divergências**. Confirmado como o core value certo — validado no v1.0 com dados reais do pacote fiscal (uma divergência real de ICMS foi corretamente detectada e destacada durante a verificação).

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

- ✓ Usuário autentica (login/sessão JWT) reaproveitando o módulo de auth do FB_APU04 — v1.0
- ✓ Estrutura mínima de ambiente/empresa/usuário (simplificada para empresa única: Ferreira Costa) — v1.0
- ✓ Usuário importa um ou vários XMLs completos de vendas (NFe saída) reaproveitando o importador do FB_APU04 — v1.0
- ✓ Sistema persiste os XMLs importados e seus itens/impostos (Postgres) — v1.0
- ✓ Para cada item, sistema conecta via ERP_BRIDGE (Oracle) e lê `prod` + `PRODB` para obter o grupo fiscal do produto — v1.0
- ✓ Sistema executa o script do pacote fiscal no FCCORP_BKP passando parâmetros herdados do XML de origem + o grupo fiscal lido — v1.0 (verificado contra Oracle real)
- ✓ Sistema carrega e persiste o retorno do script (impostos calculados) — v1.0
- ✓ Tela de comparação visual item a item: base/vlr ICMS, ICMS-ST, PIS/COFINS, DIFAL, FCP, IBS/CBS — XML (esperado) vs script (testado) — v1.0
- ✓ Divergências entre esperado e calculado são destacadas visualmente — v1.0
- ✓ Navegação clicável para todas as telas de negócio, sem loop de redirect para não-admin — v1.0 (Fase 03.1, gap encontrado pelo audit e fechado antes do fechamento do milestone)

### Active

<!-- Current scope. Building toward these. -->

(Nenhuma ainda definida — rodar `/gsd:new-milestone` para escopar v1.1)

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- Multi-tenant completo / troca de empresas — v1 foca apenas na Ferreira Costa; simplificar a complexidade herdada. **Reason ainda válido** (nenhuma segunda empresa testada).
- Importação de XMLs de entrada, CTe, EFD/SPED — fora do propósito de testar o pacote fiscal de saídas. **Reason ainda válido.**
- Painéis de apuração da Reforma Tributária e demais módulos fiscais do FB_APU04 — não relacionados ao objetivo de teste. **Reason ainda válido** (a tela de comparação só exibe os campos IBS/CBS retornados pelo pacote como auditoria, não implementa apuração).
- Testes Go automatizados / suíte em CI (AUTO-01) — o entregável escolhido é a tela de comparação visual; pode virar requisito futuro. **Reason ainda válido.**
- Gravação/escrita no FCCORP_BKP ou no ERP de produção — acesso é somente leitura para fins de validação. **Reason ainda válido**, confirmado por threat model em todas as fases (nenhum INSERT/UPDATE/DELETE contra tabelas Oracle).

## Context

- **Projeto base:** `FB_APU04` (em `/home/claudiobezerra/projetos/FB_APU04`, repo `github.com/ClaudioSBezerra/FB_APU04`). Sistema maduro de apuração fiscal em Go + React + Postgres com auth, gestão hierárquica (ambiente/grupo/empresa/usuário), ERP_BRIDGE Oracle e importação de XMLs de saída já implementados.
- **Dois acessos a dados Oracle, mesma instância:**
  1. ERP_BRIDGE → tabelas `prod` (produto) e `PRODB` (complemento) → grupo fiscal
  2. Pacote fiscal → `FCCORP_BKP` → impostos calculados via `PKG_FISCAL_FCTAX.calcula_imposto_produto` (23 params IN, ~88 campos OUT — contrato extraído de `/tmp/11_Script_Teste_Pacote_FCTAX_1S_Reforma_Tributaria.TST`, ver `backend/services/oracle_fiscal.go`)
- **Repositório FB_TESTESFC:** conectado a `github.com/ClaudioSBezerra/FB_TESTESFC` (branch `main`). ~16.769 LOC (Go+TS/TSX), 99 commits ao fechar o v1.0.
- **Driver Oracle:** `github.com/sijms/go-ora/v2`. Pegadinha não-óbvia: binds OUT de string via `sql.Out` (database/sql genérico) passam `size=0` ao driver, causando `ORA-06502 buffer too small` — usar sempre `go_ora.Out{Dest, Size: N}` (tipo nativo do driver) para OUT de VARCHAR2.

### Known Issues / Technical Debt (pós-v1.0)

- `codEmpresaPorCNPJRaiz` (`backend/handlers/fiscal_group_lookup.go`) só tem a raiz de CNPJ da filial Recife/PE mapeada — Garanhuns/PE ainda não confirmada, notas dessa filial retornam erro explícito por item.
- Defaults conservadores para parâmetros do pacote fiscal sem fonte de dado persistida (`pTipoContribuinte`, `pTipoCentroFiscal`, `pIndicadorServico`, `FornecedorSimplesNacional`, `pAliquotaSimplesNacional`) — só o caminho "normal" (não-Simples, não-serviço) foi validado contra Oracle real.
- Ícones do AppRail não têm `aria-label` (só tooltip visual) — screen readers não identificam os botões.
- Link morto `/importacoes/erp-bridge/logs` (`ERPBridgeConfig.tsx`) e ausência de rota catch-all `*` para 404 amigável.

## Constraints

- **Tech stack**: Go 1.24, Postgres (dados do app), Oracle (ERP_BRIDGE + FCCORP_BKP, via `go-ora/v2`), React + TypeScript + Vite + Tailwind, Docker — confirmado estável no v1.0, manter para v1.1+.
- **Dependencies**: Acesso de leitura à instância Oracle que hospeda `prod`/`PRODB` e `FCCORP_BKP`.
- **Security**: Acesso Oracle somente leitura — confirmado sem nenhuma superfície de escrita em todas as 4 fases do v1.0; credenciais criptografadas (AES-256) em `erp_bridge_config`.
- **Scope**: v1 restrito à empresa Ferreira Costa; evitar reconstruir a complexidade multi-tenant do projeto base.

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Cópia seletiva dos módulos do FB_APU04 (não clonar tudo) | Projeto enxuto, só o necessário para o fluxo de teste fiscal | ✓ Good |
| FCCORP_BKP e prod/PRODB na mesma instância Oracle | Uma única conexão/credencial resolve os dois acessos | ✓ Good |
| XML = esperado (gabarito), script = valor testado | A NFe autorizada é a verdade; o pacote fiscal deve reproduzi-la | ✓ Good |
| Entregável v1 = tela de comparação visual (não suíte Go) | Analista compara e julga divergências; automação fica para depois | ✓ Good |
| v1 só Ferreira Costa, simplificar multi-tenant | Reduz complexidade herdada sem perder o login/gestão básicos | ✓ Good |
| Bloco PL/SQL do pacote fiscal 100% estático/gerado por reflection (nunca `fmt.Sprintf` com valor de entrada) | Elimina injeção de SQL na chamada do pacote fiscal | ✓ Good (verificado por grep em toda execução) |
| Divergência = qualquer diferença ≠ 0 (sem tolerância de arredondamento) no v1 | É um validador fiscal — até 1 centavo pode importar | ✓ Good — pode revisitar se gerar ruído na prática |
| Item com `fiscal_status != 'ok'` vira bucket "Não calculado", nunca "Divergente" | Evita falso positivo quando o cálculo ainda não rodou | ✓ Good |
| Landing pós-login unificado em `/importacoes/comparacao-fiscal` (admin e não-admin) | É o core value do projeto e resolve o loop de redirect sem lógica condicional por role | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-07-02 after v1.0 milestone*
