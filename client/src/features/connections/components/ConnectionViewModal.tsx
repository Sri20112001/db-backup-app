import { useState, useMemo } from 'react'
import {
  X,
  Pencil,
  RefreshCw,
  Loader2,
  Lock,
  Database,
  Search,
  Copy,
  Check,
  Table2,
  HardDrive,
  Layers,
  Server,
  Activity,
  AlertCircle,
} from 'lucide-react'
import { connectionsApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import type { DatabaseConnection, TableInfo } from '@/types'
import { formatBytes } from '@/utils/format'

interface Props {
  connection: DatabaseConnection
  agentName: string
  agentOnline: boolean
  refreshing: boolean
  onRefresh: () => void
  onEdit: () => void
  onClose: () => void
}

type TabType = 'databases' | 'settings'

const ConnectionViewModal = ({
  connection: c,
  agentName,
  agentOnline,
  refreshing,
  onRefresh,
  onEdit,
  onClose,
}: Props) => {
  const [activeTab, setActiveTab] = useState<TabType>('databases')
  const [filter, setFilter] = useState('')
  const [tableFilter, setTableFilter] = useState('')
  const [copiedDb, setCopiedDb] = useState<string | null>(null)
  const [selectedDb, setSelectedDb] = useState<string | null>(c.databases[0]?.name ?? null)
  const [tables, setTables] = useState<Record<string, TableInfo[]>>({})
  const [loadingTables, setLoadingTables] = useState<string | null>(null)
  const [tablesError, setTablesError] = useState<string | null>(null)
  const { currentOrg } = useAuthStore()

  const filteredDatabases = useMemo(() => {
    if (!filter.trim()) return c.databases
    const q = filter.toLowerCase().trim()
    return c.databases.filter((db) => db.name.toLowerCase().includes(q))
  }, [c.databases, filter])

  const visibleTables = useMemo(() => {
    if (!selectedDb || !tables[selectedDb]) return []
    if (!tableFilter.trim()) return tables[selectedDb]
    const q = tableFilter.toLowerCase().trim()
    return tables[selectedDb].filter(
      (t) => t.name.toLowerCase().includes(q) || (t.schema && t.schema.toLowerCase().includes(q))
    )
  }, [tables, selectedDb, tableFilter])

  const fmtSize = (b: number) => (b >= 0 ? formatBytes(b) : '—')

  const fetchTables = async (dbName: string) => {
    setSelectedDb(dbName)
    setTablesError(null)
    setTableFilter('')
    if (tables[dbName] || !currentOrg) return

    setLoadingTables(dbName)
    try {
      const res = await connectionsApi.tables(currentOrg.id, c.id, dbName)
      setTables((prev) => ({ ...prev, [dbName]: res.tables }))
    } catch (e) {
      setTablesError(e instanceof Error ? e.message : 'Could not inspect schema.')
    } finally {
      setLoadingTables(null)
    }
  }

  const handleCopy = (val: string) => {
    navigator.clipboard.writeText(val)
    setCopiedDb(val)
    setTimeout(() => setCopiedDb(null), 1500)
  }

  // Calculate high-level stats for selected database
  const selectedStats = useMemo(() => {
    const list = selectedDb ? tables[selectedDb] : null
    if (!list) return null
    const totalRows = list.reduce((acc, t) => acc + (t.rows > 0 ? t.rows : 0), 0)
    return {
      tableCount: list.length,
      totalRows,
    }
  }, [tables, selectedDb])

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-xs p-4 sm:p-6" onClick={onClose}>
      <div
        className="w-full max-w-4xl bg-surface-container-lowest rounded-2xl border border-outline-variant shadow-2xl flex flex-col max-h-[88vh] overflow-hidden animate-in fade-in zoom-in-95 duration-150"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Topbar */}
        <div className="px-6 py-4 border-b border-outline-variant flex items-center justify-between bg-surface-container-low/40">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-primary-container text-on-primary-container flex items-center justify-center font-bold text-xs shadow-xs border border-primary/20">
              {c.type.toUpperCase().slice(0, 3)}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-[16px] font-bold text-on-surface tracking-tight">{c.name}</h2>
                <span
                  className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-mono font-medium ${
                    c.status === 'CONNECTED'
                      ? 'bg-success-container text-on-success-container'
                      : 'bg-error-container text-on-error-container'
                  }`}
                >
                  <span className={`w-1.5 h-1.5 rounded-full ${c.status === 'CONNECTED' ? 'bg-success' : 'bg-error'}`} />
                  {c.status}
                </span>
              </div>
              <p className="text-[12px] font-mono text-outline mt-0.5">
                {c.host}:{c.port} &bull; via <span className="text-on-surface-variant font-sans font-medium">{agentName}</span>
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={onRefresh}
              disabled={refreshing}
              className="h-8 px-2.5 rounded-lg border border-outline-variant hover:bg-surface-container-high text-[12px] font-medium text-on-surface flex items-center gap-1.5 transition-colors disabled:opacity-50"
            >
              <RefreshCw size={12} className={refreshing ? 'animate-spin' : ''} />
              <span className="hidden sm:inline">Refresh Data</span>
            </button>
            <button
              type="button"
              onClick={onClose}
              className="p-1.5 rounded-lg hover:bg-surface-container-high text-on-surface-variant transition-colors"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex items-center px-6 border-b border-outline-variant bg-surface-container-lowest text-[13px] font-medium gap-6">
          <button
            type="button"
            onClick={() => setActiveTab('databases')}
            className={`py-3 relative flex items-center gap-2 transition-colors ${
              activeTab === 'databases' ? 'text-primary font-semibold' : 'text-outline hover:text-on-surface'
            }`}
          >
            <Database size={15} />
            <span>Databases &amp; Schemas</span>
            <span className="px-1.5 py-0.5 rounded-full bg-surface-container-high font-mono text-[10px]">
              {c.databases.length}
            </span>
            {activeTab === 'databases' && <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />}
          </button>

          <button
            type="button"
            onClick={() => setActiveTab('settings')}
            className={`py-3 relative flex items-center gap-2 transition-colors ${
              activeTab === 'settings' ? 'text-primary font-semibold' : 'text-outline hover:text-on-surface'
            }`}
          >
            <Server size={15} />
            <span>Connection Specs</span>
            {activeTab === 'settings' && <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />}
          </button>
        </div>

        {/* Modal Body */}
        <div className="flex-1 overflow-hidden flex flex-col">
          {activeTab === 'databases' ? (
            <div className="flex-1 grid grid-cols-1 md:grid-cols-12 min-h-0 divide-y md:divide-y-0 md:divide-x divide-outline-variant">
              {/* Left Column: Database Selection List */}
              <div className="md:col-span-5 flex flex-col min-h-0 bg-surface-container-lowest">
                <div className="p-3 border-b border-outline-variant bg-surface-container-low/30">
                  <div className="relative">
                    <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-outline pointer-events-none" />
                    <input
                      type="text"
                      placeholder="Filter databases..."
                      value={filter}
                      onChange={(e) => setFilter(e.target.value)}
                      className="w-full h-8 pl-8 pr-2.5 rounded-lg bg-surface-container-lowest border border-outline-variant text-[12px] font-mono text-on-surface placeholder:text-outline focus:outline-none focus:border-primary transition-colors"
                    />
                  </div>
                </div>

                <div className="flex-1 overflow-y-auto p-2 space-y-1">
                  {filteredDatabases.length === 0 ? (
                    <div className="py-8 text-center text-outline text-[12px]">
                      {c.databases.length === 0 ? 'No databases discovered' : 'No matching databases found'}
                    </div>
                  ) : (
                    filteredDatabases.map((db) => {
                      const isSelected = selectedDb === db.name
                      const isCopied = copiedDb === db.name
                      return (
                        <div
                          key={db.name}
                          onClick={() => void fetchTables(db.name)}
                          className={`group flex items-center justify-between p-2.5 rounded-lg cursor-pointer transition-all border ${
                            isSelected
                              ? 'bg-primary-container/25 border-primary/40 shadow-xs'
                              : 'bg-surface-container-lowest border-transparent hover:bg-surface-container-low hover:border-outline-variant'
                          }`}
                        >
                          <div className="flex items-center gap-2.5 min-w-0">
                            <Database
                              size={14}
                              className={`shrink-0 transition-colors ${isSelected ? 'text-primary' : 'text-outline group-hover:text-on-surface'}`}
                            />
                            <div className="min-w-0">
                              <p className={`font-mono text-[12px] truncate ${isSelected ? 'font-semibold text-on-surface' : 'text-on-surface'}`}>
                                {db.name}
                              </p>
                              <div className="flex items-center gap-2 font-mono text-[10px] text-outline mt-0.5">
                                <span>{fmtSize(db.size_bytes)}</span>
                                {db.table_count >= 0 && (
                                  <>
                                    <span>&bull;</span>
                                    <span>{db.table_count} {db.table_count === 1 ? 'entity' : 'entities'}</span>
                                  </>
                                )}
                              </div>
                            </div>
                          </div>

                          <button
                            type="button"
                            aria-label="Copy database name"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleCopy(db.name)
                            }}
                            className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-surface-container-high text-outline hover:text-on-surface transition-opacity"
                          >
                            {isCopied ? <Check size={12} className="text-success" /> : <Copy size={12} />}
                          </button>
                        </div>
                      )
                    })
                  )}
                </div>
              </div>

              {/* Right Column: Schema Inspector */}
              <div className="md:col-span-7 flex flex-col min-h-0 bg-surface-container-low/20">
                {selectedDb ? (
                  <>
                    {/* Selected DB Header Banner */}
                    <div className="p-4 border-b border-outline-variant bg-surface-container-lowest flex items-center justify-between">
                      <div>
                        <div className="flex items-center gap-1.5">
                          <span className="text-[11px] font-bold uppercase tracking-wider text-outline">Inspecting Schema</span>
                          <span className="text-[11px] text-outline">&bull;</span>
                          <span className="font-mono text-[12px] font-bold text-on-surface">{selectedDb}</span>
                        </div>
                        {selectedStats && (
                          <div className="flex items-center gap-3 mt-1 text-[11px] font-mono text-on-surface-variant">
                            <span className="flex items-center gap-1">
                              <Layers size={11} className="text-outline" /> {selectedStats.tableCount} tables/collections
                            </span>
                            <span className="flex items-center gap-1">
                              <Activity size={11} className="text-outline" /> ~{selectedStats.totalRows.toLocaleString()} total rows
                            </span>
                          </div>
                        )}
                      </div>

                      <button
                        type="button"
                        onClick={() => void fetchTables(selectedDb)}
                        disabled={loadingTables === selectedDb}
                        className="p-1.5 rounded-lg border border-outline-variant hover:bg-surface-container-high text-outline hover:text-on-surface transition-colors disabled:opacity-50"
                        title="Re-query schema live"
                      >
                        <RefreshCw size={12} className={loadingTables === selectedDb ? 'animate-spin' : ''} />
                      </button>
                    </div>

                    {/* Filter for Tables */}
                    <div className="px-4 py-2 border-b border-outline-variant bg-surface-container-low/50">
                      <div className="relative">
                        <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-outline pointer-events-none" />
                        <input
                          type="text"
                          placeholder={`Search ${selectedDb} entities...`}
                          value={tableFilter}
                          onChange={(e) => setTableFilter(e.target.value)}
                          className="w-full h-7 pl-7 pr-2 rounded bg-surface-container-lowest border border-outline-variant text-[11px] font-mono text-on-surface placeholder:text-outline focus:outline-none focus:border-primary"
                        />
                      </div>
                    </div>

                    {/* Table Viewport */}
                    <div className="flex-1 overflow-y-auto">
                      {loadingTables === selectedDb ? (
                        <div className="h-48 flex flex-col items-center justify-center gap-2 text-outline text-[12px]">
                          <Loader2 size={18} className="animate-spin text-primary" />
                          <span>Querying engine metadata catalog…</span>
                        </div>
                      ) : tablesError && !tables[selectedDb] ? (
                        <div className="p-6 text-center">
                          <AlertCircle size={20} className="mx-auto text-error mb-2" />
                          <p className="text-[12px] font-medium text-error">{tablesError}</p>
                          <button
                            type="button"
                            onClick={() => void fetchTables(selectedDb)}
                            className="mt-3 text-[11px] underline font-medium text-on-surface-variant hover:text-on-surface"
                          >
                            Retry live inspection
                          </button>
                        </div>
                      ) : (tables[selectedDb] ?? []).length === 0 ? (
                        <div className="h-48 flex flex-col items-center justify-center text-outline text-[12px]">
                          <Table2 size={24} className="stroke-1 mb-1.5 opacity-50" />
                          <span>No tables or collections found</span>
                        </div>
                      ) : (
                        <table className="w-full text-left border-collapse">
                          <thead className="sticky top-0 bg-surface-container-low text-[10px] uppercase font-semibold text-outline tracking-wider border-b border-outline-variant">
                            <tr>
                              <th className="py-2 px-4">Entity</th>
                              <th className="py-2 px-4 text-right">Rows</th>
                              <th className="py-2 px-4 text-right">Disk Size</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-outline-variant/40 font-mono text-[11px]">
                            {visibleTables.map((t) => (
                              <tr key={`${t.schema ?? ''}.${t.name}`} className="hover:bg-surface-container-lowest/80 transition-colors">
                                <td className="py-2 px-4 text-on-surface">
                                  <div className="flex items-center gap-1.5 truncate max-w-[220px]">
                                    <Table2 size={11} className="text-outline shrink-0" />
                                    <span className="truncate" title={t.name}>
                                      {t.schema ? <span className="text-outline">{t.schema}.</span> : null}
                                      {t.name}
                                    </span>
                                  </div>
                                </td>
                                <td className="py-2 px-4 text-right text-on-surface-variant">
                                  {t.rows >= 0 ? t.rows.toLocaleString() : '—'}
                                </td>
                                <td className="py-2 px-4 text-right text-outline">
                                  {t.size_bytes >= 0 ? fmtSize(t.size_bytes) : '—'}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </div>
                  </>
                ) : (
                  <div className="h-full flex items-center justify-center text-outline text-[12px]">
                    Select a database to view its schemas and tables
                  </div>
                )}
              </div>
            </div>
          ) : (
            /* Connection Settings / Specs Tab */
            <div className="p-6 overflow-y-auto space-y-6">
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
                <div className="p-3.5 rounded-xl border border-outline-variant bg-surface-container-low/40">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-outline">Engine</span>
                  <p className="text-[14px] font-bold text-on-surface capitalize mt-0.5">{c.type}</p>
                </div>
                <div className="p-3.5 rounded-xl border border-outline-variant bg-surface-container-low/40">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-outline">Port</span>
                  <p className="text-[14px] font-mono font-bold text-on-surface mt-0.5">{c.port}</p>
                </div>
                <div className="p-3.5 rounded-xl border border-outline-variant bg-surface-container-low/40">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-outline">Authenticated User</span>
                  <p className="text-[14px] font-mono font-medium text-on-surface truncate mt-0.5">
                    {c.username || '— (no auth)'}
                  </p>
                </div>
                <div className="p-3.5 rounded-xl border border-outline-variant bg-surface-container-low/40">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-outline">Agent Host</span>
                  <p className="text-[14px] font-semibold text-on-surface truncate mt-0.5 flex items-center gap-1.5">
                    <span className={`w-2 h-2 rounded-full ${agentOnline ? 'bg-success' : 'bg-outline'}`} />
                    {agentName}
                  </p>
                </div>
              </div>

              <div className="p-4 rounded-xl border border-outline-variant bg-surface-container-low/20 space-y-3">
                <div className="flex justify-between items-center text-[12px]">
                  <span className="text-outline">Raw Host Address</span>
                  <span className="font-mono text-on-surface">{c.host}</span>
                </div>
                <div className="flex justify-between items-center text-[12px] border-t border-outline-variant/50 pt-2">
                  <span className="text-outline">Last Health Check</span>
                  <span className="font-mono text-on-surface">
                    {c.last_checked_at ? new Date(c.last_checked_at).toLocaleString() : 'Never'}
                  </span>
                </div>
                <div className="flex justify-between items-center text-[12px] border-t border-outline-variant/50 pt-2">
                  <span className="text-outline">Discovered Databases</span>
                  <span className="font-mono text-on-surface">{c.databases.length} registered</span>
                </div>
              </div>

              <div className="p-3.5 rounded-lg border border-primary/20 bg-primary-container/10 flex items-start gap-2.5">
                <Lock size={15} className="text-primary shrink-0 mt-0.5" />
                <p className="text-[12px] text-on-surface-variant leading-relaxed">
                  Database access credentials are stored encrypted via hardware-backed envelope keys. They are never transmitted back to the browser.
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Modal Bottom Action Bar */}
        <div className="px-6 py-3 border-t border-outline-variant bg-surface-container-lowest flex items-center justify-between">
          <span className="text-[11px] text-outline hidden sm:flex items-center gap-1">
            <HardDrive size={12} />
            Total engine footprint: {fmtSize(c.databases.reduce((sum, d) => sum + (d.size_bytes > 0 ? d.size_bytes : 0), 0))}
          </span>

          <div className="flex items-center gap-2 ml-auto">
            <button
              type="button"
              onClick={onClose}
              className="px-4 h-8 rounded-lg border border-outline-variant text-[12px] font-medium text-on-surface hover:bg-surface-container-low transition-colors"
            >
              Close
            </button>
            <button
              type="button"
              onClick={onEdit}
              className="px-4 h-8 rounded-lg bg-primary text-on-primary text-[12px] font-semibold hover:opacity-90 flex items-center gap-1.5 transition-opacity"
            >
              <Pencil size={12} /> Edit Configuration
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default ConnectionViewModal