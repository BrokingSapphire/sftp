"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Copy, Link2, Lock, Trash2, Share2 } from "lucide-react";
import { sharesApi } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Card, CardContent } from "@/components/ui/card";
import { Badge, Skeleton, StatusBadge } from "@/components/ui/misc";
import { timeAgo } from "@/lib/utils";

export default function SharesPage() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["shares"], queryFn: () => sharesApi.list() });

  async function copy(token: string) {
    // Link to the friendly public share page (not the raw download endpoint).
    const url = `${window.location.origin}/share/${token}`;
    await navigator.clipboard.writeText(url).catch(() => {});
    toast.success("Link copied");
  }
  async function revoke(id: string) {
    try { await sharesApi.revoke(id); toast.success("Revoked"); qc.invalidateQueries({ queryKey: ["shares"] }); }
    catch { toast.error("Failed"); }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-4">
      <PageHeader icon={Share2} title="Shared links" subtitle="Public and password-protected download links you have created. Revoke any link to disable it instantly." />
      {q.isLoading && <Skeleton className="h-40 w-full" />}
      {!q.isLoading && !q.data?.length && (
        <Card><CardContent className="flex flex-col items-center gap-2 py-16 text-muted">
          <Link2 size={36} /><p className="text-sm">No share links yet. Right-click a file in the explorer and choose <span className="font-medium text-foreground">Share</span> to create one.</p>
        </CardContent></Card>
      )}
      <div className="space-y-2">
        {q.data?.map((s) => (
          <Card key={s.id}>
            <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
              <div className="flex min-w-0 flex-1 items-center gap-3">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  {s.has_password ? <Lock size={18} /> : <Link2 size={18} />}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate font-mono text-sm">/share/{s.token}</p>
                  <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted">
                    <StatusBadge tone={s.is_active ? "success" : "danger"} dot>{s.is_active ? "Active" : "Revoked"}</StatusBadge>
                    {s.kind === "folder" && <Badge>Folder</Badge>}
                    <Badge className="capitalize">{s.permission}</Badge>
                    {s.has_password && <StatusBadge tone="warning">Password</StatusBadge>}
                    {s.download_limit != null && <span>{s.download_count}/{s.download_limit} downloads</span>}
                    <span>· created {timeAgo(s.created_at)}</span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2 self-end sm:self-auto">
                <button title="Copy link" onClick={() => copy(s.token)} className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-surface-2"><Copy size={16} /></button>
                <button title="Revoke link" onClick={() => revoke(s.id)} className="flex h-8 w-8 items-center justify-center rounded-md text-danger hover:bg-surface-2"><Trash2 size={16} /></button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
