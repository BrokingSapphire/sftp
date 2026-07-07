"use client";

import { useEffect, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import {
  FolderLock, Search, Share2, UsersRound, Sparkles, ShieldCheck, Code2, X, ArrowRight, ArrowLeft, PartyPopper,
} from "lucide-react";
import { useAuth } from "@/lib/auth";
import { BRAND } from "@/lib/brand";
import { Button } from "@/components/ui/button";

/**
 * A friendly, plain-English feature tour shown once, on a user's first visit.
 * Each slide can show a real screenshot from /public/onboarding/<img>; if the
 * image isn't there yet, it gracefully falls back to a branded icon panel.
 */
const SLIDES = [
  {
    icon: FolderLock, img: "files.png", tag: "Your files, your server",
    title: "Everything lives on your own network",
    body: "No cloud, no third parties. Upload, organise into folders, drag-and-drop, and preview images, PDFs and Office files right in the browser — all on infrastructure you control.",
  },
  {
    icon: Search, img: "search.png", tag: "Find anything, fast",
    title: "Search names — and the words inside files",
    body: "Type in the search bar to find a file by its name or by the text inside it. Ask a question and jump straight to the document that answers it.",
  },
  {
    icon: Share2, img: "share.png", tag: "Share without the risk",
    title: "Send a link, or invite specific people",
    body: "Share a file as a link (add a password or an expiry date) or give named colleagues view/edit access. You always see who has access, and can revoke it in one click.",
  },
  {
    icon: UsersRound, img: "teams.png", tag: "Better together",
    title: "Team Spaces for your department",
    body: "Create a shared drive owned by a group. Add members with roles (owner, admin, member, viewer) so the whole team works from one place — no more 'who has the latest file?'.",
  },
  {
    icon: Sparkles, img: "ask.png", tag: "Ask your files",
    title: "AI that answers from your own documents",
    body: "Ask a plain-English question and get an answer drawn from your files, with links to the sources. It runs entirely on your servers — your data never leaves.",
  },
  {
    icon: ShieldCheck, img: "audit.png", tag: "Safe by design",
    title: "Encryption, versions, trash & a full audit trail",
    body: "Files can be encrypted at rest, every version is kept, deletes go to trash first, and every action is logged — so nothing is ever truly lost or untraceable.",
  },
  {
    icon: Code2, img: "api.png", tag: "For the builders",
    title: "Automate it with the API",
    body: "Create an API key and connect scripts, backups or other apps — with ready-made code snippets in cURL, JavaScript, Python and Go on the API keys page.",
  },
];

export function WelcomeModal() {
  const { user } = useAuth();
  const [open, setOpen] = useState(false);
  const [i, setI] = useState(0);

  const storageKey = user ? `sphr_welcome_seen_${user.id}` : "";

  useEffect(() => {
    if (!user) return;
    try {
      if (!localStorage.getItem(`sphr_welcome_seen_${user.id}`)) setOpen(true);
    } catch { /* ignore */ }
  }, [user]);

  function close() {
    try { if (storageKey) localStorage.setItem(storageKey, "1"); } catch { /* ignore */ }
    setOpen(false);
  }

  if (!user) return null;
  const s = SLIDES[i];
  const last = i === SLIDES.length - 1;
  const Icon = s.icon;

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
          className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
        >
          <motion.div
            initial={{ opacity: 0, y: 16, scale: 0.98 }} animate={{ opacity: 1, y: 0, scale: 1 }} exit={{ opacity: 0, scale: 0.98 }}
            transition={{ type: "spring", stiffness: 260, damping: 24 }}
            className="w-full max-w-2xl overflow-hidden rounded-2xl border border-border bg-surface shadow-2xl"
          >
            {/* Header */}
            <div className="flex items-center justify-between border-b border-border px-5 py-3">
              <div className="flex items-center gap-2">
                <img src={BRAND.logo.full} alt="" width={22} height={22} />
                <span className="text-sm font-semibold">Welcome to {BRAND.company.product}</span>
              </div>
              <button onClick={close} className="text-muted hover:text-foreground"><X size={18} /></button>
            </div>

            {/* Visual */}
            <div className="relative aspect-[16/7] w-full overflow-hidden bg-gradient-to-br from-primary/15 via-surface-2 to-surface">
              <AnimatePresence mode="wait">
                <motion.div
                  key={i}
                  initial={{ opacity: 0, x: 24 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -24 }}
                  transition={{ duration: 0.25 }}
                  className="absolute inset-0 flex items-center justify-center"
                >
                  {/* Screenshot if present, else a branded icon panel. */}
                  <Screenshot src={`/onboarding/${s.img}`} fallback={<Icon size={72} className="text-primary/70" strokeWidth={1.25} />} />
                </motion.div>
              </AnimatePresence>
            </div>

            {/* Copy */}
            <div className="px-6 py-5">
              <p className="eyebrow text-primary">{s.tag}</p>
              <h2 className="mt-1 text-xl font-semibold tracking-tight">{s.title}</h2>
              <p className="mt-2 text-sm leading-relaxed text-muted">{s.body}</p>
            </div>

            {/* Footer / nav */}
            <div className="flex items-center justify-between border-t border-border px-5 py-3">
              <div className="flex gap-1.5">
                {SLIDES.map((_, idx) => (
                  <button key={idx} onClick={() => setI(idx)} aria-label={`Slide ${idx + 1}`}
                    className={`h-1.5 rounded-full transition-all ${idx === i ? "w-5 bg-primary" : "w-1.5 bg-border hover:bg-muted"}`} />
                ))}
              </div>
              <div className="flex items-center gap-2">
                <button onClick={close} className="text-xs text-muted hover:text-foreground">Skip</button>
                {i > 0 && (
                  <Button variant="outline" size="sm" onClick={() => setI(i - 1)}><ArrowLeft size={15} /> Back</Button>
                )}
                {last ? (
                  <Button size="sm" onClick={close}><PartyPopper size={15} /> Get started</Button>
                ) : (
                  <Button size="sm" onClick={() => setI(i + 1)}>Next <ArrowRight size={15} /></Button>
                )}
              </div>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

function Screenshot({ src, fallback }: { src: string; fallback: React.ReactNode }) {
  const [failed, setFailed] = useState(false);
  if (failed) return <>{fallback}</>;
  // eslint-disable-next-line @next/next/no-img-element
  return <img src={src} alt="" onError={() => setFailed(true)} className="max-h-full max-w-full object-contain shadow-lg" />;
}
