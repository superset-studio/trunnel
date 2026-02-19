import client from './client';

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  user: {
    id: string;
    email: string;
    name: string;
    avatarUrl?: string;
    emailVerified: boolean;
  };
  organization: {
    id: string;
    name: string;
    displayName: string;
    logoUrl?: string;
  };
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
}

export async function register(data: {
  email: string;
  password: string;
  name: string;
  orgName: string;
}): Promise<AuthResponse> {
  const res = await client.post<AuthResponse>('/auth/register', data);
  return res.data;
}

export async function login(data: {
  email: string;
  password: string;
}): Promise<AuthResponse> {
  const res = await client.post<AuthResponse>('/auth/login', data);
  return res.data;
}

export async function refresh(refreshToken: string): Promise<TokenPair> {
  const res = await client.post<TokenPair>('/auth/refresh', { refreshToken });
  return res.data;
}
