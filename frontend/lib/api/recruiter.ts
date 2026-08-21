import { getStoredToken } from './auth';

export type Difficulty = 'easy' | 'medium' | 'hard';
export type InterviewStatus = 'draft' | 'active' | 'closed';

export type Interview = {
  id: string;
  recruiter_id: string;
  title: string;
  role: string;
  description: string;
  difficulty: Difficulty;
  duration: number;
  voice_enabled: boolean;
  status: InterviewStatus;
  share_token: string;
  created_at: string;
};

export type CreateInterviewInput = {
  title: string;
  role: string;
  description?: string;
  difficulty: Difficulty;
  duration: number;
  voice_enabled?: boolean;
};

export type CandidateSessionInfo = {
  id: string;
  interview_id: string;
  name: string;
  email: string;
  status: string;
  started_at: string;
  expires_at: string;
  finished_at?: string;
};

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';

function getAuthHeaders(token?: string): Record<string, string> {
  const authToken = token || getStoredToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }
  return headers;
}

export async function createInterview(
  input: CreateInterviewInput,
  token?: string
): Promise<{ interview: Interview; share_link: string }> {
  const res = await fetch(`${BASE_URL}/api/interviews`, {
    method: 'POST',
    headers: getAuthHeaders(token),
    credentials: 'include',
    body: JSON.stringify(input),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to create interview (status ${res.status})`);
  }

  return await res.json();
}

export async function getMyInterviews(token?: string): Promise<Interview[]> {
  const res = await fetch(`${BASE_URL}/api/interviews`, {
    method: 'GET',
    headers: getAuthHeaders(token),
    credentials: 'include',
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to fetch interviews (status ${res.status})`);
  }

  const data = await res.json();
  return data.interviews || [];
}

export async function getInterviewDetails(id: string, token?: string): Promise<Interview> {
  const res = await fetch(`${BASE_URL}/api/interviews/${encodeURIComponent(id)}`, {
    method: 'GET',
    headers: getAuthHeaders(token),
    credentials: 'include',
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to fetch interview details (status ${res.status})`);
  }

  const data = await res.json();
  return data.interview;
}

export async function getCandidateSessionsForInterview(
  interviewId: string,
  token?: string
): Promise<CandidateSessionInfo[]> {
  const res = await fetch(`${BASE_URL}/api/candidates/interview/${encodeURIComponent(interviewId)}`, {
    method: 'GET',
    headers: getAuthHeaders(token),
    credentials: 'include',
  });

  if (!res.ok) {
    return [];
  }

  const data = await res.json();
  return data.sessions || [];
}

export async function getCandidateReportBySessionId(
  sessionId: string,
  token?: string
): Promise<any | null> {
  const res = await fetch(`${BASE_URL}/api/candidates/reports/${encodeURIComponent(sessionId)}`, {
    method: 'GET',
    headers: getAuthHeaders(token),
    credentials: 'include',
  });

  if (!res.ok) return null;
  const data = await res.json();
  return data.report || null;
}

export async function deleteInterview(id: string, token?: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/api/interviews/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    headers: getAuthHeaders(token),
    credentials: 'include',
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to delete interview (status ${res.status})`);
  }
}


