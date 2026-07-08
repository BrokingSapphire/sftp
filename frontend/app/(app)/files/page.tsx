"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  FolderPlus, FolderUp, Upload, Folder, FolderOpen, Download, Star, Share2, Trash2,
  Pencil, ChevronRight, Home, LayoutGrid, List as ListIcon, Eye, Globe, Check, History, Lock, LockOpen, ShieldCheck, FileText, Files,
} from "lucide-react";
import { useAuth } from "@/lib/auth";
import { filesApi, commonApi } from "@/lib/endpoints";
import type { FileItem, FolderItem } from "@/lib/types";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/misc";
import { UploadZone } from "@/components/files/upload-zone";
import { walkDir } from "@/lib/folder-upload";
import { fileIcon } from "@/components/files/icon";
import { FilePreview } from "@/components/files/file-preview";
import { ShareDialog } from "@/components/files/share-dialog";
import { VersionHistory } from "@/components/files/version-history";
import { DocumentEditor, isEditable } from "@/components/files/document-editor";
import { useRouter } from "next/navigation";
import { BRAND } from "@/lib/brand";

const OFFICE_EXT = new Set(["docx", "doc", "odt", "xlsx", "xls", "ods", "pptx", "ppt", "odp"]);
import { useContextMenu, ContextMenu, type MenuItem } from "@/components/files/context-menu";
import { useUploads } from "@/lib/upload-manager";
import { useI18n } from "@/lib/i18n";
import { formatBytes, timeAgo, cn } from "@/lib/utils";
import { StaggerList, StaggerItem, motion } from "@/components/motion";

interface Crumb { id?: string; name: string; }
type View = "list" | "grid";

const IMG = new Set(["png", "jpg", "jpeg", "gif", "svg", "webp", "bmp", "avif"]);

