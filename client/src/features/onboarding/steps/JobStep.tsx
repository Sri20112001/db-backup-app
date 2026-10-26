import JobWizard from '@/features/jobs/components/JobWizard'
import StepShell from '../components/StepShell'

interface JobStepProps {
  onDone: () => void
  onSkip: () => void
}

// Reuses the full job wizard (source → agent → schedule → processing → review)
// with the storage target and agent just created available for selection.
// Cancelling at the first step skips job creation without claiming success.
const JobStep = ({ onDone, onSkip }: JobStepProps) => (
  <StepShell
    title="Create your first backup job"
    subtitle="Pick a source, an agent and a schedule. You can refine everything later."
  >
    <JobWizard onClose={onDone} onCancel={onSkip} />
  </StepShell>
)

export default JobStep
