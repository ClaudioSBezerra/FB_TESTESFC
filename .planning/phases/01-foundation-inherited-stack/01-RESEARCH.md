# Phase 1: Foundation & Inherited Stack — Pesquisa

**Pesquisado:** 2026-06-30
**Domínio:** Cópia seletiva de módulos Go+React do FB_APU04 para novo projeto enxuto
**Confiança:** HIGH

---

<user_constraints>
## Restrições do Usuário (de 01-CONTEXT.md)

### Decisões Bloqueadas (D-01 a D-14)

- **D-01:** Cherry-pick por dependência — copiar arquivo a arquivo apenas o necessário.
- **D-02:** `main.go` novo e enxuto, registrando apenas as rotas dos módulos copiados.
- **D-03:** `go.mod` limpo + `go mod tidy` — partir do go.mod do FB_APU04, remover deps de módulos não usados. Não copiar `vendor/`.
- **D-04:** Frontend = shell compartilhado (layout, router, AuthContext, api client, UI) + apenas as páginas no escopo.
- **D-05:** Schema inicial consolidado — 1–3 migrações novas e enxutas em vez das 149 do FB_APU04.
- **D-06:** Schema da Fase 1 inclui apenas auth + tenancy. Tabelas de NFe entram na Fase 2.
- **D-07:** Usar o mesmo mecanismo/runner de migração do FB_APU04 (a confirmar pelo researcher ← este documento resolve).
- **D-08:** Manter modelo hierárquico (ambiente→grupo→empresa→usuário), semeando apenas Ferreira Costa + admin.
- **D-09:** Ambiente/empresa Ferreira Costa + admin criados via migração de seed SQL idempotente.
- **D-10:** Admin inicial: `claudio_bezerra@hotmail.com`, senha `123456` (hash pre-computado na seed).
- **D-11:** Contexto de empresa resolvido pelo mecanismo herdado (claim JWT / middleware). Sem CompanySwitcher na UI.
- **D-12:** Renomear para `fb_testesfc` — module path, imports e nome do app.
- **D-13:** Docker-compose com serviços, portas e banco Postgres próprios (portas distintas do FB_APU04).
- **D-14:** ERP_BRIDGE na Fase 1 = infra de conexão Oracle + telas ERPBridgeConfig/Credenciais + ação "testar conexão". NÃO consultar prod/PRODB (Fase 2).

### Discricionariedade do Claude (resolvidas aqui)
- Lib/abordagem exata do runner de migração (D-07) — **resolvido: seção "Runner de Migração"**.
- Corte de dependências do shell do frontend (D-04) — **resolvido: seção "Grafo de Dependências do Frontend"**.
- Mapeamento concreto de portas/volumes do compose (D-13) — **resolvido: seção "Mapa de Portas e Serviços"**.

### Ideias Adiadas (FORA DO ESCOPO desta fase)
- Tabelas de NFe saída (cabeçalho/itens/impostos) → Fase 2.
- Lookup de grupo fiscal em prod/PRODB → Fase 2.
- CompanySwitcher / multi-tenant completo (MTN-01) → v2.
- Gestão de credenciais do admin via .env / troca obrigatória no 1º login → v1 usa default fixo.

</user_constraints>

---

<phase_requirements>
## Requisitos da Fase

| ID | Descrição | Suporte desta pesquisa |
|----|-----------|------------------------|
| FND-01 | Go 1.24 + React/TS/Vite/Tailwind + Postgres inicializados (build e run locais) | Stack do FB_APU04 confirmada; portas e volumes mapeados para coexistência |
| FND-02 | Apenas módulos necessários copiados (auth, gestão, ERP_BRIDGE, sem módulos fiscais não relacionados) | Lista exata de arquivos a copiar e arquivos a excluir levantada |
| FND-03 | Migrações de banco executam limpo em base zerada | Runner custom confirmado; 4 migrações consolidadas especificadas |
| AUTH-01 | Login com e-mail/senha recebe sessão JWT | `handlers/auth.go` — LoginHandler confirmado |
| AUTH-02 | Sessão persiste entre refreshes | Cookie httpOnly `refresh_token` 7 dias + `sync.Map` in-memory + `POST /api/auth/refresh` confirmados |
| AUTH-03 | Forgot/reset de senha reaproveitando fluxo do FB_APU04 | `handlers/auth.go` + `services/email.go` confirmados |
| AUTH-04 | Rotas da API protegidas por middleware | `handlers.AuthMiddleware(handler, role)` confirmado; teste em `auth_middleware_test.go` |
| TEN-01 | Ambiente/empresa Ferreira Costa configurável com admin | Migration 004 seed especificada com hash bcrypt pré-computado do admin |
| TEN-02 | Admin gerencia usuários (criar/editar/desativar) via gestão herdada | `handlers/admin.go` + `pages/AdminUsers.tsx` confirmados |
| TEN-03 | Contexto de empresa Ferreira Costa resolvido nas requisições | `GetEffectiveCompanyID` em `handlers/auth.go` + header `X-Company-ID` interceptado em `AuthContext.tsx` |

</phase_requirements>

---

## Resumo

Esta fase é uma cópia seletiva de módulos maduros do FB_APU04 para um projeto Go + React + Postgres + Docker novo e enxuto. A inspeção do repositório irmão confirmou que todos os módulos necessários (auth, hierarquia, ERP_BRIDGE, managers) estão prontos para cópia direta, com ajustes cirúrgicos de module path (`fb_apu04` → `fb_testesfc`) e remoção de dependências fora do escopo.

O **runner de migração do FB_APU04 é custom** — cerca de 110 linhas de Go puro embutidas no `main.go`, sem biblioteca externa, usando apenas `filepath.Glob` e `database/sql`. Esse runner deve ser transplantado para o novo `main.go`.

A **cópia do frontend** é bem delimitada: 9 páginas em escopo, `AuthContext`, todos os 44 componentes `ui/` (shadcn/ui), e ajuste do `AppRail` e `navigation.ts`. Não é necessário copiar contextos de filiais, CompanySwitcher nem páginas de módulos fiscais fora do escopo.

O ponto **novo** desta fase que NÃO existe no FB_APU04 é o endpoint `POST /api/erp-bridge/test-connection`, que requer o driver Oracle puro-Go `github.com/sijms/go-ora/v2` (CGO_ENABLED=0 compatível). Este é o único acréscimo além dos módulos herdados.

**Recomendação primária:** Copie os handlers na ordem de dependência (crypto → auth → environment → hierarchy → managers → erp_bridge), escreva as 4 migrações consolidadas do zero, crie um `main.go` enxuto com as rotas listadas neste documento, e adapte o docker-compose com as portas/volumes propostos.

---

## Mapa de Responsabilidades Arquiteturais