export default function FilesPage() {
  const qc = useQueryClient();
  const [crumbs, setCrumbs] = useState<Crumb[]>([{ name: "Home" }]);
  const current = crumbs[crumbs.length - 1];
  const inputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);
  const [view, setView] = useState<View>("list");
  const [preview, setPreview] = useState<number | null>(null);
  const ctx = useContextMenu();
  const uploads = useUploads();
  const { t: tr } = useI18n();
  const { has } = useAuth();
  const router = useRouter();

  async function toggleHold(f: FileItem) {
    try { await filesApi.setLegalHold(f.id, !f.legal_hold); toast.success(f.legal_hold ? "Legal hold released" : "Legal hold placed"); refresh(); }
    catch { toast.error("Could not update legal hold"); }
  }
  async function setRetention(f: FileItem) {
    const days = prompt("Lock this file from deletion/modification for how many days? (WORM retention)", "365");
    if (!days) return;
    const n = Number(days);
    if (!n || n <= 0) return toast.error("Enter a positive number of days");
    const until = new Date(Date.now() + n * 86400000).toISOString();
    try { await filesApi.setRetention(f.id, until); toast.success(`Retained until ${new Date(until).toLocaleDateString()}`); refresh(); }
    catch { toast.error("Could not set retention (it cannot be shortened)"); }
  }
  const [sharing, setSharing] = useState<{ id: string; name: string; kind: "file" | "folder" } | null>(null);
  const [versionOf, setVersionOf] = useState<{ id: string; name: string; version: number } | null>(null);
  const [editing, setEditing] = useState<{ id: string; name: string } | null>(null);
  const [sel, setSel] = useState<Map<string, "file" | "folder">>(new Map());

  const anySel = sel.size > 0;
  function toggleSel(id: string, kind: "file" | "folder") {
    setSel((m) => {
      const n = new Map(m);
      n.has(id) ? n.delete(id) : n.set(id, kind);
      return n;
    });
  }
  const clearSel = () => setSel(new Map());
  // Reset selection when navigating folders.
  useEffect(() => { clearSel(); }, [current.id]);

  async function bulk(action: "download" | "star" | "common" | "trash") {
    const ids = [...sel.entries()];
    const fileIds = ids.filter(([, k]) => k === "file").map(([id]) => id);
    const folderIds = ids.filter(([, k]) => k === "folder").map(([id]) => id);
    try {
      if (action === "download") {
        fileIds.forEach((id) => window.open(filesApi.downloadUrl(id), "_blank"));
      } else if (action === "star") {
        await Promise.all(fileIds.map((id) => filesApi.starFile(id, true)));
        toast.success(`Starred ${fileIds.length}`);
      } else if (action === "common") {
        await Promise.all(fileIds.map((id) => commonApi.makeCommon(id)));
        toast.success(`Shared ${fileIds.length} to Common`);
      } else if (action === "trash") {
        await Promise.all([
          ...fileIds.map((id) => filesApi.trashFile(id)),
          ...folderIds.map((id) => filesApi.deleteFolder(id).catch(() => {})),
        ]);
        toast.success(`Deleted ${ids.length} item${ids.length === 1 ? "" : "s"}`);
      }
    } catch { toast.error("Some items could not be processed"); }
    clearSel();
    refresh();
  }

  useEffect(() => {
    const v = localStorage.getItem("sftp_view") as View | null;
    if (v) setView(v);
  }, []);
  function setViewPersist(v: View) { setView(v); localStorage.setItem("sftp_view", v); }

  const listing = useQuery({
    queryKey: ["files", current.id ?? "root"],
    queryFn: () => filesApi.list(current.id),
  });
  const files = listing.data?.files ?? [];
  const folders = listing.data?.folders ?? [];

  const refresh = () => qc.invalidateQueries({ queryKey: ["files", current.id ?? "root"] });

  function openFolder(f: FolderItem) { setCrumbs((c) => [...c, { id: f.id, name: f.name }]); }
  function goTo(i: number) { setCrumbs((c) => c.slice(0, i + 1)); }

  function uploadFiles(fs: File[]) {
    // Resumable, pausable, cancelable uploads via the global upload manager.
    uploads.add(fs, current.id);
  }
  async function createFolder() {
    const name = prompt("New folder name");
    if (!name) return;
    try { await filesApi.createFolder(name, current.id); toast.success("Folder created"); refresh(); }
    catch { toast.error("Could not create folder"); }
  }

  // Upload an entire folder tree (preserves structure via webkitRelativePath).
  async function uploadFolder(entries: { file: File; relPath: string }[]) {
    if (entries.length === 0) return;
    const dirCache = new Map<string, string | undefined>([["", current.id]]);

    async function ensureDir(dirPath: string): Promise<string | undefined> {
      if (dirCache.has(dirPath)) return dirCache.get(dirPath);
      const parts = dirPath.split("/");
      const name = parts.pop()!;
      const parentId = await ensureDir(parts.join("/"));
      let id: string | undefined;
      try {
        id = (await filesApi.createFolderReturning(name, parentId)).id;
      } catch {
        // Already exists — resolve its id from the parent listing.
        const listing = await filesApi.list(parentId);
        id = listing.folders.find((x) => x.name === name)?.id;
      }
      dirCache.set(dirPath, id);
      return id;
    }

    const rootName = entries[0].relPath.split("/")[0] || "folder";
    const total = entries.length;
    const t = toast.loading(`Uploading folder "${rootName}"… 0/${total}`, { position: "bottom-right" });
    let done = 0;
    try {
      for (const { file, relPath } of entries) {
        const dir = relPath.split("/").slice(0, -1).join("/");
        const folderId = await ensureDir(dir);
        await filesApi.simpleUpload(file, folderId);
        done++;
        toast.loading(`Uploading folder "${rootName}"… ${done}/${total}`, { id: t, position: "bottom-right" });
      }
      toast.success(`Uploaded "${rootName}" (${done} files)`, { id: t, position: "bottom-right" });
    } catch {
      toast.error(`Folder upload failed after ${done}/${total} files`, { id: t, position: "bottom-right" });
    }
    refresh();
  }

  // Prompt-free folder picker on Chromium (File System Access API); falls back
  // to the directory input on Firefox/Zen (which shows a browser confirm).
  async function pickFolder() {
    const w = window as unknown as { showDirectoryPicker?: () => Promise<FileSystemDirectoryHandle> };
    if (typeof w.showDirectoryPicker === "function") {
      try {
        const dir = await w.showDirectoryPicker();
        const out: { file: File; relPath: string }[] = [];
        await walkDir(dir, dir.name, out);
        uploadFolder(out);
      } catch { /* user cancelled */ }
    } else {
      folderInputRef.current?.click();
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
    try { await filesApi.trashFile(f.id); toast.success("Moved to trash"); refresh(); } catch { toast.error("Delete failed"); }
  }
  async function star(f: FileItem) {
    try { await filesApi.starFile(f.id, !f.is_starred); refresh(); } catch { toast.error("Failed"); }
  }
  function share(f: FileItem) {
    setSharing({ id: f.id, name: f.name, kind: "file" });
  }
  function shareFolder(f: FolderItem) {
    setSharing({ id: f.id, name: f.name, kind: "folder" });
  }
  async function addToCommon(f: FileItem) {
    try { await commonApi.makeCommon(f.id); toast.success(`"${f.name}" shared to Common`); }
    catch { toast.error("Could not share to Common"); }
  }

  function fileMenu(f: FileItem, i: number): MenuItem[] {
    const items: MenuItem[] = [
      { label: "Preview", icon: Eye, onClick: () => setPreview(i) },
      ...(isEditable(f.extension) ? [{ label: "Edit", icon: Pencil, onClick: () => setEditing({ id: f.id, name: f.name }) }] : []),
      ...(BRAND.editor?.enabled && OFFICE_EXT.has((f.extension || "").toLowerCase())
        ? [{ label: "Edit in Office", icon: FileText, onClick: () => router.push(`/editor/${f.id}`) }] : []),
      { label: "Download", icon: Download, onClick: () => (window.location.href = filesApi.downloadUrl(f.id)) },
      { label: "Get share link", icon: Share2, onClick: () => share(f) },
      { label: "Add to Common", icon: Globe, onClick: () => addToCommon(f) },
      { separator: true, label: "" },
      { label: f.is_starred ? "Remove star" : "Add star", icon: Star, onClick: () => star(f) },
      { label: "Make a copy", icon: Files, onClick: () => copyFile(f) },
      { label: "Rename", icon: Pencil, onClick: () => rename("file", f.id, f.name) },
      { label: "Version history", icon: History, onClick: () => setVersionOf({ id: f.id, name: f.name, version: f.version_no }) },
    ];
    if (has("storage.manage")) {
      items.push({ separator: true, label: "" });
      items.push({ label: f.legal_hold ? "Release legal hold" : "Place legal hold", icon: f.legal_hold ? LockOpen : Lock, onClick: () => toggleHold(f) });
      items.push({ label: "Set retention (WORM)…", icon: ShieldCheck, onClick: () => setRetention(f) });
    }
    items.push({ separator: true, label: "" });
    items.push({ label: "Move to trash", icon: Trash2, danger: true, onClick: () => trash(f) });
    return items;
  }
  async function setColor(f: FolderItem, color: string) {
    try { await filesApi.setFolderColor(f.id, color); refresh(); }
    catch { toast.error("Could not set colour"); }
  }
  function folderMenu(f: FolderItem): MenuItem[] {
    return [
      { label: "Open", icon: FolderOpen, onClick: () => openFolder(f) },
      { label: "Download (zip)", icon: Download, onClick: () => { window.location.href = filesApi.folderDownloadUrl(f.id); } },
      { label: "Get share link", icon: Share2, onClick: () => shareFolder(f) },
      { label: "Rename", icon: Pencil, onClick: () => rename("folder", f.id, f.name) },
      { separator: true, label: "" },
      { label: "colour", node: <ColorSwatches current={f.color} onPick={(c) => setColor(f, c)} /> },
      { separator: true, label: "" },
      { label: "Delete", icon: Trash2, danger: true, onClick: () => removeFolder(f) },
    ];
  }
  function copyFile(f: FileItem) {
    filesApi.copyFile(f.id).then(() => { toast.success(`Copied "${f.name}"`); refresh(); }).catch(() => toast.error("Could not copy"));
  }
  function removeFolder(f: FolderItem) {
    if (!confirm(`Delete folder "${f.name}"? Everything inside it will be moved to Trash.`)) return;
    filesApi.deleteFolder(f.id).then(() => { toast.success("Folder deleted"); refresh(); }).catch(() => toast.error("Could not delete folder"));
  }

  const empty = !listing.isLoading && folders.length === 0 && files.length === 0;

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
                className={cn("flex items-center gap-1 rounded px-1.5 py-0.5 hover:bg-surface-2", i === crumbs.length - 1 ? "font-semibold" : "text-muted")}
              >
                {i === 0 && <Home size={14} />}
                {i === 0 ? tr("common.home") : c.name}
              </button>
            </span>
          ))}
        </nav>
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center rounded-md border border-border p-0.5">
            <ViewBtn active={view === "list"} onClick={() => setViewPersist("list")}><ListIcon size={16} /></ViewBtn>
            <ViewBtn active={view === "grid"} onClick={() => setViewPersist("grid")}><LayoutGrid size={16} /></ViewBtn>
          </div>
          <Button variant="outline" size="sm" onClick={createFolder}><FolderPlus size={16} /> {tr("action.newFolder")}</Button>
          <Button variant="outline" size="sm" onClick={pickFolder}><FolderUp size={16} /> {tr("action.uploadFolder")}</Button>
          <Button size="sm" onClick={() => inputRef.current?.click()}><Upload size={16} /> {tr("action.upload")}</Button>
          <input ref={inputRef} type="file" multiple hidden onChange={(e) => {
            const fs = Array.from(e.target.files ?? []);
            if (fs.length) uploadFiles(fs);
            e.target.value = "";
          }} />
          <input
            ref={folderInputRef}
            type="file"
            hidden
            // @ts-expect-error non-standard directory-picker attributes
            webkitdirectory=""
            directory=""
            multiple
            onChange={(e) => {
              const fs = Array.from(e.target.files ?? []);
              if (fs.length) uploadFolder(fs.map((f) => ({ file: f, relPath: (f as unknown as { webkitRelativePath?: string }).webkitRelativePath || f.name })));
              e.target.value = "";
            }}
          />
        </div>
      </div>

      {/* Bulk-selection action bar */}
      {anySel && (
        <motion.div
          initial={{ opacity: 0, y: -6 }} animate={{ opacity: 1, y: 0 }}
          className="flex flex-wrap items-center gap-2 rounded-xl border border-primary/30 bg-primary/5 px-3 py-2"
        >
          <span className="text-sm font-medium">{sel.size} selected</span>
          <div className="mx-1 h-5 w-px bg-border" />
          <Button variant="ghost" size="sm" onClick={() => bulk("download")}><Download size={15} /> Download</Button>
          <Button variant="ghost" size="sm" onClick={() => bulk("star")}><Star size={15} /> Star</Button>
          <Button variant="ghost" size="sm" onClick={() => bulk("common")}><Globe size={15} /> To Common</Button>
          <Button variant="ghost" size="sm" onClick={() => bulk("trash")}><Trash2 size={15} /> Delete</Button>
          <button onClick={clearSel} className="ml-auto text-xs text-muted hover:text-foreground">Clear</button>
        </motion.div>
      )}

      {/* Listing */}
      <UploadZone onFiles={uploadFiles} onEntries={uploadFolder}>
        {listing.isLoading ? (
          <div className="rounded-xl border border-border bg-surface p-4">
            {[...Array(6)].map((_, i) => <Skeleton key={i} className="mb-2 h-9 w-full" />)}
          </div>
        ) : empty ? (
          <div className="flex min-h-[24rem] flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-surface text-muted">
            <Folder size={40} />
            <p className="text-sm">This folder is empty. Drag files here or use Upload.</p>
          </div>
        ) : view === "list" ? (
          <div className="min-h-[24rem] overflow-x-auto rounded-xl border border-border bg-surface">
            <div className="grid min-w-[34rem] grid-cols-[1fr_auto_8rem] gap-4 border-b border-border px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
              <span>Name</span><span>Size</span><span className="text-right">Modified</span>
            </div>
            <StaggerList>
              {folders.map((f) => (
                <StaggerItem key={f.id} onContextMenu={(e) => ctx.open(e, folderMenu(f))} className={cn("group grid min-w-[34rem] grid-cols-[1fr_auto_8rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2", sel.has(f.id) && "bg-primary/5")}>
                  <div className="flex min-w-0 items-center gap-3">
                    <SelectBox selected={sel.has(f.id)} anySel={anySel} onToggle={() => toggleSel(f.id, "folder")}>
                      <Folder size={18} style={f.color ? { color: f.color } : undefined} className={f.color ? "" : "text-primary"} />
                    </SelectBox>
                    <button onClick={() => openFolder(f)} className="min-w-0 flex-1 truncate text-left text-sm font-medium">{f.name}</button>
                  </div>
                  <span className="text-xs text-muted">—</span>
                  <div className="flex items-center justify-end gap-1">
                    <span className="text-xs text-muted group-hover:hidden">{timeAgo(f.updated_at)}</span>
                    <div className="hidden gap-1 group-hover:flex">
                      <IconBtn title="Rename" onClick={() => rename("folder", f.id, f.name)}><Pencil size={15} /></IconBtn>
                      <IconBtn title="Delete" onClick={() => removeFolder(f)}><Trash2 size={15} /></IconBtn>
                    </div>
                  </div>
                </StaggerItem>
              ))}
              {files.map((f, i) => (
                <StaggerItem key={f.id} onContextMenu={(e) => ctx.open(e, fileMenu(f, i))} className={cn("group grid min-w-[34rem] grid-cols-[1fr_auto_8rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2", sel.has(f.id) && "bg-primary/5")}>
                  <div className="flex min-w-0 items-center gap-3">
                    <SelectBox selected={sel.has(f.id)} anySel={anySel} onToggle={() => toggleSel(f.id, "file")}>
                      {fileIcon(f.extension, 18)}
                    </SelectBox>
                    <button onClick={() => setPreview(i)} className="min-w-0 flex-1 truncate text-left text-sm font-medium">{f.name}</button>
                    {f.is_starred && <Star size={13} className="shrink-0 fill-amber-400 text-amber-400" />}
                    {f.legal_hold && <Lock size={13} className="shrink-0 text-danger" aria-label="Legal hold" />}
                    {!f.legal_hold && f.retain_until && <ShieldCheck size={13} className="shrink-0 text-warning" aria-label="Retention lock" />}
                    <SensitivityTag level={f.sensitivity} types={f.pii_types} />
                  </div>
                  <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
                  <div className="flex items-center justify-end gap-1">
                    <span className="text-xs text-muted group-hover:hidden">{timeAgo(f.updated_at)}</span>
                    <div className="hidden gap-1 group-hover:flex">
                      <IconBtn title="Preview" onClick={() => setPreview(i)}><Eye size={15} /></IconBtn>
                      <a href={filesApi.downloadUrl(f.id)}><IconBtn title="Download" onClick={() => {}}><Download size={15} /></IconBtn></a>
                      <IconBtn title="Star" onClick={() => star(f)}><Star size={15} className={f.is_starred ? "fill-amber-400 text-amber-400" : ""} /></IconBtn>
                      <IconBtn title="Share" onClick={() => share(f)}><Share2 size={15} /></IconBtn>
                      <IconBtn title="Rename" onClick={() => rename("file", f.id, f.name)}><Pencil size={15} /></IconBtn>
                      <IconBtn title="Trash" onClick={() => trash(f)}><Trash2 size={15} /></IconBtn>
                    </div>
                  </div>
                </StaggerItem>
              ))}
            </StaggerList>
          </div>
        ) : (
          <StaggerList className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {folders.map((f) => (
              <StaggerItem key={f.id}>
                <motion.button
                  whileHover={{ y: -3 }} transition={{ type: "spring", stiffness: 380, damping: 26 }}
                  onClick={() => openFolder(f)}
                  onContextMenu={(e) => ctx.open(e, folderMenu(f))}
                  className="group flex w-full items-center gap-2 rounded-xl border border-border bg-surface p-3 text-left transition-shadow hover:shadow-md"
                >
                  <Folder size={20} style={f.color ? { color: f.color } : undefined} className={f.color ? "shrink-0" : "shrink-0 text-primary"} />
                  <span className="truncate text-sm font-medium">{f.name}</span>
                </motion.button>
              </StaggerItem>
            ))}
            {files.map((f, i) => (
              <StaggerItem key={f.id}>
                <motion.button
                  whileHover={{ y: -3 }} transition={{ type: "spring", stiffness: 380, damping: 26 }}
                  onClick={() => setPreview(i)}
                  onContextMenu={(e) => ctx.open(e, fileMenu(f, i))}
                  className="group flex w-full flex-col overflow-hidden rounded-xl border border-border bg-surface text-left transition-shadow hover:shadow-md"
                >
                  <div className="flex h-28 items-center justify-center overflow-hidden border-b border-border bg-surface-2">
                    {IMG.has(f.extension?.toLowerCase()) ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={filesApi.previewUrl(f.id)} alt={f.name} className="h-full w-full object-cover transition-transform group-hover:scale-105" />
                    ) : (
                      <span className="scale-[1.6]">{fileIcon(f.extension, 22)}</span>
                    )}
                  </div>
                  <div className="flex items-center gap-1.5 p-2.5">
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">{f.name}</span>
                    {f.is_starred && <Star size={12} className="fill-amber-400 text-amber-400" />}
                  </div>
                </motion.button>
              </StaggerItem>
            ))}
          </StaggerList>
        )}
      </UploadZone>

      {preview !== null && files[preview] && (
        <FilePreview
          files={files}
          index={preview}
          onChangeIndex={setPreview}
          onClose={() => setPreview(null)}
          onChanged={refresh}
        />
      )}

      <ContextMenu menu={ctx.menu} onClose={ctx.close} />
      {sharing && <ShareDialog fileId={sharing.id} fileName={sharing.name} kind={sharing.kind} onClose={() => setSharing(null)} />}
      {versionOf && (
        <VersionHistory
          fileId={versionOf.id} fileName={versionOf.name} currentVersion={versionOf.version}
          onClose={() => setVersionOf(null)} onRestored={refresh}
        />
      )}
      {editing && (
        <DocumentEditor fileId={editing.id} fileName={editing.name} onClose={() => setEditing(null)} onSaved={refresh} />
      )}
    </div>
  );
}

