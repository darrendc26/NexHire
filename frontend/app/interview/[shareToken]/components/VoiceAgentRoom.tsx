'use client';

import React, { useEffect, useState, useRef, useCallback } from 'react';
import { fetchTTSAudio, getSTTToken, Question as QuestionType } from '@/lib/api/candidate';
import { playTtsBlob, stopTtsPlayback, unlockPlayback } from '@/lib/audioSession';
import { useDeepgramSTT } from './deepgramstt';
import {
  Bot,
  Mic,
  MicOff,
  RotateCcw,
  Settings,
  LogOut,
  Clock,
  ChevronDown,
  ChevronUp,
  AudioLines,
  Volume2,
  VolumeX,
  Lock,
  Send,
} from 'lucide-react';
import styles from './VoiceAgentRoom.module.css';

type VoiceAgentRoomProps = {
  question: QuestionType;
  deepgramToken: string;
  sessionToken?: string;
  onSubmitAnswer: (answerText: string) => Promise<void>;
  submitting: boolean;
  questionNumber: number;
  totalQuestions?: number;
  candidateName: string;
  remainingSeconds: number;
  onTimeUp: () => void;
  onEndInterview: () => void;
  questionError?: string | null;
};

type AgentState = 'interviewer_speaking' | 'listening' | 'evaluating' | 'paused';

function WaveBars({
  count,
  active,
  heights,
}: {
  count: number;
  active: boolean;
  heights?: number[];
}) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => {
        const h = heights?.[i] ?? 8 + ((i * 7) % 18);
        return (
          <span
            key={i}
            className={`${styles.bar} ${active ? styles.barActive : ''}`}
            style={{
              height: `${h}px`,
              animationDelay: `${(i % 8) * 0.08}s`,
            }}
          />
        );
      })}
    </>
  );
}

function formatRemaining(totalSecs: number) {
  if (totalSecs <= 0) return "Time's up";
  const mins = Math.floor(totalSecs / 60);
  const secs = totalSecs % 60;
  return `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')} remaining`;
}

