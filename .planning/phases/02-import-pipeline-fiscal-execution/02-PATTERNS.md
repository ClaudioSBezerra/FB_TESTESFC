# Fase 2: Import Pipeline & Fiscal Execution - Mapa de Padrões

**Mapeado em:** 2026-07-01
**Arquivos analisados:** 11 (7 backend + 2 frontend + 2 migrations)
**Analogs encontrados:** 8 / 11 (3 sem analog direto — greenfield puro)

---

## File Classification

| Novo/Modificado | Papel | Fluxo de Dados | Analog mais próximo | Qualidade |
|---|---|---|---|---|
| `backend/handlers/xml_upload.go` | controller + parsing | file-I/O | `FB_APU04/backend/handlers/xml_upload.go` | exato (cópia seletiva) |
| `backend/handlers/nfe_saidas.go` | model + parsing + query | CRUD | `FB_APU04/backend/handlers/nfe_saidas.go` | exato (cópia seletiva) |
| `backend/migrations/006_nfe_saidas.sql` | migration | CRUD (schema) | `FB_APU04/backend/migrations/058_create_nfe_saidas.sql` | exato, com adaptação de escopo |
| `backend/migrations/007_nfe_saidas_itens.sql` | migration | CRUD (schema) | `FB_APU04/backend/migrations/075_create_nfe_itens_tables.sql` (seção `nfe_saidas_itens`) | exato, com adaptação de escopo |
| `backend/migrations/008_fiscal_execution_items.sql` | migration | CRUD (schema) | nenhum — schema proposto no `02-RESEARCH.md` (linhas 442-481) | sem analog — greenfield |
| `backend/handlers/fiscal_group_lookup.go` | service (Oracle read) | request-response (single-row lookup) | `backend/handlers/erp_bridge.go` (`ERPBridgeTestConnectionHandler`, conexão Oracle) | role-match — padrão de conexão reaproveitado, query é nova |
| `backend/handlers/fiscal_execution.go` | service + controller | event-driven (batch item-a-item) + CRUD (persistência) | `backend/handlers/erp_bridge.go` (conexão Oracle) + `FB_APU04/backend/handlers/xml_upload.go` (`processXMLBatch`, isolamento de erro por item) | role-match parcial — combinação de dois analogs |
| `backend/services/oracle_fiscal.go` | utility (monta bloco PL/SQL) | transform (string builder) | nenhum — greenfield, padrão inferido do script de teste do usuário (ver RESEARCH.md Pattern 1) | sem analog — greenfield |
| `backend/main.go` (rotas novas) | route registration | request-response | `backend/main.go` (rotas ERP Bridge já existentes, linhas 333-345) | exato — mesmo arquivo, mesmo padrão `withAuth`/`withDB` |
| `frontend/src/pages/ImportarXMLsSaida.tsx` | component (página) | file-I/O (upload) | `FB_APU04/frontend/src/pages/ImportarXMLsSaida.tsx` | exato (cópia seletiva) |
| `frontend/src/pages/ConsultaNFeSaidas.tsx` | component (página) | CRUD (consulta) | `FB_APU04/frontend/src/pages/ConsultaNFeSaidas.tsx` | exato (cópia seletiva), + adaptação nova (badge de status por item) |
| `frontend/src/lib/navigation.ts` (edição) | config | — | próprio arquivo (`FB_TESTESFC/frontend/src/lib/navigation.ts`) | exato — adicionar 2 tabs ao módulo `config` existente |

---

## Pattern Assignments

### `backend/handlers/xml_upload.go` (controller, file-I/O)

**Analog:** `FB_APU04/backend/handlers/xml_upload.go` (1118 linhas)

**Constantes de limite** (linhas 25-32):
```go
const (
	MaxUploadFileBytes   = 2 * 1024 * 1024 * 1024 // 2 GB — tamanho máximo do .zip/.xml enviado
	MaxUncompressedBytes = 8 * 1024 * 1024 * 1024 // 8 GB — proteção anti-ZIP bomb (total descomprimido)
	MaxSingleXMLBytes    = 10 * 1024 * 1024       // 10 MB — limite por XML individual
	MaxXMLsPerBatch      = 100_000
	BatchChunkSize       = 2000
	BatchAsyncThreshold  = 50
)
```
Copiar tal qual — já auditado (anti-ZIP-bomb, path traversal).

**Extração ZIP/RAR com mitigações de segurança** (linhas 42-159): `extractXMLsFromZip`, `extractXMLsFromZipFile`, `extractXMLsFromZipFiles`, `extractXMLsFromRarFile`. Copiar tal qual — usam `filepath.Base` (T-02-02-03, anti path-traversal) e verificação de `UncompressedSize64` acumulado (T-02-02-01, anti-ZIP-bomb).