function SensitivityTag({ level, types }: { level?: string; types?: string[] }) {
  if (!level || level === "public" || level === "internal") return null;
  const style = level === "restricted" ? "bg-danger/10 text-danger" : "bg-warning/10 text-warning";
  return (
    <span
      title={types?.length ? `Detected: ${types.join(", ")}` : level}
      className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium capitalize ${style}`}
    >
      {level}
    </span>
  );
}

function IconBtn({ children, title, onClick }: { children: React.ReactNode; title: string; onClick: () => void }) {
  return (
    <button title={title} onClick={onClick} className="flex h-7 w-7 items-center justify-center rounded-md text-muted transition-colors hover:bg-border hover:text-foreground">
      {children}
    </button>
  );
}

/** File-type icon that flips to a selectable checkbox on hover / when selecting. */
function SelectBox({ selected, anySel, onToggle, children }: { selected: boolean; anySel: boolean; onToggle: () => void; children: React.ReactNode }) {
  const showCheck = selected || anySel;
  return (
    <button
      onClick={(e) => { e.stopPropagation(); onToggle(); }}
      className="relative flex h-[18px] w-[18px] shrink-0 items-center justify-center"
      title={selected ? "Deselect" : "Select"}
    >
      <span className={cn("transition-opacity", selected ? "opacity-0" : "group-hover:opacity-0", showCheck && "opacity-0")}>{children}</span>
      <span
        className={cn(
          "absolute inset-0 flex items-center justify-center rounded border transition-opacity",
          selected ? "border-primary bg-primary opacity-100" : showCheck ? "border-border opacity-100" : "border-border opacity-0 group-hover:opacity-100",
        )}
      >
        {selected && <Check size={12} className="text-primary-foreground" />}
      </span>
    </button>
  );
}

function ViewBtn({ children, active, onClick }: { children: React.ReactNode; active: boolean; onClick: () => void }) {
  return (
    <button onClick={onClick} className={cn("flex h-7 w-7 items-center justify-center rounded transition-colors", active ? "bg-primary/10 text-primary" : "text-muted hover:text-foreground")}>
      {children}
    </button>
  );
}

const FOLDER_PALETTE = [
  "", "#064D51", "#2563eb", "#16a34a", "#d97706", "#dc2626", "#7c3aed", "#db2777", "#6b7280",
];

function ColorSwatches({ current, onPick }: { current?: string; onPick: (c: string) => void }) {
  return (
    <div>
      <p className="mb-1.5 font-mono text-[10px] uppercase tracking-wider text-muted">Folder colour</p>
      <div className="flex flex-wrap gap-1.5">
        {FOLDER_PALETTE.map((c) => (
          <button
            key={c || "none"}
            title={c || "Default"}
            onClick={() => onPick(c)}
            className={cn(
              "h-5 w-5 rounded-full border transition-transform hover:scale-110",
              (current || "") === c ? "ring-2 ring-ring ring-offset-1 ring-offset-surface" : "",
            )}
            style={{ backgroundColor: c || "var(--surface-2)", borderColor: c ? "transparent" : "var(--border)" }}
          >
            {!c && <span className="text-[9px] text-muted">×</span>}
          </button>
        ))}
      </div>
    </div>
  );
}
