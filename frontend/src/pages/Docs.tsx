const snippets = [
  {
    title: 'RPM / DNF',
    body: 'Point baseurl at the rpms directory. Metadata is regenerated beside uploads.',
    path: '/repo/rpm-demo/rpms',
    code: `[repoforge]
name=RepoForge
baseurl=http://YOUR_HOST:8080/repo/rpm-demo/rpms
enabled=1
gpgcheck=0`,
  },
  {
    title: 'DEB / APT',
    body: 'Use the codename and component from the repository record.',
    path: '/repo/deb-demo/dists/…',
    code: `deb [trusted=yes] http://YOUR_HOST:8080/repo/deb-demo stable main`,
  },
  {
    title: 'Static files',
    body: 'Plain GET for tarballs, checksums, or release notes with optional subpaths.',
    path: '/repo/files-demo/files/…',
    code: `curl -fLO http://YOUR_HOST:8080/repo/files-demo/files/releases/app.tar.gz`,
  },
] as const

export function Docs() {
  return (
    <div className="mx-auto max-w-6xl space-y-12 pb-12">
      <header className="max-w-2xl border-b border-rf-border pb-10">
        <p className="font-mono text-xs text-rf-accent">/repo/{'{slug}'}/…</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight sm:text-4xl">Client wiring</h1>
        <p className="mt-3 text-sm leading-relaxed text-rf-muted">
          RepoForge only serves HTTP trees. Swap host and slug; upload artifacts first so indexes and paths exist.
        </p>
      </header>

      <ol className="relative space-y-0 border-l border-rf-border pl-8">
        {snippets.map((item, index) => (
          <li key={item.title} className="relative pb-14 last:pb-0">
            <span className="absolute -left-[9px] top-1.5 h-4 w-4 rounded-full border-2 border-rf-accent bg-rf-void" />
            <span className="font-mono text-[10px] uppercase tracking-widest text-rf-muted">
              Step {index + 1}
            </span>
            <h2 className="mt-1 text-xl font-semibold">{item.title}</h2>
            <p className="mt-2 max-w-xl text-sm text-rf-muted">{item.body}</p>
            <p className="mt-3 font-mono text-xs text-rf-accent/90">{item.path}</p>
            <pre className="mt-4 overflow-x-auto rounded-xl border border-rf-border bg-rf-elevated p-5 font-mono text-xs leading-relaxed text-rf-fg/95 shadow-inner">
              {item.code}
            </pre>
          </li>
        ))}
      </ol>

      <section className="rounded-xl border border-rf-border bg-rf-elevated/60 p-6 sm:p-8">
        <h2 className="text-lg font-semibold">Operational notes</h2>
        <ul className="mt-4 space-y-3 text-sm text-rf-muted">
          <li className="flex gap-3">
            <span className="font-mono text-rf-accent">01</span>
            <span>
              When <span className="font-mono text-rf-fg/90">REPOFORGE_TOKEN</span> is set, every mutating and listing{' '}
              <span className="font-mono text-rf-fg/90">/v1</span> call expects{' '}
              <span className="font-mono text-rf-fg/90">Authorization: Bearer …</span>.
            </span>
          </li>
          <li className="flex gap-3">
            <span className="font-mono text-rf-accent">02</span>
            <span>
              The tooling installer requires root, a configured token, and explicit confirmation from the API.
            </span>
          </li>
          <li className="flex gap-3">
            <span className="font-mono text-rf-accent">03</span>
            <span>Uploads are multipart with field name file; optional path for file repositories.</span>
          </li>
        </ul>
      </section>
    </div>
  )
}
