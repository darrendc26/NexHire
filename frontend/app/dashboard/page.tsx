'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  fetchCurrentUser,
  logoutUser,
  setStoredToken,
  User,
} from '@/lib/api/auth';

import {
  createInterview,
  getMyInterviews,
  getCandidateSessionsForInterview,
  getCandidateReportBySessionId,
  deleteInterview,
  Interview,
  Difficulty,
  CandidateSessionInfo,
  addCandidateEmail,
} from '@/lib/api/recruiter';

import { CandidateReport } from '@/lib/api/candidate';

import {
  Sparkles,
  Plus,
  Copy,
  Check,
  LogOut,
  Briefcase,
  Clock,
  BarChart2,
  Users,
  ExternalLink,
  Bot,
  X,
  FileText,
  Award,
  ThumbsUp,
  ThumbsDown,
  BarChart,
  Trash2,
  UserPlus,
} from 'lucide-react';

export default function DashboardPage() {
  const router = useRouter();

  const [user, setUser] = useState<User | null>(null);
  const [loadingUser, setLoadingUser] = useState(true);

  const [interviews, setInterviews] = useState<Interview[]>([]);
  const [loadingInterviews, setLoadingInterviews] = useState(false);

  // ============================================================
  // Create Interview Modal
  // ============================================================

  const [showCreateModal, setShowCreateModal] = useState(false);

  const [title, setTitle] = useState('');
  const [role, setRole] = useState('');
  const [description, setDescription] = useState('');
  const [difficulty, setDifficulty] =
    useState<Difficulty>('medium');
  const [duration, setDuration] = useState(15);
  const [voiceEnabled, setVoiceEnabled] = useState(false);

  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  // ============================================================
  // Candidate / Interview Selection
  // ============================================================

  const [selectedInterview, setSelectedInterview] =
    useState<Interview | null>(null);

  // ============================================================
  // View Candidates Modal
  // ============================================================

  const [showCandidatesModal, setShowCandidatesModal] =
    useState(false);

  const [candidateSessions, setCandidateSessions] =
    useState<CandidateSessionInfo[]>([]);

  const [loadingCandidates, setLoadingCandidates] =
    useState(false);

  // ============================================================
  // Add Candidate Modal
  // ============================================================

  const [showAddCandidateModal, setShowAddCandidateModal] =
    useState(false);

  const [candidateEmail, setCandidateEmail] = useState('');
  const [addingCandidate, setAddingCandidate] = useState(false);
  const [candidateError, setCandidateError] =
    useState<string | null>(null);

  // ============================================================
  // Candidate Report Viewer
  // ============================================================

  const [selectedReportSession, setSelectedReportSession] =
    useState<CandidateSessionInfo | null>(null);

  const [selectedReport, setSelectedReport] =
    useState<CandidateReport | null>(null);

  const [loadingReport, setLoadingReport] = useState(false);

  // ============================================================
  // Toast / Copy
  // ============================================================

  const [copiedToken, setCopiedToken] =
    useState<string | null>(null);

  const [toastMessage, setToastMessage] =
    useState<string | null>(null);

  // ============================================================
  // Auth
  // ============================================================

  useEffect(() => {
    initAuth();
  }, []);

  const initAuth = async () => {
    setLoadingUser(true);

    if (typeof window !== 'undefined') {
      const urlParams = new URLSearchParams(
        window.location.search
      );

      const tokenFromUrl = urlParams.get('token');

      if (tokenFromUrl) {
        setStoredToken(tokenFromUrl);

        window.history.replaceState(
          {},
          document.title,
          window.location.pathname
        );
      }
    }

    const currentUser = await fetchCurrentUser();

    if (!currentUser) {
      router.push('/');
      return;
    }

    setUser(currentUser);
    setLoadingUser(false);

    loadInterviews();
  };

  // ============================================================
  // Toast
  // ============================================================

  const showToast = (msg: string) => {
    setToastMessage(msg);

    setTimeout(() => {
      setToastMessage(null);
    }, 3000);
  };

  // ============================================================
  // Interviews
  // ============================================================

  const loadInterviews = async () => {
    setLoadingInterviews(true);

    try {
      const list = await getMyInterviews();
      setInterviews(list);
    } catch (err) {
      console.error('Failed to load interviews:', err);
    } finally {
      setLoadingInterviews(false);
    }
  };

  // ============================================================
  // Logout
  // ============================================================

  const handleLogout = async () => {
    await logoutUser();

    setUser(null);

    router.push('/');
  };

  // ============================================================
  // Create Interview
  // ============================================================

  const handleCreateInterview = async (
    e: React.FormEvent
  ) => {
    e.preventDefault();

    if (!title.trim() || !role.trim()) {
      setCreateError(
        'Please fill in Title and Role'
      );

      return;
    }

    setCreating(true);
    setCreateError(null);

    try {
      const res = await createInterview({
        title: title.trim(),
        role: role.trim(),
        description: description.trim(),
        difficulty,
        duration: Number(duration),
        voice_enabled: voiceEnabled,
      });

      setInterviews((prev) => [
        res.interview,
        ...prev,
      ]);

      setShowCreateModal(false);

      // Reset form
      setTitle('');
      setRole('');
      setDescription('');
      setDifficulty('medium');
      setDuration(15);
      setVoiceEnabled(false);

      showToast(
        'Interview created successfully!'
      );
    } catch (err: unknown) {
      if (err instanceof Error) {
        setCreateError(err.message);
      } else {
        setCreateError(
          'Failed to create interview'
        );
      }
    } finally {
      setCreating(false);
    }
  };

  // ============================================================
  // Add Candidate
  // ============================================================

  const handleAddCandidate = (
    interview: Interview
  ) => {
    setSelectedInterview(interview);

    setCandidateEmail('');
    setCandidateError(null);

    setShowAddCandidateModal(true);
  };

  const handleAddCandidateSubmit = async (
    e: React.FormEvent
  ) => {
    e.preventDefault();

    if (
      !selectedInterview ||
      !candidateEmail.trim()
    ) {
      return;
    }

    setAddingCandidate(true);
    setCandidateError(null);

    try {
      await addCandidateEmail(
        selectedInterview.id,
        candidateEmail.trim().toLowerCase()
      );

      setCandidateEmail('');

      setShowAddCandidateModal(false);

      showToast(
        'Candidate added successfully!'
      );
    } catch (err: unknown) {
      if (err instanceof Error) {
        setCandidateError(err.message);
      } else {
        setCandidateError(
          'Failed to add candidate'
        );
      }
    } finally {
      setAddingCandidate(false);
    }
  };

  const closeAddCandidateModal = () => {
    setShowAddCandidateModal(false);
    setCandidateEmail('');
    setCandidateError(null);
  };

  // ============================================================
  // View Candidates
  // ============================================================

  const handleViewCandidates = async (
    interview: Interview
  ) => {
    setSelectedInterview(interview);

    setShowCandidatesModal(true);

    setLoadingCandidates(true);

    try {
      const sessions =
        await getCandidateSessionsForInterview(
          interview.id
        );

      setCandidateSessions(sessions);
    } catch (err) {
      console.error(
        'Failed to load candidate sessions:',
        err
      );

      setCandidateSessions([]);
    } finally {
      setLoadingCandidates(false);
    }
  };

  const closeCandidatesModal = () => {
    setShowCandidatesModal(false);
    setSelectedInterview(null);
    setCandidateSessions([]);
  };

  // ============================================================
  // Candidate Report
  // ============================================================

  const handleViewReport = async (
    session: CandidateSessionInfo
  ) => {
    setSelectedReportSession(session);
    setLoadingReport(true);
    setSelectedReport(null);

    try {
      const reportData =
        await getCandidateReportBySessionId(
          session.id
        );

      setSelectedReport(reportData);
    } catch (err) {
      console.error(
        'Failed to fetch candidate report:',
        err
      );
    } finally {
      setLoadingReport(false);
    }
  };

  // ============================================================
  // Delete Interview
  // ============================================================

  const handleDeleteInterview = async (
    interview: Interview
  ) => {
    if (
      !window.confirm(
        `Are you sure you want to delete "${interview.title}"?`
      )
    ) {
      return;
    }

    try {
      await deleteInterview(interview.id);

      setInterviews((prev) =>
        prev.filter(
          (i) => i.id !== interview.id
        )
      );

      showToast(
        'Interview deleted successfully'
      );
    } catch (err: unknown) {
      if (err instanceof Error) {
        showToast(
          `Failed to delete interview: ${err.message}`
        );
      } else {
        showToast(
          'Failed to delete interview'
        );
      }
    }
  };

  // ============================================================
  // Copy Candidate Link
  // ============================================================

  const copyShareLink = (
    shareToken: string
  ) => {
    const origin =
      typeof window !== 'undefined'
        ? window.location.origin
        : 'http://localhost:3000';

    const candidateUrl =
      `${origin}/interview/${shareToken}`;

    navigator.clipboard.writeText(
      candidateUrl
    );

    setCopiedToken(shareToken);

    showToast(
      'Candidate share link copied to clipboard!'
    );

    setTimeout(
      () => setCopiedToken(null),
      2500
    );
  };

  // ============================================================
  // Loading
  // ============================================================

  if (loadingUser) {
    return (
      <div
        style={{
          minHeight: '100vh',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
        }}
      >
        <div className="loading-container">
          <div className="spinner"></div>

          <p className="pulse-text">
            Loading Recruiter Dashboard...
          </p>
        </div>
      </div>
    );
  }

  // ============================================================
  // Dashboard
  // ============================================================

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--background)',
      }}
    >
      {/* ======================================================
          Toast
      ======================================================= */}

      {toastMessage && (
        <div
          style={{
            position: 'fixed',
            top: '20px',
            right: '20px',
            background: '#0f172a',
            color: '#fff',
            padding: '0.75rem 1.25rem',
            borderRadius: '8px',
            boxShadow:
              '0 4px 12px rgba(0,0,0,0.15)',
            zIndex: 1200,
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem',
            fontSize: '0.9rem',
            fontWeight: 500,
          }}
        >
          <Sparkles
            size={16}
            color="#818cf8"
          />

          {toastMessage}
        </div>
      )}

      {/* ======================================================
          Navigation
      ======================================================= */}

      <header className="brand-header">
        <div
          className="brand-logo"
          style={{ cursor: 'pointer' }}
          onClick={() =>
            router.push('/dashboard')
          }
        >
          <Bot
            size={28}
            color="var(--primary)"
          />

          <span>NexHire Dashboard</span>
        </div>

        {user && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '1.25rem',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '0.75rem',
              }}
            >
              {user.picture ? (
                <img
                  src={user.picture}
                  alt={user.name}
                  style={{
                    width: '36px',
                    height: '36px',
                    borderRadius: '50%',
                    border:
                      '1px solid var(--border)',
                  }}
                />
              ) : (
                <div
                  style={{
                    width: '36px',
                    height: '36px',
                    borderRadius: '50%',
                    background: 'var(--primary)',
                    color: '#fff',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontWeight: 700,
                  }}
                >
                  {user.name
                    ? user.name[0].toUpperCase()
                    : 'R'}
                </div>
              )}

              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                }}
              >
                <span
                  style={{
                    fontSize: '0.9rem',
                    fontWeight: 600,
                    color: 'var(--text-main)',
                  }}
                >
                  {user.name}
                </span>

                <span
                  style={{
                    fontSize: '0.75rem',
                    color: 'var(--text-muted)',
                  }}
                >
                  {user.email}
                </span>
              </div>
            </div>

            <button
              onClick={handleLogout}
              style={{
                background: '#ffffff',
                border:
                  '1px solid var(--border)',
                color: 'var(--text-muted)',
                padding:
                  '0.5rem 0.875rem',
                borderRadius: '6px',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '0.4rem',
                fontSize: '0.85rem',
                fontWeight: 500,
              }}
            >
              <LogOut size={16} />
              Logout
            </button>
          </div>
        )}
      </header>

      {/* ======================================================
          Main
      ======================================================= */}

      <main
        style={{
          flex: 1,
          padding: '2.5rem 1.5rem',
          maxWidth: '1100px',
          margin: '0 auto',
          width: '100%',
        }}
      >
        {/* Header */}

        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '2rem',
            flexWrap: 'wrap',
            gap: '1rem',
          }}
        >
          <div>
            <h1
              style={{
                fontSize: '1.75rem',
                fontWeight: 700,
                letterSpacing: '-0.02em',
                color: 'var(--text-main)',
              }}
            >
              Interview Manager
            </h1>

            <p
              style={{
                color: 'var(--text-muted)',
                fontSize: '0.9rem',
                marginTop: '0.2rem',
              }}
            >
              Create AI technical interviews
              and manage shareable candidate
              links.
            </p>
          </div>

          <button
            onClick={() =>
              setShowCreateModal(true)
            }
            className="btn-primary"
            style={{ width: 'auto' }}
            id="btn-new-interview"
          >
            <Plus size={18} />
            Create New Interview
          </button>
        </div>

        {/* ====================================================
            Interviews
        ===================================================== */}

        {loadingInterviews ? (
          <div className="loading-container">
            <div className="spinner"></div>

            <p className="pulse-text">
              Loading your active interviews...
            </p>
          </div>
        ) : interviews.length === 0 ? (
          <div
            className="glass-card"
            style={{
              textAlign: 'center',
              padding: '4rem 2rem',
            }}
          >
            <Briefcase
              size={44}
              color="var(--text-dim)"
              style={{
                marginBottom: '1rem',
              }}
            />

            <h3
              style={{
                fontSize: '1.25rem',
                fontWeight: 600,
                marginBottom: '0.5rem',
                color: 'var(--text-main)',
              }}
            >
              No Interviews Created Yet
            </h3>

            <p
              style={{
                color: 'var(--text-muted)',
                fontSize: '0.95rem',
                marginBottom: '1.5rem',
              }}
            >
              Get started by creating your
              first automated AI technical
              interview session.
            </p>

            <button
              onClick={() =>
                setShowCreateModal(true)
              }
              className="btn-primary"
              style={{
                width: 'auto',
                margin: '0 auto',
              }}
            >
              <Plus size={18} />
              Create Interview
            </button>
          </div>
        ) : (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns:
                'repeat(auto-fill, minmax(320px, 1fr))',
              gap: '1.5rem',
            }}
          >
            {interviews.map((interview) => {
              const candidateUrl =
                typeof window !== 'undefined'
                  ? `${window.location.origin}/interview/${interview.share_token}`
                  : `/interview/${interview.share_token}`;

              return (
                <div
                  key={interview.id}
                  className="glass-card"
                  style={{
                    padding: '1.5rem',
                    display: 'flex',
                    flexDirection: 'column',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent:
                        'space-between',
                      alignItems: 'flex-start',
                      marginBottom: '0.75rem',
                    }}
                  >
                    <span
                      className="badge"
                      style={{
                        textTransform:
                          'capitalize',
                      }}
                    >
                      {interview.difficulty}
                    </span>

                    <span
                      style={{
                        fontSize: '0.8rem',
                        color:
                          'var(--text-muted)',
                      }}
                    >
                      {new Date(
                        interview.created_at ||
                        Date.now()
                      ).toLocaleDateString()}
                    </span>
                  </div>

                  <h3
                    style={{
                      fontSize: '1.2rem',
                      fontWeight: 700,
                      marginBottom: '0.35rem',
                      color:
                        'var(--text-main)',
                    }}
                  >
                    {interview.title}
                  </h3>

                  <p
                    style={{
                      color: 'var(--primary)',
                      fontWeight: 600,
                      fontSize: '0.9rem',
                      marginBottom: '1rem',
                    }}
                  >
                    {interview.role}
                  </p>

                  {interview.description && (
                    <p
                      style={{
                        color:
                          'var(--text-muted)',
                        fontSize: '0.85rem',
                        lineHeight: 1.5,
                        marginBottom:
                          '1.25rem',
                        display:
                          '-webkit-box',
                        WebkitLineClamp: 2,
                        WebkitBoxOrient:
                          'vertical',
                        overflow: 'hidden',
                      }}
                    >
                      {interview.description}
                    </p>
                  )}

                  <div
                    style={{
                      marginTop: 'auto',
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        gap: '1rem',
                        fontSize: '0.8rem',
                        color:
                          'var(--text-muted)',
                        marginBottom:
                          '1.25rem',
                        paddingTop:
                          '0.75rem',
                        borderTop:
                          '1px solid var(--border)',
                      }}
                    >
                      <span
                        style={{
                          display: 'flex',
                          alignItems:
                            'center',
                          gap: '0.3rem',
                        }}
                      >
                        <Clock size={14} />
                        {interview.duration}{' '}
                        mins
                      </span>

                      <span
                        style={{
                          display: 'flex',
                          alignItems:
                            'center',
                          gap: '0.3rem',
                        }}
                      >
                        <BarChart2
                          size={14}
                        />
                        Status:{' '}
                        {interview.status}
                      </span>
                    </div>

                    {/* Card Actions */}

                    <div
                      style={{
                        display: 'flex',
                        gap: '0.5rem',
                      }}
                    >
                      {/* Copy Link */}

                      <button
                        onClick={() =>
                          copyShareLink(
                            interview.share_token
                          )
                        }
                        style={{
                          flex: 1,
                          background:
                            copiedToken ===
                              interview.share_token
                              ? '#ecfdf5'
                              : '#eef2ff',
                          border:
                            copiedToken ===
                              interview.share_token
                              ? '1px solid #a7f3d0'
                              : '1px solid #c7d2fe',
                          color:
                            copiedToken ===
                              interview.share_token
                              ? '#047857'
                              : '#3730a3',
                          padding: '0.6rem',
                          borderRadius: '6px',
                          cursor: 'pointer',
                          display: 'flex',
                          alignItems:
                            'center',
                          justifyContent:
                            'center',
                          gap: '0.4rem',
                          fontSize: '0.85rem',
                          fontWeight: 600,
                        }}
                      >
                        {copiedToken ===
                          interview.share_token ? (
                          <>
                            <Check size={14} />
                            Copied Link
                          </>
                        ) : (
                          <>
                            <Copy size={14} />
                            Copy Candidate Link
                          </>
                        )}
                      </button>

                      {/* Add Candidate */}

                      <button
                        onClick={() =>
                          handleAddCandidate(
                            interview
                          )
                        }
                        style={{
                          background:
                            'var(--primary)',
                          border:
                            '1px solid var(--primary)',
                          color: '#ffffff',
                          padding:
                            '0.6rem 0.8rem',
                          borderRadius: '6px',
                          cursor: 'pointer',
                          display: 'flex',
                          alignItems:
                            'center',
                          justifyContent:
                            'center',
                          gap: '0.3rem',
                          fontSize: '0.85rem',
                        }}
                        title="Add Candidate"
                      >
                        <UserPlus
                          size={14}
                        />
                      </button>

                      {/* View Candidates */}

                      <button
                        onClick={() =>
                          handleViewCandidates(
                            interview
                          )
                        }
                        style={{
                          background:
                            '#ffffff',
                          border:
                            '1px solid var(--border)',
                          color:
                            'var(--text-main)',
                          padding:
                            '0.6rem 0.8rem',
                          borderRadius: '6px',
                          cursor: 'pointer',
                          display: 'flex',
                          alignItems:
                            'center',
                          justifyContent:
                            'center',
                          gap: '0.3rem',
                          fontSize: '0.85rem',
                        }}
                        title="View Candidates"
                      >
                        <Users size={14} />
                      </button>

                      {/* Open Candidate View */}

                      <a
                        href={candidateUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{
                          background:
                            '#ffffff',
                          border:
                            '1px solid var(--border)',
                          color:
                            'var(--text-main)',
                          padding:
                            '0.6rem 0.8rem',
                          borderRadius: '6px',
                          display: 'flex',
                          alignItems:
                            'center',
                          justifyContent:
                            'center',
                        }}
                        title="Open Candidate View"
                      >
                        <ExternalLink
                          size={14}
                        />
                      </a>

                      {/* Delete */}

                      <button
                        onClick={() =>
                          handleDeleteInterview(
                            interview
                          )
                        }
                        style={{
                          background:
                            '#fef2f2',
                          border:
                            '1px solid #fecaca',
                          color: '#dc2626',
                          padding:
                            '0.6rem 0.8rem',
                          borderRadius: '6px',
                          cursor: 'pointer',
                          display: 'flex',
                          alignItems:
                            'center',
                          justifyContent:
                            'center',
                        }}
                        title="Delete Interview"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </main>

      {/* ======================================================
          Create Interview Modal
      ======================================================= */}

      {showCreateModal && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background:
              'rgba(15, 23, 42, 0.4)',
            zIndex: 900,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '1.5rem',
          }}
        >
          <div
            className="glass-card"
            style={{
              maxWidth: '540px',
              width: '100%',
              padding: '2rem',
            }}
          >
            <div
              style={{
                display: 'flex',
                justifyContent:
                  'space-between',
                alignItems: 'center',
                marginBottom: '1.5rem',
              }}
            >
              <h2
                style={{
                  fontSize: '1.35rem',
                  fontWeight: 700,
                  color:
                    'var(--text-main)',
                }}
              >
                Create AI Technical Interview
              </h2>

              <button
                type="button"
                onClick={() =>
                  setShowCreateModal(false)
                }
                style={{
                  background: 'none',
                  border: 'none',
                  color:
                    'var(--text-muted)',
                  cursor: 'pointer',
                }}
              >
                <X size={20} />
              </button>
            </div>

            {createError && (
              <div
                style={{
                  background: '#fef2f2',
                  border:
                    '1px solid #fecaca',
                  color: '#991b1b',
                  padding:
                    '0.75rem 1rem',
                  borderRadius: '6px',
                  fontSize: '0.85rem',
                  marginBottom: '1rem',
                }}
              >
                {createError}
              </div>
            )}

            <form
              onSubmit={
                handleCreateInterview
              }
            >
              <div className="form-group">
                <label
                  className="form-label"
                  htmlFor="interview-title"
                >
                  Interview Title *
                </label>

                <input
                  id="interview-title"
                  type="text"
                  className="form-input"
                  placeholder="e.g. Senior Frontend Screening"
                  value={title}
                  onChange={(e) =>
                    setTitle(
                      e.target.value
                    )
                  }
                  required
                />
              </div>

              <div className="form-group">
                <label
                  className="form-label"
                  htmlFor="interview-role"
                >
                  Target Role / Job Title *
                </label>

                <input
                  id="interview-role"
                  type="text"
                  className="form-input"
                  placeholder="e.g. React Software Engineer"
                  value={role}
                  onChange={(e) =>
                    setRole(
                      e.target.value
                    )
                  }
                  required
                />
              </div>

              <div className="form-group">
                <label
                  className="form-label"
                  htmlFor="interview-desc"
                >
                  Job Description &
                  Guidelines (Optional)
                </label>

                <textarea
                  id="interview-desc"
                  className="form-input"
                  style={{
                    minHeight: '90px',
                  }}
                  placeholder="Paste context for the AI interviewer to generate appropriate questions..."
                  value={description}
                  onChange={(e) =>
                    setDescription(
                      e.target.value
                    )
                  }
                />
              </div>

              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns:
                    '1fr 1fr',
                  gap: '1rem',
                }}
              >
                <div className="form-group">
                  <label className="form-label">
                    Difficulty Level
                  </label>

                  <select
                    className="form-input"
                    value={difficulty}
                    onChange={(e) =>
                      setDifficulty(
                        e.target.value as Difficulty
                      )
                    }
                  >
                    <option value="easy">
                      Easy
                    </option>

                    <option value="medium">
                      Medium
                    </option>

                    <option value="hard">
                      Hard
                    </option>
                  </select>
                </div>

                <div className="form-group">
                  <label
                    className="form-label"
                    htmlFor="interview-duration"
                  >
                    Duration (Minutes)
                  </label>

                  <input
                    id="interview-duration"
                    type="number"
                    min="5"
                    max="120"
                    className="form-input"
                    value={duration}
                    onChange={(e) =>
                      setDuration(
                        Number(
                          e.target.value
                        )
                      )
                    }
                    required
                  />
                </div>
              </div>

              <div
                style={{
                  display: 'flex',
                  justifyContent:
                    'flex-end',
                  gap: '1rem',
                  marginTop: '1.5rem',
                }}
              >
                <button
                  type="button"
                  onClick={() =>
                    setShowCreateModal(
                      false
                    )
                  }
                  style={{
                    background:
                      '#ffffff',
                    border:
                      '1px solid var(--border)',
                    color:
                      'var(--text-muted)',
                    padding:
                      '0.75rem 1.25rem',
                    borderRadius: '8px',
                    cursor: 'pointer',
                    fontWeight: 500,
                  }}
                >
                  Cancel
                </button>

                <button
                  type="submit"
                  className="btn-primary"
                  disabled={creating}
                  style={{
                    width: 'auto',
                  }}
                >
                  {creating
                    ? 'Creating...'
                    : 'Create & Generate Link'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ======================================================
          Add Candidate Modal
      ======================================================= */}

      {showAddCandidateModal &&
        selectedInterview && (
          <div
            style={{
              position: 'fixed',
              inset: 0,
              background:
                'rgba(15, 23, 42, 0.4)',
              zIndex: 1000,
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
                padding: '2rem',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  justifyContent:
                    'space-between',
                  alignItems: 'center',
                  marginBottom: '1.5rem',
                }}
              >
                <div>
                  <h2
                    style={{
                      fontSize: '1.3rem',
                      fontWeight: 700,
                      color:
                        'var(--text-main)',
                    }}
                  >
                    Add Candidate
                  </h2>

                  <p
                    style={{
                      color:
                        'var(--text-muted)',
                      fontSize: '0.85rem',
                      marginTop:
                        '0.25rem',
                    }}
                  >
                    {selectedInterview.title}
                  </p>
                </div>

                <button
                  type="button"
                  onClick={
                    closeAddCandidateModal
                  }
                  style={{
                    background: 'none',
                    border: 'none',
                    color:
                      'var(--text-muted)',
                    cursor: 'pointer',
                  }}
                >
                  <X size={20} />
                </button>
              </div>

              {candidateError && (
                <div
                  style={{
                    background:
                      '#fef2f2',
                    border:
                      '1px solid #fecaca',
                    color: '#991b1b',
                    padding:
                      '0.75rem 1rem',
                    borderRadius: '6px',
                    fontSize: '0.85rem',
                    marginBottom:
                      '1rem',
                  }}
                >
                  {candidateError}
                </div>
              )}

              <form
                onSubmit={
                  handleAddCandidateSubmit
                }
              >
                <div className="form-group">
                  <label
                    className="form-label"
                    htmlFor="candidate-email"
                  >
                    Candidate Email
                  </label>

                  <input
                    id="candidate-email"
                    type="email"
                    className="form-input"
                    placeholder="candidate@example.com"
                    value={
                      candidateEmail
                    }
                    onChange={(e) =>
                      setCandidateEmail(
                        e.target.value
                      )
                    }
                    disabled={
                      addingCandidate
                    }
                    required
                    autoFocus
                  />
                </div>

                <div
                  style={{
                    display: 'flex',
                    justifyContent:
                      'flex-end',
                    gap: '0.75rem',
                    marginTop: '1.5rem',
                  }}
                >
                  <button
                    type="button"
                    onClick={
                      closeAddCandidateModal
                    }
                    disabled={
                      addingCandidate
                    }
                    style={{
                      background:
                        '#ffffff',
                      border:
                        '1px solid var(--border)',
                      color:
                        'var(--text-muted)',
                      padding:
                        '0.7rem 1rem',
                      borderRadius: '7px',
                      cursor:
                        'pointer',
                    }}
                  >
                    Cancel
                  </button>

                  <button
                    type="submit"
                    className="btn-primary"
                    disabled={
                      addingCandidate ||
                      !candidateEmail.trim()
                    }
                    style={{
                      width: 'auto',
                    }}
                  >
                    {addingCandidate
                      ? 'Adding...'
                      : 'Add Candidate'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

      {/* ======================================================
          View Candidates Modal
      ======================================================= */}

      {showCandidatesModal &&
        selectedInterview && (
          <div
            style={{
              position: 'fixed',
              inset: 0,
              background:
                'rgba(15, 23, 42, 0.4)',
              zIndex: 900,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '1.5rem',
            }}
          >
            <div
              className="glass-card"
              style={{
                maxWidth: '640px',
                width: '100%',
                padding: '2rem',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  justifyContent:
                    'space-between',
                  alignItems: 'center',
                  marginBottom: '1.5rem',
                }}
              >
                <div>
                  <h2
                    style={{
                      fontSize: '1.3rem',
                      fontWeight: 700,
                      color:
                        'var(--text-main)',
                    }}
                  >
                    Candidate Submissions
                  </h2>

                  <p
                    style={{
                      color:
                        'var(--text-muted)',
                      fontSize: '0.85rem',
                    }}
                  >
                    {selectedInterview.title}
                  </p>
                </div>

                <button
                  type="button"
                  onClick={
                    closeCandidatesModal
                  }
                  style={{
                    background: 'none',
                    border: 'none',
                    color:
                      'var(--text-muted)',
                    cursor: 'pointer',
                  }}
                >
                  <X size={20} />
                </button>
              </div>

              {loadingCandidates ? (
                <div
                  className="loading-container"
                  style={{
                    padding: '2rem',
                  }}
                >
                  <div className="spinner"></div>

                  <p className="pulse-text">
                    Loading candidate
                    records...
                  </p>
                </div>
              ) : candidateSessions.length ===
                0 ? (
                <p
                  style={{
                    color:
                      'var(--text-muted)',
                    textAlign: 'center',
                    padding:
                      '2rem 0',
                  }}
                >
                  No candidates have
                  started this interview
                  session yet. Share the
                  link to receive
                  submissions.
                </p>
              ) : (
                <div
                  style={{
                    display: 'flex',
                    flexDirection:
                      'column',
                    gap: '0.75rem',
                    maxHeight: '400px',
                    overflowY:
                      'auto',
                  }}
                >
                  {candidateSessions.map(
                    (session) => (
                      <div
                        key={session.id}
                        style={{
                          background:
                            '#f8fafc',
                          border:
                            '1px solid var(--border)',
                          padding: '1rem',
                          borderRadius:
                            '8px',
                          display: 'flex',
                          justifyContent:
                            'space-between',
                          alignItems:
                            'center',
                        }}
                      >
                        <div>
                          <div
                            style={{
                              fontWeight: 600,
                              fontSize:
                                '0.95rem',
                              color:
                                'var(--text-main)',
                            }}
                          >
                            {session.name}
                          </div>

                          <div
                            style={{
                              fontSize:
                                '0.8rem',
                              color:
                                'var(--text-muted)',
                            }}
                          >
                            {session.email}
                          </div>
                        </div>

                        <div
                          style={{
                            textAlign:
                              'right',
                            display:
                              'flex',
                            alignItems:
                              'center',
                            gap: '0.75rem',
                          }}
                        >
                          <div>
                            <span
                              className="badge"
                              style={{
                                textTransform:
                                  'capitalize',
                              }}
                            >
                              {
                                session.status
                              }
                            </span>

                            <div
                              style={{
                                fontSize:
                                  '0.75rem',
                                color:
                                  'var(--text-muted)',
                                marginTop:
                                  '0.25rem',
                              }}
                            >
                              {new Date(
                                session.started_at
                              ).toLocaleString()}
                            </div>
                          </div>

                          <button
                            onClick={() =>
                              handleViewReport(
                                session
                              )
                            }
                            style={{
                              background:
                                '#eef2ff',
                              border:
                                '1px solid #c7d2fe',
                              color:
                                '#3730a3',
                              padding:
                                '0.4rem 0.75rem',
                              borderRadius:
                                '6px',
                              cursor:
                                'pointer',
                              fontSize:
                                '0.8rem',
                              fontWeight: 600,
                              display:
                                'flex',
                              alignItems:
                                'center',
                              gap: '0.3rem',
                            }}
                          >
                            <FileText
                              size={14}
                            />
                            View Report
                          </button>
                        </div>
                      </div>
                    )
                  )}
                </div>
              )}
            </div>
          </div>
        )}

      {/* ======================================================
          Candidate AI Report
      ======================================================= */}

      {selectedReportSession && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background:
              'rgba(15, 23, 42, 0.4)',
            zIndex: 1100,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '1.5rem',
          }}
        >
          <div
            className="glass-card"
            style={{
              maxWidth: '680px',
              width: '100%',
              padding: '2rem',
              maxHeight: '85vh',
              overflowY: 'auto',
            }}
          >
            <div
              style={{
                display: 'flex',
                justifyContent:
                  'space-between',
                alignItems: 'center',
                marginBottom: '1.5rem',
              }}
            >
              <div>
                <h2
                  style={{
                    fontSize: '1.3rem',
                    fontWeight: 700,
                    color:
                      'var(--text-main)',
                  }}
                >
                  AI Assessment Report:{' '}
                  {
                    selectedReportSession.name
                  }
                </h2>

                <p
                  style={{
                    color:
                      'var(--text-muted)',
                    fontSize: '0.85rem',
                  }}
                >
                  {
                    selectedReportSession.email
                  }
                </p>
              </div>

              <button
                type="button"
                onClick={() => {
                  setSelectedReportSession(
                    null
                  );
                  setSelectedReport(null);
                }}
                style={{
                  background: 'none',
                  border: 'none',
                  color:
                    'var(--text-muted)',
                  cursor: 'pointer',
                }}
              >
                <X size={20} />
              </button>
            </div>

            {loadingReport ? (
              <div
                className="loading-container"
                style={{
                  padding: '2rem',
                }}
              >
                <div className="spinner"></div>

                <p className="pulse-text">
                  Fetching AI evaluation
                  report...
                </p>
              </div>
            ) : !selectedReport ? (
              <div
                style={{
                  textAlign: 'center',
                  padding: '2rem',
                }}
              >
                <p
                  style={{
                    color:
                      'var(--text-muted)',
                  }}
                >
                  Report not available
                  yet. The report is
                  generated when the
                  candidate completes all
                  questions or the session
                  expires.
                </p>
              </div>
            ) : (
              <div
                style={{
                  background:
                    '#f8fafc',
                  border:
                    '1px solid var(--border)',
                  padding: '1.5rem',
                  borderRadius: '10px',
                }}
              >
                {/* Score */}

                <div
                  style={{
                    display: 'flex',
                    justifyContent:
                      'space-between',
                    alignItems: 'center',
                    paddingBottom:
                      '1rem',
                    marginBottom:
                      '1rem',
                    borderBottom:
                      '1px solid var(--border)',
                    flexWrap: 'wrap',
                    gap: '0.75rem',
                  }}
                >
                  <div>
                    <div
                      style={{
                        fontSize:
                          '0.75rem',
                        textTransform:
                          'uppercase',
                        color:
                          'var(--text-muted)',
                        fontWeight: 600,
                      }}
                    >
                      Overall Score
                    </div>

                    <div
                      style={{
                        fontSize:
                          '1.75rem',
                        fontWeight: 800,
                        color:
                          'var(--primary)',
                      }}
                    >
                      {selectedReport.overall_score.toFixed(
                        1
                      )}

                      <span
                        style={{
                          fontSize:
                            '1rem',
                          color:
                            'var(--text-muted)',
                        }}
                      >
                        {' '}
                        / 100
                      </span>
                    </div>
                  </div>

                  <div>
                    <div
                      style={{
                        fontSize:
                          '0.75rem',
                        textTransform:
                          'uppercase',
                        color:
                          'var(--text-muted)',
                        fontWeight: 600,
                        marginBottom:
                          '0.2rem',
                      }}
                    >
                      Recommendation
                    </div>

                    <span
                      className="badge"
                      style={{
                        textTransform:
                          'uppercase',
                        fontWeight: 700,
                        padding:
                          '0.35rem 0.85rem',
                        background:
                          selectedReport.recommendation ===
                            'strong_hire' ||
                            selectedReport.recommendation ===
                            'hire'
                            ? '#ecfdf5'
                            : selectedReport.recommendation ===
                              'maybe'
                              ? '#fefce8'
                              : '#fef2f2',
                        color:
                          selectedReport.recommendation ===
                            'strong_hire' ||
                            selectedReport.recommendation ===
                            'hire'
                            ? '#047857'
                            : selectedReport.recommendation ===
                              'maybe'
                              ? '#a16207'
                              : '#991b1b',
                        borderColor:
                          selectedReport.recommendation ===
                            'strong_hire' ||
                            selectedReport.recommendation ===
                            'hire'
                            ? '#a7f3d0'
                            : selectedReport.recommendation ===
                              'maybe'
                              ? '#fef08a'
                              : '#fecaca',
                      }}
                    >
                      {selectedReport.recommendation.replace(
                        '_',
                        ' '
                      )}
                    </span>
                  </div>
                </div>

                {/* Summary */}

                {selectedReport.summary && (
                  <div
                    style={{
                      marginBottom:
                        '1.25rem',
                    }}
                  >
                    <div
                      style={{
                        fontWeight: 600,
                        fontSize:
                          '0.9rem',
                        marginBottom:
                          '0.3rem',
                        color:
                          'var(--text-main)',
                        display: 'flex',
                        alignItems:
                          'center',
                        gap: '0.4rem',
                      }}
                    >
                      <Award
                        size={16}
                        color="var(--primary)"
                      />

                      Executive Summary
                    </div>

                    <p
                      style={{
                        fontSize:
                          '0.875rem',
                        color:
                          'var(--text-muted)',
                        lineHeight: 1.6,
                      }}
                    >
                      {
                        selectedReport.summary
                      }
                    </p>
                  </div>
                )}

                {/* Strengths / Weaknesses */}

                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns:
                      'repeat(auto-fit, minmax(240px, 1fr))',
                    gap: '1rem',
                    marginBottom:
                      '1.25rem',
                  }}
                >
                  {selectedReport.strengths &&
                    selectedReport.strengths
                      .length > 0 && (
                      <div
                        style={{
                          background:
                            '#ffffff',
                          border:
                            '1px solid var(--border)',
                          padding:
                            '0.875rem',
                          borderRadius:
                            '8px',
                        }}
                      >
                        <div
                          style={{
                            fontWeight:
                              600,
                            fontSize:
                              '0.85rem',
                            color:
                              '#047857',
                            marginBottom:
                              '0.5rem',
                            display:
                              'flex',
                            alignItems:
                              'center',
                            gap: '0.3rem',
                          }}
                        >
                          <ThumbsUp
                            size={14}
                          />

                          Demonstrated
                          Strengths
                        </div>

                        <ul
                          style={{
                            paddingLeft:
                              '1.2rem',
                            fontSize:
                              '0.8rem',
                            color:
                              'var(--text-muted)',
                            lineHeight: 1.5,
                          }}
                        >
                          {selectedReport.strengths.map(
                            (
                              strength,
                              idx
                            ) => (
                              <li
                                key={idx}
                              >
                                {strength}
                              </li>
                            )
                          )}
                        </ul>
                      </div>
                    )}

                  {selectedReport.weaknesses &&
                    selectedReport.weaknesses
                      .length > 0 && (
                      <div
                        style={{
                          background:
                            '#ffffff',
                          border:
                            '1px solid var(--border)',
                          padding:
                            '0.875rem',
                          borderRadius:
                            '8px',
                        }}
                      >
                        <div
                          style={{
                            fontWeight:
                              600,
                            fontSize:
                              '0.85rem',
                            color:
                              '#b91c1c',
                            marginBottom:
                              '0.5rem',
                            display:
                              'flex',
                            alignItems:
                              'center',
                            gap: '0.3rem',
                          }}
                        >
                          <ThumbsDown
                            size={14}
                          />

                          Areas for Growth
                        </div>

                        <ul
                          style={{
                            paddingLeft:
                              '1.2rem',
                            fontSize:
                              '0.8rem',
                            color:
                              'var(--text-muted)',
                            lineHeight: 1.5,
                          }}
                        >
                          {selectedReport.weaknesses.map(
                            (
                              weakness,
                              idx
                            ) => (
                              <li
                                key={idx}
                              >
                                {weakness}
                              </li>
                            )
                          )}
                        </ul>
                      </div>
                    )}
                </div>

                {/* Skill Scores */}

                {selectedReport.skills &&
                  selectedReport.skills
                    .length > 0 && (
                    <div>
                      <div
                        style={{
                          fontWeight: 600,
                          fontSize:
                            '0.9rem',
                          marginBottom:
                            '0.5rem',
                          color:
                            'var(--text-main)',
                          display:
                            'flex',
                          alignItems:
                            'center',
                          gap: '0.4rem',
                        }}
                      >
                        <BarChart
                          size={16}
                          color="var(--primary)"
                        />

                        Technical
                        Competency
                        Breakdown
                      </div>

                      <div
                        style={{
                          display:
                            'flex',
                          flexDirection:
                            'column',
                          gap: '0.5rem',
                        }}
                      >
                        {selectedReport.skills.map(
                          (
                            skill,
                            idx
                          ) => (
                            <div
                              key={idx}
                              style={{
                                background:
                                  '#ffffff',
                                border:
                                  '1px solid var(--border)',
                                padding:
                                  '0.69rem 0.85rem',
                                borderRadius:
                                  '6px',
                                fontSize:
                                  '0.8rem',
                              }}
                            >
                              <div
                                style={{
                                  display:
                                    'flex',
                                  justifyContent:
                                    'space-between',
                                  fontWeight:
                                    600,
                                  marginBottom:
                                    '0.2rem',
                                }}
                              >
                                <span>
                                  {
                                    skill.skill
                                  }
                                </span>

                                <span
                                  style={{
                                    color:
                                      'var(--primary)',
                                  }}
                                >
                                  {skill.score.toFixed(
                                    0
                                  )}
                                  %
                                </span>
                              </div>

                              {skill.feedback && (
                                <p
                                  style={{
                                    color:
                                      'var(--text-muted)',
                                    fontSize:
                                      '0.775rem',
                                  }}
                                >
                                  {
                                    skill.feedback
                                  }
                                </p>
                              )}
                            </div>
                          )
                        )}
                      </div>
                    </div>
                  )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}