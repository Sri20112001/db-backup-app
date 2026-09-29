import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { jobApi } from "@/services/api";
import { useAuthStore } from "@/store/authStore";
import { useUIStore } from "@/store/uiStore";
import type { BackupJob, BackupSourceType } from "@/types";
import StatusBadge from "@/components/StatusBadge";
import EmptyState from "@/components/EmptyState";
import Pagination from "@/components/Pagination";
import { usePagination } from "@/hooks/usePagination";
import EditJobModal from "./EditJobModal";
import NewJobModal from "./NewJobModal";
import { useRealtimeStore } from "@/stores/realtimeStore";
import { SkeletonCard } from "@/components/Skeleton";
import SearchInput from "@/components/ui/SearchInput";
import SortSelect from "@/components/ui/SortSelect";
import Action3DButton from "@/components/ui/Action3DButton";
import {
  ChevronRight,
  Plus,
  Layers,
  CheckCircle,
  AlertCircle,
  PauseCircle,
  Cloud,
  Clock,
  History,
  Play,
  MoreVertical,
  Archive,
  Pencil,
} from "lucide-react";
import normalizeWindowsPath from './../../../utils/normalizeWindowsPath';


const SOURCE_FILTERS: { label: string; value: BackupSourceType | "" }[] = [
  { label: "All", value: "" },
  { label: "Filesystem", value: "FILESYSTEM" },
  { label: "MSSQL Server", value: "MSSQL_SERVER" },
  { label: "PostgreSQL", value: "POSTGRES" },
  { label: "MongoDB", value: "MONGODB" },
  { label: "DBF Dataset", value: "DBF" },
];

import {
  FileSystemIcon,
  MssqlServerIcon,
  PostgresIcon,
  MongoDbIcon,
  DbfIcon,
} from "@/components/ui/SourceIcons";

const sourceIcon = {
  FILESYSTEM: FileSystemIcon,
  MSSQL_SERVER: MssqlServerIcon,
  POSTGRES: PostgresIcon,
  MONGODB: MongoDbIcon,
  DBF: DbfIcon,
};

