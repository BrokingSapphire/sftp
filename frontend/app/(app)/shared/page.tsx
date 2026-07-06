"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Users, Download, Eye, Pencil, Glasses } from "lucide-react";
import { filesApi } from "@/lib/endpoints";
import type { FileItem } from "@/lib/types";
import { PageHeader } from "@/components/files/file-list";
import { Skeleton } from "@/components/ui/misc";
import { Avatar } from "@/components/ui/avatar";
import { fileIcon } from "@/components/files/icon";
import { FilePreview } from "@/components/files/file-preview";
import { StaggerList, StaggerItem } from "@/components/motion";
import { formatBytes, timeAgo } from "@/lib/utils";

export default function SharedPage() {
  const q = useQuery({ queryKey: ["shared-with-me"], queryFn: () => filesApi.sharedWithMe() });
  const [preview, setPreview] = useState<number | null>(null);
  const files = q.data ?? [];

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <PageHeader title="Shared with me" subtitle="Files other people have shared with you" />

      {q.isLoading && <div className="rounded-xl border border-border bg-surface p-4">{[...Array(5)].map((_, i) => <Skeleton key={i} className="mb-2 h-9 w-full" />)}</div>}

      {!q.isLoading && files.length === 0 && (
        <div className="flex min-h-[18rem] flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-surface text-muted">
          <Users size={40} />
          <p className="text-sm">Nothing shared with you yet.</p>
        </div>
      )}

      {files.length > 0 && (
        <div className="rounded-xl border border-border bg-surface">
          <div className="grid grid-cols-[1fr_11rem_6rem_5rem_7rem] gap-4 border-b border-border px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
            <span>Name</span><span>Owner</span><span>Access</span><span>Size</span><span className="text-right">Shared</span>
          </div>
          <StaggerList>
            {files.map((f, i) => (
              <StaggerItem key={f.id} className="group grid grid-cols-[1fr_11rem_6rem_5rem_7rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2">
                <button onClick={() => setPreview(i)} className="flex min-w-0 items-center gap-3 text-left">
                  {fileIcon(f.extension, 18)}
                  <span className="truncate text-sm font-medium">{f.name}</span>
                </button>
                <span className="flex min-w-0 items-center gap-1.5 text-xs text-muted">
                  <Avatar userId={f.owner_id} name={f.owner_name} hasAvatar={f.owner_has_avatar} size={20} />
                  <span className="truncate">{f.owner_name}</span>
                </span>
                <span className={`flex items-center gap-1 text-xs font-medium ${f.can_write ? "text-primary" : "text-muted"}`}>
                  {f.can_write ? <Pencil size={12} /> : <Glasses size={12} />} {f.can_write ? "Editor" : "Viewer"}
                </span>
                <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
                <div className="flex items-center justify-end gap-1">
                  <span className="text-xs text-muted group-hover:hidden">{timeAgo(f.shared_at)}</span>
                  <div className="hidden gap-1 group-hover:flex">
                    <button title="Preview" onClick={() => setPreview(i)} className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-border hover:text-foreground"><Eye size={15} /></button>
                    <a href={filesApi.downloadUrl(f.id)}><button title="Download" className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-border hover:text-foreground"><Download size={15} /></button></a>
                  </div>
                </div>
              </StaggerItem>
            ))}
          </StaggerList>
        </div>
      )}

      {preview !== null && files[preview] && (
        <FilePreview files={files as unknown as FileItem[]} index={preview} onChangeIndex={setPreview} onClose={() => setPreview(null)} />
      )}
    </div>
  );
}