**Handler HTTP principal — auth + validação + parse multipart** (`XMLUploadHandler`, linhas 618-748):
```go
func XMLUploadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			jsonErr(w, http.StatusMethodNotAllowed, "Método não permitido")
			return
		}
		// T-02-02-05: autenticação via JWT
		claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
		if !ok {
			jsonErr(w, http.StatusUnauthorized, "Não autenticado")
			return
		}
		userID, _ := claims["user_id"].(string)
		companyID, err := GetEffectiveCompanyID(db, userID, r.Header.Get("X-Company-ID"))
		// ... T-02-02-07: validar Content-Length ANTES de ler o body
		if r.ContentLength > MaxUploadFileBytes { ... }
		if err := r.ParseMultipartForm(64 << 20); err != nil { ... }
		tipo := strings.TrimSpace(r.FormValue("tipo"))
		// ...
	}
}
```
**Adaptação necessária:** FB_TESTESFC v1 só trata `saidas` — remover o branch `entradas`/`ctes` do `switch tipo` e da validação `tipo != "entradas" && tipo != "saidas" && tipo != "ctes"`. Manter o padrão de auth via `ClaimsKey`/`GetEffectiveCompanyID` (já existe em `handlers/middleware.go`/`handlers/hierarchy.go` no FB_TESTESFC, reaproveitado da Fase 1).

**Processamento de lote com isolamento de erro por arquivo** (`processXMLBatch`/`processSingleXML`, linhas 176-239, 242-330 aprox.): usar como referência direta para o padrão de isolamento de erro por item na Fase 2 (`fiscal_execution.go` deve seguir a mesma filosofia: nunca abortar o lote inteiro por um erro individual — ver seção "Padrões Compartilhados" abaixo).

**Adaptação:** remover a chamada a `RefreshCestNcmKB`/`mv_operacoes_simples` (linhas 222-238) — específica do domínio SPED/Simples Nacional do FB_APU04, fora de escopo do FB_TESTESFC.

---

### `backend/handlers/nfe_saidas.go` (model/parsing, CRUD)

**Analog:** `FB_APU04/backend/handlers/nfe_saidas.go` (874 linhas)

**Imports** (linhas 1-19):
```go
import (
	"bytes"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)
```
`golang.org/x/text` precisa ser adicionado ao `go.mod` do FB_TESTESFC se ainda não estiver presente (verificar antes de copiar).

**Structs de parsing XML** (linhas 38-216) — copiar tal qual: `nfeProc`, `nfe`, `protNFe`, `infProt`, `infNFe`, `transp`, `det`, `prod`, `detImposto`, `detICMSGrupo`, `detICMS`, `detPIS`, `detCOFINS`, `detIPI`, `ide`, `emit`, `enderEmit`, `dest`, `enderDest`, `total`, `icmsTot`, `ibsCbsTot`, `gIBS`, `gIBSuf`, `gIBSMun`, `gCBS`. Nenhuma adaptação necessária — mapeamento SEFAZ é padrão de mercado, independe do projeto.

**Helpers de conversão** (linhas 221-258): `toDecimal`, `toNullDecimal`, `toNullSmallInt` — copiar tal qual.

**Parsing de encoding legado** (linhas 260-328): `nfeCharsetReader`, `parseNFeXML`, `convertWindows1252` — copiar tal qual, incluindo o tratamento de namespace (`xmlns=`, prefixo `nfe:`) e o fallback de wrapper `<nfeProc>` quando a raiz é `<NFe>`.

**Extração de chave e data** (linhas 330-367): `extractChave`, `parseDhEmi` — copiar tal qual.

**Persistência de itens** (`insertNFeItens`, linhas 373+):
```go
func insertNFeItens(tx *sql.Tx, nfeID string, companyID string, dets []det, tableName string) error {
	for _, d := range dets {
		nItem, _ := strconv.Atoi(d.NItem)
		if nItem == 0 { continue }
		// extrai CST/CSOSN e origem do primeiro grupo ICMS presente
		// ...
		_, err := tx.Exec(fmt.Sprintf(`INSERT INTO %s (...) VALUES (...) ON CONFLICT (nfe_id, n_item) DO UPDATE SET ...`, tableName), ...)
	}
}
```
**Adaptação:** FB_TESTESFC só tem `nfe_saidas_itens` (sem `nfe_entradas_itens`) — pode remover o parâmetro `tableName` e fixar a tabela, ou manter genérico para eventual reuso futuro (decisão de discretion do planner). Manter o padrão `ON CONFLICT (nfe_id, n_item) DO UPDATE` para idempotência de reimportação.

**Adaptação de escopo (v_st a nível de item):** o RESEARCH.md aponta uma lacuna — o schema do FB_APU04 tem `v_st`/`v_bc_st` apenas em `nfe_saidas` (cabeçalho), não em `nfe_saidas_itens`. Se a Fase 3 precisar comparar ST por item, avaliar adicionar `v_bc_st`/`v_st` como colunas novas em `nfe_saidas_itens` nesta fase (planner deve decidir).

