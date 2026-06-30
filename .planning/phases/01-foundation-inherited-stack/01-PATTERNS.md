# Fase 1: Foundation & Inherited Stack — Mapa de Padrões

**Mapeado:** 2026-06-30
**Arquivos analisados:** 39 (novos/modificados/copiados)
**Análogos encontrados:** 37 / 39

---

## Classificação de Arquivos

| Arquivo Alvo (FB_TESTESFC) | Papel | Fluxo de Dados | Análogo Mais Próximo (FB_APU04) | Qualidade |
|---|---|---|---|---|
| `backend/main.go` | config/roteador | request-response | `backend/main.go` linhas 1–443 | exato (transplante) |
| `backend/go.mod` | config | — | `backend/go.mod` | exato (ajuste de module path) |
| `backend/handlers/auth.go` | controller | request-response | `backend/handlers/auth.go` | exato |
| `backend/handlers/admin.go` | controller | CRUD | `backend/handlers/admin.go` | exato |
| `backend/handlers/crypto.go` | utility | transform | `backend/handlers/crypto.go` | exato |
| `backend/handlers/environment.go` | controller | CRUD | `backend/handlers/environment.go` | exato |
| `backend/handlers/hierarchy.go` | controller | request-response | `backend/handlers/hierarchy.go` | exato |
| `backend/handlers/managers.go` | controller | CRUD | `backend/handlers/managers.go` | exato |
| `backend/handlers/middleware.go` | middleware | request-response | `backend/handlers/middleware.go` | exato |
| `backend/handlers/erp_bridge.go` | controller | CRUD | `backend/handlers/erp_bridge.go` | exato + endpoint novo |
| `backend/services/email.go` | service | event-driven | `backend/services/email.go` | exato |
| `backend/services/crypto.go` | service | transform | `backend/services/crypto.go` | exato |
| `backend/migrations/001_auth_hierarchy.sql` | migration | batch | `migrations/013`, `015`, `017`, `018`, `023`, `025` | role-match (consolidação) |
| `backend/migrations/002_erp_bridge.sql` | migration | batch | `migrations/065_erp_bridge.sql` | exato |
| `backend/migrations/003_managers.sql` | migration | batch | `migrations/046_create_managers_table.sql` | exato |
| `backend/migrations/004_seed_ferreira_costa.sql` | migration | batch | `migrations/016`, `021`, `024` | role-match (adaptação) |
| `docker-compose.yml` | config | — | `docker-compose.yml` | role-match (portas distintas) |
| `backend/Dockerfile` | config | — | `backend/Dockerfile` | exato (remover -mod=vendor) |
| `.env.example` | config | — | `.env.example` | role-match (+ ENCRYPTION_KEY) |
| `frontend/src/App.tsx` | component | request-response | `frontend/src/App.tsx` | role-match (enxuto, sem FilialProvider) |
| `frontend/src/contexts/AuthContext.tsx` | provider | request-response | `frontend/src/contexts/AuthContext.tsx` | exato |
| `frontend/src/lib/navigation.ts` | utility | transform | `frontend/src/lib/navigation.ts` | role-match (conteúdo reescrito) |
| `frontend/src/lib/utils.ts` | utility | transform | `frontend/src/lib/utils.ts` | exato |
| `frontend/src/lib/logger.ts` | utility | transform | `frontend/src/lib/logger.ts` | exato |
| `frontend/src/components/AppRail.tsx` | component | request-response | `frontend/src/components/AppRail.tsx` | role-match (simplificar mainItems) |
| `frontend/src/components/ui/` (44 arquivos) | component | — | `frontend/src/components/ui/` | exato |
| `frontend/vite.config.ts` | config | — | `frontend/vite.config.ts` | exato (mudar porta/proxy) |
| `frontend/package.json` | config | — | `frontend/package.json` | exato (mudar name) |
| `frontend/src/main.tsx` | config | — | `frontend/src/main.tsx` | exato |
| `frontend/src/index.css` | config | — | `frontend/src/index.css` | exato |
| `frontend/src/pages/Login.tsx` | component | request-response | `frontend/src/pages/Login.tsx` | exato |
| `frontend/src/pages/Register.tsx` | component | request-response | `frontend/src/pages/Register.tsx` | exato |
| `frontend/src/pages/ForgotPassword.tsx` | component | request-response | `frontend/src/pages/ForgotPassword.tsx` | exato |
| `frontend/src/pages/ResetPassword.tsx` | component | request-response | `frontend/src/pages/ResetPassword.tsx` | exato |
| `frontend/src/pages/GestaoAmbiente.tsx` | component | CRUD | `frontend/src/pages/GestaoAmbiente.tsx` | exato |
| `frontend/src/pages/Managers.tsx` | component | CRUD | `frontend/src/pages/Managers.tsx` | exato |
| `frontend/src/pages/AdminUsers.tsx` | component | CRUD | `frontend/src/pages/AdminUsers.tsx` | exato |
| `frontend/src/pages/ERPBridgeConfig.tsx` | component | CRUD | `frontend/src/pages/ERPBridgeConfig.tsx` | exato |
| `frontend/src/pages/ERPBridgeCredenciais.tsx` | component | CRUD | `frontend/src/pages/ERPBridgeCredenciais.tsx` | exato + botão novo |

---

## Atribuições de Padrão

---

### `backend/main.go` (config/roteador, request-response)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/main.go`
**Estratégia:** Transplante seletivo. Copiar a lógica de inicialização do DB (linhas 51–110), o runner de migração dentro de `onDBConnected()` (linhas 112–222), os helpers `withDB`/`withAuth` (linhas 422–443), o endpoint `/api/health`, e registrar apenas as rotas do escopo. **Remover:** `worker.StartWorker`, `worker.StartXMLWorker`, goroutines de agendamento ERP/RFB, `promhttp.Handler`, importação de `fb_apu04/worker` e `prometheus`.

**Padrão de imports** (linhas 1–25 — manter apenas o necessário):
```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "sync"
    "syscall"
    "time"

    "fb_testesfc/handlers"   // ← renomear de fb_apu04

    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
    // NÃO incluir: prometheus, fb_apu04/worker
)
```

**Padrão do runner de migração — `onDBConnected()`** (linhas 112–222 do FB_APU04):
```go
func onDBConnected() {
    database := getDB()
    migrationDir := "migrations"
    if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
        if _, err := os.Stat("backend/migrations"); err == nil {
            migrationDir = "backend/migrations"
        }
    }

    files, err := filepath.Glob(filepath.Join(migrationDir, "*.sql"))
    // ... cria/garante tabela schema_migrations (filename VARCHAR PRIMARY KEY, executed_at TIMESTAMPTZ) ...
    for _, file := range files {
        baseName := filepath.Base(file)
        var alreadyExecuted bool
        _ = database.QueryRow(
            "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)", baseName,
        ).Scan(&alreadyExecuted)
        if alreadyExecuted { continue }

        migration, _ := os.ReadFile(file)
        _, err = database.Exec(string(migration))
        if err != nil {
            log.Printf("ERROR: Migration %s failed: %v — will retry on next startup", file, err)
            continue // NÃO registra — retenta no próximo startup
        }
        database.Exec(
            "INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING", baseName,
        )
    }
    // ← PARAR AQUI. Não copiar worker.StartWorker nem os goroutines de agendamento (linhas 224-310)
}
```

