import { Button } from '@/components/ui/button'
import { ArrowRight, Download } from 'lucide-react'

export function CTA() {
  return (
    <section className="py-24 relative overflow-hidden">
      <div className="absolute inset-0 bg-gradient-to-r from-blue-600/20 via-purple-600/20 to-cyan-500/20" />
      <div className="absolute inset-0 opacity-30">
        <div className="absolute inset-0" style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.03'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
        }} />
      </div>

      <div className="relative max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-6">
          Ready to Take Control of Your Data?
        </h2>
        <p className="text-lg sm:text-xl text-zinc-400 mb-10 max-w-2xl mx-auto">
          Join thousands of users who have switched to self-hosted, secure cloud storage.
          Your files deserve better.
        </p>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <Button
            size="lg"
            className="h-12 px-8 bg-white text-black hover:bg-zinc-200 font-semibold"
            asChild
          >
            <a href="https://github.com/vaultdrift/vaultdrift/releases" target="_blank" rel="noopener noreferrer">
              <Download className="w-5 h-5 mr-2" />
              Download Free
            </a>
          </Button>
          <Button
            size="lg"
            variant="outline"
            className="h-12 px-8 border-white/20 hover:bg-white/5"
            asChild
          >
            <a href="https://github.com/vaultdrift/vaultdrift" target="_blank" rel="noopener noreferrer">
              View on GitHub
              <ArrowRight className="w-4 h-4 ml-2" />
            </a>
          </Button>
        </div>
      </div>
    </section>
  )
}
