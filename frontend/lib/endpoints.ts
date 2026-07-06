import { http, unwrap, tokens, type Envelope } from "./api";
import type {
  ApiKey, AuditLog, FileItem, FolderItem, Listing, ShareLink, TokenPair, UserInfo,
} from "./types";

// ── Auth ───────────────────────────────────────────────────
export const authApi = {
  async login(identifier: string, password: string, remember: boolean) {
    const pair = await unwrap<TokenPair>(
      http.post<Envelope<TokenPair>>("/auth/login", { identifier, password, remember_me: remember }),
    );
    tokens.set(pair.access_token, pair.refresh_token);
    return pair;
  },
  me: () => unwrap<UserInfo>(http.get<Envelope<UserInfo>>("/auth/me")),
  async logout() {
    const rt = tokens.refresh();
    try {
      if (rt) await http.post("/auth/logout", { refresh_token: rt });
    } finally {
      tokens.clear();
    }
  },
  changePassword: (current_password: string, new_password: string) =>
    http.post("/auth/change-password", { current_password, new_password }),
};

// ── Files & folders ────────────────────────────────────────
export const filesApi = {
  list: (folderId?: string) =>
    unwrap<Listing>(http.get<Envelope<Listing>>("/files/", { params: { folder_id: folderId } })),
  recent: () => unwrap<FileItem[]>(http.get<Envelope<FileItem[]>>("/files/recent")),
  starred: () => unwrap<FileItem[]>(http.get<Envelope<FileItem[]>>("/files/starred")),
  inherited: () => unwrap<FileItem[]>(http.get<Envelope<FileItem[]>>("/files/inherited")),
  keepFile: (id: string) => http.post(`/files/${id}/keep`, {}),
  emptyTrash: () => http.post("/files/trash/empty", {}),
  searchContent: (q: string) => unwrap<SearchHit[]>(http.get<Envelope<SearchHit[]>>(`/files/search/content?q=${encodeURIComponent(q)}`)),
  setLegalHold: (id: string, hold: boolean) => http.post(`/files/${id}/legal-hold`, { hold }),
  setRetention: (id: string, until: string | null) => http.post(`/files/${id}/retention`, { until }),
  versions: (id: string) => unwrap<FileVersion[]>(http.get<Envelope<FileVersion[]>>(`/files/${id}/versions`)),
  restoreVersion: (id: string, v: number) => http.post(`/files/${id}/versions/${v}/restore`, {}),
  versionDownloadUrl: (id: string, v: number) => {
    const t = tokens.access();
    return `/api/v1/files/${id}/versions/${v}/download${t ? `?access_token=${encodeURIComponent(t)}` : ""}`;
  },
  sharedWithMe: () => unwrap<SharedFile[]>(http.get<Envelope<SharedFile[]>>("/files/shared-with-me")),
  shareWithUser: (id: string, recipient_email: string, can_write: boolean) =>
    unwrap<FileGrant>(http.post<Envelope<FileGrant>>(`/files/${id}/share-user`, { recipient_email, can_write })),
  listGrants: (id: string) => unwrap<FileGrant[]>(http.get<Envelope<FileGrant[]>>(`/files/${id}/shares`)),
  revokeGrant: (id: string, userId: string) => http.delete(`/files/${id}/shares/${userId}`),
  trash: () => unwrap<FileItem[]>(http.get<Envelope<FileItem[]>>("/files/trash")),
  search: (q: string) =>
    unwrap<FileItem[]>(http.get<Envelope<FileItem[]>>("/files/search", { params: { q } })),
  createFolder: (name: string, parent_id?: string) =>
    http.post("/folders/", { name, parent_id: parent_id ?? null }),
  createFolderReturning: (name: string, parent_id?: string) =>
    unwrap<FolderItem>(http.post<Envelope<FolderItem>>("/folders/", { name, parent_id: parent_id ?? null })),
  renameFile: (id: string, name: string) => http.put(`/files/${id}/rename`, { name }),
  renameFolder: (id: string, name: string) => http.put(`/folders/${id}/rename`, { name }),
  starFile: (id: string, starred: boolean) => http.put(`/files/${id}/star`, { starred }),
  setFolderColor: (id: string, color: string) => http.put(`/folders/${id}/color`, { color }),
  trashFile: (id: string) => http.post(`/files/${id}/trash`, {}),
  restoreFile: (id: string) => http.post(`/files/${id}/restore`, {}),
  deleteFile: (id: string) => http.delete(`/files/${id}`),
  deleteFolder: (id: string) => http.delete(`/folders/${id}`),
  initUpload: (body: { filename: string; total_size: number; chunk_size: number; folder_id?: string }) =>
    unwrap<{ upload_id: string; total_chunks: number; chunk_size: number; received_chunks: number[] }>(
      http.post<Envelope<{ upload_id: string; total_chunks: number; chunk_size: number; received_chunks: number[] }>>("/uploads/", body),
    ),
  putChunk: (uploadId: string, index: number, chunk: Blob, signal?: AbortSignal) =>
    http.put(`/uploads/${uploadId}/chunks/${index}`, chunk, {
      headers: { "Content-Type": "application/octet-stream" },
      signal,
    }),
  completeUpload: (uploadId: string) =>
    unwrap<FileItem>(http.post<Envelope<FileItem>>(`/uploads/${uploadId}/complete`, {})),
  abortUpload: (uploadId: string) => http.delete(`/uploads/${uploadId}`),
  downloadUrl: (id: string) => {
    const t = tokens.access();
    return `/api/v1/files/${id}/download${t ? `?access_token=${encodeURIComponent(t)}` : ""}`;
  },
  // Inline URL for in-browser rendering (img/pdf/video/audio previews).
  previewUrl: (id: string) => {
    const t = tokens.access();
    const q = new URLSearchParams({ inline: "1" });
    if (t) q.set("access_token", t);
    return `/api/v1/files/${id}/download?${q.toString()}`;
  },
  // Fetch a text/code file's content (auth via interceptor).
  fetchText: (id: string) =>
    http.get<string>(`/files/${id}/download`, { responseType: "text", transformResponse: (d) => d }).then((r) => r.data),
  // Fetch raw bytes (for client-side Office rendering: xlsx, docx).
  fetchBinary: (id: string) =>
    http.get<ArrayBuffer>(`/files/${id}/download`, { responseType: "arraybuffer" }).then((r) => r.data),
  // Save edited text content (in-app editor) — creates a new version.
  saveContent: (id: string, text: string) =>
    http.put(`/files/${id}/content`, text, { headers: { "Content-Type": "application/octet-stream" } }),
  simpleUpload: (file: File, folderId: string | undefined, onProgress?: (pct: number) => void) => {
    const form = new FormData();
    form.append("file", file);
    if (folderId) form.append("folder_id", folderId);
    return http.post("/files/upload", form, {
      headers: { "Content-Type": "multipart/form-data" },
      onUploadProgress: (e) => {
        if (onProgress && e.total) onProgress(Math.round((e.loaded / e.total) * 100));
      },
    });
  },
};

