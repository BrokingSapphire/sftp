"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Search, FileText, Download, Eye } from "lucide-react";
import { filesApi } from "@/lib/endpoints";
import type { FileItem } from "@/lib/types";
import { PageHeader } from "@/components/files/file-list";
import { Skeleton } from "@/components/ui/misc";
import { fileIcon } from "@/components/files/icon";
import { FilePreview } from "@/components/files/file-preview";
import { StaggerList, StaggerItem } from "@/components/motion";
import { formatBytes } from "@/lib/utils";

/** Renders a ts_headline snippet, turning <<term>> markers into highlights. */
function Snippet({ text }: { text: string }) {
  const parts = text.split(/(<<[^>]*>>)/g);
  return (
    <p className="mt-0.5 truncate text-xs text-muted">
      {parts.map((p, i) =>
        p.startsWith("<<") && p.endsWith(">>") ? (
          <mark key={i} className="rounded bg-primary/20 px-0.5 text-primary">{p.slice(2, -2)}</mark>
        ) : (
          <span key={i}>{p}</span>
        ),
      )}
    </p>
  );
}

export default function SearchPage() {
  const q = (useSearchParams().get("q") ?? "").trim();

  const names = useQuery({
    queryKey: ["search-name", q],
    queryFn: () => filesApi.search(q),
    enabled: q.length > 0,
  });
  const contents = useQuery({
    queryKey: ["search-content", q],
    queryFn: () => filesApi.searchContent(q),
    enabled: q.length > 0,
  });

  const [preview, setPreview] = useState<FileItem[] | null>(null);
  const [previewIdx, setPreviewIdx] = useState(0);

  const nameHits = names.data ?? [];
  const contentHits = (contents.data ?? []).filter(
    (c) => !nameHits.some((n) => n.id === c.id), // avoid duplicates already shown by name
  );
  const loading = names.isLoading || contents.isLoading;
  const empty = !loading && nameHits.length === 0 && contentHits.length === 0;

  function openPreview(list: { id: string; name: string; extension: string; mime_type: string; size_bytes: number }[], i: number) {
    setPreview(list as unknown as FileItem[]);
    setPreviewIdx(i);
  }

  return (
    <div className="mx-auto max-w-5xl space-y-5">
      <PageHeader title={q ? `Results for “${q}”` : "Search"} subtitle="Matches file names and text inside documents" />

      {!q && (
        <div className="flex min-h-[16rem] flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-surface text-muted">
          <Search size={40} />
          <p className="text-sm">Type in the search bar to find files by name or content.</p>
        </div>
      )}

      {loading && <Skeleton className="h-40 w-full" />}

      {empty && (
        <div className="flex min-h-[16rem] flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-surface text-muted">
          <Search size={40} />
          <p className="text-sm">No files match “{q}”.</p>
        </div>
      )}

      {nameHits.length > 0 && (
        <section>
          <h2 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted">Name matches · {nameHits.length}</h2>
          <div className="rounded-xl border border-border bg-surface">
            <StaggerList>
              {nameHits.map((f, i) => (
                <StaggerItem key={f.id} className="group flex items-center gap-3 border-b border-border/50 px-4 py-2.5 last:border-0 hover:bg-surface-2">
                  <button onClick={() => openPreview(nameHits, i)} className="flex min-w-0 flex-1 items-center gap-3 text-left">
                    {fileIcon(f.extension, 18)}
                    <span className="truncate text-sm font-medium">{f.name}</span>
                  </button>
                  <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
                  <RowActions id={f.id} onPreview={() => openPreview(nameHits, i)} />
                </StaggerItem>
              ))}
            </StaggerList>
          </div>
        </section>
      )}

      {contentHits.length > 0 && (
        <section>
          <h2 className="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-muted">
            <FileText size={13} /> In file contents · {contentHits.length}
          </h2>
          <div className="rounded-xl border border-border bg-surface">
            <StaggerList>
              {contentHits.map((f, i) => (
                <StaggerItem key={f.id} className="group flex items-center gap-3 border-b border-border/50 px-4 py-2.5 last:border-0 hover:bg-surface-2">
                  <button onClick={() => openPreview(contentHits, i)} className="flex min-w-0 flex-1 items-start gap-3 text-left">
                    <span className="mt-0.5">{fileIcon(f.extension, 18)}</span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium">{f.name}</span>
                      {f.snippet && <Snippet text={f.snippet} />}
                    </span>
                  </button>
                  <span className="shrink-0 text-xs text-muted">{formatBytes(f.size_bytes)}</span>
                  <RowActions id={f.id} onPreview={() => openPreview(contentHits, i)} />
                </StaggerItem>
              ))}
            </StaggerList>
          </div>
        </section>
      )}

      {preview && (
        <FilePreview files={preview} index={previewIdx} onChangeIndex={setPreviewIdx} onClose={() => setPreview(null)} />
      )}
    </div>
  );
}

function RowActions({ id, onPreview }: { id: string; onPreview: () => void }) {
  return (
    <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
      <button title="Preview" onClick={onPreview} className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-border hover:text-foreground"><Eye size={15} /></button>
      <a href={filesApi.downloadUrl(id)}><button title="Download" className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-border hover:text-foreground"><Download size={15} /></button></a>
    </div>
  );
}
