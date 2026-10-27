import type { Agent, DatabaseConnection } from '@/types'
import { FolderSearch } from 'lucide-react'
import ConnectionDatabasePicker from '../ConnectionDatabasePicker'
import { FileSystemIcon, MssqlServerIcon, PostgresIcon, MongoDbIcon, DbfIcon } from '@/components/ui/SourceIcons'
import type { JobWizardFormData } from './types'

interface WizardSourceStepProps {
  form: JobWizardFormData
  update: (key: string, value: unknown) => void
  visibleConnections: DatabaseConnection[]
  agents: Agent[]
  handleConnectionChange: (id: string) => void
  loadResources: () => void
  onOpenBrowser: () => void
}

const SOURCE_OPTIONS = [
  { type: 'FILESYSTEM', Icon: FileSystemIcon, label: 'Filesystem' },
  { type: 'MSSQL_SERVER', Icon: MssqlServerIcon, label: 'SQL Server' },
  { type: 'POSTGRES', Icon: PostgresIcon, label: 'PostgreSQL' },
  { type: 'MONGODB', Icon: MongoDbIcon, label: 'MongoDB' },
  { type: 'DBF', Icon: DbfIcon, label: 'DBF Dataset' },
] as const

const EXPORT_FORMAT_OPTIONS = [
  { v: 'ARCHIVE', t: 'Archive', d: 'Binary, full fidelity' },
  { v: 'JSON', t: 'JSON', d: 'Readable, per-doc lines' },
  { v: 'CSV', t: 'CSV', d: 'Spreadsheet friendly' },
] as const

export default function WizardSourceStep({
  form,
  update,
  visibleConnections,
  agents,
  handleConnectionChange,
  loadResources,
  onOpenBrowser,
}: WizardSourceStepProps) {
  const currentAgent = agents.find((a) => a.id === (form.agent_id || visibleConnections.find((c) => c.id === form.connection_id)?.agent_id))

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-[16px] font-semibold text-on-surface">Source Configuration</h2>
      <div>
        <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Job Name *</label>
        <input
          type="text"
          value={form.name}
          onChange={(e) => update('name', e.target.value)}
          placeholder="e.g. Daily ERP Backup"
          className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
        />
      </div>
      <div>
        <label className="block text-[13px] font-medium text-on-surface-variant mb-2">Source Type *</label>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
          {SOURCE_OPTIONS.map((s) => (
            <button
              key={s.type}
              type="button"
              onClick={() => { update('source_type', s.type); update('connection_id', ''); update('source_database', '') }}
              className={`flex flex-col items-center gap-2 p-4 rounded-xl border-2 transition-all ${
                form.source_type === s.type
                  ? 'border-primary bg-primary-container/30'
                  : 'border-surface-variant hover:border-outline'
              }`}
            >
              <s.Icon size={28} className={form.source_type === s.type ? 'text-primary' : 'text-on-surface-variant'} />
              <span className="text-[13px] font-medium text-on-surface">{s.label}</span>
            </button>
          ))}
        </div>
      </div>

      {form.source_type === 'FILESYSTEM' || form.source_type === 'DBF' ? (
        <div>
          <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">
            {form.source_type === 'DBF' ? 'Dataset Directory *' : 'Source Path *'}
          </label>
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={form.source_path}
              onChange={(e) => update('source_path', e.target.value)}
              placeholder={form.source_type === 'DBF' ? 'C:\\LegacyApp\\Data\\' : 'D:\\CompanyData\\'}
              className="flex-1 h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
            />
            <button
              type="button"
              onClick={onOpenBrowser}
              title="Browse folders on this machine"
              className="shrink-0 flex items-center gap-2 px-3 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-high transition-colors"
            >
              <FolderSearch size={16} />
              Browse
            </button>
          </div>
          <p className="mt-1.5 text-[11px] text-on-surface-variant">
            Type a path, or browse folders on this machine (works when the agent runs here).
          </p>
        </div>
      ) : form.source_type === 'MONGODB' ? (
        <div className="flex flex-col gap-4">
          <ConnectionDatabasePicker
            connections={visibleConnections}
            connectionId={form.connection_id}
            database={form.source_database}
            agent={currentAgent}
            onConnectionChange={handleConnectionChange}
            onDatabaseChange={(v) => update('source_database', v)}
            onRefreshConnections={loadResources}
          />
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-2">Export Format</label>
            <div className="grid grid-cols-3 gap-2">
              {EXPORT_FORMAT_OPTIONS.map((o) => (
                <button
                  key={o.v}
                  type="button"
                  onClick={() => update('export_format', o.v)}
                  className={`flex flex-col items-start gap-0.5 p-3 rounded-xl border-2 text-left transition-all ${
                    form.export_format === o.v
                      ? 'border-primary bg-primary-container/20'
                      : 'border-surface-variant hover:border-outline'
                  }`}
                >
                  <span className="text-[13px] font-semibold text-on-surface">{o.t}</span>
                  <span className="text-[11px] text-on-surface-variant">{o.d}</span>
                </button>
              ))}
            </div>
            {form.export_format === 'CSV' && (
              <p className="text-[11px] text-on-surface-variant mt-1.5">
                CSV restores values as strings — use Archive or JSON for exact restores.
              </p>
            )}
          </div>
        </div>
      ) : (
        <ConnectionDatabasePicker
          connections={visibleConnections}
          connectionId={form.connection_id}
          database={form.source_database}
          agent={currentAgent}
          onConnectionChange={handleConnectionChange}
          onDatabaseChange={(v) => update('source_database', v)}
          onRefreshConnections={loadResources}
        />
      )}

      {form.source_type === 'FILESYSTEM' && (
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Include Patterns</label>
            <textarea
              value={form.include_patterns}
              onChange={(e) => update('include_patterns', e.target.value)}
              placeholder="*.xlsx&#10;*.pdf&#10;*.docx"
              rows={3}
              className="w-full px-3 py-2 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[12px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-none"
            />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Exclude Patterns</label>
            <textarea
              value={form.exclude_patterns}
              onChange={(e) => update('exclude_patterns', e.target.value)}
              placeholder="*.tmp&#10;*.log&#10;node_modules/"
              rows={3}
              className="w-full px-3 py-2 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[12px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-none"
            />
          </div>
        </div>
      )}
    </div>
  )
}
