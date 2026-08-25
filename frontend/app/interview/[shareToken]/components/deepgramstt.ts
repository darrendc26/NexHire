'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

type DeepgramResult = {
    type?: string;
    channel?: {
        alternatives?: Array<{
            transcript?: string;
        }>;
    };
    is_final?: boolean;
    speech_final?: boolean;
};

type UseDeepgramSTTOptions = {
    token: string;
    onFinalTranscript?: (transcript: string) => void;
};

export function useDeepgramSTT({
    token,
    onFinalTranscript,
}: UseDeepgramSTTOptions) {
    const [listening, setListening] = useState(false);
    const [transcript, setTranscript] = useState('');
    const [interimTranscript, setInterimTranscript] = useState('');
    const [error, setError] = useState<string | null>(null);

    const socketRef = useRef<WebSocket | null>(null);
    const rawStreamRef = useRef<MediaStream | null>(null);
    const processedStreamRef = useRef<MediaStream | null>(null);
    const recorderRef = useRef<MediaRecorder | null>(null);
    const audioContextRef = useRef<AudioContext | null>(null);

    const start = useCallback(async () => {
        if (listening) return;

        try {
            setError(null);
            setTranscript('');
            setInterimTranscript('');

            if (!token) {
                throw new Error('Deepgram token is missing');
            }

            // High-fidelity microphone constraints with ideal sample rates and auto-gain
            const rawStream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    echoCancellation: { ideal: true },
                    noiseSuppression: { ideal: true },
                    autoGainControl: { ideal: true },
                    channelCount: { ideal: 1 },
                    sampleRate: { ideal: 48000, min: 16000 },
                },
            });

            rawStreamRef.current = rawStream;

            // Build Web Audio processing chain to boost quiet headset/headphone mic input
            let recordStream = rawStream;
            try {
                const AudioCtx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
                if (AudioCtx) {
                    const ctx = new AudioCtx();
                    audioContextRef.current = ctx;

                    const source = ctx.createMediaStreamSource(rawStream);

                    // Dynamic compressor to smooth out speech dynamics
                    const compressor = ctx.createDynamicsCompressor();
                    compressor.threshold.setValueAtTime(-24, ctx.currentTime);
                    compressor.knee.setValueAtTime(30, ctx.currentTime);
                    compressor.ratio.setValueAtTime(12, ctx.currentTime);
                    compressor.attack.setValueAtTime(0.003, ctx.currentTime);
                    compressor.release.setValueAtTime(0.25, ctx.currentTime);

                    // 1.4x (40%) Gain Boost to ensure headphone mics pick up clearly
                    const gainNode = ctx.createGain();
                    gainNode.gain.setValueAtTime(1.4, ctx.currentTime);

                    const destination = ctx.createMediaStreamDestination();
                    source.connect(compressor);
                    compressor.connect(gainNode);
                    gainNode.connect(destination);

                    recordStream = destination.stream;
                }
            } catch (audioErr) {
                console.warn('Web Audio gain processing fallback to raw stream:', audioErr);
            }

            processedStreamRef.current = recordStream;

            const params = new URLSearchParams({
                model: 'nova-2',
                language: 'en-US',
                smart_format: 'true',
                interim_results: 'true',
                punctuate: 'true',
                filler_words: 'false',
                vad_events: 'true',
                endpointing: '1200',
                utterance_end_ms: '2000',
            });

            const authSubprotocol = token.startsWith('eyJ') ? 'bearer' : 'token';

            const socket = new WebSocket(
                `wss://api.deepgram.com/v1/listen?${params.toString()}`,
                [authSubprotocol, token],
            );

            socketRef.current = socket;

            socket.onopen = () => {
                setListening(true);

                let mimeType = '';
                if (MediaRecorder.isTypeSupported('audio/webm;codecs=opus')) {
                    mimeType = 'audio/webm;codecs=opus';
                } else if (MediaRecorder.isTypeSupported('audio/webm')) {
                    mimeType = 'audio/webm';
                } else {
                    throw new Error('Browser does not support a compatible audio format');
                }

                const recorder = new MediaRecorder(recordStream, {
                    mimeType,
                    audioBitsPerSecond: 128000,
                });

                recorderRef.current = recorder;

                recorder.ondataavailable = async (event) => {
                    if (
                        event.data.size > 0 &&
                        socket.readyState === WebSocket.OPEN
                    ) {
                        socket.send(event.data);
                    }
                };

                recorder.onerror = () => {
                    setError('Microphone recording failed');
                };

                recorder.start(200);
            };

            socket.onmessage = (event) => {
                try {
                    const data: DeepgramResult = JSON.parse(event.data);

                    if (data.type !== 'Results') {
                        return;
                    }

                    const alternative = data.channel?.alternatives?.[0];

                    if (!alternative?.transcript) {
                        return;
                    }

                    const text = alternative.transcript.trim();

                    if (!text) {
                        return;
                    }

                    if (data.is_final) {
                        setTranscript((previous) => {
                            const combined = previous
                                ? `${previous} ${text}`
                                : text;

                            return combined.trim();
                        });

                        setInterimTranscript('');

                        // speech_final means Deepgram considers the utterance complete.
                        if (data.speech_final) {
                            onFinalTranscript?.(text);
                        }
                    } else {
                        setInterimTranscript(text);
                    }
                } catch (err) {
                    console.error('Failed to parse Deepgram message:', err);
                }
            };

            socket.onerror = () => {
                setError('Deepgram connection failed');
            };

            socket.onclose = () => {
                setListening(false);
            };
        } catch (err) {
            console.error('Failed to start STT:', err);

            setError(
                err instanceof Error
                    ? err.message
                    : 'Failed to start microphone',
            );

            rawStreamRef.current?.getTracks().forEach((track) => track.stop());
            rawStreamRef.current = null;
            processedStreamRef.current?.getTracks().forEach((track) => track.stop());
            processedStreamRef.current = null;
            if (audioContextRef.current && audioContextRef.current.state !== 'closed') {
                audioContextRef.current.close().catch(() => {});
                audioContextRef.current = null;
            }
        }
    }, [token, listening, onFinalTranscript]);

    const stop = useCallback(() => {
        recorderRef.current?.stop();
        recorderRef.current = null;

        if (socketRef.current?.readyState === WebSocket.OPEN) {
            // Tell Deepgram we're finished sending audio.
            socketRef.current.send(
                JSON.stringify({
                    type: 'CloseStream',
                }),
            );

            socketRef.current.close();
        }

        socketRef.current = null;

        rawStreamRef.current?.getTracks().forEach((track) => track.stop());
        rawStreamRef.current = null;
        processedStreamRef.current?.getTracks().forEach((track) => track.stop());
        processedStreamRef.current = null;

        if (audioContextRef.current && audioContextRef.current.state !== 'closed') {
            audioContextRef.current.close().catch(() => {});
            audioContextRef.current = null;
        }

        setListening(false);
    }, []);

    useEffect(() => {
        if (token && error === 'Deepgram token is missing') {
            setError(null);
        }
    }, [token, error]);

    useEffect(() => {
        return () => {
            recorderRef.current?.stop();

            if (socketRef.current) {
                socketRef.current.close();
            }

            rawStreamRef.current?.getTracks().forEach((track) => track.stop());
            processedStreamRef.current?.getTracks().forEach((track) => track.stop());
            if (audioContextRef.current && audioContextRef.current.state !== 'closed') {
                audioContextRef.current.close().catch(() => {});
            }
        };
    }, []);

    return {
        listening,
        transcript,
        interimTranscript,
        error,
        start,
        stop,
    };
}