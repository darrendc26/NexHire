'use client';

import React, { useState } from 'react';
import { User, Mail, Sparkles, AlertCircle } from 'lucide-react';

type CandidateFormProps = {
  onSubmit: (name: string, email: string) => Promise<void>;
  loading: boolean;
  error?: string | null;
};

export default function CandidateForm({ onSubmit, loading, error }: CandidateFormProps) {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [localError, setLocalError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLocalError(null);

    if (!name.trim()) {
      setLocalError('Please enter your name');
      return;
    }

    if (!email.trim() || !email.includes('@')) {
      setLocalError('Please enter a valid email address');
      return;
    }

    try {
      await onSubmit(name.trim(), email.trim());
    } catch (err: unknown) {
      if (err instanceof Error) {
        setLocalError(err.message);
      } else {
        setLocalError('An error occurred. Please try again.');
      }
    }
  };

  const displayError = localError || error;

  return (
    <div className="container" style={{ maxWidth: '500px' }}>
      <div className="glass-card">
        <h2 style={{ fontSize: '1.4rem', fontWeight: 700, marginBottom: '0.4rem', textAlign: 'center', color: 'var(--text-main)' }}>
          Candidate Registration
        </h2>
        <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginBottom: '1.75rem', textAlign: 'center' }}>
          Please enter your details to begin your AI technical interview.
        </p>

        {displayError && (
          <div
            style={{
              background: '#fef2f2',
              border: '1px solid #fecaca',
              color: '#991b1b',
              padding: '0.75rem 1rem',
              borderRadius: '8px',
              fontSize: '0.875rem',
              marginBottom: '1.25rem',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
            }}
          >
            <AlertCircle size={16} /> {displayError}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label" htmlFor="candidate-name">
              Full Name
            </label>
            <div style={{ position: 'relative' }}>
              <input
                id="candidate-name"
                type="text"
                className="form-input"
                style={{ width: '100%', paddingLeft: '2.5rem' }}
                placeholder="Jane Doe"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={loading}
                required
              />
              <User
                size={18}
                style={{
                  position: 'absolute',
                  left: '0.875rem',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  color: 'var(--text-muted)',
                }}
              />
            </div>
          </div>

          <div className="form-group">
            <label className="form-label" htmlFor="candidate-email">
              Email Address
            </label>
            <div style={{ position: 'relative' }}>
              <input
                id="candidate-email"
                type="email"
                className="form-input"
                style={{ width: '100%', paddingLeft: '2.5rem' }}
                placeholder="jane@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={loading}
                required
              />
              <Mail
                size={18}
                style={{
                  position: 'absolute',
                  left: '0.875rem',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  color: 'var(--text-muted)',
                }}
              />
            </div>
          </div>

          <button
            type="submit"
            className="btn-primary"
            disabled={loading}
            id="btn-begin-interview"
            style={{ marginTop: '0.75rem' }}
          >
            {loading ? (
              <>Creating Session...</>
            ) : (
              <>
                Begin Interview <Sparkles size={18} />
              </>
            )}
          </button>
        </form>
      </div>
    </div>
  );
}
