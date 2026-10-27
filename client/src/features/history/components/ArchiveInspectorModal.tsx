import { useState, useRef } from 'react'
import {
  X, UploadCloud, CheckCircle2, AlertCircle, Database, FileText,
  Search, ShieldCheck, Loader2, Copy, Check, Table2, Layers
} from 'lucide-react'
import {
  inspectFileHeader, calculateStreamingChecksum,
  type InspectionResult
} from '@/services/wasmInspector'
import { formatBytes } from '@/utils/format'

interface ArchiveInspectorModalProps {
  onClose: () => void
  expectedChecksum?: string
  initialJobName?: string
}

export const ArchiveInspectorModal = ({
  onClose,
  expectedChecksum,
  initialJobName,
}: ArchiveInspectorModalProps) => {
  const [file, setFile] = useState<File | null>(null)
  const [isAnalyzing, setIsAnalyzing] = useState(false)
  const [inspection, setInspection] = useState<InspectionResult | null>(null)
  const [isHashing, setIsHashing] = useState(false)
  const [hashProgress, setHashProgress] = useState<number | null>(null)
  const [calculatedHash, setCalculatedHash] = useState<string | null>(null)
  const [targetChecksum, setTargetChecksum] = useState(expectedChecksum ?? '')
  const [searchTerm, setSearchTerm] = useState('')
  const [copiedHash, setCopiedHash] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileChange = async (selectedFile: File) => {
    setFile(selectedFile)
    setInspection(null)
    setCalculatedHash(null)
    setHashProgress(null)
    setIsAnalyzing(true)

    try {
      const result = await inspectFileHeader(selectedFile)
      setInspection(result)
    } catch (err) {
      setInspection({
        format: 'Unknown / Error',
        header_valid: false,
        metadata: {},
        tables: [],
        error: err instanceof Error ? err.message : 'Failed to parse archive header',
      })
    } finally {
      setIsAnalyzing(false)
    }
  }

  const handleComputeChecksum = async () => {
    if (!file) return
    setIsHashing(true)
    setHashProgress(0)

    try {
      const hash = await calculateStreamingChecksum(file, (pct) => {
        setHashProgress(pct)
      })
      setCalculatedHash(hash)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Hashing failed')
    } finally {
      setIsHashing(false)
    }
  }

  const handleCopyHash = () => {
    if (!calculatedHash) return
    navigator.clipboard.writeText(calculatedHash)
    setCopiedHash(true)
    setTimeout(() => setCopiedHash(false), 2000)
  }

  const filteredTables = inspection?.tables.filter((t) =>
    `${t.schema}.${t.name}`.toLowerCase().includes(searchTerm.toLowerCase())
  ) ?? []

  const isChecksumMatch =
    calculatedHash && targetChecksum.trim()
      ? calculatedHash.toLowerCase() === targetChecksum.trim().toLowerCase()
      : null

  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-150"
    >
      <div className="w-full max-w-2xl bg-surface-container-lowest border border-outline-variant/60 rounded-2xl shadow-2xl flex flex-col max-h-[90vh] overflow-hidden animate-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="px-5 py-4 border-b border-outline-variant/40 flex items-center justify-between bg-surface-container-low shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
              <Database size={18} />
            </div>
            <div>
              <h2 className="text-[16px] font-heading font-semibold text-on-surface">
                Backup Archive Inspector
              </h2>
              <p className="text-[11.5px] text-on-surface-variant">
                Client-side Wasm header parser & streaming SHA-256 verifier
                {initialJobName ? ` for ${initialJobName}` : ''}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded-lg text-on-surface-variant hover:bg-surface-container-high transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Content */}
        <div className="p-5 flex-1 overflow-y-auto flex flex-col gap-5">
          {/* Dropzone */}
          {!file ? (
            <div
              onClick={() => fileInputRef.current?.click()}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault()
                const dropped = e.dataTransfer.files[0]
                if (dropped) void handleFileChange(dropped)
              }}
              className="border-2 border-dashed border-outline-variant rounded-xl p-8 flex flex-col items-center justify-center gap-2.5 cursor-pointer hover:border-primary/60 hover:bg-surface-container-low transition-all text-center"
            >
              <div className="w-12 h-12 rounded-full bg-primary/10 text-primary flex items-center justify-center mb-1">
                <UploadCloud size={24} />
              </div>
              <p className="text-[14px] font-medium text-on-surface">
                Select or drop a backup archive
              </p>
              <p className="text-[12px] text-on-surface-variant max-w-xs">
                Supports PostgreSQL custom dumps (<code className="font-mono text-[11px]">.dump</code>), <code className="font-mono text-[11px]">.tar.gz</code>, or <code className="font-mono text-[11px]">.sql.gz</code>
              </p>
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                accept=".dump,.sql,.gz,.tar,.tgz"
                onChange={(e) => {
                  const selected = e.target.files?.[0]
                  if (selected) void handleFileChange(selected)
                }}
              />
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              {/* File summary banner */}
              <div className="p-3.5 rounded-xl bg-surface-container-low border border-outline-variant/40 flex items-center justify-between gap-3">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-10 h-10 rounded-lg bg-surface-container-highest flex items-center justify-center shrink-0">
                    <FileText size={20} className="text-on-surface-variant" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-[13.5px] font-medium text-on-surface truncate">{file.name}</p>
                    <p className="text-[11.5px] text-on-surface-variant font-mono">{formatBytes(file.size)}</p>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => {
                    setFile(null)
                    setInspection(null)
                    setCalculatedHash(null)
                  }}
                  className="px-2.5 py-1 text-[11.5px] text-primary hover:underline font-medium shrink-0"
                >
                  Choose another
                </button>
              </div>

              {/* Parsing State */}
              {isAnalyzing && (
                <div className="py-8 flex flex-col items-center justify-center gap-2">
                  <Loader2 size={24} className="animate-spin text-primary" />
                  <span className="text-[12.5px] text-on-surface-variant">Inspecting archive header with Wasm…</span>
                </div>
              )}

              {/* Inspection Results */}
              {inspection && !isAnalyzing && (
                <div className="flex flex-col gap-4">
                  {/* Format & Validation Badge */}
                  <div className="flex items-center justify-between p-3 rounded-xl bg-surface-container-lowest border border-outline-variant/40">
                    <div className="flex items-center gap-2">
                      <span className="text-[11.5px] text-on-surface-variant">Detected Format:</span>
                      <span className="px-2 py-0.5 rounded-md bg-primary/10 text-primary font-medium text-[12px]">
                        {inspection.format}
                      </span>
                    </div>
                    {inspection.header_valid ? (
                      <span className="flex items-center gap-1 text-[11.5px] text-success font-medium">
                        <CheckCircle2 size={14} /> Valid Header
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-[11.5px] text-warning font-medium">
                        <AlertCircle size={14} /> {inspection.error || 'Header check unverified'}
                      </span>
                    )}
                  </div>

                  {/* Schema / Table Inventory */}
                  {inspection.tables.length > 0 && (
                    <div className="flex flex-col gap-2">
                      <div className="flex items-center justify-between">
                        <span className="text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant flex items-center gap-1.5">
                          <Table2 size={14} />
                          Identified Objects ({inspection.tables.length})
                        </span>
                        <div className="relative w-48">
                          <Search size={13} className="absolute left-2.5 top-2.5 text-on-surface-variant" />
                          <input
                            type="text"
                            placeholder="Filter objects…"
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                            className="w-full h-7 pl-7 pr-2 rounded-lg border border-outline-variant/60 bg-surface-container-low text-[11.5px] text-on-surface focus:outline-none focus:ring-1 focus:ring-primary"
                          />
                        </div>
                      </div>

                      <div className="max-h-48 overflow-y-auto rounded-xl border border-outline-variant/40 bg-surface-container-lowest divide-y divide-outline-variant/20">
                        {filteredTables.map((t, idx) => (
                          <div key={idx} className="px-3 py-1.5 flex items-center justify-between text-[12px]">
                            <div className="flex items-center gap-2 font-mono">
                              <span className="text-on-surface-variant">{t.schema}.</span>
                              <span className="text-on-surface font-medium">{t.name}</span>
                            </div>
                            <span className="px-1.5 py-0.2 rounded text-[10px] font-medium bg-surface-container-high text-on-surface-variant">
                              {t.type}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Tar Files List */}
                  {inspection.files && inspection.files.length > 0 && (
                    <div className="flex flex-col gap-2">
                      <span className="text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant flex items-center gap-1.5">
                        <Layers size={14} />
                        Archive File Tree ({inspection.files.length})
                      </span>
                      <div className="max-h-44 overflow-y-auto rounded-xl border border-outline-variant/40 bg-surface-container-lowest divide-y divide-outline-variant/20">
                        {inspection.files.map((f, idx) => (
                          <div key={idx} className="px-3 py-1.5 flex items-center justify-between text-[11.5px] font-mono">
                            <span className="truncate max-w-sm text-on-surface">{f.path}</span>
                            <span className="text-on-surface-variant">{formatBytes(f.size)}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Streaming Checksum Verification Card */}
                  <div className="p-4 rounded-xl bg-surface-container-low border border-outline-variant/50 flex flex-col gap-3">
                    <div className="flex items-center justify-between">
                      <span className="text-[12.5px] font-medium text-on-surface flex items-center gap-1.5">
                        <ShieldCheck size={16} className="text-primary" />
                        Integrity & Checksum Verification (SHA-256)
                      </span>
                      {!calculatedHash && (
                        <button
                          type="button"
                          onClick={handleComputeChecksum}
                          disabled={isHashing}
                          className="px-3 py-1 rounded-lg bg-primary text-on-primary text-[12px] font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-1.5"
                        >
                          {isHashing ? <Loader2 size={13} className="animate-spin" /> : null}
                          {isHashing ? 'Streaming…' : 'Compute SHA-256'}
                        </button>
                      )}
                    </div>

                    {/* Progress Bar */}
                    {isHashing && hashProgress !== null && (
                      <div className="flex flex-col gap-1">
                        <div className="flex justify-between text-[11px] text-on-surface-variant">
                          <span>Hashing file stream…</span>
                          <span>{hashProgress}%</span>
                        </div>
                        <div className="w-full h-1.5 bg-surface-container-highest rounded-full overflow-hidden">
                          <div
                            className="h-full bg-primary transition-all duration-150"
                            style={{ width: `${hashProgress}%` }}
                          />
                        </div>
                      </div>
                    )}

                    {/* Calculated Hash Output */}
                    {calculatedHash && (
                      <div className="flex flex-col gap-2">
                        <div className="flex items-center gap-2 p-2 rounded-lg bg-surface-container-lowest border border-outline-variant/40">
                          <code className="text-[11px] font-mono text-on-surface flex-1 break-all select-all">
                            {calculatedHash}
                          </code>
                          <button
                            type="button"
                            onClick={handleCopyHash}
                            className="p-1 rounded text-on-surface-variant hover:text-on-surface transition-colors"
                            title="Copy hash"
                          >
                            {copiedHash ? <Check size={14} className="text-success" /> : <Copy size={14} />}
                          </button>
                        </div>

                        {/* Compare with Expected Hash */}
                        <div className="flex items-center gap-2">
                          <input
                            type="text"
                            placeholder="Paste expected SHA-256 hash to compare…"
                            value={targetChecksum}
                            onChange={(e) => setTargetChecksum(e.target.value)}
                            className="flex-1 h-8 px-2.5 rounded-lg border border-outline-variant/60 bg-surface-container-lowest font-mono text-[11px] text-on-surface focus:outline-none focus:ring-1 focus:ring-primary"
                          />
                          {isChecksumMatch !== null && (
                            <span
                              className={`px-2.5 py-1 rounded-lg text-[11.5px] font-medium flex items-center gap-1 shrink-0 ${
                                isChecksumMatch
                                  ? 'bg-success-container text-on-success-container'
                                  : 'bg-error-container text-error'
                              }`}
                            >
                              {isChecksumMatch ? (
                                <>
                                  <CheckCircle2 size={13} /> Hash Matched
                                </>
                              ) : (
                                <>
                                  <AlertCircle size={13} /> Mismatch
                                </>
                              )}
                            </span>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-outline-variant/40 bg-surface-container-low flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-1.5 rounded-lg text-[12.5px] font-medium bg-surface-container-high hover:bg-surface-container-highest text-on-surface transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

export default ArchiveInspectorModal
