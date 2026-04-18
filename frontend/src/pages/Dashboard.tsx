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
    const formData = new FormData(event.currentTarget)
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
      event.currentTarget.reset()
    } catch (err) {
      window.alert(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploadingSlug(null)
    }
  }

  const examplePath = (repo: Repository): string => {
    const base = api.repoBaseUrl(repo)
    if (repo.type === 'rpm') {
      return `${base}/rpms/your-package.rpm`
    }
    if (repo.type === 'file') {
      return `${base}/files/notes/readme.txt`
    }
    const codename = typeof repo.config.codename === 'string' ? repo.config.codename : 'stable'
    const component = typeof repo.config.component === 'string' ? repo.config.component : 'main'
    const archList = Array.isArray(repo.config.architectures) ? repo.config.architectures : ['amd64']
    const arch = archList[0] || 'amd64'
    return `${base}/dists/${codename}/${component}/binary-${arch}/Packages`
  }

  return (
    <div className="space-y-8">
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
        <StatCard title="Health" value={health.ok ? 'ok' : 'down'} tone={health.ok ? 'emerald' : 'rose'} />
        <StatCard title="Readiness" value={ready.ok ? 'ready' : 'waiting'} tone={ready.ok ? 'emerald' : 'amber'} />
        <StatCard title="Repositories" value={String(repositories.length)} tone="blue" />
        <StatCard title="RPM / DEB" value={`${stats.rpm} / ${stats.deb}`} tone="violet" />
        <StatCard title="File repos" value={String(stats.file)} tone="slate" />
      </section>

      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
          {error}
        </div>
      )}

      <section className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h2 className="text-xl font-semibold">Repositories</h2>
              <p className="text-sm text-slate-600">Create, inspect, and upload packages.</p>
            </div>
            <button
              type="button"
              onClick={() => void loadData()}
              className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
            >
              Refresh
            </button>
          </div>

          {loading ? (
            <p className="text-sm text-slate-500">Loading repositories…</p>
          ) : repositories.length === 0 ? (
            <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 p-6 text-sm text-slate-600">
              No repositories yet. Create the first one using the form.
            </div>
          ) : (
            <div className="space-y-4">
              {repositories.map((repo) => {
                const artifacts = artifactsByRepo[repo.slug] ?? []
                return (
                  <article key={repo.id} className="rounded-xl border border-slate-200 p-4">
                    <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                      <div>
                        <div className="flex flex-wrap items-center gap-2">
                          <h3 className="text-lg font-semibold text-slate-900">{repo.name}</h3>
                          <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold uppercase text-slate-700">
                            {repo.type}
                          </span>
                        </div>
                        <p className="mt-1 text-sm text-slate-600">Slug: {repo.slug}</p>
                        <p className="text-xs text-slate-500">Created: {new Date(repo.created_at).toLocaleString()}</p>
                      </div>
                      <a
                        href={examplePath(repo)}
                        target="_blank"
                        rel="noreferrer"
                        className="text-sm font-medium text-blue-700 hover:text-blue-900"
                      >
                        Open public path →
                      </a>
                    </div>

                    {repo.type === 'deb' && (
                      <div className="mt-3 rounded-lg bg-blue-50 p-3 text-sm text-blue-900">
                        Codename: {String(repo.config.codename ?? 'stable')} • Component: {String(repo.config.component ?? 'main')} • Architectures:{' '}
                        {Array.isArray(repo.config.architectures)
                          ? repo.config.architectures.join(', ')
                          : 'amd64'}
                      </div>
                    )}

                    <form onSubmit={(event) => void handleUpload(repo, event)} className="mt-4 grid gap-3 md:grid-cols-[1fr_1fr_auto]">
                      <input
                        name="file"
                        type="file"
                        className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm"
                        required
                      />
                      <input
                        name="path"
                        value={pathByRepo[repo.slug] ?? ''}
                        onChange={(event) =>
                          setPathByRepo((current) => ({ ...current, [repo.slug]: event.target.value }))
                        }
                        placeholder={repo.type === 'file' ? 'Optional path inside files repo' : 'Optional override path'}
                        className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm"
                      />
                      <button
                        type="submit"
                        disabled={uploadingSlug === repo.slug}
                        className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        {uploadingSlug === repo.slug ? 'Uploading…' : 'Upload'}
                      </button>
                    </form>

                    <div className="mt-4">
                      <p className="mb-2 text-sm font-semibold text-slate-800">Recent artifacts</p>
                      {artifacts.length === 0 ? (
                        <p className="text-sm text-slate-500">No uploads recorded yet.</p>
                      ) : (
                        <div className="space-y-2">
                          {artifacts.slice(0, 5).map((artifact) => (
                            <div
                              key={artifact.id}
                              className="flex flex-col justify-between gap-1 rounded-lg bg-slate-50 px-3 py-2 text-sm md:flex-row md:items-center"
                            >
                              <div>
                                <p className="font-medium text-slate-800">{artifact.logical_path}</p>
                                <p className="text-xs text-slate-500">{formatBytes(artifact.size)} • {new Date(artifact.created_at).toLocaleString()}</p>
                              </div>
                              <a
                                href={`${api.repoBaseUrl(repo)}/${artifact.logical_path}`}
                                target="_blank"
                                rel="noreferrer"
                                className="text-sm font-medium text-blue-700 hover:text-blue-900"
                              >
                                Download
                              </a>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </article>
                )
              })}
            </div>
          )}
        </div>

        <div className="space-y-6">
          <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-xl font-semibold">Create repository</h2>
            <p className="mt-1 text-sm text-slate-600">Use lowercase slugs such as rpm-demo or docs-files.</p>

            <form onSubmit={(event) => void handleCreateRepository(event)} className="mt-4 space-y-3">
              <Field label="Name">
                <input
                  value={form.name}
                  onChange={(event) => updateForm('name', event.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2"
                  placeholder="Production RPMs"
                  required
                />
              </Field>

              <Field label="Slug">
                <input
                  value={form.slug}
                  onChange={(event) => updateForm('slug', event.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '-'))}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2"
                  placeholder="production-rpms"
                  required
                />
              </Field>

              <Field label="Type">
                <select
                  value={form.type}
                  onChange={(event) => updateForm('type', event.target.value as RepoType)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2"
                >
                  <option value="rpm">RPM</option>
                  <option value="deb">DEB</option>
                  <option value="file">File</option>
                </select>
              </Field>

              {form.type === 'deb' && (
                <div className="grid gap-3 md:grid-cols-2">
                  <Field label="Codename">
                    <input
                      value={form.codename}
                      onChange={(event) => updateForm('codename', event.target.value)}
                      className="w-full rounded-lg border border-slate-300 px-3 py-2"
                    />
                  </Field>
                  <Field label="Component">
                    <input
                      value={form.component}
                      onChange={(event) => updateForm('component', event.target.value)}
                      className="w-full rounded-lg border border-slate-300 px-3 py-2"
                    />
                  </Field>
                  <div className="md:col-span-2">
                    <Field label="Architectures">
                      <input
                        value={form.architectures}
                        onChange={(event) => updateForm('architectures', event.target.value)}
                        className="w-full rounded-lg border border-slate-300 px-3 py-2"
                        placeholder="amd64, arm64"
                      />
                    </Field>
                  </div>
                </div>
              )}

              <button
                type="submit"
                disabled={creating}
                className="w-full rounded-lg bg-blue-600 px-4 py-2.5 font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {creating ? 'Creating…' : 'Create repository'}
              </button>
            </form>
          </section>

          <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-xl font-semibold">Host tooling</h2>
            <p className="mt-1 text-sm text-slate-600">
              Install RPM and related repo tooling directly from the UI when the service is authorized and running as root.
            </p>
            <button
              type="button"
              onClick={() => void handleInstallTooling()}
              disabled={installing}
              className="mt-4 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {installing ? 'Installing…' : 'Install repo tooling'}
            </button>
            {toolingResult && (
              <div className="mt-4 rounded-xl bg-slate-50 p-3 text-sm text-slate-700">
                <p className="font-semibold text-slate-900">
                  {toolingResult.ok ? 'Installation completed' : 'Installation response'}
                </p>
                {toolingResult.distro && <p>Distro: {toolingResult.distro}</p>}
                {toolingResult.detail && <p>{toolingResult.detail}</p>}
                {toolingResult.error && <p className="text-rose-700">{toolingResult.error}</p>}
                {toolingResult.log && <pre className="mt-2 rounded-lg bg-white p-3 text-xs">{toolingResult.log}</pre>}
              </div>
            )}
          </section>
        </div>
      </section>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      {children}
    </label>
  )
}

function StatCard({ title, value, tone }: { title: string; value: string; tone: 'emerald' | 'rose' | 'amber' | 'blue' | 'violet' | 'slate' }) {
  const tones: Record<string, string> = {
    emerald: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    rose: 'bg-rose-50 text-rose-700 border-rose-200',
    amber: 'bg-amber-50 text-amber-700 border-amber-200',
    blue: 'bg-blue-50 text-blue-700 border-blue-200',
    violet: 'bg-violet-50 text-violet-700 border-violet-200',
    slate: 'bg-slate-100 text-slate-700 border-slate-200',
  }

  return (
    <div className={`rounded-2xl border p-4 ${tones[tone]}`}>
      <p className="text-sm font-medium">{title}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  )
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
