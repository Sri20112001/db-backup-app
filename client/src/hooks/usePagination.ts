import { useEffect, useMemo, useState } from 'react'

// Client-side pagination over an already-fetched array. Use for endpoints
// that return full lists (jobs, agents, storage, restores, members).
// Server-paginated endpoints (runs, alerts) manage page state themselves
// and only reuse the <Pagination> footer.
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
