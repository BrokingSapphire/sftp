"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Copy, Link2, Lock, Trash2 } from "lucide-react";
import { sharesApi } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Card, CardContent } from "@/components/ui/card";
import { Badge, Skeleton } from "@/components/ui/misc";
import { timeAgo } from "@/lib/utils";

export default function SharesPage() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["shares"], queryFn: () => sharesApi.list() });

  async function copy(token: string) {
    const url = `${window.location.origin}/api/v1/share/${token}/download`;
    await navigator.clipboard.writeText(url).catch(() => {});
    toast.success("Link copied");
  }
  async function revoke(id: string) {
    try { await sharesApi.revoke(id); toast.success("Revoked"); qc.invalidateQueries({ queryKey: ["shares"] }); }
    catch { toast.error("Failed"); }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-4">
      <PageHeader title="Shared links" subtitle="Public and password-protected links you have created" />
      {q.isLoading && <Skeleton className="h-40 w-full" />}
      {!q.isLoading && q.data?.length === 0 && (
        <Card><CardContent className="flex flex-col items-center gap-2 py-16 text-muted">
          <Link2 size={36} /><p className="text-sm">No share links yet. Share a file from the explorer.</p>
        </CardContent></Card>
      )}
      <div className="space-y-2">
        {q.data?.map((s) => (
          <Card key={s.id}>
            <CardContent className="flex items-center gap-4 p-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                {s.has_password ? <Lock size={18} /> : <Link2 size={18} />}
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate font-mono text-sm">/{s.token}</p>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted">
                  <Badge>{s.permission}</Badge>
                  {s.has_password && <Badge>password</Badge>}
                  {s.download_limit != null && <span>{s.download_count}/{s.download_limit} downloads</span>}
                  {!s.is_active && <span className="text-danger">revoked</span>}
                  <span>· {timeAgo(s.created_at)}</span>
                </div>
              </div>
              <button title="Copy" onClick={() => copy(s.token)} className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-surface-2"><Copy size={16} /></button>
              <button title="Revoke" onClick={() => revoke(s.id)} className="flex h-8 w-8 items-center justify-center rounded-md text-danger hover:bg-surface-2"><Trash2 size={16} /></button>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
