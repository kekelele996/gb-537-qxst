import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'

export function ProtectedRoute() {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const location = useLocation()
  return token && user ? <Outlet /> : <Navigate to="/login" replace state={{ from: location.pathname }} />
}

