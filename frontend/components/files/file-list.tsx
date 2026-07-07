"use client";

import { useState } from "react";
import { Download, RotateCcw, Trash2, Star, Eye, FileQuestion } from "lucide-react";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import type { FileItem } from "@/lib/types";
import { filesApi } from "@/lib/endpoints";
import { fileIcon } from "./icon";
import { FilePreview } from "./file-preview";
import { Skeleton } from "@/components/ui/misc";
import { EmptyState } from "@/components/ui/empty-state";
import { useI18n } from "@/lib/i18n";
import { formatBytes, timeAgo } from "@/lib/utils";
import { StaggerList, StaggerItem } from "@/components/motion";

interface Props {
  files?: FileItem[];
  loading?: boolean;
  emptyLabel: string;
  emptyIcon?: React.ElementType;
  emptySubtitle?: string;
  queryKey: string;
  mode?: "default" | "trash";
}

export function FileList({ files, loading, emptyLabel, emptyIcon, emptySubtitle, queryKey, mode = "default" }: Props) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const [preview, setPreview] = useState<number | null>(null);
  const refresh = () => qc.invalidateQueries({ queryKey: [queryKey] });

  async function act(fn: () => Promise<unknown>, ok: string) {
    try { await fn(); toast.success(ok); refresh(); }
    catch { toast.error("Action failed"); }
  }

  // Star toggle with an optimistic cache update so the UI changes immediately
  // (and unstarred items leave the Starred list right away).
  async function toggleStar(f: FileItem) {
    const next = !f.is_starred;
    qc.setQueryData<FileItem[]>([queryKey], (old) =>
      (old ?? []).map((x) => (x.id === f.id ? { ...x, is_starred: next } : x)));
    try { await filesApi.starFile(f.id, next); refresh(); }
    catch { toast.error("Action failed"); refresh(); }
  }

  if (!loading && !files?.length) {
    return <EmptyState icon={emptyIcon ?? FileQuestion} title={emptyLabel} subtitle={emptySubtitle} />;
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-border bg-surface">
      <div className="min-w-[32rem]">
      <div className="grid grid-cols-[1fr_auto_8rem] gap-4 border-b border-border px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
        <span>{t("col.name")}</span><span>{t("col.size")}</span><span className="text-right">{t("col.modified")}</span>
      </div>
      {loading && [...Array(5)].map((_, i) => <div key={i} className="px-4 py-2.5"><Skeleton className="h-6 w-full" /></div>)}
      <StaggerList>
      {files?.map((f, i) => (
        <StaggerItem key={f.id} className="group grid grid-cols-[1fr_auto_8rem] items-center gap-4 border-b border-border/50 px-4 py-2.5 transition-colors hover:bg-surface-2">
          {mode === "trash" ? (
            <div className="flex min-w-0 items-center gap-3">
              {fileIcon(f.extension, 18)}
              <span className="truncate text-sm font-medium">{f.name}</span>
            </div>
          ) : (
            <button onClick={() => setPreview(i)} className="flex min-w-0 items-center gap-3 text-left">
              {fileIcon(f.extension, 18)}
              <span className="truncate text-sm font-medium">{f.name}</span>
              {f.is_starred && <Star size={13} className="fill-amber-400 text-amber-400" />}
            </button>
          )}
          <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
          <div className="flex items-center justify-end gap-1">
            <span className="text-xs text-muted group-hover:hidden">{timeAgo(mode === "trash" ? f.deleted_at : f.updated_at)}</span>
            <div className="hidden gap-1 group-hover:flex">
              {mode === "trash" ? (
                <>
                  <IconBtn title="Restore" onClick={() => act(() => filesApi.restoreFile(f.id), "Restored")}><RotateCcw size={15} /></IconBtn>
                  <IconBtn title="Delete forever" onClick={() => act(() => filesApi.deleteFile(f.id), "Deleted")}><Trash2 size={15} /></IconBtn>
                </>
              ) : (
                <>
                  <IconBtn title="Preview" onClick={() => setPreview(i)}><Eye size={15} /></IconBtn>
                  <a href={filesApi.downloadUrl(f.id)}><IconBtn title="Download" onClick={() => {}}><Download size={15} /></IconBtn></a>
                  <IconBtn title="Star" onClick={() => toggleStar(f)}>
                    <Star size={15} className={f.is_starred ? "fill-amber-400 text-amber-400" : ""} />
                  </IconBtn>
                  <IconBtn title="Trash" onClick={() => act(() => filesApi.trashFile(f.id), "Moved to trash")}><Trash2 size={15} /></IconBtn>
                </>
              )}
            </div>
          </div>
        </StaggerItem>
      ))}
      </StaggerList>
      </div>
      {preview !== null && files?.[preview] && mode !== "trash" && (
        <FilePreview files={files} index={preview} onChangeIndex={setPreview} onClose={() => setPreview(null)} onChanged={refresh} />
      )}
    </div>
  );
}

function IconBtn({ children, title, onClick }: { children: React.ReactNode; title: string; onClick: () => void }) {
  return (
    <button title={title} onClick={onClick}
      className="flex h-7 w-7 items-center justify-center rounded-md text-muted transition-colors hover:bg-border hover:text-foreground">
      {children}
    </button>
  );
}

export function PageHeader({ title, subtitle, icon: Icon }: { title: string; subtitle?: string; icon?: React.ElementType }) {
  const { tx } = useI18n();
  return (
    <div className="flex items-start gap-3">
      {Icon && (
        <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Icon size={18} />
        </span>
      )}
      <div className="min-w-0">
        <h1 className="text-xl font-semibold tracking-tight sm:text-2xl">{tx(title)}</h1>
        {subtitle && <p className="mt-0.5 max-w-2xl text-sm text-muted">{tx(subtitle)}</p>}
      </div>
    </div>
  );
}
