'use client';

import React, { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import {
  getInterviewByShareToken,
  createCandidateSession,
  startInterviewSession,
  submitAnswer as submitAnswerAPI,
  getCandidateReport,
  getSTTToken,
  sendEmailOTP,
  verifyEmailOTP,
  InterviewDetails,
  Question as QuestionType,
  SessionProgress,
  CandidateReport,
} from '@/lib/api/candidate';
import InterviewLanding from './components/InterviewLanding';
import CandidateForm from './components/CandidateForm';
import OTPVerify from './components/OTPVerify';
import DeviceTest from './components/DeviceTest';
import VoiceAgentRoom from './components/VoiceAgentRoom';
import { unlockPlayback } from '@/lib/audioSession';
import {
  Bot,
  AlertTriangle,
  CheckCircle2,
  FileText,
  ShieldCheck,
  Clock,
  Sparkles,
  ArrowRight,
  RefreshCw,
  Award,
  ThumbsUp,
  ThumbsDown,
  BarChart,
  ShieldAlert,
  Maximize2,
  Lock,
} from 'lucide-react';

type Step = 'landing' | 'instructions' | 'register' | 'verify-otp' | 'device-test' | 'interview' | 'completed';

export default function CandidateInterviewPage() {
  const params = useParams();
  const shareToken = params?.shareToken as string;

  const [step, setStep] = useState<Step>('landing');
  const [interview, setInterview] = useState<InterviewDetails | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Session state
  const [candidateName, setCandidateName] = useState('');
  const [candidateEmail, setCandidateEmail] = useState('');
  const [sessionToken, setSessionToken] = useState<string | null>(null);
  const [creatingSession, setCreatingSession] = useState(false);
  const [sendingOtp, setSendingOtp] = useState(false);
  const [verifyingOtp, setVerifyingOtp] = useState(false);
  const [registerError, setRegisterError] = useState<string | null>(null);
  const [otpError, setOtpError] = useState<string | null>(null);

  // Question / Active Interview state
  const [currentQuestion, setCurrentQuestion] = useState<QuestionType | null>(null);
  const [progress, setProgress] = useState<SessionProgress | null>(null);
  const [submittingAnswer, setSubmittingAnswer] = useState(false);
  const [questionError, setQuestionError] = useState<string | null>(null);
  const [outroMessage, setOutroMessage] = useState<string | null>(null);

  // Completion Report state
  const [report, setReport] = useState<CandidateReport | null>(null);
  const [loadingReport, setLoadingReport] = useState(false);

  // STT Token state
  const [deepgramToken, setDeepgramToken] = useState<string>('');

  // Fullscreen & Anti-Cheating Proctoring state
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [tabSwitchCount, setTabSwitchCount] = useState(0);
  const [showFullscreenModal, setShowFullscreenModal] = useState(false);

  const enterFullscreen = async () => {
    try {
      if (document.documentElement.requestFullscreen) {
        await document.documentElement.requestFullscreen();
      }
      setIsFullscreen(true);
      setShowFullscreenModal(false);
    } catch (err) {
      console.warn('Fullscreen request prevented by browser policy:', err);
      setIsFullscreen(true);
      setShowFullscreenModal(false);
    }
  };

  useEffect(() => {
    if (step !== 'interview') return;

    const handleFullscreenChange = () => {
      const isFull = !!document.fullscreenElement;
      setIsFullscreen(isFull);
      if (!isFull) {
        setTabSwitchCount((prev) => prev + 1);
        setShowFullscreenModal(true);
      }
    };

    const handleVisibilityChange = () => {
      if (document.hidden) {
        setTabSwitchCount((prev) => prev + 1);
        setShowFullscreenModal(true);
      }
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [step]);

  useEffect(() => {
    if (shareToken) {
      loadInterview();
    }
  }, [shareToken]);

  useEffect(() => {
    if (sessionToken) {
      getSTTToken(sessionToken)
        .then((token) => {
          if (token) setDeepgramToken(token);
        })
        .catch((err) => console.error('Failed to load STT token:', err));
    }
  }, [sessionToken]);

  useEffect(() => {
    if (step === 'completed' && sessionToken) {
      if (document.fullscreenElement) {
        document.exitFullscreen().catch(() => {});
      }
      loadReport(sessionToken);
    }
  }, [step, sessionToken]);

  const loadInterview = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getInterviewByShareToken(shareToken);
      setInterview(data);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Failed to load interview details.');
      }
    } finally {
      setLoading(false);
    }
  };

  const loadReport = async (token: string) => {
    setLoadingReport(true);
    try {
      const data = await getCandidateReport(token);
      setReport(data);
    } catch (err) {
      console.error('Failed to load candidate report:', err);
    } finally {
      setLoadingReport(false);
    }
  };

  const handleStartRegister = () => {
    setStep('instructions');
  };

  const handleProceedToForm = () => {
    setStep('register');
  };

  const handleCandidateRegister = async (name: string, email: string): Promise<void> => {
    if (!interview?.id) {
      throw new Error('Interview is not loaded. Please refresh and try again.');
    }

    setSendingOtp(true);
    setRegisterError(null);
    try {
      const normalizedEmail = email.trim().toLowerCase();
      await sendEmailOTP(normalizedEmail, interview.id);
      setCandidateName(name.trim());
      setCandidateEmail(normalizedEmail);
      setOtpError(null);
      setStep('verify-otp');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to send verification code.';
      setRegisterError(message);
      throw err instanceof Error ? err : new Error(message);
    } finally {
      setSendingOtp(false);
    }
  };

  const handleVerifyOtp = async (otp: string): Promise<void> => {
    if (!interview?.id || !candidateEmail) {
      throw new Error('Interview is not loaded. Please refresh and try again.');
    }

    setVerifyingOtp(true);
    setOtpError(null);
    try {
      await verifyEmailOTP(candidateEmail, interview.id, otp);
      setStep('device-test');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Invalid or expired verification code.';
      setOtpError(message);
      throw err instanceof Error ? err : new Error(message);
    } finally {
      setVerifyingOtp(false);
    }
  };

  const handleResendOtp = async (): Promise<void> => {
    if (!interview?.id || !candidateEmail) {
      throw new Error('Interview is not loaded. Please refresh and try again.');
    }
    setOtpError(null);
    await sendEmailOTP(candidateEmail, interview.id);
  };

  const handleChangeEmail = () => {
    setOtpError(null);
    setRegisterError(null);
    setStep('register');
  };

  const handleStartInterviewSession = async () => {
    if (!candidateName || !candidateEmail) return;
    setCreatingSession(true);
    setRegisterError(null);
    try {
      await unlockPlayback();
      const sessionRes = await createCandidateSession(shareToken, candidateName, candidateEmail);
      const token = sessionRes.session_token;
      setSessionToken(token);

      // Start the interview session and get first question
      const startRes = await startInterviewSession(token);
      setCurrentQuestion(startRes.question);
      setProgress(startRes.progress);

      // Fetch STT token before launching interview step
      try {
        const stt = await getSTTToken(token);
        if (stt) setDeepgramToken(stt);
      } catch (sttErr) {
        console.error('Failed to pre-fetch STT token:', sttErr);
      }

      setStep('interview');
      enterFullscreen();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to create candidate session.';
      setRegisterError(message);
      if (message.toLowerCase().includes('verify your email') || message.toLowerCase().includes('not verified')) {
        setStep('verify-otp');
        setOtpError(message);
      }
    } finally {
      setCreatingSession(false);
    }
  };

  const handleSubmitAnswer = async (answerText: string) => {
    if (!sessionToken) return;

    setSubmittingAnswer(true);
    setQuestionError(null);

    try {
      const res = await submitAnswerAPI(sessionToken, answerText);
      setProgress(res.progress);

      if (res.next_question && res.should_continue) {
        setCurrentQuestion(res.next_question);
      } else {
        // Interview Completed
        if (res.outro_message) {
          setOutroMessage(res.outro_message);
        }
        setStep('completed');
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setQuestionError(err.message);
      } else {
        setQuestionError('Error submitting answer. Please try again.');
      }
    } finally {
      setSubmittingAnswer(false);
    }
  };

  const handleTimeUp = () => {
    setStep('completed');
    setOutroMessage('Session time limit reached. Your answers have been submitted automatically.');
  };

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', justifyContent: 'center', background: 'var(--background)' }}>
        <div className="loading-container">
          <div className="spinner"></div>
          <p className="pulse-text">Loading Interview Details...</p>
        </div>
      </div>
    );
  }

  if (error || !interview) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', justifyContent: 'center', background: 'var(--background)' }}>
        <div className="container" style={{ maxWidth: '520px' }}>
          <div className="glass-card" style={{ textAlign: 'center' }}>
            <AlertTriangle size={48} color="#dc2626" style={{ marginBottom: '1rem' }} />
            <h2 style={{ fontSize: '1.4rem', fontWeight: 700, marginBottom: '0.5rem', color: 'var(--text-main)' }}>
              Interview Not Found
            </h2>
            <p style={{ color: 'var(--text-muted)', fontSize: '0.95rem', marginBottom: '1.5rem' }}>
              {error || 'This interview link is invalid, expired, or unavailable.'}
            </p>
            <button onClick={loadInterview} className="btn-primary" style={{ width: 'auto', margin: '0 auto' }}>
              <RefreshCw size={16} /> Try Again
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (step === 'interview') {
    const totalQuestions = Math.max(3, Math.min(8, Math.round((interview.duration || 15) / 3)));

    return (
      <>
        {(!isFullscreen || showFullscreenModal) && (
          <div
            style={{
              position: 'fixed',
              inset: 0,
              zIndex: 9999,
              background: 'rgba(15, 23, 42, 0.92)',
              backdropFilter: 'blur(12px)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '1.5rem',
            }}
          >
            <div
              className="glass-card"
              style={{
                maxWidth: '480px',
                width: '100%',
                textAlign: 'center',
                background: '#ffffff',
                border: '2px solid #ef4444',
                borderRadius: '16px',
                padding: '2rem',
                boxShadow: '0 25px 50px -12px rgba(239, 68, 68, 0.25)',
              }}
            >
              <div
                style={{
                  width: '64px',
                  height: '64px',
                  borderRadius: '50%',
                  background: '#fef2f2',
                  border: '2px solid #fecaca',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  margin: '0 auto 1.25rem',
                }}
              >
                <ShieldAlert size={34} color="#dc2626" />
              </div>

              <h2 style={{ fontSize: '1.4rem', fontWeight: 800, color: '#1e293b', marginBottom: '0.5rem' }}>
                Proctored Interview Alert
              </h2>

              <p style={{ color: '#475569', fontSize: '0.9rem', lineHeight: 1.6, marginBottom: '1.25rem' }}>
                To ensure test integrity and prevent external LLM usage or secondary tab switching, this interview must be conducted in Fullscreen Mode.
              </p>

              {tabSwitchCount > 0 && (
                <div
                  style={{
                    background: '#fff7ed',
                    border: '1px solid #fed7aa',
                    color: '#c2410c',
                    padding: '0.6rem 0.85rem',
                    borderRadius: '8px',
                    fontSize: '0.825rem',
                    fontWeight: 600,
                    marginBottom: '1.5rem',
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '0.4rem',
                  }}
                >
                  <Lock size={15} /> Tab / Window Switches Recorded: {tabSwitchCount}
                </div>
              )}

              <button
                type="button"
                onClick={enterFullscreen}
                className="btn-primary"
                style={{ width: '100%', padding: '0.85rem', fontSize: '0.95rem' }}
              >
                <Maximize2 size={18} /> Re-enter Fullscreen Mode
              </button>
            </div>
          </div>
        )}

        {currentQuestion ? (
          <VoiceAgentRoom
            question={currentQuestion}
            deepgramToken={deepgramToken}
            sessionToken={sessionToken || ''}
            onSubmitAnswer={handleSubmitAnswer}
            submitting={submittingAnswer}
            questionNumber={currentQuestion.order}
            totalQuestions={totalQuestions}
            candidateName={candidateName}
            remainingSeconds={progress?.time_remaining_seconds ?? interview.duration * 60}
            onTimeUp={handleTimeUp}
            onEndInterview={() => {
              setOutroMessage('You ended the interview. Your answers have been submitted.');
              setStep('completed');
            }}
            questionError={questionError}
          />
        ) : (
          <div className="loading-container" style={{ minHeight: '100vh' }}>
            <div className="spinner"></div>
            <p className="pulse-text">Generating Next Question...</p>
          </div>
        )}
      </>
    );
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', background: 'var(--background)' }}>
      {/* Header */}
      <header className="brand-header">
        <div className="brand-logo">
          <Bot size={28} color="var(--primary)" />
          <span>NexHire AI</span>
        </div>

      </header>

      {/* Main Flow Content */}
      <main style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '2rem 1.5rem' }}>
        {step === 'landing' && (
          <InterviewLanding interview={interview} onStartClick={handleStartRegister} />
        )}

        {step === 'instructions' && (
          <div className="container" style={{ maxWidth: '600px' }}>
            <div className="glass-card">
              <div style={{ textAlign: 'center', marginBottom: '1.75rem' }}>
                <span className="badge">Candidate Instructions</span>
                <h2 style={{ fontSize: '1.5rem', fontWeight: 700, marginTop: '0.75rem', color: 'var(--text-main)' }}>
                  Interview Guidelines
                </h2>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem', marginBottom: '2rem' }}>
                <div style={{ display: 'flex', gap: '1rem', alignItems: 'flex-start' }}>
                  <div style={{ background: '#eef2ff', padding: '0.6rem', borderRadius: '8px' }}>
                    <Clock size={20} color="var(--primary)" />
                  </div>
                  <div>
                    <h4 style={{ fontWeight: 600, fontSize: '0.95rem', marginBottom: '0.2rem', color: 'var(--text-main)' }}>
                      Time Limit ({interview.duration} Minutes)
                    </h4>
                    <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.5 }}>
                      The interview is timed. Monitor the countdown timer at the top right of your screen.
                    </p>
                  </div>
                </div>

                <div style={{ display: 'flex', gap: '1rem', alignItems: 'flex-start' }}>
                  <div style={{ background: '#f3e8ff', padding: '0.6rem', borderRadius: '8px' }}>
                    <FileText size={20} color="var(--accent)" />
                  </div>
                  <div>
                    <h4 style={{ fontWeight: 600, fontSize: '0.95rem', marginBottom: '0.2rem', color: 'var(--text-main)' }}>
                      Detailed Answers Expected
                    </h4>
                    <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.5 }}>
                      Provide structured, comprehensive explanations detailing your technical approach and concepts.
                    </p>
                  </div>
                </div>

                <div style={{ display: 'flex', gap: '1rem', alignItems: 'flex-start' }}>
                  <div style={{ background: '#ecfdf5', padding: '0.6rem', borderRadius: '8px' }}>
                    <ShieldCheck size={20} color="#059669" />
                  </div>
                  <div>
                    <h4 style={{ fontWeight: 600, fontSize: '0.95rem', marginBottom: '0.2rem', color: 'var(--text-main)' }}>
                      Automated Real-Time AI Evaluation
                    </h4>
                    <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.5 }}>
                      Your answers will be evaluated to provide technical competency reports to the recruiter.
                    </p>
                  </div>
                </div>

                <div style={{ display: 'flex', gap: '1rem', alignItems: 'flex-start' }}>
                  <div style={{ background: '#fff7ed', padding: '0.6rem', borderRadius: '8px' }}>
                    <Lock size={20} color="#ea580c" />
                  </div>
                  <div>
                    <h4 style={{ fontWeight: 600, fontSize: '0.95rem', marginBottom: '0.2rem', color: 'var(--text-main)' }}>
                      Proctored Locked Fullscreen Mode
                    </h4>
                    <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.5 }}>
                      The session requires Fullscreen Mode to prevent tab switching or external AI tool usage during testing.
                    </p>
                  </div>
                </div>
              </div>

              <button onClick={handleProceedToForm} className="btn-primary" id="btn-proceed-to-form">
                I Understand & Accept <ArrowRight size={18} />
              </button>
            </div>
          </div>
        )}

        {step === 'register' && (
          <CandidateForm
            onSubmit={handleCandidateRegister}
            loading={sendingOtp}
            error={registerError}
            initialName={candidateName}
            initialEmail={candidateEmail}
          />
        )}

        {step === 'verify-otp' && (
          <OTPVerify
            email={candidateEmail}
            onVerify={handleVerifyOtp}
            onResend={handleResendOtp}
            onChangeEmail={handleChangeEmail}
            verifying={verifyingOtp}
            error={otpError}
          />
        )}

        {step === 'device-test' && (
          <DeviceTest
            candidateName={candidateName}
            onProceed={handleStartInterviewSession}
            loading={creatingSession}
            error={registerError}
          />
        )}

        {step === 'completed' && (
          <div className="container" style={{ maxWidth: '680px' }}>
            <div className="glass-card">
              <div style={{ textAlign: 'center', marginBottom: '1.5rem' }}>
                <div
                  style={{
                    width: '56px',
                    height: '56px',
                    borderRadius: '50%',
                    background: '#ecfdf5',
                    border: '1px solid #a7f3d0',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    margin: '0 auto 1rem',
                  }}
                >
                  <CheckCircle2 size={30} color="#059669" />
                </div>

                <h2 style={{ fontSize: '1.6rem', fontWeight: 700, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                  Interview Session Completed!
                </h2>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', lineHeight: 1.5 }}>
                  {outroMessage ||
                    'Your responses have been evaluated and recorded for the hiring team.'}
                </p>
              </div>

              {/* AI Report Card */}
              {loadingReport ? (
                <div className="loading-container" style={{ padding: '1.5rem' }}>
                  <div className="spinner"></div>
                  <p className="pulse-text">Generating AI Evaluation Report Card...</p>
                </div>
              ) : report ? (
                <div
                  style={{
                    background: '#f8fafc',
                    border: '1px solid var(--border)',
                    borderRadius: '10px',
                    padding: '1.5rem',
                    marginBottom: '1.5rem',
                    textAlign: 'left',
                  }}
                >
                  {/* Score & Recommendation Banner */}
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      paddingBottom: '1rem',
                      marginBottom: '1rem',
                      borderBottom: '1px solid var(--border)',
                      flexWrap: 'wrap',
                      gap: '0.75rem',
                    }}
                  >
                    <div>
                      <div style={{ fontSize: '0.75rem', textTransform: 'uppercase', color: 'var(--text-muted)', fontWeight: 600 }}>
                        Overall Score
                      </div>
                      <div style={{ fontSize: '1.75rem', fontWeight: 800, color: 'var(--primary)' }}>
                        {report.overall_score.toFixed(1)} <span style={{ fontSize: '1rem', color: 'var(--text-muted)' }}>/ 100</span>
                      </div>
                    </div>

                    <div>
                      <div style={{ fontSize: '0.75rem', textTransform: 'uppercase', color: 'var(--text-muted)', fontWeight: 600, marginBottom: '0.2rem' }}>
                        Recommendation
                      </div>
                      <span
                        className="badge"
                        style={{
                          textTransform: 'uppercase',
                          fontWeight: 700,
                          padding: '0.35rem 0.85rem',
                          background:
                            report.recommendation === 'strong_hire' || report.recommendation === 'hire'
                              ? '#ecfdf5'
                              : report.recommendation === 'maybe'
                              ? '#fefce8'
                              : '#fef2f2',
                          color:
                            report.recommendation === 'strong_hire' || report.recommendation === 'hire'
                              ? '#047857'
                              : report.recommendation === 'maybe'
                              ? '#a16207'
                              : '#991b1b',
                          borderColor:
                            report.recommendation === 'strong_hire' || report.recommendation === 'hire'
                              ? '#a7f3d0'
                              : report.recommendation === 'maybe'
                              ? '#fef08a'
                              : '#fecaca',
                        }}
                      >
                        {report.recommendation.replace('_', ' ')}
                      </span>
                    </div>
                  </div>

                  {/* Summary */}
                  {report.summary && (
                    <div style={{ marginBottom: '1.25rem' }}>
                      <div style={{ fontWeight: 600, fontSize: '0.9rem', marginBottom: '0.3rem', color: 'var(--text-main)', display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                        <Award size={16} color="var(--primary)" /> Executive Summary
                      </div>
                      <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', lineHeight: 1.6 }}>
                        {report.summary}
                      </p>
                    </div>
                  )}

                  {/* Strengths & Weaknesses */}
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '1rem', marginBottom: '1.25rem' }}>
                    {report.strengths && report.strengths.length > 0 && (
                      <div style={{ background: '#ffffff', border: '1px solid var(--border)', padding: '0.875rem', borderRadius: '8px' }}>
                        <div style={{ fontWeight: 600, fontSize: '0.85rem', color: '#047857', marginBottom: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.3rem' }}>
                          <ThumbsUp size={14} /> Strengths
                        </div>
                        <ul style={{ paddingLeft: '1.2rem', fontSize: '0.8rem', color: 'var(--text-muted)', lineHeight: 1.5 }}>
                          {report.strengths.map((s, idx) => (
                            <li key={idx}>{s}</li>
                          ))}
                        </ul>
                      </div>
                    )}

                    {report.weaknesses && report.weaknesses.length > 0 && (
                      <div style={{ background: '#ffffff', border: '1px solid var(--border)', padding: '0.875rem', borderRadius: '8px' }}>
                        <div style={{ fontWeight: 600, fontSize: '0.85rem', color: '#b91c1c', marginBottom: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.3rem' }}>
                          <ThumbsDown size={14} /> Areas for Growth
                        </div>
                        <ul style={{ paddingLeft: '1.2rem', fontSize: '0.8rem', color: 'var(--text-muted)', lineHeight: 1.5 }}>
                          {report.weaknesses.map((w, idx) => (
                            <li key={idx}>{w}</li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>

                  {/* Skill Scores */}
                  {report.skills && report.skills.length > 0 && (
                    <div>
                      <div style={{ fontWeight: 600, fontSize: '0.9 river', marginBottom: '0.5rem', color: 'var(--text-main)', display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                        <BarChart size={16} color="var(--primary)" /> Technical Competency Breakdown
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                        {report.skills.map((sk, idx) => (
                          <div key={idx} style={{ background: '#ffffff', border: '1px solid var(--border)', padding: '0.6rem 0.85rem', borderRadius: '6px', fontSize: '0.8rem' }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontWeight: 600, marginBottom: '0.2rem' }}>
                              <span>{sk.skill}</span>
                              <span style={{ color: 'var(--primary)' }}>{sk.score.toFixed(0)}%</span>
                            </div>
                            {sk.feedback && <p style={{ color: 'var(--text-muted)', fontSize: '0.775rem' }}>{sk.feedback}</p>}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : null}

              <div style={{ textAlign: 'center' }}>
                <span className="badge" style={{ background: '#f1f5f9', color: 'var(--text-muted)', border: '1px solid #cbd5e1' }}>
                  Session ID: {sessionToken ? sessionToken.substring(0, 12) + '...' : 'Completed'}
                </span>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
