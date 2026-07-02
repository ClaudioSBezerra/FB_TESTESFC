# Requirements: FB_TESTESFC

**Defined:** 2026-06-30
**Core Value:** Tela que compara, item a item e imposto a imposto, o valor esperado (do XML real) vs. o calculado pelo pacote fiscal (script no FCCORP_BKP), destacando divergências.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Foundation (Base herdada do FB_APU04)

- [x] **FND-01**: Projeto Go 1.24 + React/TS/Vite/Tailwind + Postgres inicializado com a stack do FB_APU04 (build e run locais funcionando)
- [x] **FND-02**: Apenas os módulos necessários são copiados do FB_APU04 (auth, gestão, ERP_BRIDGE, importação XML saída), sem os módulos fiscais não relacionados
- [x] **FND-03**: Migrações de banco Postgres para as tabelas reaproveitadas e novas executam de forma limpa em base zerada

### Authentication (Autenticação)

- [x] **AUTH-01**: Usuário faz login com e-mail e senha e recebe sessão JWT
- [x] **AUTH-02**: Sessão do usuário persiste entre refreshes do navegador
- [x] **AUTH-03**: Usuário pode recuperar/redefinir senha (forgot/reset) reaproveitando o fluxo do FB_APU04
- [x] **AUTH-04**: Rotas da API são protegidas por middleware de autenticação

### Tenancy (Gestão de ambiente/empresa/usuário — simplificada)

- [x] **TEN-01**: Existe ao menos um ambiente/empresa (Ferreira Costa) configurável com usuário administrador
- [x] **TEN-02**: Admin pode gerenciar usuários (criar/editar/desativar) reaproveitando a gestão do FB_APU04
- [x] **TEN-03**: Contexto de empresa (Ferreira Costa) é resolvido nas requisições para escopar os dados

### XML Import (Importação de XMLs de saída)

- [x] **XML-01**: Usuário importa um ou vários XMLs completos de NFe de saída via tela reaproveitada do FB_APU04
- [x] **XML-02**: Sistema faz parse e persiste cabeçalho da nota, itens e impostos do XML no Postgres
- [x] **XML-03**: Usuário visualiza as notas/itens importados e os valores de imposto originais do XML
- [x] **XML-04**: Importação reporta erros de parse/validação de XML de forma clara

### Fiscal Group Lookup (Grupo fiscal via ERP_BRIDGE / Oracle)

- [x] **ERP-01**: Conexão Oracle de leitura é configurável (reaproveitando credenciais/infra do ERP_BRIDGE)
- [x] **ERP-02**: Para cada item do XML, sistema consulta `prod` + `PRODB` e obtém o grupo fiscal do produto
- [x] **ERP-03**: Itens sem grupo fiscal localizado são sinalizados sem interromper o processamento dos demais

### Fiscal Package Execution (Execução do pacote fiscal no FCCORP_BKP)

- [x] **FIS-01**: Sistema executa o script do pacote fiscal no FCCORP_BKP (mesma instância Oracle) passando parâmetros herdados do XML de origem + o grupo fiscal lido de PROD/PRODB
- [x] **FIS-02**: Sistema carrega e persiste o retorno do script (impostos calculados) vinculado ao item correspondente
- [x] **FIS-03**: Falhas na execução do script por item são capturadas e exibidas sem abortar o lote

### Comparison Screen (Tela de comparação visual)

- [x] **CMP-01**: Tela lista, item a item, os impostos esperado (XML) vs. calculado (script): base ICMS, vlr ICMS, base ST, ICMS ST, base PIS/COFINS, vlr PIS/COFINS, DIFAL, FCP, entre outros
- [x] **CMP-02**: Divergências entre esperado e calculado são destacadas visualmente (ex.: cor/ícone) com a diferença numérica
- [x] **CMP-03**: Usuário pode filtrar/visualizar apenas itens com divergência
- [x] **CMP-04**: Tela apresenta um resumo (total de itens, itens OK, itens divergentes) por nota e/ou por lote importado

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Automation

- **AUTO-01**: Suíte de testes Go automatizada usando os XMLs como fixtures, com asserts contra o esperado, rodando em CI
- **AUTO-02**: Exportação dos resultados de comparação (CSV/Excel)

### Multi-tenant

- **MTN-01**: Suporte completo a múltiplas empresas com troca de contexto (CompanySwitcher) para testar além da Ferreira Costa

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Multi-tenant completo / troca de empresas | v1 foca só na Ferreira Costa; simplificar a complexidade herdada |
| Importação de XMLs de entrada, CTe, EFD/SPED | Fora do propósito de testar o pacote fiscal de saídas |
| Painéis de apuração da Reforma Tributária e demais módulos do FB_APU04 | Não relacionados ao objetivo de teste |
| Suíte de testes Go automatizada / CI | Entregável v1 é a tela de comparação visual (movido para v2) |
| Escrita no FCCORP_BKP ou no ERP de produção | Acesso somente leitura para validação |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FND-01 | Phase 1 | Complete |
| FND-02 | Phase 1 | Complete |
| FND-03 | Phase 1 | Complete |
| AUTH-01 | Phase 1 | Complete |
| AUTH-02 | Phase 1 | Complete |
| AUTH-03 | Phase 1 | Complete |
| AUTH-04 | Phase 1 | Complete |
| TEN-01 | Phase 1 | Complete |
| TEN-02 | Phase 1 | Complete |
| TEN-03 | Phase 1 | Complete |
| XML-01 | Phase 2 | Complete |
| XML-02 | Phase 2 | Complete |
| XML-03 | Phase 2 | Complete |
| XML-04 | Phase 2 | Complete |
| ERP-01 | Phase 2 | Complete |
| ERP-02 | Phase 2 | Complete |
| ERP-03 | Phase 2 | Complete |
| FIS-01 | Phase 2 | Complete |
| FIS-02 | Phase 2 | Complete |
| FIS-03 | Phase 2 | Complete |
| CMP-01 | Phase 3 | Complete |
| CMP-02 | Phase 3 | Complete |
| CMP-03 | Phase 3 | Complete |
| CMP-04 | Phase 3 | Complete |

**Coverage:**

- v1 requirements: 24 total
- Mapped to phases: 24 ✓
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-30*
*Last updated: 2026-06-30 after roadmap creation*
