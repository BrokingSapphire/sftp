"use client";

import { useCallback, useRef, useState } from "react";
import { UploadCloud } from "lucide-react";

interface Props {
  onFiles: (files: File[]) => void;
  // Optional: handle dropped folders (recursively, prompt-free). Given files
  // with their relative paths. When omitted, folder drops fall back to onFiles.
  onEntries?: (entries: { file: File; relPath: string }[]) => void;
  children: React.ReactNode;
}

/** Full-area drag-and-drop wrapper that also exposes a click-to-browse input. */
export function UploadZone({ onFiles, onEntries, children }: Props) {
  const [dragging, setDragging] = useState(false);
  const depth = useRef(0);

  const onDrop = useCallback(
    async (e: React.DragEvent) => {
      e.preventDefault();
      depth.current = 0;
      setDragging(false);

      // If the OS exposes directory entries and the caller wants them, traverse
      // folders recursively — no browser "trust this site" prompt.
      const items = onEntries ? Array.from(e.dataTransfer.items) : [];
      const roots = items
        .map((it) => (it.webkitGetAsEntry ? it.webkitGetAsEntry() : null))
        .filter((x): x is FileSystemEntry => !!x);
      const hasDir = roots.some((r) => r.isDirectory);

      if (onEntries && hasDir) {
        const { readDropEntry } = await import("@/lib/folder-upload");
        const out: { file: File; relPath: string }[] = [];
        for (const r of roots) await readDropEntry(r, "", out);
        if (out.length) onEntries(out);
        return;
      }

      const files = Array.from(e.dataTransfer.files);
      if (files.length) onFiles(files);
    },
    [onFiles, onEntries],
  );

  return (
    <div
      onDragEnter={(e) => {
        e.preventDefault();
        depth.current += 1;
        setDragging(true);
      }}
      onDragOver={(e) => e.preventDefault()}
      onDragLeave={(e) => {
        e.preventDefault();
        depth.current -= 1;
        if (depth.current <= 0) setDragging(false);
      }}
      onDrop={onDrop}
      className="relative"
    >
      {children}
      {dragging && (
        <div className="pointer-events-none absolute inset-0 z-30 flex items-center justify-center rounded-xl border-2 border-dashed border-primary bg-primary/10 backdrop-blur-sm">
          <div className="flex flex-col items-center gap-2 text-primary">
            <UploadCloud size={40} />
            <span className="font-medium">Drop files to upload</span>
          </div>
        </div>
      )}
    </div>
  );
}
