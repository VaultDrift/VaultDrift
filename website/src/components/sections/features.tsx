import { Shield, Lock, Zap, Server, Smartphone, Users, Key, Globe, Fingerprint, Clock, RefreshCw, HardDrive } from 'lucide-react'

const features = [
  {
    icon: Lock,
    title: 'End-to-End Encryption',
    description: 'Files encrypted on your device with AES-256-GCM. Only you hold the keys.',
  },
  {
    icon: Server,
    title: 'Self-Hosted',
    description: 'Deploy on your own infrastructure. Full control, no third-party dependencies.',
  },
  {
    icon: Shield,
    title: 'Zero-Knowledge',
    description: 'Server cannot access your files. Encryption keys never leave your devices.',
  },
  {
    icon: Zap,
    title: 'Delta Sync',
    description: 'Only changed chunks transferred using Rabin CDC. Lightning fast uploads.',
  },
  {
    icon: Smartphone,
    title: 'Cross-Platform',
    description: 'Web, desktop (Win/Mac/Linux), CLI tools, and mobile apps coming soon.',
  },
  {
    icon: Users,
    title: 'Team Collaboration',
    description: 'Share files with your team. Granular permissions and full audit logs.',
  },
  {
    icon: Key,
    title: 'Secure Sharing',
    description: 'Password-protected links with expiration dates and download limits.',
  },
  {
    icon: Globe,
    title: 'Federation Ready',
    description: 'Connect with other VaultDrift instances. True decentralized storage.',
  },
]

export function Features() {
  return (
    <section id="features" className="py-24 relative">
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-blue-500/5 to-transparent" />

      <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold tracking-tight mb-6">
            Everything You Need for
            <span className="text-gradient"> Secure Storage</span>
          </h2>
          <p className="text-lg text-zinc-400">
            Built with security and privacy as the foundation. Every feature designed to keep your data safe.
          </p>
        </div>

        {/* Features Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {features.map((feature) => (
            <div
              key={feature.title}
              className="group relative p-6 rounded-2xl bg-zinc-900/50 border border-white/5 hover:border-blue-500/30 transition-all duration-300 hover:-translate-y-1"
            >
              <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500/20 to-cyan-500/20 flex items-center justify-center mb-4 group-hover:from-blue-500/30 group-hover:to-cyan-500/30 transition-all">
                <feature.icon className="w-6 h-6 text-blue-400" />
              </div>
              <h3 className="font-semibold text-lg mb-2">{feature.title}</h3>
              <p className="text-sm text-zinc-400 leading-relaxed">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
