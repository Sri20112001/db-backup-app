import { useEffect, useRef, useState } from 'react'

interface CountUpProps {
  value: number
  duration?: number // ms
  formatter?: (val: number) => string
  className?: string
}

export const CountUp = ({
  value,
  duration = 900,
  formatter = (v) => Math.round(v).toLocaleString(),
  className = '',
}: CountUpProps) => {
  const [displayValue, setDisplayValue] = useState(0)
  // Mirror of displayValue for the animation's start point: kept in a ref so
  // the effect below doesn't re-run on every animation frame.
  const displayRef = useRef(0)

  useEffect(() => {
    let startTimestamp: number | null = null
    const startValue = displayRef.current

    const step = (timestamp: number) => {
      if (!startTimestamp) startTimestamp = timestamp
      const progress = Math.min((timestamp - startTimestamp) / duration, 1)

      // Ease-out cubic interpolation: 1 - (1 - t)^3
      const easeOutProgress = 1 - Math.pow(1 - progress, 3)
      const current = startValue + (value - startValue) * easeOutProgress

      displayRef.current = current
      setDisplayValue(current)

      if (progress < 1) {
        requestAnimationFrame(step)
      } else {
        displayRef.current = value
        setDisplayValue(value)
      }
    }

    const animId = requestAnimationFrame(step)
    return () => cancelAnimationFrame(animId)
  }, [value, duration])

  return <span className={className}>{formatter(displayValue)}</span>
}

export default CountUp