---

### `backend/handlers/xml_upload.go` — handler de upload principal (rota)

**Registro de rota** — seguir o padrão já usado no `main.go` do FB_TESTESFC para ERP Bridge (ver "Padrões Compartilhados" abaixo). O FB_APU04 registra em `POST /api/xml/upload`; manter o mesmo path.

---

### `backend/migrations/006_nfe_saidas.sql` (migration)

**Analog:** `FB_APU04/backend/migrations/058_create_nfe_saidas.sql`

```sql
CREATE TABLE IF NOT EXISTS nfe_saidas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    chave_nfe       VARCHAR(44) NOT NULL,
    modelo          SMALLINT NOT NULL,
    serie           VARCHAR(3),
    numero_nfe      VARCHAR(9),
    data_emissao    DATE NOT NULL,
    mes_ano         VARCHAR(7) NOT NULL,
    nat_op          VARCHAR(60),
    emit_cnpj       VARCHAR(14) NOT NULL,
    emit_nome       VARCHAR(60),
    emit_uf         VARCHAR(2),
    emit_municipio  VARCHAR(60),
    dest_cnpj_cpf   VARCHAR(14),
    dest_nome       VARCHAR(60),
    dest_uf         VARCHAR(2),
    dest_c_mun      VARCHAR(7),
    v_bc NUMERIC(15,2) DEFAULT 0, v_icms NUMERIC(15,2) DEFAULT 0,
    v_icms_deson NUMERIC(15,2) DEFAULT 0, v_fcp NUMERIC(15,2) DEFAULT 0,
    v_bc_st NUMERIC(15,2) DEFAULT 0, v_st NUMERIC(15,2) DEFAULT 0,
    v_fcp_st NUMERIC(15,2) DEFAULT 0, v_fcp_st_ret NUMERIC(15,2) DEFAULT 0,
    v_prod NUMERIC(15,2) DEFAULT 0, v_frete NUMERIC(15,2) DEFAULT 0,
    v_seg NUMERIC(15,2) DEFAULT 0, v_desc NUMERIC(15,2) DEFAULT 0,
    v_ii NUMERIC(15,2) DEFAULT 0, v_ipi NUMERIC(15,2) DEFAULT 0,
    v_ipi_devol NUMERIC(15,2) DEFAULT 0, v_pis NUMERIC(15,2) DEFAULT 0,
    v_cofins NUMERIC(15,2) DEFAULT 0, v_outro NUMERIC(15,2) DEFAULT 0, v_nf NUMERIC(15,2) DEFAULT 0,
    v_bc_ibs_cbs NUMERIC(15,2), v_ibs_uf NUMERIC(15,2), v_ibs_mun NUMERIC(15,2),
    v_ibs NUMERIC(15,2), v_cred_pres_ibs NUMERIC(15,2), v_cbs NUMERIC(15,2), v_cred_pres_cbs NUMERIC(15,2),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_nfe_saidas_company_chave UNIQUE (company_id, chave_nfe)
);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_company_mes  ON nfe_saidas(company_id, mes_ano);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_company_data ON nfe_saidas(company_id, data_emissao);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_emit_cnpj    ON nfe_saidas(company_id, emit_cnpj);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_dest_c_mun   ON nfe_saidas(company_id, dest_c_mun);
```

**Adaptação de nomenclatura/numeração:** renomear arquivo para o padrão enxuto já usado no FB_TESTESFC (`00X_nome.sql`, ver `001_auth_hierarchy.sql`...`005_seed_erp_bridge_ferreira_costa.sql`) — próximo número livre é `006`. `REFERENCES companies(id)` deve apontar para a tabela `companies` já criada na Fase 1 do FB_TESTESFC (confirmar nome exato em `001_auth_hierarchy.sql`/`003_managers.sql`). Nenhum campo precisa ser removido — o schema já é "enxuto" (não tem colunas de CTe/SPED).

---

### `backend/migrations/007_nfe_saidas_itens.sql` (migration)

**Analog:** `FB_APU04/backend/migrations/075_create_nfe_itens_tables.sql`, seção `nfe_saidas_itens` (linhas 65-116)

