import { computed, ref } from 'vue'
import { authApi } from '../services/auth'
import type { User } from '../types'

const user = ref<User | null>(null)
const initialized = ref(false)
let hydratePromise: Promise<void> | null = null

async function doHydrate(): Promise<void> {
  try {
    const res = await authApi.me()
    user.value = res.user
  } catch {
    user.value = null
  } finally {
    initialized.value = true
  }
}

export function useAuth() {
  const isLoggedIn = computed(() => user.value !== null)

  /** Hydrate auth state once per app boot via GET /auth/me. */
  function hydrate(): Promise<void> {
    if (!hydratePromise) hydratePromise = doHydrate()
    return hydratePromise
  }

  async function login(email: string, password: string): Promise<void> {
    const res = await authApi.login(email, password)
    user.value = res.user
  }

  async function register(email: string, password: string): Promise<void> {
    const res = await authApi.register(email, password)
    user.value = res.user
  }

  async function logout(): Promise<void> {
    try {
      await authApi.logout()
    } finally {
      user.value = null
    }
  }

  /** Clear local state after an unexpected 401 (session expired). */
  function handleUnauthorized(): void {
    user.value = null
  }

  return { user, isLoggedIn, initialized, hydrate, login, register, logout, handleUnauthorized }
}
