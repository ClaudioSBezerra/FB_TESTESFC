---
phase: 01-foundation-inherited-stack
reviewed: 2026-06-30T12:00:00Z
depth: standard
files_reviewed: 31
files_reviewed_list:
  - backend/main.go
  - backend/handlers/auth.go
  - backend/handlers/crypto.go
  - backend/handlers/middleware.go
  - backend/handlers/admin.go
  - backend/handlers/environment.go
  - backend/handlers/hierarchy.go
  - backend/handlers/managers.go
  - backend/handlers/erp_bridge.go
  - backend/services/crypto.go
  - backend/services/email.go
  - backend/migrations/001_auth_hierarchy.sql
  - backend/migrations/002_erp_bridge.sql
  - backend/migrations/003_managers.sql
  - backend/migrations/004_seed_ferreira_costa.sql
  - backend/migrations/005_seed_erp_bridge_ferreira_costa.sql
  - backend/Dockerfile
  - docker-compose.yml
  - frontend/Dockerfile
  - frontend/nginx.conf
  - .env.example
  - frontend/src/App.tsx
  - frontend/src/contexts/AuthContext.tsx
  - frontend/src/lib/navigation.ts
  - frontend/src/pages/ERPBridgeCredenciais.tsx
  - frontend/src/pages/AdminUsers.tsx
  - frontend/src/pages/Login.tsx
  - frontend/src/pages/ForgotPassword.tsx
  - frontend/src/pages/ResetPassword.tsx
  - frontend/src/pages/GestaoAmbiente.tsx
  - frontend/src/pages/Managers.tsx
  - frontend/src/pages/ERPBridgeConfig.tsx
  - frontend/vite.config.ts
findings:
  critical: 7
  warning: 8
  info: 5
  total: 20
status: resolved
resolution:
  resolved_at: 2026-06-30
  blockers_fixed: 7
  warnings_fixed: 8
  info_fixed: 2
  info_deferred: 3
  note: >-
    Todos os 7 blockers e 8 warnings corrigidos via gsd-code-fixer (commits fix(01): CR-01..CR-07, WR-01..WR-08).
    CR-01 (sanitização do erro do test-connection) e CR-02 (log de aviso ENCRYPTION_KEY) validados em runtime.
    Info IN-01/IN-04/IN-05 deferidos como dívida de baixa prioridade (ver corpo).
---

# Fase 01: Relatório de Code Review

**Revisado:** 2026-06-30
**Profundidade:** standard
**Arquivos Revisados:** 31
**Status:** issues_found

## Resumo

Revisão da Fase 1 — fundação herdada do FB_APU04. O núcleo de autenticação (JWT, bcrypt, refresh token rotativo, blacklist) está estruturalmente sólido. As queries ao banco usam parâmetros posicionais ($1, $2…) consistentemente — sem SQL injection nos handlers revisados. O design do crypto (AES-256-GCM com nonce aleatório) é correto.

Foram encontrados **7 blockers** e **8 warnings** que precisam de atenção antes de usar este código em produção ou antes de construir fases subsequentes sobre ele:

- O blocker mais grave é uma **vulnerabilidade de credential leak** no `test-connection`: o erro retornado ao frontend pode conter credenciais Oracle em texto claro embutidas na string de erro do driver go-ora.
- Dois blockers de **lógica de controle de acesso**: admin pode deletar a si mesmo sem restrição e o endpoint `/api/admin/diagnostic` referenciado pelo frontend não existe no backend.
- A **seed (migração 004)** força `UPDATE ... SET password_hash = '...'` toda vez que o admin já existe, o que reinicia a senha para `123456` em toda execução — problemático se a senha foi trocada.
- O **`getEncryptionKey()`** em `handlers/crypto.go` ainda usa o fallback para JWT_SECRET silenciosamente em produção (quando `DATABASE_URL` está definida mas `ENCRYPTION_KEY` não), sem fazer `log.Fatal` como o comentário promete.
- Há **vazamento de informação** no health endpoint (erro de DB retornado ao cliente) e no `test-connection` (erro do driver Oracle retornado ao cliente).

---

## Structural Findings (fallow)

Nenhuma análise estrutural pré-computada fornecida para esta fase.

