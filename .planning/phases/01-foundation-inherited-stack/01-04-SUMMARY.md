---
phase: 01-foundation-inherited-stack
plan: 04
subsystem: backend-erp-bridge
tags: [go, oracle, go-ora, erp-bridge, migrations, crypto, aes-gcm]
dependency_graph:
  requires: [plan-01-backend-auth, plan-03-hierarchy-admin]
  provides: [erp-bridge-config-endpoint, erp-bridge-test-connection-endpoint, erp-bridge-credentials-endpoint, schema-erp-bridge]
  affects: [plan-05-frontend-management]
tech_stack:
  added:
    - "github.com/sijms/go-ora/v2 v2.9.0 — driver Oracle puro-Go (CGO_ENABLED=0), registrado via import _ como driver 'oracle' para database/sql"
  patterns:
    - "sql.Open(\"oracle\", dsn) + PingContext timeout 10s — teste de conexão Oracle sem consultar prod/PRODB"
    - "DSN go-ora: suporte a URL completa (oracle://user:pass@host:port/service) e Easy Connect (host:port/service)"
    - "DecryptFieldWithFallback server-side antes de montar DSN — credenciais Oracle nunca saem do servidor em claro"
key_files:
  created:
    - backend/handlers/erp_bridge.go
    - backend/migrations/002_erp_bridge.sql
    - backend/migrations/005_seed_erp_bridge_ferreira_costa.sql
  modified:
    - backend/go.mod
    - backend/go.sum
    - backend/main.go
decisions:
  - "ERPBridgeTestConnectionHandler suporta dois formatos de DSN: URL Oracle completa (oracle://) e Easy Connect — detectado via strings.HasPrefix para máxima compatibilidade com instalações Oracle variadas"
  - "ERPBridgeCredentialsHandler retorna http.HandlerFunc (não http.Handler como no FB_APU04) para compatibilidade direta com withDB sem wrapping adicional"
  - "Handlers daemon/worker não copiados (Run/Heartbeat/Trigger/Pending/Servidores) — mantidos fora do escopo da Fase 1 conforme D-14"
metrics:
  duration: "~25min"
  completed: "2026-06-30"
  tasks_completed: 2
  files_created: 3
  files_modified: 3
---

# Phase 1 Plan 04: ERP Bridge Backend (go-ora + test-connection) Summary

Driver Oracle puro-Go adicionado ao go.mod; handler erp_bridge completo com endpoint NOVO `POST /api/erp-bridge/test-connection` usando `sql.Open("oracle", dsn) + PingContext 10s`; migrações 002 (DDL erp_bridge) e 005 (seed Ferreira Costa) presentes; rotas registradas no main.go com autenticação.

## What Was Built

### Task 1: go-ora/v2 + handler erp_bridge + endpoint test-connection

**Commit:** `e7b8d98`

- `backend/go.mod` e `backend/go.sum`: `github.com/sijms/go-ora/v2 v2.9.0` adicionado via `go get` + `go mod tidy`. Driver Oracle puro-Go, CGO_ENABLED=0, importado via `import _ "github.com/sijms/go-ora/v2"` para registro do driver "oracle" no `database/sql`.

- `backend/handlers/erp_bridge.go` (criado): Copiados do FB_APU04 — `ERPBridgeConfigHandler` (GET/PATCH `/api/erp-bridge/config`, salva credenciais Oracle criptografadas AES-GCM, retorna `*_set` flag em vez de `oracle_senha`), `ERPBridgeGenerateAPIKeyHandler` (POST, restrito a admin via withAuth), `ERPBridgeCredentialsHandler` (GET via X-API-Key para daemon futuro), e o helper `erpBridgeGetCompany`. Imports `fb_apu04/` renomeados para `fb_testesfc/`.

- `ERPBridgeTestConnectionHandler` (NOVO — D-14): Resolve empresa via `erpBridgeGetCompany`, lê `oracle_dsn/oracle_usuario/oracle_senha` de `erp_bridge_config`, descriptografa via `DecryptFieldWithFallback`. Detecta formato do DSN: URL Oracle completa (`oracle://...`) → usa diretamente; Easy Connect (`host:port/service`) → monta URL com `fmt.Sprintf("oracle://%s:%s@%s", ...)`. Abre conexão via `sql.Open("oracle", connStr)` + `PingContext` timeout 10s. DSN vazio → `{"ok":false,"error":"DSN Oracle não configurado"}`. Oracle inacessível → `{"ok":false,"error":"<dial/ping error>"}`. Sucesso → `{"ok":true}`. Credenciais NUNCA retornadas ao frontend (T-04-01, T-04-02).

- `backend/migrations/002_erp_bridge.sql`: Cópia direta do `065_erp_bridge.sql` do FB_APU04 — tabelas `erp_bridge_config` (PK company_id), `erp_bridge_runs`, `erp_bridge_run_items`, `erp_bridge_servidores`, `parceiros`. Apenas DDL, sem seed.