| Capacidade | Camada Principal | Camada Secundária | Racional |
|------------|-----------------|-------------------|----------|
| Autenticação JWT | Backend (API) | — | Geração/validação de token é server-side; frontend apenas armazena em memória |
| Sessão persistente (refresh) | Backend (API) | Browser (cookie httpOnly) | Cookie httpOnly enviado pelo backend; browser guarda opacamente |
| Hash/criptografia de credenciais | Backend (API) | — | bcrypt + AES-256-GCM nunca expostos ao cliente |
| Gestão de hierarquia (ambiente/grupo/empresa) | Backend (API) | — | Operações CRUD protegidas por JWT |
| Contexto de empresa por request | Backend (API) | Frontend (header X-Company-ID) | Middleware lê `X-Company-ID`; AuthContext injeta o header em todos os fetches |
| Telas de gestão (AdminUsers, GestaoAmbiente) | Frontend (React SPA) | — | Componentes React consomem a API REST do backend |
| Configuração de credenciais Oracle | Backend (API) | — | Dados sensíveis criptografados (AES-GCM) antes de persistir no Postgres |
| Teste de conexão Oracle | Backend (API) | — | Backend abre conexão Oracle server-side; cliente recebe apenas ok/erro |
| Schema/migrações Postgres | Backend (startup) | — | Runner embutido no `main.go` aplica .sql na inicialização |

---

## Stack Padrão

### Backend Go

| Biblioteca | Versão (go.mod) | Finalidade | Manter? |
|------------|-----------------|------------|---------|
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | JWT geração/validação | SIM |
| `github.com/lib/pq` | v1.11.2 | Driver Postgres | SIM |
| `golang.org/x/crypto` | v0.48.0 | bcrypt para hash de senha | SIM |
| `golang.org/x/text` | v0.34.0 | Processamento de texto | SIM |
| `github.com/joho/godotenv` | v1.5.1 | Carrega `.env` no startup | SIM |
| `github.com/sijms/go-ora/v2` | v2.9.0 (nova dep) | Driver Oracle puro-Go (CGO_ENABLED=0) para "testar conexão" | ADICIONAR |

**Deps do FB_APU04 a REMOVER (não usadas nos módulos copiados):**

| Biblioteca | Motivo da remoção |
|------------|------------------|
| `github.com/xuri/excelize/v2` | Exportação Excel (não usada na Fase 1) |
| `github.com/nwaples/rardecode/v2` | Descompressão RAR (SPED) |
| `github.com/prometheus/client_golang` | Métricas Prometheus (não usadas) |
| `github.com/prometheus/client_model` | Sub-dep de prometheus |
| `github.com/prometheus/common` | Sub-dep de prometheus |
| `github.com/prometheus/procfs` | Sub-dep de prometheus |
| `github.com/ledongthuc/pdf` | Processamento PDF (PRODEPE) |
| `github.com/tiendc/go-deepcopy` | Deep copy (não usada) |
| `github.com/xuri/efp` | Sub-dep excelize |
| `github.com/xuri/nfp` | Sub-dep excelize |
| `github.com/klauspost/compress` | Sub-dep excelize/rardecode |
| `github.com/richardlehane/mscfb` | Sub-dep excelize (OLE) |
| `github.com/richardlehane/msoleps` | Sub-dep excelize (OLE) |
| `github.com/beorn7/perks` | Sub-dep prometheus |
| `github.com/cespare/xxhash/v2` | Sub-dep prometheus |
| `github.com/munnerz/goautoneg` | Sub-dep prometheus |
| `google.golang.org/protobuf` | Sub-dep prometheus |
| `golang.org/x/sys` | Sub-dep de protobuf/prometheus |
| `golang.org/x/net` | Sub-dep excelize |

Após remover as páginas/handlers fora do escopo, executar `go mod tidy` resolverá automaticamente as dependências transitivas.

### Frontend React

| Biblioteca | Versão (package.json) | Finalidade | Manter? |
|------------|----------------------|------------|---------|
| `react`, `react-dom` | 18.x | Base React | SIM |
| `react-router-dom` | ^6.22.3 | Roteamento SPA | SIM |
| `@tanstack/react-query` | última | Data fetching (usado em AdminUsers, ERPBridge) | SIM |
| `lucide-react` | última | Ícones | SIM |
| `sonner` | ^2.0.7 | Toast notifications | SIM |
| `tailwindcss` | ^3.4.3 | CSS utilitário | SIM |
| `@vitejs/plugin-react-swc` | ^3.5.0 | Build Vite + React | SIM |
| Todos `@radix-ui/*` | várias | Base do shadcn/ui (components/ui/) | SIM — usados pelos componentes UI copiados |
| `tailwind-merge` | ^2.2.2 | `cn()` em lib/utils.ts | SIM |
| `tailwindcss-animate` | ^1.0.7 | Animações shadcn | SIM |
| `class-variance-authority` | última | variantes shadcn | SIM |
| `clsx` | última | cn helper | SIM |
| `recharts` | ^3.7.0 | Gráficos (não usado nas páginas copiadas) | PODE REMOVER |
| `react-simple-maps` | ^3.0.0 | Mapas geográficos (não usados) | PODE REMOVER |
| `xlsx` | ^0.18.5 | Exportação Excel (não usada) | PODE REMOVER |
| `react-resizable-panels` | ^4.5.4 | Painéis redimensionáveis (não usados) | PODE REMOVER |
| `zod` | ^4.3.6 | Validação de schema (não usada nas páginas copiadas) | PODE REMOVER |

**Nota:** Remover as dependências não usadas é opcional para Phase 1 — Vite não as inclui no bundle se não forem importadas. Priorize a cópia funcional; trimming de `package.json` pode ser feito pós-validação.

### Instalação (backend)

```bash
# No diretório backend/ do novo projeto
go mod init fb_testesfc
go get github.com/golang-jwt/jwt/v5@v5.3.1
go get github.com/lib/pq@v1.11.2
go get golang.org/x/crypto@v0.48.0
go get golang.org/x/text@v0.34.0
go get github.com/joho/godotenv@v1.5.1
go get github.com/sijms/go-ora/v2@v2.9.0
go mod tidy
```

---

## Auditoria de Legitimidade de Pacotes

> Apenas o pacote Go novo `github.com/sijms/go-ora/v2` é adicionado. Pacotes do FB_APU04 já estão em uso em produção.

| Pacote | Registry | Confirmado via | slopcheck | Disposição |
|--------|----------|----------------|-----------|------------|
| `github.com/sijms/go-ora/v2` | Go module proxy | `proxy.golang.org` v2.9.0 (2025-06-09) [CITED: proxy.golang.org] | N/A (Go pkg, não npm) | Aprovado — puro-Go, sem CGO, project page em github.com/sijms/go-ora |
| `github.com/golang-jwt/jwt/v5` | Go module proxy | Em uso no FB_APU04 em produção | — | Aprovado |
| `github.com/lib/pq` | Go module proxy | Em uso no FB_APU04 em produção | — | Aprovado |
| `golang.org/x/crypto` | Go module proxy | Pacote oficial golang.org | — | Aprovado |
| `github.com/joho/godotenv` | Go module proxy | Em uso no FB_APU04 em produção | — | Aprovado |

