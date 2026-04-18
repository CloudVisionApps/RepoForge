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

  const links = useMemo(
    () => [
      { to: '/', label: 'Repositories' },
      { to: '/docs', label: 'Usage help' },
    ],
    [],
  )

  const saveToken = () => {
    setStoredToken(token.trim())
    setSaved(true)
    window.setTimeout(() => setSaved(false), 1500)
  }

  const resetToken = () => {
    clearStoredToken()
    setToken('')
    setSaved(false)
  }

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <header className="border-b border-slate-200 bg-white/95 backdrop-blur">
        <div className="mx-auto max-w-7xl px-4 py-4 sm:px-6 lg:px-8">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <Link to="/" className="text-3xl font-bold tracking-tight text-slate-950">
                RepoForge
              </Link>
              <p className="mt-1 text-sm text-slate-600">
                Manage RPM, DEB, and file repositories from one panel.
              </p>
            </div>

            <nav className="flex flex-wrap gap-2">
              {links.map((link) => {
                const active = location.pathname === link.to
                return (
                  <Link
                    key={link.to}
                    to={link.to}
                    className={`rounded-lg px-4 py-2 text-sm font-medium transition ${
                      active
                        ? 'bg-slate-900 text-white'
                        : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
                    }`}
                  >
                    {link.label}
                  </Link>
                )
              })}
            </nav>
          </div>

          <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-3">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <p className="text-sm font-semibold text-slate-800">Bearer token</p>
                <p className="text-xs text-slate-600">
                  Only needed when REPOFORGE_TOKEN is enabled on the service.
                </p>
              </div>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <input
                  type="password"
                  value={token}
                  onChange={(event) => setToken(event.target.value)}
                  placeholder="Paste API token"
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none ring-0 sm:w-72"
                />
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={saveToken}
                    className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                  >
                    Save
                  </button>
                  <button
                    type="button"
                    onClick={resetToken}
                    className="rounded-lg bg-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-300"
                  >
                    Clear
                  </button>
                </div>
              </div>
            </div>
            {saved && <p className="mt-2 text-xs font-medium text-emerald-700">Token saved.</p>}
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">{children}</main>
    </div>
  )
}
