import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { jobApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { BackupJob, BackupSourceType } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import { SkeletonCard } from '@/components/Skeleton'
import SearchInput from '@/components/ui/SearchInput'
import Action3DButton from '@/components/ui/Action3DButton'
import {
  ChevronRight, Plus, Layers, CheckCircle, AlertCircle, PauseCircle,
  Cloud, Clock, History,
  Play, MoreVertical, Archive,
} from 'lucide-react'

const SOURCE_FILTERS: { label: string; value: BackupSourceType | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Filesystem', value: 'FILESYSTEM' },
  { label: 'MSSQL Server', value: 'MSSQL_SERVER' },
  { label: 'PostgreSQL', value: 'POSTGRES' },
  { label: 'DBF Dataset', value: 'DBF' },
]

import { FileSystemIcon, MssqlServerIcon, PostgresIcon, DbfIcon } from '@/components/ui/SourceIcons'

const sourceIcon = {
  FILESYSTEM: FileSystemIcon,
  MSSQL_SERVER: MssqlServerIcon,
  POSTGRES: PostgresIcon,
  DBF: DbfIcon,
}

const JobsPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [jobs, setJobs] = useState<BackupJob[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [sourceFilter, setSourceFilter] = useState<BackupSourceType | ''>('')

  const loadJobs = () => {
    if (!currentOrg) return
    setIsLoading(true)
    jobApi.list(currentOrg.id)
      .then(setJobs)
      .catch(() => addToast('error', 'Failed to load backup jobs'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { loadJobs() }, [currentOrg])

  const handleRunNow = async (e: React.MouseEvent, job: BackupJob) => {
    e.stopPropagation()
    if (!currentOrg) return
    try {
      await jobApi.runNow(currentOrg.id, job.id)
      addToast('success', `${job.name} started`)
    } catch {
      addToast('error', 'Failed to start backup')
    }
  }

  const handleToggle = async (e: React.MouseEvent, job: BackupJob) => {
    e.stopPropagation()
    if (!currentOrg) return
    try {
      if (job.enabled) await jobApi.disable(currentOrg.id, job.id)
      else await jobApi.enable(currentOrg.id, job.id)
      addToast('success', `${job.name} ${job.enabled ? 'disabled' : 'enabled'}`)
      loadJobs()
    } catch {
      addToast('error', 'Failed to update job')
    }
  }

  const filtered = jobs.filter((j) => {
    const matchSearch = !search || j.name.toLowerCase().includes(search.toLowerCase())
    const matchSource = !sourceFilter || j.source_type === sourceFilter
    return matchSearch && matchSource
  })

  const accentColor = (job: BackupJob) => {
    if (!job.enabled) return 'bg-[#c3c6d7]'
    return 'bg-[#006591]'
  }

  return (
    <div className="flex flex-col gap-5 pb-16">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-[12px] text-[#434655] mb-1">
            <span>VaultGuard</span>
            <ChevronRight size={14} />
            <span className="text-[#004ac6] font-medium">Backup Jobs</span>
          </div>
          <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight flex items-center gap-2.5">
            Backup Jobs
            <span className="px-2 py-0.5 rounded-full bg-[#e9edff] text-[#434655] text-[12px] font-medium">
              {jobs.length} Workloads
            </span>
          </h1>
          <p className="text-[13px] text-[#434655] mt-0.5">Configure, schedule, and monitor backup policies.</p>
        </div>
        <div className="flex items-center gap-2.5 shrink-0">
          <Action3DButton onClick={() => navigate('/jobs/new')}>
            <Plus size={16} />
            New Backup Job
          </Action3DButton>
        </div>
      </div>

      {/* Metric strip */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          { label: 'Total', value: jobs.length, Icon: Layers, color: 'text-[#004ac6]', bg: 'bg-[#e9edff]' },
          { label: 'Active', value: jobs.filter((j) => j.enabled).length, Icon: CheckCircle, color: 'text-[#006591]', bg: 'bg-[#c9e6ff]/50' },
          { label: 'Failed', value: 0, Icon: AlertCircle, color: 'text-[#ba1a1a]', bg: 'bg-[#ffdad6]/60' },
          { label: 'Paused', value: jobs.filter((j) => !j.enabled).length, Icon: PauseCircle, color: 'text-[#434655]', bg: 'bg-[#dce2f7]' },
        ].map((m) => (
          <div key={m.label} className="flex items-center justify-between p-3.5 rounded-xl bg-[#ffffff] shadow-sm border border-[#e9edff]">
            <div>
              <span className="text-[12px] text-[#737686]">{m.label}</span>
              <p className="text-[22px] font-bold text-[#141b2b] leading-tight">{m.value}</p>
            </div>
            <div className={`w-9 h-9 rounded-lg ${m.bg} flex items-center justify-center ${m.color}`}>
              <m.Icon size={18} />
            </div>
          </div>
        ))}
      </div>

      {/* Search + filters */}
      <div className="flex flex-col gap-3 p-3 rounded-xl bg-[#ffffff] shadow-sm border border-[#e9edff]">
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3">
          <div className="relative flex-1 min-w-[280px]">
            <SearchInput
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search jobs by name, path, or agent..."
              className="w-full lg:w-[400px]"
            />
          </div>
          <div className="flex items-center gap-1.5 overflow-x-auto">
            {SOURCE_FILTERS.map((f) => (
              <button
                key={f.value}
                type="button"
                onClick={() => setSourceFilter(f.value)}
                className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                  sourceFilter === f.value
                    ? 'bg-[#2563eb] text-white shadow-sm'
                    : 'bg-[#f1f3ff] text-[#434655] hover:bg-[#e9edff]'
                }`}
              >
                {f.label} {f.value === '' ? `(${jobs.length})` : `(${jobs.filter((j) => j.source_type === f.value).length})`}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Column headers */}
      <div className="hidden xl:grid grid-cols-12 gap-4 px-5 py-2 text-[12px] font-semibold uppercase tracking-wider text-[#434655]">
        <div className="col-span-4">Workload & Source</div>
        <div className="col-span-2">Agent & Storage</div>
        <div className="col-span-2">Schedule</div>
        <div className="col-span-2">Status</div>
        <div className="col-span-2 text-right">Actions</div>
      </div>

      {/* Job rows */}
      <div className="flex flex-col gap-2.5">
        {isLoading ? (
          [...Array(5)].map((_, i) => <SkeletonCard key={i} />)
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={Archive}
            title="No backup jobs"
            description="Create your first backup job to start protecting your data."
            action={
              <button
                type="button"
                onClick={() => navigate('/jobs/new')}
                className="px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors"
              >
                + New Backup Job
              </button>
            }
          />
        ) : filtered.map((job) => {
          const SourceIcon = sourceIcon[job.source_type]
          return (
            <div
              key={job.id}
              onClick={() => navigate(`/jobs/${job.id}`)}
              className="group relative flex flex-col xl:grid xl:grid-cols-12 gap-3 xl:gap-4 items-stretch xl:items-center p-4 rounded-xl bg-[#ffffff] shadow-sm hover:shadow-md transition-all duration-150 overflow-hidden cursor-pointer border border-[#e9edff]"
            >
              <div className={`absolute left-0 top-0 bottom-0 w-1.5 ${accentColor(job)}`} />

              {/* Policy info */}
              <div className="col-span-4 flex items-start gap-3 pl-2 min-w-0">
                <div className={`w-10 h-10 rounded-lg flex items-center justify-center shrink-0 mt-0.5 ${job.enabled ? 'bg-[#c9e6ff]/40 text-[#006591]' : 'bg-[#dce2f7] text-[#737686]'}`}>
                  <SourceIcon size={20} />
                </div>
                <div className="flex flex-col min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-[14px] font-semibold text-[#141b2b] truncate">{job.name}</span>
                    <span className="px-2 py-0.5 rounded bg-[#e9edff] text-[#434655] text-[11px] font-medium uppercase tracking-wide">
                      {job.source_type.replace('_', ' ')}
                    </span>
                    {job.encrypted && (
                      <span className="px-1.5 py-0.5 rounded bg-[#e1e0ff] text-[#2f2ebe] text-[10px] font-semibold uppercase">
                        AES-256
                      </span>
                    )}
                  </div>
                  <p className="font-mono text-[11px] text-[#434655] mt-0.5 truncate">
                    {job.source_path || job.source_database || '—'}
                  </p>
                </div>
              </div>

              {/* Agent & storage */}
              <div className="col-span-2 flex flex-col gap-1 min-w-0">
                <div className="flex items-center gap-1.5 font-mono text-[12px] text-[#141b2b]">
                  <span className={`w-2 h-2 rounded-full shrink-0 ${job.agent?.status === 'ONLINE' ? 'bg-[#006591]' : 'bg-[#737686]'}`} />
                  <span className="font-medium truncate">{job.agent?.name ?? '—'}</span>
                </div>
                <div className="flex items-center gap-1 text-[#434655] text-[11px] truncate">
                  <Cloud size={12} />
                  <span className="truncate">{job.storage_target?.name ?? '—'}</span>
                </div>
              </div>

              {/* Schedule */}
              <div className="col-span-2 flex flex-col gap-1 min-w-0">
                <div className="flex items-center gap-1.5 text-[13px] text-[#141b2b]">
                  <Clock size={14} className="text-[#434655] shrink-0" />
                  <span className="truncate">{job.schedule?.cron_expr ?? 'No schedule'}</span>
                </div>
                <div className="flex items-center gap-1.5 text-[11px] text-[#434655]">
                  <History size={12} />
                  <span>{job.retention_days}d retention</span>
                </div>
              </div>

              {/* Status */}
              <div className="col-span-2 flex flex-col gap-1">
                <StatusBadge status={job.enabled ? 'ONLINE' : 'OFFLINE'} size="sm" />
                <span className="text-[11px] text-[#737686]">
                  {job.mode === 'COMPRESSED' ? 'Zstandard' : 'Normal'} mode
                </span>
              </div>

              {/* Actions */}
              <div className="col-span-2 flex items-center justify-end gap-2 shrink-0" onClick={(e) => e.stopPropagation()}>
                <Action3DButton
                  onClick={(e) => handleRunNow(e, job)}
                  className="!px-2.5 !h-8 !text-[12px]"
                >
                  <Play size={14} className="text-white" />
                  Run Now
                </Action3DButton>
                <label className="relative inline-flex items-center cursor-pointer" title={job.enabled ? 'Disable' : 'Enable'}>
                  <input
                    type="checkbox"
                    checked={job.enabled}
                    onChange={(e) => handleToggle(e as unknown as React.MouseEvent, job)}
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-[#dce2f7] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[#2563eb]" />
                </label>
                <button
                  type="button"
                  onClick={(e) => { e.stopPropagation(); navigate(`/jobs/${job.id}`) }}
                  className="p-1.5 rounded-lg text-[#434655] hover:text-[#141b2b] hover:bg-[#e9edff] transition-colors"
                >
                  <MoreVertical size={18} />
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default JobsPage
