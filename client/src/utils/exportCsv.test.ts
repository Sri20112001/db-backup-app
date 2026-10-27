import { describe, it, expect } from 'vitest'
import { formatCsv, exportToCsv } from './exportCsv'

describe('formatCsv', () => {
  it('correctly escapes and formats CSV cells', () => {
    const headers = ['ID', 'Job Name', 'Status', 'Size']
    const rows = [
      ['1', 'DB-Backup', 'COMPLETED', 1024],
      ['2', 'Backup "Production", Main', 'FAILED', 0],
      ['3', 'Multi\nLine\nName', 'RUNNING', null],
    ]

    const csv = formatCsv(headers, rows)
    const lines = csv.split('\r\n')

    expect(lines).toHaveLength(4)
    expect(lines[0]).toBe('"ID","Job Name","Status","Size"')
    expect(lines[1]).toBe('"1","DB-Backup","COMPLETED","1024"')
    expect(lines[2]).toBe('"2","Backup ""Production"", Main","FAILED","0"')
    expect(lines[3]).toContain('Multi\nLine\nName')
  })

  it('handles null and undefined values as empty strings', () => {
    const csv = formatCsv(['A', 'B'], [[null, undefined]])
    expect(csv).toBe('"A","B"\r\n"",""')
  })

  it('safely handles non-browser environment without throwing', () => {
    expect(() => {
      exportToCsv('test.csv', ['A'], [['1']])
    }).not.toThrow()
  })
})