---

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Credential Leak via Oracle Driver Error Message

**File:** `backend/handlers/erp_bridge.go:344`
**Issue:** Quando `conn.PingContext` falha, o erro do driver `sijms/go-ora` é retornado diretamente ao frontend via JSON: `json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": pingErr.Error()})`. O driver go-ora frequentemente inclui o DSN completo (incluindo usuário e senha em texto claro) na mensagem de erro — por exemplo: `"ORA-01017: invalid username/password; logon denied: oracle://usuario:senha@host:1521/service"`. Isso expõe credenciais Oracle no navegador do usuário administrador.

O mesmo vazamento ocorre na linha 344 para `sql.Open` e também na linha 90 do mesmo arquivo para erros internos do banco: `http.Error(w, err.Error(), http.StatusInternalServerError)`.

**Fix:**
```go
// Em vez de retornar o erro bruto do driver:
if pingErr := conn.PingContext(ctx); pingErr != nil {
    log.Printf("ERPBridge test-connection ping error (company %s): %v", companyID, pingErr)
    // Retornar mensagem genérica sem detalhes do driver
    msg := "Falha ao conectar ao Oracle. Verifique DSN, usuário e senha."
    // Para ambientes de dev, pode-se logar o erro completo mas nunca enviá-lo ao cliente
    json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
    return
}
```
O mesmo padrão deve ser aplicado ao erro de `sql.Open` na linha 344 e aos erros internos de banco expostos em `http.Error(w, err.Error(), ...)` em todo o codebase (erp_bridge.go:90, environment.go, hierarchy.go).

---

### CR-02: Encryption Key Fallback Silencioso em Produção

**File:** `backend/handlers/crypto.go:19-27`
**Issue:** Quando `DATABASE_URL` está definida (i.e., ambiente de produção) mas `ENCRYPTION_KEY` está vazia, `getEncryptionKey()` cai silenciosamente no JWT_SECRET como chave de criptografia das credenciais Oracle. O comentário no código diz "Allow fallback but log a loud warning", mas **nenhum log é emitido** — a linha de warning está morta (`_ = "mensagem"`). Isso significa que credenciais Oracle em produção podem estar criptografadas com a mesma chave usada para assinar JWTs sem nenhum aviso no log. Se o JWT_SECRET vazar, todas as credenciais Oracle ficam comprometidas automaticamente.

**Fix:**
```go
func getEncryptionKey() []byte {
    key := os.Getenv("ENCRYPTION_KEY")
    if key == "" {
        if os.Getenv("DATABASE_URL") != "" {
            if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
                // AVISO CRÍTICO — deve aparecer nos logs de produção
                log.Println("SECURITY WARNING: ENCRYPTION_KEY not set in production — Oracle credentials encrypted with JWT_SECRET. Set ENCRYPTION_KEY immediately.")
                key = jwtSecret
            } else {
                log.Fatal("FATAL: Neither ENCRYPTION_KEY nor JWT_SECRET is set in production.")
            }
        } else {
            key = "super-secret-key-change-me-in-prod"
        }
    }
    h := sha256.Sum256([]byte(key))
    return h[:]
}
```

---

### CR-03: Seed Reinicializa Senha do Admin em Toda Execução

**File:** `backend/migrations/004_seed_ferreira_costa.sql:56-63`
**Issue:** O bloco `ELSE` do `IF v_user_id IS NULL` executa `UPDATE users SET password_hash = '$2a$14$...'` toda vez que a migração roda com o admin já existente. Como a migração 004 é idempotente por design (roda a cada reinício caso o hash não esteja no `schema_migrations` — mas **está**), isso não causa problema via o runner normal. Contudo, se o DBA executar o SQL manualmente, ou se houver um bug no runner que apague o registro de `schema_migrations`, a senha do admin será revertida para `123456` **mesmo que tenha sido trocada para uma senha forte**. Este é um vetor de escalada de privilégio.

**Fix:** O bloco `ELSE` deve **omitir** a atualização de `password_hash` e só atualizar campos não-sensíveis como `role` e `is_verified`:
```sql
ELSE
    UPDATE users SET
        role          = 'admin',
        is_verified   = true,
        full_name     = 'Claudio Bezerra (Admin)'
    WHERE email = 'claudio_bezerra@hotmail.com';
    -- NÃO atualizar password_hash para não sobrescrever senha já trocada
END IF;
```

