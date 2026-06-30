# Phase 1: Foundation & Inherited Stack - Context

**Gathered:** 2026-06-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Aplicação roda localmente com os módulos herdados do FB_APU04 funcionando de ponta a ponta: `docker compose up` sobe backend Go + frontend React + Postgres com migrações limpas em base zerada; usuário faz login (JWT) e permanece autenticado após refresh; fluxo forgot/reset funciona; admin gerencia usuários; empresa Ferreira Costa pré-configurada com contexto resolvido nas requisições; infra de conexão Oracle (ERP_BRIDGE) presente e testável.

Cobre: FND-01, FND-02, FND-03, AUTH-01..04, TEN-01..03.

**Fora desta fase:** importação/parse de XML e persistência de NFe (Fase 2), lookup de grupo fiscal em prod/PRODB (Fase 2), execução do pacote fiscal no FCCORP_BKP (Fase 2), tela de comparação (Fase 3), multi-tenant completo/CompanySwitcher (v2).

</domain>

<decisions>
## Implementation Decisions

### Estratégia de cópia seletiva
- **D-01:** Trazer código do FB_APU04 por **cherry-pick por dependência** — copiar arquivo a arquivo apenas o necessário (auth, hierarquia, erp_bridge, xml saída e suas dependências), resolvendo imports sob demanda. Não copiar tudo para podar depois; não reescrever do zero. (Confirma a decisão "cópia seletiva" do PROJECT.md.)
- **D-02:** **`main.go` novo e enxuto**, registrando apenas as rotas dos módulos copiados. Não copiar o `main.go` de 1441 linhas do FB_APU04 com rotas comentadas.
- **D-03:** Dependências Go via **`go.mod` limpo + `go mod tidy`** — partir do go.mod do FB_APU04, remover deps de módulos não usados, deixar o tidy resolver. Manter driver Oracle e Postgres. Não copiar `vendor/` inteiro.
- **D-04:** Frontend = **trazer o shell compartilhado** (layout, router, AuthContext, api client, componentes UI) + registrar no roteador **apenas as páginas no escopo** (Login, Register, Forgot/Reset, GestaoAmbiente, Managers, AdminUsers, ERPBridgeConfig/Credenciais). Demais ~40 páginas ficam de fora.

### Migrações / Schema Postgres
- **D-05:** **Schema inicial consolidado** — escrever 1-3 migrações novas e enxutas (auth, hierarquia/empresa, users/roles) em vez de trazer as 149 migrações do FB_APU04. Base zerada sobe limpa (FND-03).
- **D-06:** Schema da Fase 1 inclui **apenas auth + tenancy**. As tabelas de persistência de NFe saída (cabeçalho/itens/impostos) entram na migração da Fase 2, junto do importador.
- **D-07:** Usar o **mesmo mecanismo/runner de migração** do FB_APU04 (arquivos .sql numerados aplicados na subida). Lib exata a confirmar pelo researcher.

### Multi-tenant (simplificação v1)
- **D-08:** **Manter o modelo hierárquico** (ambiente→grupo→empresa→usuário) e os handlers herdados, **semeando apenas Ferreira Costa + admin**. Não reduzir o schema nem hard-code. Mantém o multi-tenant (MTN-01) trivial de reativar na v2.
- **D-09:** Ambiente/empresa Ferreira Costa + admin criados via **migração de seed SQL idempotente** (padrão 016/021 do FB_APU04), prontos no `docker compose up` em base zerada.
- **D-10:** Admin inicial fixo: **`claudio_bezerra@hotmail.com`**, senha padrão **`123456`** (hash gerado na seed). Demais usuários são cadastrados a partir dele na tela de gestão herdada. É um default de desenvolvimento para ferramenta interna somente-leitura, trocável depois pela tela de gestão.
- **D-11:** Contexto de empresa (TEN-03) resolvido pelo **mecanismo herdado do FB_APU04** (claim JWT / middleware), com Ferreira Costa como default único. **Sem CompanySwitcher** na UI.

### Identidade / nomes / infra local
- **D-12:** **Renomear para `fb_testesfc`** — module path do go.mod, imports e nome do app. Find/replace cuidadoso nos arquivos copiados.
- **D-13:** Docker-compose com **serviços, portas e banco Postgres próprios** (ex.: db `fb_testesfc`, portas distintas) para rodar **em paralelo ao FB_APU04** na mesma máquina sem conflito de porta/volume.
- **D-14:** **ERP_BRIDGE na Fase 1 = infra de conexão Oracle + telas ERPBridgeConfig/Credenciais + ação "testar conexão"**. NÃO consultar prod/PRODB ainda (isso é Fase 2). Cumpre "conexão operacional" sem invadir a próxima fase.

