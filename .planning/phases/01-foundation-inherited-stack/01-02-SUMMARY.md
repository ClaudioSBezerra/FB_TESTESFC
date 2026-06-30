---
phase: 01-foundation-inherited-stack
plan: "02"
subsystem: ui
tags: [react, typescript, vite, tailwind, docker, jwt, auth, nginx]

# Dependency graph
requires:
  - phase: 01-01
    provides: Backend Go com endpoints /api/auth/login e /api/auth/refresh, migrações 001+004, seed admin Ferreira Costa

provides:
  - Shell do frontend React com AuthContext (JWT em memória + restore via cookie httpOnly)
  - 4 páginas de autenticação (Login, Register, ForgotPassword, ResetPassword)
  - 5 telas de gestão antecipadas do Plano 01-05 (GestaoAmbiente, Managers, AdminUsers, ERPBridgeConfig, ERPBridgeCredenciais)
  - AppRail com mainItems=[] e botão Configurações aponta /config/erp-bridge
  - App.tsx enxuto com ProtectedRoute, sem FilialProvider/CompanySwitcher/AjudaChat
  - docker-compose.yml com serviços api/web/db em portas/volume exclusivos do FB_TESTESFC
  - Dockerfiles para backend (sem -mod=vendor) e frontend (Vite + Nginx)
  - .env.example documentando todos os segredos necessários
  - .gitignore protegendo .env e artefatos de build
  - Walking Skeleton validado: login + persistência de sessão via cookie httpOnly

affects:
  - 01-03 (gestão backend — telas já existem no frontend; rotas de API pendentes)
  - 01-04 (ERP_BRIDGE backend — tela ERPBridgeCredenciais já existe; botão Testar Conexão aguarda endpoint)
  - 01-05 (telas de gestão — parcialmente antecipado; foco muda para navegação/menus e integração do botão Testar Conexão)

# Tech tracking
tech-stack:
  added:
    - React 18 + TypeScript + Vite (porta dev 3004)
    - Tailwind CSS + shadcn/ui (46 componentes)
    - React Query (TanStack)
    - React Router DOM
    - Nginx (proxy reverso para /api, SPA fallback)
    - Docker Compose com rede fb_testesfc_net
  patterns:
    - AuthContext: JWT em memória React (não localStorage) + refresh token httpOnly SameSite=Strict
    - fetch interceptor global em AuthContext (window.fetch override para injetar Bearer token)
    - Restore de sessão na montagem: POST /api/auth/refresh com credentials:include
    - ProtectedRoute/AdminRoute como wrappers de rota
    - AppLayout como shell autenticado com AppRail lateral
    - Vite proxy em dev /api → localhost:8085; Nginx proxy em prod /api/ → fb_testesfc-api:8085

key-files:
  created:
    - frontend/src/contexts/AuthContext.tsx
    - frontend/src/App.tsx
    - frontend/src/components/AppRail.tsx
    - frontend/src/lib/navigation.ts
    - frontend/src/lib/utils.ts
    - frontend/src/lib/logger.ts
    - frontend/src/pages/Login.tsx
    - frontend/src/pages/Register.tsx
    - frontend/src/pages/ForgotPassword.tsx
    - frontend/src/pages/ResetPassword.tsx
    - frontend/src/pages/GestaoAmbiente.tsx
    - frontend/src/pages/Managers.tsx
    - frontend/src/pages/AdminUsers.tsx
    - frontend/src/pages/ERPBridgeConfig.tsx
    - frontend/src/pages/ERPBridgeCredenciais.tsx
    - frontend/src/components/ui/ (46 arquivos shadcn)
    - frontend/vite.config.ts
    - frontend/package.json
    - frontend/Dockerfile
    - frontend/nginx.conf
    - frontend/index.html
    - backend/Dockerfile
    - docker-compose.yml
    - .env.example
    - .gitignore
  modified: []

key-decisions:
  - "Token de acesso em memória React, nunca localStorage — proteção contra XSS (T-02-01)"
  - "Refresh token em cookie httpOnly SameSite=Strict — proteção contra CSRF (T-02-02)"
  - "docker-compose.yml sem redis/prometheus/grafana/exporters — escopo enxuto v1"
  - "backend/Dockerfile sem -mod=vendor; usa go mod download com go.sum verificado (P3)"
  - "AppRail com mainItems=[] neste plano — menus/navegação delegados ao Plano 01-05"
  - "5 telas de gestão antecipadas do 01-05: telas existem, integração de menus e botão Testar Conexão ficam no 01-05"
  - "Rota / redireciona para /config/erp-bridge (tela cujo backend nasce no 01-04 — comportamento parcial esperado)"

patterns-established:
  - "AuthContext pattern: fetch interceptor global + restore na montagem + token em memória"
  - "ProtectedRoute pattern: verifica isAuthenticated, redireciona /login se não autenticado"
  - "AppLayout pattern: AppRail lateral + <Outlet /> para conteúdo das rotas protegidas"
  - "Docker pattern: serviços api/web/db com nomes fb_testesfc-*, portas 8085/3004/5435, volume postgres_data_testesfc, rede fb_testesfc_net"
  - "Segredos pattern: .env real nunca versionado; .env.example com placeholders; .gitignore protege .env"

