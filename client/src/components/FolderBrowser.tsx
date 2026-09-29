import { useCallback, useEffect, useState } from 'react'
import { Folder, FolderOpen, ArrowUp, HardDrive, Loader2, X } from 'lucide-react'
import { AGENT_BROWSE_BASE_URL as BROWSE_BASE } from '@/CONSTANTS'

interface Entry {
  name: string
  path: string
  is_dir: boolean
}

interface BrowseData {
  cwd: string
  parent: string
  roots: string[]
  entries: Entry[]
}

interface Props {
  initialPath?: string
  onSelect: (path: string) => void
  onClose: () => void
}

const FolderBrowser = ({ initialPath = '', onSelect, onClose }: Props) => {
  const [data, setData] = useState<BrowseData | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async (path: string) => {
    setIsLoading(true)
    setError('')
    try {
      const res = await fetch(`${BROWSE_BASE}/browse?path=${encodeURIComponent(path)}`)
      if (!res.ok) throw new Error('bad response')
      setData(await res.json())
    } catch {
      setError(
        'Could not reach the folder browser. Make sure the agent is running on this machine (it serves the picker on 127.0.0.1:7546), or type the path manually.',
      )
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- intentional fetch-on-mount for modal data
    void load(initialPath)
  }, [load, initialPath])

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center">
      <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={onClose} />
      <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl w-full max-w-lg mx-4 border border-surface-variant overflow-hidden">
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-surface-variant">
          <h2 className="text-[15px] font-semibold text-on-surface">Choose folder on agent</h2>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded text-outline hover:text-on-surface hover:bg-surface-container-low"
          >
            <X size={16} />
          </button>
        </div>

        <div className="px-5 py-3 border-b border-surface-variant bg-surface-container-low">
          <p className="font-mono text-[12px] text-on-surface break-all">{data?.cwd || '…'}</p>
          {data && data.roots.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-2">
              {data.roots.map((r) => (
                <button
                  key={r}
                  type="button"
                  onClick={() => void load(r)}
                  className={`flex items-center gap-1 px-2 py-1 rounded text-[11px] font-medium transition-colors ${
                    data.cwd === r
                      ? 'bg-primary text-on-primary'
                      : 'bg-surface-container-high text-on-surface-variant hover:bg-surface-variant'
                  }`}
                >
                  <HardDrive size={12} />
                  {r}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="h-64 overflow-y-auto px-2 py-2">
          {isLoading ? (
            <div className="flex items-center justify-center h-full text-outline">
              <Loader2 size={20} className="animate-spin" />
            </div>
          ) : error ? (
            <p className="px-3 py-4 text-[13px] text-error">{error}</p>
          ) : (
            <>
              {data?.parent && (
                <button
                  type="button"
                  onClick={() => void load(data.parent)}
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-[13px] text-on-surface-variant hover:bg-surface-container-low transition-colors"
                >
                  <ArrowUp size={16} className="text-outline" />
                  ..
                </button>
              )}
              {data?.entries.map((e) => (
                <button
                  key={e.path}
                  type="button"
                  onDoubleClick={() => void load(e.path)}
                  onClick={() => void load(e.path)}
                  title="Click to open"
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-[13px] text-on-surface hover:bg-surface-container-low transition-colors"
                >
                  {data.cwd === e.path ? (
                    <FolderOpen size={16} className="text-primary shrink-0" />
                  ) : (
                    <Folder size={16} className="text-outline shrink-0" />
                  )}
                  <span className="truncate">{e.name}</span>
                </button>
              ))}
              {data && data.entries.length === 0 && !data.parent && (
                <p className="px-3 py-4 text-[13px] text-outline">No subfolders here.</p>
              )}
            </>
          )}
        </div>

        <div className="flex items-center justify-between gap-3 px-5 py-3.5 border-t border-surface-variant bg-surface-container-low">
          <p className="text-[11px] text-outline leading-snug">
            Browses the agent on this machine.
            <br />
            For remote agents, type the path manually.
          </p>
          <div className="flex items-center gap-2 shrink-0">
            <button
              type="button"
              onClick={onClose}
              className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={!data?.cwd || !!error}
              onClick={() => data && onSelect(data.cwd)}
              className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Select this folder
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default FolderBrowser
