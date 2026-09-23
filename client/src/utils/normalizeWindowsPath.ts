/**
 * Normalizes any Windows path string into clean, single-backslash Windows format.
 * Correctly preserves UNC network roots (\\server\share) and drive roots (C:\).
 */
const normalizeWindowsPath = (rawPath: string): string => {
  if (!rawPath || typeof rawPath !== 'string') return ''

  // 1. Trim whitespace
  let p = rawPath.trim()

  // 2. Convert all forward slashes to backslashes
  p = p.replace(/\//g, '\\')

  // 3. Detect and preserve UNC network prefix (e.g. \\server\share or \\?\UNC\)
  const isUnc = p.startsWith('\\\\')

  // 4. Collapse all consecutive backslashes into a single backslash
  p = p.replace(/\\+/g, '\\')

  // 5. Re-apply the leading double backslash for UNC paths
  if (isUnc) {
    p = '\\' + p
  }

  // 6. Resolve redundant ".\" or "\.\" references
  p = p.replace(/(^|[\\/])\.(?=[\\/]|$)/g, '$1')
  p = p.replace(/\\+/g, '\\')
  if (isUnc && !p.startsWith('\\\\')) {
    p = '\\' + p
  }

  // 7. Strip trailing slash unless it is a drive root (e.g., "D:\" or "\")
  const isDriveRoot = /^[a-zA-Z]:\\$/.test(p)
  if (!isDriveRoot && p.length > 1 && p.endsWith('\\')) {
    p = p.slice(0, -1)
  }

  return p
}

export default normalizeWindowsPath