# Roadmap: FB_TESTESFC

## Overview

O projeto parte de uma cópia seletiva dos módulos maduros do FB_APU04 (auth, gestão de empresa/usuário, ERP_BRIDGE Oracle, importação de XML de saída) e acrescenta a camada nova: leitura do grupo fiscal por produto via Oracle, execução do script do pacote fiscal no FCCORP_BKP e a tela central de comparação visual item a item. As três fases entregam capacidades completas de ponta a ponta: primeiro a fundação rodando, depois o pipeline de dados funcionando, por fim a tela de comparação — que é o core value do projeto.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation & Inherited Stack** - Scaffold do projeto com módulos herdados do FB_APU04 rodando (auth, gestão Ferreira Costa, ERP_BRIDGE) (completed 2026-06-30)
- [ ] **Phase 2: Import Pipeline & Fiscal Execution** - Importação de XML, lookup de grupo fiscal no Oracle e execução do pacote fiscal persiste resultados
- [ ] **Phase 3: Visual Comparison Screen** - Tela de comparação visual item a item (esperado vs. calculado) com divergências destacadas

## Phase Details

### Phase 1: Foundation & Inherited Stack

**Goal**: Aplicação roda localmente com todos os módulos herdados do FB_APU04 — usuário faz login, admin gerencia usuários, empresa Ferreira Costa está configurada e a conexão Oracle (ERP_BRIDGE) está operacional.
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: FND-01, FND-02, FND-03, AUTH-01, AUTH-02, AUTH-03, AUTH-04, TEN-01, TEN-02, TEN-03
**Success Criteria** (what must be TRUE):

  1. `docker compose up` inicia o backend Go, o frontend React e o Postgres sem erros e as migrações de banco executam limpo em base zerada
  2. Usuário faz login com e-mail e senha, recebe sessão JWT e permanece autenticado após refresh do navegador
  3. Usuário pode solicitar recuperação de senha e concluir o fluxo de redefinição (forgot/reset)
  4. Admin pode criar, editar e desativar usuários na tela de gestão herdada
  5. Ambiente Ferreira Costa está pré-configurado e o contexto de empresa é resolvido corretamente nas requisições de API

**Plans**: 5 plans (3 waves)
Plans:

- [x] 01-01-PLAN.md — Fundação backend: go.mod enxuto, crypto/middleware/serviços, auth.go, migrações 001+004, main.go com runner (skeleton backend)
- [x] 01-02-PLAN.md — Shell frontend + infra Docker: AuthContext, páginas de auth, App.tsx, vite/Dockerfiles/docker-compose (Walking Skeleton: login + refresh)
- [x] 01-03-PLAN.md — Gestão backend: admin/environment/hierarchy/managers + migração 003 + rotas
- [x] 01-04-PLAN.md — ERP_BRIDGE backend: migração 002, erp_bridge.go + endpoint test-connection (go-ora) + rotas
- [x] 01-05-PLAN.md — Telas de gestão: AdminUsers/GestaoAmbiente/Managers/ERPBridge + botão Testar Conexão + navegação enxuta

**UI hint**: yes

### Phase 2: Import Pipeline & Fiscal Execution

**Goal**: Usuário importa XMLs de NFe de saída, o sistema obtém o grupo fiscal de cada item via Oracle (prod + PRODB), executa o script do pacote fiscal no FCCORP_BKP e persiste os impostos calculados — com erros isolados por item sem interromper o lote.
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: XML-01, XML-02, XML-03, XML-04, ERP-01, ERP-02, ERP-03, FIS-01, FIS-02, FIS-03
**Success Criteria** (what must be TRUE):

  1. Usuário carrega um ou vários arquivos XML de NFe de saída e recebe confirmação de importação ou mensagem de erro clara por arquivo com problema de parse/validação
  2. Notas importadas e seus itens com os valores de imposto originais do XML são visíveis na tela de consulta
  3. Sistema consulta `prod` + `PRODB` no Oracle e obtém o grupo fiscal para cada item; itens sem grupo fiscal são sinalizados individualmente sem bloquear os demais
  4. Sistema executa o script do pacote fiscal no FCCORP_BKP com os parâmetros do XML e o grupo fiscal, persiste os impostos calculados; falhas por item são exibidas sem abortar o lote inteiro
  5. Usuário configura as credenciais de conexão Oracle via tela ERP_BRIDGE herdada

**Plans**: 2 plans (2 waves)
Plans:

- [ ] 02-01-PLAN.md — Importação de XML: upload/parse/persistência (nfe_saidas + itens) e tela de consulta
- [ ] 02-02-PLAN.md — Execução fiscal: lookup grupo fiscal (prod/PRODB) + pacote PL/SQL + persistência com status por item

**UI hint**: yes

### Phase 3: Visual Comparison Screen

**Goal**: Usuário visualiza, item a item e imposto a imposto, o valor esperado (do XML) versus o calculado (pelo pacote fiscal), com divergências destacadas e filtros para análise rápida.
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: CMP-01, CMP-02, CMP-03, CMP-04
**Success Criteria** (what must be TRUE):

  1. Tela de comparação exibe cada item com colunas lado a lado para esperado (XML) e calculado (script) nas colunas fiscais: base ICMS, vlr ICMS, base ST, ICMS ST, base PIS/COFINS, vlr PIS/COFINS, DIFAL, FCP e demais retornados pelo script
  2. Itens com qualquer divergência são destacados visualmente (cor/ícone) e a diferença numérica é exibida em cada campo divergente
  3. Usuário pode filtrar a visualização para exibir apenas os itens com ao menos uma divergência
  4. Tela apresenta um resumo com total de itens, itens sem divergência e itens com divergência — por nota e por lote importado

**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation & Inherited Stack | 5/5 | Complete   | 2026-06-30 |
| 2. Import Pipeline & Fiscal Execution | 0/? | Not started | - |
| 3. Visual Comparison Screen | 0/? | Not started | - |
