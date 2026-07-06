"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import {
  X, Download, Share2, Star, Trash2, Info, ChevronLeft, ChevronRight,
  FileWarning, Loader2, Copy, Check,
} from "lucide-react";
import { toast } from "sonner";
import type { FileItem } from "@/lib/types";
import { filesApi, sharesApi } from "@/lib/endpoints";
import { fileIcon } from "./icon";
import { SpreadsheetPreview, DocxPreview, PptxPreview } from "./office-preview";
import { formatBytes, timeAgo } from "@/lib/utils";

type Kind =
  | "image" | "pdf" | "video" | "audio" | "text" | "csv" | "json" | "markdown"
  | "spreadsheet" | "docx" | "pptx" | "none";

const EXT: Record<string, Kind> = {
  png: "image", jpg: "image", jpeg: "image", gif: "image", svg: "image", webp: "image", bmp: "image", ico: "image", avif: "image",
  pdf: "pdf",
  mp4: "video", webm: "video", mov: "video", mkv: "video", ogv: "video",
  mp3: "audio", wav: "audio", ogg: "audio", flac: "audio", m4a: "audio", aac: "audio",
  csv: "csv", json: "json", md: "markdown", markdown: "markdown",
  xlsx: "spreadsheet", xls: "spreadsheet", xlsm: "spreadsheet", ods: "spreadsheet",
  docx: "docx", pptx: "pptx",
  txt: "text", log: "text", ini: "text", conf: "text", yaml: "text", yml: "text", env: "text",
  js: "text", ts: "text", tsx: "text", jsx: "text", go: "text", py: "text", java: "text",
  c: "text", cpp: "text", h: "text", css: "text", html: "text", xml: "text", sh: "text",
  rb: "text", php: "text", rs: "text", sql: "text", toml: "text",
};

function kindOf(f: FileItem): Kind {
  return EXT[f.extension?.toLowerCase()] ?? (f.mime_type?.startsWith("image/") ? "image" : "none");
}

interface Props {
  files: FileItem[];
  index: number;
  onClose: () => void;
  onChangeIndex: (i: number) => void;
  onChanged?: () => void; // refetch listing after star/trash
}

