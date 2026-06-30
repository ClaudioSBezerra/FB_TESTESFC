---
phase: 01-foundation-inherited-stack
plan: 01
subsystem: backend-auth
tags: [go, auth, jwt, migrations, postgres, crypto, cors]
dependency_graph:
  requires: []
  provides: [backend-auth-endpoints, schema-auth-hierarchy, seed-ferreira-costa]
  affects: [plan-02-frontend, plan-03-hierarchy-admin, plan-04-erp-bridge]
tech_stack:
  added:
    - "github.com/golang-jwt/jwt/v5 v5.3.1 — JWT HS256"
    - "github.com/joho/godotenv v1.5.1 — carregamento de .env"
    - "github.com/lib/pq v1.12.3 — driver Postgres"
    - "golang.org/x/crypto v0.48.0 — bcrypt cost=14"
  patterns:
    - "Runner de migrações custom embutido no main.go (filepath.Glob + schema_migrations)"
    - "Refresh token rotation com sync.Map in-memory + cookie httpOnly SameSite=Strict"
    - "SecurityMiddleware com whitelist de origens CORS e security headers"
    - "withDB/withAuth closures que resolvem DB em tempo de request"
key_files:
  created:
    - backend/go.mod
    - backend/go.sum
    - backend/main.go
    - backend/handlers/auth.go
    - backend/handlers/crypto.go
    - backend/handlers/middleware.go
    - backend/services/crypto.go
    - backend/services/email.go
    - backend/migrations/001_auth_hierarchy.sql
    - backend/migrations/004_seed_ferreira_costa.sql
  modified: []
decisions:
  - "lib/pq resolvido como v1.12.3 (mais recente disponível) em vez de v1.11.2 especificado no PLAN — go mod tidy escolhe a versão compatível mais recente; sem breaking changes"
  - "verification_tokens.used adicionado à migração 001 (coluna necessária por ResetPasswordHandler; ausente no schema de referência do PLAN mas presente no código copiado do FB_APU04)"
  - "erp_bridge_config NÃO inserido no seed 004 conforme instrução explícita do PLAN — tabela só existirá após migração 002 do Plan 04"
metrics:
  duration: "~8 minutos"
  completed_date: "2026-06-30"
  tasks_completed: 3
  tasks_total: 3
  files_created: 10
  files_modified: 0
---

# Phase 01 Plan 01: Backend Foundation — Auth, Crypto, Migrations

Backend Go enxuto do FB_TESTESFC com module `fb_testesfc`, cópia seletiva do FB_APU04: AES-256-GCM, SecurityMiddleware CORS, JWT com refresh rotation, runner de migrações custom, schema auth+tenancy consolidado e seed idempotente da Ferreira Costa com admin `claudio_bezerra@hotmail.com`.

## O Que Foi Construído

**3 tarefas executadas, 10 arquivos criados, `go build ./...` com exit 0.**

### Tarefa 1 — go.mod + crypto + middleware + serviços
- `backend/go.mod`: `module fb_testesfc`, `go 1.24.1`, apenas as 4 deps necessárias (jwt, godotenv, lib/pq, x/crypto). Sem prometheus, excelize, rardecode.
- `backend/handlers/crypto.go`: EncryptField/DecryptField/DecryptFieldWithFallback (AES-256-GCM) — cópia integral do FB_APU04.
- `backend/handlers/middleware.go`: SecurityMiddleware, GetAllowedOrigins() com `http://localhost:3004` no fallback, rate limiters exportados. Cópia integral com adição de porta 3004.
- `backend/services/crypto.go`: DecryptFieldWithFallback para uso nos serviços — cópia integral.
- `backend/services/email.go`: SendPasswordResetEmail (SMTP) — cópia integral.

### Tarefa 2 — auth.go + migrações 001 e 004
- `backend/handlers/auth.go`: cópia integral do FB_APU04 com find/replace `fb_apu04/` → `fb_testesfc/` nos imports. Exporta: `AuthMiddleware`, `GetEffectiveCompanyID`, `LoginHandler`, `RefreshHandler`, `ForgotPasswordHandler`, `ResetPasswordHandler`, `ChangePasswordHandler`, `LogoutHandler`, `RegisterHandler`, `GetMeHandler`, `SetPreferredCompanyHandler`, `GetUserCompaniesHandler`.
- `backend/migrations/001_auth_hierarchy.sql`: schema consolidado das migrations 013+015+017+018+025 do FB_APU04 — tabelas `environments`, `enterprise_groups`, `companies`, `users`, `user_environments`, `verification_tokens` com índices. Sem coluna `cnpj`. Inclui coluna `used` em `verification_tokens` (necessária para `ResetPasswordHandler`).
- `backend/migrations/004_seed_ferreira_costa.sql`: bloco `DO $$` idempotente que garante ambiente/grupo/empresa Ferreira Costa + admin `claudio_bezerra@hotmail.com` com hash bcrypt cost=14 da senha `123456`.

