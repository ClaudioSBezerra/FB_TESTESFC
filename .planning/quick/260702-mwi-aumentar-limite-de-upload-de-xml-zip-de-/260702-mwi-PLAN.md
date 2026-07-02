---
phase: quick-260702-mwi
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - frontend/nginx.conf
  - backend/handlers/xml_upload.go
  - frontend/src/pages/ImportarXMLsSaida.tsx
autonomous: true
requirements: [UPLOAD-LIMIT-5GB]
must_haves:
  truths:
    - "Uploads de XML/ZIP até 5GB são aceitos por todas as camadas (nginx, backend, frontend)"
    - "Nenhuma referência residual a limite de 2GB de upload permanece nos 3 arquivos"
    - "Proteção anti-ZIP-bomb mantém o fator 4x (5GB upload → 20GB descomprimido)"
  artifacts:
    - path: "frontend/nginx.conf"
      provides: "client_max_body_size 5G — camada nginx não bloqueia mais uploads grandes"
      contains: "client_max_body_size 5G"
    - path: "backend/handlers/xml_upload.go"
      provides: "MaxUploadFileBytes 5GB + MaxUncompressedBytes 20GB + mensagens de erro atualizadas"
      contains: "5 * 1024 * 1024 * 1024"
    - path: "frontend/src/pages/ImportarXMLsSaida.tsx"
      provides: "maxSize 5GB + textos/toasts de UI atualizados para 5GB"
      contains: "5 * 1024 * 1024 * 1024"
  key_links:
    - from: "frontend/nginx.conf"
      to: "backend/handlers/xml_upload.go"
      via: "client_max_body_size (5G) >= MaxUploadFileBytes (5GB)"
      pattern: "client_max_body_size 5G"
---

<objective>
Aumentar o limite de upload de XML/ZIP de 2GB para 5GB em todas as três camadas (nginx, backend Go, frontend React) e corrigir o bug onde `nginx.conf` tinha `client_max_body_size 512M` — MENOR que o limite de 2GB do app, fazendo uploads acima de 512MB falharem prematuramente na camada errada.

Purpose: O core value do produto é importar XMLs completos de vendas; arquivos grandes (lotes de NFe) precisam passar. O limite do nginx estava efetivamente estrangulando o upload muito antes do backend.
Output: Três arquivos com limites de tamanho consistentes em 5GB, mantendo o fator de segurança anti-ZIP-bomb de 4x (20GB descomprimido).
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@./CLAUDE.md
@frontend/nginx.conf
@backend/handlers/xml_upload.go
@frontend/src/pages/ImportarXMLsSaida.tsx
</context>

<tasks>

<task type="auto">
  <name>Task 1: Corrigir client_max_body_size no nginx (bug + aumento para 5G)</name>
  <files>frontend/nginx.conf</files>
  <action>Na linha 7, trocar `client_max_body_size 512M;` por `client_max_body_size 5G;`. Este é o mesmo arquivo servido em dev e produção (referenciado por docker-compose.prod.yml). O valor 512M era MENOR que o limite de 2GB do backend — bug real que fazia uploads acima de 512MB falharem no nginx. Usar sintaxe nginx com G maiúsculo (`5G`). Não alterar mais nada no arquivo (proxy timeouts, buffering e demais diretivas permanecem intactos).</action>
  <verify>
    <automated>grep -q "client_max_body_size 5G;" frontend/nginx.conf && ! grep -q "512M" frontend/nginx.conf && echo OK</automated>
  </verify>
  <done>nginx.conf tem `client_max_body_size 5G;` e nenhuma ocorrência de `512M` resta.</done>
</task>

