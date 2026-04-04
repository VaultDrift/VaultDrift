import { useState } from 'react';
import { Cloud, Loader2, Shield } from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import { cn } from '@/lib/utils';

export function LoginPage() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [totpSession, setTotpSession] = useState<string | null>(null);
  const [error, setError] = useState('');
  const { login, verifyTotp, isLoading } = useAuthStore();

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const session = await login(username, password);
      if (session) {
        setTotpSession(session);
      }
    } catch {
      setError('Invalid username or password');
    }
  };

  const handleTotpSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!totpSession) return;
    try {
      await verifyTotp(totpSession, totpCode);
    } catch {
      setError('Invalid verification code');
      setTotpCode('');
    }
  };

  const handleTotpCodeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value.replace(/\D/g, '').slice(0, 6);
    setTotpCode(value);
  };

  // TOTP verification mode
  if (totpSession) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background to-muted p-4">
        <div className="w-full max-w-md">
          <div className="bg-card rounded-2xl shadow-lg border p-8">
            <div className="flex flex-col items-center mb-8">
              <div className="w-16 h-16 bg-primary rounded-2xl flex items-center justify-center mb-4">
                <Shield className="w-10 h-10 text-primary-foreground" />
              </div>
              <h1 className="text-2xl font-bold">Two-Factor Authentication</h1>
              <p className="text-muted-foreground">Enter the code from your authenticator app</p>
            </div>

            <form onSubmit={handleTotpSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Verification Code</label>
                <input
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  value={totpCode}
                  onChange={handleTotpCodeChange}
                  className="w-full px-4 py-3 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary text-center text-2xl tracking-[0.5em] font-mono"
                  placeholder="000000"
                  autoFocus
                  required
                />
              </div>

              {error && (
                <div className="text-sm text-destructive bg-destructive/10 p-3 rounded-lg">
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={isLoading || totpCode.length !== 6}
                className={cn(
                  'w-full py-2 px-4 rounded-lg font-medium transition-colors',
                  isLoading || totpCode.length !== 6
                    ? 'bg-primary/50 cursor-not-allowed'
                    : 'bg-primary hover:bg-primary/90 text-primary-foreground'
                )}
              >
                {isLoading ? (
                  <span className="flex items-center justify-center gap-2">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Verifying...
                  </span>
                ) : (
                  'Verify'
                )}
              </button>

              <button
                type="button"
                onClick={() => {
                  setTotpSession(null);
                  setTotpCode('');
                  setError('');
                }}
                className="w-full py-2 px-4 rounded-lg font-medium text-muted-foreground hover:text-foreground transition-colors"
              >
                Back to login
              </button>
            </form>
          </div>
        </div>
      </div>
    );
  }

  // Normal login mode
  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background to-muted p-4">
      <div className="w-full max-w-md">
        <div className="bg-card rounded-2xl shadow-lg border p-8">
          {/* Logo */}
          <div className="flex flex-col items-center mb-8">
            <div className="w-16 h-16 bg-primary rounded-2xl flex items-center justify-center mb-4">
              <Cloud className="w-10 h-10 text-primary-foreground" />
            </div>
            <h1 className="text-2xl font-bold">VaultDrift</h1>
            <p className="text-muted-foreground">Secure distributed storage</p>
          </div>

          {/* Form */}
          <form onSubmit={handleLoginSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">Username</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="Enter your username"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-2 rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="Enter your password"
                required
              />
            </div>

            {error && (
              <div className="text-sm text-destructive bg-destructive/10 p-3 rounded-lg">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={isLoading}
              className={cn(
                'w-full py-2 px-4 rounded-lg font-medium transition-colors',
                isLoading
                  ? 'bg-primary/50 cursor-not-allowed'
                  : 'bg-primary hover:bg-primary/90 text-primary-foreground'
              )}
            >
              {isLoading ? (
                <span className="flex items-center justify-center gap-2">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Signing in...
                </span>
              ) : (
                'Sign In'
              )}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
