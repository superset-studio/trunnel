import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { listOrganizations } from '../../api/organizations';

export function OrgListPage() {
  const { data: orgs, isLoading, error } = useQuery({
    queryKey: ['organizations'],
    queryFn: listOrganizations,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-8 w-8 border-2 border-brand-500 border-t-transparent" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 text-red-700 border border-red-200 rounded-md px-4 py-3 text-sm">
        Failed to load organizations
      </div>
    );
  }

  return (
    <div>
      <h2 className="text-2xl font-semibold text-slate-800 mb-6">Organizations</h2>

      {orgs && orgs.length === 0 && (
        <p className="text-sm text-slate-500">No organizations found.</p>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {orgs?.map((org) => (
          <Link
            key={org.id}
            to="/organizations/$orgId"
            params={{ orgId: org.id }}
            className="bg-white rounded-lg border border-stone-200 p-5 hover:shadow-md hover:border-brand-300 transition-all block no-underline"
          >
            <h3 className="font-semibold text-slate-800">{org.displayName}</h3>
            <p className="text-sm text-slate-500 mt-1">{org.name}</p>
          </Link>
        ))}
      </div>
    </div>
  );
}
