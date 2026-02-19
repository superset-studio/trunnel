import {
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import { Layout } from './components/Layout';
import { ProtectedRoute } from './components/ProtectedRoute';
import { LoginPage } from './pages/auth/LoginPage';
import { RegisterPage } from './pages/auth/RegisterPage';
import { OrgListPage } from './pages/organizations/OrgListPage';
import { OrgDashboardPage } from './pages/organizations/OrgDashboardPage';
import { MembersPage } from './pages/settings/MembersPage';
import { APIKeysPage } from './pages/settings/APIKeysPage';
import { ConnectionsPage } from './pages/connections/ConnectionsPage';

const rootRoute = createRootRoute({
  component: Layout,
});

// Public auth routes.
const authLoginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/login',
  component: LoginPage,
});

const authRegisterRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/register',
  component: RegisterPage,
});

// Protected routes.
const protectedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'protected',
  component: ProtectedRoute,
});

const indexRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/',
  component: OrgListPage,
});

const orgDashboardRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/organizations/$orgId',
  component: OrgDashboardPage,
});

const membersRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/organizations/$orgId/members',
  component: MembersPage,
});

const apiKeysRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/organizations/$orgId/api-keys',
  component: APIKeysPage,
});

const connectionsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/organizations/$orgId/connections',
  component: ConnectionsPage,
});

const routeTree = rootRoute.addChildren([
  authLoginRoute,
  authRegisterRoute,
  protectedRoute.addChildren([
    indexRoute,
    orgDashboardRoute,
    membersRoute,
    apiKeysRoute,
    connectionsRoute,
  ]),
]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
