import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Pencil,
  RefreshCw,
  Loader2,
  Lock,
  Database,
  Search,
  Copy,
  Check,
  Table2,
  Layers,
  Server,
  Activity,
  AlertCircle,
  ChevronRight,
} from 'lucide-react'
import { agentApi, connectionsApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, DatabaseConnection, TableInfo } from '@/types'
import { formatBytes } from '@/utils/format'
import { DbTypeIcon } from '@/components/ui/SourceIcons'
import { typeLabel } from '@/components/ui/typeLabel'
import { badgeDotClass, badgeToneClass, connectionBadge, agentLifecycleOf } from '@/utils/connectionStatus'
import { Page, PageHeader } from '@/components/Page'
import ConnectionModal from './ConnectionModal'

type TabType = 'databases' | 'settings'

// Full-page connection detail (replaces the old modal): same database
// explorer and spec sheets, but reachable by URL, back-button friendly, and
// with room to breathe on wide screens.
const ConnectionDetailPage = () => {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()

  const [conn, setConn] = useState<DatabaseConnection | null>(null)
  const [agents, setAgents] = useState<Agent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [showEdit, setShowEdit] = useState(false)

  const [activeTab, setActiveTab] = useState<TabType>('databases')
  const [filter, setFilter] = useState('')
  const [tableFilter, setTableFilter] = useState('')
  const [copiedDb, setCopiedDb] = useState<string | null>(null)
  const [selectedDb, setSelectedDb] = useState<string | null>(null)
  const [tables, setTables] = useState<Record<string, TableInfo[]>>({})
  const [loadingTables, setLoadingTables] = useState<string | null>(null)
  const [tablesError, setTablesError] = useState<string | null>(null)

  const load = useCallback(() => {
    if (!currentOrg || !id) return
    setIsLoading(true)
    setLoadError(null)
    Promise.all([
      connectionsApi.get(currentOrg.id, id),
      agentApi.list(currentOrg.id, { limit: 200 }),
    ])
      .then(([c, a]) => {
        setConn(c)
        setAgents(a.data)
        setSelectedDb((prev) => prev ?? c.databases[0]?.name ?? null)
      })
      .catch((e: unknown) => setLoadError(e instanceof Error ? e.message : 'Connection not found.'))
      .finally(() => setIsLoading(false))
  }, [currentOrg, id])

  useEffect(() => { load() }, [load])

  const c = conn
  const agentName = c ? (c.agent?.name ?? agents.find((a) => a.id === c.agent_id)?.name ?? '—') : '—'
  const agentObj = c ? (c.agent ?? agents.find((a) => a.id === c.agent_id)) : undefined
  const lifecycle = c ? (c.agent_lifecycle ?? agentLifecycleOf(agentObj)) : undefined
  const agentOnline = lifecycle ? lifecycle === 'ONLINE' : agentObj?.status === 'ONLINE'
  // Derived badge: leads with agent lifecycle so a stale "Connected" from the
  // last server-side check can't mask a pending/offline/revoked agent.
  const badge = c ? connectionBadge(c) : null

  const filteredDatabases = useMemo(() => {
    if (!c || !filter.trim()) return c?.databases ?? []
    const q = filter.toLowerCase().trim()
    return c.databases.filter((db) => db.name.toLowerCase().includes(q))
  }, [c, filter])

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
    if (tables[dbName] || !currentOrg || !c) return

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

  const handleRefresh = async () => {
    if (!currentOrg || !c) return
    setRefreshing(true)
    try {
      const res = await connectionsApi.databases(currentOrg.id, c.id)
      setConn({ ...c, databases: res.databases, database_count: res.databases.length })
      addToast('success', res.cached ? `Using cached list (${res.databases.length} databases)` : `Found ${res.databases.length} databases`)
    } catch {
      addToast('error', 'Failed to refresh databases')
    } finally {
      setRefreshing(false)
    }
  }

  const selectedStats = useMemo(() => {
    const list = selectedDb ? tables[selectedDb] : null
    if (!list) return null
    const totalRows = list.reduce((acc, t) => acc + (t.rows > 0 ? t.rows : 0), 0)
    return { tableCount: list.length, totalRows }
  }, [tables, selectedDb])

  if (isLoading) {
    return (
      <Page>
        <div className="flex items-center justify-center py-24 text-on-surface-variant">
          <Loader2 size={24} className="animate-spin" />
        </div>
      </Page>
    )
  }

  if (loadError || !c) {
    return (
      <Page>
        <PageHeader
          eyebrow={
            <nav className="flex items-center gap-1.5 text-[12px] text-on-surface-variant mb-1">
              <Link to="/connections" className="hover:text-primary">Connections</Link>
              <ChevronRight size={12} />
            </nav>
          }
          title="Connection not found"
          description={loadError ?? 'This connection may have been deleted.'}
          actions={
            <button
              type="button"
              onClick={() => navigate('/connections')}
              className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors"
            >
              Back to connections
            </button>
          }
        />
      </Page>
    )
  }

  const tabCls = (active: boolean) =>
    `py-3 relative flex items-center gap-2 transition-colors ${active ? 'text-primary font-semibold' : 'text-on-surface-variant hover:text-on-surface'}`

  return (
    <Page>
      <PageHeader
        eyebrow={
          <nav className="flex items-center gap-1.5 text-[12px] text-on-surface-variant mb-1">
            <Link to="/connections" className="hover:text-primary">Connections</Link>
            <ChevronRight size={12} />
            <span className="text-on-surface font-medium">{c.name}</span>
          </nav>
        }
        title={
          <span className="flex items-center gap-2.5">
            <span className="w-8 h-8 rounded-lg bg-primary-container text-on-primary-container flex items-center justify-center border border-primary/20">
              <DbTypeIcon type={c.type} size={17} />
            </span>
            {c.name}
            {badge && (
              <span
                title={badge.hint}
                className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium ${badgeToneClass[badge.tone]}`}
              >
                <span className={`w-1.5 h-1.5 rounded-full ${badgeDotClass[badge.tone]}`} />
                {badge.label}
              </span>
            )}
          </span>
        }
        description={`${typeLabel(c.type)} · ${c.host}:${c.port} · via ${agentName}${badge && !badge.trusted ? ` — ${badge.label.toLowerCase()}` : ''} · ${c.databases.length} databases`}
        actions={
          <>
            <button
              type="button"
              onClick={() => void handleRefresh()}
              disabled={refreshing}
              className="h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-lowest text-[13px] font-medium text-on-surface flex items-center gap-1.5 hover:bg-surface-container-high transition-colors disabled:opacity-50"
            >
              <RefreshCw size={13} className={refreshing ? 'animate-spin' : ''} />
              Refresh data
            </button>
            <button
              type="button"
              onClick={() => setShowEdit(true)}
              className="h-9 px-4 rounded-lg bg-primary text-on-primary text-[13px] font-semibold flex items-center gap-1.5 hover:opacity-90 transition-opacity"
            >
              <Pencil size={13} /> Edit
            </button>
          </>
        }
      />

      <div className="flex-1 min-h-0 flex flex-col rounded-xl border border-surface-variant bg-surface-container-lowest shadow-sm overflow-hidden">
        {/* Tab Navigation */}
        <div className="flex items-center px-5 border-b border-surface-variant text-[13px] font-medium gap-6 shrink-0">
          <button type="button" onClick={() => setActiveTab('databases')} className={tabCls(activeTab === 'databases')}>
            <Database size={15} />
            <span>Databases &amp; Schemas</span>
            <span className="px-1.5 py-0.5 rounded-full bg-surface-container-high font-mono text-[10px]">
              {c.databases.length}
            </span>
            {activeTab === 'databases' && <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />}
          </button>
          <button type="button" onClick={() => setActiveTab('settings')} className={tabCls(activeTab === 'settings')}>
            <Server size={15} />
            <span>Connection Specs</span>
            {activeTab === 'settings' && <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />}
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
          {activeTab === 'databases' ? (
            <div className="flex-1 min-h-0 grid grid-cols-1 md:grid-cols-12 divide-y md:divide-y-0 md:divide-x divide-surface-variant">
              {/* Left: database list */}
              <div className="md:col-span-5 flex flex-col min-h-0">
                <div className="p-3 border-b border-surface-variant bg-surface-container-low/40">
                  <div className="relative">
                    <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-on-surface-variant pointer-events-none" />
                    <input
                      type="text"
                      placeholder="Filter databases..."
                      value={filter}
                      onChange={(e) => setFilter(e.target.value)}
                      className="w-full h-8 pl-8 pr-2.5 rounded-lg bg-surface-container-lowest border border-surface-variant text-[12px] font-mono text-on-surface placeholder:text-on-surface-variant focus:outline-none focus:border-primary transition-colors"
                    />
                  </div>
                </div>

                <div className="flex-1 overflow-y-auto p-2 space-y-1">
                  {filteredDatabases.length === 0 ? (
                    <div className="py-8 text-center text-on-surface-variant text-[12px]">
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
                              : 'bg-surface-container-lowest border-transparent hover:bg-surface-container-low hover:border-surface-variant'
                          }`}
                        >
                          <div className="flex items-center gap-2.5 min-w-0">
                            <Database
                              size={14}
                              className={`shrink-0 transition-colors ${isSelected ? 'text-primary' : 'text-on-surface-variant group-hover:text-on-surface'}`}
                            />
                            <div className="min-w-0">
                              <p className={`font-mono text-[12px] truncate ${isSelected ? 'font-semibold text-on-surface' : 'text-on-surface'}`}>
                                {db.name}
                              </p>
                              <div className="flex items-center gap-2 font-mono text-[10px] text-on-surface-variant mt-0.5">
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
                            className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-opacity"
                          >
                            {isCopied ? <Check size={12} className="text-success" /> : <Copy size={12} />}
                          </button>
                        </div>
                      )
                    })
                  )}
                </div>
              </div>

              {/* Right: schema inspector */}
              <div className="md:col-span-7 flex flex-col min-h-0 bg-surface-container-low/20">
                {selectedDb ? (
                  <>
                    <div className="p-4 border-b border-surface-variant bg-surface-container-lowest flex items-center justify-between gap-3 shrink-0">
                      <div className="min-w-0">
                        <div className="flex items-center gap-1.5">
                          <span className="text-[11px] font-bold uppercase tracking-wider text-on-surface-variant">Inspecting Schema</span>
                          <span className="text-[11px] text-on-surface-variant">&bull;</span>
                          <span className="font-mono text-[12px] font-bold text-on-surface truncate">{selectedDb}</span>
                        </div>
                        {selectedStats && (
                          <div className="flex items-center gap-3 mt-1 text-[11px] font-mono text-on-surface-variant">
                            <span className="flex items-center gap-1">
                              <Layers size={11} className="text-on-surface-variant" /> {selectedStats.tableCount} tables/collections
                            </span>
                            <span className="flex items-center gap-1">
                              <Activity size={11} className="text-on-surface-variant" /> ~{selectedStats.totalRows.toLocaleString()} total rows
                            </span>
                          </div>
                        )}
                      </div>

                      <button
                        type="button"
                        onClick={() => void fetchTables(selectedDb)}
                        disabled={loadingTables === selectedDb}
                        className="p-1.5 rounded-lg border border-surface-variant hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors disabled:opacity-50"
                        title="Re-query schema live"
                      >
                        <RefreshCw size={12} className={loadingTables === selectedDb ? 'animate-spin' : ''} />
                      </button>
                    </div>

                    <div className="px-4 py-2 border-b border-surface-variant bg-surface-container-low/50 shrink-0">
                      <div className="relative">
                        <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-on-surface-variant pointer-events-none" />
                        <input
                          type="text"
                          placeholder={`Search ${selectedDb} entities...`}
                          value={tableFilter}
                          onChange={(e) => setTableFilter(e.target.value)}
                          className="w-full h-7 pl-7 pr-2 rounded bg-surface-container-lowest border border-surface-variant text-[11px] font-mono text-on-surface placeholder:text-on-surface-variant focus:outline-none focus:border-primary"
                        />
                      </div>
                    </div>

                    <div className="flex-1 overflow-y-auto">
                      {loadingTables === selectedDb ? (
                        <div className="h-48 flex flex-col items-center justify-center gap-2 text-on-surface-variant text-[12px]">
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
                        <div className="h-48 flex flex-col items-center justify-center text-on-surface-variant text-[12px]">
                          <Table2 size={24} className="stroke-1 mb-1.5 opacity-50" />
                          <span>No tables or collections found</span>
                        </div>
                      ) : (
                        <table className="w-full text-left border-collapse">
                          <thead className="sticky top-0 bg-surface-container-low text-[10px] uppercase font-semibold text-on-surface-variant tracking-wider border-b border-surface-variant">
                            <tr>
                              <th className="py-2 px-4">Entity</th>
                              <th className="py-2 px-4 text-right">Rows</th>
                              <th className="py-2 px-4 text-right">Disk Size</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-surface-variant/40 font-mono text-[11px]">
                            {visibleTables.map((t) => (
                              <tr key={`${t.schema ?? ''}.${t.name}`} className="hover:bg-surface-container-lowest/80 transition-colors">
                                <td className="py-2 px-4 text-on-surface">
                                  <div className="flex items-center gap-1.5 truncate max-w-[280px]">
                                    <Table2 size={11} className="text-on-surface-variant shrink-0" />
                                    <span className="truncate" title={t.name}>
                                      {t.schema ? <span className="text-on-surface-variant">{t.schema}.</span> : null}
                                      {t.name}
                                    </span>
                                  </div>
                                </td>
                                <td className="py-2 px-4 text-right text-on-surface-variant">
                                  {t.rows >= 0 ? t.rows.toLocaleString() : '—'}
                                </td>
                                <td className="py-2 px-4 text-right text-on-surface-variant">
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
                  <div className="h-full flex items-center justify-center text-on-surface-variant text-[12px]">
                    Select a database to view its schemas and tables
                  </div>
                )}
              </div>
            </div>
          ) : (
            /* Connection Specs */
            <div className="flex-1 overflow-y-auto p-5 space-y-5">
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
                <div className="p-3.5 rounded-xl border border-surface-variant bg-surface-container-low/40">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-on-surface-variant">Engine</span>
                  <p className="flex items-center gap-2 text-[14px] font-bold text-on-surface capitalize mt-0.5">
                    <DbTypeIcon type={c.type} size={16} />
                    {typeLabel(c.type)}
                  </p>
                </div>
                <div className="p-3.5 rounded-xl border border-surface-variant bg-surface-container-low/40">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-on-surface-variant">Port</span>
                  <p className="text-[14px] font-mono font-bold text-on-surface mt-0.5">{c.port}</p>
                </div>
                <div className="p-3.5 rounded-xl border border-surface-variant bg-surface-container-low/40">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-on-surface-variant">Authenticated User</span>
                  <p className="text-[14px] font-mono font-medium text-on-surface truncate mt-0.5">
                    {c.username || '— (no auth)'}
                  </p>
                </div>
                <div className="p-3.5 rounded-xl border border-surface-variant bg-surface-container-low/40">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-on-surface-variant">Agent Host</span>
                  <p className="text-[14px] font-semibold text-on-surface truncate mt-0.5 flex items-center gap-1.5">
                    <span className={`w-2 h-2 rounded-full ${agentOnline ? 'bg-success' : 'bg-surface-variant'}`} />
                    {agentName}
                  </p>
                </div>
              </div>

              <div className="p-4 rounded-xl border border-surface-variant bg-surface-container-low/20 space-y-3">
                <div className="flex justify-between items-center text-[12px]">
                  <span className="text-on-surface-variant">Raw Host Address</span>
                  <span className="font-mono text-on-surface">{c.host}</span>
                </div>
                <div className="flex justify-between items-center text-[12px] border-t border-surface-variant/50 pt-2">
                  <span className="text-on-surface-variant">Last Health Check</span>
                  <span className="font-mono text-on-surface">
                    {c.last_checked_at ? new Date(c.last_checked_at).toLocaleString() : 'Never'}
                  </span>
                </div>
                <div className="flex justify-between items-center text-[12px] border-t border-surface-variant/50 pt-2">
                  <span className="text-on-surface-variant">Discovered Databases</span>
                  <span className="font-mono text-on-surface">{c.databases.length} registered</span>
                </div>
                <div className="flex justify-between items-center text-[12px] border-t border-surface-variant/50 pt-2">
                  <span className="text-on-surface-variant">Total engine footprint</span>
                  <span className="font-mono text-on-surface">
                    {fmtSize(c.databases.reduce((sum, d) => sum + (d.size_bytes > 0 ? d.size_bytes : 0), 0))}
                  </span>
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
      </div>

      {showEdit && (
        <ConnectionModal
          agents={agents}
          editing={c}
          onClose={() => setShowEdit(false)}
          onSaved={() => { setShowEdit(false); load() }}
        />
      )}
    </Page>
  )
}

export default ConnectionDetailPage