// ── Common (organisation-wide) ─────────────────────────────
export interface CommonFile {
  id: string;
  name: string;
  extension: string;
  mime_type: string;
  size_bytes: number;
  is_starred: boolean;
  uploader_id: string;
  uploader_name: string;
  uploader_has_avatar: boolean;
  can_delete: boolean;
  created_at: string;
  updated_at: string;
  folder_id?: string;
  version_no: number;
  download_count: number;
  checksum_sha256?: string;
}

export const commonApi = {
  list: () => unwrap<CommonFile[]>(http.get<Envelope<CommonFile[]>>("/files/common")),
  remove: (id: string) => http.delete(`/files/common/${id}`),
  makeCommon: (id: string) => http.post(`/files/${id}/make-common`, {}),
  upload: (file: File, onProgress?: (pct: number) => void) => {
    const form = new FormData();
    form.append("file", file);
    return http.post("/files/common/upload", form, {
      headers: { "Content-Type": "multipart/form-data" },
      onUploadProgress: (e) => {
        if (onProgress && e.total) onProgress(Math.round((e.loaded / e.total) * 100));
      },
    });
  },
};

// ── Shares ─────────────────────────────────────────────────
export interface FileVersion {
  version_no: number; size_bytes: number; checksum_sha256?: string; author?: string; created_at: string;
}
export interface SearchHit {
  id: string; name: string; extension: string; mime_type: string; size_bytes: number;
  folder_id?: string; is_starred: boolean; version_no: number; download_count: number;
  created_at: string; updated_at: string; snippet?: string; rank: number;
}
export interface FileGrant {
  user_id: string; name: string; email: string; has_avatar: boolean; can_write: boolean;
}
export interface SharedFile {
  id: string; name: string; extension: string; mime_type: string; size_bytes: number;
  is_starred: boolean; version_no: number; download_count: number;
  created_at: string; updated_at: string;
  owner_id: string; owner_name: string; owner_has_avatar: boolean;
  can_write: boolean; shared_at: string;
}
export interface ShareCreateResult {
  token: string; url: string; has_password: boolean;
  emailed: boolean; external: boolean; expires_at?: string;
}
export const sharesApi = {
  list: () => unwrap<ShareLink[]>(http.get<Envelope<ShareLink[]>>("/shares/")),
  create: (file_id: string, opts: { password?: string; expires_in_days?: number; download_limit?: number; recipient_email?: string }) =>
    unwrap<ShareCreateResult>(http.post<Envelope<ShareCreateResult>>("/shares/", { file_id, ...opts })),
  revoke: (id: string) => http.delete(`/shares/${id}`),
};

// ── API keys ───────────────────────────────────────────────
export const apiKeysApi = {
  list: () => unwrap<ApiKey[]>(http.get<Envelope<ApiKey[]>>("/api-keys/")),
  create: (name: string, scopes: string[], expires_in_days?: number) =>
    unwrap<{ key: string; prefix: string }>(
      http.post<Envelope<{ key: string; prefix: string }>>("/api-keys/", { name, scopes, expires_in_days }),
    ),
  revoke: (id: string) => http.delete(`/api-keys/${id}`),
};

