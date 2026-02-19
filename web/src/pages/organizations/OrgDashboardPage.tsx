import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from '@tanstack/react-router';
import { getOrganization } from '../../api/organizations';

export function OrgDashboardPage() {
  const { orgId } = useParams({ strict: false }) as { orgId: string };

  const { data: org, isLoading } = useQuery({
    queryKey: ['organization', orgId],
    queryFn: () => getOrganization(orgId),
    enabled: !!orgId,
  });

  if (isLoading) return <div>Loading...</div>;

  return (
    <div className="page">
      <h2>{org?.displayName ?? 'Organization'}</h2>
      <nav className="org-nav">
        <Link to="/organizations/$orgId/members" params={{ orgId }}>
          Members
        </Link>
        <Link to="/organizations/$orgId/api-keys" params={{ orgId }}>
          API Keys
        </Link>
      </nav>
      <p>Dashboard coming in Phase 3.</p>
    </div>
  );
}