**slopcheck:** ferramenta é específica para npm. Para Go, a verificação equivalente é via `proxy.golang.org` (registry oficial). `github.com/sijms/go-ora/v2@v2.9.0` confirmado via:
```bash
curl -s "https://proxy.golang.org/github.com/sijms/go-ora/v2/@latest"
# Retornou: {"Version":"v2.9.0","Time":"2025-06-09T21:19:12Z",...}
```

---

## Padrões de Arquitetura

### D-07 RESOLVIDO: Runner de Migração

**Mecanismo:** Custom, embutido em `main.go` (linhas 112–222). Nenhuma biblioteca externa. Usa apenas Go stdlib.

**Como funciona:**
1. Na inicialização do DB (`onDBConnected()`), cria/garante tabela `schema_migrations (filename VARCHAR(255) PRIMARY KEY, executed_at TIMESTAMPTZ)`.
2. `filepath.Glob("migrations/*.sql")` lista todos os arquivos `.sql` em ordem alfabética.
3. Para cada arquivo: verifica `SELECT EXISTS(... WHERE filename=$1)` — se já executado, pula.
4. Lê o arquivo com `os.ReadFile` e executa via `database.Exec(string(migration))`.
5. Registra o nome do arquivo com `INSERT INTO schema_migrations (filename) ON CONFLICT DO NOTHING`.
6. **Migrações falhas NÃO são registradas** — serão retentadas no próximo startup.

**Regra de ordenação:** Alfabética (ASCII). Logo, nomear com prefixo numérico zero-padded: `001_`, `002_`, `003_`, `004_`.

**Para o novo `main.go`:** Transplantar a função `onDBConnected()` com a lógica do runner (linhas 112–222 do FB_APU04/backend/main.go). Remover a chamada a `worker.StartWorker`, `worker.StartXMLWorker` e os goroutines de agendamento ERP/RFB.

```go
// Fonte: FB_APU04/backend/main.go linhas 112-222 (adaptado)
func onDBConnected(database *sql.DB) {
    migrationDir := "migrations"
    // ... lógica de glob + schema_migrations + execução sequencial
    // REMOVER: worker.StartWorker, worker.StartXMLWorker
    // REMOVER: goroutines de agendamento ERP/RFB
}
```

### Diagrama de Arquitetura do Sistema

```
Browser (React SPA)
    │
    │  HTTP + cookie httpOnly (refresh_token)
    ▼
┌─────────────────────────────────────┐
│  Backend Go (port 8085)             │
│                                     │
│  SecurityMiddleware (CORS/headers)  │
│         │                           │
│  ┌──────┴──────────────────────┐   │
│  │  AuthMiddleware (JWT check)  │   │
│  └──────┬──────────────────────┘   │
│         │                           │
│  ┌──────▼──────────────────────┐   │
│  │  Handlers                    │   │
│  │  ├── auth.go (login/refresh) │   │
│  │  ├── environment.go (CRUD)   │   │
│  │  ├── hierarchy.go (contexto) │   │
│  │  ├── managers.go (CRUD)      │   │
│  │  └── erp_bridge.go (config   │   │
│  │       + test-connection NEW) │   │
│  └──────┬──────────────────────┘   │
│         │                           │
│  ┌──────▼──────────────────────┐   │
│  │  Postgres (port 5432)        │   │
│  │  DB: fb_testesfc_db          │   │
│  │  (environments, users, etc.) │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌──────────────────────────────┐  │
│  │  Oracle (externo, só test)   │  │
│  │  via go-ora/v2 (test conn.)  │  │
│  └──────────────────────────────┘  │
└─────────────────────────────────────┘
```

### D-04 RESOLVIDO: Grafo de Dependências do Frontend

#### Arquivos a COPIAR integralmente (sem modificação)

| Arquivo/Diretório de Origem | Destino no FB_TESTESFC | Modificação |
|-----------------------------|------------------------|-------------|
| `contexts/AuthContext.tsx` | `contexts/AuthContext.tsx` | Nenhuma |
| `lib/utils.ts` | `lib/utils.ts` | Nenhuma |
| `lib/logger.ts` | `lib/logger.ts` | Nenhuma |
| `components/ui/` (todos os 44 arquivos) | `components/ui/` | Nenhuma |
| `index.css` | `index.css` | Nenhuma |
| `vite-env.d.ts` | `vite-env.d.ts` | Nenhuma |
| `main.tsx` | `main.tsx` | Nenhuma |
| `pages/Login.tsx` | `pages/Login.tsx` | Nenhuma |
| `pages/Register.tsx` | `pages/Register.tsx` | Nenhuma |
| `pages/ForgotPassword.tsx` | `pages/ForgotPassword.tsx` | Nenhuma |
| `pages/ResetPassword.tsx` | `pages/ResetPassword.tsx` | Nenhuma |
| `pages/GestaoAmbiente.tsx` | `pages/GestaoAmbiente.tsx` | Nenhuma |
| `pages/Managers.tsx` | `pages/Managers.tsx` | Nenhuma |
| `pages/AdminUsers.tsx` | `pages/AdminUsers.tsx` | Nenhuma |
| `pages/ERPBridgeConfig.tsx` | `pages/ERPBridgeConfig.tsx` | Nenhuma |

#### Arquivos a COPIAR e MODIFICAR

| Arquivo/Diretório | Modificação necessária |
|-------------------|----------------------|
| `vite.config.ts` | Alterar porta dev de 3003 para **3004** (evitar colisão); alterar fallback de `VITE_API_TARGET` de `localhost:8081` para `localhost:8085` |
| `package.json` | Alterar `"name"` para `"fb_testesfc-frontend"` |
| `components/AppRail.tsx` | Remover itens de nav do SPED/Reforma/Fronteira/Auditoria; manter apenas item de Configurações e logout; remover `AjudaChat` do render |
| `lib/navigation.ts` | Substituir todos os `modules` por configuração mínima para Phase 1 (sem tabs de módulos fora do escopo) |
| `pages/ERPBridgeCredenciais.tsx` | Adicionar botão "Testar Conexão Oracle" que chama `POST /api/erp-bridge/test-connection` (novo endpoint) |

#### Arquivos a NÃO copiar

