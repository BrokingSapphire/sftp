"use client";

import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Inbox, Download, Trash2, Check, Eye, ShieldAlert, PartyPopper } from "lucide-react";
import { filesApi } from "@/lib/endpoints";
import type { FileItem } from "@/lib/types";
import { PageHeader } from "@/components/files/file-list";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/misc";
import { Avatar } from "@/components/ui/avatar";
import { EmptyState } from "@/components/ui/empty-state";
import { fileIcon } from "@/components/files/icon";
import { FilePreview } from "@/components/files/file-preview";
import { StaggerList, StaggerItem } from "@/components/motion";
import { formatBytes } from "@/lib/utils";

function deadlineLabel(iso?: string) {
  if (!iso) return "";
  const days = Math.ceil((new Date(iso).getTime() - Date.now()) / 86400000);
  if (days <= 0) return "overdue";
  return `${days} day${days === 1 ? "" : "s"} left`;
}

export default function InheritedPage() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["inherited"], queryFn: () => filesApi.inherited() });
  const [preview, setPreview] = useState<number | null>(null);
  const groups = q.data ?? [];
  const refresh = () => qc.invalidateQueries({ queryKey: ["inherited"] });

  // Flat list (across groups) for the preview lightbox — stable index.
  const allFiles = useMemo(() => groups.flatMap((g) => g.files), [groups]);
  const total = allFiles.length;

  async function keep(f: FileItem) {
    try { await filesApi.keepFile(f.id); toast.success(`Kept “${f.name}”`); refresh(); }
    catch { toast.error("Could not keep file"); }
  }
  async function remove(f: FileItem) {
    if (!confirm(`Permanently delete “${f.name}”?`)) return;
    try { await filesApi.deleteFile(f.id); toast.success("Deleted"); refresh(); }
    catch { toast.error("Could not delete"); }
  }
  async function keepGroup(files: FileItem[]) {
    try { await Promise.all(files.map((f) => filesApi.keepFile(f.id))); toast.success("Files kept"); refresh(); }
    catch { toast.error("Some files could not be kept"); }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PageHeader title="Inherited files" subtitle="Files handed over to you, grouped by who they came from" />
        {total > 0 && <Button size="sm" onClick={() => keepGroup(allFiles)}><Check size={16} /> Keep everything</Button>}
      </div>

      {total > 0 && (
        <div className="flex items-start gap-2 rounded-xl border border-warning/40 bg-warning/10 px-4 py-3 text-sm text-warning">
          <ShieldAlert size={18} className="mt-0.5 shrink-0" />
          <p>
            Review each file — <strong>keep</strong> it or <strong>delete</strong> it. Act before the deadline
            or your account may be disabled. Nothing is deleted automatically.
          </p>
        </div>
      )}

      {q.isLoading && <div className="rounded-xl border border-border bg-surface p-4">{[...Array(4)].map((_, i) => <Skeleton key={i} className="mb-2 h-9 w-full" />)}</div>}

      {!q.isLoading && total === 0 && (
        <EmptyState
          icon={PartyPopper}
          title="Inbox zero, inheritance edition 🎉"
          subtitle="Nobody's left you their digital shoebox of files. Enjoy the peace — or go make some files of your own."
        />
      )}

      {groups.map((g, gi) => {
        // Offset so the preview index matches the flat allFiles array.
        const offset = groups.slice(0, gi).reduce((n, x) => n + x.files.length, 0);
        return (
          <section key={g.from_id || gi} className="rounded-xl border border-border bg-surface">
            <div className="flex items-center gap-3 border-b border-border px-4 py-2.5">
              <Avatar userId={g.from_id} name={g.from_name} size={30} />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-semibold">{g.from_name}</p>
                {g.from_email && <p className="truncate text-xs text-muted">{g.from_email}</p>}
              </div>
              <span className="rounded-full bg-surface-2 px-2 py-0.5 text-xs text-muted">{g.files.length} file{g.files.length === 1 ? "" : "s"}</span>
              <Button variant="outline" size="sm" onClick={() => keepGroup(g.files)}><Check size={14} /> Keep all</Button>
            </div>
            <StaggerList>
              {g.files.map((f, i) => {
                const idx = offset + i;
                const dl = deadlineLabel(f.transfer_deadline);
                return (
                  <StaggerItem key={f.id} className="grid grid-cols-[1fr_auto_7rem_auto] items-center gap-4 border-b border-border/50 px-4 py-2.5 last:border-0 transition-colors hover:bg-surface-2">
                    <button onClick={() => setPreview(idx)} className="flex min-w-0 items-center gap-3 text-left">
                      {fileIcon(f.extension, 18)}
                      <span className="truncate text-sm font-medium">{f.name}</span>
                    </button>
                    <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
                    <span className={`text-xs font-medium ${dl === "overdue" ? "text-danger" : "text-warning"}`}>{dl}</span>
                    <div className="flex items-center justify-end gap-1">
                      <button title="Preview" onClick={() => setPreview(idx)} className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-border hover:text-foreground"><Eye size={15} /></button>
                      <a href={filesApi.downloadUrl(f.id)}><button title="Download" className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-border hover:text-foreground"><Download size={15} /></button></a>
                      <Button variant="outline" size="sm" onClick={() => keep(f)}><Check size={14} /> Keep</Button>
                      <Button variant="ghost" size="sm" onClick={() => remove(f)} className="text-danger"><Trash2 size={14} /></Button>
                    </div>
                  </StaggerItem>
                );
              })}
            </StaggerList>
          </section>
        );
      })}

      {preview !== null && allFiles[preview] && (
        <FilePreview files={allFiles} index={preview} onChangeIndex={setPreview} onClose={() => setPreview(null)} onChanged={refresh} />
      )}
    </div>
  );
}
