'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { Send, Mic, Square } from 'lucide-react';
import { useDeepgramSTT } from './deepgramstt';

type AnswerInputProps = {
  onSubmit: (answer: string) => Promise<void>;
  submitting: boolean;
  disabled?: boolean;
  deepgramToken?: string;
};

export default function AnswerInput({
  onSubmit,
  submitting,
  disabled,
  deepgramToken,
}: AnswerInputProps) {
  const [answer, setAnswer] = useState('');

  const activeToken = deepgramToken || process.env.NEXT_PUBLIC_DEEPGRAM_API_KEY || '';

  const handleFinalTranscript = useCallback(
    (text: string) => {
      setAnswer((previous) => {
        const combined = previous
          ? `${previous} ${text}`
          : text;

        return combined.trim();
      });
    },
    [],
  );

  const {
    listening,
    transcript,
    interimTranscript,
    error,
    start,
    stop,
  } = useDeepgramSTT({
    token: activeToken,
    onFinalTranscript: handleFinalTranscript,
  });

  // Keep the textarea synchronized with finalized STT text.
  useEffect(() => {
    if (transcript) {
      setAnswer(transcript);
    }
  }, [transcript]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (
      !answer.trim() ||
      submitting ||
      disabled ||
      listening
    ) {
      return;
    }

    const textToSubmit = answer.trim();

    setAnswer('');

    await onSubmit(textToSubmit);
  };

  const handleVoiceToggle = async () => {
    if (listening) {
      stop();
    } else {
      await start();
    }
  };

  return (
    <form onSubmit={handleSubmit} style={{ width: '100%' }}>
      <div
        className="form-group"
        style={{ marginBottom: '1.25rem' }}
      >
        <textarea
          className="form-input textarea-input"
          placeholder={
            listening
              ? 'Listening...'
              : 'Type your answer or use the microphone...'
          }
          value={answer}
          onChange={(e) => setAnswer(e.target.value)}
          disabled={submitting || disabled}
          id="answer-textarea"
          required
        />

        {listening && interimTranscript && (
          <div
            style={{
              marginTop: '0.5rem',
              fontSize: '0.85rem',
              color: 'var(--text-dim)',
              fontStyle: 'italic',
            }}
          >
            {interimTranscript}
          </div>
        )}

        {error && (
          <div
            style={{
              marginTop: '0.5rem',
              color: 'var(--error, #ef4444)',
              fontSize: '0.85rem',
            }}
          >
            {error}
          </div>
        )}

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
          <span>
            {listening
              ? 'Speak naturally...'
              : 'Minimum 10 characters recommended'}
          </span>

          <span>{answer.length} characters</span>
        </div>
      </div>

      <div
        style={{
          display: 'flex',
          gap: '0.75rem',
        }}
      >
        <button
          type="button"
          className="btn-secondary"
          onClick={handleVoiceToggle}
          disabled={submitting || disabled}
        >
          {listening ? (
            <>
              <Square size={16} />
              Stop
            </>
          ) : (
            <>
              <Mic size={16} />
              Speak
            </>
          )}
        </button>

        <button
          type="submit"
          className="btn-primary"
          disabled={
            !answer.trim() ||
            submitting ||
            disabled ||
            listening
          }
          id="btn-submit-answer"
        >
          {submitting ? (
            <>Evaluating your answer...</>
          ) : (
            <>
              Submit Answer
              <Send size={16} />
            </>
          )}
        </button>
      </div>
    </form>
  );
}