'use client';

let playbackCtx: AudioContext | null = null;
let ttsSource: AudioBufferSourceNode | null = null;

function getAudioContextConstructor(): typeof AudioContext | null {
  if (typeof window === 'undefined') return null;
  return (
    window.AudioContext ||
    (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext ||
    null
  );
}

export async function unlockPlayback(): Promise<AudioContext | null> {
  const Ctor = getAudioContextConstructor();
  if (!Ctor) return null;

  if (!playbackCtx || playbackCtx.state === 'closed') {
    playbackCtx = new Ctor();
  }

  if (playbackCtx.state === 'suspended') {
    await playbackCtx.resume().catch(() => {});
  }

  return playbackCtx;
}

export function stopTtsPlayback() {
  if (ttsSource) {
    try {
      ttsSource.onended = null;
      ttsSource.stop();
    } catch {
      // Already stopped.
    }
    ttsSource = null;
  }
}

export async function playTtsBlob(blob: Blob): Promise<void> {
  const ctx = await unlockPlayback();
  if (!ctx) {
    throw new Error('Web Audio is not available in this browser.');
  }

  const raw = await blob.arrayBuffer();
  const decoded = await ctx.decodeAudioData(raw.slice(0));

  stopTtsPlayback();

  await new Promise<void>((resolve, reject) => {
    const src = ctx.createBufferSource();
    ttsSource = src;
    src.buffer = decoded;
    src.connect(ctx.destination);
    src.onended = () => {
      if (ttsSource === src) ttsSource = null;
      resolve();
    };
    try {
      src.start();
    } catch (err) {
      ttsSource = null;
      reject(err);
    }
  });
}