**Padrão `withDB` / `withAuth`** (linhas 422–443 do FB_APU04):
```go
withDB := func(handlerFactory func(*sql.DB) http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        database := getDB()
        if database == nil { jsonServiceUnavailable(w); return }
        handlerFactory(database)(w, r)
    }
}

withAuth := func(handlerFactory func(*sql.DB) http.HandlerFunc, role string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        database := getDB()
        if database == nil { jsonServiceUnavailable(w); return }
        h := handlerFactory(database)
        handlers.AuthMiddleware(h, role)(w, r)
    }
}
```

**Padrão de registro de rotas** (linhas 502–606 do FB_APU04, seletivo):
```go
// Sem autenticação
http.HandleFunc("/api/auth/register",        withDB(handlers.RegisterHandler))
http.HandleFunc("/api/auth/login",           withDB(handlers.LoginHandler))
http.HandleFunc("/api/auth/forgot-password", withDB(handlers.ForgotPasswordHandler))
http.HandleFunc("/api/auth/reset-password",  withDB(handlers.ResetPasswordHandler))
http.HandleFunc("/api/auth/refresh",         withDB(handlers.RefreshHandler))
http.HandleFunc("/api/auth/logout",          withDB(handlers.LogoutHandler))

// Autenticados (role="")
http.HandleFunc("/api/auth/me",                withAuth(handlers.GetMeHandler, ""))
http.HandleFunc("/api/auth/change-password",   withAuth(handlers.ChangePasswordHandler, ""))
http.HandleFunc("/api/auth/preferred-company", withAuth(handlers.SetPreferredCompanyHandler, ""))
http.HandleFunc("/api/user/hierarchy",         withAuth(handlers.GetUserHierarchyHandler, ""))
http.HandleFunc("/api/user/companies",         withAuth(handlers.GetUserCompaniesHandler, ""))

// Admin (role="admin")
http.HandleFunc("/api/admin/users",          withAuth(handlers.ListUsersHandler, "admin"))
http.HandleFunc("/api/admin/users/create",   withAuth(handlers.CreateUserHandler, "admin"))
http.HandleFunc("/api/admin/users/promote",  withAuth(handlers.PromoteUserHandler, "admin"))
http.HandleFunc("/api/admin/users/delete",   withAuth(handlers.DeleteUserHandler, "admin"))
http.HandleFunc("/api/admin/users/reassign", withAuth(handlers.ReassignUserHandler, "admin"))

// Hierarquia — multi-method inline (padrão linhas 557–606)
http.HandleFunc("/api/config/environments", withAuth(func(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:    handlers.GetEnvironmentsHandler(db)(w, r)
        case http.MethodPost:   handlers.CreateEnvironmentHandler(db)(w, r)
        case http.MethodPut:    handlers.UpdateEnvironmentHandler(db)(w, r)
        case http.MethodDelete: handlers.DeleteEnvironmentHandler(db)(w, r)
        default: http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    }
}, ""))
// Repetir padrão para /api/config/groups e /api/config/companies

// ERP Bridge
http.HandleFunc("/api/erp-bridge/config",                   withAuth(handlers.ERPBridgeConfigHandler, ""))
http.HandleFunc("/api/erp-bridge/config/generate-api-key",  withAuth(handlers.ERPBridgeGenerateAPIKeyHandler, "admin"))
http.HandleFunc("/api/erp-bridge/test-connection",          withAuth(handlers.ERPBridgeTestConnectionHandler, "")) // NOVO
http.HandleFunc("/api/erp-bridge/credentials",              withDB(handlers.ERPBridgeCredentialsHandler))

// Managers
http.HandleFunc("/api/managers",        withAuth(handlers.ListManagersHandler, ""))
http.HandleFunc("/api/managers/create", withAuth(handlers.CreateManagerHandler, ""))

// Health
http.HandleFunc("/api/health", healthHandler)
```

**Padrão de aviso de startup** (linhas 329–337 do FB_APU04 — adaptar):
```go
func main() {
    _ = godotenv.Load()
    handlers.ValidateJWTSecret()

    // Aviso específico do FB_TESTESFC (não está no FB_APU04)
    if os.Getenv("ENCRYPTION_KEY") == "" {
        log.Println("AVISO: ENCRYPTION_KEY não configurada — credenciais Oracle usam JWT_SECRET como fallback. Configure ENCRYPTION_KEY no .env.")
    }

    initDBAsync()
    port := os.Getenv("PORT")
    if port == "" { port = "8085" }  // ← porta do FB_TESTESFC (não 8081/8084)
    // ...
    http.Handle("/", handlers.SecurityMiddleware(http.DefaultServeMux, handlers.GetAllowedOrigins()))
    // NÃO registrar /metrics (Prometheus removido)
}
```

**Modificações obrigatórias vs FB_APU04:**
- `module fb_apu04` → `module fb_testesfc` no go.mod (D-12)
- Porta padrão `8081` → `8085` (D-13)
- Remover `import "fb_apu04/worker"` e `promhttp.Handler()` (D-03)
- Remover goroutines de agendamento ERP/RFB (linhas 224–298) (D-02)
- Remover chamadas `worker.StartWorker` / `worker.StartXMLWorker` (linhas 225–228) (Pitfall 6)

---

### `backend/go.mod` (config)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/go.mod` (linhas 1–39)

**Padrão base** (go.mod do FB_APU04 — filtrar e adicionar go-ora):
```
module fb_testesfc    ← renomear (D-12)

go 1.24.1

require (
    github.com/golang-jwt/jwt/v5 v5.3.1
    github.com/joho/godotenv v1.5.1
    github.com/lib/pq v1.11.2
    github.com/sijms/go-ora/v2 v2.9.0    ← ADICIONAR (D-14)
    golang.org/x/crypto v0.48.0
    golang.org/x/text v0.34.0
)
```

**Remover** (D-03): `excelize`, `rardecode`, `prometheus/*`, `ledongthuc/pdf`, `tiendc/go-deepcopy`, `xuri/efp`, `xuri/nfp`, `klauspost/compress`, `richardlehane/*`, `beorn7/perks`, `cespare/xxhash`, `munnerz/goautoneg`, `google.golang.org/protobuf`, `golang.org/x/sys`, `golang.org/x/net`.

Após copiar os arquivos de handler, executar `go mod tidy` para resolver as indiretas automaticamente.

---

