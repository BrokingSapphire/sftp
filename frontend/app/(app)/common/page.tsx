"use client";

import { useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Globe, Upload, FolderPlus, FolderUp, Download, Trash2, Eye, Folder, ChevronRight, Home } from "lucide-react";
import { commonApi, filesApi, type CommonFile } from "@/lib/endpoints";
import type { FileItem, FolderItem } from "@/lib/types";
import { PageHeader } from "@/components/files/file-list";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/misc";
import { UploadZone } from "@/components/files/upload-zone";
import { fileIcon } from "@/components/files/icon";
import { Avatar } from "@/components/ui/avatar";
import { EmptyState } from "@/components/ui/empty-state";
import { FilePreview } from "@/components/files/file-preview";
import { StaggerList, StaggerItem } from "@/components/motion";
import { formatBytes, timeAgo, cn } from "@/lib/utils";

interface Crumb { id?: string; name: string }

export default function CommonPage() {
  const qc = useQueryClient();
  const [crumbs, setCrumbs] = useState<Crumb[]>([{ name: "Common" }]);
  const current = crumbs[crumbs.length - 1];
  const inputRef = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<number | null>(null);

  const q = useQuery({ queryKey: ["common", current.id ?? "root"], queryFn: () => commonApi.browse(current.id) });
  const folders = q.data?.folders ?? [];
  const files = q.data?.files ?? [];
  const refresh = () => qc.invalidateQueries({ queryKey: ["common", current.id ?? "root"] });

  function openFolder(f: FolderItem) { setCrumbs((c) => [...c, { id: f.id, name: f.name }]); }
  function goTo(i: number) { setCrumbs((c) => c.slice(0, i + 1)); }

  async function newFolder() {
    const name = prompt("New folder name (in Common)");
    if (!name) return;
    try { await commonApi.createFolder(name, current.id); toast.success("Folder created"); refresh(); }
    catch { toast.error("Could not create folder"); }
  }
  async function upload(fs: File[]) {
    for (const file of fs) {
      const t = toast.loading(`Uploading ${file.name} to Common…`, { position: "bottom-right" });
      try {
        await commonApi.upload(file, current.id, (pct) => toast.loading(`Uploading ${file.name}… ${pct}%`, { id: t, position: "bottom-right" }));
        toast.success(`Added ${file.name}`, { id: t, position: "bottom-right" });
      } catch { toast.error(`Failed to upload ${file.name}`, { id: t, position: "bottom-right" }); }
    }
    refresh();
  }
  async function remove(f: CommonFile) {
    if (!confirm(`Delete "${f.name}" from Common? This cannot be undone.`)) return;
    try { await commonApi.remove(f.id); toast.success("Deleted from Common"); refresh(); }
    catch { toast.error("Could not delete"); }
  }

  const empty = !q.isLoading && folders.length === 0 && files.length === 0;

  return (
    <div className="mx-auto max-w-6xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PageHeader title="Common" subtitle="Organisation-wide files — visible to everyone. Uploaders (or admins) can delete." />
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={newFolder}><FolderPlus size={16} /> New folder</Button>
          <Button size="sm" onClick={() => inputRef.current?.click()}><Upload size={16} /> Upload</Button>
          <input ref={inputRef} type="file" multiple hidden onChange={(e) => {
            const fs = Array.from(e.target.files ?? []);
            if (fs.length) upload(fs);
            e.target.value = "";
          }} />
        </div>
      </div>

      {/* Breadcrumb */}
      <nav className="flex min-w-0 items-center gap-1 text-sm">
        {crumbs.map((c, i) => (
          <span key={i} className="flex items-center gap-1">
            {i > 0 && <ChevronRight size={14} className="text-muted" />}
            <button onClick={() => goTo(i)}
              className={cn("flex items-center gap-1 rounded px-1.5 py-0.5 hover:bg-surface-2", i === crumbs.length - 1 ? "font-semibold" : "text-muted")}>
              {i === 0 && <Home size={14} />}{c.name}
            </button>
          </span>
        ))}
      </nav>

      <UploadZone onFiles={upload}>
        {q.isLoading ? (
          <div className="rounded-xl border border-border bg-surface p-4">
            {[...Array(6)].map((_, i) => <Skeleton key={i} className="mb-2 h-9 w-full" />)}
          </div>
        ) : empty ? (
          <EmptyState
            icon={Globe}
            title="The Common room is quiet"
            subtitle="No org-wide files here yet. Make a folder or drop something in — everyone can see it, and it's unlimited (your quota won't even flinch)."
            action={<Button size="sm" onClick={newFolder}><FolderPlus size={16} /> New folder</Button>}
          />
        ) : (
          <div className="rounded-xl border border-border bg-surface">
            <div className="grid grid-cols-[1fr_11rem_6rem_7rem] gap-4 border-b border-border px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
              <span>Name</span><span>Uploaded by</span><span>Size</span><span className="text-right">Added</span>
            </div>
            <StaggerList>
              {/* Folders first */}
              {folders.map((f) => (
                <StaggerItem key={f.id} onDoubleClick={() => openFolder(f)}
                  className="group grid grid-cols-[1fr_11rem_6rem_7rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2">
                  <button onClick={() => openFolder(f)} className="flex min-w-0 items-center gap-3 text-left">
                    <Folder size={18} className="text-primary" />
                    <span className="truncate text-sm font-medium">{f.name}</span>
                  </button>
                  <span className="text-xs text-muted">—</span>
                  <span className="text-xs text-muted">—</span>
                  <span className="text-right text-xs text-muted">folder</span>
                </StaggerItem>
              ))}
              {/* Files */}
              {files.map((f, i) => (
                <StaggerItem key={f.id} className="group grid grid-cols-[1fr_11rem_6rem_7rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2">
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
