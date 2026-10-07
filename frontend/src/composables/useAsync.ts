import { ref, type Ref } from 'vue'

export function useAsync<T>(fn: () => Promise<T>) {
  const data: Ref<T | null> = ref(null)
  const loading = ref(false)
  const error = ref<Error | null>(null)

  async function run(): Promise<T | null> {
    loading.value = true
    error.value = null
    try {
      data.value = await fn()
      return data.value
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
      return null
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, run }
}