| Arquivo | Motivo |
|---------|--------|
| `contexts/FilialContext.tsx` | Específico do seletor de filiais do SPED — fora do escopo |
| `components/FilialSelector.tsx` | Idem |
| `components/CompanySwitcher.tsx` | D-11: sem CompanySwitcher na UI |
| `components/AjudaChat.tsx` | Chat IA — fora do escopo |
| `components/AppSidebar.tsx` | Sidebar alternativo — não usado no AppRail layout |
| `components/InsightCard.tsx` | Dashboards IA — fora do escopo |
| `components/ParticipantList.tsx` | Jobs SPED — fora do escopo |
| `components/UploadProgress.tsx` | Upload SPED — fora do escopo |
| `components/Footer.tsx` | Não usado nas páginas copiadas |
| `hooks/useReformaParametros.ts` | Específico de Reforma Tributária |
| `lib/exportToExcel.ts` | Exportação Excel — fora do escopo |
| `lib/navigation.ts` (copiar, não reusar) | Reescrever conteúdo dos modules para Phase 1 |
| Todas as demais ~40 páginas | Fora do escopo desta fase |

#### Arquivo a CRIAR (não existe no FB_APU04)

| Arquivo | Descrição |
|---------|-----------|
| `App.tsx` | App root novo e enxuto — sem `FilialProvider`, sem CompanySwitcher, rotas apenas para as 9 páginas + login/register/forgot/reset |

### Estrutura recomendada do novo projeto

```
FB_TESTESFC/
├── backend/
│   ├── handlers/
│   │   ├── auth.go          # copiado do FB_APU04
│   │   ├── crypto.go        # copiado do FB_APU04
│   │   ├── environment.go   # copiado do FB_APU04
│   │   ├── hierarchy.go     # copiado do FB_APU04
│   │   ├── managers.go      # copiado do FB_APU04
│   │   ├── middleware.go    # copiado do FB_APU04 (CORS/security)
│   │   └── erp_bridge.go   # copiado + novo test-connection endpoint
│   ├── services/
│   │   ├── email.go         # copiado do FB_APU04
│   │   └── crypto.go        # copiado do FB_APU04
│   ├── migrations/
│   │   ├── 001_auth_hierarchy.sql   # novo consolidado
│   │   ├── 002_erp_bridge.sql       # novo consolidado
│   │   ├── 003_managers.sql         # novo consolidado
│   │   └── 004_seed_ferreira_costa.sql  # novo seed
│   ├── go.mod               # novo com module fb_testesfc
│   ├── go.sum
│   └── main.go              # novo e enxuto (D-02)
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── AppRail.tsx          # copiado + modificado
│   │   │   └── ui/                  # copiado integralmente (44 arquivos)
│   │   ├── contexts/
│   │   │   └── AuthContext.tsx      # copiado integralmente
│   │   ├── lib/
│   │   │   ├── navigation.ts        # copiado + conteúdo reescrito
│   │   │   ├── utils.ts             # copiado
│   │   │   └── logger.ts            # copiado
│   │   ├── pages/                   # 9 páginas copiadas + ERPBridgeCredenciais modificada
│   │   ├── App.tsx                  # NOVO
│   │   ├── main.tsx                 # copiado
│   │   └── index.css                # copiado
│   ├── package.json                 # copiado + nome alterado
│   └── vite.config.ts               # copiado + porta alterada
├── docker-compose.yml               # NOVO (portas distintas do FB_APU04)
├── .env.example                     # NOVO
└── CLAUDE.md
```

### D-13 RESOLVIDO: Mapa de Portas, Serviços e Volumes

#### Portas em uso pelo FB_APU04

| Serviço | Porta host | Porta container |
|---------|------------|-----------------|
| Backend Go | **8084** | 8084 |
| Frontend Vite dev | **3003** | — |
| Postgres | não exposta (interna) | 5432 |
| Redis | não exposta | 6379 |
| Prometheus | não exposta | 9090 |
| Grafana | não exposta | 3000 |

#### Proposta para FB_TESTESFC (sem colisão)

| Serviço | Porta host | Porta container | Nome do container |
|---------|------------|-----------------|------------------|
| Backend Go | **8085** | 8085 | `fb_testesfc-api` |
| Frontend Vite dev | **3004** | 3004 | `fb_testesfc-web` (dev) |
| Postgres | **5435** (exposta opcionalmente) | 5432 | `fb_testesfc-db` |
| Redis | **NÃO incluir** — não usado | — | — |
| Prometheus/Grafana | **NÃO incluir** | — | — |

**Banco de dados:**
- Nome do DB: `fb_testesfc_db`
- Volume: `postgres_data_testesfc`
- Network: `fb_testesfc_net` (bridge isolado)

**Variáveis de ambiente adicionadas vs FB_APU04:**
```
# .env.example do FB_TESTESFC (complementa o FB_APU04)
DB_NAME=fb_testesfc_db
PORT=8085
ENCRYPTION_KEY=GERE_COM_openssl_rand_hex_32   # nova — não estava no FB_APU04
APP_URL=http://localhost:3004                   # local dev
```

**Variáveis NOT presentes no FB_APU04 .env.example mas necessárias:**
- `ENCRYPTION_KEY`: usada por `handlers/crypto.go` para criptografar credenciais Oracle; o FB_APU04 não a documentou no `.env.example` mas o código a lê. Para dev local, o handler faz fallback para `JWT_SECRET`.

### D-05 / D-06 RESOLVIDO: Migrações Consolidadas

Escrever 4 arquivos SQL novos. Abaixo o conteúdo de referência (de qual migration de origem cada tabela/coluna vem):

#### `001_auth_hierarchy.sql`
Consolida: `013_create_environment_hierarchy.sql` + `015_create_auth_system.sql` + `017_add_owner_to_companies.sql` + `018_add_role_to_users.sql` + `025_add_indexes_auth.sql`

**NÃO incluir `cnpj` na tabela `companies`** — a migration 023 do FB_APU04 remove essa coluna; começar sem ela é mais limpo.

Tabelas criadas: `environments`, `enterprise_groups`, `companies`, `users`, `user_environments`, `verification_tokens` + índices de performance.

#### `002_erp_bridge.sql`
Cópia direta de `065_erp_bridge.sql`.

Tabelas: `erp_bridge_config`, `erp_bridge_runs`, `erp_bridge_run_items`, `erp_bridge_servidores`, `parceiros`.

#### `003_managers.sql`
Cópia direta de `046_create_managers_table.sql`.

Tabela: `managers`.

#### `004_seed_ferreira_costa.sql`
**Nova** — não existe no FB_APU04. Combina padrão de `016_seed_default_environment.sql` + `021_ensure_admin_user.sql` + `024_ensure_master_link.sql`, adaptando nomes para Ferreira Costa.

Seed idempotente (DO $$ ... END $$) que garante:
1. Ambiente `Ferreira Costa`
2. Grupo `Ferreira Costa`
3. Empresa `Ferreira Costa` (sem CNPJ — coluna não existe neste schema)
4. Usuário admin `claudio_bezerra@hotmail.com` com:
   - `password_hash = '$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC'` (hash de `123456`, copiado do `021_ensure_admin_user.sql`)
   - `role = 'admin'`, `is_verified = true`
