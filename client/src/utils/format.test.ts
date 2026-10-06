import { describe, expect, it } from 'vitest'
import { maskEnrollmentToken } from './format'

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
