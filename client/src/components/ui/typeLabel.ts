// Display names for connection and job source types — used by cards, filters
// and dialogs so a type is never shown as a bare enum value. Kept apart from
// SourceIcons so that file only exports components (fast-refresh requirement).
export const TYPE_LABELS: Record<string, string> = {
  POSTGRES: 'PostgreSQL',
  MONGODB: 'MongoDB',
  MSSQL: 'SQL Server',
  FILESYSTEM: 'Filesystem',
  DBF: 'DBF Dataset',
  MSSQL_SERVER: 'SQL Server',
}

export const typeLabel = (type: string) => TYPE_LABELS[type] ?? type
