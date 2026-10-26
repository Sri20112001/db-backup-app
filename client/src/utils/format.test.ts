import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { formatRelative, maskEnrollmentToken } from './format'

describe('maskEnrollmentToken', () => {
  it('shows first and last quartets with the middle hidden', () => {
    const full = 'abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789'
    const masked = maskEnrollmentToken(full)
    expect(masked).toBe('abcd…6789')
    expect(masked).not.toContain(full.slice(4, -4))
  })

  it('never exposes short or empty values', () => {
    expect(maskEnrollmentToken('')).toBe('••••••')
    expect(maskEnrollmentToken('short')).toBe('••••••')
    expect(maskEnrollmentToken('1234567890')).toBe('••••••')
  })
})

describe('formatRelative', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-01-15T12:00:00.000Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('describes past timestamps', () => {
    expect(formatRelative('2026-01-15T11:59:45.000Z')).toBe('Just now')
    expect(formatRelative('2026-01-15T11:55:00.000Z')).toBe('5m ago')
    expect(formatRelative('2026-01-15T09:00:00.000Z')).toBe('3h ago')
    expect(formatRelative('2026-01-13T12:00:00.000Z')).toBe('2d ago')
  })

  it('describes future timestamps as upcoming, never "Just now"', () => {
    expect(formatRelative('2026-01-15T12:00:30.000Z')).toBe('in a moment')
    expect(formatRelative('2026-01-15T12:05:00.000Z')).toBe('in 5m')
    expect(formatRelative('2026-01-15T15:00:00.000Z')).toBe('in 3h')
    expect(formatRelative('2026-01-17T12:00:00.000Z')).toBe('in 2d')
  })

  it('handles null', () => {
    expect(formatRelative(null)).toBe('—')
  })
})
