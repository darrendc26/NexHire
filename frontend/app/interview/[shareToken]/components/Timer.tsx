'use client';

import React, { useEffect, useState } from 'react';
import { Clock } from 'lucide-react';

type TimerProps = {
  initialSeconds: number;
  onTimeUp?: () => void;
};

export default function Timer({ initialSeconds, onTimeUp }: TimerProps) {
  const [seconds, setSeconds] = useState(initialSeconds);

  useEffect(() => {
    setSeconds(initialSeconds);
  }, [initialSeconds]);

  useEffect(() => {
    if (seconds <= 0) {
      if (onTimeUp) {
        onTimeUp();
      }
      return;
    }

    const interval = setInterval(() => {
      setSeconds((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          if (onTimeUp) {
            onTimeUp();
          }
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [seconds, onTimeUp]);

  const formatTime = (totalSecs: number) => {
    if (totalSecs <= 0) return "Time's up";
    const mins = Math.floor(totalSecs / 60);
    const secs = totalSecs % 60;
    return `${mins}:${secs < 10 ? '0' : ''}${secs} left`;
  };

  const isUrgent = seconds <= 120 && seconds > 0;

  return (
    <div className={`timer-badge ${isUrgent ? 'urgent' : ''}`} id="timer-display">
      <Clock size={16} />
      <span>{formatTime(seconds)}</span>
    </div>
  );
}
