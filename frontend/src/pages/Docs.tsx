export function Docs() {
  return (
    <div className="space-y-6">
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h1 className="text-2xl font-semibold">Repository usage</h1>
        <p className="mt-2 text-sm text-slate-600">
          RepoForge publishes package and file content over plain HTTP. Use the examples below after uploading artifacts.
        </p>
      </section>

      <div className="grid gap-6 lg:grid-cols-3">
        <DocCard
          title="RPM repositories"
          body="Point DNF or YUM at the rpms path for your repository. createrepo_c metadata is generated beside the uploaded packages."
          snippet={`[repoforge]\nname=RepoForge\nbaseurl=http://YOUR_HOST:8080/repo/rpm-demo/rpms\nenabled=1\ngpgcheck=0`}
        />
        <DocCard
          title="DEB repositories"
          body="APT clients should use the repository codename and component configured when the repo was created."
          snippet={`deb [trusted=yes] http://YOUR_HOST:8080/repo/deb-demo stable main`}
        />
        <DocCard
          title="File repositories"
          body="Use file repositories for generic release artifacts, notes, or tarballs with optional folder paths."
          snippet={`curl -O http://YOUR_HOST:8080/repo/files-demo/files/releases/app.tar.gz`}
        />
      </div>

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-xl font-semibold">Notes</h2>
        <ul className="mt-3 list-disc space-y-2 pl-5 text-sm text-slate-700">
          <li>Use the bearer token field in the header when the service has REPOFORGE_TOKEN enabled.</li>
          <li>The tooling installer endpoint requires root privileges and explicit confirmation.</li>
          <li>Uploads are sent using multipart form data with the file field named file.</li>
        </ul>
      </section>
    </div>
  )
}

function DocCard({ title, body, snippet }: { title: string; body: string; snippet: string }) {
  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <h2 className="text-lg font-semibold">{title}</h2>
      <p className="mt-2 text-sm text-slate-600">{body}</p>
      <pre className="mt-4 rounded-xl bg-slate-950 p-4 text-xs text-slate-100">{snippet}</pre>
    </section>
  )
}