export default function VoiceAgentRoom({
  question,
  deepgramToken,
  sessionToken,
  onSubmitAnswer,
  submitting,
  questionNumber,
  totalQuestions = 5,
  candidateName,
  remainingSeconds,
  onTimeUp,
  onEndInterview,
  questionError,
}: VoiceAgentRoomProps) {
  const [agentState, setAgentState] = useState<AgentState>('interviewer_speaking');
  const [micMuted, setMicMuted] = useState(false);
  const [speakerMuted, setSpeakerMuted] = useState(false);
  const [showTranscript, setShowTranscript] = useState(true);
  const [showSettings, setShowSettings] = useState(false);
  const [showEndConfirm, setShowEndConfirm] = useState(false);
  const [liveAnswer, setLiveAnswer] = useState('');
  const [silenceCountdown, setSilenceCountdown] = useState<number | null>(null);
  const [secondsLeft, setSecondsLeft] = useState(remainingSeconds);
  const [needsAudioUnlock, setNeedsAudioUnlock] = useState(false);
  const [ttsError, setTtsError] = useState<string | null>(null);

  const firstName = candidateName.trim().split(/\s+/)[0] || 'there';

  const [activeToken, setActiveToken] = useState(
    deepgramToken || process.env.NEXT_PUBLIC_DEEPGRAM_API_KEY || ''
  );

  const audioElRef = useRef<HTMLAudioElement | null>(null);
  const objectUrlRef = useRef<string | null>(null);
  const pendingQuestionTextRef = useRef<string | null>(null);
  const pendingBlobRef = useRef<Blob | null>(null);
  const playGenerationRef = useRef(0);
  const lastPlayedQuestionIdRef = useRef<string | null>(null);
  const autoSubmitTimerRef = useRef<NodeJS.Timeout | null>(null);
  const silenceIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const speakerMutedRef = useRef(speakerMuted);
  speakerMutedRef.current = speakerMuted;

  const timeUpFiredRef = useRef(false);

  useEffect(() => {
    setSecondsLeft(remainingSeconds);
  }, [remainingSeconds]);

  useEffect(() => {
    const interval = setInterval(() => {
      setSecondsLeft((prev) => (prev <= 1 ? 0 : prev - 1));
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (secondsLeft === 0 && !timeUpFiredRef.current) {
      timeUpFiredRef.current = true;
      onTimeUp();
    }
  }, [secondsLeft, onTimeUp]);

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

  const stopTTS = useCallback(() => {
    playGenerationRef.current += 1;
    stopTtsPlayback();
    const audio = audioElRef.current;
    if (audio) {
      audio.onended = null;
      audio.onerror = null;
      audio.pause();
      audio.removeAttribute('src');
      audio.load();
    }
    if (objectUrlRef.current) {
      URL.revokeObjectURL(objectUrlRef.current);
      objectUrlRef.current = null;
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

  useEffect(() => {
    setLiveAnswer(transcript);
  }, [transcript]);

  const handlePerformSubmit = useCallback(async () => {
    clearTimers();
    stopSTT();
    stopTTS();

    const textToSubmit = (liveAnswer || transcript || interimTranscript).trim();
    if (!textToSubmit) return;

    setAgentState('evaluating');
    await onSubmitAnswer(textToSubmit);
  }, [liveAnswer, transcript, interimTranscript, onSubmitAnswer, clearTimers, stopSTT, stopTTS]);

  const startAudioPlayback = useCallback(
    async (text: string) => {
      clearTimers();
      stopSTT();
      stopTTS();
      setLiveAnswer('');
      setTtsError(null);
      setNeedsAudioUnlock(false);
      pendingQuestionTextRef.current = text;

      if (!text) {
        setAgentState('listening');
        return;
      }

      if (speakerMutedRef.current) {
        setAgentState('listening');
        return;
      }

      setAgentState('interviewer_speaking');
      const generation = playGenerationRef.current;

      try {
        const blob = await fetchTTSAudio(text);
        if (generation !== playGenerationRef.current) return;
        pendingBlobRef.current = blob;

        const url = URL.createObjectURL(blob);
        objectUrlRef.current = url;
        if (audioElRef.current) {
          audioElRef.current.src = url;
        }

        try {
          await unlockPlayback();
          await playTtsBlob(blob);
          if (generation !== playGenerationRef.current) return;
          pendingQuestionTextRef.current = null;
          setNeedsAudioUnlock(false);
          setAgentState('listening');
        } catch (playErr) {
          console.warn('Audio play prevented by browser autoplay policy:', playErr);
          setNeedsAudioUnlock(true);
          setAgentState('listening');
        }
      } catch (err) {
        console.error('Error starting TTS audio:', err);
        setTtsError(err instanceof Error ? err.message : 'Failed to load interviewer audio.');
        setAgentState('listening');
      }
    },
    [clearTimers, stopSTT, stopTTS]
  );

  const unlockAndPlay = useCallback(async () => {
    if (speakerMutedRef.current) return;
    await unlockPlayback();

    if (pendingBlobRef.current) {
      try {
        setAgentState('interviewer_speaking');
        stopSTT();
        setNeedsAudioUnlock(false);
        setTtsError(null);
        const generation = playGenerationRef.current;
        await playTtsBlob(pendingBlobRef.current);
        if (generation !== playGenerationRef.current) return;
        pendingQuestionTextRef.current = null;
        setAgentState('listening');
        return;
      } catch (err) {
        console.warn('Retry audio play failed:', err);
        setAgentState('listening');
      }
    }

    if (pendingQuestionTextRef.current || question?.text) {
      await startAudioPlayback(pendingQuestionTextRef.current || question.text);
    }
  }, [question.text, startAudioPlayback, stopSTT]);

  useEffect(() => {
    if (question?.id && question?.text && lastPlayedQuestionIdRef.current !== question.id) {
      lastPlayedQuestionIdRef.current = question.id;
      startAudioPlayback(question.text);
    }
  }, [question.id, question.text, startAudioPlayback]);

  const handleRepeatQuestion = () => {
    void unlockAndPlay();
  };

  useEffect(() => {
    if (agentState === 'listening' && !micMuted && activeToken && !listening) {
      void startSTT();
    } else if ((agentState !== 'listening' || micMuted) && listening) {
      stopSTT();
    }
  }, [agentState, micMuted, activeToken, listening, startSTT, stopSTT]);

  useEffect(() => {
    if (!needsAudioUnlock) return;
    const onFirstGesture = () => {
      void unlockAndPlay();
    };
    window.addEventListener('pointerdown', onFirstGesture, { once: true });
    window.addEventListener('keydown', onFirstGesture, { once: true });
    return () => {
      window.removeEventListener('pointerdown', onFirstGesture);
      window.removeEventListener('keydown', onFirstGesture);
    };
  }, [needsAudioUnlock, unlockAndPlay]);

  useEffect(() => {
    return () => {
      stopTTS();
      stopSTT();
      clearTimers();
    };
  }, [stopTTS, stopSTT, clearTimers]);

  useEffect(() => {
    if (agentState !== 'listening') return;

    const spoken = `${liveAnswer} ${interimTranscript}`.trim();
    if (spoken.length <= 15) return;

    clearTimers();
    setSilenceCountdown(6);

    let current = 6;
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
    }, 6000);
  }, [liveAnswer, interimTranscript, agentState, clearTimers, handlePerformSubmit]);

  const toggleMute = () => {
    if (micMuted) {
      setMicMuted(false);
    } else {
      setMicMuted(true);
      stopSTT();
    }
  };

  const toggleSpeaker = () => {
    const next = !speakerMuted;
    setSpeakerMuted(next);
    if (next) {
      stopTTS();
      if (agentState === 'interviewer_speaking') {
        setAgentState('listening');
      }
    }
  };

  const isListening = agentState === 'listening' && !micMuted;
  const isSpeaking = agentState === 'interviewer_speaking';
  const isEvaluating = agentState === 'evaluating' || submitting;
  const waveActive = isListening || isSpeaking;

  const statusLabel = isEvaluating
    ? 'Evaluating your response...'
    : isSpeaking
    ? 'Asking the question...'
    : micMuted
    ? 'Mic is muted'
    : 'Listening to you...';

  const liveText = (liveAnswer || interimTranscript).trim();
  const timerUrgent = secondsLeft <= 120 && secondsLeft > 0;

  return (
    <div className={styles.stage}>
      <audio ref={audioElRef} preload="auto" />
      <div className={styles.backdrop} aria-hidden />
      <div className={styles.scrim} aria-hidden />

      <div className={styles.content}>
        <header className={styles.header}>
          <div className={styles.brand}>
            <div className={styles.brandMark}>
              <Bot size={22} />
            </div>
            <div>
              <div className={styles.brandName}>NexHire AI</div>
              <div className={styles.brandSub}>Your AI Interviewer</div>
            </div>
          </div>

          <div className={styles.questionPill}>
            <span className={styles.dot} />
            Question {questionNumber} of {totalQuestions}
          </div>

          <div className={styles.headerRight}>
            <div className={`${styles.timer} ${timerUrgent ? styles.timerUrgent : ''}`}>
              <Clock size={16} />
              <span>{formatRemaining(secondsLeft)}</span>
            </div>
            <button type="button" className={styles.endBtn} onClick={() => setShowEndConfirm(true)}>
              <LogOut size={15} />
              End Interview
            </button>
          </div>
        </header>

        <section className={styles.greeting}>
          <p className={styles.hello}>Hi {firstName} 👋</p>
          <h1 className={styles.title}>Let&apos;s have a great conversation.</h1>
          <p className={styles.subtitle}>I&apos;ll ask the questions, you just focus on sharing your best.</p>
        </section>

        <div className={styles.workspace}>
          <div className={styles.card}>
            <div className={styles.agentPill}>
              <img src="/ai-interviewer.png" alt="" />
              AI Interviewer
              <span className={styles.statusLive}>
                <span className={styles.dot} />
                {isEvaluating ? 'Thinking' : isSpeaking ? 'Speaking' : 'Listening'}
              </span>
            </div>

            <div className={styles.questionRow}>
              <div className={styles.sideWave}>
                <WaveBars count={6} active={waveActive} heights={[8, 16, 22, 14, 20, 10]} />
              </div>
              <h2 className={styles.questionText} id="question-text">
                {question.text}
              </h2>
              <div className={styles.sideWave}>
                <WaveBars count={6} active={waveActive} heights={[10, 20, 14, 22, 16, 8]} />
              </div>
            </div>

            <div
              className={styles.portraitWrap}
              onClick={needsAudioUnlock ? () => void unlockAndPlay() : undefined}
              style={needsAudioUnlock ? { cursor: 'pointer' } : undefined}
              role={needsAudioUnlock ? 'button' : undefined}
            >
              <div
                className={`${styles.glow} ${
                  isSpeaking ? styles.glowSpeaking : isEvaluating ? styles.glowEvaluating : ''
                }`}
              />
              <img
                src="/ai-interviewer.png"
                alt="AI interviewer"
                className={styles.portrait}
              />
            </div>

            <div
              className={styles.listenLabel}
              style={{
                color: isEvaluating ? '#d97706' : isSpeaking ? '#4f46e5' : micMuted ? '#dc2626' : '#16a34a',
              }}
            >
              <span className={styles.dot} style={{ background: 'currentColor' }} />
              {needsAudioUnlock
                ? 'Tap here or Repeat question to hear her'
                : statusLabel}
            </div>
            <p className={styles.listenHint}>Speak naturally. Take your time — pauses are okay.</p>

            {silenceCountdown !== null && agentState === 'listening' && (
              <div className={styles.silenceBanner}>Auto-submitting in {silenceCountdown}s (silence detected)...</div>
            )}

            {ttsError && <div className={styles.errorBanner}>{ttsError}</div>}
            {questionError && <div className={styles.errorBanner}>{questionError}</div>}
            {sttError && <div className={styles.errorBanner}>Microphone notice: {sttError}</div>}

            <div className={styles.bottomWave}>
              <WaveBars
                count={28}
                active={waveActive}
                heights={[6, 10, 14, 18, 12, 22, 16, 10, 20, 8, 14, 24, 12, 18, 8, 16, 22, 10, 14, 20, 8, 16, 12, 22, 10, 18, 8, 12]}
              />
            </div>
          </div>

          <aside className={styles.transcript}>
            <button
              type="button"
              className={styles.transcriptHeader}
              onClick={() => setShowTranscript(!showTranscript)}
            >
              <span className={styles.transcriptTitle}>
                <AudioLines size={16} color="#16a34a" />
                Live Transcript
              </span>
              {showTranscript ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
            </button>
            {showTranscript && (
              <div className={`${styles.transcriptBody} ${liveText ? styles.transcriptLive : ''}`}>
                {liveText ? (
                  <>
                    {liveAnswer}{' '}
                    {interimTranscript && <em>{interimTranscript}</em>}
                    {liveAnswer && !interimTranscript ? '...' : ''}
                  </>
                ) : isSpeaking ? (
                  'The interviewer is asking the question...'
                ) : (
                  'Your answer will appear here as you speak...'
                )}
                {liveText && (
                  <div style={{ marginTop: '0.7rem' }}>
                    <button
                      type="button"
                      onClick={handlePerformSubmit}
                      disabled={!liveAnswer.trim() || submitting}
                      className={styles.ghostBtn}
                      style={{ color: '#16a34a', padding: 0, fontSize: '0.75rem' }}
                    >
                      <Send size={12} /> Send answer now
                    </button>
                  </div>
                )}
              </div>
            )}
          </aside>
        </div>

        <div className={styles.controls}>
          <button
            type="button"
            className={styles.ghostBtn}
            onClick={handleRepeatQuestion}
            disabled={submitting}
          >
            <RotateCcw size={16} />
            Repeat question
          </button>

          <div className={styles.micCluster}>
            <button
              type="button"
              className={`${styles.micBtn} ${micMuted ? styles.micBtnMuted : ''}`}
              onClick={toggleMute}
              disabled={submitting}
              aria-label={micMuted ? 'Unmute microphone' : 'Mute microphone'}
            >
              {micMuted ? <MicOff size={24} /> : <Mic size={24} />}
            </button>
            <div className={styles.micMeta}>
              <span className={micMuted ? styles.micOff : styles.micOn}>
                {micMuted ? 'Mic is off' : 'Mic is on'}
              </span>
              <span className={styles.micHint}>{micMuted ? 'Click to unmute' : 'Click to mute'}</span>
            </div>
          </div>

          <div className={styles.settingsWrap}>
            <button
              type="button"
              className={styles.ghostBtn}
              onClick={() => setShowSettings((v) => !v)}
            >
              <Settings size={16} />
              Settings
            </button>
            {showSettings && (
              <div className={styles.settingsMenu}>
                <button type="button" className={styles.settingsItem} onClick={toggleSpeaker}>
                  <span>{speakerMuted ? 'Unmute speaker' : 'Mute speaker'}</span>
                  {speakerMuted ? <VolumeX size={16} /> : <Volume2 size={16} />}
                </button>
              </div>
            )}
          </div>
        </div>

        <div className={styles.footer}>
          <Lock size={12} />
          Your answers are private and secure
        </div>
      </div>

      {showEndConfirm && (
        <div className={styles.confirmOverlay}>
          <div className={styles.confirmCard}>
            <h3 style={{ fontSize: '1.15rem', fontWeight: 800, marginBottom: '0.4rem' }}>End this interview?</h3>
            <p style={{ color: '#6b7280', fontSize: '0.9rem' }}>
              You can submit your current answer, then leave. This cannot be undone.
            </p>
            <div className={styles.confirmActions}>
              <button type="button" className={styles.cancelBtn} onClick={() => setShowEndConfirm(false)}>
                Keep going
              </button>
              <button
                type="button"
                className={styles.dangerBtn}
                onClick={() => {
                  setShowEndConfirm(false);
                  if (liveAnswer.trim()) {
                    handlePerformSubmit().finally(() => onEndInterview());
                  } else {
                    onEndInterview();
                  }
                }}
              >
                End interview
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
