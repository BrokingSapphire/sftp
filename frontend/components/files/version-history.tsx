"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { motion } from "motion/react";
import { History, Download, RotateCcw, X, Check } from "lucide-react";
import { filesApi, type FileVersion } from "@/lib/endpoints";
import { Button } from "@/components/ui/button";
import { useDialogs } from "@/components/ui/dialogs";
import { Skeleton } from "@/components/ui/misc";
import { formatBytes, timeAgo } from "@/lib/utils";

/** Version history for a file: current + archived versions, restore + download. */
export function VersionHistory({
  fileId, fileName, currentVersion, onClose, onRestored,
}: {
  fileId: string; fileName: string; currentVersion: number; onClose: () => void; onRestored?: () => void;
}) {
  const qc = useQueryClient();
  const { confirm } = useDialogs();
  const q = useQuery({ queryKey: ["versions", fileId], queryFn: () => filesApi.versions(fileId) });
  const [restoring, setRestoring] = useState<number | null>(null);
  const past = q.data ?? [];

  async function restore(v: FileVersion) {
    if (!(await confirm({ title: "Restore version", message: `Restore version ${v.version_no}? The current content is saved as a new version first.`, confirmLabel: "Restore" }))) return;
    setRestoring(v.version_no);
    try {
      await filesApi.restoreVersion(fileId, v.version_no);
      toast.success(`Restored version ${v.version_no}`);
      qc.invalidateQueries({ queryKey: ["versions", fileId] });
      qc.invalidateQueries({ queryKey: ["files"] });
      onRestored?.();
    } catch { toast.error("Could not restore version"); }
    finally { setRestoring(null); }
  }

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <motion.div
        initial={{ opacity: 0, scale: 0.97, y: 8 }} animate={{ opacity: 1, scale: 1, y: 0 }}
        className="max-h-[80vh] w-full max-w-md overflow-hidden rounded-2xl border border-border bg-surface shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-border px-5 py-3">
          <div className="flex min-w-0 items-center gap-2">
            <History size={17} className="text-primary" />
            <div className="min-w-0">
              <h3 className="text-sm font-semibold">Version history</h3>
              <p className="truncate text-xs text-muted">{fileName}</p>
            </div>
          </div>
          <button onClick={onClose} className="text-muted hover:text-foreground"><X size={18} /></button>
        </div>

        <div className="max-h-[60vh] overflow-y-auto p-3">
          {/* Current version */}
          <div className="mb-1 flex items-center gap-3 rounded-lg border border-primary/30 bg-primary/5 px-3 py-2.5">
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/15 text-xs font-semibold text-primary">v{currentVersion}</span>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">Current version</p>
              <p className="text-xs text-muted">Latest content</p>
            </div>
            <Check size={16} className="text-success" />
          </div>

          {q.isLoading && <Skeleton className="mt-2 h-24 w-full" />}
          {!q.isLoading && past.length === 0 && (
            <p className="px-3 py-6 text-center text-xs text-muted">No previous versions yet. Re-upload a file with the same name to create one.</p>
          )}

          {past.map((v) => (
            <div key={v.version_no} className="flex items-center gap-3 rounded-lg px-3 py-2.5 hover:bg-surface-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-full bg-surface-2 text-xs font-semibold text-muted">v{v.version_no}</span>
              <div className="min-w-0 flex-1">
                <p className="text-sm">{timeAgo(v.created_at)}{v.author ? ` · ${v.author}` : ""}</p>
                <p className="text-xs text-muted">{formatBytes(v.size_bytes)}</p>
              </div>
              <a href={filesApi.versionDownloadUrl(fileId, v.version_no)}>
                <button title="Download this version" className="flex h-8 w-8 items-center justify-center rounded-md text-muted hover:bg-border hover:text-foreground"><Download size={15} /></button>
              </a>
              <Button variant="outline" size="sm" disabled={restoring === v.version_no} onClick={() => restore(v)}>
                <RotateCcw size={13} /> Restore
              </Button>
            </div>
          ))}
        </div>
      </motion.div>
    </div>
  );
}
