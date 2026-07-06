"use client";

import { useEffect, useState } from "react";
import { Copy, Check, Terminal, Download, FileJson, FileText, ShieldCheck } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/misc";

interface Endpoint {
  group: string;
  method: string;
  path: string;
  desc: string;
  body?: Record<string, unknown>;
  query?: string;
}

const KEY = "sftp_XXXXXXXX_YYYYYYYYYYYYYYYYYYYYYYY";

const ENDPOINTS: Endpoint[] = [
  { group: "Auth", method: "POST", path: "/auth/login", desc: "Obtain access + refresh tokens", body: { identifier: "user@corp.com", password: "••••••••", remember_me: true } },
  { group: "Auth", method: "POST", path: "/auth/refresh", desc: "Rotate the access token", body: { refresh_token: "••••" } },
  { group: "Auth", method: "GET", path: "/auth/me", desc: "Current user profile + permissions" },
  { group: "Files", method: "GET", path: "/files/", desc: "List folder contents", query: "folder_id, limit, offset" },
  { group: "Files", method: "POST", path: "/files/upload", desc: "Upload a single file (multipart form field: file)" },
  { group: "Files", method: "GET", path: "/files/{id}/download", desc: "Download a file (supports HTTP Range)" },
  { group: "Files", method: "GET", path: "/files/search", desc: "Search files by name", query: "q, limit, offset" },
  { group: "Files", method: "POST", path: "/files/{id}/trash", desc: "Move a file to the recycle bin" },
  { group: "Files", method: "DELETE", path: "/files/{id}", desc: "Permanently delete a file" },
  { group: "Folders", method: "POST", path: "/folders/", desc: "Create a folder", body: { name: "Reports", parent_id: null } },
  { group: "Folders", method: "PUT", path: "/folders/{id}/rename", desc: "Rename a folder", body: { name: "Q3 Reports" } },
  { group: "Uploads", method: "POST", path: "/uploads/", desc: "Start a resumable upload (large files)", body: { filename: "big.zip", total_size: 5368709120, chunk_size: 8388608 } },
  { group: "Uploads", method: "PUT", path: "/uploads/{id}/chunks/{index}", desc: "Upload one chunk (raw body)" },
  { group: "Uploads", method: "POST", path: "/uploads/{id}/complete", desc: "Finalise a resumable upload" },
  { group: "Shares", method: "POST", path: "/shares/", desc: "Create a share link", body: { file_id: "…", password: "", expires_in_days: 7, download_limit: 100 } },
  { group: "Shares", method: "GET", path: "/shares/", desc: "List your share links" },
  { group: "Common", method: "GET", path: "/files/common", desc: "List organisation-wide Common files" },
  { group: "Common", method: "POST", path: "/files/common/upload", desc: "Upload a file to Common (multipart)" },
];

const methodColor: Record<string, string> = {
  GET: "text-success", POST: "text-primary", PUT: "text-warning", PATCH: "text-warning", DELETE: "text-danger",
};

function curlFor(base: string, e: Endpoint): string {
  const url = `${base}${e.path}${e.query ? `?${e.query.split(",")[0].trim()}=…` : ""}`;
  const lines = [`curl -X ${e.method} "${url}"`, `  -H "X-API-Key: ${KEY}"`];
  if (e.body) {
    lines.push(`  -H "Content-Type: application/json"`);
    lines.push(`  -d '${JSON.stringify(e.body)}'`);
  } else if (e.method === "POST" && e.path.includes("upload")) {
    lines.push(`  -F "file=@./document.pdf"`);
  }
  return lines.join(" \\\n");
}