```sql
CREATE TABLE IF NOT EXISTS nfe_saidas_itens (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    nfe_id          UUID        NOT NULL REFERENCES nfe_saidas(id) ON DELETE CASCADE,
    company_id      UUID        NOT NULL,
    n_item          SMALLINT    NOT NULL,
    c_prod          VARCHAR(60),
    x_prod          VARCHAR(120) NOT NULL,
    ncm             VARCHAR(8),
    cfop            VARCHAR(4),
    cst_icms        VARCHAR(3),
    cst_pis         VARCHAR(2),
    cst_cofins      VARCHAR(2),
    v_prod          NUMERIC(15,2) DEFAULT 0,
    v_total_item    NUMERIC(15,2) DEFAULT 0,
    v_bc_icms       NUMERIC(15,2) DEFAULT 0,
    v_icms          NUMERIC(15,2) DEFAULT 0,
    v_ipi           NUMERIC(15,2) DEFAULT 0,
    v_bc_pis        NUMERIC(15,2) DEFAULT 0,
    v_pis           NUMERIC(15,2) DEFAULT 0,
    v_bc_cofins     NUMERIC(15,2) DEFAULT 0,
    v_cofins        NUMERIC(15,2) DEFAULT 0,
    v_ibs           NUMERIC(15,2) DEFAULT 0,
    v_cbs           NUMERIC(15,2) DEFAULT 0,
    cclasstrib      VARCHAR(20),
    CONSTRAINT uq_nfe_saidas_itens_nfe_item UNIQUE (nfe_id, n_item)
);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_itens_company_ncm ON nfe_saidas_itens(company_id, ncm);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_itens_nfe_id      ON nfe_saidas_itens(nfe_id);
```

**Nota:** `insertNFeItens` (FB_APU04) referencia colunas `cst_orig` e `cest` que **não aparecem** neste CREATE TABLE — checar a migração `106` (citada no RESEARCH.md como fonte lida) para confirmar se essas colunas foram adicionadas depois via `ALTER TABLE`; se sim, incluí-las diretamente no `CREATE TABLE` novo do FB_TESTESFC em vez de replicar duas migrações.

**Adaptação de escopo:** avaliar adicionar `v_bc_st`/`v_st` a nível de item (ver nota na seção `nfe_saidas.go` acima) — lacuna identificada no RESEARCH.md, necessária para a Fase 3 comparar ICMS-ST item a item.

---

### `backend/migrations/008_fiscal_execution_items.sql` (migration) — SEM ANALOG (greenfield)

Usar diretamente o schema já desenhado no RESEARCH.md (linhas 442-481), copiado aqui para referência do planner:

```sql
CREATE TABLE IF NOT EXISTS fiscal_execution_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id          UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    nfe_item_id         UUID NOT NULL REFERENCES nfe_saidas_itens(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'pending', -- pending | ok | error | sem_grupo_fiscal
    error_message        TEXT,
    executed_at         TIMESTAMPTZ,
    grupo_fiscal_codigo TEXT,
    input_params        JSONB,
    base_calculo_icms          NUMERIC(15,2),
    valor_icms                 NUMERIC(15,2),
    base_substituicao          NUMERIC(15,2),
    valor_substituicao         NUMERIC(15,2),
    base_calculo_pis           NUMERIC(15,2),
    valor_pis                  NUMERIC(15,2),
    base_calculo_cofins        NUMERIC(15,2),
    valor_cofins                NUMERIC(15,2),
    percentual_difal           NUMERIC(7,4),
    valor_icms_partilha_destino NUMERIC(15,2),
    valor_icms_pobreza         NUMERIC(15,2),
    full_result         JSONB NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_fiscal_execution_item UNIQUE (nfe_item_id)
);
CREATE INDEX IF NOT EXISTS idx_fiscal_execution_status   ON fiscal_execution_items(company_id, status);
CREATE INDEX IF NOT EXISTS idx_fiscal_execution_nfe_item ON fiscal_execution_items(nfe_item_id);
```

**Convenção a seguir:** `gen_random_uuid()`, `REFERENCES ... ON DELETE CASCADE`, `TIMESTAMPTZ NOT NULL DEFAULT now()` — mesmo estilo já usado em `001_auth_hierarchy.sql`/`002_erp_bridge.sql` do próprio FB_TESTESFC (confirmar convenção exata de nome de PK/timestamp lendo essas migrações antes de finalizar, mas o padrão UUID+gen_random_uuid() já é usado nas migrations 001-005 existentes).

---

### `backend/handlers/fiscal_group_lookup.go` — SEM ANALOG direto (greenfield, estilo herdado de `erp_bridge.go`)

**Analog de estilo:** `backend/handlers/erp_bridge.go` (399 linhas, próprio FB_TESTESFC, Fase 1)