// ── Users (admin) ──────────────────────────────────────────
export interface AdminUser {
  id: string;
  email: string;
  username: string;
  full_name: string;
  role: string;
  storage_used: number;
  storage_quota: number;
  is_active: boolean;
  is_locked: boolean;
  has_avatar?: boolean;
  last_login_at?: string;
  created_at: string;
}
export interface RoleInfo {
  id: string; name: string; slug: string; description: string;
  is_system: boolean; priority: number; permissions: string[];
}

export const usersApi = {
  list: (limit = 50, offset = 0) =>
    unwrap<AdminUser[]>(http.get<Envelope<AdminUser[]>>("/users/", { params: { limit, offset } })),
  create: (body: { email: string; username: string; password: string; full_name: string; role: string; storage_quota: number }) =>
    http.post("/users/", body),
  setActive: (id: string, is_active: boolean) => http.put(`/users/${id}/status`, { is_active }),
  setRole: (id: string, role: string) => http.put(`/users/${id}/role`, { role }),
  resetPassword: (id: string, new_password: string) => http.post(`/users/${id}/reset-password`, { new_password }),
  remove: (id: string, transfer_to: string) => http.delete(`/users/${id}`, { data: { transfer_to } }),
  enable: (id: string) => http.post(`/users/${id}/enable`, {}),
};

export const rolesApi = {
  list: () => unwrap<RoleInfo[]>(http.get<Envelope<RoleInfo[]>>("/roles/")),
};

export const avatarApi = {
  url: (userId: string) => {
    const t = tokens.access();
    return `/api/v1/users/${userId}/avatar${t ? `?access_token=${encodeURIComponent(t)}` : ""}`;
  },
  upload: (file: File) => {
    const form = new FormData();
    form.append("avatar", file);
    return http.post("/users/me/avatar", form, { headers: { "Content-Type": "multipart/form-data" } });
  },
};

export interface UserStorage {
  id: string; username: string; full_name: string; email: string; role: string;
  storage_used: number; storage_quota: number; unlimited: boolean;
  file_count: number; percent_used: number;
}
export interface MediaSlice { category: string; total: number; files: number; }
export interface StorageOverview { users: UserStorage[]; media: MediaSlice[]; system_used: number; }

export const storageApi = {
  overview: () => unwrap<StorageOverview>(http.get<Envelope<StorageOverview>>("/users/storage")),
};

// ── Security alerts (audit anomaly detection) ──────────────
export interface SecurityAlert {
  id: string; type: string; severity: string; actor_email?: string;
  summary: string; event_count: number; window_start?: string; window_end?: string;
  resolved: boolean; created_at: string;
}
export const securityApi = {
  list: () => unwrap<SecurityAlert[]>(http.get<Envelope<SecurityAlert[]>>("/security/alerts")),
  unresolvedCount: () => unwrap<{ unresolved: number }>(http.get<Envelope<{ unresolved: number }>>("/security/alerts/unresolved-count")),
  resolve: (id: string) => http.post(`/security/alerts/${id}/resolve`, {}),
};

// ── Office editor (OnlyOffice) ─────────────────────────────
export interface EditorSession { doc_server_url: string; config: Record<string, unknown> }
export const editorApi = {
  config: (fileId: string) => unwrap<EditorSession>(http.get<Envelope<EditorSession>>(`/editor/${fileId}/config`)),
};

// ── AI (semantic search + ask-your-files) ─────────────────
export interface AiSource { file_id: string; name: string }
export interface AiAnswer { answer: string; sources?: AiSource[] }
export interface AiHit { file_id: string; name: string; score: number; snippet: string }
export const aiApi = {
  status: () => unwrap<{ enabled: boolean }>(http.get<Envelope<{ enabled: boolean }>>("/ai/status")),
  ask: (question: string) => unwrap<AiAnswer>(http.post<Envelope<AiAnswer>>("/ai/ask", { question })),
  search: (q: string) => unwrap<AiHit[]>(http.get<Envelope<AiHit[]>>(`/ai/search?q=${encodeURIComponent(q)}`)),
};

// ── Notifications ──────────────────────────────────────────
export interface Notification {
  id: string; type: string; title: string; body: string;
  link?: string; is_read: boolean; created_at: string;
}
export const notificationsApi = {
  list: () => unwrap<Notification[]>(http.get<Envelope<Notification[]>>("/notifications/")),
  unreadCount: () => unwrap<{ unread: number }>(http.get<Envelope<{ unread: number }>>("/notifications/unread-count")),
  markRead: (id: string) => http.post(`/notifications/${id}/read`, {}),
  markAllRead: () => http.post("/notifications/read-all", {}),
};

// ── Audit + telemetry ──────────────────────────────────────
export const auditApi = {
  list: (limit = 100, offset = 0) =>
    unwrap<AuditLog[]>(http.get<Envelope<AuditLog[]>>("/audit/", { params: { limit, offset } })),
  track: (event_type: string, element?: string, path?: string, metadata?: Record<string, unknown>) =>
    http.post("/activity/", { event_type, element, path, metadata }).catch(() => {}),
};
