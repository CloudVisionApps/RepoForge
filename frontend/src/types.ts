export type RepoType = 'deb' | 'rpm' | 'file'

export interface DebConfig {
  codename?: string
  component?: string
  architectures?: string[]
  origin?: string
  label?: string
  suite?: string
  description?: string
}

export interface Repository {
  id: number
  name: string
  slug: string
  type: RepoType
  config: DebConfig & Record<string, unknown>
  created_at: string
}

export interface Artifact {
  id: number
  repository_id: number
  logical_path: string
  sha256: string
  size: number
  content_type: string
  created_at: string
}

export interface RepositoryListResponse {
  repositories: Repository[]
}

export interface ArtifactListResponse {
  artifacts: Artifact[]
}

export interface UploadResponse {
  artifact: Artifact
  error?: string
  detail?: string
}

export interface CreateRepositoryRequest {
  name: string
  slug: string
  type: RepoType
  config?: Record<string, unknown>
}

export interface ToolingInstallResponse {
  ok?: boolean
  distro?: string
  log?: string
  error?: string
  detail?: string
}

export interface HealthState {
  ok: boolean
  message: string
}
