"use client";

import { useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Globe, Upload, Download, Trash2, Eye } from "lucide-react";
import { commonApi, filesApi, type CommonFile } from "@/lib/endpoints";
import type { FileItem } from "@/lib/types";
import { PageHeader } from "@/components/files/file-list";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/misc";
import { UploadZone } from "@/components/files/upload-zone";
import { fileIcon } from "@/components/files/icon";
import { Avatar } from "@/components/ui/avatar";
import { FilePreview } from "@/components/files/file-preview";
import { useContextMenu, ContextMenu, type MenuItem } from "@/components/files/context-menu";
import { StaggerList, StaggerItem } from "@/components/motion";
import { formatBytes, timeAgo } from "@/lib/utils";

export default function CommonPage() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["common"], queryFn: () => commonApi.list() });
  const inputRef = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<number | null>(null);
  const ctx = useContextMenu();

  const files = q.data ?? [];
  const refresh = () => qc.invalidateQueries({ queryKey: ["common"] });

  function fileMenu(f: CommonFile, i: number): MenuItem[] {
    const items: MenuItem[] = [
      { label: "Preview", icon: Eye, onClick: () => setPreview(i) },
      { label: "Download", icon: Download, onClick: () => (window.location.href = filesApi.downloadUrl(f.id)) },
    ];
    if (f.can_delete) {
      items.push({ separator: true, label: "" });
      items.push({ label: "Delete from Common", icon: Trash2, danger: true, onClick: () => remove(f) });
    }
    return items;
  }

  async function upload(fs: File[]) {
    for (const file of fs) {
      const t = toast.loading(`Uploading ${file.name} to Common…`, { position: "bottom-right" });
      try {
        await commonApi.upload(file, (pct) => toast.loading(`Uploading ${file.name}… ${pct}%`, { id: t, position: "bottom-right" }));
        toast.success(`Added ${file.name} to Common`, { id: t, position: "bottom-right" });
      } catch { toast.error(`Failed to upload ${file.name}`, { id: t, position: "bottom-right" }); }
    }
    refresh();
  }
  async function remove(f: CommonFile) {
    if (!confirm(`Delete "${f.name}" from Common? This cannot be undone.`)) return;
    try { await commonApi.remove(f.id); toast.success("Deleted from Common"); refresh(); }
    catch { toast.error("Could not delete"); }
  }

  return (
    <div className="mx-auto max-w-6xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PageHeader title="Common" subtitle="Organisation-wide files — visible to everyone. Uploaders (or admins) can delete." />
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={() => inputRef.current?.click()}><Upload size={16} /> Add to Common</Button>
          <input ref={inputRef} type="file" multiple hidden onChange={(e) => {
            const fs = Array.from(e.target.files ?? []);
            if (fs.length) upload(fs);
            e.target.value = "";
          }} />
        </div>
      </div>

      <UploadZone onFiles={upload}>
        {q.isLoading ? (
          <div className="rounded-xl border border-border bg-surface p-4">
            {[...Array(6)].map((_, i) => <Skeleton key={i} className="mb-2 h-9 w-full" />)}
          </div>
        ) : files.length === 0 ? (
          <div className="flex min-h-[20rem] flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-surface text-muted">
            <Globe size={40} />
            <p className="text-sm">No common files yet. Drop files here to share with everyone.</p>
          </div>
        ) : (
          <div className="rounded-xl border border-border bg-surface">
            <div className="grid grid-cols-[1fr_10rem_auto_7rem] gap-4 border-b border-border px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
              <span>Name</span><span>Uploaded by</span><span>Size</span><span className="text-right">Added</span>
            </div>
            <StaggerList>
              {files.map((f, i) => (
                <StaggerItem key={f.id} onContextMenu={(e) => ctx.open(e, fileMenu(f, i))} className="group grid grid-cols-[1fr_10rem_auto_7rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2">
                  <button onClick={() => setPreview(i)} className="flex min-w-0 items-center gap-3 text-left">
                    {fileIcon(f.extension, 18)}
                    <span className="truncate text-sm font-medium">{f.name}</span>
                  </button>
                  <span className="flex min-w-0 items-center gap-1.5 text-xs text-muted">
                    <Avatar userId={f.uploader_id} name={f.uploader_name} hasAvatar={f.uploader_has_avatar} size={20} />
                    <span className="truncate">{f.uploader_name}</span>
                  </span>
                  <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
                  <div className="flex items-center justify-end gap-1">
                    <span className="text-xs text-muted group-hover:hidden">{timeAgo(f.created_at)}</span>
                    <div className="hidden gap-1 group-hover:flex">
                      <IconBtn title="Preview" onClick={() => setPreview(i)}><Eye size={15} /></IconBtn>
                      <a href={filesApi.downloadUrl(f.id)}><IconBtn title="Download" onClick={() => {}}><Download size={15} /></IconBtn></a>
                      {f.can_delete && <IconBtn title="Delete" onClick={() => remove(f)}><Trash2 size={15} /></IconBtn>}
                    </div>
                  </div>
                </StaggerItem>
              ))}
            </StaggerList>
          </div>
        )}
      </UploadZone>

      {preview !== null && files[preview] && (
        <FilePreview
          files={files as unknown as FileItem[]}
          index={preview}
          onChangeIndex={setPreview}
          onClose={() => setPreview(null)}
          onChanged={refresh}
        />
      )}

      <ContextMenu menu={ctx.menu} onClose={ctx.close} />
    </div>
  );
}

function IconBtn({ children, title, onClick }: { children: React.ReactNode; title: string; onClick: () => void }) {
  return (
    <button title={title} onClick={onClick} className="flex h-7 w-7 items-center justify-center rounded-md text-muted transition-colors hover:bg-border hover:text-foreground">
      {children}
    </button>
  );
}
