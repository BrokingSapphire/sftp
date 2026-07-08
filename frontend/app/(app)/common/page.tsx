"use client";

import { useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Globe, Upload, FolderPlus, FolderUp, Download, Trash2, Pencil, Eye, Folder, ChevronRight, Home } from "lucide-react";
import { commonApi, filesApi, type CommonFile } from "@/lib/endpoints";
import type { FileItem, FolderItem } from "@/lib/types";
import { PageHeader } from "@/components/files/file-list";
import { Button } from "@/components/ui/button";
import { useDialogs } from "@/components/ui/dialogs";
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
  const { confirm, prompt } = useDialogs();
  const [crumbs, setCrumbs] = useState<Crumb[]>([{ name: "Common" }]);
  const current = crumbs[crumbs.length - 1];
  const inputRef = useRef<HTMLInputElement>(null);
  const folderRef = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<number | null>(null);

  const q = useQuery({ queryKey: ["common", current.id ?? "root"], queryFn: () => commonApi.browse(current.id) });
  const folders = q.data?.folders ?? [];
  const files = q.data?.files ?? [];
  const refresh = () => qc.invalidateQueries({ queryKey: ["common", current.id ?? "root"] });

  function openFolder(f: FolderItem) { setCrumbs((c) => [...c, { id: f.id, name: f.name }]); }
  function goTo(i: number) { setCrumbs((c) => c.slice(0, i + 1)); }

  async function newFolder() {
    const name = await prompt({ title: "New folder in Common", placeholder: "Folder name", confirmLabel: "Create" });
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

  // Upload a whole folder tree into Common, recreating the sub-folder structure.
  async function uploadFolder(entries: { file: File; relPath: string }[]) {
    if (!entries.length) return;
    const dirCache = new Map<string, string | undefined>([["", current.id]]);
    async function ensureDir(dir: string): Promise<string | undefined> {
      if (dirCache.has(dir)) return dirCache.get(dir);
      const parts = dir.split("/");
      const name = parts.pop()!;
      const parentId = await ensureDir(parts.join("/"));
      let id: string | undefined;
      try { id = (await commonApi.createFolder(name, parentId)).id; }
      catch { const b = await commonApi.browse(parentId); id = b.folders.find((x) => x.name === name)?.id; }
      dirCache.set(dir, id);
      return id;
    }
    const root = entries[0].relPath.split("/")[0] || "folder";
    const t = toast.loading(`Uploading folder "${root}"… 0/${entries.length}`, { position: "bottom-right" });
    let done = 0;
    try {
      for (const { file, relPath } of entries) {
        const dir = relPath.split("/").slice(0, -1).join("/");
        const fid = await ensureDir(dir);
        await commonApi.upload(file, fid);
        done++;
        toast.loading(`Uploading folder "${root}"… ${done}/${entries.length}`, { id: t, position: "bottom-right" });
      }
      toast.success(`Uploaded "${root}" (${done} files)`, { id: t, position: "bottom-right" });
    } catch { toast.error(`Folder upload failed`, { id: t, position: "bottom-right" }); }
    refresh();
  }
  async function renameFile(f: CommonFile) {
    const name = await prompt({ title: "Rename file", defaultValue: f.name, placeholder: "New name", confirmLabel: "Rename" });
    if (!name || name === f.name) return;
    try { await commonApi.renameFile(f.id, name); toast.success("File renamed"); refresh(); }
    catch { toast.error("Rename failed"); }
  }
  async function renameFolder(f: FolderItem) {
    const name = await prompt({ title: "Rename folder", defaultValue: f.name, placeholder: "New name", confirmLabel: "Rename" });
    if (!name || name === f.name) return;
    try { await commonApi.renameFolder(f.id, name); toast.success("Folder renamed"); refresh(); }
    catch { toast.error("Rename failed"); }
  }
  async function remove(f: CommonFile) {
    if (!(await confirm({ title: "Delete from Common", message: `Delete “${f.name}” from Common? This cannot be undone.`, tone: "danger", confirmLabel: "Delete" }))) return;
    try { await commonApi.remove(f.id); toast.success("Deleted from Common"); refresh(); }
    catch { toast.error("Could not delete"); }
  }

  const empty = !q.isLoading && folders.length === 0 && files.length === 0;

  return (
    <div className="mx-auto max-w-6xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={Globe} title="Common" subtitle="Organisation-wide files — visible to everyone. Uploaders (or admins) can delete." />
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" onClick={newFolder}><FolderPlus size={16} /> New folder</Button>
          <Button variant="outline" size="sm" onClick={() => folderRef.current?.click()}><FolderUp size={16} /> Upload folder</Button>
          <Button size="sm" onClick={() => inputRef.current?.click()}><Upload size={16} /> Upload</Button>
          <input ref={inputRef} type="file" multiple hidden onChange={(e) => {
            const fs = Array.from(e.target.files ?? []);
            if (fs.length) upload(fs);
            e.target.value = "";
          }} />
          <input ref={folderRef} type="file" hidden {...({ webkitdirectory: "", directory: "" } as Record<string, string>)} onChange={(e) => {
            const fs = Array.from(e.target.files ?? []);
            if (fs.length) uploadFolder(fs.map((f) => ({ file: f, relPath: (f as File & { webkitRelativePath?: string }).webkitRelativePath || f.name })));
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
          <div className="overflow-x-auto rounded-xl border border-border bg-surface">
            <div className="grid min-w-[36rem] grid-cols-[1fr_11rem_6rem_7rem] gap-4 border-b border-border px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
              <span>Name</span><span>Uploaded by</span><span>Size</span><span className="text-right">Added</span>
            </div>
            <StaggerList>
              {/* Folders first */}
              {folders.map((f) => (
                <StaggerItem key={f.id} onDoubleClick={() => openFolder(f)}
                  className="group grid min-w-[36rem] grid-cols-[1fr_11rem_6rem_7rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2">
                  <button onClick={() => openFolder(f)} className="flex min-w-0 items-center gap-3 text-left">
                    <Folder size={18} className="text-primary" />
                    <span className="truncate text-sm font-medium">{f.name}</span>
                  </button>
                  <span className="text-xs text-muted">—</span>
                  <span className="text-xs text-muted">—</span>
                  <div className="flex items-center justify-end gap-1">
                    <span className="text-right text-xs text-muted group-hover:hidden">folder</span>
                    <div className="hidden gap-1 group-hover:flex">
                      <IconBtn title="Rename" onClick={() => renameFolder(f)}><Pencil size={15} /></IconBtn>
                    </div>
                  </div>
                </StaggerItem>
              ))}
              {/* Files */}
              {files.map((f, i) => (
                <StaggerItem key={f.id} className="group grid min-w-[36rem] grid-cols-[1fr_11rem_6rem_7rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2">
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
                      {f.can_delete && <IconBtn title="Rename" onClick={() => renameFile(f)}><Pencil size={15} /></IconBtn>}
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
