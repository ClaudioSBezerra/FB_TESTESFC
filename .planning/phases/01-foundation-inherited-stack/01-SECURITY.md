---
phase: 01
slug: foundation-inherited-stack
status: verified
threats_open: 0
asvs_level: 1
created: 2026-07-01
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Auditado em:** 2026-07-01
**Auditor:** gsd-security-auditor
**ASVS Level:** 1
**block_on:** high
**Threats Closed:** 26/26 (25 do register original + T-04-06 adicionada nesta auditoria)
**Threats Open:** 0/26

Esta auditoria verifica, ameaça por ameaça, se cada mitigação declarada nos `<threat_model>` dos 5 PLANs da Fase 1
(01-01 a 01-05) está de fato presente no código implementado (pós-correções do code-review, commits `fix(01): CR-01..CR-07, WR-01..WR-08`).
Nenhum arquivo de implementação foi modificado durante a auditoria automatizada — apenas leitura e verificação via grep/leitura de código.
Uma superfície de ataque não mapeada (WARNING-1 / T-04-06) foi encontrada pelo auditor e resolvida pelo orquestrador logo em seguida (ver seção dedicada).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|----------------|
| Browser → API (/api/auth/*) | Credenciais de login e tokens cruzam aqui (entrada não confiável) | Email/senha, JWT, cookie refresh |
| Browser (3004) → API (8085) | Proxy Vite/Nginx encaminha /api | Credenciais e tokens |
| API → Postgres | Queries parametrizadas; dados de auth/tenancy persistidos | Dados de usuário, hashes, tokens |
| Browser → /api/admin/*, /api/config/*, /api/managers* | Operações privilegiadas e escopadas por empresa | Comandos CRUD, X-Company-ID |
| Browser → /api/erp-bridge/* | Credenciais Oracle (DSN/usuário/senha) cruzam; exigem autenticação | DSN, usuário, senha Oracle (criptografados em repouso) |
| API → Oracle (externo) | Conexão de teste sai do servidor para host Oracle salvo pela empresa | Ping de conexão, sem consulta a prod/PRODB (D-14) |
| .env → containers | Segredos (JWT_SECRET, ENCRYPTION_KEY, SMTP) injetados via ambiente | Segredos de aplicação |

---

## Threat Register

### Plano 01-01 — Fundação backend (auth, crypto, migrations)

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-01-01 | Spoofing | POST /api/auth/login | mitigate | `bcrypt.GenerateFromPassword(..., 14)` (auth.go:131); `LoginRL` rate limiter (auth.go:562,581,596; middleware.go:142) | closed |
| T-01-02 | Elevation of Privilege | JWT (alg confusion) | mitigate | `jwt.SigningMethodHS256` (auth.go:146); `getJWTSecret()`/`ValidateJWTSecret()` (auth.go:64-80) chamada no startup em main.go:231 — `log.Fatal` se secret ausente | closed |
| T-01-03 | Information Disclosure | Cookie refresh_token | mitigate | `setRefreshCookie` httpOnly + SameSite=Strict (auth.go:163-173); access token só em memória React | closed |
| T-01-04 | Tampering | Queries Postgres | mitigate | Parâmetros `$1/$2` via lib/pq confirmados em todos os handlers | closed |
| T-01-05 | Information Disclosure | Senha padrão admin (123456) | accept | Ver Accepted Risks Log abaixo | closed |
| T-01-06 | Elevation of Privilege | CORS | mitigate | `GetAllowedOrigins()` + `SecurityMiddleware` (middleware.go:13-35,95-121) | closed |
| T-01-SC | Tampering | go mod download (supply chain) | mitigate | Apenas deps já em produção no FB_APU04; sem prometheus/excelize/rardecode | closed |

### Plano 01-02 — Frontend + Docker (Walking Skeleton)

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-02-01 | Information Disclosure | Access token via XSS | mitigate | AuthContext mantém token em memória React (useState/useRef), nunca em localStorage | closed |
| T-02-02 | Spoofing/CSRF | /api/auth/refresh | mitigate | Cookie SameSite=Strict; CORS whitelist ALLOWED_ORIGINS | closed |
| T-02-03 | Information Disclosure | Segredos em .env versionado | mitigate | `.gitignore` ignora `.env`/`.env.*`; confirmado via `git check-ignore -v .env` e `git ls-files` | closed |
| T-02-04 | Tampering | Imagem Docker do backend | mitigate | `go mod download` com go.sum; `CGO_ENABLED=0`; sem `-mod=vendor` | closed |
| T-02-05 | Elevation of Privilege | CORS no proxy | mitigate | SecurityMiddleware valida Origin antes de refletir Access-Control-Allow-Origin | closed |

### Plano 01-03 — Camada de gestão (tenancy)

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-03-01 | Elevation of Privilege | /api/admin/users* | mitigate | `withAuth(handler, "admin")` em todas as 5 rotas admin | closed |
| T-03-02 | Tampering | Escopo de empresa | mitigate | `GetEffectiveCompanyID` valida vínculo do usuário antes de aceitar X-Company-ID | closed |
| T-03-03 | Information Disclosure | Listagem de usuários | mitigate | `ListUsersHandler` não retorna password_hash | closed |
| T-03-04 | Tampering | Queries managers/environment | mitigate | Parâmetros `$1` via lib/pq | closed |

### Plano 01-04 — ERP_BRIDGE + go-ora

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-04-01 | Information Disclosure | Exfiltração de credenciais Oracle via API (/config) | mitigate | `oracle_senha` nunca retornado — apenas flags `*_set`; AES-256-GCM em crypto.go | closed |
| T-04-02 | Information Disclosure | Erro do test-connection vazando segredos | mitigate | CR-01 corrigido e validado em runtime — mensagem genérica sanitizada ao cliente | closed |
| T-04-03 | Spoofing/SSRF | test-connection host arbitrário | mitigate | `withAuth`; usa só DSN salvo em erp_bridge_config da empresa do JWT; timeout 10s | closed |
| T-04-04 | Tampering | go.mod/go.sum (supply chain go-ora) | mitigate | go-ora/v2@v2.9.0 verificado via proxy.golang.org; checkpoint blocking-human aprovado pelo usuário | closed |
| T-04-05 | Elevation of Privilege | generate-api-key | mitigate | `withAuth(..., "admin")` | closed |
| T-04-06 | Information Disclosure | GET /api/erp-bridge/credentials retorna credenciais Oracle em texto claro via X-API-Key, sem JWT e sem rate limit | mitigate | **Endpoint DESATIVADO** — rota comentada em `backend/main.go` (commit `494b5ab`, 2026-07-01). Handler mantido no código para reativação futura quando um daemon consumidor real for construído, com rate limiting e revisão de ameaça dedicada. Ver seção dedicada abaixo. | closed |

### Plano 01-05 — Telas de gestão (frontend)

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-05-01 | Elevation of Privilege | Rotas de admin no cliente | mitigate | `AdminRoute` no cliente + `withAuth(..., "admin")` no backend (defense in depth) | closed |
| T-05-02 | Information Disclosure | Exibição de credenciais Oracle | mitigate | UI exibe apenas flags `*_set`; senha nunca retornada pelo backend | closed |
| T-05-03 | Spoofing | Reset de senha | mitigate | Token com expiração; resposta genérica anti-enumeração de contas | closed |
| T-05-04 | Information Disclosure | Mensagem de erro do teste de conexão na UI | mitigate | Exibe a mensagem já sanitizada pelo backend (T-04-02) | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|--------------|------|
| AR-01 | T-01-05 | Senha padrão do admin (`123456`) — ferramenta interna de uso único (Ferreira Costa), acesso Oracle somente-leitura, sem exposição pública planejada na Fase 1. Seed nunca loga a senha em claro (só o hash bcrypt); CR-03 corrigido — o seed não reverte a senha em reexecuções se já foi trocada; `ChangePasswordHandler` disponível para troca autenticada. Débito residual não bloqueante (IN-01, code-review): sem aviso de startup detectando hash padrão ainda ativo — recomendado antes de qualquer exposição externa, não bloqueia a Fase 1. | Usuário (D-10, `01-CONTEXT.md`) | 2026-06-30 |

*Accepted risks do not resurface in future audit runs.*

---

## Unregistered Flag Resolvida — T-04-06

### `GET /api/erp-bridge/credentials` retornava credenciais Oracle em texto claro

- **Arquivo:** `backend/handlers/erp_bridge.go:267-323` (`ERPBridgeCredentialsHandler`)
- **Achado pelo auditor:** o endpoint decifrava e retornava `oracle_senha`, `fbtax_password` e `oracle_usuario` em texto claro no JSON para qualquer requisição com um `X-API-Key` válido — autenticado apenas via hash SHA-256 comparado no banco, **sem JWT e sem rate limiting**. Distinto de T-04-01 (que cobre `/api/erp-bridge/config`, usado pela UI, e nunca retorna a senha).
- **Por que era um risco real e não apenas teórico:** o endpoint estava **registrado e alcançável pela rede** em `main.go:340` (porta 8085), mesmo sem nenhum consumidor real (o daemon que o usaria ainda não existe). D-14 escopou a Fase 1 estritamente a "infra de conexão + testar conexão" — este endpoint já era preparação para uma fase futura, portanto superfície de ataque desnecessária hoje.
- **Decisão do usuário (2026-07-01):** desativar o endpoint agora, em vez de aceitar como risco documentado ou apenas adicionar rate limit.
- **Correção aplicada:** rota comentada em `backend/main.go` (commit `494b5ab`); handler `ERPBridgeCredentialsHandler` permanece no código, documentado, para reativação quando o daemon consumidor for de fato construído (fase futura), junto com rate limiting e uma revisão de ameaça dedicada nesse momento.
- **Validação:** `cd backend && go build ./...` e `go vet ./...` — exit 0 após a mudança.
- **Status:** CLOSED (mitigado por desativação)

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|----------------|--------|------|--------|
| 2026-07-01 | 26 | 26 | 0 | gsd-security-auditor + orquestrador (desativação do T-04-06) |

---

## Achados do Code-Review (contexto, reforçam mitigações acima)

| ID | Descrição | Status verificado |
|----|-----------|--------------------|
| CR-01 | Sanitização erro test-connection | Confirmado (reforça T-04-02) |
| CR-02 | Log de aviso ENCRYPTION_KEY ausente | Confirmado — `crypto.go:24`, `ValidateEncryptionKey()` chamada em main.go:234 |
| CR-03 | Seed não reverte senha do admin | Confirmado (reforça T-01-05) |
| CR-04 | Admin não pode se auto-deletar | Confirmado — admin.go:338-343 |
| CR-06 | JWT via query string removido | Confirmado — auth.go só aceita Authorization: Bearer |
| CR-07 | Managers.tsx aguarda token antes de fetch | Confirmado |
| WR-01 | `rand.Read` com checagem de erro | Confirmado |
| WR-03 | `PromoteUserHandler` valida role | Confirmado |
| WR-04 | Filtro por ambiente em grupos/empresas | Confirmado |
| WR-05 | `isAuthenticated` baseado em `!!token` | Confirmado |
| WR-06 | `SetPreferredCompanyHandler` valida acesso | Confirmado |
| WR-07 | Erros de UPDATE checados no PATCH ERPBridge | Confirmado |
| WR-08 | Healthcheck usa `${DB_USER}` | Confirmado |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-01
