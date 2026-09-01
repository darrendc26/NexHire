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

function mapEmailVerificationError(raw: string, fallback: string): string {
  const message = raw.toLowerCase();
  if (message.includes('invalid otp')) {
    return 'That code is incorrect. Please try again.';
  }
  if (message.includes('expired') || message.includes('otp has expired')) {
    return 'This code has expired. Request a new one.';
  }
  if (message.includes('email is not verified')) {
    return 'Please verify your email before starting the interview.';
  }
  if (message.includes('api key is invalid')) {
    return 'The Resend API key is invalid. Check RESEND_API_KEY in backend/.env.';
  }
  if (message.includes('domain is not verified')) {
    return 'EMAIL_FROM must use a domain verified in Resend (or beth.t@example.com for testing).';
  }
  if (message.includes('testing email address') || message.includes('only send testing emails')) {
    return 'Resend test sending only allows your Resend account email until you verify a domain.';
  }
  if (message.includes('failed to send verification email')) {
    return 'Could not send the verification email. Please try again in a moment.';
  }
  if (message.includes('failed on the \'email\' tag') || message.includes('failed on the "email" tag')) {
    return 'Please enter a valid email address.';
  }
  return raw || fallback;
}

export async function sendEmailOTP(email: string, interviewId: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/api/candidates/send-otp`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, interview_id: interviewId }),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(
      mapEmailVerificationError(errorData.error || '', `Failed to send verification code (status ${res.status})`)
    );
  }
}

export async function verifyEmailOTP(email: string, interviewId: string, otp: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/api/candidates/verify-otp`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, interview_id: interviewId, otp }),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(
      mapEmailVerificationError(errorData.error || '', `Failed to verify code (status ${res.status})`)
    );
  }
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
    throw new Error(
      mapEmailVerificationError(errorData.error || '', `Failed to create session (status ${res.status})`)
    );
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
  const url = `${BASE_URL}/api/candidates/sessions/${encodeURIComponent(sessionToken)}/start`;
  let lastError: Error | null = null;

  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      const res = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${sessionToken}`,
        },
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to start interview (status ${res.status})`);
      }

      return await res.json();
    } catch (err) {
      lastError = err instanceof Error ? err : new Error('Failed to start interview');
      const retryable =
        lastError.message.includes('socket hang up') ||
        lastError.message.includes('Failed to fetch') ||
        lastError.message.includes('ECONNRESET') ||
        lastError.message.includes('Backend is not reachable');
      if (!retryable || attempt === 2) {
        throw lastError;
      }
      await new Promise((resolve) => setTimeout(resolve, 600 * (attempt + 1)));
    }
  }

  throw lastError || new Error('Failed to start interview');
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

export async function fetchTTSAudio(text: string): Promise<Blob> {
  const res = await fetch(`${BASE_URL}/api/candidates/tts`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ text }),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to generate speech (status ${res.status})`);
  }

  return await res.blob();
}

export async function getSTTToken(sessionToken: string): Promise<string> {
  const res = await fetch(`${BASE_URL}/api/candidates/sessions/${encodeURIComponent(sessionToken)}/stt-token`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${sessionToken}`,
    },
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to fetch STT token (status ${res.status})`);
  }

  const data = await res.json();
  return data.token || '';
}

