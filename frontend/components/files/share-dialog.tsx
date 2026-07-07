"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { motion } from "motion/react";
import {
  Link2, Copy, Check, Lock, Clock, X, ShieldAlert, Loader2, Users, Globe, ChevronDown, Trash2, UserPlus,
} from "lucide-react";
import { sharesApi, filesApi, type ShareCreateResult, type FileGrant } from "@/lib/endpoints";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar } from "@/components/ui/avatar";
import { isExternalEmail as isExternal } from "@/lib/brand";

export function ShareDialog({ fileId, fileName, onClose }: { fileId: string; fileName: string; onClose: () => void }) {
  // People-with-access (internal shares)
  const [grants, setGrants] = useState<FileGrant[]>([]);
  const [personEmail, setPersonEmail] = useState("");
  const [canWrite, setCanWrite] = useState(false);
  const [adding, setAdding] = useState(false);

  // General access (link)
  const [showLink, setShowLink] = useState(false);
  const [password, setPassword] = useState("");
  const [expires, setExpires] = useState<number | "">("");
  const [limit, setLimit] = useState<number | "">("");
  const [linkBusy, setLinkBusy] = useState(false);
  const [link, setLink] = useState<ShareCreateResult | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    // The API omits an empty array, so coalesce to [] — otherwise grants.map
    // below would crash (files with no recipients are the common case).
    filesApi.listGrants(fileId).then((g) => setGrants(g ?? [])).catch(() => setGrants([]));
  }, [fileId]);

  const personExternal = personEmail ? isExternal(personEmail) : false;

  async function addPerson() {
    if (!personEmail) return;
    setAdding(true);
    try {
      const g = await filesApi.shareWithUser(fileId, personEmail, canWrite);
      setGrants((gs) => [...gs.filter((x) => x.user_id !== g.user_id), g]);
      setPersonEmail("");
      toast.success(`Shared with ${g.name}`);
    } catch (e) {
      const msg = (e as { message?: string })?.message;
      toast.error(msg?.includes("not found") ? "No user with that email" : "Could not share");
    } finally { setAdding(false); }
  }
  async function removePerson(g: FileGrant) {
    try { await filesApi.revokeGrant(fileId, g.user_id); setGrants((gs) => gs.filter((x) => x.user_id !== g.user_id)); }
    catch { toast.error("Could not remove access"); }
  }
  async function setRole(g: FileGrant, write: boolean) {
    try { const ng = await filesApi.shareWithUser(fileId, g.email, write); setGrants((gs) => gs.map((x) => (x.user_id === g.user_id ? ng : x))); }
    catch { toast.error("Could not update role"); }
  }

  async function createLink() {
    setLinkBusy(true);
    try {
      const res = await sharesApi.create(fileId, {
        password: password || undefined,
        expires_in_days: expires === "" ? undefined : Number(expires),
        download_limit: limit === "" ? undefined : Number(limit),
      });
      setLink(res);
      toast.success("Link created");
    } catch (e) {
      const msg = (e as { message?: string })?.message;
      toast.error(msg && msg.includes("restricted") ? msg : "Could not create link");
    }
    finally { setLinkBusy(false); }
  }
  function copyLink() {
    if (!link) return;
    navigator.clipboard.writeText(link.url);
    setCopied(true); setTimeout(() => setCopied(false), 1200);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <motion.div
        initial={{ opacity: 0, scale: 0.97, y: 8 }} animate={{ opacity: 1, scale: 1, y: 0 }}
        className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-surface p-5 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h3 className="truncate text-base font-semibold">Share &ldquo;{fileName}&rdquo;</h3>
          <button onClick={onClose} className="text-muted hover:text-foreground"><X size={18} /></button>
        </div>

        {/* Add people */}
        <div className="flex items-start gap-2">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 rounded-lg border border-border px-2 focus-within:ring-2 focus-within:ring-ring/40">
              <UserPlus size={15} className="shrink-0 text-muted" />
              <input
                value={personEmail}
                onChange={(e) => setPersonEmail(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && addPerson()}
                placeholder="Add people by email"
                className="h-10 min-w-0 flex-1 bg-transparent text-sm focus:outline-none"
              />
            </div>
            {personExternal && (
              <p className="mt-1 flex items-center gap-1 text-[11px] text-warning"><ShieldAlert size={12} /> Outside your organisation</p>
            )}
          </div>
          <select value={canWrite ? "edit" : "view"} onChange={(e) => setCanWrite(e.target.value === "edit")}
            className="h-10 rounded-lg border border-border bg-surface px-2 text-sm">
            <option value="view">Viewer</option>
            <option value="edit">Editor</option>
          </select>
          <Button size="sm" className="h-10" onClick={addPerson} disabled={adding || !personEmail}>
            {adding ? <Loader2 size={15} className="animate-spin" /> : "Share"}
          </Button>
        </div>

        {/* People with access */}
        <div className="mt-4">
          <p className="mb-2 flex items-center gap-1.5 text-xs font-medium text-muted"><Users size={13} /> People with access</p>
          <div className="space-y-1">
            <div className="flex items-center gap-2 rounded-lg px-1 py-1.5">
              <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/15 text-xs font-semibold text-primary">Me</span>
              <div className="min-w-0 flex-1"><p className="truncate text-sm font-medium">You</p><p className="text-xs text-muted">Owner</p></div>
            </div>
            {(grants ?? []).map((g) => (
              <div key={g.user_id} className="flex items-center gap-2 rounded-lg px-1 py-1.5 hover:bg-surface-2">
                <Avatar userId={g.user_id} name={g.name} hasAvatar={g.has_avatar} size={32} />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{g.name}</p>
                  <p className="truncate text-xs text-muted">{g.email}</p>
                </div>
                <select value={g.can_write ? "edit" : "view"} onChange={(e) => setRole(g, e.target.value === "edit")}
                  className="h-8 rounded-md border border-border bg-surface px-1.5 text-xs">
                  <option value="view">Viewer</option>
                  <option value="edit">Editor</option>
                </select>
                <button title="Remove" onClick={() => removePerson(g)} className="flex h-8 w-8 items-center justify-center rounded-md text-danger hover:bg-surface-2"><Trash2 size={14} /></button>
              </div>
            ))}
            {(grants ?? []).length === 0 && <p className="px-1 text-xs text-muted">Only you have access.</p>}
          </div>
        </div>

        {/* General access — link */}
        <div className="mt-4 rounded-xl border border-border p-3">
          <button onClick={() => setShowLink((s) => !s)} className="flex w-full items-center gap-2 text-left">
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-surface-2 text-muted"><Globe size={15} /></span>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">General access — link</p>
              <p className="text-xs text-muted">{link ? "Anyone with the link" : "Create a link anyone can open"}</p>
            </div>
            <motion.span animate={{ rotate: showLink ? 180 : 0 }} className="text-muted"><ChevronDown size={16} /></motion.span>
          </button>

          {showLink && (
            <div className="mt-3 space-y-3">
              {!link ? (
                <>
                  <div className="grid grid-cols-3 gap-2">
                    <IconField icon={Lock} label="Password"><Input type="password" placeholder="None" value={password} onChange={(e) => setPassword(e.target.value)} /></IconField>
                    <IconField icon={Clock} label="Expiry (d)"><Input type="number" min={1} placeholder="Never" value={expires} onChange={(e) => setExpires(e.target.value === "" ? "" : Number(e.target.value))} /></IconField>
                    <IconField icon={Link2} label="Max dl"><Input type="number" min={1} placeholder="∞" value={limit} onChange={(e) => setLimit(e.target.value === "" ? "" : Number(e.target.value))} /></IconField>
                  </div>
                  <Button className="w-full" size="sm" onClick={createLink} disabled={linkBusy}>
                    {linkBusy ? <Loader2 size={15} className="animate-spin" /> : <Link2 size={15} />} Create link
                  </Button>
                </>
              ) : (
                <div className="flex items-center gap-2">
                  <code className="min-w-0 flex-1 truncate rounded-md bg-surface-2 px-3 py-2 font-mono text-xs">{link.url}</code>
                  <Button size="sm" variant="outline" onClick={copyLink}>{copied ? <Check size={14} className="text-success" /> : <Copy size={14} />} Copy</Button>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="mt-4 flex items-center justify-end gap-2">
          {link && <Button variant="outline" size="sm" onClick={copyLink} className="mr-auto"><Link2 size={14} /> Copy link</Button>}
          <Button size="sm" onClick={onClose}>Done</Button>
        </div>
      </motion.div>
    </div>
  );
}

function IconField({ icon: Icon, label, children }: { icon: React.ElementType; label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <label className="flex items-center gap-1 text-[11px] font-medium text-muted"><Icon size={11} /> {label}</label>
      {children}
    </div>
  );
}
