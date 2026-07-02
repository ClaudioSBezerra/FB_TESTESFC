import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '@/contexts/AuthContext';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { Search, Filter } from 'lucide-react';

// ---------------------------------------------------------------------------
// Types — espelha o contrato JSON de backend/handlers/fiscal_comparison.go
// ---------------------------------------------------------------------------
export interface ComparisonItemRow {
  item_id: string;
  n_item: number;
  x_prod: string;
  ncm: string;
  cfop: string;
  nfe_id: string;
  numero_nfe: string;
  serie: string;
  dest_nome: string;
  dest_cnpj_cpf: string;
  data_emissao: string;

  esp_bc_icms: number;
  esp_icms: number;
  esp_bc_st: number;
  esp_st: number;
  esp_bc_pis: number;
  esp_pis: number;
  esp_bc_cofins: number;
  esp_cofins: number;

  calc_bc_icms: number | null;
  calc_icms: number | null;
  calc_bc_st: number | null;
  calc_st: number | null;
  calc_bc_pis: number | null;
  calc_pis: number | null;
  calc_bc_cofins: number | null;
  calc_cofins: number | null;

  fiscal_status: string; // '' | 'pending' | 'ok' | 'error' | 'sem_grupo_fiscal'
  error_message: string;
}

type ItemBucket = 'ok' | 'divergente' | 'nao_calculado';

// ---------------------------------------------------------------------------
// Lógica de comparação (exportada — reutilizada pelo Plano 03-02)
// ---------------------------------------------------------------------------

/** Diferença calculado - esperado. null quando ainda não há valor calculado. */
export function pairDiff(esperado: number, calculado: number | null): number | null {
  return calculado == null ? null : calculado - esperado;
}

/**
 * D-06: qualquer diferença ≠ 0 em qualquer um dos 4 pares comparáveis conta
 * como divergência — sem tolerância de arredondamento no v1.
 * D-10: item ainda sem cálculo concluído (fiscal_status !== 'ok') nunca é
 * classificado como divergente — é "Não calculado".
 */
export function isDivergente(item: ComparisonItemRow): boolean {
  if (item.fiscal_status !== 'ok') return false;

  const pares: Array<[number, number | null]> = [
    [item.esp_icms, item.calc_icms],
    [item.esp_st, item.calc_st],
    [item.esp_pis, item.calc_pis],
    [item.esp_cofins, item.calc_cofins],
  ];

  return pares.some(([esperado, calculado]) => {
    const diff = pairDiff(esperado, calculado);
    return diff !== null && diff !== 0;
  });
}

export function itemBucket(item: ComparisonItemRow): ItemBucket {
  if (item.fiscal_status !== 'ok') return 'nao_calculado';
  return isDivergente(item) ? 'divergente' : 'ok';
}

// ---------------------------------------------------------------------------
// Badge de 3 estados (OK / Divergente / Não calculado)
// ---------------------------------------------------------------------------
const BUCKET_META: Record<ItemBucket, { label: string; className: string; tooltip: string }> = {
  ok: {
    label: 'OK',
    className: 'bg-green-50 text-green-700 border-green-200',
    tooltip: 'Pacote fiscal reproduziu exatamente os valores esperados do XML.',
  },
  divergente: {
    label: 'Divergente',
    className: 'bg-red-50 text-red-700 border-red-200',
    tooltip: 'Ao menos um imposto (ICMS, ICMS-ST, PIS ou COFINS) tem diferença ≠ 0 entre esperado e calculado.',
  },
  nao_calculado: {
    label: 'Não calculado',
    className: 'bg-yellow-50 text-yellow-700 border-yellow-200',
    tooltip: 'O cálculo fiscal deste item ainda não foi concluído com sucesso.',
  },
};