---

### CR-04: Admin Pode Deletar a Si Mesmo (Sem Restrição)

**File:** `backend/handlers/admin.go:326-342`
**Issue:** `DeleteUserHandler` não verifica se o `userID` da requisição corresponde ao usuário autenticado. Um admin pode enviar `DELETE /api/admin/users/delete?id=<seu-próprio-id>` e se auto-deletar, causando perda imediata de acesso ao sistema. No contexto de ferramenta interna com usuário admin único (Ferreira Costa), isso resulta em perda total de acesso — sem nenhuma rota de recuperação além de recriar o admin manualmente no banco.

**Fix:**
```go
func DeleteUserHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := r.URL.Query().Get("id")
        if userID == "" || !isValidUUID(userID) {
            http.Error(w, "Valid User ID required", http.StatusBadRequest)
            return
        }
        // Impedir auto-deleção
        callerID := GetUserIDFromContext(r)
        if callerID != "" && callerID == userID {
            http.Error(w, "Cannot delete your own account", http.StatusForbidden)
            return
        }
        _, err := db.Exec("DELETE FROM users WHERE id = $1", userID)
        // ...
```

---

### CR-05: Endpoint `/api/admin/diagnostic` Referenciado mas Inexistente

**File:** `frontend/src/pages/AdminUsers.tsx:365`
**Issue:** O botão "Diagnóstico de Dados" (`handleDiagnostic`) faz `fetch('/api/admin/diagnostic')` (linha 365), mas esse endpoint **não está registrado em nenhum lugar do backend** (`main.go` não tem este path, nenhum handler com esta função existe). A requisição sempre retorna 404 do Go's DefaultServeMux. Como o handler verifica `if (!res.ok)` e lança o erro, o diagnóstico nunca funciona. Este é um recurso completamente quebrado.

**Fix:** Registrar o handler em `main.go` e implementar `DiagnosticHandler` no backend, ou remover o botão da UI até que o endpoint seja implementado. Dado que o projeto está em Fase 1, a segunda opção é mais segura:
```tsx
// Remover ou desabilitar o botão até que o endpoint exista:
<Button variant="outline" onClick={handleDiagnostic} disabled title="Disponível na Fase 2">
  <Stethoscope className="mr-2 h-4 w-4" /> Diagnóstico de Dados
</Button>
```

---

### CR-06: JWT aceito via Query String — Token em Logs de Servidor e Histórico do Browser

**File:** `backend/handlers/auth.go:216`
**Issue:** `AuthMiddleware` aceita o JWT via `?token=<value>` na query string. Tokens na URL são registrados em logs de acesso do nginx (ver `nginx.conf` — sem filtragem de parâmetros sensíveis), aparecem no histórico do browser, podem vazar em headers `Referer` para recursos externos, e ficam em logs de proxies. O comentário justifica `window.open` para downloads, mas no FB_TESTESFC (validador fiscal) não há nenhum download direto registrado no roteamento da Fase 1.

**Fix:** Se downloads diretos não existem na Fase 1, remover o suporte a `?token=` completamente. Se for necessário para fases futuras, implementar URLs assinadas de curta duração (tokens de download one-time) em vez de reutilizar o JWT de sessão.

---

### CR-07: Managers.tsx Envia Requests sem Token de Autenticação

**File:** `frontend/src/pages/Managers.tsx:24-29`
**Issue:** `fetchManagers` faz `fetch('/api/managers', { headers: { 'X-Company-ID': companyId || '' } })` sem incluir o header `Authorization`. O interceptor global de `AuthContext.tsx` deveria cobrir isso, mas apenas injeta `Authorization` se `tokenRef.current` não está vazio E o header ainda não está presente — a inicialização de `tokenRef` começa como `null` (linha 39) e só é preenchida após o `refresh` assincrono completar. Se `fetchManagers` rodar antes do refresh terminar (o que é provável dado o `useEffect` vazio `[]`), a request vai sem token e retorna 401.

Além disso, o código lê `localStorage.getItem('companyId')` diretamente em vez de usar `useAuth()`, duplicando lógica e ficando dessincronizado com o estado React.

