export interface UserInfo {
  id: string;
  email: string;
  username: string;
  full_name: string;
  role: string;
  permissions?: string[];
  storage_used: number;
  storage_quota: number;
  must_change_pw: boolean;
  has_avatar?: boolean;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user?: UserInfo;
}

export interface FolderItem {
  id: string;
  name: string;
  parent_id?: string;
  path: string;
  depth: number;
  size_bytes: number;
  color?: string;
  is_starred: boolean;
  is_pinned: boolean;
  created_at: string;
  updated_at: string;
}

export interface FileItem {
  id: string;
  name: string;
  extension: string;
  mime_type: string;
  size_bytes: number;
  checksum_sha256?: string;
  folder_id?: string;
  is_starred: boolean;
  version_no: number;
  download_count: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  transfer_pending?: boolean;
  transfer_deadline?: string;
}

export interface Listing {
  folders: FolderItem[];
  files: FileItem[];
}

export interface ShareLink {
  id: string;
  token: string;
  file_id?: string;
  permission: string;
  has_password: boolean;
  download_limit?: number;
  download_count: number;
  is_active: boolean;
  expires_at?: string;
  created_at: string;
}

export interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  last_used_at?: string;
  expires_at?: string;
  created_at: string;
}

export interface AuditLog {
  id: number;
  actor_email?: string;
  action: string;
  category: string;
  result: string;
  ip_address?: string;
  browser?: string;
  os?: string;
  object_id?: string;
  object_name?: string;
  user_agent?: string;
  request_id?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}