### `backend/handlers/auth.go` (controller, request-response)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/auth.go`
**Estratégia:** Copiar integralmente. Executar find/replace `fb_apu04/` → `fb_testesfc/` nos imports.

**Padrão de imports** (linhas 1–18):
```go
package handlers

import (
    "context"
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "sync"
    "time"

    "fb_testesfc/services"    // ← renomear de fb_apu04/services
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)
```

**Padrão JWT secret** (linhas 64–80):
```go
func getJWTSecret() []byte {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" { return []byte("super-secret-key-change-me-in-prod") }
    return []byte(secret)
}
func ValidateJWTSecret() {
    if os.Getenv("JWT_SECRET") == "" {
        if os.Getenv("DATABASE_URL") != "" {
            log.Fatal("FATAL: JWT_SECRET not set — set it to a 32+ byte random value before deploying.")
        }
        log.Println("WARNING: JWT_SECRET not set — using insecure default (OK for local dev only).")
    }
}
```

**Padrão refresh token store** (linhas 84–114):
```go
var (
    refreshTokenStore sync.Map // token string → refreshTokenData
    tokenBlacklist    sync.Map // token string → time.Time (expiry)
)
// + goroutine de limpeza periódica em init()
```

**Padrão AuthMiddleware** (linhas 209–258 — padrão central de proteção de rotas):
```go
func AuthMiddleware(next http.HandlerFunc, requiredRole string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        tokenString := ""
        authHeader := r.Header.Get("Authorization")
        if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
            tokenString = authHeader[7:]
        } else if qt := r.URL.Query().Get("token"); qt != "" {
            tokenString = qt
        } else {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }

        if _, revoked := tokenBlacklist.Load(tokenString); revoked {
            http.Error(w, "Token revoked", http.StatusUnauthorized)
            return
        }

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return getJWTSecret(), nil
        })
        if err != nil || !token.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok || !token.Valid {
            http.Error(w, "Invalid token claims", http.StatusUnauthorized)
            return
        }

        userRole, ok := claims["role"].(string)
        if !ok { http.Error(w, "Role not found in token", http.StatusUnauthorized); return }

        if requiredRole != "" && userRole != requiredRole && userRole != "admin" {
            http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
            return
        }

        ctx := context.WithValue(r.Context(), ClaimsKey, claims)
        next(w, r.WithContext(ctx))
    }
}
```

**Padrão `GetEffectiveCompanyID`** (linhas 274–356 — contexto de empresa por request, TEN-03):
```go
func GetEffectiveCompanyID(db *sql.DB, userID, requestedCompanyID string) (string, error) {
    // 1. Valida requestedCompanyID se fornecido (header X-Company-ID)
    // 2. Fallback: empresa do owner_id = userID
    // 3. Fallback: empresa via user_environments + enterprise_groups
    // Retorna primeiro companyID válido encontrado
}
```

**Funções a copiar integralmente:** `LoginHandler` (ln 544), `RegisterHandler` (ln 402), `ForgotPasswordHandler` (ln 741), `ResetPasswordHandler` (ln 812), `ChangePasswordHandler` (ln 924), `RefreshHandler` (ln 986), `LogoutHandler` (ln 1036), `GetMeHandler` (ln 184), `SetPreferredCompanyHandler` (ln 1072), `GetUserCompaniesHandler` (ln 358).

---

### `backend/handlers/admin.go` (controller, CRUD)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/admin.go`
**Estratégia:** Copiar apenas as funções necessárias (gestão de usuários). Excluir funções de reset de DB e diagnóstico que dependem de módulos SPED fora do escopo.

**Funções a copiar** (localizadas via grep):
- `ListUsersHandler` (linha 634) — `GET /api/admin/users`
- `CreateUserHandler` (linha 543) — `POST /api/admin/users/create`
- `PromoteUserHandler` (linha 687) — `POST /api/admin/users/promote`
- `DeleteUserHandler` (linha 837) — `DELETE /api/admin/users/delete`
- `ReassignUserHandler` (linha 767) — `POST /api/admin/users/reassign`

**Padrão de imports** (linhas 1–16):
```go
package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
)
```

**Funções a NÃO copiar:** `ResetDatabaseHandler`, `ResetCompanyDataHandler`, `DiagnosticDataHandler`, `AdminNFCancelamentoHandler`, `RefreshViewsHandler` — dependem de tabelas SPED/mv_mercadorias que não existem no FB_TESTESFC.

---

### `backend/handlers/crypto.go` (utility, transform)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/crypto.go` (100 linhas — copiar integral)

**Padrão completo** (copiar como está — sem imports de módulo interno):
```go
package handlers

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "errors"
    "io"
    "os"
)

// getEncryptionKey() — lê ENCRYPTION_KEY, fallback JWT_SECRET (dev), fallback hardcoded
// EncryptField(plaintext string) (string, error) — AES-256-GCM, retorna base64(nonce+ciphertext)
// DecryptField(encoded string) (string, error)
// DecryptFieldWithFallback(encoded string) string — retorna plaintext se decrypt falhar (migração)
```

**Sem modificações necessárias** — não tem imports de módulos internos fb_apu04.

---

### `backend/handlers/environment.go` (controller, CRUD)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/environment.go` (640 linhas)

**Padrão de imports** (linhas 1–12):
```go
package handlers

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "regexp"

    "github.com/golang-jwt/jwt/v5"
    "github.com/lib/pq"
)
```

**Padrão de handler CRUD** (linha 48 — padrão replicado em todos handlers com JWT):
```go
func GetEnvironmentsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
        if !ok { http.Error(w, "Unauthorized", http.StatusUnauthorized); return }
        userID, _ := claims["user_id"].(string)
        role, _   := claims["role"].(string)
        if userID == "" { http.Error(w, "Unauthorized", http.StatusUnauthorized); return }
        // ... lógica do handler
    }
}
```

