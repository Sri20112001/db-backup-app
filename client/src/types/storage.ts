export type StorageType = 'S3' | 'LOCAL' | 'SMB'

export interface S3Region {
  id: string
  code: string
  name: string
  provider: string
  endpoint?: string
  is_system: boolean
  active: boolean
}

export interface StorageTarget {
  id: string
  organization_id: string
  name: string
  type: StorageType
  bucket: string
  region: string
  endpoint: string
  use_path_style: boolean
  path: string
  has_credentials: boolean
  created_at: string
}
