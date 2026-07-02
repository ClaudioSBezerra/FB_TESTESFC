import { useState, useRef } from 'react';
import { useDropzone } from 'react-dropzone';
import { useAuth } from '@/contexts/AuthContext';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Upload, CloudUpload, CheckCircle, XCircle, Loader2, FolderOpen } from 'lucide-react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type UploadState = 'idle' | 'scanning' | 'uploading' | 'polling' | 'done' | 'error';

interface UploadErro {
  arquivo: string;
  erro: string;
}

interface UploadApiResult {
  importados: number;
  rejeitados: number;
  total: number;
  erros: UploadErro[];
}

interface HistoricoEntry {
  id: string;
  timestamp: string;
  filename: string;
  total: number;
  importados: number;
  rejeitados: number;
  erros: UploadErro[];
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function fmtDateTime(d: Date): string {
  return d.toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
export default function ImportarXMLsSaida() {
  const { token, companyId } = useAuth();
  const authHeaders = { Authorization: `Bearer ${token}`, 'X-Company-ID': companyId || '' };

  const [uploadState, setUploadState] = useState<UploadState>('idle');
  const [uploadResult, setUploadResult] = useState<UploadApiResult | null>(null);
  const [progress, setProgress] = useState(0);
  const [historico, setHistorico] = useState<HistoricoEntry[]>([]);

  // ── Upload handler ─────────────────────────────────────────────────────────
  const handleUpload = async (files: File[]) => {
    if (files.length === 0) return;

    setUploadState('scanning');
    setUploadResult(null);
    setProgress(0);

    // Yield to React so the 'scanning' state renders before FormData assembly
    await new Promise(resolve => setTimeout(resolve, 0));

    setUploadState('uploading');
    setProgress(30);

    try {
      const formData = new FormData();
      files.forEach(f => formData.append('file', f));

      const res = await fetch('/api/xml/upload', {
        method: 'POST',
        headers: authHeaders,
        body: formData,
      });

      setProgress(80);

      if (res.status === 413) {
        toast.error('Arquivo excede o limite de 5GB.');
        setUploadState('error');
        return;
      }

      const data = await res.json();

      if (!res.ok) {
        toast.error('Erro no upload: ' + (data.error || res.statusText));
        setUploadState('error');
        return;
      }

      const result: UploadApiResult = {
        importados: data.importados ?? 0,
        rejeitados: data.rejeitados ?? 0,
        total: data.total ?? 0,
        erros: data.erros ?? [],
      };

      setProgress(100);
      setUploadResult(result);
      setUploadState('done');
      toast.success(
        `Upload concluído: ${result.importados} NF-e(s) importadas, ${result.rejeitados} rejeitadas.`
      );

      setHistorico(prev => [
        {
          id: `${Date.now()}`,
          timestamp: fmtDateTime(new Date()),
          filename: files.length === 1 ? files[0].name : `${files.length} arquivos`,
          total: result.total,
          importados: result.importados,
          rejeitados: result.rejeitados,
          erros: result.erros,
        },
        ...prev,
      ]);
    } catch (err: unknown) {
      toast.error('Erro inesperado: ' + String(err));
      setUploadState('error');
    }
  };

  // ── Dropzone ───────────────────────────────────────────────────────────────
  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    accept: {
      'text/xml': ['.xml'],
      'application/zip': ['.zip'],
      'application/x-zip-compressed': ['.zip'],
    },
    maxSize: 5 * 1024 * 1024 * 1024,
    multiple: true,
    onDropRejected: (rejected) => {
      toast.error(`${rejected.length} arquivo(s) rejeitado(s). Apenas XML e ZIP até 5GB.`);
    },
    onDrop: handleUpload,
    disabled: uploadState === 'uploading' || uploadState === 'polling',
  });

  const isProcessing = uploadState === 'scanning' || uploadState === 'uploading' || uploadState === 'polling';

