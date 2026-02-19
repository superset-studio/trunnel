import client from './client';

export interface Organization {
  id: string;
  name: string;
  displayName: string;
  logoUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export async function listOrganizations(): Promise<Organization[]> {
  const res = await client.get<Organization[]>('/organizations');
  return res.data;
}

export async function getOrganization(orgId: string): Promise<Organization> {
  const res = await client.get<Organization>(`/organizations/${orgId}`);
  return res.data;
}

export async function createOrganization(displayName: string): Promise<Organization> {
  const res = await client.post<Organization>('/organizations', { displayName });
  return res.data;
}

export async function updateOrganization(
  orgId: string,
  data: { displayName: string; logoUrl?: string }
): Promise<Organization> {
  const res = await client.put<Organization>(`/organizations/${orgId}`, data);
  return res.data;
}
