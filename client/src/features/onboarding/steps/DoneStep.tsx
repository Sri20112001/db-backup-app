import { useNavigate } from 'react-router-dom'
import { CheckCircle2, Clock } from 'lucide-react'

interface DoneStepProps {
  onFinish: () => void
  done: boolean
  missing: string[]
}

const DoneStep = ({ onFinish, done, missing }: DoneStepProps) => {
  const navigate = useNavigate()

  // "a storage target, a registered agent and a backup job"
  const missingList = missing.length > 1
    ? `${missing.slice(0, -1).join(', ')} and ${missing[missing.length - 1]}`
    : missing[0]

  return (
    <div className="w-full max-w-xl mx-auto flex flex-col items-center gap-6 text-center">
      <div className={`w-16 h-16 rounded-2xl flex items-center justify-center ${done ? 'bg-primary-container text-primary' : 'bg-surface-container-high text-on-surface-variant'}`}>
        {done ? <CheckCircle2 size={30} /> : <Clock size={30} />}
      </div>
      {done ? (
        <div>
          <h2 className="text-[24px] font-semibold text-on-surface">You're all set</h2>
          <p className="text-[14px] text-on-surface-variant mt-2">
            Your storage, agent and first backup job are ready. The job will run on its schedule —
            you can also run it now from the dashboard.
          </p>
        </div>
      ) : (
        <div>
          <h2 className="text-[24px] font-semibold text-on-surface">Finish setup anytime</h2>
          <p className="text-[14px] text-on-surface-variant mt-2">
            Your setup is saved. Backups start once you have {missingList}. The dashboard checklist
            picks up right where you left off.
          </p>
        </div>
      )}
      <div className="flex items-center gap-3">
        {done && (
          <button
            type="button"
            onClick={() => navigate('/jobs')}
            className="px-5 h-10 rounded-lg border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
          >
            View backup jobs
          </button>
        )}
        <button
          type="button"
          onClick={onFinish}
          className="px-5 h-10 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors"
        >
          Go to dashboard
        </button>
      </div>
    </div>
  )
}

export default DoneStep
