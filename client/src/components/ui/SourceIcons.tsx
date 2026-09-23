import type { SVGProps, ImgHTMLAttributes } from 'react'
import filesystemSvg from '@/icons/filesystem.svg'
import mssqlSvg from '@/icons/mssql.svg'
import postgresSvg from '@/icons/postgres.svg'
import mongodbSvg from '@/icons/mongodb.svg'

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
