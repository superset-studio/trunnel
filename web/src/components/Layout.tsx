import { Link, Outlet } from '@tanstack/react-router';
import { useAuth } from '../hooks/useAuth';

export function Layout() {
  const { user, organization, logout } = useAuth();

  return (
    <div className="layout">
      <nav className="navbar">
        <div className="nav-left">
          <Link to="/" className="nav-brand">
            Kapstan
          </Link>
          {organization && (
            <span className="nav-org">{organization.displayName}</span>
          )}
        </div>
        <div className="nav-right">
          {user && (
            <>
              <span className="nav-user">{user.name}</span>
              <button onClick={logout} className="nav-logout">
                Logout
              </button>
            </>
          )}
        </div>
      </nav>
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  );
}
