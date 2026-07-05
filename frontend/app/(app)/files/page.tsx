"use client";

import { useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  FolderPlus, Upload, Folder, Download, Star, Share2, Trash2,
  Pencil, ChevronRight, Home,
} from "lucide-react";
import { filesApi, sharesApi } from "@/lib/endpoints";
import type { FileItem, FolderItem } from "@/lib/types";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/misc";
import { UploadZone } from "@/components/files/upload-zone";
import { fileIcon } from "@/components/files/icon";
import { formatBytes, timeAgo } from "@/lib/utils";

interface Crumb { id?: string; name: string; }

export default function FilesPage() {
  const qc = useQueryClient();
  const [crumbs, setCrumbs] = useState<Crumb[]>([{ name: "Home" }]);
  const current = crumbs[crumbs.length - 1];
  const inputRef = useRef<HTMLInputElement>(null);

  const listing = useQuery({
    queryKey: ["files", current.id ?? "root"],
    queryFn: () => filesApi.list(current.id),
  });

  const refresh = () => qc.invalidateQueries({ queryKey: ["files", current.id ?? "root"] });

  function openFolder(f: FolderItem) {
    setCrumbs((c) => [...c, { id: f.id, name: f.name }]);
  }
  function goTo(i: number) {
    setCrumbs((c) => c.slice(0, i + 1));
  }

  async function uploadFiles(files: File[]) {
    for (const file of files) {
      const t = toast.loading(`Uploading ${file.name}…`);
      try {
        await filesApi.simpleUpload(file, current.id, (pct) =>
          toast.loading(`Uploading ${file.name}… ${pct}%`, { id: t }),
        );
        toast.success(`Uploaded ${file.name}`, { id: t });
      } catch {
        toast.error(`Failed to upload ${file.name}`, { id: t });
      }
    }
    refresh();
  }

  async function createFolder() {
    const name = prompt("New folder name");
    if (!name) return;
    try {
      await filesApi.createFolder(name, current.id);
      toast.success("Folder created");
      refresh();
    } catch {
      toast.error("Could not create folder");
    }
  }

  async function rename(kind: "file" | "folder", id: string, cur: string) {
    const name = prompt("Rename to", cur);
    if (!name || name === cur) return;
    try {
      kind === "file" ? await filesApi.renameFile(id, name) : await filesApi.renameFolder(id, name);
      refresh();
    } catch { toast.error("Rename failed"); }
  }

  async function trash(f: FileItem) {
    try { await filesApi.trashFile(f.id); toast.success("Moved to trash"); refresh(); }
    catch { toast.error("Delete failed"); }
  }

  async function star(f: FileItem) {
    try { await filesApi.starFile(f.id, !f.is_starred); refresh(); }
    catch { toast.error("Failed"); }
  }

  async function share(f: FileItem) {
    try {
      const res = await sharesApi.create(f.id, {});
      await navigator.clipboard.writeText(res.url).catch(() => {});
      toast.success("Share link copied to clipboard");
    } catch { toast.error("Could not create share"); }
  }

  return (
    <div className="mx-auto max-w-6xl space-y-4">
      {/* Header */}
      <div className="flex flex-wrap items-center gap-3">
        <nav className="flex min-w-0 flex-1 items-center gap-1 text-sm">
          {crumbs.map((c, i) => (
            <span key={i} className="flex items-center gap-1">
              {i > 0 && <ChevronRight size={14} className="text-muted" />}
              <button
                onClick={() => goTo(i)}
                className={`flex items-center gap-1 rounded px-1.5 py-0.5 hover:bg-surface-2 ${
                  i === crumbs.length - 1 ? "font-semibold" : "text-muted"
                }`}
              >
                {i === 0 && <Home size={14} />}
                {c.name}
              </button>
            </span>
          ))}
        </nav>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={createFolder}>
            <FolderPlus size={16} /> New folder
          </Button>
          <Button size="sm" onClick={() => inputRef.current?.click()}>
            <Upload size={16} /> Upload
          </Button>
          <input
            ref={inputRef}
            type="file"
            multiple
            hidden
            onChange={(e) => {
              const files = Array.from(e.target.files ?? []);
              if (files.length) uploadFiles(files);
              e.target.value = "";
            }}
          />
        </div>
      </div>

      {/* Listing */}
      <UploadZone onFiles={uploadFiles}>
        <div className="min-h-[24rem] rounded-xl border border-border bg-surface">
          <div className="grid grid-cols-[1fr_auto_8rem] gap-4 border-b border-border px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
            <span>Name</span><span>Size</span><span className="text-right">Modified</span>
          </div>

          {listing.isLoading &&
            [...Array(6)].map((_, i) => <div key={i} className="px-4 py-2.5"><Skeleton className="h-6 w-full" /></div>)}

          {!listing.isLoading &&
            listing.data?.folders.length === 0 &&
            listing.data?.files.length === 0 && (
              <div className="flex flex-col items-center justify-center gap-2 py-20 text-muted">
                <Folder size={40} />
                <p className="text-sm">This folder is empty. Drag files here or use Upload.</p>
              </div>
            )}

          {listing.data?.folders.map((f) => (
            <div key={f.id} className="group grid grid-cols-[1fr_auto_8rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 hover:bg-surface-2">
              <button onClick={() => openFolder(f)} className="flex min-w-0 items-center gap-3 text-left">
                <Folder size={18} className="text-primary" />
                <span className="truncate text-sm font-medium">{f.name}</span>
              </button>
              <span className="text-xs text-muted">—</span>
              <div className="flex items-center justify-end gap-1">
                <span className="text-xs text-muted group-hover:hidden">{timeAgo(f.updated_at)}</span>
                <div className="hidden gap-1 group-hover:flex">
                  <IconBtn title="Rename" onClick={() => rename("folder", f.id, f.name)}><Pencil size={15} /></IconBtn>
                  <IconBtn title="Delete" onClick={() => filesApi.deleteFolder(f.id).then(refresh).catch(() => toast.error("Folder not empty"))}><Trash2 size={15} /></IconBtn>
                </div>
              </div>
            </div>
          ))}

          {listing.data?.files.map((f) => (
            <div key={f.id} className="group grid grid-cols-[1fr_auto_8rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 hover:bg-surface-2">
              <a href={filesApi.downloadUrl(f.id)} className="flex min-w-0 items-center gap-3">
                {fileIcon(f.extension, 18)}
                <span className="truncate text-sm font-medium">{f.name}</span>
                {f.is_starred && <Star size={13} className="fill-amber-400 text-amber-400" />}
              </a>
              <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
              <div className="flex items-center justify-end gap-1">
                <span className="text-xs text-muted group-hover:hidden">{timeAgo(f.updated_at)}</span>
                <div className="hidden gap-1 group-hover:flex">
                  <IconBtn title="Download" onClick={() => (window.location.href = filesApi.downloadUrl(f.id))}><Download size={15} /></IconBtn>
                  <IconBtn title="Star" onClick={() => star(f)}><Star size={15} className={f.is_starred ? "fill-amber-400 text-amber-400" : ""} /></IconBtn>
                  <IconBtn title="Share" onClick={() => share(f)}><Share2 size={15} /></IconBtn>
                  <IconBtn title="Rename" onClick={() => rename("file", f.id, f.name)}><Pencil size={15} /></IconBtn>
                  <IconBtn title="Trash" onClick={() => trash(f)}><Trash2 size={15} /></IconBtn>
                </div>
              </div>
            </div>
          ))}
        </div>
      </UploadZone>
    </div>
  );
}

function IconBtn({ children, title, onClick }: { children: React.ReactNode; title: string; onClick: () => void }) {
  return (
    <button
      title={title}
      onClick={onClick}
      className="flex h-7 w-7 items-center justify-center rounded-md text-muted transition-colors hover:bg-border hover:text-foreground"
    >
      {children}
    </button>
  );
}
