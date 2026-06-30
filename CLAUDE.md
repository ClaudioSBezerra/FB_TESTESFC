<!-- GSD:project-start source:PROJECT.md -->

## Project

**FB_TESTESFC — Validador de Testes Unitários do Pacote Fiscal (Ferreira Costa)**

Ferramenta de **validação fiscal** para a empresa **Ferreira Costa**. Importa XMLs completos de vendas (NFe de saída) e, para cada item vendido, recalcula os impostos chamando o "pacote fiscal" (script no banco `FCCORP_BKP`) e compara o resultado contra os valores que vieram no próprio XML. O objetivo é **testar unitariamente o pacote fiscal** — confirmar que ele reproduz os impostos corretos da nota real.

Herda do projeto irmão **FB_APU04** (cópia seletiva): autenticação/login, gestão de ambiente/grupo/empresa/usuário, conexão **ERP_BRIDGE (Oracle)** e o módulo de **importação de XMLs de saída**.

**Core Value:** Dado um XML de venda real, a tela mostra **item a item, imposto a imposto**, o valor esperado (do XML) vs. o calculado pelo pacote fiscal (script no FCCORP_BKP), **destacando divergências**. Se essa comparação funcionar com confiabilidade, o projeto cumpre seu propósito.

### Constraints

- **Tech stack**: Go 1.24, Postgres (dados do app), Oracle (ERP_BRIDGE + FCCORP_BKP), React + TypeScript + Vite + Tailwind, Docker — manter a mesma stack do FB_APU04 para reaproveitar código sem fricção.
- **Dependencies**: Acesso de leitura à instância Oracle que hospeda `prod`/`PRODB` e `FCCORP_BKP`; script do pacote fiscal fornecido pelo usuário.
- **Security**: Acesso Oracle somente leitura; credenciais geridas como no ERP_BRIDGE herdado (não versionar segredos).
- **Scope**: v1 restrito à empresa Ferreira Costa; evitar reconstruir a complexidade multi-tenant do projeto base.

<!-- GSD:project-end -->

<!-- GSD:stack-start source:STACK.md -->

## Technology Stack

Technology stack not yet documented. Will populate after codebase mapping or first phase.
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:

- `/gsd:quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd:debug` for investigation and bug fixing
- `/gsd:execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd:profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
