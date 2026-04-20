import { useEffect, useMemo, useState } from 'react'
import { api } from '../api'
import type {
  Artifact,
  CreateRepositoryRequest,
  HealthState,
  RepoType,
  Repository,
  ToolingInstallResponse,
} from '../types'

interface RepoFormState {
  name: string
  slug: string
  type: RepoType
  codename: string
  component: string
  architectures: string
}

const defaultForm: RepoFormState = {
  name: '',
  slug: '',
  type: 'rpm',
  codename: 'stable',
  component: 'main',
  architectures: 'amd64',
}

const inputClass =
  'w-full rounded-md border border-rf-border bg-rf-surface px-3 py-2 text-sm text-rf-fg outline-none placeholder:text-rf-muted focus:border-rf-accent/40 focus:ring-1 focus:ring-rf-accent/30'

const labelClass = 'text-xs font-medium uppercase tracking-wider text-rf-muted'

export function Dashboard() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [repositories, setRepositories] = useState<Repository[]>([])
  const [artifactsByRepo, setArtifactsByRepo] = useState<Record<string, Artifact[]>>({})
  const [health, setHealth] = useState<HealthState>({ ok: false, message: 'loading' })
  const [ready, setReady] = useState<HealthState>({ ok: false, message: 'loading' })
  const [form, setForm] = useState<RepoFormState>(defaultForm)
  const [creating, setCreating] = useState(false)
  const [uploadingSlug, setUploadingSlug] = useState<string | null>(null)
  const [deletingRepoSlug, setDeletingRepoSlug] = useState<string | null>(null)
  const [deletingArtifactID, setDeletingArtifactID] = useState<number | null>(null)
  const [installing, setInstalling] = useState(false)
  const [toolingResult, setToolingResult] = useState<ToolingInstallResponse | null>(null)
  const [pathByRepo, setPathByRepo] = useState<Record<string, string>>({})

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const [healthState, readyState, repoResponse] = await Promise.all([
        api.healthz(),
        api.readyz(),
        api.listRepositories(),
      ])
      setHealth(healthState)
      setReady(readyState)
      setRepositories(repoResponse.repositories)

      const artifactEntries = await Promise.all(
        repoResponse.repositories.map(async (repo) => {
          try {
            const response = await api.listArtifacts(repo.slug)
            return [repo.slug, response.artifacts] as const
          } catch {
            return [repo.slug, []] as const
          }
        }),
      )
      setArtifactsByRepo(Object.fromEntries(artifactEntries))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load RepoForge data')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [])

  const stats = useMemo(() => {
    const counts = { rpm: 0, deb: 0, file: 0 }
    repositories.forEach((repo) => {
      counts[repo.type] += 1
    })
    return counts
  }, [repositories])

  const updateForm = <K extends keyof RepoFormState>(key: K, value: RepoFormState[K]) => {
    setForm((current) => ({ ...current, [key]: value }))
  }

  const handleCreateRepository = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setCreating(true)
    setError(null)

    const body: CreateRepositoryRequest = {
      name: form.name.trim(),
      slug: form.slug.trim(),
      type: form.type,
    }

    if (form.type === 'deb') {
      body.config = {
        codename: form.codename.trim() || 'stable',
        component: form.component.trim() || 'main',
        architectures: form.architectures
          .split(',')
          .map((item) => item.trim())
          .filter(Boolean),
      }
    }

    try {
      await api.createRepository(body)
      setForm(defaultForm)
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create repository')
    } finally {
      setCreating(false)
    }
  }

  const handleInstallTooling = async () => {
    const approved = window.confirm(
      'Install repository tooling on this host? This may require root privileges and a configured bearer token.',
    )
    if (!approved) {
      return
    }

    setInstalling(true)
    setToolingResult(null)
    try {
      const response = await api.installRepoTooling()
      setToolingResult(response)
    } catch (err) {
      setToolingResult({ error: err instanceof Error ? err.message : 'Installation failed' })
    } finally {
      setInstalling(false)
    }
  }

  const handleUpload = async (repo: Repository, event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const formEl = event.currentTarget
    const formData = new FormData(formEl)
    const file = formData.get('file')
    const path = formData.get('path')

    if (!(file instanceof File) || file.size === 0) {
      window.alert('Choose a file first.')
      return
    }

    setUploadingSlug(repo.slug)
    try {
      const response = await api.uploadArtifact(
        repo.slug,
        file,
        typeof path === 'string' ? path : undefined,
      )
      setArtifactsByRepo((current) => ({
        ...current,
        [repo.slug]: [response.artifact, ...(current[repo.slug] ?? [])],
      }))
      setPathByRepo((current) => ({ ...current, [repo.slug]: '' }))
      formEl.reset()
    } catch (err) {
      window.alert(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploadingSlug(null)
    }
  }

  const handleDeleteRepository = async (repo: Repository) => {
    const approved = window.confirm(`Delete repository "${repo.name}" and all uploaded artifacts?`)
    if (!approved) {
      return
    }
    setDeletingRepoSlug(repo.slug)
    try {
      await api.deleteRepository(repo.slug)
      setRepositories((current) => current.filter((item) => item.slug !== repo.slug))
      setArtifactsByRepo((current) => {
        const next = { ...current }
        delete next[repo.slug]
        return next
      })
    } catch (err) {
      window.alert(err instanceof Error ? err.message : 'Delete repository failed')
    } finally {
      setDeletingRepoSlug(null)
    }
  }

  const handleDeleteArtifact = async (repo: Repository, artifact: Artifact) => {
    const approved = window.confirm(`Delete uploaded file "${artifact.logical_path}"?`)
    if (!approved) {
      return
    }
    setDeletingArtifactID(artifact.id)
    try {
      await api.deleteArtifact(repo.slug, artifact.id)
      setArtifactsByRepo((current) => ({
        ...current,
        [repo.slug]: (current[repo.slug] ?? []).filter((item) => item.id !== artifact.id),
      }))
    } catch (err) {
      window.alert(err instanceof Error ? err.message : 'Delete artifact failed')
    } finally {
      setDeletingArtifactID(null)
    }
  }

  const publicIndexUrl = (repo: Repository, artifacts: Artifact[]): string => {
    const base = api.repoBaseUrl(repo)
    if (repo.type === 'rpm') {
      return `${base}/rpms/repodata/repomd.xml`
    }
    if (repo.type === 'file') {
      const first = artifacts[0]
      if (first) {
        return `${base}/${first.logical_path}`
      }
      return `${base}/files/`
    }
    const codename = typeof repo.config.codename === 'string' ? repo.config.codename : 'stable'
    const component = typeof repo.config.component === 'string' ? repo.config.component : 'main'
    const archList = Array.isArray(repo.config.architectures) ? repo.config.architectures : ['amd64']
    const arch = archList[0] || 'amd64'
    return `${base}/dists/${codename}/${component}/binary-${arch}/Packages`
  }

  return (
    <div className="mx-auto max-w-6xl space-y-10">
      <header className="flex flex-col gap-6 border-b border-rf-border pb-8 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="font-mono text-xs text-rf-accent">/v1/repositories</p>
          <h2 className="mt-1 text-3xl font-semibold tracking-tight sm:text-4xl">Repositories</h2>
          <p className="mt-2 max-w-xl text-sm text-rf-muted">
            Health checks, creation, uploads, and host tooling — same API the UI calls.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <StatusChip label="Process" ok={health.ok} detail={health.message} />
          <StatusChip label="SQLite" ok={ready.ok} detail={ready.message} />
          <span className="hidden h-6 w-px bg-rf-border sm:block" aria-hidden />
          <span className="rounded-full border border-rf-border bg-rf-elevated px-3 py-1 font-mono text-xs text-rf-muted">
            {repositories.length} repos
          </span>
          <span className="rounded-full border border-rf-border bg-rf-elevated px-3 py-1 font-mono text-xs text-rf-muted">
            rpm {stats.rpm} · deb {stats.deb} · file {stats.file}
          </span>
          <button
            type="button"
            onClick={() => void loadData()}
            className="rounded-full border border-rf-border px-4 py-1.5 text-sm font-medium text-rf-fg hover:border-rf-accent/50 hover:text-rf-accent"
          >
            Reload
          </button>
        </div>
      </header>

      {error && (
        <div
          role="alert"
          className="border-l-4 border-rf-danger bg-rf-elevated px-4 py-3 text-sm text-rf-fg"
        >
          {error}
        </div>
      )}

      <div className="grid gap-10 xl:grid-cols-[1fr_min(380px,100%)]">
        <section className="space-y-5">
          <div className="flex items-baseline justify-between gap-4">
            <h3 className="text-lg font-semibold">Published trees</h3>
            {loading && <span className="text-sm text-rf-muted">Loading…</span>}
          </div>

          {!loading && repositories.length === 0 ? (
            <div className="rounded-xl border border-dashed border-rf-border bg-rf-surface/50 px-6 py-12 text-center">
              <p className="text-sm text-rf-muted">No repositories yet.</p>
              <p className="mt-2 text-xs text-rf-muted">Create one in the panel on the right.</p>
            </div>
          ) : (
            <ul className="space-y-5">
              {repositories.map((repo) => {
                const artifacts = artifactsByRepo[repo.slug] ?? []
                const accent = repoAccent(repo.type)
                return (
                  <li
                    key={repo.id}
                    className={`overflow-hidden rounded-xl border border-rf-border bg-rf-elevated/80 shadow-[0_0_0_1px_rgba(0,0,0,0.25)] backdrop-blur-sm ${accent}`}
                  >
                    <div className="flex flex-col gap-4 border-b border-rf-border/80 p-5 lg:flex-row lg:items-start lg:justify-between">
                      <div className="min-w-0 space-y-2">
                        <div className="flex flex-wrap items-center gap-2">
                          <h4 className="text-lg font-semibold">{repo.name}</h4>
                          <TypeBadge type={repo.type} />
                        </div>
                        <p className="font-mono text-xs text-rf-muted">
                          slug <span className="text-rf-fg">{repo.slug}</span>
                        </p>
                        <p className="text-xs text-rf-muted">
                          created {new Date(repo.created_at).toLocaleString()}
                        </p>
                      </div>
                      <div className="flex shrink-0 flex-wrap gap-2">
                        <a
                          href={publicIndexUrl(repo, artifacts)}
                          target="_blank"
                          rel="noreferrer"
                          className="rounded-md border border-rf-border px-3 py-1.5 text-xs font-medium text-rf-accent hover:bg-rf-surface"
                        >
                          Open index URL
                        </a>
                        <button
                          type="button"
                          onClick={() => void handleDeleteRepository(repo)}
                          disabled={deletingRepoSlug === repo.slug}
                          className="rounded-md border border-rf-danger/50 px-3 py-1.5 text-xs font-medium text-rf-danger hover:bg-rf-danger/10 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {deletingRepoSlug === repo.slug ? 'Deleting…' : 'Delete repo'}
                        </button>
                      </div>
                    </div>

                    {repo.type === 'deb' && (
                      <div className="border-b border-rf-border/80 bg-rf-surface/40 px-5 py-3 font-mono text-xs text-rf-muted">
                        <span className="text-rf-fg">{String(repo.config.codename ?? 'stable')}</span>
                        {' / '}
                        <span className="text-rf-fg">{String(repo.config.component ?? 'main')}</span>
                        {' · '}
                        {Array.isArray(repo.config.architectures)
                          ? repo.config.architectures.join(', ')
                          : 'amd64'}
                      </div>
                    )}

                    <form
                      onSubmit={(event) => void handleUpload(repo, event)}
                      className="grid gap-3 border-b border-rf-border/80 p-5 md:grid-cols-[1fr_1fr_auto] md:items-end"
                    >
                      <label className={`block space-y-1.5 ${labelClass}`}>
                        <span>Package or file</span>
                        <input
                          name="file"
                          type="file"
                          required
                          className={`${inputClass} file:mr-3 file:rounded file:border-0 file:bg-rf-border file:px-2 file:py-1 file:text-xs file:text-rf-fg`}
                        />
                      </label>
                      <label className={`block space-y-1.5 ${labelClass}`}>
                        <span>Path override</span>
                        <input
                          name="path"
                          value={pathByRepo[repo.slug] ?? ''}
                          onChange={(event) =>
                            setPathByRepo((current) => ({ ...current, [repo.slug]: event.target.value }))
                          }
                          placeholder={
                            repo.type === 'file' ? 'files/…' : 'optional'
                          }
                          className={inputClass}
                        />
                      </label>
                      <button
                        type="submit"
                        disabled={uploadingSlug === repo.slug}
                        className="h-10 rounded-md bg-rf-accent px-4 text-sm font-semibold text-rf-void hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        {uploadingSlug === repo.slug ? 'Uploading…' : 'Upload'}
                      </button>
                    </form>

                    <div className="grid gap-3 border-b border-rf-border/80 bg-rf-surface/30 p-5 lg:grid-cols-2">
                      <label className={`block space-y-1.5 ${labelClass}`}>
                        <span>Upload API URL</span>
                        <input
                          readOnly
                          value={api.repoInstallHints(repo).upload_url}
                          className={`${inputClass} font-mono text-xs`}
                        />
                      </label>
                      <label className={`block space-y-1.5 ${labelClass}`}>
                        <span>Linux install script URL</span>
                        <input
                          readOnly
                          value={api.repoInstallHints(repo).script_url}
                          className={`${inputClass} font-mono text-xs`}
                        />
                      </label>
                    </div>

                    <div className="p-5">
                      <p className={`${labelClass} mb-3`}>Recent artifacts</p>
                      {artifacts.length === 0 ? (
                        <p className="text-sm text-rf-muted">Nothing uploaded yet.</p>
                      ) : (
                        <ul className="divide-y divide-rf-border/60 rounded-lg border border-rf-border bg-rf-surface/40">
                          {artifacts.slice(0, 5).map((artifact) => (
                            <li
                              key={artifact.id}
                              className="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
                            >
                              <div className="min-w-0">
                                <p className="truncate font-mono text-sm text-rf-fg">{artifact.logical_path}</p>
                                <p className="mt-0.5 text-xs text-rf-muted">
                                  {formatBytes(artifact.size)} · {new Date(artifact.created_at).toLocaleString()}
                                </p>
                              </div>
                              <div className="flex shrink-0 items-center gap-3">
                                <a
                                  href={`${api.repoBaseUrl(repo)}/${artifact.logical_path}`}
                                  target="_blank"
                                  rel="noreferrer"
                                  className="text-xs font-medium text-rf-accent hover:underline"
                                >
                                  GET
                                </a>
                                <button
                                  type="button"
                                  onClick={() => void handleDeleteArtifact(repo, artifact)}
                                  disabled={deletingArtifactID === artifact.id}
                                  className="text-xs font-medium text-rf-danger hover:underline disabled:cursor-not-allowed disabled:opacity-50"
                                >
                                  {deletingArtifactID === artifact.id ? 'Deleting…' : 'Delete'}
                                </button>
                              </div>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  </li>
                )
              })}
            </ul>
          )}
        </section>

        <aside className="space-y-6">
          <div className="rounded-xl border border-rf-border bg-rf-elevated/90 p-6">
            <h3 className="text-lg font-semibold">New repository</h3>
            <p className="mt-1 text-sm text-rf-muted">Lowercase slug; letters, digits, hyphen.</p>

            <form onSubmit={(event) => void handleCreateRepository(event)} className="mt-5 space-y-4">
              <Field label="Name">
                <input
                  value={form.name}
                  onChange={(event) => updateForm('name', event.target.value)}
                  className={inputClass}
                  placeholder="Production RPMs"
                  required
                />
              </Field>

              <Field label="Slug">
                <input
                  value={form.slug}
                  onChange={(event) =>
                    updateForm('slug', event.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '-'))
                  }
                  className={`${inputClass} font-mono`}
                  placeholder="production-rpms"
                  required
                />
              </Field>

              <Field label="Type">
                <select
                  value={form.type}
                  onChange={(event) => updateForm('type', event.target.value as RepoType)}
                  className={inputClass}
                >
                  <option value="rpm">RPM (DNF/YUM)</option>
                  <option value="deb">DEB (APT)</option>
                  <option value="file">Static files</option>
                </select>
              </Field>

              {form.type === 'deb' && (
                <div className="grid gap-4 rounded-lg border border-rf-border bg-rf-surface/50 p-4 md:grid-cols-2">
                  <Field label="Codename">
                    <input
                      value={form.codename}
                      onChange={(event) => updateForm('codename', event.target.value)}
                      className={`${inputClass} font-mono`}
                    />
                  </Field>
                  <Field label="Component">
                    <input
                      value={form.component}
                      onChange={(event) => updateForm('component', event.target.value)}
                      className={`${inputClass} font-mono`}
                    />
                  </Field>
                  <div className="md:col-span-2">
                    <Field label="Architectures">
                      <input
                        value={form.architectures}
                        onChange={(event) => updateForm('architectures', event.target.value)}
                        className={`${inputClass} font-mono`}
                        placeholder="amd64, arm64"
                      />
                    </Field>
                  </div>
                </div>
              )}

              <button
                type="submit"
                disabled={creating}
                className="w-full rounded-md border border-rf-accent/40 bg-rf-accent/10 py-2.5 text-sm font-semibold text-rf-accent hover:bg-rf-accent/15 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {creating ? 'Creating…' : 'Create repository'}
              </button>
            </form>
          </div>

          <div className="rounded-xl border border-rf-border bg-rf-elevated/90 p-6">
            <h3 className="text-lg font-semibold">Host packages</h3>
            <p className="mt-2 text-sm leading-relaxed text-rf-muted">
              Installs createrepo_c (and related) plus optional Debian tooling when the API token is set and the process runs as root.
            </p>
            <button
              type="button"
              onClick={() => void handleInstallTooling()}
              disabled={installing}
              className="mt-4 w-full rounded-md border border-rf-warn/40 bg-rf-warn/10 py-2.5 text-sm font-semibold text-rf-warn hover:bg-rf-warn/15 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {installing ? 'Running installer…' : 'Install repo tooling'}
            </button>
            {toolingResult && (
              <div className="mt-4 space-y-2 rounded-lg border border-rf-border bg-rf-surface/50 p-4 text-sm text-rf-muted">
                <p className="font-medium text-rf-fg">
                  {toolingResult.ok ? 'Finished' : 'Response'}
                </p>
                {toolingResult.distro && (
                  <p>
                    Distro: <span className="font-mono text-rf-fg">{toolingResult.distro}</span>
                  </p>
                )}
                {toolingResult.detail && <p>{toolingResult.detail}</p>}
                {toolingResult.error && <p className="text-rf-danger">{toolingResult.error}</p>}
                {toolingResult.log && (
                  <pre className="max-h-48 overflow-auto rounded-md border border-rf-border bg-rf-void p-3 font-mono text-xs text-rf-fg/90">
                    {toolingResult.log}
                  </pre>
                )}
              </div>
            )}
          </div>
        </aside>
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className={labelClass}>{label}</span>
      {children}
    </label>
  )
}

function StatusChip({ label, ok, detail }: { label: string; ok: boolean; detail: string }) {
  return (
    <span
      className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium ${
        ok
          ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300'
          : 'border-rf-danger/30 bg-rf-danger/10 text-rf-danger'
      }`}
      title={detail}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${ok ? 'bg-emerald-400' : 'bg-rf-danger'}`} />
      {label}
      <span className="font-mono text-[10px] uppercase text-rf-muted">{ok ? 'ok' : 'fail'}</span>
    </span>
  )
}

function TypeBadge({ type }: { type: RepoType }) {
  const map: Record<RepoType, string> = {
    rpm: 'text-rose-300 border-rose-500/30 bg-rose-500/10',
    deb: 'text-sky-300 border-sky-500/30 bg-sky-500/10',
    file: 'text-amber-300 border-amber-500/30 bg-amber-500/10',
  }
  return (
    <span className={`rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${map[type]}`}>
      {type}
    </span>
  )
}

function repoAccent(type: RepoType): string {
  if (type === 'rpm') {
    return 'border-l-4 border-l-rose-500/70'
  }
  if (type === 'deb') {
    return 'border-l-4 border-l-sky-400/70'
  }
  return 'border-l-4 border-l-amber-400/70'
}

function formatBytes(size: number): string {
  if (size < 1024) {
    return `${size} B`
  }
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = size / 1024
  let unit = units[0]
  for (let i = 1; i < units.length && value >= 1024; i += 1) {
    value /= 1024
    unit = units[i]
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${unit}`
}
