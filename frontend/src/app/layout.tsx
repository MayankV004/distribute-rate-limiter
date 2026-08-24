import type { Metadata } from 'next';
import './globals.css';
import { Navbar } from '@/components/Navbar';
import { ThemeProvider } from '@/context/ThemeContext';

export const metadata: Metadata = {
  title: 'THROTTLE | Distributed Rate Limiter Gateway',
  description:
    'A production-grade stateless Go API Gateway with swappable rate limiting algorithms, atomic Redis Lua script quota evaluation, multi-replica scaling, and real-time observability.',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className="bg-[var(--background)] text-[var(--foreground)] min-h-screen flex flex-col font-sans antialiased transition-colors">
        <ThemeProvider>
          <Navbar />
          <main className="flex-1 w-full max-w-[1800px] mx-auto p-4 sm:p-6">{children}</main>
        </ThemeProvider>
      </body>
    </html>
  );
}
