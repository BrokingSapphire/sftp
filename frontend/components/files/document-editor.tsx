"use client";

import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { motion } from "motion/react";
import { Save, X, Loader2, FileCode } from "lucide-react";
import { filesApi } from "@/lib/endpoints";
import { Button } from "@/components/ui/button";

const EDITABLE = new Set([
  "txt", "md", "markdown", "csv", "tsv", "log", "json", "yaml", "yml", "xml",
  "html", "htm", "ini", "toml", "env", "conf", "sql",
  "go", "py", "js", "ts", "tsx", "jsx", "java", "c", "h", "cpp", "cc", "cs",
  "rb", "php", "rs", "sh",
]);

export function isEditable(extension?: string) {
  return !!extension && EDITABLE.has(extension.toLowerCase());
}

export function DocumentEditor({
  fileId, fileName, onClose, onSaved,
}: {
  fileId: string; fileName: string; onClose: () => void; onSaved?: () => void;
}) {
  const [text, setText] = useState<string | null>(null);
  const [original, setOriginal] = useState("");
  const [saving, setSaving] = useState(false);
  const taRef = useRef<HTMLTextAreaElement>(null);

  const dirty = text !== null && text !== original;

  useEffect(() => {
    filesApi.fetchText(fileId)
      .then((t) => { setText(t); setOriginal(t); })
      .catch(() => { toast.error("Could not open file"); onClose(); });
  }, [fileId, onClose]);

  const save = async () => {
    if (text === null || !dirty) return;
    setSaving(true);
    try {
      await filesApi.saveContent(fileId, text);
      setOriginal(text);
      toast.success("Saved — new version created");
      onSaved?.();
    } catch (e) {
      const msg = (e as { message?: string })?.message;
      toast.error(msg?.includes("legal hold") || msg?.includes("retention") ? msg : "Could not save");
    } finally { setSaving(false); }
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") { e.preventDefault(); save(); }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  function requestClose() {
    if (dirty && !confirm("Discard unsaved changes?")) return;
    onClose();
  }

  // Tab inserts two spaces instead of moving focus.
  function onKeyDownTA(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Tab") {
      e.preventDefault();
      const ta = e.currentTarget;
      const s = ta.selectionStart, en = ta.selectionEnd;
      const val = ta.value;
      const next = val.slice(0, s) + "  " + val.slice(en);
      setText(next);
      requestAnimationFrame(() => { ta.selectionStart = ta.selectionEnd = s + 2; });
    }
  }

  const lines = (text ?? "").split("\n").length;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4" onClick={requestClose}>
      <motion.div
        initial={{ opacity: 0, scale: 0.98 }} animate={{ opacity: 1, scale: 1 }}
        className="flex h-[85vh] w-full max-w-4xl flex-col overflow-hidden rounded-2xl border border-border bg-surface shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
          <FileCode size={16} className="text-primary" />
          <span className="min-w-0 flex-1 truncate text-sm font-medium">{fileName}</span>
          {dirty && <span className="rounded-full bg-warning/15 px-2 py-0.5 text-[10px] font-medium text-warning">Unsaved</span>}
          <span className="hidden text-xs text-muted sm:inline">{lines} lines</span>
          <Button size="sm" onClick={save} disabled={saving || !dirty}>
            {saving ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />} Save
          </Button>
          <button onClick={requestClose} className="ml-1 text-muted hover:text-foreground"><X size={18} /></button>
        </div>

        {text === null ? (
          <div className="flex flex-1 items-center justify-center text-muted"><Loader2 className="animate-spin" /></div>
        ) : (
          <textarea
            ref={taRef}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={onKeyDownTA}
            spellCheck={false}
            className="flex-1 resize-none bg-[#0d1214] p-4 font-mono text-[13px] leading-relaxed text-zinc-100 focus:outline-none"
          />
        )}
        <div className="border-t border-border px-4 py-1.5 text-[11px] text-muted">
          Saving creates a new version · ⌘/Ctrl+S to save
        </div>
      </motion.div>
    </div>
  );
}
