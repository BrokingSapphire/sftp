"use client";

import { useEffect, useState } from "react";
import {
  Copy, Check, Terminal, FileJson, FileText, ShieldCheck, Rocket, Bot, DatabaseBackup,
  Workflow, PlugZap, KeyRound,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/misc";
import { BRAND } from "@/lib/brand";

interface Endpoint {
  group: string;
  method: string;
  path: string;
  desc: string;
  body?: Record<string, unknown>;
  query?: string;
  upload?: boolean;   // multipart file upload
  download?: boolean; // binary response
}

const KEY = "sftp_XXXXXXXX_YYYYYYYYYYYYYYYYYYYYYYY";
const LANGS = ["cURL", "JavaScript", "Python", "Go"] as const;
type Lang = (typeof LANGS)[number];

// ── What is an API key? (plain-English use cases) ───────────────
const USE_CASES = [
  { icon: Workflow, title: "Automate boring jobs", text: "Let a script upload the nightly report or clean up old files — no human, no password typed anywhere." },
  { icon: DatabaseBackup, title: "Back things up", text: "A scheduled task pulls files out to another server or disk on its own." },
  { icon: PlugZap, title: "Connect other apps", text: "Wire your CRM, CI pipeline, or a Slack bot straight into your files." },
  { icon: Bot, title: "Feed AI & tools", text: "Give an internal tool read-only access to search and fetch documents." },
];

const ENDPOINTS: Endpoint[] = [
  { group: "Auth", method: "POST", path: "/auth/login", desc: "Obtain access + refresh tokens", body: { identifier: "user@corp.com", password: "••••••••", remember_me: true } },
  { group: "Auth", method: "POST", path: "/auth/refresh", desc: "Rotate the access token", body: { refresh_token: "…" } },
  { group: "Auth", method: "GET", path: "/auth/me", desc: "Current user profile + permissions" },
  { group: "Files", method: "GET", path: "/files/", desc: "List a folder's contents", query: "folder_id, limit, offset" },
  { group: "Files", method: "POST", path: "/files/upload", desc: "Upload a single file", upload: true },
  { group: "Files", method: "GET", path: "/files/{id}/download", desc: "Download a file (supports HTTP Range)", download: true },
  { group: "Files", method: "POST", path: "/files/{id}/copy", desc: "Duplicate a file" },
  { group: "Files", method: "GET", path: "/files/search", desc: "Search files by name", query: "q" },
  { group: "Files", method: "GET", path: "/files/search/content", desc: "Full-text search inside documents", query: "q" },
  { group: "Files", method: "POST", path: "/files/{id}/trash", desc: "Move a file to the recycle bin" },
  { group: "Files", method: "DELETE", path: "/files/{id}", desc: "Permanently delete a file" },
  { group: "Folders", method: "POST", path: "/folders/", desc: "Create a folder", body: { name: "Reports", parent_id: null } },
  { group: "Folders", method: "PUT", path: "/folders/{id}/rename", desc: "Rename a folder", body: { name: "Q3 Reports" } },
  { group: "Folders", method: "GET", path: "/folders/{id}/download", desc: "Download a whole folder as a zip", download: true },
  { group: "Uploads (resumable)", method: "POST", path: "/uploads/", desc: "Start a resumable upload for large files", body: { filename: "big.zip", total_size: 5368709120, chunk_size: 8388608 } },
  { group: "Uploads (resumable)", method: "PUT", path: "/uploads/{id}/chunks/{index}", desc: "Upload one chunk (raw body)" },
  { group: "Uploads (resumable)", method: "POST", path: "/uploads/{id}/complete", desc: "Finalise a resumable upload" },
  { group: "Versions", method: "GET", path: "/files/{id}/versions", desc: "List a file's version history" },
  { group: "Versions", method: "POST", path: "/files/{id}/versions/{version}/restore", desc: "Restore a previous version" },
  { group: "Shares", method: "POST", path: "/shares/", desc: "Create a share link", body: { file_id: "…", expires_in_days: 7, download_limit: 100 } },
  { group: "Shares", method: "GET", path: "/shares/", desc: "List your share links" },
  { group: "Shares", method: "DELETE", path: "/shares/{id}", desc: "Revoke a share link" },
  { group: "Common", method: "GET", path: "/files/common", desc: "List organisation-wide Common files" },
  { group: "Common", method: "POST", path: "/files/common/upload", desc: "Upload to Common (unlimited, off-quota)", upload: true },
  { group: "Teams", method: "GET", path: "/teams/", desc: "List teams you belong to" },
  { group: "Teams", method: "POST", path: "/teams/", desc: "Create a team", body: { name: "Finance", description: "Finance dept drive", storage_quota: 0 } },
  { group: "Teams", method: "POST", path: "/teams/{id}/members", desc: "Add a member by email", body: { email: "cfo@corp.com", role: "member" } },
  { group: "AI", method: "GET", path: "/ai/search", desc: "Semantic search over your files", query: "q" },
  { group: "AI", method: "POST", path: "/ai/ask", desc: "Ask a question about your files (RAG)", body: { question: "What is our leave policy?" } },
];

const methodColor: Record<string, string> = {
  GET: "text-success", POST: "text-primary", PUT: "text-warning", PATCH: "text-warning", DELETE: "text-danger",
};

// ── Per-language snippet generators ─────────────────────────────
function fullURL(base: string, e: Endpoint) {
  const qs = e.query ? `?${e.query.split(",")[0].trim()}=…` : "";
  return `${base}${e.path}${qs}`;
}

function snippet(lang: Lang, base: string, e: Endpoint): string {
  const url = fullURL(base, e);
  switch (lang) {
    case "cURL": {
      const l = [`curl -X ${e.method} "${url}" \\`, `  -H "X-API-Key: ${KEY}"`];
      if (e.upload) l.push(` \\\n  -F "file=@./document.pdf"`);
      else if (e.body) l.push(` \\\n  -H "Content-Type: application/json" \\\n  -d '${JSON.stringify(e.body)}'`);
      if (e.download) l.push(` \\\n  -o ./download.bin`);
      return l.join("");
    }
    case "JavaScript": {
      if (e.upload) {
        return `const form = new FormData();\nform.append("file", fileInput.files[0]);\n\nconst res = await fetch("${url}", {\n  method: "${e.method}",\n  headers: { "X-API-Key": "${KEY}" },\n  body: form,\n});\nconsole.log(await res.json());`;
      }
      const opts = [`  method: "${e.method}"`, `  headers: { "X-API-Key": "${KEY}"${e.body ? `, "Content-Type": "application/json"` : ""} }`];
      if (e.body) opts.push(`  body: JSON.stringify(${JSON.stringify(e.body, null, 2).replace(/\n/g, "\n  ")})`);
      const tail = e.download ? `const blob = await res.blob();` : `console.log(await res.json());`;
      return `const res = await fetch("${url}", {\n${opts.join(",\n")},\n});\n${tail}`;
    }
    case "Python": {
      if (e.upload) {
        return `import requests\n\nwith open("document.pdf", "rb") as f:\n    r = requests.${e.method.toLowerCase()}(\n        "${url}",\n        headers={"X-API-Key": "${KEY}"},\n        files={"file": f},\n    )\nprint(r.json())`;
      }
      const args = [`    "${url}"`, `    headers={"X-API-Key": "${KEY}"}`];
      if (e.body) args.push(`    json=${pyDict(e.body)}`);
      const tail = e.download ? `open("download.bin", "wb").write(r.content)` : `print(r.json())`;
      return `import requests\n\nr = requests.${e.method.toLowerCase()}(\n${args.join(",\n")},\n)\n${tail}`;
    }
    case "Go": {
      if (e.body) {
        return `package main\n\nimport (\n\t"bytes"\n\t"net/http"\n)\n\nfunc main() {\n\tbody := bytes.NewBufferString(\`${JSON.stringify(e.body)}\`)\n\treq, _ := http.NewRequest("${e.method}", "${url}", body)\n\treq.Header.Set("X-API-Key", "${KEY}")\n\treq.Header.Set("Content-Type", "application/json")\n\tres, _ := http.DefaultClient.Do(req)\n\tdefer res.Body.Close()\n}`;
      }
      return `package main\n\nimport "net/http"\n\nfunc main() {\n\treq, _ := http.NewRequest("${e.method}", "${url}", nil)\n\treq.Header.Set("X-API-Key", "${KEY}")\n\tres, _ := http.DefaultClient.Do(req)\n\tdefer res.Body.Close()\n}`;
    }
  }
}

function pyDict(obj: Record<string, unknown>): string {
  return JSON.stringify(obj).replace(/true/g, "True").replace(/false/g, "False").replace(/null/g, "None");
}

function gettingStarted(lang: Lang, base: string): string {
  const e: Endpoint = { group: "", method: "GET", path: "/files/", desc: "" };
  return snippet(lang, base, e);
}

export function ApiDocs() {
  const [base, setBase] = useState("http://localhost/api/v1");
  const [lang, setLang] = useState<Lang>("cURL");
  const [copied, setCopied] = useState<string | null>(null);
  useEffect(() => setBase(`${window.location.origin}/api/v1`), []);

  function copy(text: string, id: string) {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 1200);
  }

  function downloadPostman() {
    const collection = {
      info: { name: `${BRAND.company.product} API`, schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" },
      variable: [{ key: "baseUrl", value: base }, { key: "apiKey", value: "" }],
      item: ENDPOINTS.map((e) => ({
        name: `${e.method} ${e.path}`,
        request: {
          method: e.method,
          header: [{ key: "X-API-Key", value: "{{apiKey}}" }, ...(e.body ? [{ key: "Content-Type", value: "application/json" }] : [])],
          url: { raw: `{{baseUrl}}${e.path}`, host: ["{{baseUrl}}"], path: e.path.split("/").filter(Boolean) },
          ...(e.body ? { body: { mode: "raw", raw: JSON.stringify(e.body, null, 2) } } : {}),
        },
      })),
    };
    triggerDownload(new Blob([JSON.stringify(collection, null, 2)], { type: "application/json" }), "sapphire-sftp.postman_collection.json");
  }

  function downloadPdf() {
    const rows = ENDPOINTS.map((e) => `<tr><td class="m ${e.method}">${e.method}</td><td class="p">${e.path}</td><td>${e.desc}</td></tr>`).join("");
    const c = BRAND.colors.primary;
    const html = `<!doctype html><html><head><meta charset="utf-8"><title>${BRAND.company.product} API</title>
<style>body{font-family:-apple-system,Segoe UI,Roboto,sans-serif;color:#18181b;margin:40px;}h1{color:${c};margin:0 0 4px;}.sub{color:#666;margin:0 0 24px;font-size:13px;}h2{color:${c};border-bottom:2px solid #eee;padding-bottom:4px;margin-top:28px;font-size:16px;}code{background:#f3f5f9;padding:2px 6px;border-radius:4px;font-size:12px;}table{width:100%;border-collapse:collapse;margin-top:8px;font-size:12px;}td{border:1px solid #e5e7eb;padding:6px 8px;vertical-align:top;}.m{font-weight:700;width:60px;}.GET{color:#16a34a}.POST{color:#4f46e5}.PUT{color:#d97706}.DELETE{color:#dc2626}.p{font-family:monospace;white-space:nowrap;}</style></head><body>
<h1>${BRAND.company.product} — API Reference</h1><p class="sub">Base URL: <code>${base}</code> · Auth: <code>X-API-Key: &lt;key&gt;</code></p>
<h2>Endpoints</h2><table><tr><td class="m">METHOD</td><td class="p">PATH</td><td>DESCRIPTION</td></tr>${rows}</table></body></html>`;
    const w = window.open("", "_blank");
    if (!w) return;
    w.document.write(html); w.document.close(); w.focus();
    setTimeout(() => w.print(), 300);
  }

  const langTabs = (
    <div className="flex gap-1 rounded-lg bg-surface-2 p-1">
      {LANGS.map((l) => (
        <button key={l} onClick={() => setLang(l)}
          className={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${lang === l ? "bg-primary text-primary-foreground shadow-sm" : "text-muted hover:text-foreground"}`}>
          {l}
        </button>
      ))}
    </div>
  );

  function CodeBlock({ code, id }: { code: string; id: string }) {
    return (
      <div className="relative border-t border-border bg-[#0d1214]">
        <pre className="overflow-x-auto px-3 py-2.5 font-mono text-[11px] leading-relaxed text-zinc-100">{code}</pre>
        <button onClick={() => copy(code, id)} className="absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded text-white/50 hover:bg-white/10 hover:text-white">
          {copied === id ? <Check size={13} className="text-emerald-400" /> : <Copy size={13} />}
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* What is an API key? */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><KeyRound size={16} /> What is an API key — and why use one?</CardTitle>
          <p className="text-sm text-muted">
            An API key is a long secret string that lets a <strong>program</strong> (not a person) talk to {BRAND.company.product} on your behalf —
            like a password made just for scripts. You paste it into a header, and the software can do exactly what you allowed it to. No browser, no login screen.
          </p>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
            {USE_CASES.map((u) => (
              <div key={u.title} className="flex gap-3 rounded-lg border border-border bg-surface-2 p-3">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary"><u.icon size={18} /></span>
                <div><p className="text-sm font-medium">{u.title}</p><p className="text-xs text-muted">{u.text}</p></div>
              </div>
            ))}
          </div>
          <div className="mt-3 rounded-lg border border-warning/30 bg-warning/5 p-3 text-xs text-muted">
            <strong className="text-foreground">Keep it secret.</strong> Anyone with the key can act within its scopes. Never commit it to git or share it in chat.
            Give each key only the scopes it needs, and revoke it the moment it leaks.
          </div>
        </CardContent>
      </Card>

      {/* Reference */}
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CardTitle className="flex items-center gap-2"><Terminal size={16} /> API Reference</CardTitle>
            <div className="flex items-center gap-2">
              {langTabs}
              <Button variant="outline" size="sm" onClick={downloadPostman}><FileJson size={15} /> Postman</Button>
              <Button variant="outline" size="sm" onClick={downloadPdf}><FileText size={15} /> PDF</Button>
            </div>
          </div>
          <p className="text-sm text-muted">
            Every endpoint below, with a copy-paste example in your language. Base URL:{" "}
            <code className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-xs">{base}</code>
          </p>
        </CardHeader>
        <CardContent className="space-y-5">
          {/* Authentication */}
          <div className="rounded-lg border border-border bg-surface-2 p-3">
            <p className="mb-1 flex items-center gap-1.5 text-sm font-medium"><ShieldCheck size={14} className="text-primary" /> Step 1 — Authenticate</p>
            <p className="mb-2 text-xs text-muted">Send your key on <em>every</em> request as this header. That's the whole login — no cookies, no session.</p>
            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate font-mono text-xs">X-API-Key: {KEY}</code>
              <button onClick={() => copy(`X-API-Key: ${KEY}`, "hdr")} className="text-muted hover:text-foreground">
                {copied === "hdr" ? <Check size={14} className="text-success" /> : <Copy size={14} />}
              </button>
            </div>
          </div>

          {/* Getting started */}
          <div className="overflow-hidden rounded-lg border border-border">
            <div className="flex items-center gap-2 px-3 py-2">
              <Rocket size={14} className="text-primary" />
              <span className="text-sm font-medium">Step 2 — Your first call</span>
              <span className="hidden text-xs text-muted sm:block">list your files ({lang})</span>
            </div>
            <CodeBlock code={gettingStarted(lang, base)} id="gs" />
          </div>

          {/* Endpoints grouped */}
          {[...new Set(ENDPOINTS.map((e) => e.group))].map((group) => (
            <div key={group}>
              <p className="eyebrow mb-2">{group}</p>
              <div className="space-y-2">
                {ENDPOINTS.filter((e) => e.group === group).map((e) => {
                  const id = e.method + e.path;
                  return (
                    <div key={id} className="overflow-hidden rounded-lg border border-border">
                      <div className="flex items-center gap-2 px-3 py-2">
                        <span className={`w-14 shrink-0 font-mono text-xs font-semibold ${methodColor[e.method]}`}>{e.method}</span>
                        <code className="min-w-0 flex-1 truncate font-mono text-xs">{e.path}</code>
                        <span className="hidden text-xs text-muted sm:block">{e.desc}</span>
                      </div>
                      <CodeBlock code={snippet(lang, base, e)} id={id} />
                    </div>
                  );
                })}
              </div>
            </div>
          ))}

          <div className="flex flex-wrap items-center gap-1.5 text-xs text-muted">
            <span>Scopes limit what a key can do:</span>
            {["files.read", "files.upload", "files.write", "files.delete", "files.share"].map((s) => <Badge key={s}>{s}</Badge>)}
          </div>
          <p className="text-xs text-muted">
            Responses use a uniform envelope (<code className="font-mono">success</code>, <code className="font-mono">data</code>, <code className="font-mono">message</code>).
            Errors are <a className="text-primary underline" href="https://www.rfc-editor.org/rfc/rfc7807" target="_blank" rel="noreferrer">RFC 7807</a> problem+json.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url; a.download = filename; a.click();
  URL.revokeObjectURL(url);
}