5. Vínculo `user_environments` do admin com o ambiente Ferreira Costa (`role = 'admin'`)
6. Registro em `erp_bridge_config` para a empresa Ferreira Costa (row vazia, aguardando config)

**ATENÇÃO:** O hash bcrypt `$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC` é **retirado diretamente do `021_ensure_admin_user.sql` do FB_APU04** e corresponde à senha `123456`. [VERIFIED: código do FB_APU04]

### D-14 RESOLVIDO: Endpoint "Testar Conexão Oracle" (novo)

O FB_APU04 **não tem** este endpoint. Precisa ser criado em `handlers/erp_bridge.go`:

```go
// Fonte: novo endpoint a adicionar ao FB_TESTESFC
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
            http.Error(w, "empresa não encontrada", http.StatusUnauthorized)
            return
        }

        // Ler credenciais Oracle armazenadas
        var oracleDsn, oracleUsuario, oracleSenha sql.NullString
        db.QueryRow(`
            SELECT oracle_dsn, oracle_usuario, oracle_senha
            FROM erp_bridge_config WHERE company_id = $1
        `, companyID).Scan(&oracleDsn, &oracleUsuario, &oracleSenha)

        if !oracleDsn.Valid || oracleDsn.String == "" {
            json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "DSN Oracle não configurado"})
            return
        }

        // Abrir conexão via sijms/go-ora/v2
        // url: "oracle://usuario:senha@host:port/service_name"
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

**Dependência:** Adicionar `import _ "github.com/sijms/go-ora/v2"` em `erp_bridge.go` (registro do driver `"oracle"`).

### Rotas a registrar no novo `main.go` (D-02)

```go
// Auth (sem autenticação)
http.HandleFunc("/api/auth/register",       withDB(handlers.RegisterHandler))
http.HandleFunc("/api/auth/login",          withDB(handlers.LoginHandler))
http.HandleFunc("/api/auth/forgot-password",withDB(handlers.ForgotPasswordHandler))
http.HandleFunc("/api/auth/reset-password", withDB(handlers.ResetPasswordHandler))
http.HandleFunc("/api/auth/refresh",        withDB(handlers.RefreshHandler))
http.HandleFunc("/api/auth/logout",         withDB(handlers.LogoutHandler))

// Auth (com autenticação)
http.HandleFunc("/api/auth/me",                   withAuth(handlers.GetMeHandler, ""))
http.HandleFunc("/api/auth/change-password",      withAuth(handlers.ChangePasswordHandler, ""))
http.HandleFunc("/api/auth/preferred-company",    withAuth(handlers.SetPreferredCompanyHandler, ""))
http.HandleFunc("/api/user/hierarchy",            withAuth(handlers.GetUserHierarchyHandler, ""))
http.HandleFunc("/api/user/companies",            withAuth(handlers.GetUserCompaniesHandler, ""))

// Admin (role=admin)
http.HandleFunc("/api/admin/users",          withAuth(handlers.ListUsersHandler, "admin"))
http.HandleFunc("/api/admin/users/create",   withAuth(handlers.CreateUserHandler, "admin"))
http.HandleFunc("/api/admin/users/promote",  withAuth(handlers.PromoteUserHandler, "admin"))
http.HandleFunc("/api/admin/users/delete",   withAuth(handlers.DeleteUserHandler, "admin"))
http.HandleFunc("/api/admin/users/reassign", withAuth(handlers.ReassignUserHandler, "admin"))

// Hierarquia (environments/groups/companies) — authenticated
http.HandleFunc("/api/config/environments", withAuth(multiMethodEnvHandler, ""))
http.HandleFunc("/api/config/groups",       withAuth(multiMethodGroupHandler, ""))
http.HandleFunc("/api/config/companies",    withAuth(multiMethodCompanyHandler, ""))

// ERP Bridge (Phase 1: config + credenciais + test-connection)
http.HandleFunc("/api/erp-bridge/config",                withAuth(handlers.ERPBridgeConfigHandler, ""))
http.HandleFunc("/api/erp-bridge/config/generate-api-key",withAuth(handlers.ERPBridgeGenerateAPIKeyHandler, "admin"))
http.HandleFunc("/api/erp-bridge/test-connection",       withAuth(handlers.ERPBridgeTestConnectionHandler, "")) // NOVO
http.HandleFunc("/api/erp-bridge/credentials",           withDB(handlers.ERPBridgeCredentialsHandler)) // para daemon futuro

// Managers
http.HandleFunc("/api/managers",        withAuth(handlers.ListManagersHandler, ""))
http.HandleFunc("/api/managers/create", withAuth(handlers.CreateManagerHandler, ""))
http.HandleFunc("/api/managers/",       /* update/delete by id */ ...)

// Health
http.HandleFunc("/api/health", healthHandler)
```

**Middlewares obrigatórios:**
- `handlers.SecurityMiddleware(http.DefaultServeMux, handlers.GetAllowedOrigins())` — CORS + cabeçalhos de segurança (copiado de `handlers/middleware.go`)
- Sem `MetricsMiddleware` (Prometheus removido)

### Walking Skeleton (fatia mínima fim-a-fim)

O menor caminho que valida toda a pilha:

```
1. docker compose up
   └── backend: migrações 001-004 executam → Ferreira Costa + admin semeados
   └── frontend: build estático servido
   └── db: postgres pronto

2. Browser http://localhost:3004
   └── App.tsx → /login → Login.tsx
   └── POST /api/auth/login {email, password}
       └── auth.go → bcrypt check → gera access token (30min) + refresh cookie (7d)
       └── AuthContext.login() → token em memória + metadata em localStorage
   └── Redirect para / → AppLayout → AppRail → empresa "Ferreira Costa" no header

3. F5 (refresh do browser)
   └── AuthContext useEffect → POST /api/auth/refresh (cookie httpOnly enviado)
       └── auth.go → valida refresh token in-memory → novo access token
   └── GET /api/auth/me → dados do usuário atualizados
   └── Sessão restaurada sem novo login ← AUTH-02 validado

4. /config/usuarios → AdminUsers.tsx
   └── GET /api/admin/users (Bearer token)
       └── AuthMiddleware → role=admin ✓ → lista usuários

5. /config/erp-bridge → ERPBridgeCredenciais.tsx
   └── Salvar DSN/credenciais Oracle → PATCH /api/erp-bridge/config
   └── "Testar Conexão" → POST /api/erp-bridge/test-connection
       └── go-ora/v2 → oracle.Ping() → {ok: true/false}
