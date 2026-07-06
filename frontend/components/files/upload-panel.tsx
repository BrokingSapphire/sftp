"use client";

import { useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Pause, Play, X, ChevronDown, CheckCircle2, AlertCircle, UploadCloud } from "lucide-react";
import { useUploads } from "@/lib/upload-manager";
import { formatBytes } from "@/lib/utils";

export function UploadPanel() {
  const { tasks, pause, resume, cancel, clearDone } = useUploads();
  const [collapsed, setCollapsed] = useState(false);
  if (tasks.length === 0) return null;

  const active = tasks.filter((t) => t.status === "uploading" || t.status === "paused").length;
  const doneCount = tasks.filter((t) => t.status === "done").length;

  return (
    <motion.div
      initial={{ opacity: 0, y: 24 }}
      animate={{ opacity: 1, y: 0 }}
      className="fixed bottom-4 right-4 z-40 w-80 overflow-hidden rounded-xl border border-border bg-surface shadow-2xl"
    >
      <div className="flex items-center gap-2 border-b border-border bg-surface-2 px-3 py-2">
        <UploadCloud size={15} className="text-primary" />
        <span className="flex-1 text-sm font-medium">
          {active > 0 ? `Uploading ${active} item${active === 1 ? "" : "s"}` : `${doneCount} upload${doneCount === 1 ? "" : "s"} complete`}
        </span>
        {doneCount > 0 && active === 0 && (
          <button onClick={clearDone} className="text-xs text-muted hover:text-foreground">Clear</button>
        )}
        <button onClick={() => setCollapsed((c) => !c)} className="text-muted hover:text-foreground">
          <motion.span animate={{ rotate: collapsed ? 180 : 0 }}><ChevronDown size={16} /></motion.span>
        </button>
      </div>

      <AnimatePresence>
        {!collapsed && (
          <motion.div
            initial={{ height: 0 }} animate={{ height: "auto" }} exit={{ height: 0 }}
            className="max-h-72 overflow-y-auto"
          >
            {tasks.map((t) => (
              <div key={t.id} className="flex items-center gap-2.5 border-b border-border/50 px-3 py-2.5">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">{t.name}</span>
                    <span className="shrink-0 text-[11px] text-muted">{formatBytes(t.size)}</span>
                  </div>
                  <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-surface-2">
                    <div
                      className={`h-full rounded-full transition-all ${t.status === "error" ? "bg-danger" : t.status === "paused" ? "bg-warning" : "bg-primary"}`}
                      style={{ width: `${t.progress}%` }}
                    />
                  </div>
                </div>

                {t.status === "done" ? (
                  <CheckCircle2 size={17} className="text-success" />
                ) : t.status === "error" ? (
                  <AlertCircle size={17} className="text-danger" />
                ) : (
                  <div className="flex items-center gap-0.5">
                    {t.status === "paused" ? (
                      <button title="Resume" onClick={() => resume(t.id)} className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-surface-2 hover:text-foreground"><Play size={14} /></button>
                    ) : (
                      <button title="Pause" onClick={() => pause(t.id)} className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-surface-2 hover:text-foreground"><Pause size={14} /></button>
                    )}
                    <button title="Cancel" onClick={() => cancel(t.id)} className="flex h-7 w-7 items-center justify-center rounded-md text-danger hover:bg-surface-2"><X size={14} /></button>
                  </div>
                )}
              </div>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
