'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  fetchCurrentUser,
  verifyGoogleIdToken,
  setStoredToken,
  getGoogleLoginUrl,
  User,
} from '@/lib/api/auth';
import {
  Sparkles,
  Bot,
  Zap,
  ShieldCheck,
  UserCheck,
} from 'lucide-react';

export default function HomePage() {
  const router = useRouter();
  const [loadingUser, setLoadingUser] = useState(true);

  // ID Token Input (Dev / Alternative Auth)
  const [showIdTokenInput, setShowIdTokenInput] = useState(false);
  const [idTokenInput, setIdTokenInput] = useState('');
  const [verifyingIdToken, setVerifyingIdToken] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);

  useEffect(() => {
    initAuth();
  }, []);

  const initAuth = async () => {
    setLoadingUser(true);
    // Check URL parameters for token passed from backend redirect
    if (typeof window !== 'undefined') {
      const urlParams = new URLSearchParams(window.location.search);
      const tokenFromUrl = urlParams.get('token');
      if (tokenFromUrl) {
        setStoredToken(tokenFromUrl);
        window.history.replaceState({}, document.title, window.location.pathname);
      }
    }

    const currentUser = await fetchCurrentUser();
    setLoadingUser(false);

    if (currentUser) {
      router.push('/dashboard');
    }
  };

  const handleGoogleLogin = () => {
    window.location.href = getGoogleLoginUrl();
  };

  const handleVerifyIdToken = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!idTokenInput.trim()) return;

    setVerifyingIdToken(true);
    setAuthError(null);

    try {
      await verifyGoogleIdToken(idTokenInput.trim());
      router.push('/dashboard');
    } catch (err: unknown) {
      if (err instanceof Error) {
        setAuthError(err.message);
      } else {
        setAuthError('Token verification failed');
      }
    } finally {
      setVerifyingIdToken(false);
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', background: 'var(--background)' }}>
      {/* Header */}
      <header className="brand-header">
        <div className="brand-logo">
          <Bot size={28} color="var(--primary)" />
          <span>NexHire AI</span>
        </div>
      </header>

      {/* Main Content */}
      <main style={{ flex: 1, padding: '3rem 1.5rem', maxWidth: '1000px', margin: '0 auto', width: '100%' }}>
        {loadingUser ? (
          <div className="loading-container">
            <div className="spinner"></div>
            <p className="pulse-text">Loading NexHire Platform...</p>
          </div>
        ) : (
          <div style={{ textAlign: 'center', paddingTop: '1rem', paddingBottom: '3rem' }}>
            <div style={{ marginBottom: '1.25rem' }}>
              <span className="badge">AI-Powered Technical Interviews</span>
            </div>

            <h1
              style={{
                fontSize: '2.75rem',
                fontWeight: 800,
                letterSpacing: '-0.03em',
                lineHeight: 1.2,
                marginBottom: '1rem',
                color: 'var(--text-main)',
              }}
            >
              Automate Technical Screening <br /> with Generative AI
            </h1>

            <p
              style={{
                fontSize: '1.1rem',
                color: 'var(--text-muted)',
                maxWidth: '600px',
                margin: '0 auto 2.5rem',
                lineHeight: 1.6,
              }}
            >
              Recruiter platform to create adaptive AI interview sessions. Generate shareable candidate links and get instant AI evaluation reports.
            </p>

            {/* Auth Action Card */}
            <div
              className="glass-card"
              style={{ maxWidth: '440px', margin: '0 auto', padding: '2rem', textAlign: 'center' }}
            >
              <h2 style={{ fontSize: '1.2rem', fontWeight: 700, marginBottom: '1.25rem', color: 'var(--text-main)' }}>
                Recruiter Google Sign-In
              </h2>

              <button
                onClick={handleGoogleLogin}
                className="btn-primary"
                style={{
                  marginBottom: '1rem',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '0.75rem',
                }}
              >
                <svg width="18" height="18" viewBox="0 0 24 24">
                  <path
                    fill="#ffffff"
                    d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                  />
                  <path
                    fill="#ffffff"
                    d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                  />
                  <path
                    fill="#ffffff"
                    d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"
                  />
                  <path
                    fill="#ffffff"
                    d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"
                  />
                </svg>
                Sign in with Google
              </button>

              <div>
                <button
                  type="button"
                  onClick={() => setShowIdTokenInput(!showIdTokenInput)}
                  style={{
                    background: 'none',
                    border: 'none',
                    color: 'var(--text-muted)',
                    fontSize: '0.85rem',
                    cursor: 'pointer',
                    textDecoration: 'underline',
                  }}
                >
                  {showIdTokenInput ? 'Hide ID Token Input' : 'Development: Enter Google ID Token'}
                </button>
              </div>

              {showIdTokenInput && (
                <form onSubmit={handleVerifyIdToken} style={{ marginTop: '1.25rem', textAlign: 'left' }}>
                  <div className="form-group">
                    <label className="form-label" htmlFor="id-token">
                      Paste Google ID Token
                    </label>
                    <textarea
                      id="id-token"
                      className="form-input"
                      style={{ minHeight: '80px', fontSize: '0.85rem' }}
                      placeholder="eyJhbGciOiJSUzI1NiIs..."
                      value={idTokenInput}
                      onChange={(e) => setIdTokenInput(e.target.value)}
                    />
                  </div>
                  {authError && (
                    <div style={{ color: '#dc2626', fontSize: '0.85rem', marginBottom: '1rem' }}>{authError}</div>
                  )}
                  <button type="submit" className="btn-primary" disabled={verifyingIdToken}>
                    {verifyingIdToken ? 'Verifying...' : 'Verify Token & Go to Dashboard'}
                  </button>
                </form>
              )}
            </div>

            {/* Platform Feature Cards */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
                gap: '1.5rem',
                marginTop: '3.5rem',
                textAlign: 'left',
              }}
            >
              <div className="glass-card">
                <Zap size={24} color="var(--primary)" style={{ marginBottom: '0.75rem' }} />
                <h3 style={{ fontSize: '1.05rem', fontWeight: 600, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                  Dynamic Question AI
                </h3>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', lineHeight: 1.5 }}>
                  Generates technical questions tailored to your target job role and difficulty settings.
                </p>
              </div>

              <div className="glass-card">
                <ShieldCheck size={24} color="var(--accent)" style={{ marginBottom: '0.75rem' }} />
                <h3 style={{ fontSize: '1.05rem', fontWeight: 600, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                  Timed Shareable Links
                </h3>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', lineHeight: 1.5 }}>
                  Generate one-click candidate URLs with embedded session timers for structured evaluation.
                </p>
              </div>

              <div className="glass-card">
                <UserCheck size={24} color="#059669" style={{ marginBottom: '0.75rem' }} />
                <h3 style={{ fontSize: '1.05rem', fontWeight: 600, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                  Automated Candidate Reports
                </h3>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', lineHeight: 1.5 }}>
                  Inspect candidate answers, scoring breakdowns, and technical strengths in your dashboard.
                </p>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
