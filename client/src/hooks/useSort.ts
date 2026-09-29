import { useState } from 'react'

export type SortDir = 'asc' | 'desc'

export type SortGetters<T> = Record<string, (row: T) => string | number>

// Client-side column sorting for tables and card lists. `getters` maps a
// sort key to a comparable value extractor — define it at module level so
// its identity is stable. Returns rows sorted by the active key/dir.
export function useSort<T>(
  rows: T[],
  getters: SortGetters<T>,
  defaultKey: string,
  defaultDir: SortDir = 'desc',
) {
  const [sortKey, setSortKey] = useState(defaultKey)
  const [sortDir, setSortDir] = useState<SortDir>(defaultDir)

  const toggleSort = (key: string) => {
    if (!(key in getters)) return
    if (key === sortKey) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('desc')
    }
  }

  const get = getters[sortKey]
  const sorted = get
    ? [...rows].sort((a, b) => {
        const va = get(a)
        const vb = get(b)
        const cmp =
          typeof va === 'number' && typeof vb === 'number'
            ? va - vb
            : String(va).localeCompare(String(vb))
        return sortDir === 'asc' ? cmp : -cmp
      })
    : rows

  return { sortKey, sortDir, toggleSort, sorted }
}
