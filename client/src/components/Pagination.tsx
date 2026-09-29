import { ChevronLeft, ChevronRight } from 'lucide-react'

interface Props {
  page: number
  totalPages: number
  total: number
  perPage: number
  onPage: (page: number) => void
}

const Pagination = ({ page, totalPages, total, perPage, onPage }: Props) => {
  if (totalPages <= 1) return null
  const start = (page - 1) * perPage + 1
  const end = Math.min(page * perPage, total)

  // Compact window: 1 … p-1 p p+1 … N
  const pages: (number | '…')[] = []
  for (let p = 1; p <= totalPages; p++) {
    if (p === 1 || p === totalPages || Math.abs(p - page) <= 1) {
      pages.push(p)
    } else if (pages[pages.length - 1] !== '…') {
      pages.push('…')
    }
  }

  const btn =
    'min-w-8 h-8 px-2 rounded-lg text-[12px] font-medium transition-colors flex items-center justify-center'

  return (
    <div className="flex items-center justify-between gap-3 pt-1 shrink-0">
      <p className="text-[12px] text-outline">
        Showing {start}–{end} of {total}
      </p>
      <div className="flex items-center gap-1">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => onPage(page - 1)}
          aria-label="Previous page"
          className={`${btn} text-on-surface-variant hover:bg-surface-container-low disabled:opacity-40 disabled:pointer-events-none`}
        >
          <ChevronLeft size={15} />
        </button>
        {pages.map((p, i) =>
          p === '…' ? (
            <span key={`gap-${i}`} className={`${btn} text-outline pointer-events-none`}>
              …
            </span>
          ) : (
            <button
              key={p}
              type="button"
              onClick={() => onPage(p)}
              aria-current={p === page ? 'page' : undefined}
              className={`${btn} ${
                p === page
                  ? 'bg-primary text-on-primary shadow-sm'
                  : 'text-on-surface-variant hover:bg-surface-container-low'
              }`}
            >
              {p}
            </button>
          ),
        )}
        <button
          type="button"
          disabled={page >= totalPages}
          onClick={() => onPage(page + 1)}
          aria-label="Next page"
          className={`${btn} text-on-surface-variant hover:bg-surface-container-low disabled:opacity-40 disabled:pointer-events-none`}
        >
          <ChevronRight size={15} />
        </button>
      </div>
    </div>
  )
}

export default Pagination
