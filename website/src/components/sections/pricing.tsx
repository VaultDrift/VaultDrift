import { Button } from '@/components/ui/button'
import { Check } from 'lucide-react'

const plans = [
  {
    name: 'Self-Hosted',
    description: 'Free forever. Run on your own hardware.',
    price: 'Free',
    period: '',
    features: [
      'Unlimited storage (your hardware)',
      'Unlimited users',
      'End-to-end encryption',
      'Web interface',
      'Desktop apps',
      'CLI tools',
      'Community support',
    ],
    cta: 'Download',
    href: 'https://github.com/vaultdrift/vaultdrift/releases',
    popular: true,
  },
  {
    name: 'Enterprise',
    description: 'Managed hosting and premium support.',
    price: 'Custom',
    period: '',
    features: [
      'Everything in Self-Hosted',
      'Managed cloud hosting',
      '99.99% SLA',
      'Priority support',
      'Custom integrations',
      'Dedicated infrastructure',
      'Security audits',
    ],
    cta: 'Contact Sales',
    href: 'mailto:enterprise@vaultdrift.com',
    popular: false,
  },
]

export function Pricing() {
  return (
    <section id="pricing" className="py-24 relative">
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-cyan-500/5 to-transparent" />

      <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold tracking-tight mb-6">
            Simple, Transparent
            <span className="text-gradient"> Pricing</span>
          </h2>
          <p className="text-lg text-zinc-400">
            Free for self-hosting. Pay only if you need managed hosting.
          </p>
        </div>

        {/* Pricing Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 max-w-4xl mx-auto">
          {plans.map((plan) => (
            <div
              key={plan.name}
              className={`relative rounded-2xl p-8 ${
                plan.popular
                  ? 'bg-gradient-to-b from-blue-500/10 to-transparent border-2 border-blue-500/30 glow'
                  : 'bg-zinc-900/50 border border-white/5'
              }`}
            >
              {plan.popular && (
                <div className="absolute -top-4 left-1/2 -translate-x-1/2">
                  <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-gradient-to-r from-blue-600 to-cyan-500 text-white">
                    Most Popular
                  </span>
                </div>
              )}

              <div className="mb-6">
                <h3 className="text-xl font-semibold mb-2">{plan.name}</h3>
                <p className="text-sm text-zinc-500">{plan.description}</p>
              </div>

              <div className="mb-6">
                <span className="text-4xl font-bold">{plan.price}</span>
                {plan.period && <span className="text-zinc-500">/{plan.period}</span>}
              </div>

              <ul className="space-y-3 mb-8">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-3">
                    <Check className="w-5 h-5 text-cyan-400 shrink-0 mt-0.5" />
                    <span className="text-sm text-zinc-400">{feature}</span>
                  </li>
                ))}
              </ul>

              <Button
                className={`w-full h-11 ${
                  plan.popular
                    ? 'bg-gradient-to-r from-blue-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 border-0'
                    : 'bg-white/5 hover:bg-white/10 border border-white/10'
                }`}
                asChild
              >
                <a href={plan.href} target="_blank" rel="noopener noreferrer">
                  {plan.cta}
                </a>
              </Button>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