**Fix:**
```tsx
// Use useAuth() para token e companyId, e react-query para cache/retry
import { useAuth } from '@/contexts/AuthContext';
// ...
const { token, companyId } = useAuth();
// Usar useQuery com enabled: !!token para aguardar autenticação
const { data } = useQuery({
  queryKey: ['managers', companyId],
  queryFn: () => fetch('/api/managers', {
    headers: {
      Authorization: `Bearer ${token}`,
      'X-Company-ID': companyId || '',
    }
  }).then(r => r.json()),
  enabled: !!token && !!companyId,
});
```

---

## Warnings

### WR-01: `rand.Read` sem Checagem de Erro

**File:** `backend/handlers/auth.go:148`
**Issue:** `rand.Read(b)` na função `generateRefreshTokenString` ignora o erro retornado. Embora `crypto/rand` raramente falhe no Linux, em ambientes exóticos (containers sem `/dev/urandom`) isso pode gerar tokens com bytes zerados — tokens previsíveis. O mesmo padrão ocorre em `auth.go:777`.

**Fix:**
```go
func generateRefreshTokenString() string {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        panic("crypto/rand unavailable: " + err.Error())
    }
    return hex.EncodeToString(b)
}
```

---

### WR-02: Token Blacklist em Memória — Não Sobrevive a Restart

**File:** `backend/handlers/auth.go:92-93`
**Issue:** `refreshTokenStore` e `tokenBlacklist` são `sync.Map` em memória. Após restart do container (comum com `restart: always` no docker-compose), todos os refresh tokens e blacklistings são perdidos. Tokens revogados via logout voltam a ser válidos. Refresh tokens válidos também são perdidos, forçando re-login (menor problema). O maior problema é que logout não tem efeito durável — um token revogado antes do restart pode ser reutilizado após.

**Fix:** Para Fase 1 (ferramenta interna), documentar a limitação explicitamente. Para produção, persistir blacklist em Redis ou Postgres com TTL automático via `expires_at`.

---

### WR-03: `PromoteUserHandler` Aceita Role Arbitrário sem Validação

**File:** `backend/handlers/admin.go:204`
**Issue:** `db.Exec("UPDATE users SET role = $1 WHERE id = $2", req.Role, userID)` aceita qualquer string como role. Um admin pode criar roles inválidos (`"superadmin"`, `"root"`, etc.) que o `AuthMiddleware` nunca irá verificar explicitamente (só verifica `"admin"`), mas polui o banco e pode causar comportamento inesperado em checks futuros.

**Fix:**
```go
if req.Role != "" {
    if req.Role != "admin" && req.Role != "user" {
        http.Error(w, "Invalid role. Must be 'admin' or 'user'", http.StatusBadRequest)
        return
    }
    _, err := db.Exec("UPDATE users SET role = $1 WHERE id = $2", req.Role, userID)
    // ...
}
```

---

### WR-04: `GetGroupsHandler` e `GetCompaniesHandler` sem Filtro por Empresa/Usuário

**File:** `backend/handlers/environment.go:193-231` (GetGroupsHandler), `backend/handlers/environment.go:330-379` (GetCompaniesHandler)
**Issue:** `GetGroupsHandler` retorna todos os grupos sem filtrar por usuário (apenas por `environment_id` se fornecido). Um usuário não-admin sem `environment_id` na query recebe **todos os grupos de todas as empresas**. Idem para `GetCompaniesHandler` — sem `group_id` retorna todas as empresas do banco. Isso expõe dados de hierarquia de outros tenants para qualquer usuário autenticado.

Isso é explorado na UI: `HierarchyCascadeSelects` em `AdminUsers.tsx` busca todos os ambientes sem filtro para popular o dropdown de reatribuição — o que é intencional para admin — mas o endpoint é acessível a qualquer usuário autenticado (role `""`).

**Fix:** Adicionar filtro por role: non-admins só veem grupos/empresas vinculados ao seu ambiente:
```go
// Em GetGroupsHandler, se não é admin:
if role != "admin" {
    query += " AND environment_id IN (SELECT environment_id FROM user_environments WHERE user_id = $X)"
    args = append(args, userID)
}
```

---

