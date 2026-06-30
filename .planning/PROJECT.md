# FB_TESTESFC — Validador de Testes Unitários do Pacote Fiscal (Ferreira Costa)

## What This Is

Ferramenta de **validação fiscal** para a empresa **Ferreira Costa**. Importa XMLs completos de vendas (NFe de saída) e, para cada item vendido, recalcula os impostos chamando o "pacote fiscal" (script no banco `FCCORP_BKP`) e compara o resultado contra os valores que vieram no próprio XML. O objetivo é **testar unitariamente o pacote fiscal** — confirmar que ele reproduz os impostos corretos da nota real.

Herda do projeto irmão **FB_APU04** (cópia seletiva): autenticação/login, gestão de ambiente/grupo/empresa/usuário, conexão **ERP_BRIDGE (Oracle)** e o módulo de **importação de XMLs de saída**.

## Core Value

Dado um XML de venda real, a tela mostra **item a item, imposto a imposto**, o valor esperado (do XML) vs. o calculado pelo pacote fiscal (script no FCCORP_BKP), **destacando divergências**. Se essa comparação funcionar com confiabilidade, o projeto cumpre seu propósito.

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

(None yet — ship to validate)

### Active

<!-- Current scope. Building toward these. -->

- [ ] Usuário autentica (login/sessão JWT) reaproveitando o módulo de auth do FB_APU04
- [ ] Estrutura mínima de ambiente/empresa/usuário (simplificada para empresa única: Ferreira Costa)
- [ ] Usuário importa um ou vários XMLs completos de vendas (NFe saída) reaproveitando o importador do FB_APU04
- [ ] Sistema persiste os XMLs importados e seus itens/impostos (Postgres)
- [ ] Para cada item, sistema conecta via ERP_BRIDGE (Oracle) e lê `prod` + `PRODB` para obter o grupo fiscal do produto
- [ ] Sistema executa o script do pacote fiscal no FCCORP_BKP passando parâmetros herdados do XML de origem + o grupo fiscal lido
- [ ] Sistema carrega e persiste o retorno do script (impostos calculados)
- [ ] Tela de comparação visual item a item: base ICMS, vlr ICMS, base ST, ICMS ST, base PIS/COFINS, vlr PIS/COFINS, DIFAL, FCP, entre outros — XML (esperado) vs script (testado)
- [ ] Divergências entre esperado e calculado são destacadas visualmente

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- Multi-tenant completo / troca de empresas — v1 foca apenas na Ferreira Costa; simplificar a complexidade herdada
- Importação de XMLs de entrada, CTe, EFD/SPED — fora do propósito de testar o pacote fiscal de saídas
- Painéis de apuração da Reforma Tributária e demais módulos fiscais do FB_APU04 — não relacionados ao objetivo de teste
- Testes Go automatizados / suíte em CI — o entregável escolhido é a tela de comparação visual; pode virar requisito futuro
- Gravação/escrita no FCCORP_BKP ou no ERP de produção — acesso é somente leitura para fins de validação

## Context

- **Projeto base:** `FB_APU04` (em `/home/claudiobezerra/projetos/FB_APU04`, repo `github.com/ClaudioSBezerra/FB_APU04`). Sistema maduro de apuração fiscal em Go + React + Postgres com auth, gestão hierárquica (ambiente/grupo/empresa/usuário), ERP_BRIDGE Oracle e importação de XMLs de saída já implementados.
- **Módulos a reaproveitar (cópia seletiva):**
  - Backend: `handlers/auth.go`, `handlers/environment.go`, `handlers/hierarchy.go`, `handlers/erp_bridge*.go`, `handlers/xml_upload.go`, `handlers/nfe_saidas.go` (e dependências)
  - Frontend: `pages/Login.tsx`, `Register.tsx`, `ForgotPassword.tsx`, `ResetPassword.tsx`, `GestaoAmbiente.tsx`, `Managers.tsx`, `AdminUsers.tsx`, `ImportarXMLsSaida.tsx`, `ConsultaNFeSaidas.tsx`, `ERPBridgeConfig.tsx`/`ERPBridgeCredenciais.tsx`, `CompanySwitcher.tsx`
- **Dois acessos a dados Oracle, mesma instância:**
  1. ERP_BRIDGE → tabelas `prod` (produto) e `PRODB` (complemento) → grupo fiscal
  2. Script do pacote fiscal → `FCCORP_BKP` → impostos calculados
- **Script do pacote fiscal:** será fornecido pelo usuário; recebe parâmetros herdados do XML de origem + o grupo fiscal lido de PROD/PRODB e retorna as colunas de imposto. Formato exato (SQL puro vs. procedure PL/SQL) a confirmar quando o script chegar.
- **Repositório FB_TESTESFC:** já inicializado e conectado a `github.com/ClaudioSBezerra/FB_TESTESFC` (branch `main`).

## Constraints

- **Tech stack**: Go 1.24, Postgres (dados do app), Oracle (ERP_BRIDGE + FCCORP_BKP), React + TypeScript + Vite + Tailwind, Docker — manter a mesma stack do FB_APU04 para reaproveitar código sem fricção.
- **Dependencies**: Acesso de leitura à instância Oracle que hospeda `prod`/`PRODB` e `FCCORP_BKP`; script do pacote fiscal fornecido pelo usuário.
- **Security**: Acesso Oracle somente leitura; credenciais geridas como no ERP_BRIDGE herdado (não versionar segredos).
- **Scope**: v1 restrito à empresa Ferreira Costa; evitar reconstruir a complexidade multi-tenant do projeto base.

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Cópia seletiva dos módulos do FB_APU04 (não clonar tudo) | Projeto enxuto, só o necessário para o fluxo de teste fiscal | — Pending |
| FCCORP_BKP e prod/PRODB na mesma instância Oracle | Uma única conexão/credencial resolve os dois acessos | — Pending |
| XML = esperado (gabarito), script = valor testado | A NFe autorizada é a verdade; o pacote fiscal deve reproduzi-la | — Pending |
| Entregável v1 = tela de comparação visual (não suíte Go) | Analista compara e julga divergências; automação fica para depois | — Pending |
| v1 só Ferreira Costa, simplificar multi-tenant | Reduz complexidade herdada sem perder o login/gestão básicos | — Pending |

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
*Last updated: 2026-06-30 after initialization*
