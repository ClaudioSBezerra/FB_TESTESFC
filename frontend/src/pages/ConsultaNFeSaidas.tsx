import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/contexts/AuthContext';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { Search, X, Play } from 'lucide-react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface NfeSaidaRow {
  id: string;
  chave_nfe: string;
  modelo: number;
  serie: string;
  numero_nfe: string;
  data_emissao: string;
  mes_ano: string;
  nat_op: string;
  emit_cnpj: string;
  emit_nome: string;
  emit_uf: string;
  emit_municipio: string;
  dest_cnpj_cpf: string;
  dest_nome: string;
  dest_uf: string;
  dest_c_mun: string;
  v_bc: number;
  v_icms: number;
  v_icms_deson: number;
  v_fcp: number;
  v_bc_st: number;
  v_st: number;
  v_fcp_st: number;
  v_fcp_st_ret: number;
  v_prod: number;
  v_frete: number;
  v_seg: number;
  v_desc: number;
  v_ii: number;
  v_ipi: number;
  v_ipi_devol: number;
  v_pis: number;
  v_cofins: number;
  v_outro: number;
  v_nf: number;
  v_bc_ibs_cbs: number | null;
  v_ibs_uf: number | null;
  v_ibs_mun: number | null;
  v_ibs: number | null;
  v_cred_pres_ibs: number | null;
  v_cbs: number | null;
  v_cred_pres_cbs: number | null;
}

interface NfeSaidaItemRow {
  id: string;
  n_item: number;
  c_prod: string;
  x_prod: string;
  ncm: string;
  cest: string;
  cfop: string;
  cst_icms: string;
  cst_orig: string;
  cst_pis: string;
  cst_cofins: string;
  v_prod: number;
  v_bc_icms: number;
  v_icms: number;
  v_bc_st: number;
  v_st: number;
  v_ipi: number;
  v_bc_pis: number;
  v_pis: number;
  v_bc_cofins: number;
  v_cofins: number;
  v_ibs: number;
  v_cbs: number;
  cclasstrib: string;
  fiscal_status: string; // '' | 'ok' | 'error' | 'sem_grupo_fiscal'
  fiscal_error_message: string;
}

interface NfeSaidaDetail {
  nfe: NfeSaidaRow;
  itens: NfeSaidaItemRow[];
}

interface FiscalExecutionSummary {
  total: number;
  ok: number;
  sem_grupo_fiscal: number;
  error: number;
}