**Sem modificações necessárias** — imports são apenas de pacotes externos (github.com/*, stdlib).

---

### `backend/handlers/hierarchy.go` (controller, request-response)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/hierarchy.go` (128 linhas — copiar integral)

**Padrão de imports** (linhas 1–7 — sem deps de módulo interno):
```go
package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
)
```

**Padrão `GetUserHierarchyHandler`** (linha 26): usa `GetUserIDFromContext(r)` e `GetEffectiveCompanyID` de auth.go (mesmo pacote), resolve ambiente→grupo→empresa→filiais para o usuário autenticado.

**Nota:** `hierarchy.go` faz JOIN em `import_jobs` para listar filiais (linhas 71–113). Essa tabela NÃO existe no schema da Fase 1. Ao copiar, manter o código — a query simplesmente retornará `branches = []` sem erro (LEFT JOIN + verificação `if err == nil`). Isso é correto para a Fase 1 (filiais aparecem na Fase 2 quando a importação de XMLs criar as `import_jobs`).

**Sem modificações de import necessárias.**

---

### `backend/handlers/managers.go` (controller, CRUD)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/managers.go` (363 linhas — copiar integral)

**Padrão de imports** (linhas 1–12):
```go
package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
)
```

**Padrão de handler com `GetEffectiveCompanyID`** (linhas 27–55 — padrão para todos os handlers que operam por empresa):
```go
func ListManagersHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Access-Control-Allow-Origin", "*")

        claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
        if !ok { http.Error(w, "Unauthorized", http.StatusUnauthorized); return }
        userID := claims["user_id"].(string)

        companyID, err := GetEffectiveCompanyID(db, userID, r.Header.Get("X-Company-ID"))
        if err != nil { http.Error(w, "Error getting company: "+err.Error(), http.StatusInternalServerError); return }

        rows, err := db.Query(`SELECT ... FROM managers WHERE company_id = $1 ORDER BY nome_completo ASC`, companyID)
        // ...
    }
}
```

**Sem modificações de import necessárias.**

---

### `backend/handlers/middleware.go` (middleware, request-response)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/middleware.go` (218 linhas — copiar integral)

**Padrão `SecurityMiddleware`** (linhas 94–120 — aplica CORS + security headers a todo o mux):
```go
func SecurityMiddleware(next http.Handler, allowedOrigins map[string]bool) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        var allowedOrigin string
        if origin != "" && allowedOrigins[origin] { allowedOrigin = origin }

        srw := &secureResponseWriter{ResponseWriter: w, origin: allowedOrigin}

        if r.Method == http.MethodOptions {
            srw.applyHeaders()
            h := w.Header()
            h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
            h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Company-ID")
            h.Set("Access-Control-Max-Age", "86400")
            w.WriteHeader(http.StatusNoContent)
            return
        }
        next.ServeHTTP(srw, r)
    })
}
```

**`GetAllowedOrigins()`** (linhas 13–34): lê `ALLOWED_ORIGINS` do env; fallback lista hardcoded. Para o FB_TESTESFC, adicionar `http://localhost:3004` (porta do Vite dev) à lista de fallback.

**Rate limiters exportados** (linhas 141–145): `LoginRL`, `RegisterRL`, `ForgotPasswordRL` — usados por auth.go. Copiar como estão.

**Sem modificações de import necessárias.**

---

### `backend/handlers/erp_bridge.go` (controller, CRUD)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/erp_bridge.go` (793 linhas — copiar + acrescentar)

**Padrão de imports** (linhas 1–15 — adicionar sijms/go-ora):
```go
package handlers

import (
    "context"          // ← necessário para o novo endpoint (timeout)
    "crypto/rand"
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "fmt"              // ← necessário para o novo endpoint (DSN format)
    "log"
    "net/http"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
    _ "github.com/sijms/go-ora/v2"  // ← ADICIONAR (registro do driver "oracle")
)
```

**Padrão helper `erpBridgeGetCompany`** (linhas 68–75 — extrair companyID de qualquer request de ERP):
```go
func erpBridgeGetCompany(db *sql.DB, r *http.Request) (string, error) {
    claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
    if !ok { return "", sql.ErrNoRows }
    userID := claims["user_id"].(string)
    return GetEffectiveCompanyID(db, userID, r.Header.Get("X-Company-ID"))
}
```

**Funções a copiar do FB_APU04:** `ERPBridgeConfigHandler` (ln 79), `ERPBridgeGenerateAPIKeyHandler` (ln 537), `ERPBridgeCredentialsHandler` (ln 576 — para autenticação do daemon futuro).

**Funções a NÃO copiar** (dependem de daemon/batch/workers fora da Fase 1): `ERPBridgeRunsHandler`, `ERPBridgeRunHandler`, `ERPBridgeServidoresHandler`, `ERPBridgeRegistrarServidoresHandler`, `ERPBridgeTriggerHandler`, `ERPBridgePendingHandler`, `ERPBridgeHeartbeatHandler`.

**Novo endpoint `ERPBridgeTestConnectionHandler`** (D-14 — não existe no FB_APU04):
```go
// POST /api/erp-bridge/test-connection
func ERPBridgeTestConnectionHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        companyID, err := erpBridgeGetCompany(db, r)
        if err != nil {
            json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "empresa não encontrada"})
            return
        }

        var oracleDsn, oracleUsuario, oracleSenha sql.NullString
        db.QueryRow(`SELECT oracle_dsn, oracle_usuario, oracle_senha
            FROM erp_bridge_config WHERE company_id = $1`, companyID,
        ).Scan(&oracleDsn, &oracleUsuario, &oracleSenha)

        if !oracleDsn.Valid || oracleDsn.String == "" {
            json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "DSN Oracle não configurado"})
            return
        }

        dsn := fmt.Sprintf("oracle://%s:%s@%s",
            DecryptFieldWithFallback(oracleUsuario.String),
            DecryptFieldWithFallback(oracleSenha.String),
            DecryptFieldWithFallback(oracleDsn.String),
        )
        conn, err := sql.Open("oracle", dsn)
        if err != nil {
            json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
            return
        }
        defer conn.Close()

        ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
        defer cancel()
        if err := conn.PingContext(ctx); err != nil {
            json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
            return
        }
        json.NewEncoder(w).Encode(map[string]any{"ok": true})
    }
}
```

---

### `backend/services/email.go` (service, event-driven)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/services/email.go`
**Estratégia:** Copiar integral. Sem imports de módulo interno.

**Padrão de imports** (linhas 1–13 — apenas stdlib):
```go
package services

import (
    "crypto/tls"
    "fmt"
    "log"
    "math"
    "net/smtp"
    "os"
    "strconv"
    "strings"
    "time"
)
```

**Sem modificações necessárias.**

---

### `backend/services/crypto.go` (service, transform)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/services/crypto.go` (50 linhas — copiar integral)

**Função central** `DecryptFieldWithFallback(encoded string) string`: mesma lógica AES-256-GCM de `handlers/crypto.go`, mas no pacote `services` para uso em serviços que precisam descriptografar credenciais (ex.: email com senha do Oracle).

**Sem modificações necessárias.**

---

### `backend/migrations/001_auth_hierarchy.sql` (migration, batch)

**Análogos de referência:**
- `013_create_environment_hierarchy.sql` — tabelas `environments`, `enterprise_groups`, `companies`
- `015_create_auth_system.sql` — tabelas `users`, `user_environments`, `verification_tokens`
- `017_add_owner_to_companies.sql` — coluna `owner_id UUID REFERENCES users(id)`
- `018_add_role_to_users.sql` — coluna `role VARCHAR(50) DEFAULT 'admin'`
- `023_remove_cnpj_from_companies.sql` — remove `cnpj` (NÃO incluir desde o início)
- `025_add_indexes_auth.sql` — índices de performance

**Schema final consolidado** (escrever do zero, sem cnpj):
```sql
-- 001_auth_hierarchy.sql
-- Consolida 013 + 015 + 017 + 018 + 025 (sem cnpj — ver 023)

CREATE TABLE IF NOT EXISTS environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS enterprise_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES enterprise_groups(id) ON DELETE CASCADE,
    owner_id UUID,               -- adicionado direto (017)
    name VARCHAR(255) NOT NULL,
    trade_name VARCHAR(255),
    regime_tributario VARCHAR(50),
    inscricao_estadual VARCHAR(50),
    cnae_principal VARCHAR(10),
    segmento_economico VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
    -- SEM coluna cnpj (023 a removeu; começar sem ela é mais limpo)
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'admin',         -- adicionado direto (018)
    is_verified BOOLEAN DEFAULT FALSE,
    trial_ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_environments (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'admin',
    preferred_company_id UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, environment_id)
);

CREATE TABLE IF NOT EXISTS verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Índices de performance (025)
CREATE INDEX IF NOT EXISTS idx_enterprise_groups_env ON enterprise_groups(environment_id);
CREATE INDEX IF NOT EXISTS idx_companies_group ON companies(group_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_verification_tokens_token ON verification_tokens(token);
CREATE INDEX IF NOT EXISTS idx_user_environments_user ON user_environments(user_id);
```

---

### `backend/migrations/002_erp_bridge.sql` (migration, batch)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/migrations/065_erp_bridge.sql` (cópia direta)

Tabelas: `erp_bridge_config` (PK company_id), `erp_bridge_runs`, `erp_bridge_run_items`, `erp_bridge_servidores`, `parceiros`.

**Cópia direta** — nenhuma modificação. Referência nas linhas 1–80 já lidas.

---

### `backend/migrations/003_managers.sql` (migration, batch)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/migrations/046_create_managers_table.sql` (cópia direta — 45 linhas)

Tabela: `managers` + índices + trigger `update_managers_updated_at`.

**Cópia direta** — nenhuma modificação.

---

### `backend/migrations/004_seed_ferreira_costa.sql` (migration, batch)

**Análogos de referência:**
- `016_seed_default_environment.sql` — padrão DO $$ com SELECT/INSERT idempotente
- `021_ensure_admin_user.sql` — hash bcrypt pré-computado `$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC` (senha `123456`, cost 14)
- `024_ensure_master_link.sql` — padrão de vínculo `user_environments`

**Seed idempotente** (D-09, D-10 — escrever do zero com nomes Ferreira Costa):
```sql
DO $$
DECLARE
    v_env_id     UUID;
    v_group_id   UUID;
    v_company_id UUID;
    v_user_id    UUID;
BEGIN
    -- 1. Ambiente Ferreira Costa
    SELECT id INTO v_env_id FROM environments WHERE name = 'Ferreira Costa';
    IF v_env_id IS NULL THEN
        INSERT INTO environments (name, description)
        VALUES ('Ferreira Costa', 'Ambiente principal da Ferreira Costa')
        RETURNING id INTO v_env_id;
    END IF;

    -- 2. Grupo Ferreira Costa
    SELECT id INTO v_group_id FROM enterprise_groups
    WHERE name = 'Ferreira Costa' AND environment_id = v_env_id;
    IF v_group_id IS NULL THEN
        INSERT INTO enterprise_groups (environment_id, name)
        VALUES (v_env_id, 'Ferreira Costa')
        RETURNING id INTO v_group_id;
    END IF;

    -- 3. Empresa Ferreira Costa (sem cnpj — coluna não existe neste schema)
    SELECT id INTO v_company_id FROM companies
    WHERE name = 'Ferreira Costa' AND group_id = v_group_id;
    IF v_company_id IS NULL THEN
        INSERT INTO companies (group_id, name, trade_name)
        VALUES (v_group_id, 'Ferreira Costa', 'Ferreira Costa')
        RETURNING id INTO v_company_id;
    END IF;

    -- 4. Admin claudio_bezerra@hotmail.com / 123456 (D-10)
    -- Hash bcrypt cost=14 retirado de 021_ensure_admin_user.sql do FB_APU04
    IF EXISTS (SELECT 1 FROM users WHERE email = 'claudio_bezerra@hotmail.com') THEN
        UPDATE users SET
            password_hash = '$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC',
            role = 'admin', is_verified = true,
            full_name = 'Claudio Bezerra (Admin)'
        WHERE email = 'claudio_bezerra@hotmail.com'
        RETURNING id INTO v_user_id;
    ELSE
        INSERT INTO users (email, password_hash, full_name, role, is_verified)
        VALUES ('claudio_bezerra@hotmail.com',
                '$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC',
                'Claudio Bezerra (Admin)', 'admin', true)
        RETURNING id INTO v_user_id;
    END IF;

    -- 5. Vínculo user ↔ ambiente
    INSERT INTO user_environments (user_id, environment_id, role)
    VALUES (v_user_id, v_env_id, 'admin')
    ON CONFLICT DO NOTHING;

    -- 6. Owner da empresa → admin
    UPDATE companies SET owner_id = v_user_id WHERE id = v_company_id AND owner_id IS NULL;

    -- 7. Linha de config ERP Bridge para a empresa (credenciais vazias, preenchidas pela UI)
    INSERT INTO erp_bridge_config (company_id)
    VALUES (v_company_id)
    ON CONFLICT DO NOTHING;
END $$;
```

---

### `docker-compose.yml` (config)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/docker-compose.yml`
**Estratégia:** Copiar serviços `api`, `web`, `db`. **Remover:** `redis`, `prometheus`, `grafana`, `alertmanager`, `postgres-exporter` (D-13).

**Padrão de serviços adaptado** (modificações vs FB_APU04):
```yaml
services:
  api:
    build:
      context: .
      dockerfile: backend/Dockerfile
    container_name: fb_testesfc-api          # ← renomear
    restart: always
    ports:
      - "8085:8085"                           # ← porta distinta (D-13)
    expose:
      - "8085"
    environment:
      - PORT=8085
      - DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@db:5432/${DB_NAME}?sslmode=disable
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_PORT=${SMTP_PORT}
      - SMTP_USER=${SMTP_USER}
      - SMTP_PASSWORD=${SMTP_PASSWORD}
      - SMTP_FROM=${SMTP_FROM}
      - APP_URL=${APP_URL:-http://localhost:3004}
      - JWT_SECRET=${JWT_SECRET}
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}     # ← adicionado (Pitfall 4)
      - COOKIE_SECURE=${COOKIE_SECURE:-false}
      - ALLOWED_ORIGINS=${ALLOWED_ORIGINS:-}
    depends_on:
      db:
        condition: service_healthy
    networks:
      - fb_testesfc_net                      # ← rede própria (D-13)

  web:
    build:
      context: ./frontend
    container_name: fb_testesfc-web
    restart: always
    ports:
      - "3004:80"                            # ← porta distinta (D-13)
    depends_on:
      - api
    networks:
      - fb_testesfc_net

  db:
    image: postgres:15-alpine
    container_name: fb_testesfc-db          # ← renomear
    restart: always
    environment:
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=${DB_NAME}
    ports:
      - "5435:5432"                          # ← porta host distinta, opcional (D-13, A3)
    volumes:
      - postgres_data_testesfc:/var/lib/postgresql/data  # ← volume próprio (D-13)
    networks:
      - fb_testesfc_net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s

networks:
  fb_testesfc_net:
    driver: bridge

volumes:
  postgres_data_testesfc:
```

---

### `backend/Dockerfile` (config)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/backend/Dockerfile`

**Padrão base** (copiar e remover vendor):
```dockerfile
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
# NÃO copiar vendor/ (D-03, Pitfall 3)
RUN go mod download                    # ← substituir COPY vendor
COPY backend/ .
# NÃO usar -mod=vendor (Pitfall 3)
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-w -s" -o server .

FROM alpine:latest
RUN apk add --no-cache ca-certificates  # necessário para TLS (Oracle, SMTP)
# Remover postgresql-client e poppler-utils — não usados na Fase 1
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8085
CMD ["./server"]
```

---

### `.env.example` (config)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/.env.example`

**Padrão adaptado** (adicionar ENCRYPTION_KEY — Pitfall 4):
```
# ── Banco de Dados ────────────────────────────────────────────────────────────
DB_USER=postgres
DB_PASSWORD=TROQUE_AQUI
DB_NAME=fb_testesfc_db              # ← banco próprio (D-13)

# ── Backend ───────────────────────────────────────────────────────────────────
PORT=8085                            # ← porta própria (D-13)
JWT_SECRET=GERE_COM_openssl_rand_hex_32
ENCRYPTION_KEY=GERE_COM_openssl_rand_hex_32  # ← ADICIONADO (Pitfall 4)
COOKIE_SECURE=false

# ── App URL ───────────────────────────────────────────────────────────────────
APP_URL=http://localhost:3004        # ← porta do frontend (D-13)

# ── SMTP ─────────────────────────────────────────────────────────────────────
SMTP_HOST=smtp.hostinger.com
SMTP_PORT=465
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=

# ── CORS (origens permitidas, separadas por vírgula) ─────────────────────────
ALLOWED_ORIGINS=http://localhost:3004,http://localhost:3003
```

---

### `frontend/src/App.tsx` (component, request-response) — NOVO

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/frontend/src/App.tsx`
**Estratégia:** Escrever do zero baseado no App.tsx do FB_APU04, removendo FilialProvider, CompanySwitcher, AjudaChat, ModuleTabs e todas as 40+ páginas fora do escopo.

**Padrão de imports** (baseado em linhas 1–57 do FB_APU04 — versão enxuta):
```tsx
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from '@/components/ui/sonner'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import ResetPassword from './pages/ResetPassword'
import GestaoAmbiente from './pages/GestaoAmbiente'
import Managers from './pages/Managers'
import AdminUsers from './pages/AdminUsers'
import ERPBridgeConfig from './pages/ERPBridgeConfig'
import ERPBridgeCredenciais from './pages/ERPBridgeCredenciais'
import { AppRail } from '@/components/AppRail'
import { AuthProvider, useAuth } from './contexts/AuthContext'
// NÃO importar: FilialProvider, CompanySwitcher, AjudaChat, getActiveModule, modules
```

**Padrão ProtectedRoute / AdminRoute** (linhas 69–84 do FB_APU04 — copiar como está):
```tsx
function ProtectedRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, loading } = useAuth()
    const location = useLocation()
    if (loading) return null
    if (!isAuthenticated) return <Navigate to="/login" state={{ from: location }} replace />
    return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, loading, user } = useAuth()
    const location = useLocation()
    if (loading) return null
    if (!isAuthenticated) return <Navigate to="/login" state={{ from: location }} replace />
    if (user?.role !== 'admin') return <Navigate to="/" replace />
    return <>{children}</>
}
```

**Padrão AppLayout enxuto** (baseado em linhas 169–258 do FB_APU04 — sem ModuleTabs, sem AjudaChat):
```tsx
function AppLayout() {
    const { company } = useAuth()
    return (
        <div className="flex h-screen overflow-hidden bg-background">
            <AppRail />
            <div className="flex flex-col flex-1 min-w-0">
                <header className="flex items-center justify-between h-12 border-b bg-white px-4 shrink-0">
                    <span className="text-sm font-semibold">FB_TESTESFC — Validador Fiscal</span>
                    {company && (
                        <span className="flex items-center gap-1.5 text-xs font-medium text-sky-700 bg-sky-50 border border-sky-200 px-2.5 py-1 rounded-full">
                            {company}
                        </span>
                    )}
                    {/* SEM CompanySwitcher (D-11) */}
                </header>
                {/* SEM ModuleTabs */}
                <main className="flex-1 overflow-auto">
                    <div className="p-4">
                        <Routes>
                            <Route path="/"                        element={<Navigate to="/config/erp-bridge" replace />} />
                            <Route path="/config/ambiente"         element={<ProtectedRoute><GestaoAmbiente /></ProtectedRoute>} />
                            <Route path="/config/gestores"         element={<ProtectedRoute><Managers /></ProtectedRoute>} />
                            <Route path="/config/usuarios"         element={<AdminRoute><AdminUsers /></AdminRoute>} />
                            <Route path="/importacoes/erp-bridge"  element={<AdminRoute><ERPBridgeConfig /></AdminRoute>} />
                            <Route path="/config/erp-bridge"       element={<AdminRoute><ERPBridgeCredenciais /></AdminRoute>} />
                        </Routes>
                    </div>
                </main>
            </div>
            <Toaster />
            {/* SEM AjudaChat */}
        </div>
    )
}

