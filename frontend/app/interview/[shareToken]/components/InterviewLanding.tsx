'use client';

import React from 'react';
import { InterviewDetails } from '@/lib/api/candidate';
import { Clock, Briefcase, BarChart2, ArrowRight } from 'lucide-react';

type InterviewLandingProps = {
  interview: InterviewDetails;
  onStartClick: () => void;
};

export default function InterviewLanding({ interview, onStartClick }: InterviewLandingProps) {
  return (
    <div className="container" style={{ maxWidth: '640px' }}>
      <div className="glass-card" style={{ textAlign: 'center' }}>
        <div style={{ marginBottom: '1.25rem' }}>
          <span className="badge">NexHire Technical Interview</span>
        </div>

        <h1
          style={{
            fontSize: '2rem',
            fontWeight: 700,
            marginBottom: '0.75rem',
            lineHeight: 1.25,
            color: 'var(--text-main)',
            letterSpacing: '-0.02em',
          }}
        >
          {interview.title}
        </h1>

        {interview.description && (
          <p style={{ color: 'var(--text-muted)', marginBottom: '1.5rem', fontSize: '0.95rem', lineHeight: 1.6 }}>
            {interview.description}
          </p>
        )}

        <div className="meta-grid">
          <div className="meta-item">
            <div className="meta-item-label" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.25rem' }}>
              <Clock size={14} /> Duration
            </div>
            <div className="meta-item-value">{interview.duration} mins</div>
          </div>

          <div className="meta-item">
            <div className="meta-item-label" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.25rem' }}>
              <Briefcase size={14} /> Target Role
            </div>
            <div className="meta-item-value">{interview.role}</div>
          </div>

          <div className="meta-item">
            <div className="meta-item-label" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.25rem' }}>
              <BarChart2 size={14} /> Difficulty
            </div>
            <div className="meta-item-value" style={{ textTransform: 'capitalize' }}>
              {interview.difficulty}
            </div>
          </div>
        </div>

        <button onClick={onStartClick} className="btn-primary" id="btn-start-interview">
          Start Interview Session <ArrowRight size={18} />
        </button>
      </div>
    </div>
  );
}
