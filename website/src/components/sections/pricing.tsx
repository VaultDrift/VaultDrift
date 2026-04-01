import { Check } from 'lucide-react'
import { Button } from '@/components/ui/button'

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
    cta: 'Download Now',
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
    <section id="pricing" className="py-24 relative bg-muted/50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-4">
            Simple, Transparent
            <span className="text-primary"> Pricing</span>
          </h2>
          <p className="text-lg text-muted-foreground">
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
                  ? 'bg-card border-2 border-primary shadow-lg'
                  : 'bg-card border border-border'
              }`}
            >
              {plan.popular && (
                <div className="absolute -top-4 left-1/2 -translate-x-1/2">
                  <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-primary text-primary-foreground">
                    Most Popular
                  </span>
                </div>
              )}

              <div className="mb-6">
                <h3 className="text-xl font-semibold mb-2">{plan.name}</h3>
                <p className="text-sm text-muted-foreground">{plan.description}</p>
              </div>

              <div className="mb-6">
                <span className="text-4xl font-bold">{plan.price}</span>
                {plan.period && (
                  <span className="text-muted-foreground">/{plan.period}</span>
                )}
              </div>

              <ul className="space-y-3 mb-8">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-3">
                    <Check className="w-5 h-5 text-primary shrink-0 mt-0.5" />
                    <span className="text-sm text-muted-foreground">{feature}</span>
                  </li>
                ))}
              </ul>

              <Button
                className="w-full"
                variant={plan.popular ? 'default' : 'outline'}
                size="lg"
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
