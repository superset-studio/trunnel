import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { getOrganization } from '../../api/organizations';

export function OrgDashboardPage() {
  const { orgId } = useParams({ strict: false }) as { orgId: string };

  const { data: org, isLoading } = useQuery({
    queryKey: ['organization', orgId],
    queryFn: () => getOrganization(orgId),
    enabled: !!orgId,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-8 w-8 border-2 border-brand-500 border-t-transparent" />
      </div>
    );
  }

  return (
    <div>
      <h2 className="text-2xl font-semibold text-slate-800 mb-4">
        {org?.displayName ?? 'Organization'}
      </h2>
      <p className="text-sm text-slate-500">Dashboard coming in Phase 3.</p>
    </div>
  );
}
