import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useState } from 'react';
import { inviteMember, listMembers, removeMember, updateMemberRole } from '../../api/members';
import { Tooltip } from '../../components/Tooltip';

const roleDescriptions: Record<string, string> = {
  owner: 'Full access, can delete org',
  admin: 'Manage members and settings',
  member: 'Create and manage resources',
  viewer: 'Read-only access',
};

export function MembersPage() {
  const { orgId } = useParams({ strict: false }) as { orgId: string };
  const queryClient = useQueryClient();
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('member');

  const { data: members, isLoading } = useQuery({
    queryKey: ['members', orgId],
    queryFn: () => listMembers(orgId),
    enabled: !!orgId,
  });

  const inviteMutation = useMutation({
    mutationFn: () => inviteMember(orgId, { email: inviteEmail, role: inviteRole }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members', orgId] });
      setInviteEmail('');
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ memberId, role }: { memberId: string; role: string }) =>
      updateMemberRole(orgId, memberId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members', orgId] });
    },
  });

  const removeMutation = useMutation({
    mutationFn: (memberId: string) => removeMember(orgId, memberId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members', orgId] });
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
      <h2 className="text-2xl font-semibold text-slate-800 mb-6">Members</h2>

      {/* Invite form */}
      <div className="bg-white rounded-lg border border-stone-200 p-5 mb-6">
        <h3 className="text-sm font-semibold text-slate-800 mb-3">Invite member</h3>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            inviteMutation.mutate();
          }}
          className="flex items-center gap-3 flex-wrap"
        >
          <input
            type="email"
            placeholder="Email address"
            value={inviteEmail}
            onChange={(e) => setInviteEmail(e.target.value)}
            required
            className="flex-1 min-w-[200px] rounded-md border border-stone-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
          />
          <select
            value={inviteRole}
            onChange={(e) => setInviteRole(e.target.value)}
            className="rounded-md border border-stone-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
          >
            <option value="viewer">Viewer</option>
            <option value="member">Member</option>
            <option value="admin">Admin</option>
          </select>
          <button
            type="submit"
            disabled={inviteMutation.isPending}
            className="bg-brand-500 hover:bg-brand-600 text-white text-sm font-medium px-4 py-2 rounded-md transition-colors disabled:opacity-50 cursor-pointer"
          >
            {inviteMutation.isPending ? 'Inviting...' : 'Invite'}
          </button>
        </form>
        {inviteMutation.isError && (
          <p className="text-red-600 text-sm mt-2">Failed to invite member</p>
        )}
      </div>

      {/* Members table */}
      <div className="bg-white rounded-lg border border-stone-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="bg-stone-50">
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Name</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Email</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                <Tooltip content="Owner > Admin > Member > Viewer">
                  <span className="border-b border-dashed border-slate-400 cursor-help">Role</span>
                </Tooltip>
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody>
            {members?.map((member) => (
              <tr key={member.id} className="border-t border-stone-100">
                <td className="px-4 py-3 text-sm text-slate-800">{member.name}</td>
                <td className="px-4 py-3 text-sm text-slate-500">{member.email}</td>
                <td className="px-4 py-3 text-sm">
                  <Tooltip content={roleDescriptions[member.role] ?? ''}>
                    <select
                      value={member.role}
                      onChange={(e) =>
                        updateRoleMutation.mutate({
                          memberId: member.id,
                          role: e.target.value,
                        })
                      }
                      className="rounded-md border border-stone-300 text-sm px-2 py-1 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                    >
                      <option value="viewer">Viewer</option>
                      <option value="member">Member</option>
                      <option value="admin">Admin</option>
                      <option value="owner">Owner</option>
                    </select>
                  </Tooltip>
                </td>
                <td className="px-4 py-3 text-sm">
                  <button
                    onClick={() => removeMutation.mutate(member.id)}
                    disabled={removeMutation.isPending}
                    className="text-red-600 hover:bg-red-50 rounded px-2 py-1 text-sm transition-colors cursor-pointer"
                  >
                    Remove
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
