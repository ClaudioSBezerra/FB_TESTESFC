# Phase 3: Visual Comparison Screen - Pattern Map

**Mapped:** 2026-07-02
**Files analyzed:** 6 (new/modified)
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|-----------------|---------------|
| `backend/handlers/fiscal_comparison.go` (new — `FiscalComparisonListHandler`, GET `/api/fiscal-comparison`) | handler | CRUD (read, JOIN) | `backend/handlers/nfe_saidas.go` → `NFeSaidaDetailHandler` (LEFT JOIN section, lines 756-793) | exact |
| `backend/main.go` (modified — register new route) | route registration | request-response | `backend/main.go` lines 350-358 (`/api/nfe-saidas`, `/api/fiscal-execution/run` registrations) | exact |
| `frontend/src/pages/ComparacaoFiscal.tsx` (new) | component (page) | request-response (list + client filter) | `frontend/src/pages/ConsultaNFeSaidas.tsx` (whole file — list page + Dialog pattern) | exact |
| `frontend/src/pages/ComparacaoFiscal.tsx` — Dialog de detalhe do item | component (dialog) | request-response | `ConsultaNFeSaidas.tsx` → `DetalheNFe` (lines 220-393) | exact |
| `frontend/src/pages/ComparacaoFiscal.tsx` — badge de divergência (OK/Divergente/Não calculado) | component | transform (derived state) | `ConsultaNFeSaidas.tsx` → `FiscalStatusBadge` + `FISCAL_STATUS_META` (lines 123-161) | exact |
| `frontend/src/lib/navigation.ts` (modified — add tab) | config | — | `frontend/src/lib/navigation.ts` (existing `tabs` array, lines 17-25) | exact |
| `frontend/src/App.tsx` (modified — add route) | route | — | `frontend/src/App.tsx` lines 63-64 (`/importacoes/notas-saida` route) | exact |

## Pattern Assignments

### `backend/handlers/fiscal_comparison.go` (new handler, CRUD/read + JOIN)

**Analog:** `backend/handlers/nfe_saidas.go` (`NFeSaidaDetailHandler`, `NFeSaidasListHandler`)

**Imports pattern** (nfe_saidas.go lines 1-18 — for a comparison-only handler you need a subset, no XML parsing needed):
```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)
```

**Auth/company-scoping pattern** (nfe_saidas.go lines 593-607, and `erp_bridge.go` lines 43-50 — this is the shared cross-cutting pattern, copy verbatim):
```go
companyID, err := erpBridgeGetCompany(db, r)
if err != nil {
    jsonErr(w, http.StatusUnauthorized, "Não autenticado")
    return
}
```
```go
// erp_bridge.go:43-50 — what erpBridgeGetCompany does under the hood
func erpBridgeGetCompany(db *sql.DB, r *http.Request) (string, error) {
	claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
	if !ok {
		return "", sql.ErrNoRows
	}
	userID := claims["user_id"].(string)
	return GetEffectiveCompanyID(db, userID, r.Header.Get("X-Company-ID"))
}
```

**Core JOIN pattern to copy** (nfe_saidas.go lines 760-793 — `NFeSaidaDetailHandler` item query; this is the exact expected-vs-calculated join the new list endpoint needs, just without the `WHERE i.nfe_id = $1` restriction and with `nfe_saidas` joined in too for nota/cliente columns):
```go
itemRows, err := db.Query(`
    SELECT
        i.id, i.n_item, COALESCE(i.c_prod,''), i.x_prod, COALESCE(i.ncm,''), COALESCE(i.cest,''),
        COALESCE(i.cfop,''), COALESCE(i.cst_icms,''), COALESCE(i.cst_orig,''),
        COALESCE(i.cst_pis,''), COALESCE(i.cst_cofins,''),
        i.v_prod, i.v_bc_icms, i.v_icms, i.v_bc_st, i.v_st, i.v_ipi,
        i.v_bc_pis, i.v_pis, i.v_bc_cofins, i.v_cofins, i.v_ibs, i.v_cbs, COALESCE(i.cclasstrib,''),
        COALESCE(f.status,''), COALESCE(f.error_message,'')
    FROM nfe_saidas_itens i
    LEFT JOIN fiscal_execution_items f ON f.nfe_item_id = i.id
    WHERE i.nfe_id = $1
    ORDER BY i.n_item ASC`, row.ID)
```

