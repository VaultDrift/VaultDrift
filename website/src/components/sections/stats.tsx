export function Stats() {
  const stats = [
    { value: '99.9%', label: 'Uptime SLA' },
    { value: '256-bit', label: 'AES Encryption' },
    { value: 'Zero', label: 'Data Breaches' },
    { value: '100%', label: 'Open Source' },
  ]

  return (
    <section className="py-16 border-y border-white/5 bg-zinc-900/30">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-8">
          {stats.map((stat) => (
            <div key={stat.label} className="text-center">
              <div className="text-3xl sm:text-4xl font-bold text-gradient mb-2">
                {stat.value}
              </div>
              <div className="text-sm text-zinc-500">{stat.label}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
