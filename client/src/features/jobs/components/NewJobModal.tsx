import Modal from '@/components/ui/modal/Modal'
import JobWizard from './JobWizard'

interface Props {
  onClose: () => void
  onSaved: () => void
}

const NewJobModal = ({ onClose, onSaved }: Props) => {
  return (
    <Modal isOpen onClose={onClose} title="Create Backup Job" size="3xl">
      <JobWizard onClose={onClose} onSaved={onSaved} />
    </Modal>
  )
}

export default NewJobModal
