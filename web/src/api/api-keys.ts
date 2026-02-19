import client from './client';

export interface APIKey {
  id: string;
  tenantId: string;
  name: string;
  keyPrefix: string;
  accessLevel: string;
  createdBy?: string;
  createdAt: string;
  lastUsedAt?: string;
}

export interface CreateAPIKeyResponse {
  apiKey: APIKey;
  key: string;
}

export async function createAPIKey(
  orgId: string,
  data: { name: string; accessLevel: string }
): Promise<CreateAPIKeyResponse> {
  const res = await client.post<CreateAPIKeyResponse>(
    `/organizations/${orgId}/api-keys`,
    data
  );
  return res.data;
}

export async function listAPIKeys(orgId: string): Promise<APIKey[]> {
  const res = await client.get<APIKey[]>(`/organizations/${orgId}/api-keys`);
  return res.data;
}

export async function revokeAPIKey(
  orgId: string,
  keyId: string
): Promise<void> {
  await client.delete(`/organizations/${orgId}/api-keys/${keyId}`);
}
