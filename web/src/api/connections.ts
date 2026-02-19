import client from './client';

export interface Connection {
  id: string;
  tenantId: string;
  name: string;
  category: string;
  status: string;
  lastValidated?: string;
  config?: Record<string, string>;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateConnectionData {
  name: string;
  category: string;
  credentials: Record<string, string>;
  config?: Record<string, string>;
}

export interface UpdateConnectionData {
  name?: string;
  credentials?: Record<string, string>;
  config?: Record<string, string>;
}

export async function createConnection(
  orgId: string,
  data: CreateConnectionData
): Promise<Connection> {
  const res = await client.post<Connection>(
    `/organizations/${orgId}/connections`,
    data
  );
  return res.data;
}

export async function listConnections(orgId: string): Promise<Connection[]> {
  const res = await client.get<Connection[]>(
    `/organizations/${orgId}/connections`
  );
  return res.data;
}

export async function getConnection(
  orgId: string,
  connId: string
): Promise<Connection> {
  const res = await client.get<Connection>(
    `/organizations/${orgId}/connections/${connId}`
  );
  return res.data;
}

export async function updateConnection(
  orgId: string,
  connId: string,
  data: UpdateConnectionData
): Promise<Connection> {
  const res = await client.put<Connection>(
    `/organizations/${orgId}/connections/${connId}`,
    data
  );
  return res.data;
}

export async function deleteConnection(
  orgId: string,
  connId: string
): Promise<void> {
  await client.delete(`/organizations/${orgId}/connections/${connId}`);
}

export async function validateConnection(
  orgId: string,
  connId: string
): Promise<Connection> {
  const res = await client.post<Connection>(
    `/organizations/${orgId}/connections/${connId}/validate`
  );
  return res.data;
}
