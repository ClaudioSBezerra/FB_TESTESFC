import { useLocation, useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { useAuth } from '@/contexts/AuthContext'
import { modules, getActiveModule } from '@/lib/navigation'

export function ModuleTabs() {
  const location = useLocation()
  const navigate = useNavigate()
  const { user } = useAuth()
  const active = getActiveModule(location.pathname)
  const config = modules[active]

  if (!config || config.tabs.length <= 1) return null

  return (
    <div className="flex items-center gap-1 h-10 border-b bg-white px-4 shrink-0">
      {config.tabs.map(tab => {
        if (tab.adminOnly && user?.role !== 'admin') return null
        const isActive = location.pathname.startsWith(tab.path)
        return (
          <button
            key={tab.path}
            onClick={() => navigate(tab.path)}
            className={cn(
              'px-3 py-1.5 text-sm font-medium rounded-md transition-colors',
              isActive
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-gray-100 hover:text-foreground'
            )}
          >
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}
