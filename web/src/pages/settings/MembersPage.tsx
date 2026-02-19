import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useState } from 'react';
import { inviteMember, listMembers, removeMember, updateMemberRole } from '../../api/members';

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

  if (isLoading) return <div>Loading members...</div>;

  return (
    <div className="page">
      <h2>Members</h2>

      <div className="invite-form">
        <h3>Invite member</h3>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            inviteMutation.mutate();
          }}
        >
          <input
            type="email"
            placeholder="Email address"
            value={inviteEmail}
            onChange={(e) => setInviteEmail(e.target.value)}
            required
          />
          <select value={inviteRole} onChange={(e) => setInviteRole(e.target.value)}>
            <option value="viewer">Viewer</option>
            <option value="member">Member</option>
            <option value="admin">Admin</option>
          </select>
          <button type="submit" disabled={inviteMutation.isPending}>
            {inviteMutation.isPending ? 'Inviting...' : 'Invite'}
          </button>
        </form>
        {inviteMutation.isError && (
          <div className="error">Failed to invite member</div>
        )}
      </div>

      <table className="members-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Email</th>
            <th>Role</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {members?.map((member) => (
            <tr key={member.id}>
              <td>{member.name}</td>
              <td>{member.email}</td>
              <td>
                <select
                  value={member.role}
                  onChange={(e) =>
                    updateRoleMutation.mutate({
                      memberId: member.id,
                      role: e.target.value,
                    })
                  }
                >
                  <option value="viewer">Viewer</option>
                  <option value="member">Member</option>
                  <option value="admin">Admin</option>
                  <option value="owner">Owner</option>
                </select>
              </td>
              <td>
                <button
                  onClick={() => removeMutation.mutate(member.id)}
                  disabled={removeMutation.isPending}
                  className="btn-danger"
                >
                  Remove
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
