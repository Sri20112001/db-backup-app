import type { SVGProps, ImgHTMLAttributes } from 'react'
import { Database } from 'lucide-react'
import filesystemSvg from '@/assets/icons/filesystem.svg'
import mssqlSvg from '@/assets/icons/mssql.svg'
import postgresSvg from '@/assets/icons/postgres.svg'
import mongodbSvg from '@/assets/icons/mongodb.svg'

interface IconProps extends ImgHTMLAttributes<HTMLImageElement> {
  size?: number | string
}

export const FileSystemIcon = ({ size = 20, className = '', ...props }: IconProps) => (
  <img src={filesystemSvg} width={size} height={size} className={className} alt="Filesystem" {...props} />
)

export const MssqlServerIcon = ({ size = 20, className = '', ...props }: IconProps) => (
  <img src={mssqlSvg} width={size} height={size} className={className} alt="MSSQL Server" {...props} />
)

export const PostgresIcon = ({ size = 20, className = '', ...props }: IconProps) => (
  <img src={postgresSvg} width={size} height={size} className={className} alt="PostgreSQL" {...props} />
)

export const MongoDbIcon = ({ size = 20, className = '', ...props }: IconProps) => (
  <img src={mongodbSvg} width={size} height={size} className={className} alt="MongoDB" {...props} />
)

// Legacy DBF icon (inline SVG as it doesn't have an asset yet)
export const DbfIcon = ({ size = 20, className = '', ...props }: SVGProps<SVGSVGElement> & { size?: number | string }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    className={className}
    {...props}
  >
    <rect x="3" y="3.5" width="18" height="17" rx="2.5" />
    <line x1="3" y1="8.5" x2="21" y2="8.5" />
    <line x1="8.5" y1="8.5" x2="8.5" y2="20.5" />
    <line x1="8.5" y1="14.5" x2="21" y2="14.5" />
    <text x="5.5" y="7.2" fill="currentColor" stroke="none" fontSize="4" fontWeight="800" fontFamily="monospace" letterSpacing="0.4">
      DBF
    </text>
  </svg>
)

// The engine's own mark for a connection or job source type. Falls back to a
// neutral database glyph so an unknown type still renders something sensible.
export const DbTypeIcon = ({
  type,
  size = 20,
  className = '',
}: {
  type: string
  size?: number | string
  className?: string
}) => {
  switch (type) {
    case 'POSTGRES':
      return <PostgresIcon size={size} className={className} />
    case 'MONGODB':
      return <MongoDbIcon size={size} className={className} />
    case 'MSSQL':
    case 'MSSQL_SERVER':
      return <MssqlServerIcon size={size} className={className} />
    case 'FILESYSTEM':
      return <FileSystemIcon size={size} className={className} />
    case 'DBF':
      return <DbfIcon size={size} className={className} />
    default:
      return <Database size={size} className={className} aria-hidden="true" />
  }
}
