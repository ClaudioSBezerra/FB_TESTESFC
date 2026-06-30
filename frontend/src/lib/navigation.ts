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
  if (pathname.startsWith('/config/')) return 'config'
  if (pathname.startsWith('/importacoes/')) return 'config'
  return 'config'
}