export function ApiDocs() {
  const [base, setBase] = useState("http://localhost/api/v1");
  const [copied, setCopied] = useState<string | null>(null);
  useEffect(() => setBase(`${window.location.origin}/api/v1`), []);

  function copy(text: string, id: string) {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 1200);
  }

  function downloadPostman() {
    const collection = {
      info: { name: "Sapphire SFTP API", schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" },
      variable: [
        { key: "baseUrl", value: base },
        { key: "apiKey", value: "" },
      ],
      item: ENDPOINTS.map((e) => ({
        name: `${e.method} ${e.path}`,
        request: {
          method: e.method,
          header: [
            { key: "X-API-Key", value: "{{apiKey}}" },
            ...(e.body ? [{ key: "Content-Type", value: "application/json" }] : []),
          ],
          url: {
            raw: `{{baseUrl}}${e.path}`,
            host: ["{{baseUrl}}"],
            path: e.path.split("/").filter(Boolean),
          },
          ...(e.body ? { body: { mode: "raw", raw: JSON.stringify(e.body, null, 2) } } : {}),
        },
      })),
    };
    const blob = new Blob([JSON.stringify(collection, null, 2)], { type: "application/json" });
    triggerDownload(blob, "sapphire-sftp.postman_collection.json");
  }

  function downloadPdf() {
    const rows = ENDPOINTS.map(
      (e) => `<tr><td class="m ${e.method}">${e.method}</td><td class="p">${e.path}</td><td>${e.desc}</td></tr>`,
    ).join("");
    const html = `<!doctype html><html><head><meta charset="utf-8"><title>Sapphire SFTP API</title>
<style>
  body{font-family:-apple-system,Segoe UI,Roboto,sans-serif;color:#18181b;margin:40px;}
  h1{color:#064D51;margin:0 0 4px;} .sub{color:#666;margin:0 0 24px;font-size:13px;}
  h2{color:#064D51;border-bottom:2px solid #eee;padding-bottom:4px;margin-top:28px;font-size:16px;}
  code{background:#f3f5f9;padding:2px 6px;border-radius:4px;font-size:12px;}
  table{width:100%;border-collapse:collapse;margin-top:8px;font-size:12px;}
  td{border:1px solid #e5e7eb;padding:6px 8px;vertical-align:top;}
  .m{font-weight:700;width:60px;} .GET{color:#16a34a}.POST{color:#4f46e5}.PUT{color:#d97706}.DELETE{color:#dc2626}
  .p{font-family:monospace;white-space:nowrap;}
</style></head><body>
  <h1>Sapphire SFTP — API Reference</h1>
  <p class="sub">Base URL: <code>${base}</code> · Auth: <code>X-API-Key: &lt;key&gt;</code> or <code>Authorization: Bearer &lt;jwt&gt;</code></p>
  <h2>Endpoints</h2>
  <table><tr><td class="m">METHOD</td><td class="p">PATH</td><td>DESCRIPTION</td></tr>${rows}</table>
  <p class="sub" style="margin-top:24px">Errors are returned as RFC 7807 problem+json. Responses use a uniform envelope with <code>success</code>, <code>data</code>, <code>message</code>.</p>
</body></html>`;
    const w = window.open("", "_blank");
    if (!w) return;
    w.document.write(html);
    w.document.close();
    w.focus();
    setTimeout(() => w.print(), 300);
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="flex items-center gap-2"><Terminal size={16} /> API Reference</CardTitle>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={downloadPostman}><FileJson size={15} /> Postman</Button>
            <Button variant="outline" size="sm" onClick={downloadPdf}><FileText size={15} /> PDF</Button>
          </div>
        </div>
        <p className="text-sm text-muted">
          Programmatic access to every file operation. Base URL:{" "}
          <code className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-xs">{base}</code>
        </p>
      </CardHeader>
      <CardContent className="space-y-5">
        {/* Auth */}
        <div className="rounded-lg border border-border bg-surface-2 p-3">
          <p className="mb-1 flex items-center gap-1.5 text-sm font-medium"><ShieldCheck size={14} className="text-primary" /> Authentication</p>
          <div className="flex items-center gap-2">
            <code className="min-w-0 flex-1 truncate font-mono text-xs">X-API-Key: {KEY}</code>
            <button onClick={() => copy(`X-API-Key: ${KEY}`, "hdr")} className="text-muted hover:text-foreground">
              {copied === "hdr" ? <Check size={14} className="text-success" /> : <Copy size={14} />}
            </button>
          </div>
          <p className="mt-2 text-xs text-muted">
            Or a session token: <code className="font-mono">Authorization: Bearer &lt;jwt&gt;</code>. Keys carry scopes that limit their reach.
          </p>
        </div>

        {/* Endpoints grouped */}
        {[...new Set(ENDPOINTS.map((e) => e.group))].map((group) => (
          <div key={group}>
            <p className="eyebrow mb-2">{group}</p>
            <div className="space-y-2">
              {ENDPOINTS.filter((e) => e.group === group).map((e) => {
                const cmd = curlFor(base, e);
                const id = e.method + e.path;
                return (
                  <div key={id} className="overflow-hidden rounded-lg border border-border">
                    <div className="flex items-center gap-2 px-3 py-2">
                      <span className={`w-14 shrink-0 font-mono text-xs font-semibold ${methodColor[e.method]}`}>{e.method}</span>
                      <code className="min-w-0 flex-1 truncate font-mono text-xs">{e.path}</code>
                      <span className="hidden text-xs text-muted sm:block">{e.desc}</span>
                    </div>
                    <div className="relative border-t border-border bg-[#0d1214]">
                      <pre className="overflow-x-auto px-3 py-2.5 font-mono text-[11px] leading-relaxed text-zinc-100">{cmd}</pre>
                      <button onClick={() => copy(cmd, id)} className="absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded text-white/50 hover:bg-white/10 hover:text-white">
                        {copied === id ? <Check size={13} className="text-emerald-400" /> : <Copy size={13} />}
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        ))}

        <div className="flex flex-wrap items-center gap-1.5 text-xs text-muted">
          <span>Scopes:</span>
          {["files.read", "files.upload", "files.write", "files.delete", "files.share"].map((s) => <Badge key={s}>{s}</Badge>)}
        </div>
      </CardContent>
    </Card>
  );
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
