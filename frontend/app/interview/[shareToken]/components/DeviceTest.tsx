'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Mic,
  MicOff,
  Volume2,
  VolumeX,
  CheckCircle2,
  AlertCircle,
  Play,
  RotateCcw,
  ArrowRight,
  ShieldCheck,
  Radio,
  Sparkles,
} from 'lucide-react';

type DeviceTestProps = {
  candidateName: string;
  onProceed: () => void;
  loading?: boolean;
};

export default function DeviceTest({ candidateName, onProceed, loading = false }: DeviceTestProps) {
  // Mic state
  const [micPermission, setMicPermission] = useState<'prompt' | 'granted' | 'denied'>('prompt');
  const [micActive, setMicActive] = useState(false);
  const [micLevel, setMicLevel] = useState(0);
  const [micTested, setMicTested] = useState(false);
  const [micDevices, setMicDevices] = useState<MediaDeviceInfo[]>([]);
  const [selectedMicId, setSelectedMicId] = useState<string>('');
  const [micError, setMicError] = useState<string | null>(null);

  // Speaker state
  const [isPlayingTestSound, setIsPlayingTestSound] = useState(false);
  const [speakerTested, setSpeakerTested] = useState(false);

  // Confirmation state
  const [userConfirmed, setUserConfirmed] = useState(false);

  // Audio Context & Stream references
  const audioCtxRef = useRef<AudioContext | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const animFrameRef = useRef<number | null>(null);

  // Stop active microphone stream safely
  const stopMicStream = useCallback(() => {
    if (animFrameRef.current) {
      cancelAnimationFrame(animFrameRef.current);
      animFrameRef.current = null;
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }
    if (audioCtxRef.current && audioCtxRef.current.state !== 'closed') {
      audioCtxRef.current.close().catch(() => {});
      audioCtxRef.current = null;
    }
    setMicActive(false);
    setMicLevel(0);
  }, []);

  // Request & start microphone stream
  const startMicStream = useCallback(async (deviceId?: string) => {
    stopMicStream();
    setMicError(null);

    try {
      const constraints: MediaStreamConstraints = {
        audio: deviceId ? { deviceId: { exact: deviceId } } : true,
        video: false,
      };

      const stream = await navigator.mediaDevices.getUserMedia(constraints);
      streamRef.current = stream;
      setMicPermission('granted');
      setMicActive(true);

      // Enumerate audio input devices
      const devices = await navigator.mediaDevices.enumerateDevices();
      const audioInputs = devices.filter((d) => d.kind === 'audioinput');
      setMicDevices(audioInputs);

      if (!selectedMicId && audioInputs.length > 0) {
        setSelectedMicId(audioInputs[0].deviceId);
      }

      // Audio Context setup for volume meter
      const AudioCtx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
      const audioCtx = new AudioCtx();
      audioCtxRef.current = audioCtx;

      const source = audioCtx.createMediaStreamSource(stream);
      const analyser = audioCtx.createAnalyser();
      analyser.fftSize = 256;
      source.connect(analyser);
      analyserRef.current = analyser;

      const dataArray = new Uint8Array(analyser.frequencyBinCount);

      const updateMeter = () => {
        if (!analyserRef.current) return;
        analyserRef.current.getByteFrequencyData(dataArray);

        // Calculate peak amplitude
        let sum = 0;
        for (let i = 0; i < dataArray.length; i++) {
          sum += dataArray[i];
        }
        const average = sum / dataArray.length;
        const normalized = Math.min(100, Math.round((average / 128) * 100));
        setMicLevel(normalized);

        // Automatically pass mic test if user speaks above threshold
        if (normalized > 12) {
          setMicTested(true);
        }

        animFrameRef.current = requestAnimationFrame(updateMeter);
      };

      updateMeter();
    } catch (err: unknown) {
      console.error('Failed to access microphone:', err);
      setMicPermission('denied');
      setMicActive(false);
      if (err instanceof Error) {
        setMicError(err.message || 'Microphone access denied or unavailable.');
      } else {
        setMicError('Microphone access was denied. Please allow microphone permissions in your browser.');
      }
    }
  }, [selectedMicId, stopMicStream]);

  // Request mic access on mount
  useEffect(() => {
    startMicStream();
    return () => {
      stopMicStream();
    };
  }, []);

  // Handle device switch
  const handleMicDeviceChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newId = e.target.value;
    setSelectedMicId(newId);
    startMicStream(newId);
  };

  // Play synthetic test sound for speaker verification
  const playTestSound = () => {
    setIsPlayingTestSound(true);

    try {
      const AudioCtx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
      const ctx = new AudioCtx();

      // Create melodic 3-tone chime sequence (C5 -> E5 -> G5)
      const notes = [523.25, 659.25, 783.99];
      const now = ctx.currentTime;

      notes.forEach((freq, idx) => {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();

        osc.type = 'sine';
        osc.frequency.setValueAtTime(freq, now + idx * 0.25);

        gain.gain.setValueAtTime(0, now + idx * 0.25);
        gain.gain.linearRampToValueAtTime(0.3, now + idx * 0.25 + 0.05);
        gain.gain.exponentialRampToValueAtTime(0.001, now + idx * 0.25 + 0.4);

        osc.connect(gain);
        gain.connect(ctx.destination);

        osc.start(now + idx * 0.25);
        osc.stop(now + idx * 0.25 + 0.45);
      });

      setTimeout(() => {
        setIsPlayingTestSound(false);
        setSpeakerTested(true);
        ctx.close().catch(() => {});
      }, 1200);
    } catch (err) {
      console.error('Error playing synth test sound:', err);
      setIsPlayingTestSound(false);
      setSpeakerTested(true);
    }
  };

  const isReadyToProceed = (micTested || micActive) && speakerTested && userConfirmed;

  return (
    <div className="container" style={{ maxWidth: '640px' }}>
      <div className="glass-card">
        {/* Header */}
        <div style={{ textAlign: 'center', marginBottom: '1.75rem' }}>
          <span className="badge" style={{ background: '#eef2ff', color: 'var(--primary)', borderColor: '#c7d2fe' }}>
            Pre-Interview Hardware Check
          </span>
          <h2 style={{ fontSize: '1.5rem', fontWeight: 700, marginTop: '0.75rem', color: 'var(--text-main)' }}>
            Audio & Microphone Setup
          </h2>
          <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', marginTop: '0.25rem' }}>
            Welcome {candidateName}! Please confirm your microphone and speaker working state before beginning the interview.
          </p>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem', marginBottom: '2rem' }}>
          {/* MICROPHONE TEST CARD */}
          <div
            style={{
              background: '#f8fafc',
              border: micTested ? '1px solid #10b981' : '1px solid var(--border)',
              borderRadius: '12px',
              padding: '1.25rem',
              transition: 'all 0.3s ease',
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}>
                <div
                  style={{
                    background: micActive ? '#ecfdf5' : '#fef2f2',
                    padding: '0.5rem',
                    borderRadius: '8px',
                  }}
                >
                  {micActive ? <Mic size={20} color="#059669" /> : <MicOff size={20} color="#dc2626" />}
                </div>
                <div>
                  <h4 style={{ fontWeight: 600, fontSize: '0.95rem', color: 'var(--text-main)' }}>
                    1. Microphone Input Check
                  </h4>
                  <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                    {micActive ? 'Microphone active — speak to test level' : 'Microphone inactive'}
                  </span>
                </div>
              </div>

              {micTested ? (
                <span
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '0.3rem',
                    background: '#ecfdf5',
                    color: '#047857',
                    border: '1px solid #a7f3d0',
                    fontSize: '0.75rem',
                    fontWeight: 700,
                    padding: '0.25rem 0.65rem',
                    borderRadius: '9999px',
                  }}
                >
                  <CheckCircle2 size={14} /> Verified
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => startMicStream(selectedMicId)}
                  style={{
                    fontSize: '0.775rem',
                    padding: '0.3rem 0.65rem',
                    borderRadius: '6px',
                    border: '1px solid var(--border)',
                    background: '#ffffff',
                    cursor: 'pointer',
                    color: 'var(--text-main)',
                  }}
                >
                  Retry Mic Access
                </button>
              )}
            </div>

            {/* Mic Device Selector */}
            {micDevices.length > 1 && (
              <div style={{ marginBottom: '1rem' }}>
                <label className="form-label" style={{ fontSize: '0.8rem', marginBottom: '0.25rem', display: 'block' }}>
                  Select Microphone:
                </label>
                <select
                  value={selectedMicId}
                  onChange={handleMicDeviceChange}
                  className="form-input"
                  style={{ fontSize: '0.85rem', padding: '0.4rem 0.75rem', width: '100%' }}
                >
                  {micDevices.map((device, idx) => (
                    <option key={device.deviceId || idx} value={device.deviceId}>
                      {device.label || `Microphone ${idx + 1}`}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {/* Live Volume Meter Bar */}
            <div style={{ marginTop: '0.5rem' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.35rem' }}>
                <span>Input Level Meter</span>
                <span style={{ fontWeight: 600, color: micLevel > 12 ? '#059669' : 'var(--text-muted)' }}>
                  {micLevel}% {micLevel > 12 ? '(Voice Detected!)' : ''}
                </span>
              </div>

              {/* Progress bar container */}
              <div
                style={{
                  height: '14px',
                  width: '100%',
                  background: '#e2e8f0',
                  borderRadius: '7px',
                  overflow: 'hidden',
                  position: 'relative',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    width: `${micLevel}%`,
                    background: micLevel > 40 ? 'linear-gradient(90deg, #10b981 0%, #059669 100%)' : micLevel > 10 ? '#34d399' : '#94a3b8',
                    transition: 'width 0.08s ease-out',
                    borderRadius: '7px',
                  }}
                />
              </div>
              <p style={{ fontSize: '0.775rem', color: 'var(--text-muted)', marginTop: '0.5rem' }}>
                💡 Say something out loud like <em>&ldquo;Hello, audio test 1, 2, 3&rdquo;</em> to confirm your microphone reacts.
              </p>
            </div>

            {micError && (
              <div
                style={{
                  marginTop: '0.75rem',
                  background: '#fef2f2',
                  border: '1px solid #fecaca',
                  color: '#991b1b',
                  padding: '0.5rem 0.75rem',
                  borderRadius: '6px',
                  fontSize: '0.8rem',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.4rem',
                }}
              >
                <AlertCircle size={16} /> {micError}
              </div>
            )}
          </div>

          {/* SPEAKER / AUDIO OUTPUT TEST CARD */}
          <div
            style={{
              background: '#f8fafc',
              border: speakerTested ? '1px solid #10b981' : '1px solid var(--border)',
              borderRadius: '12px',
              padding: '1.25rem',
              transition: 'all 0.3s ease',
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}>
                <div style={{ background: '#eef2ff', padding: '0.5rem', borderRadius: '8px' }}>
                  <Volume2 size={20} color="var(--primary)" />
                </div>
                <div>
                  <h4 style={{ fontWeight: 600, fontSize: '0.95rem', color: 'var(--text-main)' }}>
                    2. Speaker / Audio Output Check
                  </h4>
                  <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                    Test your speakers or headphones to ensure you can hear the AI interviewer.
                  </span>
                </div>
              </div>

              {speakerTested && (
                <span
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '0.3rem',
                    background: '#ecfdf5',
                    color: '#047857',
                    border: '1px solid #a7f3d0',
                    fontSize: '0.75rem',
                    fontWeight: 700,
                    padding: '0.25rem 0.65rem',
                    borderRadius: '9999px',
                  }}
                >
                  <CheckCircle2 size={14} /> Verified
                </span>
              )}
            </div>

            <div style={{ marginTop: '1rem', display: 'flex', gap: '0.75rem', alignItems: 'center', flexWrap: 'wrap' }}>
              <button
                type="button"
                onClick={playTestSound}
                disabled={isPlayingTestSound}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '0.4rem',
                  padding: '0.55rem 1rem',
                  borderRadius: '8px',
                  background: isPlayingTestSound ? '#e0e7ff' : '#4f46e5',
                  color: '#ffffff',
                  fontSize: '0.85rem',
                  fontWeight: 600,
                  border: 'none',
                  cursor: 'pointer',
                  transition: 'all 0.2s ease',
                }}
              >
                {isPlayingTestSound ? (
                  <>
                    <Radio size={16} className="pulse-icon" /> Playing Tone...
                  </>
                ) : (
                  <>
                    <Play size={16} /> Play Test Sound Tone
                  </>
                )}
              </button>

              {!speakerTested && (
                <button
                  type="button"
                  onClick={() => setSpeakerTested(true)}
                  style={{
                    fontSize: '0.825rem',
                    padding: '0.55rem 0.85rem',
                    borderRadius: '8px',
                    border: '1px solid #cbd5e1',
                    background: '#ffffff',
                    color: 'var(--text-main)',
                    cursor: 'pointer',
                    fontWeight: 500,
                  }}
                >
                  I heard it clearly
                </button>
              )}
            </div>
          </div>
        </div>

        {/* CONFIRMATION CHECKBOX */}
        <div
          style={{
            background: '#ffffff',
            border: '1px solid var(--border)',
            borderRadius: '10px',
            padding: '1rem 1.25rem',
            marginBottom: '1.5rem',
            display: 'flex',
            alignItems: 'center',
            gap: '0.75rem',
          }}
        >
          <input
            type="checkbox"
            id="chk-audio-confirm"
            checked={userConfirmed}
            onChange={(e) => setUserConfirmed(e.target.checked)}
            style={{ width: '18px', height: '18px', cursor: 'pointer', accentColor: 'var(--primary)' }}
          />
          <label htmlFor="chk-audio-confirm" style={{ fontSize: '0.875rem', fontWeight: 500, color: 'var(--text-main)', cursor: 'pointer' }}>
            I confirm that my microphone and speakers are working properly and my environment is quiet.
          </label>
        </div>

        {/* PROCEED BUTTON */}
        <button
          type="button"
          onClick={onProceed}
          disabled={!isReadyToProceed || loading}
          className="btn-primary"
          id="btn-proceed-to-interview"
          style={{ padding: '0.85rem 1.5rem', fontSize: '1rem' }}
        >
          {loading ? (
            'Initializing Interview Session...'
          ) : (
            <>
              Confirm & Start Interview Session <ArrowRight size={18} />
            </>
          )}
        </button>
      </div>
    </div>
  );
}