  const folderInputRef = useRef<HTMLInputElement>(null);
  const handleFolderSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []).filter(f => f.name.toLowerCase().endsWith('.xml'));
    e.target.value = '';
    if (files.length === 0) { toast.error('Nenhum arquivo .xml encontrado na pasta.'); return; }
    handleUpload(files);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Importar XMLs</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Importe NF-e (mod. 55) e NFC-e (mod. 65) de SAÍDA da Ferreira Costa. Arraste arquivos
          XML ou ZIP, ou clique para selecionar. Limite: 5GB por envio.
        </p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div
            {...getRootProps()}
            className={[
              'flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed px-6 py-10 cursor-pointer transition-colors',
              isDragActive
                ? 'border-blue-500 bg-blue-50 text-blue-700'
                : isProcessing
                  ? 'border-muted bg-muted/30 cursor-not-allowed text-muted-foreground'
                  : 'border-muted-foreground/25 hover:border-primary/50 hover:bg-muted/20 text-muted-foreground',
            ].join(' ')}
          >
            <input {...getInputProps()} />
            {isProcessing ? (
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            ) : isDragActive ? (
              <CloudUpload className="h-8 w-8 text-blue-500" />
            ) : (
              <Upload className="h-8 w-8" />
            )}
            <div className="text-center">
              {uploadState === 'scanning' && <p className="text-sm font-bold">Lendo arquivos...</p>}
              {uploadState === 'uploading' && <p className="text-sm font-bold">Enviando arquivos...</p>}
              {!isProcessing && isDragActive && <p className="text-sm font-bold">Solte os arquivos aqui</p>}
              {!isProcessing && !isDragActive && (
                <>
                  <p className="text-sm font-bold">Arraste XMLs ou .zip de NF-e de saída aqui, ou clique</p>
                  <p className="text-xs text-muted-foreground mt-1">Aceita .xml, .zip — máximo 5GB</p>
                  <button
                    type="button"
                    onClick={e => { e.stopPropagation(); folderInputRef.current?.click(); }}
                    disabled={isProcessing}
                    className="mt-2 inline-flex items-center gap-1.5 text-xs text-primary hover:underline disabled:opacity-50"
                  >
                    <FolderOpen className="h-3.5 w-3.5" />
                    Selecionar Pasta
                  </button>
                  <input ref={folderInputRef} type="file" className="hidden" onChange={handleFolderSelect}
                    {...({ webkitdirectory: '', directory: '' } as React.InputHTMLAttributes<HTMLInputElement>)} />
                </>
              )}
            </div>
          </div>

          {isProcessing && (
            <div className="mt-4 space-y-1.5">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>Processando XMLs...</span>
                <span>{progress}%</span>
              </div>
              <Progress value={progress} className="h-2" />
            </div>
          )}

          {uploadResult && (uploadState === 'done' || uploadState === 'error') && (
            <div className="mt-4 rounded-lg border p-4 space-y-3">
              <div className="flex gap-4 flex-wrap items-center">
                <div className="flex items-center gap-2">
                  <CheckCircle className="h-4 w-4 text-green-600" />
                  <span className="text-sm font-bold">Importados:</span>
                  <Badge className="bg-green-600">{uploadResult.importados}</Badge>
                </div>
                {uploadResult.rejeitados > 0 && (
                  <div className="flex items-center gap-2">
                    <XCircle className="h-4 w-4 text-red-500" />
                    <span className="text-sm font-bold">Rejeitados:</span>
                    <Badge variant="destructive">{uploadResult.rejeitados}</Badge>
                  </div>
                )}
                <span className="text-xs text-muted-foreground ml-auto">Total: {uploadResult.total}</span>
              </div>
              {uploadResult.erros.length > 0 && (
                <div className="text-xs space-y-1 max-h-40 overflow-auto bg-red-50 text-red-700 rounded p-2">
                  {uploadResult.erros.map((e, i) => (
                    <div key={i}>
                      <span className="font-bold">{e.arquivo}:</span> {e.erro}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* ── Histórico de uploads (sessão atual) ── */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Histórico de Uploads — NF-e Saídas</CardTitle>
        </CardHeader>
        <CardContent>
          {historico.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-6">
              Nenhum upload registrado ainda nesta sessão.
            </p>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="py-1.5 px-2 text-xs">Data</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs">Arquivo</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs">Total</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs text-green-700">Importados</TableHead>
                    <TableHead className="py-1.5 px-2 text-xs text-red-600">Rejeitados</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {historico.map(row => (
                    <TableRow key={row.id} className="h-8">
                      <TableCell className="py-1 px-2 text-xs whitespace-nowrap">{row.timestamp}</TableCell>
                      <TableCell className="py-1 px-2 text-xs max-w-[200px] truncate" title={row.filename}>{row.filename}</TableCell>
                      <TableCell className="py-1 px-2 text-xs text-center">{row.total}</TableCell>
                      <TableCell className="py-1 px-2 text-xs text-center text-green-700 font-bold">{row.importados}</TableCell>
                      <TableCell className="py-1 px-2 text-xs text-center text-red-600">{row.rejeitados || '—'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
