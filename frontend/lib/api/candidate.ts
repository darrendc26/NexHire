export type InterviewDetails = {
  id: string;
  title: string;
  role: string;
  difficulty: string;
  duration: number;
  description?: string;
  status: string;
};

export type SessionResponse = {
  session_id: string;
  session_token: string;
  status: string;
};

export type Question = {
  id: string;
  text: string;
  order: number;
};

export type SessionProgress = {
  questions_asked: number;
  time_remaining_seconds: number;
};

export type StartQuestionResponse = {
  question: Question;
  progress: SessionProgress;
};

export type SubmitAnswerResponse = {
  next_question?: Question;
  outro_message?: string;
  should_continue: boolean;
  progress: SessionProgress;
};

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';

export async function getInterviewByShareToken(shareToken: string): Promise<InterviewDetails> {
  const res = await fetch(`${BASE_URL}/api/interviews/share/${encodeURIComponent(shareToken)}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    cache: 'no-store',
  });

  if (!res.ok) {
    // Try fallback candidate endpoint
    const fallbackRes = await fetch(`${BASE_URL}/api/candidates/token/${encodeURIComponent(shareToken)}`, {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
      cache: 'no-store',
    });

    if (!fallbackRes.ok) {
      throw new Error(`Interview not found or unavailable (status ${res.status})`);
    }

    const fallbackData = await fallbackRes.json();
    const interview = fallbackData.interview || fallbackData;
    return {
      id: interview.id || '',
      title: interview.title || 'Technical Interview',
      role: interview.role || 'Software Engineer',
      difficulty: interview.difficulty || 'Medium',
      duration: interview.duration || 15,
      description: interview.description || '',
      status: interview.status || 'active',
    };
  }

  const data = await res.json();
  const interview = data.interview || data;

  return {
    id: interview.id || '',
    title: interview.title || data.title || 'Technical Interview',
    role: interview.role || data.role || 'Software Engineer',
    difficulty: interview.difficulty || data.difficulty || 'Medium',
    duration: interview.duration || data.duration || 15,
    description: interview.description || data.description || '',
    status: interview.status || data.status || 'active',
  };
}

export async function createCandidateSession(
  shareToken: string,
  name: string,
  email: string
): Promise<SessionResponse> {
  const res = await fetch(`${BASE_URL}/api/candidates/${encodeURIComponent(shareToken)}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ name, email }),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to create session (status ${res.status})`);
  }

  const data = await res.json();
  const session = data.session || data;

  return {
    session_id: session.session_id || data.session_id || session.id || '',
    session_token: session.session_token || data.session_token || session.raw_token || '',
    status: session.status || data.status || 'active',
  };
}

export async function startInterviewSession(sessionToken: string): Promise<StartQuestionResponse> {
  const res = await fetch(`${BASE_URL}/api/candidates/sessions/${encodeURIComponent(sessionToken)}/start`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${sessionToken}`,
    },
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to start interview (status ${res.status})`);
  }

  return await res.json();
}

export type SkillScore = {
  skill: string;
  score: number;
  feedback: string;
};

export type CandidateReport = {
  id: string;
  session_id: string;
  overall_score: number;
  recommendation: string;
  summary: string;
  strengths: string[];
  weaknesses: string[];
  skills: SkillScore[];
  created_at: string;
};

export async function submitAnswer(sessionToken: string, answer: string): Promise<SubmitAnswerResponse> {
  const res = await fetch(`${BASE_URL}/api/candidates/sessions/${encodeURIComponent(sessionToken)}/answer`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${sessionToken}`,
    },
    body: JSON.stringify({ answer }),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to submit answer (status ${res.status})`);
  }

  return await res.json();
}

export async function getCandidateReport(sessionToken: string): Promise<CandidateReport | null> {
  const res = await fetch(`${BASE_URL}/api/candidates/sessions/${encodeURIComponent(sessionToken)}/report`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${sessionToken}`,
    },
  });

  if (!res.ok) return null;
  const data = await res.json();
  return data.report || null;
}