<task type="auto">
  <name>Task 2: Aumentar limites e atualizar mensagens no backend Go</name>
  <files>backend/handlers/xml_upload.go</files>
  <action>Atualizar as constantes e mensagens de erro relacionadas ao limite de upload, mantendo o fator de segurança anti-ZIP-bomb 4x:
  - Linha 24: `MaxUploadFileBytes = 2 * 1024 * 1024 * 1024 // 2 GB ...` → `5 * 1024 * 1024 * 1024 // 5 GB — tamanho máximo do .zip/.xml enviado`.
  - Linha 25: `MaxUncompressedBytes = 8 * 1024 * 1024 * 1024 // 8 GB ...` → `20 * 1024 * 1024 * 1024 // 20 GB — proteção anti-ZIP bomb (total descomprimido)`. (5GB × fator 4 = 20GB — MESMA proporção que já existia com 2GB→8GB; NÃO deixar o fator de segurança cair.)
  - Linha 78: mensagem `"conteúdo do ZIP excede limite de 8GB após descompressão"` → substituir `8GB` por `20GB`.
  - Linhas 211, 217, 256: mensagens `"Arquivo excede limite de 2GB"` → `"Arquivo excede limite de 5GB"`.
  NÃO alterar `MaxSingleXMLBytes` (10MB, linha 26) nem a mensagem de 10MB por XML (linhas 91, 276) — fora de escopo.</action>
  <verify>
    <automated>grep -q "MaxUploadFileBytes   = 5 \* 1024 \* 1024 \* 1024" backend/handlers/xml_upload.go && grep -q "MaxUncompressedBytes = 20 \* 1024 \* 1024 \* 1024" backend/handlers/xml_upload.go && grep -q "excede limite de 20GB" backend/handlers/xml_upload.go && [ "$(grep -c 'excede limite de 5GB' backend/handlers/xml_upload.go)" -eq 3 ] && ! grep -Eq "limite de 2GB|limite de 8GB|2 \* 1024 \* 1024 \* 1024|8 \* 1024 \* 1024 \* 1024" backend/handlers/xml_upload.go && (cd backend && gofmt -l handlers/xml_upload.go | grep -q . && echo "FMT_DIRTY" || echo OK)</automated>
  </verify>
  <done>MaxUploadFileBytes=5GB, MaxUncompressedBytes=20GB (fator 4x mantido), três mensagens "5GB" e uma "20GB", zero referências residuais a 2GB/8GB de upload, arquivo bem formatado (gofmt).</done>
</task>

<task type="auto">
  <name>Task 3: Atualizar limite e textos de UI no frontend React</name>
  <files>frontend/src/pages/ImportarXMLsSaida.tsx</files>
  <action>Atualizar o limite programático e todos os textos/toasts de UI de 2GB para 5GB:
  - Linha 91: toast `'Arquivo excede o limite de 2GB.'` → `'Arquivo excede o limite de 5GB.'`.
  - Linha 143: `maxSize: 2 * 1024 * 1024 * 1024,` → `maxSize: 5 * 1024 * 1024 * 1024,`.
  - Linha 146: toast `... Apenas XML e ZIP até 2GB.` → `... Apenas XML e ZIP até 5GB.`.
  - Linha 168: texto `Limite: 2GB por envio.` → `Limite: 5GB por envio.`.
  - Linha 200: texto `máximo 2GB` → `máximo 5GB`.
  Não alterar mais nada no arquivo.</action>
  <verify>
    <automated>grep -q "maxSize: 5 \* 1024 \* 1024 \* 1024" frontend/src/pages/ImportarXMLsSaida.tsx && ! grep -q "2GB\|2 \* 1024 \* 1024 \* 1024" frontend/src/pages/ImportarXMLsSaida.tsx && [ "$(grep -c '5GB' frontend/src/pages/ImportarXMLsSaida.tsx)" -eq 4 ] && echo OK</automated>
  </verify>
  <done>maxSize é 5GB e as quatro strings de UI dizem "5GB"; nenhuma referência residual a "2GB" resta no arquivo.</done>
</task>

</tasks>

<verification>
Escopo mínimo confirmado — apenas limites de tamanho de upload foram alterados nos 3 arquivos:

1. `grep -rn "2GB\|512M" frontend/nginx.conf backend/handlers/xml_upload.go frontend/src/pages/ImportarXMLsSaida.tsx` retorna vazio (zero referências residuais).
2. nginx: `client_max_body_size 5G` (>= 5GB do backend — bug do 512M corrigido).
3. backend: fator 4x preservado (5GB upload → 20GB descomprimido); `gofmt -l` limpo.
4. frontend: `maxSize` = 5GB e textos coerentes.
5. Sanidade de build (sem rodar build completo):
   `docker compose config` (dev) e `docker compose -f docker-compose.prod.yml config` continuam válidos. nginx.conf não é YAML — apenas confirmar que os composes ainda parseiam corretamente é suficiente.
</verification>

<success_criteria>
- Uploads de XML/ZIP até 5GB são aceitos em nginx, backend e frontend de forma consistente.
- Bug do nginx (512M < limite do app) corrigido: nginx agora aceita 5G.
- Proteção anti-ZIP-bomb mantém fator 4x (MaxUncompressedBytes = 20GB).
- Nenhuma referência residual a "2GB" / "512M" / "8GB" de upload nos 3 arquivos.
- `docker compose config` (dev e prod) permanecem válidos.
</success_criteria>

<output>
Create `.planning/quick/260702-mwi-aumentar-limite-de-upload-de-xml-zip-de-/260702-mwi-SUMMARY.md` when done
</output>