function App() {
    return (
        <QueryClientProvider client={new QueryClient()}>
            <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                <AuthProvider>
                    <Routes>
                        <Route path="/login"           element={<Login />} />
                        <Route path="/register"        element={<Register />} />
                        <Route path="/forgot-password" element={<ForgotPassword />} />
                        <Route path="/reset-senha"     element={<ResetPassword />} />
                        {/* SEM FilialProvider (D-11) */}
                        <Route path="/*" element={<ProtectedRoute><AppLayout /></ProtectedRoute>} />
                    </Routes>
                </AuthProvider>
            </BrowserRouter>
        </QueryClientProvider>
    )
}
export default App
```

---

### `frontend/src/contexts/AuthContext.tsx` (provider, request-response)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/frontend/src/contexts/AuthContext.tsx` (254 linhas — copiar integral)

**Padrão crítico — interceptor de fetch** (linhas 50–63 — injeta Bearer + X-Company-ID em toda chamada):
```tsx
useEffect(() => {
    const originalFetch = window.fetch.bind(window);
    window.fetch = (input: RequestInfo | URL, init: RequestInit = {}) => {
        const headers = new Headers(init.headers || {});
        if (!headers.has('Authorization') && tokenRef.current) {
            headers.set('Authorization', `Bearer ${tokenRef.current}`);
        }
        if (companyIdRef.current) {
            headers.set('X-Company-ID', companyIdRef.current);
        }
        return originalFetch(input, { ...init, headers });
    };
    return () => { window.fetch = originalFetch; };
}, []);
```

