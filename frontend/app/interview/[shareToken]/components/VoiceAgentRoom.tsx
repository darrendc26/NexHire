'use client';

import React, { useEffect, useState, useRef, useCallback } from 'react';
import { Question as QuestionType, getSTTToken } from '@/lib/api/candidate';
import { useDeepgramSTT } from './deepgramstt';
import {
  Bot,
  Mic,
  MicOff,
  Volume2,
  VolumeX,
  RotateCcw,
  Send,
  Sparkles,
  ChevronDown,
  ChevronUp,
  Radio,
  CheckCircle2,
} from 'lucide-react';

type VoiceAgentRoomProps = {
  question: QuestionType;
  deepgramToken: string;
  sessionToken?: string;
  onSubmitAnswer: (answerText: string) => Promise<void>;
  submitting: boolean;
  questionNumber: number;
  totalQuestions?: number;
};

type AgentState = 'interviewer_speaking' | 'listening' | 'evaluating' | 'paused';

export default function VoiceAgentRoom({
  question,
  deepgramToken,
  sessionToken,
  onSubmitAnswer,
  submitting,
  questionNumber,
}: VoiceAgentRoomProps) {
  const [agentState, setAgentState] = useState<AgentState>('interviewer_speaking');
  const [micMuted, setMicMuted] = useState(false);
  const [showTranscript, setShowTranscript] = useState(true);
  const [liveAnswer, setLiveAnswer] = useState('');
  const [silenceCountdown, setSilenceCountdown] = useState<number | null>(null);

  const [activeToken, setActiveToken] = useState(
    deepgramToken || process.env.NEXT_PUBLIC_DEEPGRAM_API_KEY || ''
  );

  const audioRef = useRef<HTMLAudioElement | null>(null);
  const lastPlayedQuestionIdRef = useRef<string | null>(null);
  const autoSubmitTimerRef = useRef<NodeJS.Timeout | null>(null);
  const silenceIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // Synchronize or dynamically fetch STT Token if missing
  useEffect(() => {
    if (deepgramToken) {
      setActiveToken(deepgramToken);
    } else if (sessionToken) {
      getSTTToken(sessionToken)
        .then((tok) => {
          if (tok) setActiveToken(tok);
        })
        .catch((err) => console.error('Failed to load STT token dynamically:', err));
    }
  }, [deepgramToken, sessionToken]);

  // Clear auto-submit timers
  const clearTimers = useCallback(() => {
    if (autoSubmitTimerRef.current) {
      clearTimeout(autoSubmitTimerRef.current);
      autoSubmitTimerRef.current = null;
    }
    if (silenceIntervalRef.current) {
      clearInterval(silenceIntervalRef.current);
      silenceIntervalRef.current = null;
    }
    setSilenceCountdown(null);
  }, []);

  // Stop any active TTS audio playback cleanly
  const stopTTS = useCallback(() => {
    if (audioRef.current) {
      audioRef.current.pause();
      audioRef.current.currentTime = 0;
      audioRef.current = null;
    }
  }, []);

  const {
    listening,
    transcript,
    interimTranscript,
    error: sttError,
    start: startSTT,
    stop: stopSTT,
  } = useDeepgramSTT({
    token: activeToken,
  });

  // Sync live answer with accumulated Deepgram STT transcript
  useEffect(() => {
    setLiveAnswer(transcript);
  }, [transcript]);

  // Handle Answer Submission
  const handlePerformSubmit = useCallback(async () => {
    clearTimers();
    stopSTT();
    stopTTS();

    const textToSubmit = (liveAnswer || transcript || interimTranscript).trim();
    if (!textToSubmit) return;

    setAgentState('evaluating');
    await onSubmitAnswer(textToSubmit);
  }, [liveAnswer, transcript, interimTranscript, onSubmitAnswer, clearTimers, stopSTT, stopTTS]);

  // Single-instance Audio Playback Function
  const startAudioPlayback = useCallback((text: string) => {
    clearTimers();
    stopSTT();
    stopTTS();
    setLiveAnswer('');
    setAgentState('interviewer_speaking');

    if (!text) return;

    try {
      const BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';
      const ttsUrl = `${BASE_URL}/api/candidates/tts?text=${encodeURIComponent(text)}`;
      const audio = new Audio(ttsUrl);
      audioRef.current = audio;

      audio.onplay = () => {
        setAgentState('interviewer_speaking');
      };

      audio.onended = () => {
        audioRef.current = null;
        setAgentState('listening');
      };

      audio.onerror = (err) => {
        console.warn('TTS audio playback error, enabling listening state:', err);
        audioRef.current = null;
        setAgentState('listening');
      };

      audio.play().catch((playErr) => {
        console.warn('Audio play prevented by browser autoplay policy:', playErr);
        audioRef.current = null;
        setAgentState('listening');
      });
    } catch (err) {
      console.error('Error starting TTS audio:', err);
      setAgentState('listening');
    }
  }, [clearTimers, stopSTT, stopTTS]);

  // Play question audio ONCE per question ID
  useEffect(() => {
    if (question?.id && question?.text && lastPlayedQuestionIdRef.current !== question.id) {
      lastPlayedQuestionIdRef.current = question.id;
      startAudioPlayback(question.text);
    }
  }, [question.id, question.text, startAudioPlayback]);

  // Replay handler for Repeat Question button
  const handleRepeatQuestion = () => {
    if (question?.text) {
      startAudioPlayback(question.text);
    }
  };

  // Safe STT lifecycle manager based on agent state and mute
  useEffect(() => {
    if (agentState === 'listening' && !micMuted && activeToken && !listening) {
      startSTT();
    } else if ((agentState !== 'listening' || micMuted) && listening) {
      stopSTT();
    }
  }, [agentState, micMuted, activeToken, listening, startSTT, stopSTT]);

  // Clean up on component unmount only
  useEffect(() => {
    return () => {
      stopTTS();
      stopSTT();
      clearTimers();
    };
  }, [stopTTS, stopSTT, clearTimers]);

  // Auto-submit silence detection when candidate stops speaking after giving an answer
  useEffect(() => {
    if (agentState === 'listening' && liveAnswer.trim().length > 15) {
      clearTimers();
      setSilenceCountdown(4);

      let current = 4;
      silenceIntervalRef.current = setInterval(() => {
        current -= 1;
        if (current <= 0) {
          if (silenceIntervalRef.current) clearInterval(silenceIntervalRef.current);
          setSilenceCountdown(null);
        } else {
          setSilenceCountdown(current);
        }
      }, 1000);

      autoSubmitTimerRef.current = setTimeout(() => {
        handlePerformSubmit();
      }, 4000);
    }
  }, [liveAnswer, agentState, clearTimers, handlePerformSubmit]);

  // Toggle Mute
  const toggleMute = () => {
    if (micMuted) {
      setMicMuted(false);
      if (agentState === 'listening') {
        startSTT();
      }
    } else {
      setMicMuted(true);
      stopSTT();
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: '100%' }}>
      {/* Top Status Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          width: '100%',
          marginBottom: '1.5rem',
          padding: '0.5rem 1rem',
          background: 'rgba(241, 245, 249, 0.6)',
          borderRadius: '12px',
          border: '1px solid var(--border)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <Radio
            size={18}
            className={agentState === 'listening' ? 'pulse-icon' : ''}
            color={
              agentState === 'interviewer_speaking'
                ? '#6366f1'
                : agentState === 'listening'
                ? '#10b981'
                : '#f59e0b'
            }
          />
          <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-main)' }}>
            Question {questionNumber} of 5
          </span>
        </div>

        <div style={{ fontSize: '0.8rem', fontWeight: 500, color: 'var(--text-muted)' }}>
          {agentState === 'interviewer_speaking' && (
            <span style={{ color: '#4f46e5', fontWeight: 600 }}>Interviewer Speaking...</span>
          )}
          {agentState === 'listening' && (
            <span style={{ color: '#059669', fontWeight: 600 }}>
              {listening ? 'Listening to you...' : 'Mic Ready'}
            </span>
          )}
          {agentState === 'evaluating' && (
            <span style={{ color: '#d97706', fontWeight: 600 }}>Evaluating your response...</span>
          )}
        </div>
      </div>

      {/* Interactive AI Interviewer Avatar Box */}
      <div
        style={{
          position: 'relative',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '2.5rem 1.5rem',
          width: '100%',
          borderRadius: '20px',
          background:
            agentState === 'interviewer_speaking'
              ? 'linear-gradient(135deg, rgba(99, 102, 241, 0.08) 0%, rgba(168, 85, 247, 0.08) 100%)'
              : agentState === 'listening'
              ? 'linear-gradient(135deg, rgba(16, 185, 129, 0.08) 0%, rgba(59, 130, 246, 0.08) 100%)'
              : 'linear-gradient(135deg, rgba(245, 158, 11, 0.08) 0%, rgba(239, 68, 68, 0.08) 100%)',
          border:
            agentState === 'interviewer_speaking'
              ? '1px solid rgba(99, 102, 241, 0.25)'
              : agentState === 'listening'
              ? '1px solid rgba(16, 185, 129, 0.25)'
              : '1px solid rgba(245, 158, 11, 0.25)',
          marginBottom: '1.5rem',
          transition: 'all 0.4s ease',
        }}
      >
        {/* Pulsating Ring Avatar Container */}
        <div
          style={{
            position: 'relative',
            width: '110px',
            height: '110px',
            borderRadius: '50%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: '1.25rem',
            background:
              agentState === 'interviewer_speaking'
                ? '#4f46e5'
                : agentState === 'listening'
                ? '#10b981'
                : '#f59e0b',
            boxShadow:
              agentState === 'interviewer_speaking'
                ? '0 0 25px rgba(99, 102, 241, 0.4)'
                : agentState === 'listening'
                ? '0 0 25px rgba(16, 185, 129, 0.4)'
                : '0 0 25px rgba(245, 158, 11, 0.4)',
            transition: 'all 0.3s ease',
          }}
        >
          <Bot size={54} color="#ffffff" />

          {/* Sound wave pulse animation elements */}
          {agentState === 'interviewer_speaking' && (
            <div
              style={{
                position: 'absolute',
                inset: '-12px',
                borderRadius: '50%',
                border: '2px solid rgba(99, 102, 241, 0.6)',
                animation: 'ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite',
              }}
            />
          )}

          {agentState === 'listening' && listening && (
            <div
              style={{
                position: 'absolute',
                inset: '-12px',
                borderRadius: '50%',
                border: '2px solid rgba(16, 185, 129, 0.6)',
                animation: 'ping 1.2s cubic-bezier(0, 0, 0.2, 1) infinite',
              }}
            />
          )}
        </div>

        {/* AI Interviewer Persona Title */}
        <h3 style={{ fontSize: '1.25rem', fontWeight: 700, color: 'var(--text-main)', marginBottom: '0.25rem' }}>
          Alex • NexHire AI Interviewer
        </h3>

        {/* Subtitle helper */}
        <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', textAlign: 'center', maxWidth: '420px' }}>
          {agentState === 'interviewer_speaking' && 'Speaking question aloud...'}
          {agentState === 'listening' && 'Listening hands-free. Speak your answer naturally.'}
          {agentState === 'evaluating' && 'Analyzing response & preparing feedback...'}
        </p>

        {/* Auto-submit silence banner indicator */}
        {silenceCountdown !== null && agentState === 'listening' && (
          <div
            style={{
              marginTop: '1rem',
              padding: '0.4rem 0.85rem',
              background: '#ecfdf5',
              border: '1px solid #a7f3d0',
              borderRadius: '20px',
              color: '#047857',
              fontSize: '0.8rem',
              fontWeight: 600,
              display: 'flex',
              alignItems: 'center',
              gap: '0.4rem',
            }}
          >
            <Sparkles size={14} /> Auto-submitting in {silenceCountdown}s (silence detected)...
          </div>
        )}
      </div>

      {/* Hands-Free Interactive Controls Bar */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '1rem',
          marginBottom: '1.5rem',
          flexWrap: 'wrap',
        }}
      >
        <button
          type="button"
          onClick={handleRepeatQuestion}
          disabled={submitting}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '0.4rem',
            padding: '0.6rem 1.1rem',
            borderRadius: '10px',
            background: '#ffffff',
            border: '1px solid var(--border)',
            color: 'var(--text-main)',
            fontSize: '0.875rem',
            fontWeight: 500,
            cursor: 'pointer',
            boxShadow: '0 2px 4px rgba(0,0,0,0.02)',
          }}
          title="Repeat Question"
        >
          <RotateCcw size={16} /> Repeat Question
        </button>

        <button
          type="button"
          onClick={toggleMute}
          disabled={submitting}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '0.4rem',
            padding: '0.6rem 1.1rem',
            borderRadius: '10px',
            background: micMuted ? '#fef2f2' : '#ffffff',
            border: micMuted ? '1px solid #fecaca' : '1px solid var(--border)',
            color: micMuted ? '#dc2626' : 'var(--text-main)',
            fontSize: '0.875rem',
            fontWeight: 500,
            cursor: 'pointer',
            boxShadow: '0 2px 4px rgba(0,0,0,0.02)',
          }}
        >
          {micMuted ? <MicOff size={16} color="#dc2626" /> : <Mic size={16} color="#059669" />}
          {micMuted ? 'Unmute Mic' : 'Mute Mic'}
        </button>

        <button
          type="button"
          onClick={handlePerformSubmit}
          disabled={!liveAnswer.trim() || submitting}
          className="btn-primary"
          style={{
            width: 'auto',
            padding: '0.6rem 1.25rem',
            fontSize: '0.875rem',
          }}
        >
          {submitting ? (
            'Evaluating...'
          ) : (
            <>
              Send Answer Now <Send size={16} />
            </>
          )}
        </button>
      </div>

      {/* Live Recognized Speech Transcript Section */}
      <div style={{ width: '100%', borderRadius: '12px', border: '1px solid var(--border)', overflow: 'hidden' }}>
        <button
          type="button"
          onClick={() => setShowTranscript(!showTranscript)}
          style={{
            width: '100%',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '0.75rem 1rem',
            background: 'var(--card-bg, #f8fafc)',
            border: 'none',
            cursor: 'pointer',
            fontSize: '0.85rem',
            fontWeight: 600,
            color: 'var(--text-main)',
          }}
        >
          <span style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
            <Volume2 size={16} color="var(--primary)" /> Real-Time Answer Transcript
          </span>
          {showTranscript ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </button>

        {showTranscript && (
          <div style={{ padding: '1rem', background: '#ffffff' }}>
            <div
              style={{
                minHeight: '80px',
                maxHeight: '160px',
                overflowY: 'auto',
                padding: '0.75rem',
                borderRadius: '8px',
                background: '#f8fafc',
                border: '1px solid #e2e8f0',
                fontSize: '0.9rem',
                lineHeight: 1.6,
                color: liveAnswer ? 'var(--text-main)' : 'var(--text-muted)',
              }}
            >
              {liveAnswer ? (
                <span>
                  {liveAnswer}{' '}
                  {interimTranscript && (
                    <span style={{ color: '#94a3b8', fontStyle: 'italic' }}>{interimTranscript}</span>
                  )}
                </span>
              ) : (
                <span style={{ fontStyle: 'italic' }}>
                  {agentState === 'interviewer_speaking'
                    ? 'Interviewer is currently asking the question...'
                    : 'Start speaking when ready. Your recognized response will appear here live...'}
                </span>
              )}
            </div>

            {sttError && (
              <div style={{ marginTop: '0.5rem', color: '#dc2626', fontSize: '0.8rem' }}>
                Microphone / Speech Notice: {sttError}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
