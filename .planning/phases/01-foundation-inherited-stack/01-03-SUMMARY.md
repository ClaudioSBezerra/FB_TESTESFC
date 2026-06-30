---
phase: 01-foundation-inherited-stack
plan: 03
subsystem: backend-tenancy
tags: [go, admin, hierarchy, managers, tenancy, jwt, postgres]
dependency_graph:
  requires: [plan-01-backend-auth]
  provides: [backend-admin-endpoints, backend-config-endpoints, backend-managers-endpoints, schema-managers]
  affects: [plan-04-erp-bridge, plan-05-frontend-pages]
tech_stack:
  added: []
  patterns:
    - "Cópia seletiva de admin.go: apenas 5 handlers de usuário (sem SPED/reset)"
    - "environment.go adaptado ao schema sem cnpj/cnae_secundario/municipio/incentivos_fiscais"
    - "hierarchy.go com LEFT JOIN import_jobs que retorna branches=[] na Fase 1 sem erro"
    - "managers.go com GetEffectiveCompanyID para escopo por empresa (TEN-03)"
    - "Rotas multi-method inline em main.go para /api/config/* (switch r.Method)"
key_files:
  created:
    - backend/handlers/admin.go
    - backend/handlers/environment.go
    - backend/handlers/hierarchy.go
    - backend/handlers/managers.go
    - backend/migrations/003_managers.sql
  modified:
    - backend/main.go
decisions:
  - "environment.go adaptado sem cnpj/cnae_secundario/municipio/incentivos_fiscais — schema do FB_TESTESFC parte do estado final (migration 001 não criou essas colunas)"
  - "admin.go copiado somente com 5 handlers de usuário — sem ResetDatabaseHandler/ResetCompanyDataHandler/DiagnosticDataHandler/AdminNFCancelamentoHandler/RefreshViewsHandler (dependem de tabelas SPED inexistentes)"
  - "hierarchy.go mantém LEFT JOIN em import_jobs com P7 — tabela não existe na Fase 1 mas a query retorna branches=[] sem erro"
metrics:
  duration: "~12 minutos"
  completed_date: "2026-06-30"
  tasks_completed: 2
  tasks_total: 2
  files_created: 5
  files_modified: 1
---

# Phase 01 Plan 03: Camada de Gestão (Tenancy) — Admin, Hierarquia, Managers

Handlers de gestão (admin/environment/hierarchy/managers) copiados seletivamente do FB_APU04, migração 003 criando a tabela `managers`, e rotas registradas no `main.go`. Backend compila com `go build ./...` exit 0. Entrega TEN-02 (gestão de usuários via `/api/admin/users*`) e TEN-03 (contexto de empresa via `GetEffectiveCompanyID`).

## O Que Foi Construído

**2 tarefas executadas, 5 arquivos criados, 1 modificado, `go build ./...` exit 0.**

### Tarefa 1 — Handlers de gestão + migração 003

- `backend/handlers/admin.go`: 5 handlers de usuário (`ListUsersHandler`, `CreateUserHandler`, `PromoteUserHandler`, `DeleteUserHandler`, `ReassignUserHandler`). Sem `ResetDatabaseHandler`, `ResetCompanyDataHandler`, `DiagnosticDataHandler`, `AdminNFCancelamentoHandler`, `RefreshViewsHandler` (dependentes de tabelas SPED/mv_mercadorias inexistentes).
- `backend/handlers/environment.go`: CRUD de ambientes/grupos/empresas. Adaptado ao schema do FB_TESTESFC: queries de `companies` removeram as colunas `cnpj`, `cnae_secundario`, `municipio`, `incentivos_fiscais` que não existem no schema local (migration 001). Sem imports de módulo interno.
- `backend/handlers/hierarchy.go`: `GetUserHierarchyHandler` com `GetEffectiveCompanyID` e LEFT JOIN em `import_jobs` — tabela não existe na Fase 1 mas retorna `branches=[]` sem erro (P7 do PATTERNS.md).
- `backend/handlers/managers.go`: CRUD completo (`ListManagersHandler`, `CreateManagerHandler`, `UpdateManagerHandler`, `DeleteManagerHandler`, `GetActiveManagersByCompany`). Usa `GetEffectiveCompanyID` para escopo por empresa em todos os handlers.
- `backend/migrations/003_managers.sql`: tabela `managers` + 3 índices + trigger `update_managers_updated_at`. Cópia direta de `046_create_managers_table.sql` do FB_APU04.