### Claude's Discretion
- Lib/abordagem exata do runner de migração (D-07) — researcher inspeciona o FB_APU04 e confirma.
- Detalhe fino do corte de dependências do shell do frontend (D-04) — researcher mapeia o grafo real de componentes se necessário.
- Mapeamento concreto de portas/serviços/volumes do compose (D-13).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Projeto base (fonte da cópia seletiva)
- `/home/claudiobezerra/projetos/FB_APU04` — repositório irmão maduro (repo `github.com/ClaudioSBezerra/FB_APU04`); fonte de TODO o código herdado nesta fase.
- `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/auth.go` — autenticação JWT, login.
- `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/auth_middleware_test.go` — middleware de auth (referência de proteção de rotas, AUTH-04).
- `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/environment.go` — ambiente.
- `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/hierarchy.go` — hierarquia ambiente/grupo/empresa/usuário.
- `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/erp_bridge.go` — conexão/infra ERP_BRIDGE Oracle.
- `/home/claudiobezerra/projetos/FB_APU04/backend/services/email.go` — serviço de e-mail (dependência do forgot/reset, AUTH-03).
- `/home/claudiobezerra/projetos/FB_APU04/backend/services/crypto.go` — hashing de senha (relevante para a seed do admin, D-10).
- `/home/claudiobezerra/projetos/FB_APU04/backend/main.go` — referência das rotas a registrar no main.go novo (D-02).
- `/home/claudiobezerra/projetos/FB_APU04/backend/migrations/` — referência das migrações relevantes a consolidar (013 hierarchy, 015 auth, 016 seed environment, 017/018 owner/role, 021 ensure admin, 024/025).

### Frontend herdado
- `/home/claudiobezerra/projetos/FB_APU04/frontend/src/pages/` — páginas no escopo: `Login.tsx`, `Register.tsx`, `ForgotPassword.tsx`, `ResetPassword.tsx`, `GestaoAmbiente.tsx`, `Managers.tsx`, `AdminUsers.tsx`, `ERPBridgeConfig.tsx`, `ERPBridgeCredenciais.tsx`.

### Infra
- `/home/claudiobezerra/projetos/FB_APU04/docker-compose.yml` — base do compose (D-13).
- `/home/claudiobezerra/projetos/FB_APU04/backend/Dockerfile` e `/home/claudiobezerra/projetos/FB_APU04/.env.example` — referência de build e variáveis.

### Planejamento do projeto
- `.planning/PROJECT.md` — visão, decisões-chave, módulos a reaproveitar.
- `.planning/REQUIREMENTS.md` — FND/AUTH/TEN-* desta fase.
- `.planning/ROADMAP.md` — goal e critérios de sucesso da Fase 1.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Handlers de auth/hierarquia/erp_bridge do FB_APU04 (cherry-pick) — base direta de FND-02, AUTH-*, TEN-*.
- Páginas React do shell + telas listadas — base de toda a UI da fase.
- Migrações 013/015/016/017/018/021 do FB_APU04 — referência para consolidar o schema enxuto.
- `services/email.go` + `services/crypto.go` — necessários para forgot/reset (AUTH-03) e hash do admin (D-10).

### Established Patterns
- FB_APU04: backend Go com handlers + services + migrations .sql numeradas; frontend React/TS/Vite/Tailwind com AuthContext + api client; Docker Compose para dev. Manter esses padrões reduz fricção da cópia.
- Modelo de tenancy hierárquico com seed de ambiente/empresa default (016) e ensure-admin (021) — reaproveitado com seed única (D-08/D-09).

### Integration Points
- Conexão Postgres do app (novo banco `fb_testesfc`).
- Infra de conexão Oracle ERP_BRIDGE (somente "testar conexão" nesta fase; lookup real na Fase 2).
- JWT/middleware protegendo as rotas da API copiadas.

</code_context>

<specifics>
## Specific Ideas

- Admin padrão fixo `claudio_bezerra@hotmail.com` / `123456`, a partir do qual os demais usuários são cadastrados na tela de gestão (D-10).
- FB_TESTESFC deve poder rodar simultaneamente ao FB_APU04 na mesma máquina (portas/DB próprios, D-13).
- Projeto deve nascer enxuto: nada de módulos fiscais fora de escopo (SPED/EFD/CFOP/MVs/conciliação/painéis).

</specifics>

<deferred>
## Deferred Ideas

- **Tabelas de NFe saída (cabeçalho/itens/impostos)** — entram na migração da Fase 2 com o importador, não na Fase 1 (D-06).
- **Lookup de grupo fiscal em prod/PRODB** — Fase 2 (D-14 limita ERP_BRIDGE a infra + teste de conexão na Fase 1).
- **CompanySwitcher / multi-tenant completo (MTN-01)** — v2; modelo mantido pronto para reativar (D-08).
- **Gestão de credenciais do admin via .env / troca obrigatória no 1º login** — considerado, mas v1 usa default fixo de dev (D-10); pode ser endurecido depois.

None pendente além do acima — discussão permaneceu no escopo da fase.

</deferred>

---

*Phase: 1-Foundation & Inherited Stack*
*Context gathered: 2026-06-30*
