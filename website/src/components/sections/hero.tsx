import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ArrowRight, Download, Shield, Lock, Zap } from 'lucide-react'

export function Hero() {
  return (
    <section className="relative min-h-screen flex items-center justify-center pt-20 overflow-hidden">
      {/* Background Grid */}
      <div className="absolute inset-0 bg-grid opacity-30" />

      {/* Gradient Orbs */}
      <div className="absolute top-1/4 left-1/4 w-[500px] h-[500px] bg-blue-500/20 rounded-full blur-[120px] animate-pulse-glow" />
      <div className="absolute bottom-1/4 right-1/4 w-[400px] h-[400px] bg-cyan-500/20 rounded-full blur-[100px] animate-pulse-glow" style={{ animationDelay: '2s' }} />

      <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid lg:grid-cols-2 gap-12 items-center">
          {/* Left Content */}
          <div className="text-center lg:text-left">
            <Badge className="mb-6 bg-blue-500/10 text-blue-400 border-blue-500/20 hover:bg-blue-500/20">
              <Shield className="w-3 h-3 mr-1" />
              Self-Hosted & Private
            </Badge>

            <h1 className="text-5xl sm:text-6xl lg:text-7xl font-bold tracking-tight mb-6">
              <span className="block">Secure Cloud</span>
              <span className="block text-gradient mt-2">Storage</span>
            </h1>

            <p className="text-lg sm:text-xl text-zinc-400 mb-8 max-w-xl mx-auto lg:mx-0">
              End-to-end encrypted, self-hosted file storage. Your data stays yours.
              No tracking, no mining, complete privacy.
            </p>

            <div className="flex flex-col sm:flex-row items-center justify-center lg:justify-start gap-4 mb-8">
              <Button size="lg" className="bg-gradient-to-r from-blue-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 border-0 glow-strong h-12 px-8" asChild>
                <a href="https://github.com/vaultdrift/vaultdrift/releases" target="_blank" rel="noopener noreferrer">
                  <Download className="w-5 h-5 mr-2" />
                  Download Free
                </a>
              </Button>
              <Button size="lg" variant="outline" className="h-12 px-8 border-white/20 hover:bg-white/5" asChild>
                <a href="https://github.com/vaultdrift/vaultdrift" target="_blank" rel="noopener noreferrer">
                  <Github className="w-5 h-5 mr-2" />
                  View on GitHub
                  <ArrowRight className="w-4 h-4 ml-2" />
                </a>
              </Button>
            </div>

            <div className="flex flex-wrap items-center justify-center lg:justify-start gap-4">
              <div className="flex items-center gap-2 text-sm text-zinc-500">
                <Lock className="w-4 h-4 text-blue-400" />
                <span>End-to-End Encrypted</span>
              </div>
              <div className="flex items-center gap-2 text-sm text-zinc-500">
                <Zap className="w-4 h-4 text-cyan-400" />
                <span>Lightning Fast</span>
              </div>
            </div>
          </div>

          {/* Right Content - Code Snippet */}
          <div className="relative">
            <div className="absolute inset-0 bg-gradient-to-r from-blue-500/20 to-cyan-500/20 rounded-2xl blur-2xl" />
            <div className="relative bg-zinc-900/90 border border-white/10 rounded-2xl p-6 glow">
              <div className="flex items-center gap-2 mb-4 pb-4 border-b border-white/10">
                <div className="w-3 h-3 rounded-full bg-red-500" />
                <div className="w-3 h-3 rounded-full bg-yellow-500" />
                <div className="w-3 h-3 rounded-full bg-green-500" />
                <span className="ml-4 text-sm text-zinc-500 font-mono">terminal</span>
              </div>
              <pre className="text-sm font-mono overflow-x-auto">
                <code className="text-zinc-300">
                  <span className="text-zinc-500"># Install VaultDrift</span>{"\n"}
                  <span className="text-cyan-400">$</span> wget vaultdrift.com/install.sh{"\n"}
                  <span className="text-cyan-400">$</span> chmod +x install.sh && ./install.sh{"\n"}
                  {"\n"}
                  <span className="text-zinc-500"># Start the server</span>{"\n"}
                  <span className="text-cyan-400">$</span> vaultdrift server{"\n"}
                  <span className="text-green-400">✓</span> Server running on http://localhost:8080{"\n"}
                  {"\n"}
                  <span className="text-zinc-500"># Upload a file</span>{"\n"}
                  <span className="text-cyan-400">$</span> vaultdrift upload ./document.pdf{"\n"}
                  <span className="text-green-400">✓</span> Uploaded: document.pdf (2.4 MB)
                </code>
              </pre>
            </div>

            {/* Floating badges */}
            <div className="absolute -top-4 -right-4 bg-zinc-900 border border-white/10 rounded-lg p-3 animate-float">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                <span className="text-xs text-zinc-300">Connected</span>
              </div>
            </div>

            <div className="absolute -bottom-4 -left-4 bg-zinc-900 border border-white/10 rounded-lg p-3 animate-float" style={{ animationDelay: '1s' }}>
              <div className="text-xs text-zinc-400">Storage Used</div>
              <div className="text-sm font-bold">2.4 TB / 10 TB</div>
              <div className="w-24 h-1 bg-zinc-800 rounded-full mt-1">
                <div className="w-16 h-1 bg-gradient-to-r from-blue-500 to-cyan-400 rounded-full" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

// GitHub icon component
function Github({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.2-6.085 8.2-11.386 0-6.627-5.373-12-12-12z"/>
    </svg>
  )
}