```

---

## Não Implementar do Zero

| Problema | Não construir | Usar de | Motivo |
|----------|--------------|---------|--------|
| JWT geração/validação | Custom crypto | `github.com/golang-jwt/jwt/v5` (já em uso) | Clock skew, replay, alg confusion attacks |
| Hash de senha | Custom hash | bcrypt via `golang.org/x/crypto/bcrypt` (já em uso) | Timing attacks, salt management |
| Criptografia de credenciais | Custom cipher | AES-256-GCM em `handlers/crypto.go` (já implementado) | Padding oracle, nonce reutilizado |
| Conexão Oracle | Custom TCP | `github.com/sijms/go-ora/v2` | Protocol TNS complexo, negotiation, charsets |
| Migração de schema | Wrapper custom | Runner já em `main.go` do FB_APU04 | Idempotência, retry, tracking já resolvidos |
| CORS + security headers | Custom handler | `handlers/middleware.go` (SecurityMiddleware) | CSP, HSTS, X-Frame-Options — difícil de acertar |
| UI components | Custom CSS | shadcn/ui em `components/ui/` (já copiado) | Acessibilidade, animações, variants |

---

## Armadilhas Comuns

### Pitfall 1: Colisão de nomes de migration com prefixo duplicado

**O que dá errado:** No FB_APU04 existem dois arquivos com prefixo `021`: `021_create_mv_mercadorias.sql` e `021_ensure_admin_user.sql`. A ordem de execução fica não-determinística entre sistemas de arquivo. O `021_create_mv_mercadorias.sql` referencia uma view materializada de dados SPED que não existirá no FB_TESTESFC.

**Por que acontece:** O FB_APU04 teve inserções urgentes de migrações com o mesmo número.

**Como evitar:** Usar prefixos zero-padded únicos nas migrações consolidadas do FB_TESTESFC (`001`, `002`, `003`, `004`). Confirma D-05: nunca copiar as 149 migrações individuais.

**Sinal de alerta:** Erro `relation "mv_mercadorias_agregada" does not exist` no startup se qualquer migration do FB_APU04 for copiada com a vista materializada.

### Pitfall 2: Import path `fb_apu04` nos arquivos copiados

**O que dá errado:** Os handlers copiados têm `import "fb_apu04/handlers"`, `import "fb_apu04/services"`. Após copiar, o Go não encontra os imports e a build falha.

**Por que acontece:** module path no go.mod do FB_APU04 é `module fb_apu04`.

**Como evitar:** Após copiar cada arquivo Go, executar find/replace de `fb_apu04/` → `fb_testesfc/` em todos os imports. Verificar com `go build ./...` antes de prosseguir.

**Sinal de alerta:** `cannot find module providing package fb_apu04/handlers` na build.

### Pitfall 3: Dockerfile com `-mod=vendor` sem diretório vendor

**O que dá errado:** O Dockerfile do FB_APU04 usa `go build -mod=vendor`. O FB_TESTESFC segue D-03 (sem `vendor/`), portanto o mesmo Dockerfile falhará com `go: inconsistent vendoring`.

**Por que acontece:** FB_APU04 usa vendor para builds reproduzíveis em CI sem internet.

**Como evitar:** No Dockerfile do FB_TESTESFC, usar `go build` (sem `-mod=vendor`) e adicionar `go mod download` antes do build. O stage builder precisa de acesso à internet no momento do build.

**Sinal de alerta:** `fatal: go: inconsistent vendoring` durante `docker compose build`.

### Pitfall 4: `ENCRYPTION_KEY` não documentada causa fallback silencioso

**O que dá errado:** O `handlers/crypto.go` lê `ENCRYPTION_KEY`; se ausente, faz fallback para `JWT_SECRET`. Se `JWT_SECRET` também não estiver setado, usa um valor hardcoded. Credenciais Oracle são salvas encriptadas com uma chave que pode mudar entre restarts se as variáveis não estiverem configuradas.

**Por que acontece:** O `.env.example` do FB_APU04 não documenta `ENCRYPTION_KEY`.

**Como evitar:** Incluir `ENCRYPTION_KEY=GERE_COM_openssl_rand_hex_32` no `.env.example` do FB_TESTESFC. No `main.go` novo, adicionar aviso de inicialização se `ENCRYPTION_KEY` estiver ausente.

**Sinal de alerta:** Credenciais Oracle salvas corretamente mas falham a descriptografar após restart do container (chave diferente).

### Pitfall 5: Refresh token armazenado in-memory (`sync.Map`) — perdido no restart

**O que dá errado:** O `refreshTokenStore` em `auth.go` é um `sync.Map` em memória. Ao reiniciar o container backend, todos os tokens de refresh são perdidos. Todos os usuários são deslogados no próximo refresh do browser.

**Por que acontece:** Decisão de design do FB_APU04 — simples e seguro para SPA de uso interno.

**Como evitar:** É comportamento esperado e aceitável para uma ferramenta interna local. Documentar no README. Não tentar persistir no banco sem necessidade (cria surface de ataque).

**Sinal de alerta:** Todos os usuários redirecionados para `/login` após `docker compose restart` — esperado, não é bug.

### Pitfall 6: `worker` package não copiado mas chamado no main.go

**O que dá errado:** Ao copiar a função `onDBConnected()` do FB_APU04, as chamadas `worker.StartWorker(database)` e `worker.StartXMLWorker(database)` ficam referenciando um pacote não copiado.

**Por que acontece:** `worker` é um pacote interno do FB_APU04 que processa SPED em background — fora do escopo.

**Como evitar:** Ao transplantar `onDBConnected()`, remover as 3 linhas que chamam `worker.*` e os goroutines de agendamento ERP (linhas 224-298 do FB_APU04 main.go). Verificar com `go build ./...` logo após.

### Pitfall 7: `CompanySwitcher` no AppHeader causa chamadas de API desnecessárias

**O que dá errado:** Se `CompanySwitcher` for copiado junto com `AppRail`/`AppHeader`, ele faz `GET /api/user/companies` em cada render e oferece troca de empresa na UI. Conflita com D-11.

**Por que acontece:** `App.tsx` do FB_APU04 inclui `<CompanySwitcher compact />` no header.

**Como evitar:** No novo `App.tsx`, não importar nem renderizar `CompanySwitcher`. No `AppHeader` simplificado, exibir o nome da empresa como `<span>` estático (usando `company` do `useAuth()`).

---

## Exemplos de Código

### Runner de migração (transplantado de main.go do FB_APU04)

```go
// Fonte: FB_APU04/backend/main.go linhas 112-222
// Função onDBConnected() — transplantar para o novo main.go,
// removendo worker.Start* e goroutines de agendamento ERP/RFB.