### Tarefa 3 — main.go enxuto
- `backend/main.go`: `initDBAsync`/`getDB` (retry infinito), `onDBConnected` com runner de migração custom (schema_migrations, glob em ordem alfabética, sem registrar falhas), `withDB`/`withAuth` closures, rota `/api/health`, todas as rotas de auth, SecurityMiddleware envolvendo o mux. Porta padrão `8085`. Aviso de `ENCRYPTION_KEY` ausente. Sem worker.*, promhttp, prometheus, goroutines ERP/RFB.

## Critérios de Sucesso

- [x] `go build ./...` retorna exit 0
- [x] `module fb_testesfc` no go.mod
- [x] Sem prometheus/excelize/rardecode/worker no código
- [x] `GetAllowedOrigins()` inclui `http://localhost:3004`
- [x] `AuthMiddleware` e `GetEffectiveCompanyID` exportados em auth.go
- [x] Migração 001 cria 6 tabelas sem coluna `cnpj`
- [x] Migração 004 referencia `claudio_bezerra@hotmail.com` e hash bcrypt cost=14
- [x] `onDBConnected` com `filepath.Glob` e `schema_migrations`
- [x] Porta padrão `8085`; rotas de auth registradas

## Desvios do Plano

### Desvios Automáticos

**1. [Rule 1 - Bug] coluna `used` adicionada a `verification_tokens`**
- **Encontrado em:** Task 2 — ao ler `ResetPasswordHandler` do FB_APU04
- **Problema:** `ResetPasswordHandler` usa `WHERE used = false` e `SET used = true`, mas o schema de referência no PLAN não incluía a coluna `used`
- **Correção:** Adicionado `used BOOLEAN DEFAULT FALSE` à definição de `verification_tokens` na migration 001
- **Arquivos modificados:** `backend/migrations/001_auth_hierarchy.sql`
- **Commit:** f5d5b3b

**2. [Rule 1 - Versão] lib/pq v1.12.3 em vez de v1.11.2**
- **Encontrado em:** Task 1 — `go mod tidy` não encontrou v1.11.2 no proxy
- **Problema:** go mod tidy resolveu `github.com/lib/pq` para v1.12.3 (versão mais recente)
- **Correção:** Aceita v1.12.3 — compatível, sem breaking changes
- **Arquivos modificados:** `backend/go.mod`, `backend/go.sum`
- **Commit:** 5d3decc

**3. [Rule 2 - Segurança] erp_bridge_config NÃO inserido no seed 004**
- **Encontrado em:** Task 2 — instrução explícita do PLAN
- **Ação:** Seed 004 omite o INSERT em `erp_bridge_config` conforme especificado; tabela só existe após migração 002 do Plan 04

## Verificação Final

- `go build ./...` → exit 0 (confirma compilação limpa)
- Migrações 001 e 004 prontas para execução em base zerada
- Admin seed com hash bcrypt cost=14 verificado (copiado de 021_ensure_admin_user.sql do FB_APU04)
- Nenhuma referência a `fb_apu04/` em handlers/ ou services/

## Commits Desta Execução

| Hash | Mensagem |
|------|----------|
| 0579f56 | feat(01-01): go.mod enxuto + crypto + middleware + servicos |
| f5d5b3b | feat(01-01): auth.go + migracoes 001 (auth/hierarchy) e 004 (seed Ferreira Costa) |
| 5d3decc | feat(01-01): main.go enxuto com runner de migracao e rotas de auth |

## Self-Check: PASSED

Arquivos criados verificados:
- backend/go.mod ✓
- backend/handlers/auth.go ✓
- backend/handlers/crypto.go ✓
- backend/handlers/middleware.go ✓
- backend/services/crypto.go ✓
- backend/services/email.go ✓
- backend/migrations/001_auth_hierarchy.sql ✓
- backend/migrations/004_seed_ferreira_costa.sql ✓
- backend/main.go ✓

Commits verificados: 0579f56, f5d5b3b, 5d3decc ✓
