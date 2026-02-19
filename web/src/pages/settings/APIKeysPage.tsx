import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useState } from 'react';
import { createAPIKey, listAPIKeys, revokeAPIKey } from '../../api/api-keys';
import { Tooltip } from '../../components/Tooltip';

const accessDescriptions: Record<string, string> = {
  read: 'Can read resources only',
  write: 'Can read and modify resources',
  admin: 'Full access including settings',
};

export function APIKeysPage() {
  const { orgId } = useParams({ strict: false }) as { orgId: string };
  const queryClient = useQueryClient();
  const [keyName, setKeyName] = useState('');
  const [accessLevel, setAccessLevel] = useState('read');
  const [newKey, setNewKey] = useState<string | null>(null);

  const { data: keys, isLoading } = useQuery({
    queryKey: ['api-keys', orgId],
    queryFn: () => listAPIKeys(orgId),
    enabled: !!orgId,
  });

  const createMutation = useMutation({
    mutationFn: () => createAPIKey(orgId, { name: keyName, accessLevel }),
    onSuccess: (data) => {
      setNewKey(data.key);
      setKeyName('');
      queryClient.invalidateQueries({ queryKey: ['api-keys', orgId] });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: (keyId: string) => revokeAPIKey(orgId, keyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys', orgId] });
    },
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
      <h2 className="text-2xl font-semibold text-slate-800 mb-6">API Keys</h2>

      {/* Create key form */}
      <div className="bg-white rounded-lg border border-stone-200 p-5 mb-6">
        <h3 className="text-sm font-semibold text-slate-800 mb-3">Create API Key</h3>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            setNewKey(null);
            createMutation.mutate();
          }}
          className="flex items-center gap-3 flex-wrap"
        >
          <input
            type="text"
            placeholder="Key name"
            value={keyName}
            onChange={(e) => setKeyName(e.target.value)}
            required
            className="flex-1 min-w-[200px] rounded-md border border-stone-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
          />
          <select
            value={accessLevel}
            onChange={(e) => setAccessLevel(e.target.value)}
            className="rounded-md border border-stone-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
          >
            <option value="read">Read</option>
            <option value="write">Write</option>
            <option value="admin">Admin</option>
          </select>
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="bg-brand-500 hover:bg-brand-600 text-white text-sm font-medium px-4 py-2 rounded-md transition-colors disabled:opacity-50 cursor-pointer"
          >
            {createMutation.isPending ? 'Creating...' : 'Create'}
          </button>
        </form>
      </div>

      {/* New key banner */}
      {newKey && (
        <div className="bg-brand-50 border border-brand-200 rounded-lg p-4 mb-6">
          <p className="text-sm font-semibold text-slate-800 mb-2">
            Your new API key (copy it now — it won't be shown again):
          </p>
          <div className="bg-slate-100 font-mono text-sm p-3 rounded-md break-all mb-3">
            {newKey}
          </div>
          <Tooltip content="Copied!">
            <button
              onClick={() => navigator.clipboard.writeText(newKey)}
              className="bg-brand-500 hover:bg-brand-600 text-white text-sm px-3 py-1 rounded-md transition-colors cursor-pointer"
            >
              Copy
            </button>
          </Tooltip>
        </div>
      )}

      {/* Keys table */}
      <div className="bg-white rounded-lg border border-stone-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="bg-stone-50">
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Name</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Prefix</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                <Tooltip content="Read < Write < Admin">
                  <span className="border-b border-dashed border-slate-400 cursor-help">Access</span>
                </Tooltip>
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Created</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Last Used</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody>
            {keys?.map((key) => (
              <tr key={key.id} className="border-t border-stone-100">
                <td className="px-4 py-3 text-sm text-slate-800">{key.name}</td>
                <td className="px-4 py-3 text-sm">
                  <code className="bg-slate-100 px-1.5 py-0.5 rounded text-xs font-mono">{key.keyPrefix}...</code>
                </td>
                <td className="px-4 py-3 text-sm">
                  <Tooltip content={accessDescriptions[key.accessLevel] ?? ''}>
                    <span className="border-b border-dashed border-slate-300 cursor-help text-slate-600">
                      {key.accessLevel}
                    </span>
                  </Tooltip>
                </td>
                <td className="px-4 py-3 text-sm text-slate-500">
                  {new Date(key.createdAt).toLocaleDateString()}
                </td>
                <td className="px-4 py-3 text-sm text-slate-500">
                  {key.lastUsedAt
                    ? new Date(key.lastUsedAt).toLocaleDateString()
                    : 'Never'}
                </td>
                <td className="px-4 py-3 text-sm">
                  <button
                    onClick={() => revokeMutation.mutate(key.id)}
                    disabled={revokeMutation.isPending}
                    className="text-red-600 hover:bg-red-50 rounded px-2 py-1 text-sm transition-colors cursor-pointer"
                  >
                    Revoke
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