**Padrão de restore de sessão** (linhas 65–131 — `POST /api/auth/refresh` com cookie httpOnly na inicialização):
```tsx
useEffect(() => {
    const storedUser = localStorage.getItem('user');
    if (storedUser) {
        // Restaura metadata (não-sensível) imediatamente para UI rápida
        // Depois troca o cookie httpOnly por novo access token
        fetch('/api/auth/refresh', { method: 'POST', credentials: 'include' })
            .then(res => { if (!res.ok && res.status === 401) { /* clear + redirect /login */ } })
            .then(data => { setToken(data.token); tokenRef.current = data.token; })
            .finally(() => setLoading(false));
    } else { setLoading(false); }
}, []);
```

**Sem modificações necessárias** — `switchCompany` pode ser mantido (usado por AuthContext internamente via `preferred-company`); apenas não haverá UI que o chame (sem CompanySwitcher).

---

### `frontend/src/lib/navigation.ts` (utility, transform)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/frontend/src/lib/navigation.ts` (107 linhas)
**Estratégia:** Copiar estrutura/tipos; substituir **conteúdo** dos `modules` e `getActiveModule`.

**Padrão de tipos** (linhas 1–12 — copiar as interfaces):
```ts
export interface ModuleTab {
    label: string
    path: string
    disabled?: boolean
    danger?: boolean
    adminOnly?: boolean
}
export interface ModuleConfig {
    label: string
    tabs: ModuleTab[]
}
```

