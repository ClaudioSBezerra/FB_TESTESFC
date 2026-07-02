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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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

// Contrato de GET /api/fiscal-comparison/{id} (Plano 03-02) — os mesmos pares
// esperado x calculado do item + campos "só calculado" (DIFAL/FCP/full_result).
export interface ComparisonItemDetail {
  item_id: string;
  n_item: number;
  x_prod: string;
  ncm: string;
  cfop: string;
  nfe_id: string;
  numero_nfe: string;
  serie: string;
  dest_nome: string;

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

  percentual_difal: number | null;
  valor_icms_partilha_destino: number | null; // DIFAL
  valor_icms_pobreza: number | null; // FCP

  grupo_fiscal_codigo: string;
  fiscal_status: string;
  error_message: string;
  full_result: Record<string, unknown>;
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
// Helpers de layout do Dialog de detalhe (padrão ConsultaNFeSaidas.tsx:191-218)
// ---------------------------------------------------------------------------
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

// Linha de comparação com 3 valores (Esperado | Calculado | Diferença) — D-05.
// Destaca em vermelho quando a diferença for != 0 (D-06).
function LinhaComparativa({
  label,
  esperado,
  calculado,
}: {
  label: string;
  esperado: number;
  calculado: number | null;
}) {
  const diff = pairDiff(esperado, calculado);
  const divergente = diff !== null && diff !== 0;
  return (
    <div
      className={`grid grid-cols-4 gap-2 py-0.5 border-b border-dashed last:border-0 text-xs ${
        divergente ? 'bg-red-50' : ''
      }`}
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-bold">{fmtBRL(esperado)}</span>
      <span className="text-right font-bold">{calculado == null ? '—' : fmtBRL(calculado)}</span>
      <span className={`text-right font-bold ${divergente ? 'text-red-700' : ''}`}>
        {diff == null ? '—' : fmtBRL(diff)}
      </span>
    </div>
  );
}

// Rótulos amigáveis para os campos de maior valor de auditoria em full_result
// (~88 campos do pacote fiscal — nomes conforme FiscalResult em oracle_fiscal.go).
// Escolha de campos/labels é discretion (03-02-PLAN.md); os demais campos de
// full_result são renderizados de forma genérica (chave → valor) logo abaixo.
const FULL_RESULT_LABELS: Record<string, string> = {
  ValorIbsUF: 'IBS UF',
  AliquotaIbsUF: 'Alíquota IBS UF (%)',
  ValorIbsMUN: 'IBS Município',
  AliquotaIbsMUN: 'Alíquota IBS Município (%)',
  ValorCbs: 'CBS',
  AliquotaCbs: 'Alíquota CBS (%)',
  AliquotaImposto: 'Alíquota ICMS (%)',
  Mva: 'MVA (%)',
  AliquotaUFDestino: 'Alíquota UF Destino (%)',
  ValorIcmsUFDestino: 'Valor ICMS UF Destino',
  PercentualPartilhaDestino: '% Partilha Destino',
  AliquotaFundoPobreza: 'Alíquota Fundo Pobreza (%)',
};

function fmtRawValue(v: unknown): string {
  if (v == null || v === '') return '—';
  if (typeof v === 'number') return v.toLocaleString('pt-BR');
  return String(v);
}

// ---------------------------------------------------------------------------
// Dialog de detalhe do item (D-03/D-05/D-07)
// ---------------------------------------------------------------------------
function DetalheItem({
  id,
  onClose,
  authHeaders,
  items,
}: {
  id: string;
  onClose: () => void;
  authHeaders: Record<string, string>;
  items: ComparisonItemRow[];
}) {
  const { data, isLoading, isError } = useQuery<ComparisonItemDetail>({
    queryKey: ['fiscal-comparison-item', id],
    queryFn: async () => {
      const res = await fetch(`/api/fiscal-comparison/${id}`, { headers: authHeaders });
      if (!res.ok) throw new Error(res.statusText);
      return res.json();
    },
  });

  // Resumo por nota (CMP-04/D-09) — calculado client-side a partir dos itens
  // já carregados na lista, filtrados pela mesma nfe_id do item selecionado.
  const resumoNota = useMemo(() => {
    if (!data) return null;
    const itensDaNota = items.filter(i => i.nfe_id === data.nfe_id);
    return {
      total: itensDaNota.length,
      ok: itensDaNota.filter(i => itemBucket(i) === 'ok').length,
      divergente: itensDaNota.filter(i => itemBucket(i) === 'divergente').length,
      nao_calculado: itensDaNota.filter(i => itemBucket(i) === 'nao_calculado').length,
    };
  }, [data, items]);

  const fullResult = data?.full_result ?? {};

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        {isLoading && (
          <p className="text-sm text-muted-foreground text-center py-8">Carregando...</p>
        )}
        {isError && (
          <p className="text-sm text-red-600 text-center py-8">Erro ao carregar detalhe do item.</p>
        )}
        {data && (
          <>
            <DialogHeader>
              <DialogTitle className="text-sm">
                Item {data.n_item} — {data.x_prod}
                <div className="text-xs font-normal text-muted-foreground mt-0.5">
                  NF-e {data.numero_nfe}/{data.serie} · {data.dest_nome || '—'} · NCM{' '}
                  {data.ncm || '—'} · CFOP {data.cfop || '—'}
                </div>
              </DialogTitle>
            </DialogHeader>

            <div className="space-y-1 mt-1">
              {resumoNota && (
                <Secao title="Resumo da nota">
                  <Linha label="Total de itens" value={resumoNota.total} />
                  <Linha label="OK" value={resumoNota.ok} />
                  <Linha label="Divergente" value={resumoNota.divergente} />
                  <Linha label="Não calculado" value={resumoNota.nao_calculado} />
                </Secao>
              )}

              {data.fiscal_status !== 'ok' ? (
                <p className="text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded p-2">
                  Item ainda sem cálculo fiscal — nada a comparar.
                  {data.error_message ? ` Detalhe: ${data.error_message}` : ''}
                </p>
              ) : (
                <>
                  <Secao title="Comparação — Esperado vs. Calculado">
                    <div className="grid grid-cols-4 gap-2 py-0.5 text-xs font-bold text-muted-foreground border-b">
                      <span></span>
                      <span className="text-right">Esperado</span>
                      <span className="text-right">Calculado</span>
                      <span className="text-right">Diferença</span>
                    </div>
                    <LinhaComparativa label="Base ICMS" esperado={data.esp_bc_icms} calculado={data.calc_bc_icms} />
                    <LinhaComparativa label="Valor ICMS" esperado={data.esp_icms} calculado={data.calc_icms} />
                    <LinhaComparativa label="Base ICMS-ST" esperado={data.esp_bc_st} calculado={data.calc_bc_st} />
                    <LinhaComparativa label="Valor ICMS-ST" esperado={data.esp_st} calculado={data.calc_st} />
                    <LinhaComparativa label="Base PIS" esperado={data.esp_bc_pis} calculado={data.calc_bc_pis} />
                    <LinhaComparativa label="Valor PIS" esperado={data.esp_pis} calculado={data.calc_pis} />
                    <LinhaComparativa
                      label="Base COFINS"
                      esperado={data.esp_bc_cofins}
                      calculado={data.calc_bc_cofins}
                    />
                    <LinhaComparativa label="Valor COFINS" esperado={data.esp_cofins} calculado={data.calc_cofins} />
                  </Secao>

                  <Secao title="Só calculado (sem par no XML)">
                    <LinhaBRL label="DIFAL (partilha destino)" value={data.valor_icms_partilha_destino} />
                    <LinhaBRL label="FCP (pobreza)" value={data.valor_icms_pobreza} />
                    <Linha
                      label="% DIFAL"
                      value={data.percentual_difal != null ? `${data.percentual_difal}%` : '—'}
                    />
                    <Linha label="Grupo Fiscal" value={data.grupo_fiscal_codigo || '—'} />
                    {Object.entries(FULL_RESULT_LABELS).map(([key, label]) => {
                      const raw = fullResult[key];
                      if (raw == null || raw === '') return null;
                      return <Linha key={key} label={label} value={fmtRawValue(raw)} />;
                    })}
                  </Secao>

                  <Secao title="Demais campos do pacote fiscal (full_result)">
                    {Object.entries(fullResult)
                      .filter(([key, v]) => !(key in FULL_RESULT_LABELS) && v != null && v !== '')
                      .map(([key, v]) => (
                        <Linha key={key} label={key} value={fmtRawValue(v)} />
                      ))}
                  </Secao>
                </>
              )}
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Página principal
// ---------------------------------------------------------------------------
export default function ComparacaoFiscal() {
  const { token, companyId } = useAuth();

  const [somenteDivergentes, setSomenteDivergentes] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);

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

  // CMP-04/D-09: cards derivam de displayItems (respeitam o filtro "só divergentes" atual)
  const countOK = useMemo(() => displayItems.filter(i => itemBucket(i) === 'ok').length, [displayItems]);
  const countDivergente = useMemo(
    () => displayItems.filter(i => itemBucket(i) === 'divergente').length,
    [displayItems]
  );
  const countNaoCalculado = useMemo(
    () => displayItems.filter(i => itemBucket(i) === 'nao_calculado').length,
    [displayItems]
  );

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

      {/* ── Resumo (D-09) — respeita o filtro "só divergentes" atual ── */}
      {items.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
          {[
            { label: 'Total itens', value: displayItems.length },
            { label: 'OK', value: countOK },
            { label: 'Divergente', value: countDivergente },
            { label: 'Não calculado', value: countNaoCalculado },
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
                        className={`h-8 cursor-pointer hover:bg-muted/50 ${
                          itemBucket(item) === 'divergente' ? 'bg-red-50' : ''
                        }`}
                        onClick={() => setSelectedId(item.item_id)}
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

      {selectedId && (
        <DetalheItem
          id={selectedId}
          onClose={() => setSelectedId(null)}
          authHeaders={authHeaders}
          items={items}
        />
      )}
    </div>
  );
}
