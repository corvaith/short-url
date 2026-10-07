export class ApiError extends Error {
  status: number
  code: string
  fields?: Record<string, string>
  retryAfter?: number

  constructor(
    status: number,
    code: string,
    message: string,
    fields?: Record<string, string>,
    retryAfter?: number,
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
    this.retryAfter = retryAfter
  }
}

const BASE = '/api/v1'

/** Endpoints where a 401 is an expected, non-session-expiring result. */
const AUTH_PATHS = ['/auth/me', '/auth/login', '/auth/register', '/auth/logout', '/auth/password']

let unauthorizedHandler: (() => void) | null = null

export function setUnauthorizedHandler(fn: () => void): void {
  unauthorizedHandler = fn
}

export interface RequestOptions {
  method?: string
  body?: unknown
  params?: Record<string, string | number | undefined>
}

export async function api<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const url = new URL(BASE + path, window.location.origin)
  for (const [key, value] of Object.entries(opts.params ?? {})) {
    if (value !== undefined) url.searchParams.set(key, String(value))
  }

  let res: Response
  try {
    res = await fetch(url.toString(), {
      method: opts.method ?? 'GET',
      credentials: 'same-origin',
      headers: opts.body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    })
  } catch {
    throw new ApiError(0, 'network_error', "Can't reach the server. Check your connection.")
  }

  if (res.status === 204) return undefined as T

  let data: unknown = null
  try {
    data = await res.json()
  } catch {
    // Non-JSON body; fall through with data = null.
  }

  if (!res.ok) {
    const shape = data as {
      error?: { code?: string; message?: string; fields?: Record<string, string> }
    } | null
    const code = shape?.error?.code ?? `http_${res.status}`
    const message = shape?.error?.message ?? `Request failed with status ${res.status}`
    if (res.status === 401 && !AUTH_PATHS.includes(path) && unauthorizedHandler) {
      unauthorizedHandler()
    }
    const retryAfter =
      res.status === 429 ? Number(res.headers.get('Retry-After') ?? '0') || 0 : undefined
    throw new ApiError(res.status, code, message, shape?.error?.fields, retryAfter)
  }

  return data as T
}

/** Human-friendly message for any thrown error, per the app's state contract. */
export function describeError(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.code === 'network_error') return e.message
    if (e.status === 429) return `Too many requests. Try again in ${e.retryAfter ?? 5} seconds.`
    if (e.code === 'rate_limited')
      return 'Too many requests. Please slow down and try again shortly.'
    return e.message
  }
  if (e instanceof Error && e.message) return e.message
  return 'Something went wrong. Please try again.'
}
