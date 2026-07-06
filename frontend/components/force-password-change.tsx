"use client";

import { useState } from "react";
import { toast } from "sonner";
import { motion } from "motion/react";
import { KeyRound, Loader2, ShieldAlert } from "lucide-react";
import { authApi } from "@/lib/endpoints";
import { ApiError, tokens } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Logo } from "@/components/logo";

/**
 * Blocking screen shown when a user must change their password (first login /
 * admin reset). Since change-password revokes all sessions, we log the user out
 * afterwards so they sign in fresh with the new credentials.
 */
export function ForcePasswordChange() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (next.length < 12) return toast.error("New password must be at least 12 characters");
    if (next !== confirm) return toast.error("Passwords do not match");
    setBusy(true);
    try {
      await authApi.changePassword(current, next);
      tokens.clear();
      toast.success("Password updated — please sign in again");
      window.location.href = "/login";
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Could not change password");
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-6">
      <motion.div
        initial={{ opacity: 0, y: 14 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
        className="w-full max-w-sm"
      >
        <div className="mb-6 flex justify-center"><Logo size={34} /></div>
        <div className="rounded-2xl border border-border bg-surface p-6 shadow-xl">
          <div className="mb-4 flex items-center gap-2 rounded-lg bg-warning/10 px-3 py-2 text-warning">
            <ShieldAlert size={16} className="shrink-0" />
            <p className="text-xs font-medium">For your security, set a new password before continuing.</p>
          </div>

          <form onSubmit={submit} className="space-y-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Current (temporary) password</label>
              <Input type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">New password</label>
              <Input type="password" autoComplete="new-password" placeholder="At least 12 characters" value={next} onChange={(e) => setNext(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Confirm new password</label>
              <Input type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
            </div>
            <Button type="submit" className="w-full" disabled={busy}>
              {busy ? <Loader2 size={16} className="animate-spin" /> : <KeyRound size={16} />}
              Set new password
            </Button>
          </form>
        </div>
      </motion.div>
    </div>
  );
}
