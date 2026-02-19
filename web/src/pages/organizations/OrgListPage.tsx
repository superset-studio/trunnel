import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { listOrganizations } from '../../api/organizations';

export function OrgListPage() {
  const { data: orgs, isLoading, error } = useQuery({
    queryKey: ['organizations'],
    queryFn: listOrganizations,
  });

  if (isLoading) return <div>Loading organizations...</div>;
  if (error) return <div>Failed to load organizations</div>;

  return (
    <div className="page">
      <h2>Organizations</h2>
      {orgs && orgs.length === 0 && <p>No organizations found.</p>}
      <div className="org-list">
        {orgs?.map((org) => (
          <Link
            key={org.id}
            to="/organizations/$orgId"
            params={{ orgId: org.id }}
            className="org-card"
          >
            <h3>{org.displayName}</h3>
            <span className="org-slug">{org.name}</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
