export type User = {
  id: string;
  google_id?: string;
  email: string;
  name: string;
  picture?: string;
  created_at?: string;
};

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';

export function getStoredToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('nexhire_token');
}

export function setStoredToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem('nexhire_token', token);
}

export function removeStoredToken(): void {
  if (typeof window === 'undefined') return;
  localStorage.removeItem('nexhire_token');
}

export async function fetchCurrentUser(token?: string): Promise<User | null> {
  const authToken = token || getStoredToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }

  try {
    const res = await fetch(`${BASE_URL}/api/auth/me`, {
      method: 'GET',
      headers,
      credentials: 'include',
    });

    if (!res.ok) return null;
    return await res.json();
  } catch (err) {
    console.error('Error fetching current user:', err);
    return null;
  }
}

export async function verifyGoogleIdToken(idToken: string): Promise<{ token: string; user: User }> {
  const res = await fetch(`${BASE_URL}/api/auth/google/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ id_token: idToken }),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Verification failed (status ${res.status})`);
  }

  const data = await res.json();
  if (data.token) {
    setStoredToken(data.token);
  }
  return data;
}

export async function logoutUser(): Promise<void> {
  try {
    await fetch(`${BASE_URL}/api/auth/logout`, {
      method: 'POST',
      credentials: 'include',
    });
  } catch (err) {
    console.error('Logout request failed:', err);
  } finally {
    removeStoredToken();
  }
}

export function getGoogleLoginUrl(): string {
  const apiBase = BASE_URL || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080');
  return `${apiBase}/api/auth/google/login`;
}
