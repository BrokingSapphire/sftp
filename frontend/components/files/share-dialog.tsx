"use client";

import { useState } from "react";
import { toast } from "sonner";
import { motion } from "motion/react";
import { Link2, Copy, Check, Mail, Lock, Clock, X, ShieldAlert, Loader2 } from "lucide-react";
import { sharesApi, type ShareCreateResult } from "@/lib/endpoints";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const ORG_DOMAINS = (process.env.NEXT_PUBLIC_ORG_DOMAINS ?? "")
  .split(",").map((d) => d.trim().toLowerCase()).filter(Boolean);

function isExternal(email: string) {
  const at = email.lastIndexOf("@");
  if (at < 0) return false;
  const domain = email.slice(at + 1).toLowerCase();
  return ORG_DOMAINS.length > 0 && !ORG_DOMAINS.includes(domain);
}

export function ShareDialog({ fileId, fileName, onClose }: { fileId: string; fileName: string; onClose: () => void }) {
  const [password, setPassword] = useState("");
  const [email, setEmail] = useState("");
  const [expires, setExpires] = useState<number | "">("");
  const [limit, setLimit] = useState<number | "">("");
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<ShareCreateResult | null>(null);
  const [copied, setCopied] = useState(false);

  const external = email ? isExternal(email) : false;

  async function create() {
    setBusy(true);
    try {
      const res = await sharesApi.create(fileId, {
        password: password || undefined,
        expires_in_days: expires === "" ? undefined : Number(expires),
        download_limit: limit === "" ? undefined : Number(limit),
        recipient_email: email || undefined,
      });
      setResult(res);
      if (res.emailed) toast.success(`Link emailed to ${email}`);
      else toast.success("Share link created");
    } catch { toast.error("Could not create share"); }
    finally { setBusy(false); }
  }

  function copyLink() {
    if (!result) return;
    navigator.clipboard.writeText(result.url);
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <motion.div
        initial={{ opacity: 0, scale: 0.97, y: 8 }} animate={{ opacity: 1, scale: 1, y: 0 }}
        className="w-full max-w-md rounded-2xl border border-border bg-surface p-5 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary"><Link2 size={18} /></div>
            <div>
              <h3 className="text-sm font-semibold">Share file</h3>
              <p className="truncate text-xs text-muted">{fileName}</p>
            </div>
          </div>
          <button onClick={onClose} className="text-muted hover:text-foreground"><X size={18} /></button>
        </div>

        {!result ? (
          <div className="space-y-3">
            <Field icon={Lock} label="Password (optional)">
              <Input type="password" placeholder="Leave empty for a public link" value={password} onChange={(e) => setPassword(e.target.value)} />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field icon={Clock} label="Expires (days)">
                <Input type="number" min={1} placeholder="Never" value={expires} onChange={(e) => setExpires(e.target.value === "" ? "" : Number(e.target.value))} />
              </Field>
              <Field icon={Link2} label="Download limit">
                <Input type="number" min={1} placeholder="Unlimited" value={limit} onChange={(e) => setLimit(e.target.value === "" ? "" : Number(e.target.value))} />
              </Field>
            </div>
            <Field icon={Mail} label="Email to (optional)">
              <Input type="email" placeholder="person@company.com" value={email} onChange={(e) => setEmail(e.target.value)} />
            </Field>

            {external && (
              <div className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning">
                <ShieldAlert size={15} className="mt-0.5 shrink-0" />
                <p><strong>{email.slice(email.lastIndexOf("@") + 1)}</strong> is outside your organisation. This external share will be logged.</p>
              </div>
            )}

            <Button className="w-full" onClick={create} disabled={busy}>
              {busy ? <Loader2 size={16} className="animate-spin" /> : <Link2 size={16} />}
              {email ? "Create & send link" : "Create link"}
            </Button>
          </div>
        ) : (
          <div className="space-y-3">
            {result.external && (
              <div className="flex items-center gap-2 rounded-lg border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning">
                <ShieldAlert size={15} /> Shared outside the organisation — this action was recorded.
              </div>
            )}
            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded-md bg-surface-2 px-3 py-2 font-mono text-xs">{result.url}</code>
              <Button size="sm" variant="outline" onClick={copyLink}>
                {copied ? <Check size={14} className="text-success" /> : <Copy size={14} />} Copy
              </Button>
            </div>
            {result.emailed && <p className="text-xs text-success">✓ Link emailed to {email}</p>}
            <Button variant="ghost" className="w-full" onClick={onClose}>Done</Button>
          </div>
        )}
      </motion.div>
    </div>
  );
}

function Field({ icon: Icon, label, children }: { icon: React.ElementType; label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <label className="flex items-center gap-1.5 text-xs font-medium text-muted"><Icon size={13} /> {label}</label>
      {children}
    </div>
  );
}
