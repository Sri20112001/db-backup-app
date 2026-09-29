import { useEffect, useState } from 'react'

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

  useEffect(() => {
    let startTimestamp: number | null = null
    const startValue = displayValue

    const step = (timestamp: number) => {
      if (!startTimestamp) startTimestamp = timestamp
      const progress = Math.min((timestamp - startTimestamp) / duration, 1)

      // Ease-out cubic interpolation: 1 - (1 - t)^3
      const easeOutProgress = 1 - Math.pow(1 - progress, 3)
      const current = startValue + (value - startValue) * easeOutProgress

      setDisplayValue(current)

      if (progress < 1) {
        requestAnimationFrame(step)
      } else {
        setDisplayValue(value)
      }
    }

    const animId = requestAnimationFrame(step)
    return () => cancelAnimationFrame(animId)
  }, [value, duration])

  return <span className={className}>{formatter(displayValue)}</span>
}

export default CountUp