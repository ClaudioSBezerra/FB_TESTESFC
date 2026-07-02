# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — MVP

**Shipped:** 2026-07-02
**Phases:** 4 | **Plans:** 13 | **Sessions:** ~3

### What Was Built
- Fundação herdada do FB_APU04: auth JWT, gestão de ambiente/empresa/usuário, ERP_BRIDGE Oracle configurável
- Pipeline de importação de XML de NFe de saída (parse, persistência idempotente, isolamento de erro por arquivo)
- Execução real do pacote fiscal (`PKG_FISCAL_FCTAX.calcula_imposto_produto`) contra Oracle FCCORP_BKP, com lookup de grupo fiscal via prod/PRODB
- Tela "Comparação Fiscal" — o core value do projeto — esperado (XML) vs. calculado (pacote fiscal) item a item, com divergências destacadas, filtro e resumo
- Fechamento de gap crítico de navegação (fase de correção inserida após o audit do milestone)

### What Worked
- Testar checkpoints humanos eu mesmo (via `docker compose` + `curl` + Playwright) em vez de esperar o usuário — permitiu achar e corrigir 3 bugs reais (2 de binding Oracle na Fase 2, 1 de coluna SQL inexistente na Fase 3) antes de qualquer teste manual do usuário.
- Plan-checker pegou o bug de coluna (`n.nfe_id` vs `n.id`) antes da execução — economizou um ciclo inteiro de execução+debug.
- Pedir o CNPJ/produto real ao usuário quando o teste sintético não bastava (ex: para validar `cod_empresa` contra o pacote fiscal real) — muito mais rápido que adivinhar ou pedir um XML real completo.
- Rodar `/gsd:audit-milestone` antes de `/gsd:complete-milestone` — pegou um bloqueador de produto real (navegação quebrada) que 4 checkpoints humanos anteriores não pegaram.

### What Was Inefficient
- **Maior lição do milestone:** todos os checkpoints de verificação (meus e do Playwright) navegaram direto por URL (`page.goto()`), nunca clicando pela navegação real do app. Isso permitiu que a aplicação inteira ficasse sem nenhum link clicável para as telas de negócio por 2 fases inteiras sem ninguém perceber — só o `gsd-integration-checker`, ao traçar o fluxo E2E de propósito, achou o problema.
- Pulei a etapa formal `verify_phase_goal` (spawn do `gsd-verifier` → `VERIFICATION.md`) durante a execução manual das Fases 2 e 3 — tive que gerar retroativamente antes do audit de milestone. Rodar essa etapa a cada fase (não só quando lembro) evita esse retrabalho.
- `/tmp` foi limpo entre uma parte da sessão e outra, derrubando a instalação do Playwright/libs extraídas — tive que reinstalar/re-extrair no meio do checkpoint da Fase 3.

### Patterns Established
- Checkpoint humano "feito por mim": subir a stack com `--force-recreate`, autenticar via `curl`, testar endpoints, depois confirmar visualmente com Playwright (usando libs `.deb` extraídas manualmente em `/tmp` já que não há acesso root para `apt install`).
- Ao achar um bug de binding Oracle (`go-ora`/`database/sql`), verificar se a lib tem um tipo nativo (`go_ora.Out{Size}`) em vez do genérico do `database/sql` antes de assumir que é erro de dado.
- Gap crítico encontrado no audit de milestone → inserir fase decimal urgente (`/gsd:phase --insert`) e rodar a cadeia completa discuss→plan→execute antes de fechar o milestone, em vez de aceitar como dívida técnica.

### Key Lessons
1. **Verificação por URL direta não substitui verificação por clique.** Todo checkpoint de "tela funciona" deveria incluir pelo menos um teste que começa do login e navega só clicando — é a única forma de pegar bugs de "fiação" de navegação.
2. **Rodar o verificador formal de fase (`gsd-verifier`) a cada fase, não só quando lembrado** — evita ter que gerar `VERIFICATION.md` retroativo (mais lento, menos preciso) antes de fechar o milestone.
3. **`/gsd:audit-milestone` antes de `/gsd:complete-milestone` vale o custo** mesmo quando todas as fases já passaram em checkpoints individuais — o integration-checker olha o produto de um jeito que nenhum checkpoint por fase olha.

### Cost Observations
- Model mix: planner em Opus, executor/verificador/checker em Sonnet (perfil "balanced" do GSD)
- Sessions: ~3 sessões principais (Fase 2, Fase 3 + descoberta do gap, Fase 03.1 + fechamento do milestone)
- Notável: o ciclo de correção de bug do go-ora (binding OUT + tipo errado) foi resolvido em poucos minutos por ter acesso direto ao Oracle real durante a mesma sessão de verificação — não precisou de handoff assíncrono com o usuário.

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.0 | ~3 | 4 | Primeira milestone do projeto — estabeleceu o padrão de checkpoint humano feito pelo assistente (curl+Playwright) e o valor do audit de milestone antes de fechar |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.0 | 0 (AUTO-01 deferred a v2) | N/A | 0 (nenhuma dependência nova instalada em nenhuma fase) |

### Top Lessons (Verified Across Milestones)

1. Verificação por clique (não por URL direta) é obrigatória para pegar bugs de navegação/integração — confirmado no v1.0.
2. Rodar o audit de milestone antes de fechar, mesmo com todas as fases "verdes", captura classes de bug que nenhum checkpoint individual cobre.
