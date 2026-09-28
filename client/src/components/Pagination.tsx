import { useEffect, useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'

// Client-side pagination over an already-fetched array. Use for endpoints
// that return full lists (jobs, agents, storage, restores, members).
// Server-paginated endpoints (runs, alerts) manage page state themselves
// and only reuse the <Pagination> footer below.
//
// resetKey: pass a stable primitive (search text, org id, route id) to jump
// back to page 1 when the listing context changes. Never pass a fresh array
// or object here — its identity changes every render and would loop.
export function usePagination<T>(items: T[], perPage: number, resetKey?: unknown) {
  const [page, setPage] = useState(1)
  const totalPages = Math.max(1, Math.ceil(items.length / perPage))

  const [lastKey, setLastKey] = useState(resetKey)
  if (lastKey !== resetKey) {
    setLastKey(resetKey)
    setPage(1)
  }

  useEffect(() => {
    if (page > totalPages) setPage(totalPages)
  }, [page, totalPages])

  const pageItems = useMemo(
    () => items.slice((page - 1) * perPage, page * perPage),
    [items, page, perPage],
  )
  return { page, setPage, totalPages, pageItems, total: items.length, perPage }
}

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
      <p className="text-[12px] text-[#737686]">
        Showing {start}–{end} of {total}
      </p>
      <div className="flex items-center gap-1">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => onPage(page - 1)}
          aria-label="Previous page"
          className={`${btn} text-[#434655] hover:bg-[#f1f3ff] disabled:opacity-40 disabled:pointer-events-none`}
        >
          <ChevronLeft size={15} />
        </button>
        {pages.map((p, i) =>
          p === '…' ? (
            <span key={`gap-${i}`} className={`${btn} text-[#737686] pointer-events-none`}>
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
                  ? 'bg-[#2563eb] text-white shadow-sm'
                  : 'text-[#434655] hover:bg-[#f1f3ff]'
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
          className={`${btn} text-[#434655] hover:bg-[#f1f3ff] disabled:opacity-40 disabled:pointer-events-none`}
        >
          <ChevronRight size={15} />
        </button>
      </div>
    </div>
  )
}

export default Pagination