### Tarefa 2 — Registro de rotas no main.go

- Rotas admin via `withAuth(..., "admin")`: `/api/admin/users` (List), `/api/admin/users/create` (Create), `/api/admin/users/promote` (Promote), `/api/admin/users/delete` (Delete), `/api/admin/users/reassign` (Reassign).
- Hierarquia via `withAuth(..., "")` com padrão multi-method inline (switch em `r.Method`): `/api/config/environments`, `/api/config/groups`, `/api/config/companies` (GET/POST/PUT/DELETE mapeados aos handlers).
- `/api/user/hierarchy` com `withAuth(..., "")` — `GetUserHierarchyHandler`.
- Managers via `withAuth(..., "")`: `/api/managers` (List), `/api/managers/create` (Create), `/api/managers/` (Update/Delete por ID).
- Rotas de auth do Plan 01 mantidas sem alteração.

## Critérios de Sucesso

- [x] `go build ./...` retorna exit 0
- [x] `admin.go` contém os 5 handlers de usuário e nenhuma função de reset/diagnóstico SPED
- [x] `environment.go`, `hierarchy.go`, `managers.go` copiados; nenhum import `"fb_apu04/"` remanescente
- [x] `migrations/003_managers.sql` cria a tabela `managers`
- [x] `GetEffectiveCompanyID` em `managers.go` (TEN-03)
- [x] Rotas `/api/admin/users*`, `/api/config/{environments,groups,companies}`, `/api/managers*` registradas
- [x] Rotas de auth do Plan 01 permanecem registradas

## Desvios do Plano

### Desvios Automáticos

**1. [Rule 1 - Bug] environment.go adaptado: colunas ausentes no schema removidas das queries**
- **Encontrado em:** Task 1 — análise do schema 001_auth_hierarchy.sql vs environment.go do FB_APU04
- **Problema:** `GetCompaniesHandler`, `CreateCompanyHandler` e `UpdateCompanyHandler` referenciam colunas `cnpj`, `cnae_secundario`, `municipio`, `incentivos_fiscais` que não existem no schema do FB_TESTESFC (migration 001 parte do estado final sem essas colunas). Compilaria mas falharia em runtime com erro de coluna inexistente.
- **Correção:** Struct `Company` simplificada (sem CNPJ, CNAESecundario, Municipio, IncentivosFiscais); queries de SELECT, INSERT e UPDATE em `companies` adaptadas ao schema disponível. Import `"github.com/lib/pq"` (necessário apenas para `pq.Array`) removido (não mais usado).
- **Arquivos modificados:** `backend/handlers/environment.go`
- **Commit:** 2982c67

## Verificação Final

- `go build ./...` → exit 0
- Sem imports `"fb_apu04/"` em nenhum handler
- `admin.go` contém apenas os 5 handlers de usuário; sem funções SPED
- `003_managers.sql` cria tabela `managers` com índices e trigger
- Rotas `/api/admin/users`, `/api/config/environments`, `/api/managers` presentes em `main.go`

## Commits Desta Execução

| Hash | Mensagem |
|------|----------|
| 2982c67 | feat(01-03): handlers de gestao (admin/environment/hierarchy/managers) + migração 003 |
| b6019d9 | feat(01-03): registrar rotas admin/config/managers no main.go |

## Self-Check: PASSED

Arquivos criados verificados:
- backend/handlers/admin.go ✓
- backend/handlers/environment.go ✓
- backend/handlers/hierarchy.go ✓
- backend/handlers/managers.go ✓
- backend/migrations/003_managers.sql ✓
- backend/main.go (modificado) ✓

Commits verificados: 2982c67, b6019d9 ✓
