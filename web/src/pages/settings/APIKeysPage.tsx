import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useState } from 'react';
import { createAPIKey, listAPIKeys, revokeAPIKey } from '../../api/api-keys';

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

  if (isLoading) return <div>Loading API keys...</div>;

  return (
    <div className="page">
      <h2>API Keys</h2>

      <div className="create-key-form">
        <h3>Create API Key</h3>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            setNewKey(null);
            createMutation.mutate();
          }}
        >
          <input
            type="text"
            placeholder="Key name"
            value={keyName}
            onChange={(e) => setKeyName(e.target.value)}
            required
          />
          <select value={accessLevel} onChange={(e) => setAccessLevel(e.target.value)}>
            <option value="read">Read</option>
            <option value="write">Write</option>
            <option value="admin">Admin</option>
          </select>
          <button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? 'Creating...' : 'Create'}
          </button>
        </form>
      </div>

      {newKey && (
        <div className="new-key-display">
          <p>
            <strong>Your new API key (copy it now — it won't be shown again):</strong>
          </p>
          <code className="key-value">{newKey}</code>
          <button onClick={() => navigator.clipboard.writeText(newKey)}>
            Copy
          </button>
        </div>
      )}

      <table className="keys-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Prefix</th>
            <th>Access</th>
            <th>Created</th>
            <th>Last Used</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {keys?.map((key) => (
            <tr key={key.id}>
              <td>{key.name}</td>
              <td><code>{key.keyPrefix}...</code></td>
              <td>{key.accessLevel}</td>
              <td>{new Date(key.createdAt).toLocaleDateString()}</td>
              <td>
                {key.lastUsedAt
                  ? new Date(key.lastUsedAt).toLocaleDateString()
                  : 'Never'}
              </td>
              <td>
                <button
                  onClick={() => revokeMutation.mutate(key.id)}
                  disabled={revokeMutation.isPending}
                  className="btn-danger"
                >
                  Revoke
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
