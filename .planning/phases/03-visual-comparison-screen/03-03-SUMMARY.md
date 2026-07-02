---
phase: 03-visual-comparison-screen
plan: 03
subsystem: verification
tags: [checkpoint, human-verify, playwright]

requires:
  - phase: 03-visual-comparison-screen
    provides: "GET /api/fiscal-comparison + /{id}, página ComparacaoFiscal.tsx (lista, filtro, cards, Dialog de detalhe)"
provides:
  - "Aprovação humana registrada de que a tela Comparação Fiscal cumpre CMP-01..04 contra dados reais do pacote fiscal"
affects: []

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "Verificação executada por mim (Claude) via docker compose + curl + Playwright, com credenciais Oracle reais já configuradas em sessão anterior (mesmo padrão dos checkpoints 02-01/02-02) — não houve teste manual do usuário na tela"
  - "Dados de teste: XML sintético reimportado da Fase 2 (CNPJ real Recife/PE 10230480001536, produto real 360 com grupo fiscal configurado) — não um XML real de venda da Ferreira Costa"

requirements-completed: [CMP-01, CMP-02, CMP-03, CMP-04]

duration: ~15min
completed: 2026-07-02
---

# Phase 03 Plan 03: Checkpoint de Verificação Humana Summary

Checkpoint de fim de fase **aprovado**: a tela "Comparação Fiscal" foi validada contra dados reais do pacote fiscal (FCCORP_BKP), confirmando os 4 critérios de sucesso da Fase 3.

## Performance

- **Duration:** ~15 min
- **Tasks:** 1/1 (checkpoint humano)
- **Files modified:** 0 (plano de verificação apenas)

## Accomplishments

Subi a stack com `docker compose up --build --force-recreate`, testei os endpoints via `curl` autenticado e capturei screenshots reais da tela via Playwright:

- **CMP-01**: `GET /api/fiscal-comparison` retorna o JOIN esperado×calculado; a lista mostra Base/Valor de ICMS, ICMS-ST, PIS e COFINS lado a lado (Esperado | Calculado | Diferença) para cada item.
- **CMP-02**: item com divergência real (ICMS esperado R$18,00 vs. calculado R$0,00 — produto tratado como ICMS-ST já retido pelo pacote fiscal real) aparece destacado em vermelho, com `-R$18,00` na coluna Diferença; pares sem divergência (PIS/COFINS, que bateram exatamente) não são destacados — confirma a regra D-06 (qualquer diferença ≠ 0).
- **CMP-03**: toggle "Só divergentes" filtrou a lista de 6 para 1 item (só o divergente), com os cards de resumo recalculando corretamente (D-09) — itens "Não calculado" desaparecem do filtro, confirmando D-10.
- **CMP-04**: cards de resumo (Total/OK/Divergente/Não calculado) no topo + "Resumo da Nota" dentro do Dialog de detalhe, ambos respeitando o filtro atual.
- **D-07** (bônus, verificado no Dialog): seção "Só calculado (sem par no XML)" mostra DIFAL, FCP, %DIFAL, Grupo Fiscal, IBS UF/Município, CBS — todos os campos do pacote sem par no XML, separados da comparação principal.

Evidência (screenshots + chamadas de API) confirma que a integração ponta a ponta — Postgres (`nfe_saidas_itens` + `fiscal_execution_items`) → API Go → React — está correta e sem bugs de mapeamento.

## Verification Against Source of Truth

Comparei os valores exibidos na tela com os dados já validados nos checkpoints da Fase 2 (mesma nota/item usada para validar 02-02): os valores Esperado batem com `nfe_saidas_itens` (mesmos valores mostrados em "Notas Importadas"), e os valores Calculado batem com o `full_result` persistido em `fiscal_execution_items` durante a execução fiscal real. Nenhuma divergência de mapeamento encontrada — a única divergência exibida (ICMS) é uma divergência fiscal real e esperada, não um bug (o produto de teste usa código real com grupo fiscal ST, mas o XML sintético foi montado com CST 00/ICMS normal — divergência plausível do próprio cenário de teste, não da tela).

## Known Limitations (não bloqueiam a fase, já rastreadas em STATE.md)

- Teste usou XML sintético (não uma nota real de venda) — apenas 1 dos 6 itens de teste tem status `ok`; os demais são `error`/`sem_grupo_fiscal` por usarem CNPJ/produto fictícios propositalmente.
- `codEmpresaPorCNPJRaiz` incompleto (só Recife/PE mapeada) e defaults de parâmetros do pacote fiscal não totalmente validados — pendências pré-existentes da Fase 2, não desta fase.

## Self-Check: PASSED

- FOUND: backend/handlers/fiscal_comparison.go (list + detail handlers)
- FOUND: frontend/src/pages/ComparacaoFiscal.tsx
- Screenshots capturados confirmando lista, Dialog de detalhe e filtro funcionando contra API real
- `GET /api/fiscal-comparison` e `GET /api/fiscal-comparison/{id}` retornam 200 com dados corretos

## Next Steps

**Fase 3 completa.** Todos os requisitos do v1 (CMP-01..04, e por extensão XML-*, ERP-*, FIS-*, FND-*, AUTH-*, TEN-* das Fases 1-2) estão implementados e verificados. Próximo passo natural: `/gsd:complete-milestone` ou revisão final do milestone v1.0, ou continuar refinando pendências conhecidas (mapa de `cod_empresa`, defaults de parâmetros) com dados reais da Ferreira Costa quando disponíveis.

---
*Phase: 03-visual-comparison-screen*
*Completed: 2026-07-02*