### WR-05: Race Condition no `AuthContext` — `isAuthenticated` baseado em `!!user` mas `user` vem do localStorage

**File:** `frontend/src/contexts/AuthContext.tsx:241`
**Issue:** `isAuthenticated: !!user` é `true` imediatamente após restauração do `user` do localStorage (linha 75-76), **antes** do refresh token ser validado (chamada assíncrona à linha 85). Isso significa que `ProtectedRoute` deixa o usuário entrar com `isAuthenticated=true` mesmo que o refresh cookie esteja expirado, e o usuário só é redirecionado ao `/login` depois que o fetch de refresh retorna 401. Durante esse intervalo, componentes protegidos são montados com dados stale do localStorage e fazem requests que retornam 401.

**Fix:** Iniciar `user` como `null` e só setá-lo após o refresh confirmar a sessão. Enquanto `loading=true`, `ProtectedRoute` já retorna `null` — este comportamento está correto. O problema é que `loading` é setado como `true` inicialmente e vai para `false` no `.finally()`, mas `user` é setado antes disso (linha 76). A solução é não restaurar `user` do localStorage diretamente, apenas restaurar dados não-sensíveis de display até o refresh confirmar.

---

### WR-06: `SetPreferredCompanyHandler` Não Valida se a Empresa Pertence ao Usuário

**File:** `backend/handlers/auth.go:1095-1115`
**Issue:** A query do `INSERT INTO user_environments ... SELECT ... FROM companies WHERE c.id = $2::uuid` vincula o usuário ao ambiente da empresa solicitada sem verificar se o usuário tem acesso a essa empresa. Qualquer usuário autenticado pode definir como preferida uma empresa de outro tenant se souber o UUID dela. Embora não dê acesso direto (o `GetEffectiveCompanyID` faz a validação de acesso), cria uma entrada espúria em `user_environments` vinculando o usuário a um ambiente ao qual não pertence — o que pode causar vazamento de dados via `GetGroupsHandler`/`GetCompaniesHandler` (WR-04) e confusão nos logs.

**Fix:** Adicionar validação de ownership antes do upsert:
```go
// Verificar que o usuário tem acesso à empresa antes de salvar preferência
_, err := GetEffectiveCompanyID(db, userID, body.CompanyID)
if err != nil {
    http.Error(w, "company not accessible", http.StatusForbidden)
    return
}
```

---

### WR-07: Erro Silenciado no `ERPBridgeConfigHandler` PATCH — Falhas de UPDATE Ignoradas

**File:** `backend/handlers/erp_bridge.go:153-176`
**Issue:** As chamadas `db.Exec(...)` para atualizar credenciais individualmente (fbtax_email, fbtax_password, oracle_usuario, oracle_senha, erp_type, oracle_dsn) **ignoram completamente o erro retornado**. Se uma dessas updates falhar (timeout, constraint violation, conexão perdida), a API retorna `204 No Content` indicando sucesso enquanto as credenciais não foram salvas. O usuário não recebe feedback de erro.

**Fix:**
```go
if req.FBTaxEmail != nil {
    if _, err := db.Exec(`UPDATE erp_bridge_config SET fbtax_email = $2 WHERE company_id = $1`, companyID, *req.FBTaxEmail); err != nil {
        log.Printf("ERPBridgeConfig PATCH fbtax_email error (company %s): %v", companyID, err)
        http.Error(w, "Erro ao salvar fbtax_email", http.StatusInternalServerError)
        return
    }
}
// Aplicar o mesmo padrão para todos os campos individuais
```
Idealmente, todas essas updates deveriam ser parte de uma única query `UPDATE ... SET ... WHERE company_id = $1` em vez de múltiplas queries separadas.

---

### WR-08: `docker-compose.yml` — Healthcheck usa `postgres` Hardcoded, Ignora `DB_USER`

**File:** `docker-compose.yml:62`
**Issue:** O healthcheck é `pg_isready -U postgres`, mas a variável `POSTGRES_USER` é configurada via `${DB_USER}`. Se `DB_USER` for diferente de `postgres`, `pg_isready -U postgres` pode falhar ou dar falso negativo. O `depends_on: condition: service_healthy` do serviço `api` ficará aguardando indefinidamente se o usuário do banco não for `postgres`.

