'use client';

import React from 'react';
import { Question as QuestionType } from '@/lib/api/candidate';
import { MessageSquare } from 'lucide-react';

type QuestionProps = {
  question: QuestionType;
};

export default function Question({ question }: QuestionProps) {
  return (
    <div style={{ marginBottom: '1.75rem' }}>
      <div
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '0.4rem',
          color: 'var(--primary)',
          fontSize: '0.875rem',
          fontWeight: 600,
          marginBottom: '0.5rem',
        }}
      >
        <MessageSquare size={16} /> Question {question.order}
      </div>

      <h2
        style={{
          fontSize: '1.3rem',
          fontWeight: 600,
          color: 'var(--text-main)',
          lineHeight: 1.5,
          letterSpacing: '-0.01em',
        }}
        id="question-text"
      >
        {question.text}
      </h2>
    </div>
  );
}