export function FilePreview({ files, index, onClose, onChangeIndex, onChanged }: Props) {
  const file = files[index];
  const kind = useMemo(() => (file ? kindOf(file) : "none"), [file]);
  const [showInfo, setShowInfo] = useState(false);

  const next = useCallback(() => index < files.length - 1 && onChangeIndex(index + 1), [index, files.length, onChangeIndex]);
  const prev = useCallback(() => index > 0 && onChangeIndex(index - 1), [index, onChangeIndex]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
      else if (e.key === "ArrowRight") next();
      else if (e.key === "ArrowLeft") prev();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose, next, prev]);

  if (!file) return null;

  async function star() {
    try { await filesApi.starFile(file.id, !file.is_starred); onChanged?.(); toast.success(file.is_starred ? "Unstarred" : "Starred"); }
    catch { toast.error("Failed"); }
  }
  async function trash() {
    try { await filesApi.trashFile(file.id); toast.success("Moved to trash"); onChanged?.(); onClose(); }
    catch { toast.error("Failed"); }
  }
  async function share() {
    try {
      const res = await sharesApi.create(file.id, {});
      await navigator.clipboard.writeText(res.url).catch(() => {});
      toast.success("Share link copied");
    } catch { toast.error("Could not share"); }
  }

  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0 z-50 flex flex-col bg-black/80 backdrop-blur-sm"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
      >
        {/* Top bar */}
        <div
          className="flex items-center gap-3 px-4 py-3 text-white"
          onClick={(e) => e.stopPropagation()}
        >
          <span className="shrink-0">{fileIcon(file.extension, 20)}</span>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{file.name}</p>
            <p className="font-mono text-[11px] text-white/50">
              {formatBytes(file.size_bytes)} · {file.mime_type}
            </p>
          </div>
          <PreviewBtn title="Info" active={showInfo} onClick={() => setShowInfo((s) => !s)}><Info size={18} /></PreviewBtn>
          <PreviewBtn title="Star" onClick={star}><Star size={18} className={file.is_starred ? "fill-amber-400 text-amber-400" : ""} /></PreviewBtn>
          <PreviewBtn title="Share" onClick={share}><Share2 size={18} /></PreviewBtn>
          <a href={filesApi.downloadUrl(file.id)} onClick={(e) => e.stopPropagation()}>
            <PreviewBtn title="Download" onClick={() => {}}><Download size={18} /></PreviewBtn>
          </a>
          <PreviewBtn title="Delete" onClick={trash}><Trash2 size={18} /></PreviewBtn>
          <div className="mx-1 h-6 w-px bg-white/20" />
          <PreviewBtn title="Close" onClick={onClose}><X size={18} /></PreviewBtn>
        </div>

        {/* Body */}
        <div className="relative flex min-h-0 flex-1" onClick={(e) => e.stopPropagation()}>
          {/* Prev / next */}
          {index > 0 && (
            <NavArrow side="left" onClick={prev} />
          )}
          {index < files.length - 1 && (
            <NavArrow side="right" onClick={next} />
          )}

          <motion.div
            key={file.id}
            className="flex min-h-0 flex-1 items-center justify-center overflow-auto p-6"
            initial={{ opacity: 0, scale: 0.98 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
          >
            <PreviewContent file={file} kind={kind} />
          </motion.div>

          {/* Info panel */}
          <AnimatePresence>
            {showInfo && (
              <motion.aside
                className="w-80 shrink-0 overflow-y-auto border-l border-white/10 bg-[#0d1214] p-5 text-white"
                initial={{ x: 320, opacity: 0 }}
                animate={{ x: 0, opacity: 1 }}
                exit={{ x: 320, opacity: 0 }}
                transition={{ type: "spring", stiffness: 320, damping: 32 }}
              >
                <h3 className="mb-4 text-sm font-semibold">File details</h3>
                <Detail label="Name" value={file.name} />
                <Detail label="Type" value={file.mime_type} />
                <Detail label="Size" value={formatBytes(file.size_bytes)} />
                <Detail label="Version" value={`v${file.version_no}`} />
                <Detail label="Downloads" value={String(file.download_count)} />
                <Detail label="Modified" value={timeAgo(file.updated_at)} />
                <Detail label="Created" value={new Date(file.created_at).toLocaleString()} />
                {file.checksum_sha256 && <ChecksumRow value={file.checksum_sha256} />}
              </motion.aside>
            )}
          </AnimatePresence>
        </div>
      </motion.div>
    </AnimatePresence>
  );
}

// ── content renderers ─────────────────────────────────────

function PreviewContent({ file, kind }: { file: FileItem; kind: Kind }) {
  const src = filesApi.previewUrl(file.id);

  if (kind === "image")
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={src} alt={file.name} className="max-h-full max-w-full rounded-lg object-contain shadow-2xl" />;

  if (kind === "pdf")
    return <iframe src={src} title={file.name} className="h-full w-full rounded-lg bg-white shadow-2xl" />;

  if (kind === "video")
    return <video src={src} controls autoPlay className="max-h-full max-w-full rounded-lg shadow-2xl" />;

  if (kind === "audio")
    return (
      <div className="w-full max-w-lg rounded-xl bg-[#12191b] p-8 text-white shadow-2xl">
        <div className="mb-6 flex items-center gap-3">{fileIcon(file.extension, 28)}<span className="truncate font-medium">{file.name}</span></div>
        <audio src={src} controls autoPlay className="w-full" />
      </div>
    );

  if (kind === "text" || kind === "json" || kind === "markdown" || kind === "csv")
    return <TextPreview file={file} kind={kind} />;

  if (kind === "spreadsheet") return <SpreadsheetPreview fileId={file.id} />;
  if (kind === "docx") return <DocxPreview fileId={file.id} />;
  if (kind === "pptx") return <PptxPreview fileId={file.id} />;

  return (
    <div className="flex flex-col items-center gap-3 text-white/70">
      <FileWarning size={44} />
      <p className="text-sm">No preview available for this file type.</p>
      <a href={filesApi.downloadUrl(file.id)}>
        <span className="inline-flex items-center gap-2 rounded-md bg-white/10 px-4 py-2 text-sm font-medium hover:bg-white/20">
          <Download size={16} /> Download
        </span>
      </a>
    </div>
  );
}

