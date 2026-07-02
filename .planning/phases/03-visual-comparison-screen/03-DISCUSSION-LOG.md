# Phase 3: Visual Comparison Screen - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-02
**Phase:** 3-Visual Comparison Screen
**Areas discussed:** Localização/navegação, Quais impostos mostrar e como agrupar, Escopo do filtro e do resumo

---

## Localização/navegação

| Option | Description | Selected |
|--------|-------------|----------|
| Nova aba/página dedicada "Comparação Fiscal" | Tela própria, focada só em comparar, permite listar/filtrar itens de múltiplas notas de uma vez | ✓ |
| Expandir o Dialog de detalhe existente | Reaproveita o Dialog de "Notas Importadas" | |
| As duas coisas | Nova aba + indicador resumido no Dialog existente | |

**User's choice:** Nova aba/página dedicada "Comparação Fiscal"

| Option | Description | Selected |
|--------|-------------|----------|
| Lista de itens direto | Tabela com todos os itens (de todas as notas), filtrável | ✓ |
| Escolhe a nota primeiro | Fluxo em 2 passos (nota → itens) | |

**User's choice:** Lista de itens direto

| Option | Description | Selected |
|--------|-------------|----------|
| Dialog/modal de detalhe | Mesmo padrão já usado em ConsultaNFeSaidas.tsx | ✓ |
| Página própria do item | Rota dedicada por item | |

**User's choice:** Dialog/modal de detalhe

---

## Quais impostos mostrar e como agrupar

| Option | Description | Selected |
|--------|-------------|----------|
| Só os pares 1:1 disponíveis | Base/Valor ICMS, ICMS-ST, PIS, COFINS — únicos campos com par no XML | ✓ |
| Resumo agregado por imposto | Coluna única OK/Divergente por imposto | |
| Só uma coluna "Status geral" | Um resumo geral por item | |

**User's choice:** Só os pares 1:1 disponíveis

| Option | Description | Selected |
|--------|-------------|----------|
| 3 colunas separadas: Esperado \| Calculado \| Diferença | Mais claro, tabela mais larga | ✓ |
| 1 coluna compacta "Esperado → Calculado" | Mais estreito, menos detalhe direto | |

**User's choice:** 3 colunas separadas

| Option | Description | Selected |
|--------|-------------|----------|
| Qualquer diferença ≠ 0 | Rigoroso — até 1 centavo conta | ✓ |
| Tolerância de arredondamento (> R$0,01) | Ignora ruído de arredondamento | |

**User's choice:** Qualquer diferença ≠ 0

| Option | Description | Selected |
|--------|-------------|----------|
| Seção "Só calculado" | Lista campos sem par no XML (DIFAL/FCP/IBS/CBS/etc.), só valor calculado | ✓ |
| Não mostrar no v1 | Oculta esses campos | |

**User's choice:** Seção "Só calculado"

---

## Escopo do filtro e do resumo

| Option | Description | Selected |
|--------|-------------|----------|
| Toggle simples: qualquer imposto divergente | Um único toggle "Só divergentes" | ✓ |
| Filtro por imposto específico | Checkboxes por imposto | |

**User's choice:** Toggle simples

| Option | Description | Selected |
|--------|-------------|----------|
| Global + por nota | Cards no topo + resumo por nota no Dialog | ✓ |
| Só global | Um resumo agregado só | |
| Por lote de importação | Precisaria criar entidade "lote" | |

**User's choice:** Global + por nota

| Option | Description | Selected |
|--------|-------------|----------|
| Categoria própria "Não calculado" | Terceiro balde separado de OK/Divergente | ✓ |
| Contam como divergente | Tratados como divergência por padrão | |

**User's choice:** Categoria própria "Não calculado"

---

## Claude's Discretion

- Larguras/responsividade exata da tabela
- Cores/ícones exatos de divergência (seguir padrão FiscalStatusBadge)
- Paginação/virtualização da lista de itens
- Mapeamento de nomes de campo em `full_result` (JSONB) para labels amigáveis
- Confirmar durante research se DIFAL/FCP têm algum valor esperado equivalente não capturado na Fase 2

## Deferred Ideas

- Filtro granular por imposto específico — v2 se necessário
- Tolerância de arredondamento configurável — revisitar se gerar ruído
- Conceito de "lote de importação" como entidade — descartado, não introduzido nesta fase
- Exportação CSV/Excel — já listado como AUTO-02 (v2)
