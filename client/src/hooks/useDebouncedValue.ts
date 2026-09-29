import { useEffect, useState } from 'react'

// Delays propagating a fast-changing value (e.g. search keystrokes) until
// it settles, so server-filtered lists don't fire a request per keystroke.
export function useDebouncedValue<T>(value: T, delayMs = 300): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(t)
  }, [value, delayMs])
  return debounced
}
