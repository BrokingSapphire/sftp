"use client";

import { useState } from "react";
import { toast } from "sonner";
import { motion, AnimatePresence } from "motion/react";
import { Sparkles, Send, Loader2, FileText, User } from "lucide-react";
import { aiApi, type AiAnswer } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Button } from "@/components/ui/button";
import { BRAND } from "@/lib/brand";

interface Turn { q: string; a?: AiAnswer; error?: boolean }

const SUGGESTIONS = [
  "Summarise my most recent documents",
  "What contracts mention payment terms?",
  "Find anything about compliance deadlines",
];

export default function AskPage() {
  const [input, setInput] = useState("");
  const [turns, setTurns] = useState<Turn[]>([]);
  const [busy, setBusy] = useState(false);

  const enabled = BRAND.ai?.enabled ?? false;

  async function ask(question: string) {
    const q = question.trim();
    if (!q || busy) return;
    setInput("");
    setTurns((t) => [...t, { q }]);
    setBusy(true);
    try {
      const a = await aiApi.ask(q);
      setTurns((t) => t.map((turn, i) => (i === t.length - 1 ? { ...turn, a } : turn)));
    } catch {
      setTurns((t) => t.map((turn, i) => (i === t.length - 1 ? { ...turn, error: true } : turn)));
      toast.error("Could not get an answer");
    } finally { setBusy(false); }
  }

  if (!enabled) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageHeader title="Ask your files" subtitle="On-premise AI over your documents" />
        <div className="mt-4 flex min-h-[16rem] flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border bg-surface p-6 text-center text-muted">
          <Sparkles size={40} />
          <p className="max-w-sm text-sm">AI features are turned off. An administrator can enable them in <code className="font-mono">brand.config.json</code> (the AI section) with a self-hosted Ollama server — nothing leaves your network.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto flex h-[calc(100vh-8rem)] max-w-3xl flex-col">
      <PageHeader title="Ask your files" subtitle={`${BRAND.company.shortName} AI · answers grounded in your own documents`} />

      <div className="mt-4 flex-1 space-y-4 overflow-y-auto pb-4">
        {turns.length === 0 && (
          <div className="flex flex-col items-center gap-4 pt-12 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 text-primary"><Sparkles size={26} /></div>
            <p className="text-sm text-muted">Ask a question — answers are drawn only from files you own.</p>
            <div className="flex flex-wrap justify-center gap-2">
              {SUGGESTIONS.map((s) => (
                <button key={s} onClick={() => ask(s)} className="rounded-full border border-border px-3 py-1.5 text-xs text-muted transition-colors hover:border-primary hover:text-primary">{s}</button>
              ))}
            </div>
          </div>
        )}

        <AnimatePresence initial={false}>
          {turns.map((t, i) => (
            <motion.div key={i} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} className="space-y-3">
              <div className="flex justify-end">
                <div className="flex max-w-[80%] items-start gap-2 rounded-2xl rounded-tr-sm bg-primary px-4 py-2.5 text-sm text-primary-foreground">
                  <span>{t.q}</span>
                  <User size={15} className="mt-0.5 shrink-0 opacity-70" />
                </div>
              </div>
              <div className="flex justify-start">
                <div className="flex max-w-[85%] items-start gap-2.5">
                  <span className="mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary"><Sparkles size={15} /></span>
                  <div className="rounded-2xl rounded-tl-sm border border-border bg-surface px-4 py-2.5">
                    {t.a ? (
                      <>
                        <p className="whitespace-pre-wrap text-sm leading-relaxed">{t.a.answer}</p>
                        {t.a.sources && t.a.sources.length > 0 && (
                          <div className="mt-2.5 flex flex-wrap gap-1.5 border-t border-border/50 pt-2">
                            {t.a.sources.map((s) => (
                              <span key={s.file_id} className="flex items-center gap-1 rounded-md bg-surface-2 px-2 py-0.5 text-[11px] text-muted">
                                <FileText size={11} /> {s.name}
                              </span>
                            ))}
                          </div>
                        )}
                      </>
                    ) : t.error ? (
                      <p className="text-sm text-danger">Something went wrong. Try again.</p>
                    ) : (
                      <Loader2 size={16} className="animate-spin text-muted" />
                    )}
                  </div>
                </div>
              </div>
            </motion.div>
          ))}
        </AnimatePresence>
      </div>

      <form onSubmit={(e) => { e.preventDefault(); ask(input); }} className="flex items-center gap-2 border-t border-border pt-3">
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Ask about your files…"
          className="h-11 flex-1 rounded-xl border border-border bg-surface px-4 text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
        />
        <Button type="submit" disabled={busy || !input.trim()} className="h-11">
          {busy ? <Loader2 size={16} className="animate-spin" /> : <Send size={16} />}
        </Button>
      </form>
      <p className="pt-1.5 text-center text-[11px] text-muted">AI can be wrong — verify important details against the source files.</p>
    </div>
  );
}
