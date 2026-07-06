"use client";

import { useEffect, useState } from "react";
import { Copy, Check, Terminal } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/misc";

interface Endpoint {
  method: string;
  path: string;
  desc: string;
  curl: (base: string) => string;
}

const KEY = "sftp_XXXXXXXX_YYYYYYYYYYYYYYYYYYYYYYY";

const ENDPOINTS: Endpoint[] = [
  {
    method: "GET", path: "/files/", desc: "List folder contents (root, or ?folder_id=)",
    curl: (b) => `curl -H "X-API-Key: ${KEY}" \\\n  ${b}/files/`,
  },
  {
    method: "POST", path: "/files/upload", desc: "Upload a file (multipart)",
    curl: (b) => `curl -H "X-API-Key: ${KEY}" \\\n  -F "file=@./report.pdf" \\\n  ${b}/files/upload`,
  },
  {
    method: "GET", path: "/files/{id}/download", desc: "Download a file (supports Range)",
    curl: (b) => `curl -H "X-API-Key: ${KEY}" -OJ \\\n  ${b}/files/{id}/download`,
  },
  {
    method: "POST", path: "/folders/", desc: "Create a folder",
    curl: (b) => `curl -H "X-API-Key: ${KEY}" -H "Content-Type: application/json" \\\n  -d '{"name":"Reports"}' ${b}/folders/`,
  },
  {
    method: "GET", path: "/files/search?q=", desc: "Search files by name",
    curl: (b) => `curl -H "X-API-Key: ${KEY}" \\\n  "${b}/files/search?q=invoice"`,
  },
  {
    method: "POST", path: "/uploads/", desc: "Start a resumable upload (large files)",
    curl: (b) => `curl -H "X-API-Key: ${KEY}" -H "Content-Type: application/json" \\\n  -d '{"filename":"big.zip","total_size":5368709120,"chunk_size":8388608}' \\\n  ${b}/uploads/`,
  },
];

const methodColor: Record<string, string> = {
  GET: "text-success",
  POST: "text-primary",
  PUT: "text-warning",
  DELETE: "text-danger",
};

export function ApiDocs() {
  const [base, setBase] = useState("http://localhost/api/v1");
  const [copied, setCopied] = useState<string | null>(null);

  useEffect(() => setBase(`${window.location.origin}/api/v1`), []);

  function copy(text: string, id: string) {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 1200);
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2"><Terminal size={16} /> Using the API</CardTitle>
        <p className="text-sm text-muted">
          Authenticate any request with your key. Base URL:{" "}
          <code className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-xs">{base}</code>
        </p>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="rounded-lg border border-border bg-surface-2 p-3">
          <p className="eyebrow mb-1">Authentication header</p>
          <div className="flex items-center gap-2">
            <code className="min-w-0 flex-1 truncate font-mono text-xs">X-API-Key: {KEY}</code>
            <button onClick={() => copy(`X-API-Key: ${KEY}`, "hdr")} className="text-muted hover:text-foreground">
              {copied === "hdr" ? <Check size={14} className="text-success" /> : <Copy size={14} />}
            </button>
          </div>
          <p className="mt-2 text-xs text-muted">
            A JWT works too: <code className="font-mono">Authorization: Bearer &lt;token&gt;</code>. Scopes limit what a key can do.
          </p>
        </div>

        <div className="space-y-2">
          {ENDPOINTS.map((e) => {
            const cmd = e.curl(base);
            const id = e.method + e.path;
            return (
              <div key={id} className="rounded-lg border border-border">
                <div className="flex items-center gap-2 px-3 py-2">
                  <span className={`w-12 shrink-0 font-mono text-xs font-semibold ${methodColor[e.method]}`}>{e.method}</span>
                  <code className="min-w-0 flex-1 truncate font-mono text-xs">{e.path}</code>
                  <span className="hidden text-xs text-muted sm:block">{e.desc}</span>
                </div>
                <div className="relative border-t border-border bg-[#0d1214]">
                  <pre className="overflow-x-auto px-3 py-2.5 font-mono text-[11px] leading-relaxed text-zinc-100">{cmd}</pre>
                  <button
                    onClick={() => copy(cmd, id)}
                    className="absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded text-white/50 hover:bg-white/10 hover:text-white"
                  >
                    {copied === id ? <Check size={13} className="text-emerald-400" /> : <Copy size={13} />}
                  </button>
                </div>
              </div>
            );
          })}
        </div>

        <div className="flex flex-wrap gap-1.5 text-xs text-muted">
          <span>Available scopes:</span>
          {["files.read", "files.upload", "files.write", "files.delete", "files.share"].map((s) => <Badge key={s}>{s}</Badge>)}
        </div>
      </CardContent>
    </Card>
  );
}