// ---------------------------------------------------------------------------
// Badge de status de execução fiscal por item (ERP-03/FIS-03 — UI-SPEC linhas 116-136)
// ---------------------------------------------------------------------------
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function fmtBRL(v: number | null | undefined, dash = '—'): string {
  if (v == null) return dash;
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

function fmtCNPJ(v: string): string {
  if (!v) return '—';
  const d = v.replace(/\D/g, '');
  if (d.length === 14)
    return `${d.slice(0,2)}.${d.slice(2,5)}.${d.slice(5,8)}/${d.slice(8,12)}-${d.slice(12)}`;
  if (d.length === 11)
    return `${d.slice(0,3)}.${d.slice(3,6)}.${d.slice(6,9)}-${d.slice(9)}`;
  return v;
}

/** Converte "DD/MM/YYYY" → Date (para comparação de range) */
function parseDMY(s: string): Date | null {
  const m = s?.match(/^(\d{2})\/(\d{2})\/(\d{4})$/);
  if (!m) return null;
  return new Date(+m[3], +m[2] - 1, +m[1]);
}

// ---------------------------------------------------------------------------
// Detalhe da Nota (Dialog) — cabeçalho + itens com valores esperados do XML
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

function DetalheNFe({ id, onClose, authHeaders }: {
  id: string;
  onClose: () => void;
  authHeaders: Record<string, string>;
}) {
  const qc = useQueryClient();
  const { data, isLoading, isError } = useQuery<NfeSaidaDetail>({
    queryKey: ['nfe-saida-detail', id],
    queryFn: async () => {
      const res = await fetch(`/api/nfe-saidas/${id}`, { headers: authHeaders });
      if (!res.ok) throw new Error(res.statusText);
      return res.json();
    },
  });

  // Dispara o pipeline de execução fiscal (lookup grupo fiscal + pacote
  // fiscal) para todos os itens desta nota — ERP-02/FIS-01/FIS-02.
  const runFiscalMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/fiscal-execution/run', {
        method: 'POST',
        headers: { ...authHeaders, 'Content-Type': 'application/json' },
        body: JSON.stringify({ nfe_id: id }),
      });
      if (!res.ok) throw new Error(await res.text());
      return res.json() as Promise<FiscalExecutionSummary>;
    },
    onSuccess: (summary) => {
      toast.success(
        `Cálculo fiscal concluído: ${summary.total} item(ns) — ${summary.ok} ok, ${summary.sem_grupo_fiscal} sem grupo fiscal, ${summary.error} erro(s).`
      );
      qc.invalidateQueries({ queryKey: ['nfe-saida-detail', id] });
    },
    onError: (e: Error) => toast.error(`Erro ao executar cálculo fiscal: ${e.message}`),
  });

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        {isLoading && (
          <p className="text-sm text-muted-foreground text-center py-8">Carregando...</p>
        )}
        {isError && (
          <p className="text-sm text-red-600 text-center py-8">Erro ao carregar detalhe da nota.</p>
        )}
        {data && (
          <>
            <DialogHeader>
              <DialogTitle className="text-sm flex items-start justify-between gap-2">
                <div>
                  NF-e {data.nfe.modelo} · Série {data.nfe.serie} · Nº {data.nfe.numero_nfe}
                  <div className="text-xs font-normal text-muted-foreground mt-0.5 break-all">
                    Chave: {data.nfe.chave_nfe}
                  </div>
                </div>
                <Button
                  size="sm"
                  onClick={() => runFiscalMutation.mutate()}
                  disabled={runFiscalMutation.isPending}
                  className="shrink-0"
                >
                  <Play className="h-3 w-3 mr-1" />
                  {runFiscalMutation.isPending ? 'Calculando...' : 'Executar cálculo fiscal'}
                </Button>
              </DialogTitle>
            </DialogHeader>

            <div className="space-y-1 mt-1">
              <Secao title="Identificação">
                <Linha label="Modelo" value={data.nfe.modelo} />
                <Linha label="Série" value={data.nfe.serie} />
                <Linha label="Número" value={data.nfe.numero_nfe} />
                <Linha label="Data Emissão" value={data.nfe.data_emissao} />
                <Linha label="Mês/Ano" value={data.nfe.mes_ano} />
                <Linha label="Natureza Operação" value={data.nfe.nat_op} />
              </Secao>

              <Secao title="Emitente (Filial)">
                <Linha label="CNPJ" value={fmtCNPJ(data.nfe.emit_cnpj)} />
                <Linha label="Razão Social" value={data.nfe.emit_nome} />
                <Linha label="Município" value={data.nfe.emit_municipio} />
                <Linha label="UF" value={data.nfe.emit_uf} />
              </Secao>

              <Secao title="Destinatário (Cliente)">
                <Linha label="CNPJ/CPF" value={fmtCNPJ(data.nfe.dest_cnpj_cpf)} />
                <Linha label="Nome/Razão Social" value={data.nfe.dest_nome} />
                <Linha label="UF" value={data.nfe.dest_uf} />
                <Linha label="Município (IBGE)" value={data.nfe.dest_c_mun} />
              </Secao>

              <Secao title="ICMSTot — Totais da Nota (esperado, do XML)">
                <LinhaBRL label="vProd" value={data.nfe.v_prod} />
                <LinhaBRL label="vFrete" value={data.nfe.v_frete} />
                <LinhaBRL label="vSeg" value={data.nfe.v_seg} />
                <LinhaBRL label="vDesc" value={data.nfe.v_desc} />
                <LinhaBRL label="vII" value={data.nfe.v_ii} />
                <LinhaBRL label="vIPI" value={data.nfe.v_ipi} />
                <LinhaBRL label="vIPIDevol" value={data.nfe.v_ipi_devol} />
                <LinhaBRL label="vPIS" value={data.nfe.v_pis} />
                <LinhaBRL label="vCOFINS" value={data.nfe.v_cofins} />
                <LinhaBRL label="vOutro" value={data.nfe.v_outro} />
                <LinhaBRL label="vNF (Valor Total)" value={data.nfe.v_nf} />
                <LinhaBRL label="vBC (Base ICMS)" value={data.nfe.v_bc} />
                <LinhaBRL label="vICMS" value={data.nfe.v_icms} />
                <LinhaBRL label="vICMSDeson" value={data.nfe.v_icms_deson} />
                <LinhaBRL label="vFCP" value={data.nfe.v_fcp} />
                <LinhaBRL label="vBCST" value={data.nfe.v_bc_st} />
                <LinhaBRL label="vST" value={data.nfe.v_st} />
                <LinhaBRL label="vFCPST" value={data.nfe.v_fcp_st} />
                <LinhaBRL label="vFCPSTRet" value={data.nfe.v_fcp_st_ret} />
              </Secao>

              <Secao title="IBSCBSTot — Reforma Tributária">
                <LinhaBRL label="vBCIBSCBS (Base)" value={data.nfe.v_bc_ibs_cbs} />
                <LinhaBRL label="vIBSUF" value={data.nfe.v_ibs_uf} />
                <LinhaBRL label="vIBSMun" value={data.nfe.v_ibs_mun} />
                <LinhaBRL label="vIBS (Total)" value={data.nfe.v_ibs} />
                <LinhaBRL label="vCredPres IBS" value={data.nfe.v_cred_pres_ibs} />
                <LinhaBRL label="vCBS" value={data.nfe.v_cbs} />
                <LinhaBRL label="vCredPres CBS" value={data.nfe.v_cred_pres_cbs} />
              </Secao>

              <Secao title={`Itens (${data.itens.length}) — valores esperados do XML`}>
                {data.itens.length === 0 ? (
                  <p className="text-xs text-muted-foreground py-2">Nenhum item persistido para esta nota.</p>
                ) : (
                  <TooltipProvider delayDuration={200}>
                    <div className="overflow-x-auto">
                      <Table>
                        <TableHeader>
                          <TableRow className="hover:bg-transparent">
                            <TableHead className="py-1 px-2 text-xs">Item</TableHead>
                            <TableHead className="py-1 px-2 text-xs">Produto</TableHead>
                            <TableHead className="py-1 px-2 text-xs">NCM</TableHead>
                            <TableHead className="py-1 px-2 text-xs">CFOP</TableHead>
                            <TableHead className="py-1 px-2 text-xs text-right">vProd</TableHead>
                            <TableHead className="py-1 px-2 text-xs text-right">Base ICMS</TableHead>
                            <TableHead className="py-1 px-2 text-xs text-right">vICMS</TableHead>
                            <TableHead className="py-1 px-2 text-xs text-right">vPIS</TableHead>
                            <TableHead className="py-1 px-2 text-xs text-right">vCOFINS</TableHead>
                            <TableHead className="py-1 px-2 text-xs text-center">Status</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {data.itens.map(item => (
                            <TableRow key={item.id} className="h-8">
                              <TableCell className="py-1 px-2 text-xs text-center">{item.n_item}</TableCell>
                              <TableCell className="py-1 px-2 text-xs max-w-[180px] truncate" title={item.x_prod}>{item.x_prod}</TableCell>
                              <TableCell className="py-1 px-2 text-xs font-mono">{item.ncm || '—'}</TableCell>
                              <TableCell className="py-1 px-2 text-xs font-mono">{item.cfop || '—'}</TableCell>
                              <TableCell className="py-1 px-2 text-xs text-right">{fmtBRL(item.v_prod)}</TableCell>
                              <TableCell className="py-1 px-2 text-xs text-right">{fmtBRL(item.v_bc_icms)}</TableCell>
                              <TableCell className="py-1 px-2 text-xs text-right">{fmtBRL(item.v_icms)}</TableCell>
                              <TableCell className="py-1 px-2 text-xs text-right">{fmtBRL(item.v_pis)}</TableCell>
                              <TableCell className="py-1 px-2 text-xs text-right">{fmtBRL(item.v_cofins)}</TableCell>
                              <TableCell className="py-1 px-2 text-center">
                                <FiscalStatusBadge status={item.fiscal_status} errorMessage={item.fiscal_error_message} />
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  </TooltipProvider>
                )}
              </Secao>
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
export default function ConsultaNFeSaidas() {
  const { token, companyId } = useAuth();

  const [selectedId, setSelectedId] = useState<string | null>(null);

  // Filtros client-side
  const [filterCliente, setFilterCliente] = useState('');
  const [filterDataDe, setFilterDataDe] = useState('');
  const [filterDataAte, setFilterDataAte] = useState('');

  const authHeaders = {
    Authorization: `Bearer ${token}`,
    'X-Company-ID': companyId || '',
  };

  const { data, isLoading, refetch } = useQuery<{ total: number; items: NfeSaidaRow[] }>({
    queryKey: ['nfe-saidas', companyId],
    queryFn: async () => {
      const res = await fetch('/api/nfe-saidas', { headers: authHeaders });
      if (!res.ok) throw new Error(res.statusText);
      return res.json();
    },
    enabled: !!token && !!companyId,
  });

  const items = useMemo(() => data?.items ?? [], [data]);

  const clearFilters = () => {
    setFilterCliente('');
    setFilterDataDe('');
    setFilterDataAte('');
  };

  const displayItems = useMemo(() => {
    const dataDe  = filterDataDe  ? new Date(filterDataDe)  : null;
    const dataAte = filterDataAte ? new Date(filterDataAte) : null;

    return items.filter(r => {
      if (filterCliente) {
        const nomeOk = r.dest_nome?.toLowerCase().includes(filterCliente.toLowerCase());
        const cnpjOk = r.dest_cnpj_cpf?.replace(/\D/g, '').includes(filterCliente.replace(/\D/g, ''));
        if (!nomeOk && !cnpjOk) return false;
      }

      if (dataDe || dataAte) {
        const d = parseDMY(r.data_emissao);
        if (!d) return false;
        if (dataDe && d < dataDe) return false;
        if (dataAte && d > dataAte) return false;
      }

      return true;
    });
  }, [items, filterCliente, filterDataDe, filterDataAte]);

  const hasClientFilters = filterCliente || filterDataDe || filterDataAte;

  const totalVNF  = displayItems.reduce((s, r) => s + r.v_nf,           0);
  const totalICMS = displayItems.reduce((s, r) => s + r.v_icms,          0);
  const totalIBS  = displayItems.reduce((s, r) => s + (r.v_ibs  ?? 0),  0);
  const totalCBS  = displayItems.reduce((s, r) => s + (r.v_cbs  ?? 0),  0);

  const handleRefetch = () => {
    refetch().catch(err => toast.error('Erro ao buscar notas: ' + String(err)));
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Notas Importadas</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Consulta de NF-e e NFC-e de saída importadas via XML. Clique em uma linha para ver
          todos os dados e itens da nota.
        </p>
      </div>

      {/* ── Filtros ── */}
      <Card>
        <CardContent className="pt-4 space-y-3">
          <div className="flex flex-wrap gap-3 items-end">
            <Button size="sm" onClick={handleRefetch} disabled={isLoading}>
              <Search className="h-3 w-3 mr-1" />
              {isLoading ? 'Carregando...' : 'Recarregar'}
            </Button>
            {hasClientFilters && (
              <Button size="sm" variant="ghost" onClick={clearFilters}>
                <X className="h-3 w-3 mr-1" />
                Limpar filtros
              </Button>
            )}
            <span className="text-xs text-muted-foreground ml-auto self-end">
              {displayItems.length} de {items.length} nota(s)
            </span>
          </div>

          {items.length > 0 && (
            <div className="flex flex-wrap gap-3 items-end border-t pt-3">
              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">Cliente (nome ou CNPJ/CPF)</label>
                <Input
                  placeholder="Digite nome ou documento..."
                  value={filterCliente}
                  onChange={e => setFilterCliente(e.target.value)}
                  className="h-8 w-60"
                />
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">Emissão De</label>
                <Input
                  type="date"
                  value={filterDataDe}
                  onChange={e => setFilterDataDe(e.target.value)}
                  className="h-8 w-36"
                />
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">Emissão Até</label>
                <Input
                  type="date"
                  value={filterDataAte}
                  onChange={e => setFilterDataAte(e.target.value)}
                  className="h-8 w-36"
                />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* ── Totalizador ── */}
      {displayItems.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
          {[
            { label: 'Total vNF',   value: totalVNF },
            { label: 'Total vICMS', value: totalICMS },
            { label: 'Total vIBS',  value: totalIBS },
            { label: 'Total vCBS',  value: totalCBS },
          ].map(c => (
            <Card key={c.label} className="p-2">
              <p className="text-xs text-muted-foreground">{c.label}</p>
              <p className="text-sm font-bold mt-0.5">{fmtBRL(c.value)}</p>
            </Card>
          ))}
        </div>
      )}

      {/* ── Tabela ── */}
      <Card>
        <CardHeader className="py-2 px-4">
          <CardTitle className="text-xs text-muted-foreground font-normal">
            Clique em uma linha para ver todos os dados da nota
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {displayItems.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-sm font-bold">Nenhuma nota encontrada</p>
              <p className="text-xs text-muted-foreground mt-1">
                {isLoading ? 'Carregando...' : 'Importe XMLs na tab "Importar XMLs" ou ajuste os filtros acima.'}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="py-1.5 px-2 text-xs">CNPJ Emitente</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs">Filial / UF</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs">Cliente</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs">Data</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs text-center">Série</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs text-center">Nº Nota</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs text-center">Mod</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs text-right">Valor Total (vNF)</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {displayItems.map(row => (
                    <TableRow
                      key={row.id}
                      className="cursor-pointer hover:bg-muted/50 h-8"
                      onClick={() => setSelectedId(row.id)}
                    >
                      <TableCell className="py-1 px-2 font-mono text-xs">
                        {fmtCNPJ(row.emit_cnpj)}
                      </TableCell>
                      <TableCell className="py-1 px-2">
                        <div className="text-xs font-bold leading-tight">{row.emit_nome || '—'}</div>
                        <div className="text-xs text-muted-foreground leading-tight">{row.emit_uf}</div>
                      </TableCell>
                      <TableCell className="py-1 px-2">
                        <div className="text-xs font-bold leading-tight">{row.dest_nome || '—'}</div>
                        <div className="text-xs text-muted-foreground font-mono leading-tight">
                          {fmtCNPJ(row.dest_cnpj_cpf)}
                        </div>
                      </TableCell>
                      <TableCell className="py-1 px-2 text-xs whitespace-nowrap">
                        {row.data_emissao}
                      </TableCell>
                      <TableCell className="py-1 px-2 text-xs text-center">{row.serie}</TableCell>
                      <TableCell className="py-1 px-2 text-xs text-center font-mono">{row.numero_nfe}</TableCell>
                      <TableCell className="py-1 px-2 text-center">
                        <Badge variant="outline" className="text-xs px-1 py-0">{row.modelo}</Badge>
                      </TableCell>
                      <TableCell className="py-1 px-2 text-xs text-right font-bold">
                        {fmtBRL(row.v_nf)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* ── Dialog de detalhe ── */}
      {selectedId && (
        <DetalheNFe id={selectedId} onClose={() => setSelectedId(null)} authHeaders={authHeaders} />
      )}
    </div>
  );
}
