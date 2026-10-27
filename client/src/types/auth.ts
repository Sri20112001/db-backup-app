export interface User {
  id: string
  email: string
  name: string
}

export interface Organization {
  id: string
  name: string
  slug: string
  user_limit: number
}

export interface OrganizationMember {
  id: string
  organization_id: string
  user_id: string
  role: MemberRole
  user: User
}

export type MemberRole = 'OWNER' | 'ADMIN' | 'OPERATOR' | 'VIEWER'

export interface AuthTokens {
  access_token: string
  refresh_token: string
  user: User
}