requirements-completed: [FND-01, AUTH-01, AUTH-02, AUTH-03]

# Metrics
duration: ~30min
completed: "2026-06-30"
---

# Phase 1 Plan 02: Walking Skeleton Frontend + Infra Docker Summary

**Shell React com AuthContext (JWT em memória + cookie httpOnly), 4 páginas de auth + 5 telas de gestão antecipadas, Dockerfiles e docker-compose.yml exclusivos do FB_TESTESFC — Walking Skeleton validado com login e persistência de sessão.**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-06-30T17:44:09-03:00
- **Completed:** 2026-06-30T17:48:02-03:00
- **Tasks:** 2 tasks auto + 1 checkpoint humano (aprovado)
- **Files modified:** 78 arquivos criados

## Accomplishments

- Walking Skeleton completo e validado: `docker compose up` sobe api(8085) + web(3004) + db, migrações 001+004 rodam em base zerada, admin semeado
- Login com `claudio_bezerra@hotmail.com` / `123456` retorna sessão JWT; F5 (refresh) restaura sessão via cookie httpOnly sem novo login
- 46 componentes shadcn/ui + AuthContext (interceptor fetch global + restore de sessão) copiados integralmente do FB_APU04
- Infraestrutura Docker própria do FB_TESTESFC: portas/volumes/rede distintos do FB_APU04 para coexistência local

## Task Commits

Cada task foi commitada atomicamente:

1. **Task 1: Shell do frontend + páginas de auth + App.tsx enxuto** - `80be77c` (feat)
2. **Task 2: vite/package.json/Dockerfiles + docker-compose + .env.example** - `90556fa` (feat)
3. **Desvio: .gitignore para proteger .env** - `fdbddb1` (chore — aplicado pelo orquestrador após checkpoint)

**Checkpoint Walking Skeleton:** Aprovado pelo usuário via validação da API (HTTP 200 + JWT + cookie refresh_token httpOnly/SameSite=Strict) e confirmação visual do login no browser.

## Files Created/Modified

