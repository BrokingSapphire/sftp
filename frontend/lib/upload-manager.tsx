"use client";

import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { filesApi } from "./endpoints";

export type UploadStatus = "uploading" | "paused" | "done" | "error" | "canceled";

export interface UploadTask {
  id: string;
  name: string;
  size: number;
  progress: number; // 0-100
  status: UploadStatus;
}

interface Control {
  file: File;
  folderId?: string;
  uploadId?: string;
  totalChunks: number;
  chunkSize: number;
  received: Set<number>;
  paused: boolean;
  canceled: boolean;
  controller?: AbortController;
}

interface Ctx {
  tasks: UploadTask[];
  add: (files: File[], folderId?: string) => void;
  pause: (id: string) => void;
  resume: (id: string) => void;
  cancel: (id: string) => void;
  clearDone: () => void;
}

const UploadContext = createContext<Ctx | null>(null);
const DEFAULT_CHUNK = 8 * 1024 * 1024; // 8 MiB
let seq = 0;

export function UploadProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient();
  const [tasks, setTasks] = useState<UploadTask[]>([]);
  const controls = useRef<Map<string, Control>>(new Map());

  const patch = useCallback((id: string, p: Partial<UploadTask>) => {
    setTasks((ts) => ts.map((t) => (t.id === id ? { ...t, ...p } : t)));
  }, []);

  const run = useCallback(
    async (id: string) => {
      const c = controls.current.get(id);
      if (!c) return;
      patch(id, { status: "uploading" });

      try {
        if (!c.uploadId) {
          const init = await filesApi.initUpload({
            filename: c.file.name, total_size: c.file.size || 1,
            chunk_size: DEFAULT_CHUNK, folder_id: c.folderId,
          });
          c.uploadId = init.upload_id;
          c.totalChunks = init.total_chunks;
          c.chunkSize = init.chunk_size;
          c.received = new Set(init.received_chunks);
        }

        for (let i = 0; i < c.totalChunks; i++) {
          if (c.canceled) return;
          if (c.paused) { patch(id, { status: "paused" }); return; }
          if (c.received.has(i)) continue;

          const start = i * c.chunkSize;
          const blob = c.file.slice(start, Math.min(start + c.chunkSize, c.file.size));
          c.controller = new AbortController();
          try {
            await filesApi.putChunk(c.uploadId!, i, blob, c.controller.signal);
          } catch (err) {
            if (c.paused || c.canceled) { if (c.paused) patch(id, { status: "paused" }); return; }
            throw err;
          }
          c.received.add(i);
          patch(id, { progress: Math.round((c.received.size / c.totalChunks) * 100) });
        }

        await filesApi.completeUpload(c.uploadId!);
        patch(id, { status: "done", progress: 100 });
        qc.invalidateQueries({ queryKey: ["files"] });
        qc.invalidateQueries({ queryKey: ["recent"] });
      } catch {
        patch(id, { status: "error" });
        toast.error(`Upload failed: ${c.file.name}`, { position: "bottom-right" });
      }
    },
    [patch, qc],
  );

  const add = useCallback(
    (files: File[], folderId?: string) => {
      const next: UploadTask[] = [];
      for (const file of files) {
        const id = `u${++seq}`;
        controls.current.set(id, {
          file, folderId, totalChunks: 0, chunkSize: DEFAULT_CHUNK,
          received: new Set(), paused: false, canceled: false,
        });
        next.push({ id, name: file.name, size: file.size, progress: 0, status: "uploading" });
      }
      setTasks((ts) => [...next, ...ts]);
      next.forEach((t) => run(t.id));
    },
    [run],
  );

  const pause = useCallback((id: string) => {
    const c = controls.current.get(id);
    if (!c) return;
    c.paused = true;
    c.controller?.abort();
    patch(id, { status: "paused" });
  }, [patch]);

  const resume = useCallback((id: string) => {
    const c = controls.current.get(id);
    if (!c) return;
    c.paused = false;
    run(id);
  }, [run]);

  const cancel = useCallback(async (id: string) => {
    const c = controls.current.get(id);
    if (!c) return;
    c.canceled = true;
    c.controller?.abort();
    if (c.uploadId) filesApi.abortUpload(c.uploadId).catch(() => {});
    controls.current.delete(id);
    setTasks((ts) => ts.filter((t) => t.id !== id));
  }, []);

  const clearDone = useCallback(() => {
    setTasks((ts) => ts.filter((t) => t.status !== "done"));
  }, []);

  return (
    <UploadContext.Provider value={{ tasks, add, pause, resume, cancel, clearDone }}>
      {children}
    </UploadContext.Provider>
  );
}

export function useUploads() {
  const ctx = useContext(UploadContext);
  if (!ctx) throw new Error("useUploads must be used within UploadProvider");
  return ctx;
}
