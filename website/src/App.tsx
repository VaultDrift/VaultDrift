import { Navigation } from '@/components/navigation'
import { Hero } from '@/components/sections/hero'
import { Features } from '@/components/sections/features'
import { Stats } from '@/components/sections/stats'
import { Pricing } from '@/components/sections/pricing'
import { CTA } from '@/components/sections/cta'
import { Footer } from '@/components/sections/footer'

function App() {
  return (
    <div className="min-h-screen bg-background text-foreground overflow-x-hidden">
      <Navigation />
      <main>
        <Hero />
        <Stats />
        <Features />
        <Pricing />
        <CTA />
      </main>
      <Footer />
    </div>
  )
}

export default App
