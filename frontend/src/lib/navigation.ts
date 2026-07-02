export interface ModuleTab {
  label: string
  path: string
  disabled?: boolean
  danger?: boolean
  adminOnly?: boolean
}

export interface ModuleConfig {
  label: string
  tabs: ModuleTab[]
}

export const modules: Record<string, ModuleConfig> = {
  importacoes: {
    label: 'Importações',
    tabs: [
      { label: 'Importar XMLs', path: '/importacoes/xmls-saida' },
    ],
  },
  notas: {
    label: 'Notas Importadas',
    tabs: [
      { label: 'Notas Importadas', path: '/importacoes/notas-saida' },
    ],
  },
  comparacao: {
    label: 'Comparação Fiscal',
    tabs: [
      { label: 'Comparação Fiscal', path: '/importacoes/comparacao-fiscal' },
    ],
  },
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
}

export function getActiveModule(pathname: string): string {
  if (pathname.startsWith('/importacoes/xmls-saida')) return 'importacoes'
  if (pathname.startsWith('/importacoes/notas-saida')) return 'notas'
  if (pathname.startsWith('/importacoes/comparacao-fiscal')) return 'comparacao'
  if (pathname.startsWith('/config/') || pathname.startsWith('/importacoes/erp-bridge')) return 'config'
  return 'comparacao'
}
