import { useEffect } from 'react'
import { useAuthStore } from '../stores/auth'
import { can, type Permission } from '../utils/permissions'

export function useAuth() {
  const state = useAuthStore()
  useEffect(() => {
    const unauthorized = () => state.logout()
    window.addEventListener('certrollover:unauthorized', unauthorized)
    return () => window.removeEventListener('certrollover:unauthorized', unauthorized)
  }, [state.logout])
  return { ...state, can: (permission: Permission) => can(state.user, permission) }
}

