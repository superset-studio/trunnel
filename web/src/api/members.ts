import client from './client';

export interface Member {
  id: string;
  organizationId: string;
  userId: string;
  role: string;
  invitedBy?: string;
  invitedAt?: string;
  acceptedAt?: string;
  createdAt: string;
  email: string;
  name: string;
  avatarUrl?: string;
}

export async function listMembers(orgId: string): Promise<Member[]> {
  const res = await client.get<Member[]>(`/organizations/${orgId}/members`);
  return res.data;
}

export async function inviteMember(
  orgId: string,
  data: { email: string; role: string }
): Promise<Member> {
  const res = await client.post<Member>(
    `/organizations/${orgId}/members/invite`,
    data
  );
  return res.data;
}

export async function updateMemberRole(
  orgId: string,
  memberId: string,
  role: string
): Promise<void> {
  await client.put(`/organizations/${orgId}/members/${memberId}`, { role });
}

export async function removeMember(
  orgId: string,
  memberId: string
): Promise<void> {
  await client.delete(`/organizations/${orgId}/members/${memberId}`);
}
