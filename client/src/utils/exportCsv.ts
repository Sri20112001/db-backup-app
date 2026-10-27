/**
 * Formats data into a RFC 4180 compliant CSV string with quoted fields.
 */
export function formatCsv(
  headers: string[],
  rows: (string | number | boolean | null | undefined)[][]
): string {
  const escapeCell = (cell: string | number | boolean | null | undefined): string => {
    if (cell === null || cell === undefined) return '""'
    const str = String(cell)
    if (/[",\n\r]/.test(str)) {
      return `"${str.replace(/"/g, '""')}"`
    }
    return `"${str}"`
  }

  return [
    headers.map(escapeCell).join(','),
    ...rows.map((row) => row.map(escapeCell).join(',')),
  ].join('\r\n')
}

/**
 * Exports data rows to a downloadable CSV file.
 * Includes UTF-8 BOM so Microsoft Excel and other spreadsheet viewers render correctly.
 */
export function exportToCsv(
  filename: string,
  headers: string[],
  rows: (string | number | boolean | null | undefined)[][]
): void {
  const csvContent = formatCsv(headers, rows)
  if (typeof window === 'undefined' || typeof document === 'undefined') return

  const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.setAttribute('href', url)
  link.setAttribute('download', filename.endsWith('.csv') ? filename : `${filename}.csv`)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
