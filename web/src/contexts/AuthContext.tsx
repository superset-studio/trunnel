import {
  createContext,
  useCallback,
  useEffect,
  useState,
  type ReactNode,
} from 'react';
import {
  login as apiLogin,
  register as apiRegister,
  refresh as apiRefresh,
  type AuthResponse,
} from '../api/auth';
import { setAccessToken } from '../api/client';

interface User {
  id: string;
  email: string;
  name: string;
}

interface Organization {
  id: string;
  name: string;
  displayName: string;
}

interface AuthContextValue {
  user: User | null;
  organization: Organization | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (
    email: string,
    password: string,
    name: string,
    orgName: string
  ) => Promise<void>;
  logout: () => void;
}

export const AuthContext = createContext<AuthContextValue | null>(null);

function handleAuthResponse(data: AuthResponse) {
  setAccessToken(data.accessToken);
  localStorage.setItem('refreshToken', data.refreshToken);
  return {
    user: {
      id: data.user.id,
      email: data.user.email,
      name: data.user.name,
    },
    organization: {
      id: data.organization.id,
      name: data.organization.name,
      displayName: data.organization.displayName,
    },
  };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [organization, setOrganization] = useState<Organization | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check if we have a refresh token on mount.
    const refreshToken = localStorage.getItem('refreshToken');
    if (!refreshToken) {
      setIsLoading(false);
      return;
    }
    // Try to refresh the session.
    apiRefresh(refreshToken)
      .then((data) => {
        setAccessToken(data.accessToken);
        localStorage.setItem('refreshToken', data.refreshToken);
        // We don't get user/org from refresh, so we stay logged out visually
        // until next page load with full auth. For a better UX we'd call /me.
        // For now, mark as not loading — the token interceptor handles the rest.
        setIsLoading(false);
      })
      .catch(() => {
        localStorage.removeItem('refreshToken');
        setIsLoading(false);
      });
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const data = await apiLogin({ email, password });
    const { user: u, organization: org } = handleAuthResponse(data);
    setUser(u);
    setOrganization(org);
  }, []);

  const register = useCallback(
    async (email: string, password: string, name: string, orgName: string) => {
      const data = await apiRegister({ email, password, name, orgName });
      const { user: u, organization: org } = handleAuthResponse(data);
      setUser(u);
      setOrganization(org);
    },
    []
  );

  const logout = useCallback(() => {
    setUser(null);
    setOrganization(null);
    setAccessToken(null);
    localStorage.removeItem('refreshToken');
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        organization,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
