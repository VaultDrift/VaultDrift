import { Shield, Lock, Zap, Server, Smartphone, Users, Key, Globe } from 'lucide-react'

const features = [
  {
    icon: Lock,
    title: 'End-to-End Encryption',
    description: 'Your files are encrypted on your device before they ever reach the server. Only you hold the keys.',
  },
  {
    icon: Server,
    title: 'Self-Hosted',
    description: 'Deploy on your own infrastructure. Full control over your data and no third-party dependencies.',
  },
  {
    icon: Shield,
    title: 'Zero-Knowledge',
    description: 'We cannot access your files. Your encryption keys never leave your devices.',
  },
  {
    icon: Zap,
    title: 'Lightning Fast',
    description: 'Chunked uploads and downloads with resume support. Optimized for speed and reliability.',
  },
  {
    icon: Smartphone,
    title: 'Cross-Platform',
    description: 'Web interface, desktop apps for Windows/Mac/Linux, and CLI tools for power users.',
  },
  {
    icon: Users,
    title: 'Team Sharing',
    description: 'Share files securely with your team. Granular permissions and audit logs included.',
  },
  {
    icon: Key,
    title: 'Secure Sharing',
    description: 'Create password-protected share links with expiration dates and download limits.',
  },
  {
    icon: Globe,
    title: 'Federation Ready',
    description: 'Coming soon: Connect with other VaultDrift instances. Decentralized cloud storage.',
  },
]

export function Features() {
  return (
    <section id="features" className="py-24 relative">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-4">
            Everything You Need for
            <span className="text-primary"> Secure Storage</span>
          </h2>
          <p className="text-lg text-muted-foreground">
            Built with security and privacy as the foundation. Every feature
designed to keep your data safe.
          </p>
        </div>

        {/* Features Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {features.map((feature) => (
            <div
              key={feature.title}
              className="group relative p-6 rounded-2xl bg-card border border-border hover:border-primary/50 transition-all duration-300 hover:-translate-y-1"
            >
              <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-4 group-hover:bg-primary/20 transition-colors">
                <feature.icon className="w-6 h-6 text-primary" />
              </div>
              <h3 className="font-semibold text-lg mb-2">{feature.title}</h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
