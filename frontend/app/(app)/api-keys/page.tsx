"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Copy, KeyRound, Plus, Trash2 } from "lucide-react";
import { apiKeysApi } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge, Skeleton } from "@/components/ui/misc";
import { ApiDocs } from "@/components/api-docs";
import { timeAgo } from "@/lib/utils";

const SCOPES = ["files.read", "files.upload", "files.write", "files.delete", "files.share"];

export default function ApiKeysPage() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["api-keys"], queryFn: () => apiKeysApi.list() });
  const [name, setName] = useState("");
  const [scopes, setScopes] = useState<string[]>(["files.read"]);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  async function create() {
    if (!name.trim()) return toast.error("Name is required");
    setCreating(true);
    try {
      const res = await apiKeysApi.create(name.trim(), scopes);
      setNewKey(res.key);
      setName("");
      qc.invalidateQueries({ queryKey: ["api-keys"] });
    } catch { toast.error("Could not create key"); }
    finally { setCreating(false); }
  }
  async function revoke(id: string) {
    try { await apiKeysApi.revoke(id); toast.success("Revoked"); qc.invalidateQueries({ queryKey: ["api-keys"] }); }
    catch { toast.error("Failed"); }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-5">
      <PageHeader icon={KeyRound} title="API keys & developer docs" subtitle="Let scripts and apps use your files securely — create a key, then copy a ready-made snippet below" />

      {newKey && (
        <Card className="border-primary">
          <CardContent className="space-y-2 p-4">
            <p className="text-sm font-medium text-primary">Copy your new key now — it will not be shown again.</p>
            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded-md bg-surface-2 px-3 py-2 font-mono text-sm">{newKey}</code>
              <Button size="sm" variant="outline" onClick={() => { navigator.clipboard.writeText(newKey); toast.success("Copied"); }}>
                <Copy size={14} /> Copy
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setNewKey(null)}>Done</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardContent className="space-y-3 p-4">
          <Input placeholder="Key name (e.g. CI pipeline)" value={name} onChange={(e) => setName(e.target.value)} />
          <div className="flex flex-wrap gap-2">
            {SCOPES.map((s) => {
              const on = scopes.includes(s);
              return (
                <button key={s} onClick={() => setScopes((cur) => on ? cur.filter((x) => x !== s) : [...cur, s])}
                  className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${on ? "border-primary bg-primary/10 text-primary" : "border-border text-muted hover:bg-surface-2"}`}>
                  {s}
                </button>
              );
            })}
          </div>
          <Button size="sm" onClick={create} disabled={creating}><Plus size={16} /> Create key</Button>
        </CardContent>
      </Card>

      {q.isLoading && <Skeleton className="h-24 w-full" />}
      <div className="space-y-2">
        {q.data?.map((k) => (
          <Card key={k.id}>
            <CardContent className="flex items-center gap-4 p-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary"><KeyRound size={18} /></div>
              <div className="min-w-0 flex-1">
                <p className="font-medium">{k.name}</p>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted">
                  <code className="font-mono">sftp_{k.prefix}…</code>
                  {k.scopes.map((s) => <Badge key={s}>{s}</Badge>)}
                  <span>· {k.last_used_at ? `used ${timeAgo(k.last_used_at)}` : "never used"}</span>
                </div>
              </div>
              <button title="Revoke" onClick={() => revoke(k.id)} className="flex h-8 w-8 items-center justify-center rounded-md text-danger hover:bg-surface-2"><Trash2 size={16} /></button>
            </CardContent>
          </Card>
        ))}
      </div>

      <ApiDocs />
    </div>
  );
}
