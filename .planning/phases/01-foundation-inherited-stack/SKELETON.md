# Walking Skeleton — FB_TESTESFC (Validador de Testes do Pacote Fiscal)

**Phase:** 1
**Generated:** 2026-06-30

## Capability Proven End-to-End

> A menor capacidade visível ao usuário que exercita a pilha inteira.

"Com `docker compose up`, o backend Go aplica as migrações em base Postgres zerada e semeia a empresa Ferreira Costa + o admin; o usuário acessa http://localhost:3004, faz login com `claudio_bezerra@hotmail.com` / `123456`, recebe uma sessão JWT e **permanece autenticado após dar refresh (F5)** no navegador — provando frontend → API → cookie de sessão → Postgres de ponta a ponta."

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Linguagem/Backend | Go 1.24, `net/http` + `http.DefaultServeMux`, handlers + services | Stack do FB_APU04 reaproveitada por cópia seletiva (D-01); zero fricção de port |
| Camada de dados (app) | Postgres 15 (`lib/pq`), schema via SQL puro | Padrão maduro do FB_APU04; sem ORM |
| Runner de migração | Custom embutido em `main.go` (glob `migrations/*.sql` + tabela `schema_migrations`, ordem alfabética, idempotente) | D-07; sem lib externa; transplantado do FB_APU04 (linhas 112-222) |
| Migrações | 4 arquivos consolidados (001 auth+hierarquia, 002 erp_bridge, 003 managers, 004 seed Ferreira Costa) | D-05/D-06; não copiar as 149 migrações (Pitfall do prefixo duplicado 021) |
| Auth | JWT HS256 (`golang-jwt/v5`) — access token em memória React + refresh token em cookie httpOnly SameSite=Strict; bcrypt cost 14 | Copiado do FB_APU04; mais seguro que token em localStorage |
| Tenancy | Modelo hierárquico ambiente→grupo→empresa→usuário mantido; semeada APENAS Ferreira Costa + admin; contexto por header `X-Company-ID` + `GetEffectiveCompanyID` | D-08/D-11; multi-tenant trivial de reativar na v2; sem CompanySwitcher |
| Oracle (ERP_BRIDGE) | Driver puro-Go `github.com/sijms/go-ora/v2` (CGO_ENABLED=0); apenas open+ping em `/api/erp-bridge/test-connection` | D-14; NÃO consultar prod/PRODB nesta fase (Fase 2) |
| Frontend | React 18 + TypeScript + Vite + Tailwind + shadcn/ui; shell compartilhado + 9 páginas no escopo | D-04; copiar shell, registrar só as rotas da fase |
| Deploy local | `docker compose up` com serviços api(8085)+web(3004)+db; portas/volume próprios | D-13; coexiste com o FB_APU04 (8084/3003) na mesma máquina |
| Module path | `fb_testesfc` (go.mod + imports renomeados de `fb_apu04`) | D-12 |
| Diretórios | `backend/{handlers,services,migrations}`, `frontend/src/{pages,components/ui,contexts,lib}` | Espelha o FB_APU04 |

## Stack Touched in Phase 1

- [x] Project scaffold — go.mod enxuto + `go mod tidy`, Vite + package.json, Dockerfiles, docker-compose
- [x] Routing — backend registra rotas reais; frontend roteia /login + AppLayout protegido
- [x] Database — leitura real (login consulta `users`) E escrita real (migrações criam schema + seed insere Ferreira Costa/admin; reset/create de usuário no Plan 03/05)
- [x] UI — Login.tsx wired a `POST /api/auth/login`; botão "Testar Conexão Oracle" wired a `POST /api/erp-bridge/test-connection`
- [x] Deployment — `docker compose up --build` sobe a pilha completa em dev (api+web+db)

## Out of Scope (Deferred to Later Slices)

> O que **não** está no skeleton. Lista explícita para impedir re-litígio da minimalidade da Fase 1.

- Importação/parse de XMLs de NFe de saída e persistência de cabeçalho/itens/impostos → Fase 2 (D-06)
- Lookup de grupo fiscal em `prod`/`PRODB` no Oracle → Fase 2 (D-14 limita ERP_BRIDGE a test-connection)
- Execução do pacote fiscal no FCCORP_BKP → Fase 2
- Tela de comparação visual item a item (core value) → Fase 3
- Multi-tenant completo / CompanySwitcher (MTN-01) → v2 (modelo mantido pronto)
- Endurecimento da credencial padrão do admin (.env / troca obrigatória no 1º login) → v1 usa default fixo `123456` (D-10, risco de dev aceito)
- Redis, Prometheus, Grafana, Alertmanager, postgres-exporter → não incluídos (D-13)
- Suíte de testes Go automatizada / CI → v2 (AUTO-01)

## Subsequent Slice Plan

Cada fase posterior acrescenta uma fatia vertical sobre este skeleton sem alterar suas decisões arquiteturais:

- **Phase 2:** Usuário importa XMLs de NFe de saída; sistema faz lookup do grupo fiscal em prod/PRODB e executa o pacote fiscal no FCCORP_BKP, persistindo os impostos calculados (erros isolados por item).
- **Phase 3:** Tela de comparação visual item a item (esperado do XML vs. calculado pelo script), com divergências destacadas, filtro de divergentes e resumo por nota/lote.
