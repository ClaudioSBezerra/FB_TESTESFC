import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from '@/components/ui/sonner'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import ResetPassword from './pages/ResetPassword'
import GestaoAmbiente from './pages/GestaoAmbiente'
import Managers from './pages/Managers'
import AdminUsers from './pages/AdminUsers'
import ERPBridgeConfig from './pages/ERPBridgeConfig'
import ERPBridgeCredenciais from './pages/ERPBridgeCredenciais'
import { AppRail } from '@/components/AppRail'
import { AuthProvider, useAuth } from './contexts/AuthContext'
// Sem FilialProvider, CompanySwitcher, AjudaChat, ModuleTabs (D-04, D-11)

const queryClient = new QueryClient()

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth()
  const location = useLocation()
  if (loading) return null
  if (!isAuthenticated) return <Navigate to="/login" state={{ from: location }} replace />
  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading, user } = useAuth()
  const location = useLocation()
  if (loading) return null
  if (!isAuthenticated) return <Navigate to="/login" state={{ from: location }} replace />
  if (user?.role !== 'admin') return <Navigate to="/" replace />
  return <>{children}</>
}

function AppLayout() {
  const { company } = useAuth()
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppRail />
      <div className="flex flex-col flex-1 min-w-0">
        <header className="flex items-center justify-between h-12 border-b bg-white px-4 shrink-0">
          <span className="text-sm font-semibold">FB_TESTESFC — Validador Fiscal</span>
          {company && (
            <span className="flex items-center gap-1.5 text-xs font-medium text-sky-700 bg-sky-50 border border-sky-200 px-2.5 py-1 rounded-full">
              {company}
            </span>
          )}
          {/* Sem CompanySwitcher (D-11) */}
        </header>
        {/* Sem ModuleTabs */}
        <main className="flex-1 overflow-auto">
          <div className="p-4">
            <Routes>
              <Route path="/"                       element={<Navigate to="/config/erp-bridge" replace />} />
              <Route path="/config/ambiente"        element={<ProtectedRoute><GestaoAmbiente /></ProtectedRoute>} />
              <Route path="/config/gestores"        element={<ProtectedRoute><Managers /></ProtectedRoute>} />
              <Route path="/config/usuarios"        element={<AdminRoute><AdminUsers /></AdminRoute>} />
              <Route path="/importacoes/erp-bridge" element={<AdminRoute><ERPBridgeConfig /></AdminRoute>} />
              <Route path="/config/erp-bridge"      element={<AdminRoute><ERPBridgeCredenciais /></AdminRoute>} />
            </Routes>
          </div>
        </main>
      </div>
      <Toaster />
      {/* Sem AjudaChat */}
    </div>
  )
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <AuthProvider>
          <Routes>
            <Route path="/login"           element={<Login />} />
            <Route path="/register"        element={<Register />} />
            <Route path="/forgot-password" element={<ForgotPassword />} />
            <Route path="/reset-senha"     element={<ResetPassword />} />
            {/* Sem FilialProvider (D-11) */}
            <Route path="/*" element={<ProtectedRoute><AppLayout /></ProtectedRoute>} />
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
