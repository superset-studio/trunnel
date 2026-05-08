import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useMemo, useState } from 'react';
import {
  createConnection,
  deleteConnection,
  getServerAWSIdentity,
  listConnections,
  validateConnection,
} from '../../api/connections';
import type { Connection, PermissionCheck } from '../../api/connections';
import { Select } from '../../components/Select';
import { InfoIcon, Tooltip } from '../../components/Tooltip';

type AuthMethod = 'access-keys' | 'iam-role';

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  valid: 'bg-green-100 text-green-800',
  invalid: 'bg-red-100 text-red-800',
  expired: 'bg-slate-100 text-slate-600',
  partial: 'bg-orange-100 text-orange-800',
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

const AWS_PERMISSIONS_POLICY = JSON.stringify(
  {
    Version: '2012-10-17',
    Statement: [
      {
        Effect: 'Allow',
        Action: ['ec2:*', 'eks:*', 'iam:*', 's3:*', 'rds:*'],
        Resource: '*',
      },
    ],
  },
  null,
  2
);

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      className="absolute top-2 right-2 bg-slate-700 hover:bg-slate-600 text-slate-200 text-xs px-2 py-1 rounded transition-colors cursor-pointer"
    >
      {copied ? 'Copied!' : 'Copy'}
    </button>
  );
}

function CodeBlock({ code }: { code: string }) {
  return (
    <div className="relative">
      <CopyButton text={code} />
      <pre className="bg-slate-800 text-green-400 p-3 rounded text-xs font-mono overflow-x-auto">
        {code}
      </pre>
    </div>
  );
}

function PermissionBadges({ permissions }: { permissions?: PermissionCheck[] }) {
  if (!permissions || permissions.length === 0) return null;
  return (
    <div className="flex flex-wrap gap-1 mt-1">
      {permissions.map((p) => (
        <Tooltip
          key={p.service}
          content={p.passed ? `${p.service.toUpperCase()} — OK` : `${p.service.toUpperCase()} — ${p.error ?? 'Permission denied'}`}
        >
          <span
            className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium uppercase ${
              p.passed
                ? 'bg-green-100 text-green-700'
                : 'bg-red-100 text-red-700'
            }`}
          >
            {p.service}
          </span>
        </Tooltip>
      ))}
    </div>
  );
}

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
  const [showSetupHelp, setShowSetupHelp] = useState(false);

  const { data: connections, isLoading } = useQuery({
    queryKey: ['connections', orgId],
    queryFn: () => listConnections(orgId),
    enabled: !!orgId,
  });

  const { data: serverIdentity } = useQuery({
    queryKey: ['aws-identity'],
    queryFn: getServerAWSIdentity,
    enabled: authMethod === 'iam-role' && showSetupHelp,
    staleTime: 5 * 60 * 1000,
  });

  const trustPolicy = useMemo(() => {
    if (!serverIdentity?.accountId) return null;
    return JSON.stringify(
      {
        Version: '2012-10-17',
        Statement: [
          {
            Effect: 'Allow',
            Principal: {
              AWS: `arn:aws:iam::${serverIdentity.accountId}:root`,
            },
            Action: 'sts:AssumeRole',
            Condition: {
              StringEquals: {
                'sts:ExternalId':
                  externalId || '<enter external ID above>',
              },
            },
          },
        ],
      },
      null,
      2
    );
  }, [serverIdentity?.accountId, externalId]);

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
            <button
              type="button"
              onClick={() => setShowSetupHelp(!showSetupHelp)}
              className="text-sm text-brand-600 hover:text-brand-700 underline underline-offset-2 cursor-pointer"
            >
              {showSetupHelp ? 'Hide setup instructions' : 'Show setup instructions'}
            </button>
          </div>

          {/* Setup instructions */}
          {showSetupHelp && (
            <div className="bg-stone-50 border border-stone-200 rounded-lg p-4 space-y-3">
              {authMethod === 'access-keys' ? (
                <>
                  <p className="text-sm text-slate-700 font-medium">How to set up Access Keys</p>
                  <ol className="text-sm text-slate-600 space-y-3 list-decimal list-inside">
                    <li>Create an IAM user with programmatic access in your AWS account.</li>
                    <li>
                      Attach the following permissions policy to the user:
                      <div className="mt-2">
                        <CodeBlock code={AWS_PERMISSIONS_POLICY} />
                      </div>
                    </li>
                    <li>Copy the Access Key ID and Secret Access Key and enter them below.</li>
                  </ol>
                </>
              ) : (
                <>
                  <p className="text-sm text-slate-700 font-medium">How to set up an IAM Role</p>
                  <ol className="text-sm text-slate-600 space-y-3 list-decimal list-inside">
                    <li>Create an IAM role in your AWS account.</li>
                    <li>
                      Set this trust policy on the role:
                      <div className="mt-2">
                        {trustPolicy ? (
                          <CodeBlock code={trustPolicy} />
                        ) : (
                          <p className="text-xs text-slate-400 italic">
                            Loading Kapstan server identity...
                          </p>
                        )}
                      </div>
                    </li>
                    <li>
                      Attach this permissions policy to the role:
                      <div className="mt-2">
                        <CodeBlock code={AWS_PERMISSIONS_POLICY} />
                      </div>
                    </li>
                    <li>Copy the Role ARN and enter it above.</li>
                  </ol>
                </>
              )}
            </div>
          )}

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
                    <PermissionBadges permissions={conn.config?.permissions} />
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
