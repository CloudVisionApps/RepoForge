import type {
  ArtifactListResponse,
  CreateRepositoryRequest,
  HealthState,
  Repository,
  RepositoryListResponse,
  ToolingInstallResponse,
  UploadResponse,
} from './types'

const STORAGE_KEY = 'repoforge-bearer-token'

export function getStoredToken(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) ?? ''
  } catch {
    return ''
  }
}

export function setStoredToken(token: string): void {
  localStorage.setItem(STORAGE_KEY, token)
}

export function clearStoredToken(): void {
  localStorage.removeItem(STORAGE_KEY)
}

async function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const token = getStoredToken().trim()
  const headers = new Headers(init.headers)

  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  if (init.body && !headers.has('Content-Type') && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  return fetch(path, { ...init, headers })
}

async function handleResponse<T>(response: Response): Promise<T> {
  const contentType = response.headers.get('content-type') ?? ''
  const payload = contentType.includes('application/json')
    ? await response.json().catch(() => null)
    : await response.text().catch(() => '')

  if (!response.ok) {
    if (payload && typeof payload === 'object' && 'error' in payload && typeof payload.error === 'string') {
      throw new Error(payload.error)
    }
    if (typeof payload === 'string' && payload.trim()) {
      throw new Error(payload.trim())
    }
    throw new Error(`HTTP ${response.status}`)
  }

  return payload as T
}

async function readHealth(path: string): Promise<HealthState> {
  const response = await fetch(path)
  const message = (await response.text().catch(() => '')).trim() || response.statusText
  return { ok: response.ok, message }
}

export const api = {
  healthz(): Promise<HealthState> {
    return readHealth('/healthz')
  },

  readyz(): Promise<HealthState> {
    return readHealth('/readyz')
  },

  listRepositories(): Promise<RepositoryListResponse> {
    return apiFetch('/v1/repositories').then(handleResponse<RepositoryListResponse>)
  },

  createRepository(body: CreateRepositoryRequest): Promise<Repository> {
    return apiFetch('/v1/repositories', {
      method: 'POST',
      body: JSON.stringify(body),
    }).then(handleResponse<Repository>)
  },

  listArtifacts(slug: string): Promise<ArtifactListResponse> {
    return apiFetch(`/v1/repositories/${encodeURIComponent(slug)}/artifacts`).then(
      handleResponse<ArtifactListResponse>,
    )
  },

  async uploadArtifact(slug: string, file: File, path?: string): Promise<UploadResponse> {
    const form = new FormData()
    form.set('file', file)
    if (path?.trim()) {
      form.set('path', path.trim())
    }

    const response = await apiFetch(`/v1/repositories/${encodeURIComponent(slug)}/uploads`, {
      method: 'POST',
      body: form,
    })
    return handleResponse<UploadResponse>(response)
  },

  installRepoTooling(): Promise<ToolingInstallResponse> {
    return apiFetch('/v1/system/install-repo-tooling', {
      method: 'POST',
      body: JSON.stringify({ confirm: true }),
    }).then(handleResponse<ToolingInstallResponse>)
  },

  repoBaseUrl(repo: Repository): string {
    return `${window.location.origin}/repo/${repo.slug}`
  },
}