- `backend/migrations/005_seed_erp_bridge_ferreira_costa.sql`: `INSERT INTO erp_bridge_config (company_id) SELECT id FROM companies WHERE name='Ferreira Costa' ON CONFLICT (company_id) DO NOTHING` — linha idempotente, depende da 002 (tabela) e 004 (empresa), executa depois de ambas por ordenação alfabética.

### Task 2: Rotas ERP Bridge no main.go

**Commit:** `8f681d4`

- Bloco ERP Bridge acrescentado ao `main.go` sem remover rotas existentes:
  - `GET|PATCH /api/erp-bridge/config` → `withAuth(ERPBridgeConfigHandler, "")`
  - `POST /api/erp-bridge/config/generate-api-key` → `withAuth(ERPBridgeGenerateAPIKeyHandler, "admin")` (T-04-05)
  - `POST /api/erp-bridge/test-connection` → `withAuth(ERPBridgeTestConnectionHandler, "")` (NOVO, D-14; T-04-03 SSRF mitigado)
  - `GET /api/erp-bridge/credentials` → `withDB(ERPBridgeCredentialsHandler)` (X-API-Key, daemon futuro)

## Verification Evidence

```
go build ./...: EXIT 0
go.mod sijms/go-ora/v2: OK
ERPBridgeTestConnectionHandler: OK
msg DSN vazio: OK
002 DDL (CREATE TABLE): OK
002 sem INSERT: OK (DDL-only verificado)
005 INSERT presente: OK
005 Ferreira Costa: OK
sem handlers daemon: OK (Heartbeat/Trigger/Pending/Servidores não copiados)
rota test-connection: OK
rota config: OK
```

## Deviations from Plan

### Auto-ajuste de tipo de retorno — ERPBridgeCredentialsHandler

- **Encontrado durante:** Task 2 (registro de rotas)
- **Problema:** No FB_APU04, `ERPBridgeCredentialsHandler` retorna `http.Handler` (interface mais ampla). O wrapper `withDB` espera `func(*sql.DB) http.HandlerFunc`.
- **Correção:** Na implementação do handler para este projeto, a função retorna `http.HandlerFunc` diretamente, eliminando a necessidade de `.ServeHTTP` wrapping ou uma adaptação especial no main.go. A assinatura mantém compatibilidade com o padrão `withDB`.
- **Arquivos:** `backend/handlers/erp_bridge.go`
- **Regra:** Rule 3 (auto-fix blocking issue)

### DSN fallback Easy Connect adicionado

- **Encontrado durante:** Task 1 (implementação de ERPBridgeTestConnectionHandler)
- **Conforme RESEARCH.md § A1:** Suposição A1 indica "sintaxe do DSN pode ser diferente; resolver na implementação com a doc oficial do pacote". O go-ora/v2 suporta ambos os formatos.
- **Correção:** Detecção do formato via `strings.HasPrefix(dsnPlain, "oracle://")` — URL completa usada diretamente; Easy Connect montado com usuario/senha. Isso é o comportamento documentado em RESEARCH.md como "fallback Easy Connect".
- **Arquivos:** `backend/handlers/erp_bridge.go`
- **Regra:** Rule 2 (funcionalidade crítica ausente — robustez do DSN)

## Known Stubs

Nenhum. Todas as funcionalidades implementadas são operacionais (handler chama Oracle real via go-ora, credenciais lidas do banco, AES-GCM descriptografado corretamente). A ausência de Oracle configurado retorna `{"ok":false}` com mensagem de erro descritiva — comportamento esperado e documentado.

## Threat Surface Scan

Nenhuma superfície nova além do registrado no threat model do PLAN.md:

| Threat ID | Mitigação Implementada |
|-----------|------------------------|
| T-04-01 | `oracle_senha` → `oracle_senha_set: bool`; `DecryptFieldWithFallback` apenas server-side |
| T-04-02 | Erro de Ping logado via `log.Printf` sem senha; mensagem ao frontend não inclui credencial |
| T-04-03 | `withAuth` em test-connection; usa só DSN de `erp_bridge_config` (empresa do JWT), não host do corpo |
| T-04-04 | go-ora/v2@v2.9.0 verificado via proxy.golang.org (checkpoint blocking-human aprovado); go.sum com hash |
| T-04-05 | `withAuth(..., "admin")` em generate-api-key |

## Self-Check: PASSED

Arquivos criados:
- `backend/handlers/erp_bridge.go` — FOUND
- `backend/migrations/002_erp_bridge.sql` — FOUND (commit 19f4028)
- `backend/migrations/005_seed_erp_bridge_ferreira_costa.sql` — FOUND (commit 19f4028)

Commits verificados:
- `19f4028` — migrações 002 e 005 (executor anterior)
- `e7b8d98` — go-ora + handler erp_bridge
- `8f681d4` — rotas ERP Bridge no main.go
