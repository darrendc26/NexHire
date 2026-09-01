'use client';

import React, { useEffect, useRef, useState } from 'react';
import { AlertCircle, ArrowLeft, ShieldCheck } from 'lucide-react';

type OTPVerifyProps = {
  email: string;
  onVerify: (otp: string) => Promise<void>;
  onResend: () => Promise<void>;
  onChangeEmail: () => void;
  verifying: boolean;
  error?: string | null;
};

const OTP_LENGTH = 6;
const RESEND_SECONDS = 60;

export default function OTPVerify({
  email,
  onVerify,
  onResend,
  onChangeEmail,
  verifying,
  error,
}: OTPVerifyProps) {
  const [digits, setDigits] = useState<string[]>(Array(OTP_LENGTH).fill(''));
  const [localError, setLocalError] = useState<string | null>(null);
  const [resendSeconds, setResendSeconds] = useState(RESEND_SECONDS);
  const [resending, setResending] = useState(false);
  const [resentNotice, setResentNotice] = useState(false);
  const inputsRef = useRef<Array<HTMLInputElement | null>>([]);
  const submitLock = useRef(false);

  useEffect(() => {
    inputsRef.current[0]?.focus();
  }, []);

  useEffect(() => {
    if (!verifying) {
      submitLock.current = false;
    }
  }, [verifying]);

  useEffect(() => {
    if (resendSeconds <= 0) return;
    const timer = window.setTimeout(() => setResendSeconds((s) => s - 1), 1000);
    return () => window.clearTimeout(timer);
  }, [resendSeconds]);

  const otpValue = digits.join('');

  const focusIndex = (index: number) => {
    const next = Math.max(0, Math.min(OTP_LENGTH - 1, index));
    inputsRef.current[next]?.focus();
    inputsRef.current[next]?.select();
  };

  const applyDigits = (next: string[]) => {
    setDigits(next);
    setLocalError(null);
    if (next.every((d) => d !== '') && !verifying) {
      void submitOtp(next.join(''));
    }
  };

  const handleChange = (index: number, raw: string) => {
    const value = raw.replace(/\D/g, '');
    if (!value) {
      const next = [...digits];
      next[index] = '';
      setDigits(next);
      return;
    }

    if (value.length > 1) {
      const next = [...digits];
      const chars = value.slice(0, OTP_LENGTH - index).split('');
      chars.forEach((char, offset) => {
        next[index + offset] = char;
      });
      applyDigits(next);
      focusIndex(index + chars.length);
      return;
    }

    const next = [...digits];
    next[index] = value;
    applyDigits(next);
    if (index < OTP_LENGTH - 1) {
      focusIndex(index + 1);
    }
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace' && !digits[index] && index > 0) {
      e.preventDefault();
      const next = [...digits];
      next[index - 1] = '';
      setDigits(next);
      focusIndex(index - 1);
    }

    if (e.key === 'ArrowLeft') {
      e.preventDefault();
      focusIndex(index - 1);
    }

    if (e.key === 'ArrowRight') {
      e.preventDefault();
      focusIndex(index + 1);
    }
  };

  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault();
    const pasted = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, OTP_LENGTH);
    if (!pasted) return;
    const next = Array(OTP_LENGTH).fill('');
    pasted.split('').forEach((char, i) => {
      next[i] = char;
    });
    applyDigits(next);
    focusIndex(pasted.length >= OTP_LENGTH ? OTP_LENGTH - 1 : pasted.length);
  };

  const submitOtp = async (otp: string) => {
    if (otp.length !== OTP_LENGTH) {
      setLocalError('Enter the 6-digit code sent to your email');
      return;
    }
    if (submitLock.current) return;

    submitLock.current = true;
    setLocalError(null);
    try {
      await onVerify(otp);
    } catch (err: unknown) {
      submitLock.current = false;
      if (err instanceof Error) {
        setLocalError(err.message);
      } else {
        setLocalError('Could not verify the code. Please try again.');
      }
      setDigits(Array(OTP_LENGTH).fill(''));
      focusIndex(0);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await submitOtp(otpValue);
  };

  const handleResend = async () => {
    if (resendSeconds > 0 || resending) return;
    setResending(true);
    setLocalError(null);
    try {
      await onResend();
      setDigits(Array(OTP_LENGTH).fill(''));
      setResendSeconds(RESEND_SECONDS);
      setResentNotice(true);
      focusIndex(0);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setLocalError(err.message);
      } else {
        setLocalError('Could not resend the code. Please try again.');
      }
    } finally {
      setResending(false);
    }
  };

  const displayError = localError || error;

  return (
    <div className="container" style={{ maxWidth: '500px' }}>
      <div className="glass-card">
        <h2
          style={{
            fontSize: '1.4rem',
            fontWeight: 700,
            marginBottom: '0.4rem',
            textAlign: 'center',
            color: 'var(--text-main)',
          }}
        >
          Verify your email
        </h2>
        <p
          style={{
            color: 'var(--text-muted)',
            fontSize: '0.9rem',
            marginBottom: '1.75rem',
            textAlign: 'center',
            lineHeight: 1.5,
          }}
        >
          Enter the 6-digit code we sent to <strong style={{ color: 'var(--text-main)' }}>{email}</strong>.
          It expires in 5 minutes.
        </p>

        {resentNotice && !displayError && (
          <div
            style={{
              background: '#ecfdf5',
              border: '1px solid #a7f3d0',
              color: '#047857',
              padding: '0.75rem 1rem',
              borderRadius: '8px',
              fontSize: '0.875rem',
              marginBottom: '1.25rem',
              textAlign: 'center',
            }}
          >
            A new code was sent to your email.
          </div>
        )}

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
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              flexWrap: 'wrap',
              gap: '0.5rem',
              marginBottom: '1.5rem',
            }}
          >
            {digits.map((digit, index) => (
              <input
                key={index}
                ref={(el) => {
                  inputsRef.current[index] = el;
                }}
                id={index === 0 ? 'otp-digit-0' : undefined}
                type="text"
                inputMode="numeric"
                autoComplete={index === 0 ? 'one-time-code' : 'off'}
                pattern="[0-9]*"
                maxLength={OTP_LENGTH}
                value={digit}
                disabled={verifying}
                onChange={(e) => handleChange(index, e.target.value)}
                onKeyDown={(e) => handleKeyDown(index, e)}
                onPaste={handlePaste}
                aria-label={`Digit ${index + 1} of verification code`}
                className="form-input"
                style={{
                  width: '2.75rem',
                  height: '3.25rem',
                  padding: 0,
                  textAlign: 'center',
                  fontSize: '1.25rem',
                  fontWeight: 700,
                  letterSpacing: 0,
                }}
              />
            ))}
          </div>

          <button type="submit" className="btn-primary" disabled={verifying || otpValue.length !== OTP_LENGTH}>
            {verifying ? (
              'Verifying...'
            ) : (
              <>
                Verify email <ShieldCheck size={18} />
              </>
            )}
          </button>
        </form>

        <div style={{ marginTop: '1.25rem', textAlign: 'center' }}>
          <button
            type="button"
            onClick={handleResend}
            disabled={resendSeconds > 0 || resending || verifying}
            style={{
              background: 'none',
              border: 'none',
              color: resendSeconds > 0 ? 'var(--text-dim)' : 'var(--primary)',
              fontWeight: 600,
              fontSize: '0.875rem',
              cursor: resendSeconds > 0 ? 'not-allowed' : 'pointer',
            }}
          >
            {resending
              ? 'Sending a new code...'
              : resendSeconds > 0
                ? `Resend code in ${resendSeconds}s`
                : 'Resend code'}
          </button>
        </div>

        <button
          type="button"
          onClick={onChangeEmail}
          disabled={verifying}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '0.4rem',
            width: '100%',
            marginTop: '0.75rem',
            background: 'none',
            border: 'none',
            color: 'var(--text-muted)',
            fontSize: '0.85rem',
            cursor: 'pointer',
          }}
        >
          <ArrowLeft size={14} /> Use a different email
        </button>
      </div>
    </div>
  );
}