const JobsPage = () => {
  const { currentOrg } = useAuthStore();
  const { addToast } = useUIStore();
  const navigate = useNavigate();
  const [jobs, setJobs] = useState<BackupJob[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [sourceFilter, setSourceFilter] = useState<BackupSourceType | "">("");
  const [sort, setSort] = useState("name-asc");
  const [editId, setEditId] = useState<string | null>(null);
  const [showNew, setShowNew] = useState(false);

  const loadJobs = () => {
    if (!currentOrg) return;
    setIsLoading(true);
    jobApi
      .list(currentOrg.id)
      // Normalize the legacy 'SQL_SERVER' spelling stored by older jobs so
      // icons, filters, and counts treat them as MSSQL_SERVER.
      .then((list) =>
        setJobs(
          list.map((j) =>
            (j.source_type as string) === "SQL_SERVER"
              ? { ...j, source_type: "MSSQL_SERVER" as BackupSourceType }
              : j,
          ),
        ),
      )
      .catch(() => addToast("error", "Failed to load backup jobs"))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadJobs();
  }, [currentOrg]);

  // Cross-operator job changes refresh the list.
  const jobsSeq = useRealtimeStore((s) => s.entitySeq.jobs);
  useEffect(() => {
    loadJobs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [jobsSeq]);

  const handleRunNow = async (e: React.MouseEvent, job: BackupJob) => {
    e.stopPropagation();
    if (!currentOrg) return;
    try {
      await jobApi.runNow(currentOrg.id, job.id);
      addToast("success", `${job.name} started`);
    } catch {
      addToast("error", "Failed to start backup");
    }
  };

  const handleToggle = async (e: React.SyntheticEvent, job: BackupJob) => {
    e.stopPropagation();
    if (!currentOrg) return;
    try {
      if (job.enabled) await jobApi.disable(currentOrg.id, job.id);
      else await jobApi.enable(currentOrg.id, job.id);
      addToast(
        "success",
        `${job.name} ${job.enabled ? "disabled" : "enabled"}`,
      );
      loadJobs();
    } catch {
      addToast("error", "Failed to update job");
    }
  };

  const filtered = jobs.filter((j) => {
    const q = search.toLowerCase();
    const matchSearch =
      !search ||
      j.name.toLowerCase().includes(q) ||
      (j.source_path || "").toLowerCase().includes(q) ||
      (j.agent?.name || "").toLowerCase().includes(q);
    const matchSource = !sourceFilter || j.source_type === sourceFilter;
    return matchSearch && matchSource;
  });
  const sorted = [...filtered].sort((a, b) => {
    switch (sort) {
      case "name-desc":
        return b.name.localeCompare(a.name);
      case "enabled-first":
        return Number(b.enabled) - Number(a.enabled) || a.name.localeCompare(b.name);
      case "disabled-first":
        return Number(a.enabled) - Number(b.enabled) || a.name.localeCompare(b.name);
      default:
        return a.name.localeCompare(b.name);
    }
  });
  const paged = usePagination(
    sorted,
    8,
    `${search}|${sourceFilter}|${sort}|${currentOrg?.id ?? ""}`,
  );

  const accentColor = (job: BackupJob) => {
    if (!job.enabled) return "bg-[#c3c6d7]";
    return "bg-primary";
  };

  return (
    <div className="h-full min-h-0 flex flex-col gap-4 pb-1">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-[12px] text-on-surface-variant mb-1">
            <span>VaultGuard</span>
            <ChevronRight size={14} />
            <span className="text-primary font-medium">Backup Jobs</span>
          </div>
          <h1 className="text-[20px] font-semibold text-on-surface tracking-tight flex items-center gap-2.5">
            Backup Jobs
            <span className="px-2 py-0.5 rounded-full bg-surface-container-high text-on-surface-variant text-[12px] font-medium">
              {jobs.length} Workloads
            </span>
          </h1>
          <p className="text-[13px] text-on-surface-variant mt-0.5">
            Configure, schedule, and monitor backup policies.
          </p>
        </div>
        <div className="flex items-center gap-2.5 shrink-0">
          <Action3DButton onClick={() => setShowNew(true)}>
            <Plus size={16} />
            New Backup Job
          </Action3DButton>
        </div>
      </div>

      {/* Metric strip */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          {
            label: "Total",
            value: jobs.length,
            Icon: Layers,
            color: "text-primary",
            bg: "bg-surface-container-high",
          },
          {
            label: "Active",
            value: jobs.filter((j) => j.enabled).length,
            Icon: CheckCircle,
            color: "text-on-primary-container",
            bg: "bg-primary-container/50",
          },
          {
            label: "Failed",
            value: 0,
            Icon: AlertCircle,
            color: "text-error",
            bg: "bg-error-container/60",
          },
          {
            label: "Paused",
            value: jobs.filter((j) => !j.enabled).length,
            Icon: PauseCircle,
            color: "text-on-surface-variant",
            bg: "bg-surface-variant",
          },
        ].map((m) => (
          <div
            key={m.label}
            className="flex items-center justify-between p-3.5 rounded-xl bg-surface-container-lowest shadow-sm border border-surface-variant"
          >
            <div>
              <span className="text-[12px] text-outline">{m.label}</span>
              <p className="text-[22px] font-bold text-on-surface leading-tight">
                {m.value}
              </p>
            </div>
            <div
              className={`w-9 h-9 rounded-lg ${m.bg} flex items-center justify-center ${m.color}`}
            >
              <m.Icon size={18} />
            </div>
          </div>
        ))}
      </div>

      {/* Search + filters */}
      <div className="flex flex-col gap-3 p-3 rounded-xl bg-surface-container-lowest shadow-sm border border-surface-variant">
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3">
          <div className="relative flex-1 min-w-[280px]">
            <SearchInput
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search jobs by name, path, or agent..."
              className="w-full"
            />
          </div>
          <div className="flex items-center gap-1.5 overflow-x-auto">
            <SortSelect
              value={sort}
              onChange={setSort}
              options={[
                { label: "Name A–Z", value: "name-asc" },
                { label: "Name Z–A", value: "name-desc" },
                { label: "Enabled first", value: "enabled-first" },
                { label: "Disabled first", value: "disabled-first" },
              ]}
            />
            {SOURCE_FILTERS.map((f) => (
              <button
                key={f.value}
                type="button"
                onClick={() => setSourceFilter(f.value)}
                className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                  sourceFilter === f.value
                    ? "bg-primary text-on-primary shadow-sm"
                    : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
                }`}
              >
                {f.label}{" "}
                {f.value === ""
                  ? `(${jobs.length})`
                  : `(${jobs.filter((j) => j.source_type === f.value).length})`}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Column headers */}
      <div className="hidden xl:grid grid-cols-12 gap-4 px-5 py-2 shrink-0 text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant">
        <div className="col-span-4">Workload & Source</div>
        <div className="col-span-2">Agent & Storage</div>
        <div className="col-span-2">Schedule</div>
        <div className="col-span-2">Status</div>
        <div className="col-span-2 text-right">Actions</div>
      </div>

      {/* Job rows */}
      <div className="flex-1 min-h-0 overflow-y-auto flex flex-col gap-2.5 pr-0.5">
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
                onClick={() => setShowNew(true)}
                className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors"
              >
                + New Backup Job
              </button>
            }
          />
        ) : (
          paged.pageItems.map((job) => {
            const SourceIcon = sourceIcon[job.source_type] ?? Archive;
            return (
              <div
                key={job.id}
                role="button"
                tabIndex={0}
                onClick={() => navigate(`/jobs/${job.id}`)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    navigate(`/jobs/${job.id}`);
                  }
                }}
                className="group relative flex flex-col xl:grid xl:grid-cols-12 gap-3 xl:gap-4 items-stretch xl:items-center p-4 rounded-xl bg-surface-container-lowest shadow-sm hover:shadow-md transition-shadow duration-150 cursor-pointer border border-surface-variant focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
              >
                {/* Accent status stripe */}
                <div
                  className={`absolute left-0 top-0 bottom-0 w-1.5 ${accentColor(job)}`}
                />

                {/* Policy info */}
                <div className="xl:col-span-4 flex items-start gap-3 pl-2 min-w-0">
                  <div
                    className={`w-10 h-10 rounded-lg flex items-center justify-center shrink-0 mt-0.5 ${
                      job.enabled
                        ? "bg-primary-container/40 text-on-primary-container"
                        : "bg-surface-variant text-outline"
                    }`}
                  >
                    <SourceIcon size={20} />
                  </div>
                  <div className="flex flex-col min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-[14px] font-semibold text-on-surface truncate">
                        {job.name}
                      </span>
                      <span className="px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant text-[11px] font-medium uppercase tracking-wide">
                        {normalizeWindowsPath(job.source_type)}
                      </span>
                      {job.encrypted && (
                        <span className="px-1.5 py-0.5 rounded bg-primary-container text-primary text-[10px] font-semibold uppercase">
                          AES-256
                        </span>
                      )}
                    </div>
                    <p className="font-mono text-[11px] text-on-surface-variant mt-0.5 truncate">
                      {job.source_path || job.source_database || "—"}
                    </p>
                  </div>
                </div>

                {/* Agent & storage */}
                <div className="xl:col-span-2 flex flex-col gap-1 min-w-0">
                  <div className="flex items-center gap-1.5 font-mono text-[12px] text-on-surface">
                    <span
                      className={`w-2 h-2 rounded-full shrink-0 ${
                        job.agent?.status === "ONLINE"
                          ? "bg-primary"
                          : "bg-[#737686]"
                      }`}
                    />
                    <span className="font-medium truncate">
                      {job.agent?.name ?? "—"}
                    </span>
                  </div>
                  <div className="flex items-center gap-1 text-on-surface-variant text-[11px] truncate">
                    <Cloud size={12} className="shrink-0" />
                    <span className="truncate">
                      {job.storage_target?.name ?? "—"}
                    </span>
                  </div>
                </div>

                {/* Schedule */}
                <div className="xl:col-span-2 flex flex-col gap-1 min-w-0">
                  <div className="flex items-center gap-1.5 text-[13px] text-on-surface">
                    <Clock size={14} className="text-on-surface-variant shrink-0" />
                    <span className="truncate">
                      {job.schedule?.cron_expr ?? "No schedule"}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5 text-[11px] text-on-surface-variant">
                    <History size={12} className="shrink-0" />
                    <span>{job.retention_days}d retention</span>
                  </div>
                </div>

                {/* Status */}
                <div className="xl:col-span-2 flex flex-col gap-1">
                  <StatusBadge
                    status={job.enabled ? "ONLINE" : "OFFLINE"}
                    size="sm"
                  />
                  <span className="text-[11px] text-outline">
                    {job.mode === "COMPRESSED" ? "Zstandard" : "Normal"} mode
                  </span>
                </div>

                {/* Actions */}
                <div
                  className="xl:col-span-2 flex items-center justify-end gap-2 shrink-0 pt-2 xl:pt-0 border-t border-slate-100 xl:border-none"
                  onClick={(e) => e.stopPropagation()}
                >
                  <Action3DButton
                    onClick={(e) => handleRunNow(e, job)}
                    className="!px-2.5 !h-8 !text-[12px]"
                  >
                    <Play size={14} className="text-on-primary" />
                    Run Now
                  </Action3DButton>

                  <label
                    className="relative inline-flex items-center cursor-pointer"
                    title={job.enabled ? "Disable" : "Enable"}
                  >
                    <input
                      type="checkbox"
                      checked={job.enabled}
                      onChange={(e) => handleToggle(e, job)}
                      className="sr-only peer"
                    />
                    <div className="w-9 h-5 bg-surface-variant peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-surface-container-lowest after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary" />
                  </label>

                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      setEditId(job.id);
                    }}
                    title="Edit job"
                    className="p-1.5 rounded-lg text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-colors"
                  >
                    <Pencil size={16} />
                  </button>

                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      navigate(`/jobs/${job.id}`);
                    }}
                    title="View details"
                    className="p-1.5 rounded-lg text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-colors"
                  >
                    <MoreVertical size={18} />
                  </button>
                </div>
              </div>
            );
          })
        )}
      </div>

      <Pagination
        page={paged.page}
        totalPages={paged.totalPages}
        total={paged.total}
        perPage={paged.perPage}
        onPage={paged.setPage}
      />

      {editId && (
        <EditJobModal
          jobId={editId}
          onClose={() => setEditId(null)}
          onSaved={loadJobs}
        />
      )}

      {showNew && (
        <NewJobModal onClose={() => setShowNew(false)} onSaved={loadJobs} />
      )}
    </div>
  );
};

export default JobsPage;