function TextPreview({ file, kind }: { file: FileItem; kind: Kind }) {
  const [text, setText] = useState<string | null>(null);
  const [err, setErr] = useState(false);

  useEffect(() => {
    let alive = true;
    filesApi.fetchText(file.id).then((t) => alive && setText(t)).catch(() => alive && setErr(true));
    return () => { alive = false; };
  }, [file.id]);

  if (err) return <p className="text-sm text-white/60">Could not load file.</p>;
  if (text === null) return <Loader2 className="animate-spin text-white/70" size={28} />;

  if (kind === "csv") {
    const rows = text.split(/\r?\n/).filter(Boolean).slice(0, 500).map((r) => r.split(","));
    return (
      <div className="h-full w-full max-w-5xl overflow-auto rounded-lg bg-white text-zinc-900 shadow-2xl dark:bg-[#12191b] dark:text-zinc-100">
        <table className="w-full border-collapse text-sm">
          <tbody>
            {rows.map((cells, i) => (
              <tr key={i} className={i === 0 ? "sticky top-0 bg-surface-2 font-semibold" : "odd:bg-black/[0.02]"}>
                {cells.map((c, j) => <td key={j} className="border border-black/5 px-3 py-1.5 dark:border-white/5">{c}</td>)}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  const body = kind === "json" ? safeJson(text) : text;
  return (
    <pre className="h-full w-full max-w-5xl overflow-auto rounded-lg bg-[#0d1214] p-5 font-mono text-[13px] leading-relaxed text-zinc-100 shadow-2xl">
      {body}
    </pre>
  );
}

function safeJson(t: string) {
  try { return JSON.stringify(JSON.parse(t), null, 2); } catch { return t; }
}

// ── bits ──────────────────────────────────────────────────

function PreviewBtn({ children, title, onClick, active }: { children: React.ReactNode; title: string; onClick: () => void; active?: boolean }) {
  return (
    <button
      title={title}
      onClick={(e) => { e.stopPropagation(); onClick(); }}
      className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${active ? "bg-white/20 text-white" : "text-white/70 hover:bg-white/10 hover:text-white"}`}
    >
      {children}
    </button>
  );
}

function NavArrow({ side, onClick }: { side: "left" | "right"; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`absolute top-1/2 z-10 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-white/10 text-white backdrop-blur transition-all hover:bg-white/25 ${side === "left" ? "left-4" : "right-4"}`}
    >
      {side === "left" ? <ChevronLeft size={22} /> : <ChevronRight size={22} />}
    </button>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="mb-3">
      <p className="font-mono text-[10px] uppercase tracking-wider text-white/40">{label}</p>
      <p className="mt-0.5 break-words text-sm text-white/90">{value}</p>
    </div>
  );
}

function ChecksumRow({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="mb-3">
      <p className="font-mono text-[10px] uppercase tracking-wider text-white/40">SHA-256</p>
      <button
        onClick={() => { navigator.clipboard.writeText(value); setCopied(true); setTimeout(() => setCopied(false), 1200); }}
        className="mt-0.5 flex w-full items-center gap-2 rounded-md bg-white/5 px-2 py-1.5 text-left font-mono text-[11px] text-white/70 hover:bg-white/10"
      >
        <span className="min-w-0 flex-1 truncate">{value}</span>
        {copied ? <Check size={13} className="text-emerald-400" /> : <Copy size={13} />}
      </button>
    </div>
  );
}
