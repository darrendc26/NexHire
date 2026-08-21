import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'NexHire Candidate Interview',
  description: 'AI-Powered Hiring Candidate Portal',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
