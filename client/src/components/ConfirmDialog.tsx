import Dialog from './ui/modal/Dialog'

interface Props {
  title: string
  message: string
  confirmLabel?: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

const ConfirmDialog = ({
  title,
  message,
  confirmLabel = 'Confirm',
  danger = false,
  onConfirm,
  onCancel,
}: Props) => (
  <Dialog
    isOpen
    title={title}
    message={message}
    confirmLabel={confirmLabel}
    danger={danger}
    onConfirm={onConfirm}
    onCancel={onCancel}
  />
)

export default ConfirmDialog
