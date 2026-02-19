import { Link, Outlet, useMatches } from '@tanstack/react-router';
import { useAuth } from '../hooks/useAuth';

export function Layout() {
  const { user, organization, logout, isAuthenticated } = useAuth();
  const matches = useMatches();

  // Extract orgId from route params if we're inside an org route
  const orgMatch = matches.find(
    (m) => m.pathname.startsWith('/organizations/') && 'orgId' in (m.params as Record<string, string>)
  );
  const orgId = orgMatch ? (orgMatch.params as { orgId: string }).orgId : null;

  // Determine current path for active nav highlighting
  const currentPath = matches[matches.length - 1]?.pathname ?? '';

  // Auth pages: no sidebar, just centered content
  if (!isAuthenticated) {
    return <Outlet />;
  }

  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <aside className="fixed left-0 top-0 h-screen w-64 bg-slate-900 text-slate-300 flex flex-col z-40">
        {/* Brand */}
        <div className="px-5 py-5">
          <Link to="/" className="flex items-center gap-2 no-underline">
            <span className="flex items-center justify-center w-8 h-8 rounded-lg bg-brand-500 text-white font-bold text-sm">
              K
            </span>
            <span className="text-xl font-bold text-brand-400">Kapstan</span>
          </Link>
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-3 space-y-1">
          {/* Top-level nav */}
          <Link
            to="/"
            className={`block px-3 py-2 rounded-md text-sm no-underline transition-colors ${
              currentPath === '/'
                ? 'bg-slate-800 text-brand-400 border-l-2 border-brand-500'
                : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
            }`}
          >
            Organizations
          </Link>

          {/* Org-scoped nav */}
          {orgId && (
            <>
              <div className="pt-4 pb-1 px-3">
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">
                  {organization?.displayName ?? 'Organization'}
                </p>
              </div>
              <Link
                to="/organizations/$orgId"
                params={{ orgId }}
                className={`block px-3 py-2 rounded-md text-sm no-underline transition-colors ${
                  currentPath === `/organizations/${orgId}`
                    ? 'bg-slate-800 text-brand-400 border-l-2 border-brand-500'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
                }`}
              >
                Dashboard
              </Link>
              <Link
                to="/organizations/$orgId/connections"
                params={{ orgId }}
                className={`block px-3 py-2 rounded-md text-sm no-underline transition-colors ${
                  currentPath === `/organizations/${orgId}/connections`
                    ? 'bg-slate-800 text-brand-400 border-l-2 border-brand-500'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
                }`}
              >
                Connections
              </Link>
              <Link
                to="/organizations/$orgId/members"
                params={{ orgId }}
                className={`block px-3 py-2 rounded-md text-sm no-underline transition-colors ${
                  currentPath === `/organizations/${orgId}/members`
                    ? 'bg-slate-800 text-brand-400 border-l-2 border-brand-500'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
                }`}
              >
                Members
              </Link>
              <Link
                to="/organizations/$orgId/api-keys"
                params={{ orgId }}
                className={`block px-3 py-2 rounded-md text-sm no-underline transition-colors ${
                  currentPath === `/organizations/${orgId}/api-keys`
                    ? 'bg-slate-800 text-brand-400 border-l-2 border-brand-500'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
                }`}
              >
                API Keys
              </Link>
            </>
          )}
        </nav>

        {/* User section at bottom */}
        {user && (
          <div className="border-t border-slate-700 px-5 py-4">
            <p className="text-sm font-medium text-slate-300 truncate">{user.name}</p>
            <p className="text-xs text-slate-500 truncate">{user.email}</p>
          </div>
        )}
      </aside>

      {/* Main area */}
      <div className="ml-64 flex-1 flex flex-col min-h-screen">
        {/* Top header bar */}
        <header className="bg-white border-b border-stone-200 h-14 flex items-center justify-between px-6 sticky top-0 z-30">
          <div>
            {organization && orgId && (
              <span className="text-sm text-slate-500">{organization.displayName}</span>
            )}
          </div>
          <div className="flex items-center gap-4">
            {user && (
              <>
                <span className="text-sm text-slate-600">{user.name}</span>
                <button
                  onClick={logout}
                  className="text-sm text-slate-500 hover:text-slate-800 cursor-pointer transition-colors"
                >
                  Logout
                </button>
              </>
            )}
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 bg-stone-50 p-6">
          <div className="max-w-5xl mx-auto">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
