import { ArrowRight, Sparkles, Database, Server, Archive } from 'lucide-react'

const STEPS = [
  { icon: Database, title: 'Add storage', desc: 'Choose where your backup data lives — S3-compatible or a local path.' },
  { icon: Server, title: 'Register an agent', desc: 'Install the VaultGuard agent on a machine to run backups.' },
  { icon: Archive, title: 'Create a backup job', desc: 'Point a job at a folder, schedule it, and you are done.' },
]

interface WelcomeStepProps {
  onStart: () => void
  onSkip: () => void
}

const WelcomeStep = ({ onStart, onSkip }: WelcomeStepProps) => (
  <div className="w-full max-w-xl mx-auto flex flex-col items-center gap-6 text-center">
    <div className="w-14 h-14 rounded-2xl bg-primary-container flex items-center justify-center text-primary">
      <Sparkles size={26} />
    </div>
    <div>
      <h2 className="text-[24px] font-semibold text-on-surface">Welcome to VaultGuard</h2>
      <p className="text-[14px] text-on-surface-variant mt-2">
        Let's get your first backup running. It only takes three quick steps.
      </p>
    </div>
    <div className="w-full flex flex-col gap-3 text-left">
      {STEPS.map((s, i) => (
        <div key={s.title} className="flex items-center gap-4 p-4 rounded-xl bg-surface-container-lowest border border-surface-variant">
          <div className="w-10 h-10 rounded-lg bg-primary-container/40 flex items-center justify-center text-primary shrink-0">
            <s.icon size={20} />
          </div>
          <div>
            <p className="text-[14px] font-semibold text-on-surface">{i + 1}. {s.title}</p>
            <p className="text-[12px] text-on-surface-variant">{s.desc}</p>
          </div>
        </div>
      ))}
    </div>
    <div className="flex items-center gap-3">
      <button
        type="button"
        onClick={onStart}
        className="flex items-center gap-2 px-5 h-10 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors"
      >
        Get started
        <ArrowRight size={16} />
      </button>
      <button
        type="button"
        onClick={onSkip}
        className="px-5 h-10 rounded-lg border border-surface-variant text-on-surface-variant text-[13px] font-medium hover:bg-surface-container-low transition-colors"
      >
        Skip for now
      </button>
    </div>
  </div>
)

export default WelcomeStep