**Padrão de conexão Oracle a reaproveitar** (`ERPBridgeTestConnectionHandler`, linhas 331-399):
```go
dsnPlain := DecryptFieldWithFallback(oracleDsn.String)
usuarioPlain := DecryptFieldWithFallback(oracleUsuario.String)
senhaPlain := DecryptFieldWithFallback(oracleSenha.String)

var connStr string
if strings.HasPrefix(dsnPlain, "oracle://") {
	connStr = dsnPlain
} else {
	connStr = fmt.Sprintf("oracle://%s:%s@%s", usuarioPlain, senhaPlain, dsnPlain)
}

conn, err := sql.Open("oracle", connStr)
if err != nil { ... }
defer conn.Close()

ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
defer cancel()
if pingErr := conn.PingContext(ctx); pingErr != nil { ... }
```
**Adaptação:** em vez de `PingContext`, `fiscal_group_lookup.go` deve fazer `conn.QueryRowContext(ctx, query, cProd, codEmpresa).Scan(&grupoFiscal, ...)` usando a mesma lógica de resolução de `connStr` a partir de `erp_bridge_config`. **Nunca** enviar `err.Error()` bruto ao cliente (o driver go-ora pode vazar DSN/senha) — seguir o mesmo padrão de mensagem genérica sanitizada usado em `ERPBridgeTestConnectionHandler` (linhas 383, 393).

