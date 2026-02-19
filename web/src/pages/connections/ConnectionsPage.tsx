import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useState } from 'react';
import {
  createConnection,
  deleteConnection,
  listConnections,
  validateConnection,
} from '../../api/connections';
import type { Connection } from '../../api/connections';
import { Select } from '../../components/Select';
import { InfoIcon, Tooltip } from '../../components/Tooltip';

type AuthMethod = 'access-keys' | 'iam-role';

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  valid: 'bg-green-100 text-green-800',
  invalid: 'bg-red-100 text-red-800',
  expired: 'bg-slate-100 text-slate-600',
};

const regionOptions = [
  'us-east-1',
  'us-east-2',
  'us-west-1',
  'us-west-2',
  'eu-west-1',
  'eu-west-2',
  'eu-central-1',
  'ap-southeast-1',
  'ap-northeast-1',
].map((r) => ({ value: r, label: r }));

function getAuthLabel(conn: Connection): string {
  const method = conn.config?.authMethod;
  if (method === 'iam-role') return 'IAM Role';
  return 'Access Keys';
}

export function ConnectionsPage() {
  const { orgId } = useParams({ strict: false }) as { orgId: string };
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [region, setRegion] = useState('us-east-1');
  const [authMethod, setAuthMethod] = useState<AuthMethod>('access-keys');
  const [accessKeyId, setAccessKeyId] = useState('');
  const [secretAccessKey, setSecretAccessKey] = useState('');
  const [roleArn, setRoleArn] = useState('');
  const [externalId, setExternalId] = useState('');

  const { data: connections, isLoading } = useQuery({
    queryKey: ['connections', orgId],
    queryFn: () => listConnections(orgId),
    enabled: !!orgId,
  });

  const createMutation = useMutation({
    mutationFn: () => {
      const creds: Record<string, string> =
        authMethod === 'access-keys'
          ? { accessKeyId, secretAccessKey, region }
          : { roleArn, externalId, region };
      return createConnection(orgId, {
        name,
        category: 'aws',
        credentials: creds,
        config: { region, authMethod },
      });
    },
    onSuccess: () => {
      setName('');
      setAccessKeyId('');
      setSecretAccessKey('');
      setRoleArn('');
      setExternalId('');
      queryClient.invalidateQueries({ queryKey: ['connections', orgId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (connId: string) => deleteConnection(orgId, connId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['connections', orgId] });
    },
  });

  const validateMutation = useMutation({
    mutationFn: (connId: string) => validateConnection(orgId, connId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['connections', orgId] });
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
      <h2 className="text-2xl font-semibold text-slate-800 mb-6 flex items-center gap-2">
        Connections
        <Tooltip
          wide
          content="A connection links Kapstan to your cloud account so it can provision and manage infrastructure on your behalf. Credentials are encrypted at rest."
        >
          <InfoIcon />
        </Tooltip>
      </h2>

      {/* Create connection form */}
      <div className="bg-white rounded-lg border border-stone-200 p-5 mb-6">
        <h3 className="text-sm font-semibold text-slate-800 mb-3">Add AWS Connection</h3>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate();
          }}
          className="space-y-3"
        >
          <div className="flex items-center gap-3 flex-wrap">
            <input
              type="text"
              placeholder="Connection name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="flex-1 min-w-[200px] rounded-md border border-stone-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
            />
            <Select
              value={region}
              onChange={setRegion}
              options={regionOptions}
              className="min-w-[160px]"
            />
          </div>

          {/* Auth method toggle */}
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1 bg-stone-100 rounded-md p-0.5 w-fit">
              <button
                type="button"
                onClick={() => setAuthMethod('access-keys')}
                className={`px-3 py-1.5 text-sm font-medium rounded transition-colors cursor-pointer ${
                  authMethod === 'access-keys'
                    ? 'bg-white text-slate-800 shadow-sm'
                    : 'text-slate-500 hover:text-slate-700'
                }`}
              >
                Access Keys
              </button>
              <button
                type="button"
                onClick={() => setAuthMethod('iam-role')}
                className={`px-3 py-1.5 text-sm font-medium rounded transition-colors cursor-pointer ${
                  authMethod === 'iam-role'
                    ? 'bg-white text-slate-800 shadow-sm'
                    : 'text-slate-500 hover:text-slate-700'
                }`}
              >
                IAM Role
              </button>
            </div>
            <Tooltip
              wide
              content={
                authMethod === 'access-keys' ? (
                  <span>
                    Create an IAM user with programmatic access. The user needs
                    these managed policies:
                    <br />
                    <br />
                    <strong>Required:</strong> AmazonEC2FullAccess, AmazonEKSClusterPolicy, AmazonVPCFullAccess, IAMFullAccess
                    <br />
                    <br />
                    <strong>Optional:</strong> AmazonS3FullAccess, AmazonRDSFullAccess (for managed databases)
                  </span>
                ) : (
                  <span>
                    Create an IAM role in your AWS account that trusts
                    Kapstan's account. Set the External ID to a unique secret you choose.
                    <br />
                    <br />
                    <strong>Trust policy:</strong> Allow sts:AssumeRole from Kapstan's AWS account with the External ID condition.
                    <br />
                    <br />
                    <strong>Permissions:</strong> Attach the same policies as Access Keys (EC2, EKS, VPC, IAM).
                  </span>
                )
              }
            >
              <InfoIcon />
            </Tooltip>
          </div>

          <div className="flex items-center gap-3 flex-wrap">
            {authMethod === 'access-keys' ? (
              <>
                <input
                  type="text"
                  placeholder="Access Key ID"
                  value={accessKeyId}
                  onChange={(e) => setAccessKeyId(e.target.value)}
                  required
                  className="flex-1 min-w-[200px] rounded-md border border-stone-300 px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                />
                <input
                  type="password"
                  placeholder="Secret Access Key"
                  value={secretAccessKey}
                  onChange={(e) => setSecretAccessKey(e.target.value)}
                  required
                  className="flex-1 min-w-[200px] rounded-md border border-stone-300 px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                />
              </>
            ) : (
              <>
                <input
                  type="text"
                  placeholder="Role ARN (arn:aws:iam::123456789012:role/...)"
                  value={roleArn}
                  onChange={(e) => setRoleArn(e.target.value)}
                  required
                  className="flex-1 min-w-[200px] rounded-md border border-stone-300 px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                />
                <input
                  type="text"
                  placeholder="External ID"
                  value={externalId}
                  onChange={(e) => setExternalId(e.target.value)}
                  required
                  className="flex-1 min-w-[200px] rounded-md border border-stone-300 px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                />
              </>
            )}
            <button
              type="submit"
              disabled={createMutation.isPending}
              className="bg-brand-500 hover:bg-brand-600 text-white text-sm font-medium px-4 py-2 rounded-md transition-colors disabled:opacity-50 cursor-pointer"
            >
              {createMutation.isPending ? 'Adding...' : 'Add Connection'}
            </button>
          </div>
          {createMutation.isError && (
            <p className="text-sm text-red-600">
              Failed to create connection. Please check your inputs.
            </p>
          )}
        </form>
      </div>

      {/* Connections table */}
      <div className="bg-white rounded-lg border border-stone-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="bg-stone-50">
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                Name
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                Provider
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                Auth Type
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                Status
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                Region
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                Last Validated
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {connections && connections.length > 0 ? (
              connections.map((conn: Connection) => (
                <tr key={conn.id} className="border-t border-stone-100">
                  <td className="px-4 py-3 text-sm text-slate-800">{conn.name}</td>
                  <td className="px-4 py-3 text-sm text-slate-600 uppercase">{conn.category}</td>
                  <td className="px-4 py-3 text-sm text-slate-600">{getAuthLabel(conn)}</td>
                  <td className="px-4 py-3 text-sm">
                    <span
                      className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${statusColors[conn.status] ?? 'bg-slate-100 text-slate-600'}`}
                    >
                      {conn.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-500">
                    {conn.config?.region ?? '-'}
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-500">
                    {conn.lastValidated
                      ? new Date(conn.lastValidated).toLocaleString()
                      : 'Never'}
                  </td>
                  <td className="px-4 py-3 text-sm space-x-2">
                    <button
                      onClick={() => validateMutation.mutate(conn.id)}
                      disabled={validateMutation.isPending}
                      className="text-brand-600 hover:bg-brand-50 rounded px-2 py-1 text-sm transition-colors cursor-pointer"
                    >
                      Validate
                    </button>
                    <button
                      onClick={() => deleteMutation.mutate(conn.id)}
                      disabled={deleteMutation.isPending}
                      className="text-red-600 hover:bg-red-50 rounded px-2 py-1 text-sm transition-colors cursor-pointer"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-sm text-slate-400">
                  No connections yet. Add one above to get started.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
