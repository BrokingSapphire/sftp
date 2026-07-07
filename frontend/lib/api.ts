import axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from "axios";

// ── Token storage ──────────────────────────────────────────
const ACCESS_KEY = "sftp_access_token";
const REFRESH_KEY = "sftp_refresh_token";

export const tokens = {
  access: () => (typeof window !== "undefined" ? localStorage.getItem(ACCESS_KEY) : null),
  refresh: () => (typeof window !== "undefined" ? localStorage.getItem(REFRESH_KEY) : null),
  set(access: string, refresh: string) {
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
  },
  setAccess(access: string) {
    localStorage.setItem(ACCESS_KEY, access);
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

// ── API envelope types ─────────────────────────────────────
export interface Envelope<T> {
  success: boolean;
  message?: string;
  data?: T;
  meta?: unknown;
  error?: { code: number; type: string; message: string }[];
  request_id?: string;
}

export interface Problem {
  title: string;
  status: number;
  detail?: string;
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

// ── Axios client with token refresh ────────────────────────
export const http: AxiosInstance = axios.create({
  baseURL: "/api/v1",
  headers: { "Content-Type": "application/json" },
});

http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const t = tokens.access();
  if (t) config.headers.Authorization = `Bearer ${t}`;
  return config;
});

let refreshing: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  const rt = tokens.refresh();
  if (!rt) return null;
  try {
    const res = await axios.post<Envelope<{ access_token: string }>>(
      "/api/v1/auth/refresh",
      { refresh_token: rt },
    );
    const access = res.data.data?.access_token;
    if (access) {
      tokens.setAccess(access);
      return access;
    }
  } catch {
    /* fall through */
  }
  tokens.clear();
  return null;
}

http.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config as AxiosRequestConfig & { _retry?: boolean };
    if (error.response?.status === 401 && !original._retry && tokens.refresh()) {
      original._retry = true;
      refreshing ??= refreshAccessToken().finally(() => (refreshing = null));
      const newToken = await refreshing;
      if (newToken) {
        original.headers = { ...original.headers, Authorization: `Bearer ${newToken}` };
        return http(original);
      }
      // Refresh failed (session revoked / expired — e.g. signed in elsewhere).
      // Send the user back to login instead of leaving them on a broken page.
      if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
        tokens.clear();
        window.location.href = "/login";
      }
    }
    const problem = error.response?.data as Problem | undefined;
    throw new ApiError(problem?.title ?? error.message ?? "Request failed", error.response?.status ?? 0);
  },
);

/** Unwrap the standard success envelope, returning `data`. */
export async function unwrap<T>(p: Promise<{ data: Envelope<T> }>): Promise<T> {
  const res = await p;
  return res.data.data as T;
}