- `frontend/src/contexts/AuthContext.tsx` — JWT em memória + interceptor fetch global + restore via POST /api/auth/refresh
- `frontend/src/App.tsx` — Router enxuto: rotas públicas /login /register /forgot-password /reset-senha + árvore /* protegida → AppLayout
- `frontend/src/components/AppRail.tsx` — Sidebar lateral com mainItems=[] (menus no 01-05) + botão Configurações → /config/erp-bridge
- `frontend/src/lib/navigation.ts` — Itens de navegação Phase 1 (módulos de gestão)
- `frontend/src/pages/Login.tsx`, `Register.tsx`, `ForgotPassword.tsx`, `ResetPassword.tsx` — Páginas de autenticação
- `frontend/src/pages/GestaoAmbiente.tsx`, `Managers.tsx`, `AdminUsers.tsx`, `ERPBridgeConfig.tsx`, `ERPBridgeCredenciais.tsx` — Telas de gestão antecipadas do Plano 01-05
- `frontend/src/components/ui/` (46 arquivos) — Biblioteca shadcn/ui completa
- `frontend/vite.config.ts` — Porta dev 3004, proxy /api → localhost:8085
- `frontend/package.json` — name=fb_testesfc-frontend
- `frontend/Dockerfile` — Build Vite + Nginx serve
- `frontend/nginx.conf` — Proxy /api/ → fb_testesfc-api:8085, SPA fallback
- `backend/Dockerfile` — go mod download + CGO_ENABLED=0 (sem -mod=vendor)
- `docker-compose.yml` — Serviços api/web/db, portas 8085/3004/5435, volume postgres_data_testesfc, rede fb_testesfc_net, sem redis/prometheus/grafana
- `.env.example` — DB_NAME=fb_testesfc_db, PORT=8085, ENCRYPTION_KEY (P4), APP_URL=localhost:3004
- `.gitignore` — Protege .env, node_modules, dist/build

## Decisions Made

- Token de acesso mantido em memória React (não localStorage) para proteção contra XSS — refresh token em cookie httpOnly SameSite=Strict para proteção contra CSRF
- docker-compose.yml sem redis/prometheus/grafana/postgres-exporter — escopo v1 enxuto, sem overhead de observabilidade ainda
- backend/Dockerfile sem flag `-mod=vendor`; usa `go mod download` com go.sum verificado (P3) — elimina a dependência do diretório vendor do FB_APU04
- AppRail com `mainItems = []` neste plano — menus de navegação delegados ao Plano 01-05 conforme decisão da PATTERNS.md

## Deviations from Plan

### Trabalho Antecipado do Plano 01-05

**1. [Antecipação - Escopo Expandido] 5 telas de gestão copiadas antes do previsto**

- **Encontrado em:** Task 1 (Shell do frontend + páginas de auth)
- **O que ocorreu:** O executor do 01-02 copiou as 5 telas de gestão do FB_APU04 (`GestaoAmbiente.tsx`, `Managers.tsx`, `AdminUsers.tsx`, `ERPBridgeConfig.tsx`, `ERPBridgeCredenciais.tsx`) e registrou todas as rotas correspondentes no `App.tsx`. O PLAN do 01-02 previa apenas as 4 páginas de auth.
- **Consequência para o Plano 01-05:** As telas e rotas já existem. O 01-05 deve focar em: (a) preencher `mainItems` do AppRail com itens de menu funcionais via `navigation.ts`, e (b) integrar o botão "Testar Conexão Oracle" (`ERPBridgeCredenciais.tsx`) após o endpoint `POST /api/erp-bridge/test-connection` ser criado no Plano 01-04. O 01-05 **não deve recopiar** essas telas.
- **Consequência para o Plano 01-04:** O endpoint `POST /api/erp-bridge/test-connection` é aguardado por `ERPBridgeCredenciais.tsx` que já existe no frontend. Prioridade no 01-04.
- **Estado atual (comportamento parcial esperado):** A rota `/` redireciona para `/config/erp-bridge` (tela de configuração do ERP_BRIDGE), cujo backend nasce no Plano 01-04 (Wave 3). No estado atual, logar leva a uma tela cujo backend ainda não existe — este é o comportamento parcial esperado no meio da Phase 1.
- **Arquivos afetados:** `frontend/src/pages/GestaoAmbiente.tsx`, `Managers.tsx`, `AdminUsers.tsx`, `ERPBridgeConfig.tsx`, `ERPBridgeCredenciais.tsx`, `frontend/src/App.tsx`, `frontend/src/lib/navigation.ts`
- **Commit:** `80be77c`

### Correção de Segurança pelo Orquestrador

**2. [Rule 2 - Segurança] .gitignore criado para proteger .env**

- **Encontrado em:** Após o checkpoint humano (entre checkpoint e finalização)
- **Problema:** O arquivo `.env` (com segredos reais — JWT_SECRET, ENCRYPTION_KEY, DB_PASSWORD) não estava sendo ignorado pelo Git. Apenas `.env.example` estava versionado, mas sem `.gitignore`, um `git add .` acidental poderia versionar credenciais reais.
- **Correção:** Criado `.gitignore` na raiz do projeto cobrindo `.env`, `node_modules/`, `dist/`, `build/`, arquivos de log e binários Go. Mitigação direta de T-02-03 do threat model.
- **Arquivos criados:** `.gitignore`
- **Commit:** `fdbddb1`

---

**Total de desvios:** 2 (1 antecipação de escopo, 1 correção de segurança)
**Impacto no plano:** A antecipação das telas de gestão reduz o escopo do Plano 01-05; o .gitignore era requisito de segurança crítico (T-02-03).

## Issues Encountered

Nenhum problema bloqueante. O Walking Skeleton foi validado com sucesso:
- `POST /api/auth/login` retornou HTTP 200 + JWT + cookie `refresh_token` httpOnly/SameSite=Strict
- `POST /api/auth/refresh` retornou HTTP 200 com novo access token
- Migrações 001 e 004 aplicadas em base zerada
- Admin `claudio_bezerra@hotmail.com` semeado e funcional
- Confirmação visual do login no browser pelo usuário

## User Setup Required

Para subir o ambiente pela primeira vez:
1. Copiar `.env.example` para `.env` e preencher `DB_PASSWORD`, `JWT_SECRET` (openssl rand -hex 32), `ENCRYPTION_KEY` (openssl rand -hex 32)
2. `docker compose up --build` na raiz do projeto
3. Aguardar os 3 containers subirem e as migrações 001+004 serem aplicadas
4. Acessar http://localhost:3004 e fazer login com `claudio_bezerra@hotmail.com` / `123456`

## Next Phase Readiness

**Plano 01-03 (Gestão backend):** Pronto para iniciar. As telas de gestão (AdminUsers, Managers, GestaoAmbiente) já existem no frontend e aguardam as rotas de API.

**Plano 01-04 (ERP_BRIDGE backend):** A tela `ERPBridgeCredenciais.tsx` com botão "Testar Conexão Oracle" já existe. O Plano 01-04 deve criar o endpoint `POST /api/erp-bridge/test-connection` para que o botão funcione.

**Plano 01-05 (Telas de gestão — navegação):** Escopo reduzido pela antecipação: focar em (a) preencher `mainItems` no AppRail/navigation.ts e (b) integrar botão "Testar Conexão" após o 01-04. **Não recopiar telas.**

**Bloqueador conhecido:** Rota `/` aponta para `/config/erp-bridge` (ERP_BRIDGE), mas o backend dessa tela nasce no 01-04. No estado atual, após login o usuário vê a tela de ERP_BRIDGE sem dados — comportamento parcial esperado até o 01-04 ser executado.

---
*Phase: 01-foundation-inherited-stack*
*Completed: 2026-06-30*