**Query real (fonte: RESEARCH.md A1, confirmada pelo usuário):**
```sql
SELECT pb.grupo_fiscal, p.especial AS origem, p.ncm
FROM prodb pb, prod p
WHERE p.codigo = pb.codigo
  AND pb.codigo = :codigoProduto
  AND pb.cod_empresa = :codEmpresa
```
O filtro `cod_empresa` é obrigatório (evita múltiplas linhas/grupo fiscal errado — ver A6 no RESEARCH.md). `cod_empresa` é resolvido por um mapeamento estático Go (`2`=Recife/PE, `1`=Garanhuns/PE) a partir de um campo do XML — decisão de implementação em aberto (RESEARCH.md, Open Question #5).

**Reaproveitar também:** `erpBridgeGetCompany` (linhas 43-50, `erp_bridge.go`) para obter `companyID` a partir do JWT antes de buscar as credenciais Oracle salvas.

---

### `backend/handlers/fiscal_execution.go` — SEM ANALOG direto (greenfield, combina 2 padrões)

**Analog 1 (conexão Oracle):** `backend/handlers/erp_bridge.go` — mesmo padrão de `sql.Open("oracle", connStr)` + `SetMaxOpenConns` (RESEARCH.md Pitfall 5 recomenda configurar isso explicitamente; `erp_bridge.go` atual não configura `SetMaxOpenConns` na conexão de teste — **fiscal_execution.go deve adicionar isso**, já que fará chamadas repetidas item a item, diferente do teste pontual de conexão).

**Analog 2 (isolamento de erro por item em lote):** `FB_APU04/backend/handlers/xml_upload.go` (`processXMLBatch`/`processSingleXML`, linhas 176-330) — padrão de nunca abortar o lote por erro de um item, registrar `error_details`/status por item, log com prefixo `[Modulo] batch=...`.

**Padrão a seguir (já validado no RESEARCH.md, "Code Examples"):**
```go
type ItemResult struct {
    ItemID string
    Status string // "ok" | "error" | "sem_grupo_fiscal"
    ErrMsg string
}

func processFiscalBatch(ctx context.Context, oracleDB *sql.DB, pgDB *sql.DB, itens []NFeItem) []ItemResult {
    sem := make(chan struct{}, 5) // limita concorrência Oracle (Pitfall 5)
    // ... goroutine por item com defer recover() + timeout de contexto (15s) por item
}
```
**Nota de adaptação:** `db.SetMaxOpenConns(5)` deve ser aplicado na conexão Oracle dedicada a este handler (não na conexão de teste do `erp_bridge.go`), conforme Pitfall 5 do RESEARCH.md.

**Persistência de resultado/erro:** seguir o padrão de `UPDATE ... SET status=..., error_details=$N` já usado em `processXMLBatch` (linhas 205-215 do FB_APU04) — adaptado para `fiscal_execution_items` (`status`, `error_message`, `full_result` em vez de `error_details` agregado).

---

### `backend/services/oracle_fiscal.go` — SEM ANALOG (greenfield puro)

Nenhum código existente no FB_APU04 ou FB_TESTESFC monta blocos PL/SQL anônimos. Usar o Pattern 1 do RESEARCH.md (linhas 140-200) como especificação de implementação — reproduzido aqui:

```go
const calculaImpostoBlock = `
declare
  result PKG_FISCAL_FCTAX.RDADOS_FISCAIS_PRODUTO;
begin
  result := PKG_FISCAL_FCTAX.calcula_imposto_produto(
    pCnpjEmpresa => :pCnpjEmpresa,
    ...
  );
  :oTipoImposto := result.TipoImposto;
  ...
end;`

_, err := db.ExecContext(ctx, calculaImpostoBlock,
    sql.Named("pCnpjEmpresa", cnpjEmpresa),
    ...
    sql.Named("oTipoImposto", sql.Out{Dest: &out.TipoImposto}),
    ...
)
```
**Regra de segurança (RESEARCH.md, Security Domain — Injeção via bloco PL/SQL):** a string do bloco deve ser **estática/gerada a partir de metadados fixos no código**, nunca a partir de entrada do usuário; todos os valores passam por `sql.Named`/bind variables, nunca `fmt.Sprintf` concatenando valor no SQL. Gerar a string + a lista de binds programaticamente a partir de uma única tabela de metadados (campo Oracle → nome do bind → tipo Go), não escrever os ~65 binds à mão em múltiplos lugares.

**Reaproveitar o padrão de criptografia** de `backend/handlers/crypto.go` (`EncryptField`/`DecryptFieldWithFallback`, linhas 49-104) sem modificação — já é usado por `erp_bridge.go` para o DSN/usuário/senha Oracle; `oracle_fiscal.go`/`fiscal_execution.go` apenas consomem essas credenciais já descriptografadas, não implementam nova criptografia.

---

### `backend/main.go` — registro de rotas novas

**Analog:** próprio `backend/main.go`, seção "ERP Bridge" (linhas 333-345)

```go
http.HandleFunc("/api/erp-bridge/config",                  withAuth(handlers.ERPBridgeConfigHandler, ""))
http.HandleFunc("/api/erp-bridge/config/generate-api-key", withAuth(handlers.ERPBridgeGenerateAPIKeyHandler, "admin"))
http.HandleFunc("/api/erp-bridge/test-connection",         withAuth(handlers.ERPBridgeTestConnectionHandler, ""))
```

**Padrão a replicar para as rotas novas desta fase:**
```go
http.HandleFunc("/api/xml/upload",                withAuth(handlers.XMLUploadHandler, ""))
http.HandleFunc("/api/nfe-saidas",                withAuth(handlers.NFeSaidasListHandler, ""))  // GET consulta
http.HandleFunc("/api/nfe-saidas/",               withAuth(handlers.NFeSaidaDetailHandler, "")) // GET detalhe + itens
http.HandleFunc("/api/fiscal-execution/run",      withAuth(handlers.FiscalExecutionRunHandler, "")) // dispara pipeline
```
Usar sempre `withAuth(handlerFactory, role)` (role vazio = qualquer usuário autenticado; `"admin"` quando aplicável) e `withDB` apenas para rotas sem necessidade de autenticação (nenhuma rota nova desta fase deve ser pública — todas tocam dados fiscais da empresa). Ver helpers `withDB`/`withAuth` em `main.go` linhas 201-223.

---

### `frontend/src/pages/ImportarXMLsSaida.tsx` (component, file-I/O)

**Analog:** `FB_APU04/frontend/src/pages/ImportarXMLsSaida.tsx` (464 linhas)

**Imports** (linhas 1-16):
```tsx
import { useState, useEffect, useRef } from 'react';
import { useDropzone } from 'react-dropzone';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Upload, CloudUpload, CheckCircle, XCircle, Loader2, FolderOpen } from 'lucide-react';
```
Todos esses componentes/libs já existem no FB_TESTESFC (`frontend/src/components/ui/{card,badge,progress,table}.tsx`, `react-dropzone` e `sonner` já no `package.json`) — nenhuma dependência nova a instalar.

**Máquina de estados de upload** (`UploadState`, linha 21): `'idle' | 'scanning' | 'uploading' | 'polling' | 'done' | 'error'` — copiar tal qual (UI-SPEC confirma reaproveitar os mesmos estados).

**Handler de upload com FormData multi-arquivo** (linhas 166-235): copiar tal qual, adaptando `formData.append('tipo', TIPO)` — no FB_TESTESFC não há múltiplos tipos, então o backend pode nem exigir esse campo, mas manter por consistência de endpoint se `XMLUploadHandler` for reaproveitado com a mesma assinatura.

**Adaptação obrigatória (UI-SPEC, linha 104):** remover o campo "Mês de competência" (linhas 278-315) — FB_TESTESFC não é ferramenta de apuração/competência, é validação fiscal por nota. Remover também qualquer lógica de `competencia` no `handleUpload`.

**Adaptação de densidade de tabela (UI-SPEC, linhas 36-59):** trocar `text-[11px]` (linhas 429-453) por `text-sm` (14px) nas células/cabeçalhos da tabela de histórico — contrato de tipografia desta fase eliminou o tamanho `text-[11px]`. Trocar `font-medium`/`font-semibold` por `font-bold` onde usado (contrato limita a 2 pesos: 400 e 700).

**Adaptação de formatos aceitos:** o FB_APU04 aceita `.xml`, `.zip`, `.rar`, `.7z` (linha 343-344) — confirmar com o planner se `.7z` deve ser removido (RESEARCH.md não lista `7z` como dependência auditada; apenas `zip`/`rar` têm libs confirmadas no Standard Stack).

---

### `frontend/src/pages/ConsultaNFeSaidas.tsx` (component, CRUD/consulta)

**Analog:** `FB_APU04/frontend/src/pages/ConsultaNFeSaidas.tsx` (490 linhas)

**Imports** (linhas 1-30) — incluir `Dialog`/`DialogContent`/`DialogHeader`/`DialogTitle` (já existentes em `frontend/src/components/ui/dialog.tsx` do FB_TESTESFC), `Select` (idem). Remover import de `formatCnpjComApelido` de `@/lib/formatFilial` se esse helper não existir no FB_TESTESFC — checar/copiar o util junto, ou substituir por `fmtCNPJ` simples (já presente no próprio arquivo, linhas 90-98).

**Interface de linha da tabela** (`NfeSaidaRow`, linhas 35-80) — copiar tal qual, já reflete 1:1 as colunas de `nfe_saidas` propostas na migration 006.

**Componente de detalhe em Dialog** (`DetalheNFe`, linhas 110-180+) — copiar a estrutura (`Secao`, `Linha`, `LinhaBRL`) e os valores de `ICMSTot`. **Extensão nova desta fase (UI-SPEC linhas 116-136):** dentro do Dialog de detalhe, adicionar uma sub-tabela/lista de itens com badge de status (`ok`/`sem_grupo_fiscal`/`error`) por item — não existe no analog original, é a única adição de UI greenfield desta tela. Usar o mesmo padrão de badge já demonstrado em `StatusBadge` de `ImportarXMLsSaida.tsx` (linhas 54-65) como modelo de implementação (`Record<string, {label, className}>` + `<Badge variant="outline" className={...}>`), adaptando as 3 cores exigidas pelo UI-SPEC (`--accent` verde / `--warning` âmbar / `--destructive` vermelho).

**Adaptação de tipografia/densidade:** mesmo ajuste de `ImportarXMLsSaida.tsx` — eliminar `text-[11px]`/`text-[10px]` em favor de `text-sm`/`text-xs` conforme contrato desta fase (UI-SPEC linhas 53-59).

---

### `frontend/src/lib/navigation.ts` (edição, config)

**Analog:** próprio arquivo, módulo `config` já existente

**Padrão atual do FB_TESTESFC:**
```ts
config: {
  label: 'Configurações',
  tabs: [
    { label: 'Credenciais ERP',  path: '/config/erp-bridge',         adminOnly: true },
    { label: 'Config ERP',       path: '/importacoes/erp-bridge',     adminOnly: true },
    { label: 'Ambiente',         path: '/config/ambiente' },
    { label: 'Gestores',         path: '/config/gestores' },
    { label: 'Usuários',         path: '/config/usuarios',            adminOnly: true },
  ],
},
```

**Alteração desta fase (UI-SPEC linhas 146-163):** inserir duas tabs novas logo após `'Config ERP'` e antes de `'Ambiente'`:
```ts
{ label: 'Importar XMLs',    path: '/importacoes/xmls-saida' },
{ label: 'Notas Importadas', path: '/importacoes/notas-saida' },
```
Sem `adminOnly` (qualquer usuário autenticado acessa, conforme UI-SPEC não menciona restrição). `getActiveModule` já trata `pathname.startsWith('/importacoes/')` (linha existente) — nenhuma mudança adicional necessária nessa função.

---

## Padrões Compartilhados (Shared Patterns)

### Autenticação/Autorização
**Fonte:** `backend/main.go` (linhas 213-223, helpers `withAuth`/`withDB`) + `backend/handlers/erp_bridge.go` (`erpBridgeGetCompany`, linhas 43-50)
**Aplicar a:** todos os handlers novos desta fase (`XMLUploadHandler`, `NFeSaidasListHandler`, `NFeSaidaDetailHandler`, `FiscalGroupLookupHandler` interno, `FiscalExecutionRunHandler`).
```go
func withAuth(handlerFactory func(*sql.DB) http.HandlerFunc, role string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		database := getDB()
		if database == nil { jsonServiceUnavailable(w); return }
		h := handlerFactory(database)
		handlers.AuthMiddleware(h, role)(w, r)
	}
}
```
Nenhum endpoint novo desta fase deve aceitar `company_id` arbitrário do cliente — sempre resolver via `GetEffectiveCompanyID(db, userID, r.Header.Get("X-Company-ID"))`, igual ao padrão já usado em `erpBridgeGetCompany`.

### Criptografia de credenciais Oracle
**Fonte:** `backend/handlers/crypto.go` (linhas 49-104, `EncryptField`/`DecryptField`/`DecryptFieldWithFallback`)
**Aplicar a:** `fiscal_group_lookup.go` e `fiscal_execution.go` — ambos consomem `oracle_dsn`/`oracle_usuario`/`oracle_senha` já salvos em `erp_bridge_config` (Fase 1), sempre via `DecryptFieldWithFallback`, nunca reimplementar criptografia.

### Conexão Oracle via go-ora
**Fonte:** `backend/handlers/erp_bridge.go` (linhas 362-390, montagem de `connStr` + `sql.Open("oracle", connStr)` + `PingContext` com timeout)
**Aplicar a:** `fiscal_group_lookup.go` (query pontual) e `fiscal_execution.go` (múltiplas chamadas — adicionar `SetMaxOpenConns(5)` conforme Pitfall 5 do RESEARCH.md, ausente no handler de teste de conexão original).
**Regra crítica:** nunca propagar `err.Error()`/`pingErr.Error()` bruto ao cliente — o driver go-ora pode incluir DSN com senha em texto claro na mensagem de erro. Sempre usar mensagem genérica sanitizada, como já feito em `ERPBridgeTestConnectionHandler` (linhas 383, 393).

### Isolamento de erro por item em lote
**Fonte:** `FB_APU04/backend/handlers/xml_upload.go` (`processXMLBatch`, linhas 176-239) + Code Example do RESEARCH.md (linhas 288-350)
**Aplicar a:** `fiscal_execution.go` — cada item tem seu próprio status (`pending`/`ok`/`error`/`sem_grupo_fiscal`), erro de um item nunca aborta o lote, log com prefixo padronizado (`[FiscalExecution] item=%s err=%v`, seguindo o estilo `[XMLUpload] batch=%s file=%s err=%v` já usado).

### Migrations — convenção de numeração e idempotência
**Fonte:** `backend/migrations/001_auth_hierarchy.sql` a `005_seed_erp_bridge_ferreira_costa.sql` (já existentes no FB_TESTESFC)
**Aplicar a:** `006_nfe_saidas.sql`, `007_nfe_saidas_itens.sql`, `008_fiscal_execution_items.sql` — sempre `CREATE TABLE IF NOT EXISTS`, `gen_random_uuid()` para PK, `TIMESTAMPTZ`/`TIMESTAMP WITH TIME ZONE DEFAULT` para timestamps, `REFERENCES ... ON DELETE CASCADE` para FKs de escopo empresa/nota. O runner de migrations em `main.go` (`onDBConnected`, linhas 84-149) já trata cada arquivo como idempotente via tabela `schema_migrations` — nenhuma mudança necessária nesse mecanismo.

### Componentes de UI (shadcn vendorizado)
**Fonte:** `frontend/src/components/ui/*.tsx` (já existentes: `card`, `badge`, `table`, `dialog`, `progress`, `select`, `input`, `button`)
**Aplicar a:** ambas as telas novas — nenhum componente novo via CLI/registry necessário nesta fase (confirmado no UI-SPEC, seção Registry Safety).

---

## No Analog Found

Arquivos sem correspondência direta no codebase (planner deve usar RESEARCH.md como fonte primária de padrão):

| Arquivo | Papel | Fluxo de Dados | Razão |
|---|---|---|---|
| `backend/services/oracle_fiscal.go` | utility (monta bloco PL/SQL) | transform | Nenhum código Go do FB_APU04 ou da Fase 1 do FB_TESTESFC monta blocos PL/SQL anônimos com binds OUT escalares — funcionalidade 100% nova, especificada em detalhe no RESEARCH.md (Pattern 1, linhas 140-200) |
| `backend/migrations/008_fiscal_execution_items.sql` | migration (schema) | CRUD | Tabela nova sem precedente — schema já desenhado no RESEARCH.md (linhas 442-481), a copiar diretamente |
| `backend/handlers/fiscal_group_lookup.go` (a query em si) | service | request-response | O padrão de *conexão* Oracle é reaproveitado de `erp_bridge.go`, mas a *query* contra `prod`/`PRODB` é nova (nenhum código Go do projeto consulta essas tabelas hoje) |

---

## Metadata

**Escopo da busca de analogs:** `FB_APU04/backend/handlers/`, `FB_APU04/backend/migrations/`, `FB_APU04/frontend/src/pages/`, `FB_TESTESFC/backend/handlers/`, `FB_TESTESFC/backend/migrations/`, `FB_TESTESFC/frontend/src/{pages,components/ui,lib}/`
**Arquivos lidos integralmente:** `erp_bridge.go` (399 linhas), `crypto.go` (104 linhas), `main.go` (383 linhas) do FB_TESTESFC; `ImportarXMLsSaida.tsx` (464 linhas) do FB_APU04 — todos ≤ 2000 linhas, lidos em uma única passada
**Arquivos grandes lidos em trechos não sobrepostos:** `FB_APU04/backend/handlers/xml_upload.go` (1118 linhas — lido em 3 trechos: 1-200, 200-420, 610-750) e `nfe_saidas.go` (874 linhas — lido em 2 trechos: 1-240, 240-440); `ConsultaNFeSaidas.tsx` (490 linhas — lido parcialmente, 1-180, suficiente para extrair padrão de tipos/Dialog)
**Data de mapeamento:** 2026-07-01