func onDBConnected() {
    database := getDB()
    migrationDir := "migrations"
    if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
        if _, err := os.Stat("backend/migrations"); err == nil {
            migrationDir = "backend/migrations"
        }
    }

    files, err := filepath.Glob(filepath.Join(migrationDir, "*.sql"))
    if err != nil { /* log e retornar */ }

    // Garante tabela schema_migrations
    // ... (lógica de criação/verificação das colunas)

    for _, file := range files {
        baseName := filepath.Base(file)
        var alreadyExecuted bool
        _ = database.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)", baseName).Scan(&alreadyExecuted)
        if alreadyExecuted { continue }

        migration, _ := os.ReadFile(file)
        _, err = database.Exec(string(migration))
        if err != nil {
            log.Printf("ERROR: Migration %s failed: %v", file, err)
            continue // NÃO registra — retenta no próximo startup
        }
        database.Exec("INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING", baseName)
    }
    // REMOVER: worker.StartWorker(database) etc.
}
```

### Seed de Ferreira Costa (004_seed_ferreira_costa.sql)

```sql
-- Fonte: adaptado de FB_APU04/backend/migrations/016, 021, 024
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
    SELECT id INTO v_group_id FROM enterprise_groups WHERE name = 'Ferreira Costa' AND environment_id = v_env_id;
    IF v_group_id IS NULL THEN
        INSERT INTO enterprise_groups (environment_id, name)
        VALUES (v_env_id, 'Ferreira Costa')
        RETURNING id INTO v_group_id;
    END IF;

    -- 3. Empresa Ferreira Costa
    SELECT id INTO v_company_id FROM companies WHERE name = 'Ferreira Costa' AND group_id = v_group_id;
    IF v_company_id IS NULL THEN
        INSERT INTO companies (group_id, name, trade_name)
        VALUES (v_group_id, 'Ferreira Costa', 'Ferreira Costa')
        RETURNING id INTO v_company_id;
    END IF;

    -- 4. Admin user
    IF EXISTS (SELECT 1 FROM users WHERE email = 'claudio_bezerra@hotmail.com') THEN
        UPDATE users SET
            password_hash = '$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC', -- 123456
            role = 'admin', is_verified = true, full_name = 'Claudio Bezerra (Admin)'
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

    -- 6. Row de config ERP Bridge para a empresa (credenciais vazias, a preencher pela UI)
    INSERT INTO erp_bridge_config (company_id)
    VALUES (v_company_id)
    ON CONFLICT DO NOTHING;
END $$;
```

### Novo App.tsx (esboço para planner)

```tsx
// Novo — não copiado do FB_APU04
// Sem FilialProvider, sem CompanySwitcher, rotas apenas Phase 1
function AppLayout() {
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppRail />   {/* versão simplificada */}
      <div className="flex flex-col flex-1 min-w-0">
        <header className="flex items-center justify-between h-12 border-b bg-white px-4 shrink-0">
          <span className="text-sm font-semibold">FB_TESTESFC — Validador Fiscal</span>
          <CompanyDisplay />  {/* span estático com nome da empresa — sem switcher */}
        </header>
        <main className="flex-1 overflow-auto p-4">
          <Routes>
            <Route path="/config/ambiente"    element={<ProtectedRoute><GestaoAmbiente /></ProtectedRoute>} />
            <Route path="/config/gestores"    element={<ProtectedRoute><Managers /></ProtectedRoute>} />
            <Route path="/config/usuarios"    element={<AdminRoute><AdminUsers /></AdminRoute>} />
            <Route path="/importacoes/erp-bridge" element={<AdminRoute><ERPBridgeConfig /></AdminRoute>} />
            <Route path="/config/erp-bridge"  element={<AdminRoute><ERPBridgeCredenciais /></AdminRoute>} />
            <Route path="/"                   element={<Navigate to="/config/erp-bridge" replace />} />
          </Routes>
        </main>
      </div>
      <Toaster />
    </div>
  )
}
```

---

## Estado da Arte

| Abordagem antiga | Abordagem atual (FB_APU04) | Relevância para FB_TESTESFC |
|------------------|---------------------------|------------------------------|
| Access token em localStorage | Token em memória (React state) + refresh cookie httpOnly | Copiar como está — mais seguro |
| Bcrypt cost factor 10 | Bcrypt cost 14 (mais lento, mais seguro) | Copiar como está — seed usa cost 14 |
| Driver Oracle CGO (godror) | Nenhum driver Oracle no backend FB_APU04 | NOVO: `sijms/go-ora/v2` (puro-Go, CGO_ENABLED=0) |
| Migrations com ORM | Migrations SQL puro + runner custom | Manter o padrão — copiar o runner |

---

## Log de Suposições

| # | Afirmação | Seção | Risco se errada |
|---|-----------|-------|-----------------|
| A1 | `github.com/sijms/go-ora/v2` suporta DSN no formato `oracle://user:pass@host:port/service` | Standard Stack | Sintaxe do DSN pode ser diferente; resolver na implementação com a doc oficial do pacote |
| A2 | A porta 5435 do Postgres não está em uso na máquina do desenvolvedor | Mapa de Portas | Trocar para outra porta livre (5436, 5437, etc.) |
| A3 | O Postgres do FB_TESTESFC não precisa ser exposto na porta host para o desenvolvimento local | Mapa de Portas | Se houver necessidade de acesso via DBeaver/psql, adicionar `ports: "5435:5432"` |

---

## Perguntas em Aberto (RESOLVED)

> Ambas as perguntas abaixo foram resolvidas para o escopo da Fase 1. Mantidas aqui com a resolução inline para rastreabilidade. Nenhuma bloqueia o planejamento ou a execução.

1. **DSN Oracle para o teste de conexão**
   - O que sabemos: o campo `oracle_dsn` armazena o DSN/connection string. O driver `sijms/go-ora/v2` aceita URLs no formato `oracle://user:pass@host:port/service_name` ou strings no formato Oracle Easy Connect.
   - O que está incerto: qual formato exato a Ferreira Costa usa para o DSN da instância Oracle que hospeda FCCORP_BKP.
   - Recomendação: deixar o campo `oracle_dsn` livre para o usuário inserir; documentar ambos os formatos no tooltip da UI. Implementar fallback: tentar como URL Oracle; se falhar, tentar como Easy Connect.
   - **RESOLVIDO:** Para a Fase 1 (test-connection apenas), a abordagem `oracle://user:pass@host:port/service_name` com fallback Easy Connect é suficiente — o formato exato do cliente Ferreira Costa é irrelevante nesta fase (a consulta real a prod/PRODB é Fase 2). O Plan 04 Task 1 já marca a sintaxe DSN como `[ASSUMED A1]` e instrui o executor a VERIFICAR o formato exato na doc do `go-ora/v2` durante a implementação. Sem bloqueio.

2. **Versionamento do frontend no container**
   - O que sabemos: o FB_APU04 serve o frontend via Nginx no container `web` (porta 80, sem expor diretamente — usa Traefik para prod). Para dev local, usa `vite dev` na porta 3003.
   - O que está incerto: se o desenvolvedor quer usar `docker compose up` para tudo (incluindo frontend via Nginx) ou só para api+db, com Vite dev separado.
   - Recomendação: para Phase 1, incluir um serviço `web` no compose baseado na imagem Nginx (mesmo padrão do FB_APU04) mas também documentar que `cd frontend && npm run dev` funciona com proxy para 8085 para ciclos de desenvolvimento mais rápidos.
   - **RESOLVIDO:** O Plan 02 Task 2 especifica explicitamente o serviço `web` Nginx no `docker-compose.yml` (porta Vite dev 3004 documentada como alternativa para ciclos rápidos). Decisão tomada conforme a recomendação. Sem bloqueio.