**Conteúdo de `modules` para Fase 1** (substituir o objeto completo):
```ts
export const modules: Record<string, ModuleConfig> = {
    config: {
        label: 'Configurações',
        tabs: [
            { label: 'Credenciais ERP',  path: '/config/erp-bridge',          adminOnly: true },
            { label: 'Config ERP',       path: '/importacoes/erp-bridge',      adminOnly: true },
            { label: 'Ambiente',         path: '/config/ambiente' },
            { label: 'Gestores',         path: '/config/gestores' },
            { label: 'Usuários',         path: '/config/usuarios',             adminOnly: true },
        ],
    },
}
```

**`getActiveModule` simplificada** (substituir a função de 18 linhas):
```ts
export function getActiveModule(pathname: string): string {
    if (pathname.startsWith('/config/')) return 'config'
    if (pathname.startsWith('/importacoes/')) return 'config'
    return 'config'
}
```

---

### `frontend/src/components/AppRail.tsx` (component, request-response)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/frontend/src/components/AppRail.tsx`
**Estratégia:** Copiar e substituir apenas o array `mainItems` e o destino do botão Configurações.

**Padrão de imports** (linhas 1–31 do FB_APU04 — remover ícones não usados):
```tsx
import { Settings, LogOut, KeyRound } from 'lucide-react'
// Remover: TrendingUp, FolderInput, BarChart3, Scale, MapPin, ShieldCheck
// Manter: Tooltip*, DropdownMenu*, Dialog*, Button, Input, Label, useAuth, getActiveModule, sonner
```

**`mainItems` substituído** (linhas 33–40 do FB_APU04 — versão Fase 1):
```tsx
// Fase 1: nenhum módulo de negócio — AppRail apenas com Configurações
const mainItems: { id: string; icon: typeof Settings; label: string; path: string }[] = []
// mainItems vazio: o rail exibe apenas o botão Configurações + avatar do usuário
```

**Botão Configurações** (linhas 141–158 do FB_APU04 — ajustar path de destino):
```tsx
// Alterar onClick de '/config/aliquotas' para '/config/erp-bridge' (rota default da Fase 1)
<button onClick={() => navigate('/config/erp-bridge')} ...>
    <Settings className="h-5 w-5" />
</button>
```

**Logo** (linhas 107–114 — substituir favicon): usar `/favicon.ico` ou logo genérico do projeto em vez de `/favicon-fc.png`.

**Manter integralmente:** lógica de change-password dialog, logout dropdown, avatar com initials.

---

### `frontend/src/pages/ERPBridgeCredenciais.tsx` (component, CRUD)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/frontend/src/pages/ERPBridgeCredenciais.tsx` (262 linhas)
**Estratégia:** Copiar integral + adicionar botão "Testar Conexão Oracle" que chama `POST /api/erp-bridge/test-connection`.

**Padrão de imports** (linhas 1–10 do FB_APU04 — manter todos, adicionar estado para teste):
```tsx
import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/contexts/AuthContext';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Loader2, KeyRound, Eye, EyeOff, Copy, Wifi, WifiOff } from 'lucide-react';  // ← Wifi/WifiOff adicionados
import { toast } from 'sonner';
```

**Padrão do botão "Testar Conexão"** (acrescentar após o formulário existente):
```tsx
// Estado adicional:
const [testResult, setTestResult] = useState<{ ok: boolean; error?: string } | null>(null)
const [testing, setTesting] = useState(false)

async function handleTestConnection() {
    setTesting(true)
    setTestResult(null)
    try {
        const res = await fetch('/api/erp-bridge/test-connection', {
            method: 'POST',
            headers: authHeaders,
        })
        const data = await res.json()
        setTestResult(data)
        if (data.ok) toast.success('Conexão Oracle estabelecida com sucesso')
        else toast.error(`Falha na conexão: ${data.error}`)
    } catch {
        toast.error('Erro ao testar conexão')
        setTestResult({ ok: false, error: 'Erro de rede' })
    } finally {
        setTesting(false)
    }
}

// Botão no JSX (após o botão "Salvar Credenciais"):
<Button
    variant="outline"
    onClick={handleTestConnection}
    disabled={testing || !cfg?.oracle_dsn}
>
    {testing ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : null}
    Testar Conexão Oracle
</Button>
{testResult && (
    <span className={testResult.ok ? 'text-green-600 text-sm' : 'text-red-600 text-sm'}>
        {testResult.ok ? 'Conexão OK' : testResult.error}
    </span>
)}
```

---

### `frontend/vite.config.ts` (config)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/frontend/vite.config.ts` (34 linhas — copiar e alterar porta/proxy)

**Modificações** (linhas 9 e 16 do FB_APU04):
```ts
const target = env.VITE_API_TARGET || "http://localhost:8085"; // ← 8081 → 8085 (D-13)
// ...
server: {
    host: "0.0.0.0",
    port: 3004,  // ← 3003 → 3004 (D-13)
    proxy: { "/api": { target, changeOrigin: true, secure: false } },
},
```

---

### `frontend/package.json` (config)

**Análogo:** `/home/claudiobezerra/projetos/FB_APU04/frontend/package.json`
**Única modificação:** campo `"name"` → `"fb_testesfc-frontend"`.