**Fix:**
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-postgres}"]
```

---

## Info

### IN-01: Senha Admin de Dev (`123456`) em Seed — Documentado mas sem Aviso de Startup

**File:** `backend/migrations/004_seed_ferreira_costa.sql:42-55`
**Issue:** Conforme decisão D-10, a senha `123456` para o admin de dev é aceita. Porém, não há nenhum aviso em startup (similar ao que existe para `JWT_SECRET` e `ENCRYPTION_KEY`) para lembrar de trocar a senha antes de expor em produção. A verificação em `main.go` poderia detectar o hash padrão e emitir um aviso.
**Fix:** Adicionar ao `main.go` (após `onDBConnected`) uma verificação que detecta se `password_hash = '$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC'` ainda está ativo para o admin e emite `log.Println("AVISO: Admin com senha padrão de dev detectada. Troque via /change-password antes de ir a produção.")`.

---

### IN-02: `Login.tsx` Redireciona para `/mercadorias` — Rota Inexistente

**File:** `frontend/src/pages/Login.tsx:58`
**Issue:** `navigate("/mercadorias")` após login bem-sucedido, mas a rota `/mercadorias` não existe em `App.tsx`. O usuário é redirecionado para `ProtectedRoute` que cai no path `/*` → `AppLayout` → `<Route path="/" element={<Navigate to="/config/erp-bridge" replace />} />`. Funciona acidentalmente, mas é confuso e pode quebrar se a navegação mudar.
**Fix:** `navigate("/config/erp-bridge")` ou `navigate(location.state?.from?.pathname || "/config/erp-bridge")`.

---

### IN-03: `useState` Usado Indevidamente para Side Effect no Login.tsx

**File:** `frontend/src/pages/Login.tsx:31-35`
**Issue:** `useState(() => { fetch("/api/health")... })` usa o argumento de inicialização do `useState` (que recebe uma função) como um side effect. O correto é `useEffect`. Embora funcione (o inicializador roda uma vez), é um anti-pattern que pode se comportar incorretamente no modo Strict do React (que chama inicializadores duas vezes em desenvolvimento), causando dois fetches ao `/api/health`.
**Fix:** Substituir por `useEffect(() => { fetch("/api/health")... }, [])`.

---

### IN-04: `services/email.go` — `SendAIReportEmail` e Funções Auxiliares de Relatório IA são Código Morto na Fase 1

**File:** `backend/services/email.go:216-940`
**Issue:** `SendAIReportEmail`, `generateTaxComparisonSVG`, `generateTaxTableHTML`, e as funções auxiliares de HTML do relatório executivo são código herdado do FB_APU04 que não tem caller no FB_TESTESFC (validador fiscal). Este código compila mas não é utilizado, aumentando o tamanho do binário e a superfície de manutenção. `TaxComparisonData` e funções de formatação de ICMS/IBS/CBS são do domínio de apuração fiscal, não de validação de pacote fiscal.
**Fix:** Remover as funções não utilizadas do serviço de email, mantendo apenas `SendPasswordResetEmail` e `GetEmailConfig`. Adicionar ticket para avaliar se alguma delas será necessária em fases futuras.

---

### IN-05: `GestaoAmbiente.tsx` — Interface `Company` com Campos Inexistentes no Schema

**File:** `frontend/src/pages/GestaoAmbiente.tsx:56-69`
**Issue:** A interface `Company` no frontend declara `cnpj`, `cnae_secundario`, `municipio` e `incentivos_fiscais`, mas esses campos **não existem na tabela `companies` do schema** (conforme declarado em `backend/handlers/environment.go:29-41` e `migration 001`). As queries em `GetCompaniesHandler` e `CreateCompanyHandler` não retornam esses campos. Isso não causa erro de runtime (TypeScript não verifica JSON em runtime), mas os campos aparecem como `undefined` no frontend, criando confusão.
**Fix:** Atualizar a interface `Company` no frontend para espelhar exatamente os campos retornados pelo backend: remover `cnpj`, `cnae_secundario`, `municipio`, `incentivos_fiscais`.

---

_Revisado: 2026-06-30_
_Revisor: Claude (gsd-code-reviewer)_
_Profundidade: standard_