**Extended JOIN for Phase 3** — the new query needs `nfe_saidas` (for nota/cliente columns per D-02) plus the calculated columns from `fiscal_execution_items` for ICMS/ICMS-ST/PIS/COFINS (D-04) plus `full_result` for the Dialog's "Só calculado" section (D-07):
```sql
SELECT
    i.id AS item_id, i.n_item, i.x_prod, i.ncm, i.cfop,
    n.id AS nfe_id, n.numero_nfe, n.serie, n.dest_nome, n.dest_cnpj_cpf, n.data_emissao,
    -- esperado (XML)
    i.v_bc_icms, i.v_icms, i.v_bc_st, i.v_st,
    i.v_bc_pis, i.v_pis, i.v_bc_cofins, i.v_cofins,
    -- calculado (pacote fiscal)
    f.status, f.error_message,
    f.base_calculo_icms, f.valor_icms, f.base_substituicao, f.valor_substituicao,
    f.base_calculo_pis, f.valor_pis, f.base_calculo_cofins, f.valor_cofins,
    f.full_result
FROM nfe_saidas_itens i
JOIN nfe_saidas n ON n.id = i.nfe_id
LEFT JOIN fiscal_execution_items f ON f.nfe_item_id = i.id
WHERE i.company_id = $1
ORDER BY n.data_emissao DESC, i.n_item ASC
```
Note: `fiscal_execution_items.base_substituicao`/`valor_substituicao` map to ICMS-ST (D-04's 4th pair) — confirmed via `oracle_fiscal.go` `FiscalResult.BaseSubstituicao`/`ValorSubstituicao` and `persistFiscalItemResult` (fiscal_execution.go lines 349-350).

**Response shape pattern** (nfe_saidas.go lines 652-656, 795-798 — flat `map[string]interface{}` with a `total`/`items` or nested `nfe`/`itens` envelope):
```go
json.NewEncoder(w).Encode(map[string]interface{}{
    "total": len(list),
    "items": list,
})
```

**Error handling pattern** (nfe_saidas.go lines 624-628, 750-754 — log with context then generic `jsonErr`, never leak raw `err.Error()` to client):
```go
if err != nil {
    log.Printf("NFeSaidasList query error (company %s): %v", companyID, err)
    jsonErr(w, http.StatusInternalServerError, "Erro ao consultar notas")
    return
}
```

**Method-guard pattern** (nfe_saidas.go lines 597-600):
```go
if r.Method != http.MethodGet {
    jsonErr(w, http.StatusMethodNotAllowed, "Método não permitido")
    return
}
```

---

### `backend/main.go` (route registration)

**Analog:** `backend/main.go` lines 350-358

**Pattern to copy** (exact block — plain-GET routes use `withAuth(handler, "")`, no role restriction, comment marks the module boundary):
```go
// T-02-05: todas as rotas exigem autenticação (withAuth, role vazia).
http.HandleFunc("/api/xml/upload", withAuth(handlers.XMLUploadHandler, ""))
http.HandleFunc("/api/nfe-saidas", withAuth(handlers.NFeSaidasListHandler, ""))
http.HandleFunc("/api/nfe-saidas/", withAuth(handlers.NFeSaidaDetailHandler, ""))
...
http.HandleFunc("/api/fiscal-execution/run", withAuth(handlers.FiscalExecutionRunHandler, ""))
```
New line to add near this block:
```go
http.HandleFunc("/api/fiscal-comparison", withAuth(handlers.FiscalComparisonListHandler, ""))
```

`withAuth` definition for reference (main.go lines 212-223 — do not modify, just consume):
```go
func withAuth(handlerFactory func(*sql.DB) http.HandlerFunc, role string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		database := getDB()
		if database == nil {
			jsonServiceUnavailable(w)
			return
		}
		h := handlerFactory(database)
		handlers.AuthMiddleware(h, role)(w, r)
	}
}
```

---

### `frontend/src/pages/ComparacaoFiscal.tsx` (new page, request-response + client-side filter)

**Analog:** `frontend/src/pages/ConsultaNFeSaidas.tsx` (entire file, 622 lines — use as structural template)

**Imports pattern** (ConsultaNFeSaidas.tsx lines 1-29 — copy verbatim, same UI kit/react-query/lucide icons; swap `Search, X, Play` icons for whatever the filter toggle needs, e.g. add `Filter` or `AlertTriangle`):
```typescript
import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '@/contexts/AuthContext';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
} from '@/components/ui/dialog';
import {
  Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,
} from '@/components/ui/tooltip';
```

**Data fetching pattern** (ConsultaNFeSaidas.tsx lines 408-421 — auth headers built from `useAuth()`, react-query with `enabled: !!token && !!companyId`):
```typescript
const { token, companyId } = useAuth();
const authHeaders = {
  Authorization: `Bearer ${token}`,
  'X-Company-ID': companyId || '',
};

const { data, isLoading, refetch } = useQuery<{ total: number; items: ComparisonItemRow[] }>({
  queryKey: ['fiscal-comparison', companyId],
  queryFn: async () => {
    const res = await fetch('/api/fiscal-comparison', { headers: authHeaders });
    if (!res.ok) throw new Error(res.statusText);
    return res.json();
  },
  enabled: !!token && !!companyId,
});
```

**Divergence badge pattern to adapt** (ConsultaNFeSaidas.tsx lines 123-161 — `FISCAL_STATUS_META` + `FiscalStatusBadge`; per D-10/discretion, extend with a 3-way OK/Divergente/Não calculado map using the same `bg-X-50/text-X-700/border-X-200` convention):
```typescript
const FISCAL_STATUS_META: Record<string, { label: string; className: string; tooltip: string }> = {
  ok: {
    label: 'OK',
    className: 'bg-green-50 text-green-700 border-green-200',
    tooltip: 'Grupo fiscal encontrado e pacote fiscal executado com sucesso.',
  },
  sem_grupo_fiscal: {
    label: 'Sem grupo fiscal',
    className: 'bg-yellow-50 text-yellow-700 border-yellow-200',
    tooltip: 'Produto não encontrado em PROD/PRODB — grupo fiscal não pôde ser determinado.',
  },
  error: {
    label: 'Erro no cálculo',
    className: 'bg-red-50 text-red-700 border-red-200',
    tooltip: 'Falha ao executar o pacote fiscal.',
  },
};

function FiscalStatusBadge({ status, errorMessage }: { status: string; errorMessage?: string }) {
  if (!status) {
    return <span className="text-xs text-muted-foreground">—</span>;
  }
  const meta = FISCAL_STATUS_META[status] ?? {
    label: status,
    className: 'bg-muted text-muted-foreground border-muted',
    tooltip: status,
  };
  const tooltipText = status === 'error' && errorMessage ? `Falha ao executar o pacote fiscal: ${errorMessage}.` : meta.tooltip;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge variant="outline" className={`text-xs px-1.5 py-0 ${meta.className}`}>
          {meta.label}
        </Badge>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs text-xs">{tooltipText}</TooltipContent>
    </Tooltip>
  );
}
```
New "OK"/"Divergente" states are derived client-side (not from the backend `status` field) — compute as `Math.abs(esperado - calculado) !== 0` per pair (D-06), and `status !== 'ok'` → "Não calculado" bucket (D-10) takes precedence over divergence check.

**Currency formatting helper** (ConsultaNFeSaidas.tsx lines 166-169 — copy verbatim):
```typescript
function fmtBRL(v: number | null | undefined, dash = '—'): string {
  if (v == null) return dash;
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}
```

**Dialog detail helpers to reuse verbatim** (ConsultaNFeSaidas.tsx lines 191-218 — `Linha`, `LinhaBRL`, `Secao`):
```typescript
function Linha({ label, value }: { label: string; value: string | number | null | undefined }) {
  return (
    <div className="flex justify-between py-0.5 border-b border-dashed last:border-0">
      <span className="text-xs text-muted-foreground w-36 shrink-0">{label}</span>
      <span className="text-xs font-bold text-right">{value ?? '—'}</span>
    </div>
  );
}

function LinhaBRL({ label, value }: { label: string; value: number | null | undefined }) {
  return (
    <div className="flex justify-between py-0.5 border-b border-dashed last:border-0">
      <span className="text-xs text-muted-foreground w-36 shrink-0">{label}</span>
      <span className="text-xs font-bold text-right">{fmtBRL(value, '—')}</span>
    </div>
  );
}

function Secao({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-2">
      <h3 className="text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1 pb-0.5 border-b">
        {title}
      </h3>
      {children}
    </div>
  );
}
```
For D-05 (Esperado | Calculado | Diferença as 3 separate columns) and D-07 ("Só calculado" section), extend with a variant that renders 3 values in one row instead of `Linha`'s 2 (label/value):
```typescript
function LinhaComparativa({ label, esperado, calculado }: { label: string; esperado: number; calculado: number | null }) {
  const diff = calculado == null ? null : calculado - esperado;
  const divergente = diff != null && diff !== 0;
  return (
    <div className={`grid grid-cols-4 gap-2 py-0.5 border-b border-dashed last:border-0 text-xs ${divergente ? 'bg-red-50' : ''}`}>
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-bold">{fmtBRL(esperado)}</span>
      <span className="text-right font-bold">{calculado == null ? '—' : fmtBRL(calculado)}</span>
      <span className={`text-right font-bold ${divergente ? 'text-red-700' : ''}`}>{diff == null ? '—' : fmtBRL(diff)}</span>
    </div>
  );
}
```

**Dialog structure pattern** (ConsultaNFeSaidas.tsx lines 220-393 — `DetalheNFe`: `useQuery` inside the Dialog component keyed by the row id, loading/error states, `Dialog open onOpenChange={onClose}` + `DialogContent` + `DialogHeader`/`DialogTitle`, sections built from `Secao`):
```typescript
function DetalheItem({ id, onClose, authHeaders }: {
  id: string; onClose: () => void; authHeaders: Record<string, string>;
}) {
  const { data, isLoading, isError } = useQuery<ComparisonItemDetail>({
    queryKey: ['fiscal-comparison-item', id],
    queryFn: async () => {
      const res = await fetch(`/api/fiscal-comparison/${id}`, { headers: authHeaders });
      if (!res.ok) throw new Error(res.statusText);
      return res.json();
    },
  });
  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        {isLoading && <p className="text-sm text-muted-foreground text-center py-8">Carregando...</p>}
        {isError && <p className="text-sm text-red-600 text-center py-8">Erro ao carregar detalhe do item.</p>}
        {data && (
          <>
            <DialogHeader><DialogTitle className="text-sm">...</DialogTitle></DialogHeader>
            <Secao title="Comparação — Esperado vs. Calculado">{/* LinhaComparativa x4 pairs */}</Secao>
            <Secao title="Só calculado (sem par no XML)">{/* full_result fields, D-07 */}</Secao>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
```
Note D-03: this can be a lighter inline expansion of the row instead of a second network round-trip if the list payload already carries `full_result` — Claude's Discretion notes list "paginação/virtualização" as open; if the list endpoint already returns everything needed per item, prefer opening the Dialog from already-fetched row data (no second fetch) to avoid an extra endpoint. Decide during planning based on payload size.

**List table structure + row click → Dialog pattern** (ConsultaNFeSaidas.tsx lines 561-619):
```typescript
<TableBody>
  {displayItems.map(row => (
    <TableRow key={row.id} className="cursor-pointer hover:bg-muted/50 h-8" onClick={() => setSelectedId(row.id)}>
      ...
    </TableRow>
  ))}
</TableBody>
...
{selectedId && (
  <DetalheItem id={selectedId} onClose={() => setSelectedId(null)} authHeaders={authHeaders} />
)}
```

**Summary cards pattern (D-09 global cards)** (ConsultaNFeSaidas.tsx lines 529-544 — `grid grid-cols-2 md:grid-cols-4 gap-2` of small `Card`s):
```typescript
{displayItems.length > 0 && (
  <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
    {[
      { label: 'OK',           value: countOK },
      { label: 'Divergente',   value: countDivergente },
      { label: 'Não calculado', value: countNaoCalculado },
      { label: 'Total itens',  value: displayItems.length },
    ].map(c => (
      <Card key={c.label} className="p-2">
        <p className="text-xs text-muted-foreground">{c.label}</p>
        <p className="text-sm font-bold mt-0.5">{c.value}</p>
      </Card>
    ))}
  </div>
)}
```

**Client-side filter toggle pattern (D-08 "só divergentes")** — adapt the existing client-side filter approach (ConsultaNFeSaidas.tsx lines 431-451, `displayItems = useMemo(...)` over raw `items`) using a boolean toggle state instead of text/date inputs:
```typescript
const [somenteDivergentes, setSomenteDivergentes] = useState(false);

const displayItems = useMemo(() => {
  return items.filter(item => {
    if (!somenteDivergentes) return true;
    if (item.fiscal_status !== 'ok') return false; // D-10: "não calculado" nunca aparece no filtro
    return isDivergente(item); // qualquer par com diff != 0 (D-06/D-08)
  });
}, [items, somenteDivergentes]);
```

---

### `frontend/src/lib/navigation.ts` (tab registration)

**Analog:** existing `modules.config.tabs` array (navigation.ts lines 17-25)

**Pattern to copy** (add one entry to the `tabs` array, same object shape as `{ label, path }`):
```typescript
export const modules: Record<string, ModuleConfig> = {
  config: {
    label: 'Configurações',
    tabs: [
      { label: 'Credenciais ERP',  path: '/config/erp-bridge',         adminOnly: true },
      { label: 'Config ERP',       path: '/importacoes/erp-bridge',     adminOnly: true },
      { label: 'Importar XMLs',    path: '/importacoes/xmls-saida' },
      { label: 'Notas Importadas', path: '/importacoes/notas-saida' },
      { label: 'Comparação Fiscal', path: '/importacoes/comparacao-fiscal' }, // NEW (D-01)
      { label: 'Ambiente',         path: '/config/ambiente' },
      { label: 'Gestores',         path: '/config/gestores' },
      { label: 'Usuários',         path: '/config/usuarios',            adminOnly: true },
    ],
  },
}
```
Note: `getActiveModule()` (navigation.ts lines 29-33) already treats every `/importacoes/*` path as the `config` module — no change needed there since the new path also starts with `/importacoes/`.

---

### `frontend/src/App.tsx` (route registration)

**Analog:** App.tsx lines 13-14 (import) and line 64 (route)

**Pattern to copy**:
```typescript
import ComparacaoFiscal from './pages/ComparacaoFiscal'
...
<Route path="/importacoes/comparacao-fiscal" element={<ProtectedRoute><ComparacaoFiscal /></ProtectedRoute>} />
```
Placed inside the protected `<Routes>` block (App.tsx lines 56-65), same as `ConsultaNFeSaidas`. Not an `AdminRoute` — comparison screen is available to all authenticated users, same as "Notas Importadas".

---

## Shared Patterns

### Backend: company scoping via JWT (never accept company_id from client)
**Source:** `backend/handlers/erp_bridge.go` lines 43-50, used throughout `nfe_saidas.go` and `fiscal_execution.go`
**Apply to:** `fiscal_comparison.go` — every query must include `WHERE i.company_id = $1` (or via JOIN on `n.company_id`) using `companyID` from `erpBridgeGetCompany(db, r)`.
```go
companyID, err := erpBridgeGetCompany(db, r)
if err != nil {
    jsonErr(w, http.StatusUnauthorized, "Não autenticado")
    return
}
```

### Backend: error handling / never leak raw error to client
**Source:** `backend/handlers/nfe_saidas.go` (repeated pattern, e.g. lines 624-628, 750-754)
**Apply to:** `fiscal_comparison.go`
```go
if err != nil {
    log.Printf("<HandlerName> query error (company %s): %v", companyID, err)
    jsonErr(w, http.StatusInternalServerError, "Erro ao consultar <recurso>")
    return
}
```
`jsonErr` is a shared helper already defined in the `handlers` package (used by all handlers) — no need to redefine.

### Backend: route registration via withAuth
**Source:** `backend/main.go` lines 212-223, 350-358
**Apply to:** new `/api/fiscal-comparison` route — `withAuth(handlers.FiscalComparisonListHandler, "")` (empty role = any authenticated user, same tier as `/api/nfe-saidas`).

### Frontend: FiscalStatusBadge color convention (verde/âmbar/vermelho)
**Source:** `frontend/src/pages/ConsultaNFeSaidas.tsx` lines 123-161
**Apply to:** the new page's "OK"/"Divergente"/"Não calculado" badge — reuse `bg-green-50/text-green-700/border-green-200` (OK), `bg-yellow-50/...` (não calculado, matching "sem_grupo_fiscal" semantics), `bg-red-50/text-red-700/border-red-200` (Divergente). Per D-06, "divergente" should use the red variant (not amber) since it's the primary alert of this validator.

### Frontend: react-query fetch + auth headers convention
**Source:** `frontend/src/pages/ConsultaNFeSaidas.tsx` lines 408-421
**Apply to:** all new data fetching in `ComparacaoFiscal.tsx` — `queryKey` includes `companyId`, headers built once from `useAuth()`, `enabled: !!token && !!companyId` guard.

### Frontend: Dialog-based item detail (not separate page)
**Source:** `frontend/src/pages/ConsultaNFeSaidas.tsx` lines 220-393 (`DetalheNFe`)
**Apply to:** `DetalheItem` in `ComparacaoFiscal.tsx` per D-03 — same `Dialog`/`DialogContent`/`DialogHeader` shadcn structure, `max-w-3xl max-h-[85vh] overflow-y-auto`.

## No Analog Found

None. All files in scope have a direct structural analog already in the codebase (this phase is explicitly a "read and present already-persisted data" phase — no new schema, no new external integration, so the existing NFe list/detail + fiscal execution status patterns cover 100% of the required roles).

One open finding to flag to the planner (per CONTEXT.md "Claude's Discretion" item): DIFAL (`valor_icms_partilha_destino`) and FCP (`valor_icms_pobreza`) exist as *calculated* columns in `fiscal_execution_items`, but `nfe_saidas_itens` (migration 007) has **no per-item DIFAL/FCP column** — only nota-level `v_fcp`/`v_fcp_st` exist in `nfe_saidas` (migration for that table, referenced but not re-read here since out of Phase 3 scope). This confirms D-07's assumption: DIFAL/FCP belong in the Dialog's "Só calculado" section with no expected-value counterpart at the item grain.

## Metadata

**Analog search scope:** `backend/handlers/`, `backend/services/`, `backend/migrations/`, `backend/main.go`, `frontend/src/pages/`, `frontend/src/lib/`, `frontend/src/App.tsx`
**Files scanned:** `nfe_saidas.go` (801 lines, read in full), `oracle_fiscal.go` (380 lines, read in full), `fiscal_execution.go` (415 lines, read in full), `007_nfe_saidas_itens.sql`, `008_fiscal_execution_items.sql`, `009_nfe_saidas_itens_desconto.sql` (all read in full), `ConsultaNFeSaidas.tsx` (622 lines, read in full), `navigation.ts` (33 lines, read in full), `App.tsx` (95 lines, read in full), `main.go` (targeted grep + read of route block, lines 205-254, 350-358)
**Pattern extraction date:** 2026-07-02
