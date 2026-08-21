'use client';

import React, { useState } from 'react';
import { Send } from 'lucide-react';

type AnswerInputProps = {
  onSubmit: (answer: string) => Promise<void>;
  submitting: boolean;
  disabled?: boolean;
};

export default function AnswerInput({ onSubmit, submitting, disabled }: AnswerInputProps) {
  const [answer, setAnswer] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!answer.trim() || submitting || disabled) return;

    const textToSubmit = answer;
    setAnswer('');
    await onSubmit(textToSubmit);
  };

  return (
    <form onSubmit={handleSubmit} style={{ width: '100%' }}>
      <div className="form-group" style={{ marginBottom: '1.25rem' }}>
        <textarea
          className="form-input textarea-input"
          placeholder="Type your answer..."
          value={answer}
          onChange={(e) => setAnswer(e.target.value)}
          disabled={submitting || disabled}
          id="answer-textarea"
          required
        />
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginTop: '0.5rem',
            fontSize: '0.8rem',
            color: 'var(--text-dim)',
          }}
        >
          <span>Minimum 10 characters recommended</span>
          <span>{answer.length} characters</span>
        </div>
      </div>

      <button
        type="submit"
        className="btn-primary"
        disabled={!answer.trim() || submitting || disabled}
        id="btn-submit-answer"
      >
        {submitting ? (
          <>Evaluating your answer...</>
        ) : (
          <>
            Submit Answer <Send size={16} />
          </>
        )}
      </button>
    </form>
  );
}
