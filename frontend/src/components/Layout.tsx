import { useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { clearStoredToken, getStoredToken, setStoredToken } from '../api'

interface LayoutProps {
  children: React.ReactNode
}

export function Layout({ children }: LayoutProps) {
  const location = useLocation()
  const [token, setToken] = useState(getStoredToken())
  const [saved, setSaved] = useState(false)
  const [tokenOpen, setTokenOpen] = useState(false)

  const links = useMemo(
    () => [
      { to: '/', label: 'Control', desc: 'Repos & uploads' },
      { to: '/docs', label: 'Clients', desc: 'APT, DNF, curl' },
    ],
    [],
  )

  const saveToken = () => {
    setStoredToken(token.trim())
    setSaved(true)
    window.setTimeout(() => setSaved(false), 1800)
  }

  const resetToken = () => {
    clearStoredToken()
    setToken('')
    setSaved(false)
  }

  return (
    <div className="flex min-h-screen flex-col bg-rf-void text-rf-fg md:flex-row">
      <aside className="flex w-full shrink-0 flex-col border-b border-rf-border bg-rf-surface md:w-[280px] md:border-b-0 md:border-r">
        <div className="border-b border-rf-border px-5 py-6">
          <Link to="/" className="block">
            <span className="font-mono text-xs font-medium tracking-[0.2em] text-rf-accent">REPOFORGE</span>
            <h1 className="mt-1 text-2xl font-semibold tracking-tight">Artifact hub</h1>
            <p className="mt-2 text-sm leading-relaxed text-rf-muted">
              RPM, DEB, and static trees — one HTTP surface.
            </p>
          </Link>
        </div>

        <nav className="flex flex-col gap-1 p-3">
          {links.map((link) => {
            const active = location.pathname === link.to
            return (
              <Link
                key={link.to}
                to={link.to}
                className={`rounded-lg px-3 py-2.5 transition-colors ${
                  active
                    ? 'bg-rf-elevated text-rf-fg ring-1 ring-rf-accent/30'
                    : 'text-rf-muted hover:bg-rf-elevated/60 hover:text-rf-fg'
                }`}
              >
                <span className="block text-sm font-medium">{link.label}</span>
                <span className="mt-0.5 block text-xs text-rf-muted">{link.desc}</span>
              </Link>
            )
          })}
        </nav>

        <div className="mt-auto border-t border-rf-border p-3">
          <button
            type="button"
            onClick={() => setTokenOpen((v) => !v)}
            className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm font-medium text-rf-muted hover:bg-rf-elevated hover:text-rf-fg"
          >
            <span>API token</span>
            <span className="font-mono text-xs text-rf-accent">{tokenOpen ? '−' : '+'}</span>
          </button>
          {tokenOpen && (
            <div className="mt-2 space-y-2 rounded-lg border border-rf-border bg-rf-elevated p-3">
              <p className="text-xs leading-relaxed text-rf-muted">
                Stored in this browser. Required when the server sets <span className="font-mono text-rf-fg/80">REPOFORGE_TOKEN</span>.
              </p>
              <input
                type="password"
                value={token}
                onChange={(event) => setToken(event.target.value)}
                placeholder="Bearer token"
                className="w-full rounded-md border border-rf-border bg-rf-surface px-2.5 py-2 font-mono text-sm text-rf-fg outline-none ring-0 placeholder:text-rf-muted focus:border-rf-accent/50 focus:ring-1 focus:ring-rf-accent/40"
              />
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={saveToken}
                  className="flex-1 rounded-md bg-rf-accent px-3 py-2 text-sm font-medium text-rf-void hover:brightness-110"
                >
                  Save
                </button>
                <button
                  type="button"
                  onClick={resetToken}
                  className="rounded-md border border-rf-border px-3 py-2 text-sm text-rf-muted hover:border-rf-muted hover:text-rf-fg"
                >
                  Clear
                </button>
              </div>
              {saved && <p className="text-xs font-medium text-rf-ok">Saved locally.</p>}
            </div>
          )}
        </div>
      </aside>

      <div className="relative flex min-w-0 flex-1 flex-col">
        <div
          className="pointer-events-none absolute inset-0 opacity-[0.04]"
          style={{
            backgroundImage:
              'linear-gradient(to right, #fff 1px, transparent 1px), linear-gradient(to bottom, #fff 1px, transparent 1px)',
            backgroundSize: '48px 48px',
          }}
        />
        <main className="relative z-[1] flex-1 px-4 py-8 sm:px-8 lg:px-12">{children}</main>
      </div>
    </div>
  )
}