function ComparisonBucketBadge({ item }: { item: ComparisonItemRow }) {
  const bucket = itemBucket(item);
  const meta = BUCKET_META[bucket];
  const tooltipText =
    bucket === 'nao_calculado' && item.error_message
      ? `${meta.tooltip} Detalhe: ${item.error_message}`
      : meta.tooltip;
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function fmtBRL(v: number | null | undefined, dash = '—'): string {
  if (v == null) return dash;
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

// ---------------------------------------------------------------------------
// Célula de imposto: Esperado | Calculado | Diferença (D-05)
// ---------------------------------------------------------------------------
function DiffCells({ esperado, calculado }: { esperado: number; calculado: number | null }) {
  const diff = pairDiff(esperado, calculado);
  const divergente = diff !== null && diff !== 0;
  return (
    <>
      <TableCell className="py-1 px-2 text-xs text-right">{fmtBRL(esperado)}</TableCell>
      <TableCell className="py-1 px-2 text-xs text-right">{calculado == null ? '—' : fmtBRL(calculado)}</TableCell>
      <TableCell className={`py-1 px-2 text-xs text-right ${divergente ? 'text-red-700 font-bold' : ''}`}>
        {diff == null ? '—' : fmtBRL(diff)}
      </TableCell>
    </>
  );
}

// ---------------------------------------------------------------------------
// Página principal
// ---------------------------------------------------------------------------
export default function ComparacaoFiscal() {
  const { token, companyId } = useAuth();

  const [somenteDivergentes, setSomenteDivergentes] = useState(false);

  const authHeaders = {
    Authorization: `Bearer ${token}`,
    'X-Company-ID': companyId || '',
  };

  const { data, isLoading, isError, refetch } = useQuery<{ total: number; items: ComparisonItemRow[] }>({
    queryKey: ['fiscal-comparison', companyId],
    queryFn: async () => {
      const res = await fetch('/api/fiscal-comparison', { headers: authHeaders });
      if (!res.ok) throw new Error(res.statusText);
      return res.json();
    },
    enabled: !!token && !!companyId,
  });

  const items = useMemo(() => data?.items ?? [], [data]);

  const displayItems = useMemo(() => {
    if (!somenteDivergentes) return items;
    // D-10: "não calculado" nunca aparece no filtro "só divergentes"
    return items.filter(item => itemBucket(item) === 'divergente');
  }, [items, somenteDivergentes]);

  const countOK = useMemo(() => items.filter(i => itemBucket(i) === 'ok').length, [items]);
  const countDivergente = useMemo(() => items.filter(i => itemBucket(i) === 'divergente').length, [items]);
  const countNaoCalculado = useMemo(() => items.filter(i => itemBucket(i) === 'nao_calculado').length, [items]);

  const handleRefetch = () => {
    refetch().catch(err => toast.error('Erro ao buscar comparação fiscal: ' + String(err)));
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Comparação Fiscal</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Item a item, imposto a imposto: valor esperado (do XML) vs. calculado pelo pacote
          fiscal. Itens divergentes são destacados em vermelho.
        </p>
      </div>

      {/* ── Ações / Filtro ── */}
      <Card>
        <CardContent className="pt-4 space-y-3">
          <div className="flex flex-wrap gap-3 items-end">
            <Button size="sm" onClick={handleRefetch} disabled={isLoading}>
              <Search className="h-3 w-3 mr-1" />
              {isLoading ? 'Carregando...' : 'Recarregar'}
            </Button>
            <Button
              size="sm"
              variant={somenteDivergentes ? 'default' : 'outline'}
              onClick={() => setSomenteDivergentes(v => !v)}
            >
              <Filter className="h-3 w-3 mr-1" />
              Só divergentes
            </Button>
            <span className="text-xs text-muted-foreground ml-auto self-end">
              {displayItems.length} de {items.length} item(ns)
            </span>
          </div>
        </CardContent>
      </Card>

      {/* ── Resumo (D-09) ── */}
      {items.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
          {[
            { label: 'OK', value: countOK },
            { label: 'Divergente', value: countDivergente },
            { label: 'Não calculado', value: countNaoCalculado },
            { label: 'Total itens', value: items.length },
          ].map(c => (
            <Card key={c.label} className="p-2">
              <p className="text-xs text-muted-foreground">{c.label}</p>
              <p className="text-sm font-bold mt-0.5">{c.value}</p>
            </Card>
          ))}
        </div>
      )}

      {/* ── Tabela ── */}
      <Card>
        <CardHeader className="py-2 px-4">
          <CardTitle className="text-xs text-muted-foreground font-normal">
            ICMS, ICMS-ST, PIS e COFINS — Esperado | Calculado | Diferença por item
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isError ? (
            <div className="text-center py-8">
              <p className="text-sm font-bold text-red-600">Erro ao carregar comparação fiscal</p>
              <p className="text-xs text-muted-foreground mt-1">Tente recarregar a página.</p>
            </div>
          ) : displayItems.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-sm font-bold">Nenhum item para comparar</p>
              <p className="text-xs text-muted-foreground mt-1">
                {isLoading
                  ? 'Carregando...'
                  : 'Importe e execute notas na Fase de importação.'}
              </p>
            </div>
          ) : (
            <TooltipProvider delayDuration={200}>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="hover:bg-transparent">
                      <TableHead rowSpan={2} className="py-1.5 px-2 text-xs align-bottom">Nº Nota</TableHead>
                      <TableHead rowSpan={2} className="py-1.5 px-2 text-xs align-bottom">Cliente</TableHead>
                      <TableHead rowSpan={2} className="py-1.5 px-2 text-xs align-bottom">Item</TableHead>
                      <TableHead rowSpan={2} className="py-1.5 px-2 text-xs align-bottom">Produto</TableHead>
                      <TableHead rowSpan={2} className="py-1.5 px-2 text-xs text-center align-bottom">Status</TableHead>
                      <TableHead colSpan={3} className="py-1 px-2 text-xs text-center border-l">ICMS</TableHead>
                      <TableHead colSpan={3} className="py-1 px-2 text-xs text-center border-l">ICMS-ST</TableHead>
                      <TableHead colSpan={3} className="py-1 px-2 text-xs text-center border-l">PIS</TableHead>
                      <TableHead colSpan={3} className="py-1 px-2 text-xs text-center border-l">COFINS</TableHead>
                    </TableRow>
                    <TableRow className="hover:bg-transparent">
                      <TableHead className="py-1 px-2 text-xs text-right border-l">Esperado</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right">Calculado</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right">Diferença</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right border-l">Esperado</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right">Calculado</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right">Diferença</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right border-l">Esperado</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right">Calculado</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right">Diferença</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right border-l">Esperado</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right">Calculado</TableHead>
                      <TableHead className="py-1 px-2 text-xs text-right">Diferença</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {displayItems.map(item => (
                      <TableRow
                        key={item.item_id}
                        className={`h-8 ${itemBucket(item) === 'divergente' ? 'bg-red-50' : ''}`}
                      >
                        <TableCell className="py-1 px-2 text-xs font-mono whitespace-nowrap">
                          {item.numero_nfe}/{item.serie}
                        </TableCell>
                        <TableCell className="py-1 px-2 text-xs max-w-[160px] truncate" title={item.dest_nome}>
                          {item.dest_nome || '—'}
                        </TableCell>
                        <TableCell className="py-1 px-2 text-xs text-center">{item.n_item}</TableCell>
                        <TableCell className="py-1 px-2 text-xs max-w-[180px] truncate" title={item.x_prod}>
                          {item.x_prod}
                        </TableCell>
                        <TableCell className="py-1 px-2 text-center">
                          <ComparisonBucketBadge item={item} />
                        </TableCell>
                        <DiffCells esperado={item.esp_icms} calculado={item.calc_icms} />
                        <DiffCells esperado={item.esp_st} calculado={item.calc_st} />
                        <DiffCells esperado={item.esp_pis} calculado={item.calc_pis} />
                        <DiffCells esperado={item.esp_cofins} calculado={item.calc_cofins} />
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </TooltipProvider>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
