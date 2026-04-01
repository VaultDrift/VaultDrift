import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'

const faqs = [
  {
    question: 'Is VaultDrift really free?',
    answer:
      'Yes! VaultDrift is completely free and open source. You can self-host it on your own hardware without any licensing fees. We only charge for managed hosting if you prefer not to manage the infrastructure yourself.',
  },
  {
    question: 'How secure is my data?',
    answer:
      'VaultDrift uses industry-standard AES-256-GCM encryption for your files. All data is encrypted on your device before being uploaded to the server. The server only stores encrypted chunks and cannot access your actual files or encryption keys.',
  },
  {
    question: 'Can I access my files from anywhere?',
    answer:
      'Yes. As long as your VaultDrift server is accessible from the internet (via port forwarding, VPN, or reverse proxy), you can access your files from any web browser or using our desktop and mobile apps.',
  },
  {
    question: 'What are the system requirements?',
    answer:
      'VaultDrift is lightweight and can run on a Raspberry Pi or any small VPS. Minimum requirements: 512MB RAM, 1 CPU core, and as much storage as you need for your files. For better performance, we recommend 2GB RAM or more.',
  },
  {
    question: 'How do I backup my VaultDrift data?',
    answer:
      'Since VaultDrift stores files in chunks on disk, you can back up the entire data directory. The database (SQLite) and storage chunks can be backed up using standard tools like rsync, restic, or any backup solution you prefer.',
  },
  {
    question: 'Is there a mobile app?',
    answer:
      'The web interface is fully responsive and works great on mobile browsers. Native mobile apps for iOS and Android are on our roadmap. You can also use the PWA (Progressive Web App) functionality by adding the site to your home screen.',
  },
  {
    question: 'Can I share files with people who do not use VaultDrift?',
    answer:
      'Yes! You can create share links with optional password protection, expiration dates, and download limits. External users can download shared files without needing an account.',
  },
  {
    question: 'How does federation work?',
    answer:
      'Federation (coming soon) allows VaultDrift instances to communicate with each other. This means you can share files between different VaultDrift servers, similar to how email works across different providers.',
  },
]

export function FAQ() {
  return (
    <section id="faq" className="py-24 relative">
      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold tracking-tight mb-4">
            Frequently Asked
            <span className="text-primary"> Questions</span>
          </h2>
          <p className="text-lg text-muted-foreground">
            Everything you need to know about VaultDrift.
          </p>
        </div>

        {/* FAQ Accordion */}
        <Accordion type="single" collapsible className="w-full">
          {faqs.map((faq, index) => (
            <AccordionItem key={index} value={`item-${index}`}>
              <AccordionTrigger className="text-left text-base font-medium">
                {faq.question}
              </AccordionTrigger>
              <AccordionContent className="text-muted-foreground">
                {faq.answer}
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </section>
  )
}
