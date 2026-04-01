import { Github, Twitter, MessageCircle } from 'lucide-react'

const footerLinks = {
  Product: [
    { label: 'Features', href: '#features' },
    { label: 'Pricing', href: '#pricing' },
    { label: 'Download', href: 'https://github.com/vaultdrift/vaultdrift/releases' },
    { label: 'Changelog', href: 'https://github.com/vaultdrift/vaultdrift/releases' },
  ],
  Resources: [
    { label: 'Documentation', href: '#' },
    { label: 'API Reference', href: '#' },
    { label: 'Self-Hosting Guide', href: '#' },
  ],
  Community: [
    { label: 'GitHub', href: 'https://github.com/vaultdrift/vaultdrift' },
    { label: 'Discord', href: '#' },
    { label: 'Twitter', href: '#' },
  ],
}

const socialLinks = [
  { icon: Github, href: 'https://github.com/vaultdrift/vaultdrift', label: 'GitHub' },
  { icon: Twitter, href: '#', label: 'Twitter' },
  { icon: MessageCircle, href: '#', label: 'Discord' },
]

export function Footer() {
  return (
    <footer className="border-t border-white/5 bg-zinc-900/30">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mb-12">
          {Object.entries(footerLinks).map(([category, links]) => (
            <div key={category}>
              <h3 className="font-semibold text-sm mb-4 text-zinc-300">{category}</h3>
              <ul className="space-y-3">
                {links.map((link) => (
                  <li key={link.label}>
                    <a
                      href={link.href}
                      target={link.href.startsWith('http') ? '_blank' : undefined}
                      rel={link.href.startsWith('http') ? 'noopener noreferrer' : undefined}
                      className="text-sm text-zinc-500 hover:text-white transition-colors"
                    >
                      {link.label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}

          <div>
            <h3 className="font-semibold text-sm mb-4 text-zinc-300">Connect</h3>
            <div className="flex gap-3">
              {socialLinks.map((social) => (
                <a
                  key={social.label}
                  href={social.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="w-10 h-10 rounded-lg bg-zinc-800 flex items-center justify-center text-zinc-400 hover:text-white hover:bg-zinc-700 transition-colors"
                  aria-label={social.label}
                >
                  <social.icon className="w-5 h-5" />
                </a>
              ))}
            </div>
          </div>
        </div>

        <div className="border-t border-white/5 pt-8 flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <span className="font-bold text-lg">VaultDrift</span>
          </div>
          <p className="text-sm text-zinc-500">
            © 2024 VaultDrift. Open source under MIT License.
          </p>
        </div>
      </div>
    </footer>
  )
}
