'use client';

import React, { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import {
  getInterviewByShareToken,
  createCandidateSession,
  startInterviewSession,
  submitAnswer as submitAnswerAPI,
  getCandidateReport,
  InterviewDetails,
  Question as QuestionType,
  SessionProgress,
  CandidateReport,
} from '@/lib/api/candidate';
import InterviewLanding from './components/InterviewLanding';
import CandidateForm from './components/CandidateForm';
import Question from './components/Question';
import AnswerInput from './components/AnswerInput';
import Timer from './components/Timer';
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
} from 'lucide-react';

type Step = 'landing' | 'instructions' | 'register' | 'interview' | 'completed';

export default function CandidateInterviewPage() {
  const params = useParams();
  const shareToken = params?.shareToken as string;

  const [step, setStep] = useState<Step>('landing');
  const [interview, setInterview] = useState<InterviewDetails | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Session state
  const [sessionToken, setSessionToken] = useState<string | null>(null);
  const [creatingSession, setCreatingSession] = useState(false);
  const [registerError, setRegisterError] = useState<string | null>(null);

  // Question / Active Interview state
  const [currentQuestion, setCurrentQuestion] = useState<QuestionType | null>(null);
  const [progress, setProgress] = useState<SessionProgress | null>(null);
  const [submittingAnswer, setSubmittingAnswer] = useState(false);
  const [questionError, setQuestionError] = useState<string | null>(null);
  const [outroMessage, setOutroMessage] = useState<string | null>(null);

  // Completion Report state
  const [report, setReport] = useState<CandidateReport | null>(null);
  const [loadingReport, setLoadingReport] = useState(false);

  useEffect(() => {
    if (shareToken) {
      loadInterview();
    }
  }, [shareToken]);

  useEffect(() => {
    if (step === 'completed' && sessionToken) {
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

  const handleCandidateRegister = async (name: string, email: string) => {
    setCreatingSession(true);
    setRegisterError(null);
    try {
      const sessionRes = await createCandidateSession(shareToken, name, email);
      const token = sessionRes.session_token;
      setSessionToken(token);

      // Start the interview session and get first question
      const startRes = await startInterviewSession(token);
      setCurrentQuestion(startRes.question);
      setProgress(startRes.progress);
      setStep('interview');
    } catch (err: unknown) {
      if (err instanceof Error) {
        setRegisterError(err.message);
      } else {
        setRegisterError('Failed to create candidate session.');
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

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', background: 'var(--background)' }}>
      {/* Header */}
      <header className="brand-header">
        <div className="brand-logo">
          <Bot size={28} color="var(--primary)" />
          <span>NexHire AI</span>
        </div>

        {step === 'interview' && progress && (
          <Timer initialSeconds={progress.time_remaining_seconds} onTimeUp={handleTimeUp} />
        )}
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
              </div>

              <button onClick={handleProceedToForm} className="btn-primary" id="btn-proceed-to-form">
                I Understand & Accept <ArrowRight size={18} />
              </button>
            </div>
          </div>
        )}

        {step === 'register' && (
          <CandidateForm onSubmit={handleCandidateRegister} loading={creatingSession} error={registerError} />
        )}

        {step === 'interview' && (
          <div className="container" style={{ maxWidth: '720px' }}>
            <div className="glass-card">
              {questionError && (
                <div
                  style={{
                    background: '#fef2f2',
                    border: '1px solid #fecaca',
                    color: '#991b1b',
                    padding: '0.75rem 1rem',
                    borderRadius: '8px',
                    fontSize: '0.875rem',
                    marginBottom: '1.25rem',
                  }}
                >
                  {questionError}
                </div>
              )}

              {currentQuestion ? (
                <>
                  <Question question={currentQuestion} />
                  <AnswerInput onSubmit={handleSubmitAnswer} submitting={submittingAnswer} />
                </>
              ) : (
                <div className="loading-container" style={{ padding: '2rem' }}>
                  <div className="spinner"></div>
                  <p className="pulse-text">Generating Next Question...</p>
                </div>
              )}
            </div>
          </div>
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