---

### Páginas de cópia direta (sem modificação)

Os 8 arquivos abaixo são copiados integralmente. Não requerem modificação de import (não há imports de módulos internos Go):

| Arquivo | Análogo (caminho completo no FB_APU04) |
|---|---|
| `frontend/src/pages/Login.tsx` | `frontend/src/pages/Login.tsx` |
| `frontend/src/pages/Register.tsx` | `frontend/src/pages/Register.tsx` |
| `frontend/src/pages/ForgotPassword.tsx` | `frontend/src/pages/ForgotPassword.tsx` |
| `frontend/src/pages/ResetPassword.tsx` | `frontend/src/pages/ResetPassword.tsx` |
| `frontend/src/pages/GestaoAmbiente.tsx` | `frontend/src/pages/GestaoAmbiente.tsx` |
| `frontend/src/pages/Managers.tsx` | `frontend/src/pages/Managers.tsx` |
| `frontend/src/pages/AdminUsers.tsx` | `frontend/src/pages/AdminUsers.tsx` |
| `frontend/src/pages/ERPBridgeConfig.tsx` | `frontend/src/pages/ERPBridgeConfig.tsx` |
| `frontend/src/main.tsx` | `frontend/src/main.tsx` |
| `frontend/src/index.css` | `frontend/src/index.css` |
| `frontend/src/lib/utils.ts` | `frontend/src/lib/utils.ts` |
| `frontend/src/lib/logger.ts` | `frontend/src/lib/logger.ts` |
| `frontend/src/components/ui/` (44 arquivos) | `frontend/src/components/ui/` |

---

## Padrões Compartilhados

### Autenticação por JWT
**Fonte:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/auth.go` linhas 209–258
**Aplicar a:** Todos os handlers de rota protegida via `withAuth()` em `main.go`
```go
handlers.AuthMiddleware(handlerFunc, requiredRole)(w, r)
// requiredRole = "" → usuário autenticado qualquer role
// requiredRole = "admin" → somente role="admin" (ou se userRole == "admin")
```

### Contexto de empresa por request (TEN-03)
**Fonte:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/auth.go` linhas 274–356
**Aplicar a:** Todos os handlers que operam por empresa (`managers`, `erp_bridge`, `environment`, `hierarchy`)
```go
companyID, err := GetEffectiveCompanyID(db, userID, r.Header.Get("X-Company-ID"))
// Prioridade: X-Company-ID header → owner_id → user_environments
```

### Claims JWT no contexto do handler
**Fonte:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/environment.go` linhas 48–60
**Aplicar a:** Todo handler que precisa de userID ou role
```go
claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
if !ok { http.Error(w, "Unauthorized", http.StatusUnauthorized); return }
userID, _ := claims["user_id"].(string)
role, _   := claims["role"].(string)
```

### Criptografia de credenciais Oracle
**Fonte:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/crypto.go` (integral)
**Aplicar a:** `erp_bridge.go` (salvar/ler credenciais Oracle criptografadas)
```go
encrypted, _ := EncryptField(plaintext)  // antes de persistir no Postgres
plaintext := DecryptFieldWithFallback(encrypted)  // ao ler do Postgres
```

### CORS + Security Headers
**Fonte:** `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/middleware.go` linhas 94–120
**Aplicar a:** `main.go` — envolver o `http.DefaultServeMux` inteiro
```go
http.Handle("/", handlers.SecurityMiddleware(http.DefaultServeMux, handlers.GetAllowedOrigins()))
```

### Bearer + X-Company-ID no frontend
**Fonte:** `/home/claudiobezerra/projetos/FB_APU04/frontend/src/contexts/AuthContext.tsx` linhas 50–63
**Aplicar a:** Injeção automática via interceptor de `window.fetch` em `AuthContext.tsx`. Todas as páginas que usam `useAuth()` herdam esse comportamento sem código adicional.

### Padrão de query React Query nas páginas
**Fonte:** `/home/claudiobezerra/projetos/FB_APU04/frontend/src/pages/ERPBridgeCredenciais.tsx` linhas 25–35
**Aplicar a:** Qualquer página que faz fetch de dados na montagem
```tsx
const { data, isLoading } = useQuery({
    queryKey: ['erp-bridge-config', companyId],
    queryFn: async () => {
        const res = await fetch('/api/erp-bridge/config', { headers: authHeaders });
        if (!res.ok) throw new Error(res.statusText);
        return res.json();
    },
    enabled: !!token && !!companyId,
});
```

---

## Sem Análogo — Arquivos Sem Par no FB_APU04

| Arquivo | Papel | Fluxo de Dados | Motivo |
|---|---|---|---|
| `backend/handlers/erp_bridge.go` (função `ERPBridgeTestConnectionHandler`) | controller | request-response | Endpoint `POST /api/erp-bridge/test-connection` não existe no FB_APU04; usar padrão descrito na seção "RESEARCH.md § D-14 RESOLVIDO" acima |

---

## Armadilhas Críticas (extraídas do RESEARCH.md)

| # | Arquivo(s) afetado(s) | Problema | Solução |
|---|---|---|---|
| P1 | Todos os handlers `.go` copiados | Import path `fb_apu04/` não encontrado | Find/replace `"fb_apu04/` → `"fb_testesfc/` em todos os arquivos Go após cópia |
| P2 | `backend/main.go` | `worker.StartWorker` referencia pacote não copiado | Remover linhas 225–228 e 231–298 ao transplantar `onDBConnected()` |
| P3 | `backend/Dockerfile` | `-mod=vendor` falha sem diretório `vendor/` | Remover flag `-mod=vendor`; adicionar `RUN go mod download` antes do build |
| P4 | `.env.example` + `main.go` | `ENCRYPTION_KEY` ausente causa fallback silencioso | Documentar no `.env.example`; adicionar aviso de startup em `main.go` |
| P5 | `frontend/src/App.tsx` | `CompanySwitcher` faz GET `/api/user/companies` desnecessário | Não importar nem renderizar `CompanySwitcher` no novo `App.tsx` |
| P6 | `backend/migrations/` | Conflito de prefixos duplicados `021_` do FB_APU04 | Usar prefixos únicos `001–004`; nunca copiar as 149 migrações individuais |
| P7 | `backend/handlers/hierarchy.go` | JOIN em `import_jobs` (Fase 2) pode causar erro | A tabela não existe na Fase 1; `GetUserHierarchyHandler` retorna `branches=[]` sem erro graças ao `if err == nil` na linha 94 — comportamento correto |

---

## Metadados

**Escopo da busca:** `/home/claudiobezerra/projetos/FB_APU04/backend/` e `frontend/src/`
**Arquivos lidos:** ~30 arquivos fontes no FB_APU04
**Data de extração de padrões:** 2026-06-30
**Repositório fonte:** `github.com/ClaudioSBezerra/FB_APU04` (read-only)
