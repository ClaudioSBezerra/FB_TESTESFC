# Milestones

## v1.0 MVP (Shipped: 2026-07-02)

**Phases completed:** 4 phases, 13 plans, 20 tasks

**Delivered:** Validador fiscal para a Ferreira Costa — importa XMLs de NFe de saída, executa o pacote fiscal real (FCCORP_BKP) e compara item a item o valor esperado (XML) vs. calculado (pacote), destacando divergências.

**Key accomplishments:**

- **Fase 1 — Fundação:** cópia seletiva do FB_APU04 (auth JWT, gestão de ambiente/empresa/usuário, ERP_BRIDGE Oracle configurável) rodando em `docker compose up`, com Walking Skeleton de login validado no browser.
- **Fase 2 — Pipeline fiscal:** importação de XML de NFe de saída (parse + persistência idempotente, isolamento de erro por arquivo) e execução real do pacote `PKG_FISCAL_FCTAX` contra o Oracle FCCORP_BKP — dois bugs de binding do driver go-ora (`sql.Out` sem tamanho, campos `IdRegraCalculo*` tipados errado) encontrados e corrigidos durante a verificação com Oracle real.
- **Fase 3 — Comparação Fiscal (core value):** tela que lista item a item os 4 pares de imposto (ICMS/ICMS-ST/PIS/COFINS) esperado vs. calculado, com divergência destacada, filtro "só divergentes" e resumo por status — verificada com dados reais do pacote fiscal.
- **Fase 03.1 — Fechamento de gap crítico:** o audit do milestone encontrou que nenhuma tela de negócio era alcançável clicando na UI (navegação nunca ligada desde a Fase 1) e que usuários não-admin caíam num loop de redirect. Corrigido e reverificado por click-through real (Playwright clicando, não digitando URL) como admin e não-admin.
- **Todos os 24 requisitos v1** implementados e verificados; milestone fechado com audit `status: passed`.

**Lição principal:** verificação via API/URL direta não é suficiente para pegar bugs de navegação — só clicar pela UI de verdade revelou o gap que bloqueava o uso real do produto.

---
