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
export const sharesApi = {
  list: () => unwrap<ShareLink[]>(http.get<Envelope<ShareLink[]>>("/shares/")),
  create: (file_id: string, opts: { password?: string; expires_in_days?: number; download_limit?: number }) =>
    unwrap<{ token: string; url: string }>(http.post<Envelope<{ token: string; url: string }>>("/shares/", { file_id, ...opts })),
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
