export interface TableEntry {
  schema: string
  name: string
  type: string
  detail?: string
}

export interface FileEntry {
  path: string
  size: number
  mod_time: string
}

export interface InspectionResult {
  format: string
  metadata: Record<string, string>
  tables: TableEntry[]
  files?: FileEntry[]
  header_valid: boolean
  error?: string
}

declare global {
  class Go {
    importObject: WebAssembly.Imports
    run(instance: WebAssembly.Instance): Promise<void>
  }
  interface Window {
    VaultGuardWasm?: {
      initHasher: () => number
      updateHasher: (id: number, chunk: Uint8Array) => boolean
      finishHasher: (id: number) => string
      quickSha256: (chunk: Uint8Array) => string
      inspectHeader: (header: Uint8Array, filename: string) => string
      version: string
    }
  }
}

let wasmInitPromise: Promise<void> | null = null

export async function initWasm(): Promise<void> {
  if (typeof window === 'undefined') return
  if (window.VaultGuardWasm) return

  if (!wasmInitPromise) {
    wasmInitPromise = (async () => {
      // 1. Dynamically load wasm_exec.js if Go runtime is not present
      if (typeof Go === 'undefined') {
        await new Promise<void>((resolve, reject) => {
          const script = document.createElement('script')
          const base = import.meta.env.BASE_URL.endsWith('/')
            ? import.meta.env.BASE_URL
            : `${import.meta.env.BASE_URL}/`
          script.src = `${base}wasm_exec.js`
          script.onload = () => resolve()
          script.onerror = () => reject(new Error('Failed to load wasm_exec.js'))
          document.head.appendChild(script)
        })
      }

      // 2. Fetch and instantiate vaultguard.wasm
      const go = new Go()
      const base = import.meta.env.BASE_URL.endsWith('/')
        ? import.meta.env.BASE_URL
        : `${import.meta.env.BASE_URL}/`
      const wasmUrl = `${base}vaultguard.wasm`
      const resp = await fetch(wasmUrl)
      if (!resp.ok) {
        throw new Error(`Failed to load vaultguard.wasm: ${resp.status} ${resp.statusText}`)
      }
      const buffer = await resp.arrayBuffer()
      const result = await WebAssembly.instantiate(buffer, go.importObject)
      // Run the Go wasm instance asynchronously
      void go.run(result.instance)
    })()
  }

  return wasmInitPromise
}

/**
 * Inspects the Table of Contents & header of any backup archive by reading
 * only the initial 1MB slice. Works with .dump (pg_dump -Fc), .tar.gz, and .sql.
 */
export async function inspectFileHeader(file: File): Promise<InspectionResult> {
  await initWasm()
  const wasm = window.VaultGuardWasm
  if (!wasm) throw new Error('Wasm engine failed to initialize')

  // Slicing 1MB takes ~2ms regardless of total file size
  const headerSlice = file.slice(0, 1024 * 1024)
  const buffer = await headerSlice.arrayBuffer()
  const uint8 = new Uint8Array(buffer)

  const jsonResult = wasm.inspectHeader(uint8, file.name)
  return JSON.parse(jsonResult) as InspectionResult
}

/**
 * Calculates a streaming SHA-256 checksum over large files in 2MB chunks.
 * Keeps memory footprint constant (~2MB) and emits live progress percentage.
 */
export async function calculateStreamingChecksum(
  file: File,
  onProgress?: (percent: number) => void
): Promise<string> {
  await initWasm()
  const wasm = window.VaultGuardWasm
  if (!wasm) throw new Error('Wasm engine failed to initialize')

  const hasherId = wasm.initHasher()
  const chunkSize = 2 * 1024 * 1024 // 2MB chunk buffer
  let offset = 0
  const total = file.size

  while (offset < total) {
    const chunkBlob = file.slice(offset, offset + chunkSize)
    const arrayBuffer = await chunkBlob.arrayBuffer()
    const uint8 = new Uint8Array(arrayBuffer)
    wasm.updateHasher(hasherId, uint8)

    offset += chunkSize
    if (onProgress) {
      onProgress(Math.min(100, Math.round((offset / total) * 100)))
    }
  }

  return wasm.finishHasher(hasherId)
}