---

## Disponibilidade do Ambiente

| Dependência | Requerida por | Disponível | Versão | Fallback |
|-------------|--------------|------------|--------|---------|
| Docker + Docker Compose | `docker compose up` | A verificar no dev | — | — |
| Go 1.24 | Backend build | A verificar | — | — |
| Node.js ≥ 18 | Frontend build | A verificar | — | — |
| Postgres 15 (Docker image) | DB do app | Via docker compose | 15-alpine | — |
| Instância Oracle (ERP) | Teste de conexão (D-14) | Externo — não controlado | — | Feature "test-connection" retorna erro descritivo se inacessível |

**Dependências sem fallback que podem bloquear:**
- Docker indisponível: bloqueia `docker compose up`; planner deve incluir verificação prévia no Wave 0.

**Dependências com fallback:**
- Oracle inacessível: test-connection retorna `{ok: false, error: "dial tcp..."}` — comportamento esperado para o dev que ainda não tem acesso configurado.

---

## Domínio de Segurança

> `security_enforcement: true`, `security_asvs_level: 1`

### Categorias ASVS Aplicáveis

| Categoria ASVS | Aplica | Controle Padrão |
|----------------|--------|-----------------|
| V2 Autenticação | Sim | bcrypt cost=14 em `handlers/auth.go`; JWT HS256; sem senhas em logs |
| V3 Gestão de Sessão | Sim | httpOnly cookie `refresh_token` com `SameSite=Strict`, `MaxAge=7d`; access token 30min em memória |
| V4 Controle de Acesso | Sim | `AuthMiddleware(handler, role)` — role="" permite autenticado; role="admin" exige role=admin no JWT |
| V5 Validação de Entrada | Sim (parcial) | Handlers fazem validação básica; nenhuma biblioteca externa de validação nos módulos copiados |
| V6 Criptografia | Sim | AES-256-GCM para credenciais Oracle em `handlers/crypto.go`; bcrypt para senhas; nunca hand-roll |

### Padrões de Ameaça Conhecidos

| Padrão | STRIDE | Mitigação Padrão |
|--------|--------|-----------------|
| Exfiltração de credenciais Oracle via API | Information Disclosure | Campos `oracle_senha`, `fbtax_password` NUNCA retornados ao frontend — apenas `*_set: bool`; descriptografia apenas server-side |
| Sessão roubada via XSS | Elevation of Privilege | Access token em memória React (não localStorage); refresh token httpOnly (não acessível por JS) |
| CSRF no endpoint de login/refresh | Spoofing | `SameSite=Strict` no cookie de refresh; tokens Bearer em memória não enviados automaticamente |
| Bypass de autenticação por JWT alg confusion | Elevation of Privilege | `jwt/v5` com método explícito `jwt.SigningMethodHS256`; `getJWTSecret()` valida na startup |
| SQL injection | Tampering | Todos os queries usam `$1`, `$2` (Postgres parameterized); `lib/pq` |
| CORS bypass | Elevation of Privilege | `handlers.SecurityMiddleware` com whitelist de origens; CORS headers via `secureResponseWriter` |
| Senha padrão admin vazada em logs | Information Disclosure | Seed SQL nunca loga a senha; hash bcrypt nos logs apenas em modo debug |

### Risco específico desta fase

`ENCRYPTION_KEY` ausente faz o sistema usar `JWT_SECRET` como chave AES. Se `JWT_SECRET` for comprometido, as credenciais Oracle criptografadas no banco também serão comprometidas. Para dev local (ferramenta interna), é aceitável; documentar claramente.

---

## Fontes

### Primárias (confiança ALTA — inspeção direta do código)
- `FB_APU04/backend/main.go` — runner de migração (linhas 112-222), roteamento, padrão `withDB`/`withAuth`
- `FB_APU04/backend/handlers/auth.go` — AuthMiddleware, JWT, refresh cookie, bcrypt
- `FB_APU04/backend/handlers/crypto.go` — EncryptField, DecryptFieldWithFallback
- `FB_APU04/backend/handlers/environment.go` — CRUD hierarquia, GetEffectiveCompanyID
- `FB_APU04/backend/handlers/erp_bridge.go` — config/credenciais Oracle (793 linhas inspecionadas)
- `FB_APU04/backend/handlers/managers.go` — CRUD managers
- `FB_APU04/backend/handlers/middleware.go` — SecurityMiddleware (CORS)
- `FB_APU04/backend/services/email.go` — SMTP para forgot/reset
- `FB_APU04/backend/services/crypto.go` — DecryptFieldWithFallback para serviços
- `FB_APU04/backend/go.mod` — dependências Go confirmadas
- `FB_APU04/backend/Dockerfile` — padrão de build (CGO_ENABLED=0, vendor)
- `FB_APU04/docker-compose.yml` — serviços, portas, volumes do projeto irmão
- `FB_APU04/.env.example` — variáveis de ambiente
- `FB_APU04/backend/migrations/013, 015, 016, 017, 018, 021 (ensure_admin), 023, 024, 025, 046, 065` — schema de referência para consolidação
- `FB_APU04/frontend/src/App.tsx` — roteamento do App original
- `FB_APU04/frontend/src/contexts/AuthContext.tsx` — lógica de sessão + interceptor de fetch
- `FB_APU04/frontend/src/pages/Login.tsx`, `ERPBridgeConfig.tsx`, `ERPBridgeCredenciais.tsx`, `AdminUsers.tsx` — imports e API calls verificados
- `FB_APU04/frontend/vite.config.ts` — porta 3003, proxy para backend

### Secundárias (confiança ALTA — registry oficial)
- `proxy.golang.org` — confirmação de `github.com/sijms/go-ora/v2@v2.9.0` (2025-06-09) [CITED: proxy.golang.org]

---

## Metadados

**Breakdown de confiança:**
- Stack padrão: HIGH — confirmada via inspeção direta dos arquivos do FB_APU04
- Arquitetura/padrões: HIGH — código fonte inspecionado linha a linha
- Armadilhas: HIGH — derivadas de observações diretas do código (migration duplicada 021, vendor, etc.)
- Driver Oracle: MEDIUM — `sijms/go-ora/v2` confirmado via proxy oficial; sintaxe do DSN marcada como [ASSUMED A1]

**Data da pesquisa:** 2026-06-30
**Válido até:** 2026-07-30 (projeto em início, FB_APU04 estável)
