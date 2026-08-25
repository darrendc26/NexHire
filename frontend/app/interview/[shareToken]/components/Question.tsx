'use client';

import React, { useEffect, useState, useCallback, useRef } from 'react';
import { Question as QuestionType } from '@/lib/api/candidate';
import { MessageSquare, Volume2, VolumeX } from 'lucide-react';

type QuestionProps = {
  question: QuestionType;
  autoSpeak?: boolean;
};

export default function Question({ question, autoSpeak = false }: QuestionProps) {
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [supported, setSupported] = useState(false);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setSupported(true);
    }
  }, []);

  const stopSpeech = useCallback(() => {
    if (audioRef.current) {
      audioRef.current.pause();
      audioRef.current.currentTime = 0;
      audioRef.current = null;
    }
    if (typeof window !== 'undefined' && 'speechSynthesis' in window) {
      window.speechSynthesis.cancel();
    }
    setIsSpeaking(false);
  }, []);

  const speakQuestion = useCallback(() => {
    stopSpeech();

    if (typeof window === 'undefined' || !question.text) return;

    try {
      const BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';
      const ttsUrl = `${BASE_URL}/api/candidates/tts?text=${encodeURIComponent(question.text)}`;

      const audio = new Audio(ttsUrl);
      audioRef.current = audio;

      audio.onplay = () => setIsSpeaking(true);
      audio.onended = () => {
        setIsSpeaking(false);
        audioRef.current = null;
      };
      audio.onerror = (err) => {
        console.error('Deepgram TTS audio error:', err);
        setIsSpeaking(false);
        audioRef.current = null;
      };

      audio.play().catch((err) => {
        console.warn('Audio play prevented by browser autoplay policy:', err);
        setIsSpeaking(false);
      });
    } catch (e) {
      console.error('Failed to start TTS:', e);
      setIsSpeaking(false);
    }
  }, [question.text, stopSpeech]);

  // Handle auto-speak ONLY when autoSpeak is explicitly enabled and question changes
  useEffect(() => {
    if (supported && autoSpeak && question.text) {
      speakQuestion();
    }
  }, [question.id, autoSpeak]); // eslint-disable-line react-hooks/exhaustive-deps

  // Cleanup speech on component unmount only
  useEffect(() => {
    return () => {
      stopSpeech();
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const toggleSpeech = () => {
    if (isSpeaking) {
      stopSpeech();
    } else {
      speakQuestion();
    }
  };

  return (
    <div style={{ marginBottom: '1.75rem' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: '0.5rem',
        }}
      >
        <div
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '0.4rem',
            color: 'var(--primary)',
            fontSize: '0.875rem',
            fontWeight: 600,
          }}
        >
          <MessageSquare size={16} /> Question {question.order}
        </div>

        {supported && (
          <button
            type="button"
            onClick={toggleSpeech}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '0.35rem',
              background: isSpeaking ? '#e0e7ff' : 'var(--card-bg, #f8fafc)',
              border: `1px solid ${isSpeaking ? 'var(--primary)' : 'var(--border, #e2e8f0)'}`,
              color: isSpeaking ? 'var(--primary)' : 'var(--text-muted, #64748b)',
              padding: '0.3rem 0.65rem',
              borderRadius: '6px',
              fontSize: '0.8rem',
              fontWeight: 500,
              cursor: 'pointer',
              transition: 'all 0.2s ease',
            }}
            title={isSpeaking ? 'Stop reading' : 'Read question aloud'}
            id="btn-toggle-question-tts"
          >
            {isSpeaking ? (
              <>
                <VolumeX size={15} color="var(--primary)" />
                <span>Stop</span>
              </>
            ) : (
              <>
                <Volume2 size={15} />
                <span>Listen</span>
              </>
            )}
          </button>
        )}
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
