import { create } from 'zustand'
import { login as loginRequest } from '../api/auth'
import { errorMessage, getAccessToken, setAccessToken } from '../api/client'
import type { Actor } from '../types/auth'

const USER_KEY = 'certrollover-user'

function storedUser(): Actor | null {
  try {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? (JSON.parse(raw) as Actor) : null
  } catch {
    return null
  }
}

interface AuthState {
  token: string | null
  user: Actor | null
  loading: boolean
  error: string
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  clearError: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  token: getAccessToken(),
  user: storedUser(),
  loading: false,
  error: '',
  login: async (username, password) => {
    set({ loading: true, error: '' })
    try {
      const response = await loginRequest(username, password)
      setAccessToken(response.access_token)
      localStorage.setItem(USER_KEY, JSON.stringify(response.user))
      set({ token: response.access_token, user: response.user, loading: false })
    } catch (error) {
      set({ loading: false, error: errorMessage(error) })
      throw error
    }
  },
  logout: () => {
    setAccessToken(null)
    localStorage.removeItem(USER_KEY)
    set({ token: null, user: null, error: '' })
  },
  clearError: () => set({ error: '' }),
}))

