"use client";

import { use, useEffect, useState } from "react";
import { Download, FileDown, FolderDown, Lock, Loader2, ShieldAlert } from "lucide-react";
import { BRAND } from "@/lib/brand";

interface PublicInfo {
  token: string; kind?: "file" | "folder"; file_name: string; size_bytes: number; mime_type: string;
  item_count?: number; has_password: boolean; permission: string;
}

function humanSize(n: number) {
  if (!n) return "0 B";
  const u = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(n) / Math.log(1024));
  return `${(n / 1024 ** i).toFixed(i ? 1 : 0)} ${u[i]}`;
}

export default function PublicSharePage({ params }: { params: Promise<{ token: string }> }) {
  const { token } = use(params);
  const [info, setInfo] = useState<PublicInfo | null>(null);
  const [status, setStatus] = useState<"loading" | "ok" | "notfound">("loading");
  const [password, setPassword] = useState("");
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    fetch(`/api/v1/share/${token}`)
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((j) => { setInfo(j.data as PublicInfo); setStatus("ok"); })
      .catch(() => setStatus("notfound"));
  }, [token]);

  async function download() {
    if (!info) return;
    setDownloading(true); setError("");
    const url = `/api/v1/share/${token}/download${info.has_password ? `?password=${encodeURIComponent(password)}` : ""}`;
    try {
      const res = await fetch(url);
      if (!res.ok) {
        setError(res.status === 401 || res.status === 403 ? "Wrong password, or this link has expired / hit its limit." : "This link is no longer available.");
        setDownloading(false);
        return;
      }
      const blob = await res.blob();
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = info.kind === "folder" ? `${info.file_name}.zip` : info.file_name;
      a.click();
      URL.revokeObjectURL(a.href);
    } catch {
      setError("Download failed. Please try again.");
    } finally {
      setDownloading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-2 p-4">
      <div className="w-full max-w-md rounded-2xl border border-border bg-surface p-6 shadow-lg">
        <div className="mb-5 flex items-center gap-2">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={BRAND.logo.full} alt="" width={26} height={26} />
          <span className="font-semibold">{BRAND.company.product}</span>
        </div>

        {status === "loading" && (
          <div className="flex flex-col items-center gap-3 py-10 text-muted">
            <Loader2 className="animate-spin" /> Loading…
          </div>
        )}

        {status === "notfound" && (
          <div className="flex flex-col items-center gap-3 py-10 text-center">
            <ShieldAlert size={40} className="text-danger" />
            <p className="font-medium">Link not found</p>
            <p className="text-sm text-muted">This share link is invalid, expired, or has been revoked.</p>
          </div>
        )}

        {status === "ok" && info && (
          <>
            <div className="flex items-center gap-3 rounded-xl border border-border bg-surface-2 p-4">
              {info.kind === "folder"
                ? <FolderDown size={28} className="shrink-0 text-primary" />
                : <FileDown size={28} className="shrink-0 text-primary" />}
              <div className="min-w-0">
                <p className="truncate font-medium">{info.file_name}</p>
                <p className="text-xs text-muted">
                  {info.kind === "folder"
                    ? `${info.item_count ?? 0} file${(info.item_count ?? 0) === 1 ? "" : "s"} · downloads as a zip`
                    : `${humanSize(info.size_bytes)} · shared with you`}
                </p>
              </div>
            </div>

            {info.has_password && (
              <div className="mt-4">
                <label className="mb-1 flex items-center gap-1.5 text-sm font-medium"><Lock size={14} /> Password required</label>
                <input
                  type="password" value={password} onChange={(e) => setPassword(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && download()}
                  placeholder="Enter the password"
                  className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>
            )}

            {error && <p className="mt-3 text-sm text-danger">{error}</p>}

            <button
              onClick={download} disabled={downloading || (info.has_password && !password)}
              className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 font-medium text-primary-foreground transition hover:brightness-110 disabled:opacity-50"
            >
              {downloading ? <Loader2 size={16} className="animate-spin" /> : <Download size={16} />} Download
            </button>
          </>
        )}

        <p className="mt-6 text-center text-[11px] text-muted">Powered by {BRAND.company.product}</p>
      </div>
    </div>
  );
}
