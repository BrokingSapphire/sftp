"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Clock, LogOut } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { auditApi } from "@/lib/endpoints";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/button";

// Auto sign-out after IDLE_MS of no activity. During the final WARN_MS the user
// is shown a countdown and can choose to stay signed in.
const IDLE_MS = 15 * 60 * 1000; // 15 minutes total
const WARN_MS = 2 * 60 * 1000; //  last 2 minutes: show the prompt
const ACTIVITY = ["mousemove", "mousedown", "keydown", "scroll", "touchstart", "wheel"];

export function IdleTimeout() {
  const { user, logout } = useAuth();
  const { t, num } = useI18n();
  const [warning, setWarning] = useState(false);
  const [remaining, setRemaining] = useState(Math.floor(WARN_MS / 1000));

  const warnTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const logoutTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const countdown = useRef<ReturnType<typeof setInterval> | null>(null);
  const warningRef = useRef(false);

  const clearAll = useCallback(() => {
    if (warnTimer.current) clearTimeout(warnTimer.current);
    if (logoutTimer.current) clearTimeout(logoutTimer.current);
    if (countdown.current) clearInterval(countdown.current);
  }, []);

  const doLogout = useCallback(async () => {
    clearAll();
    warningRef.current = false;
    setWarning(false);
    // Record the inactivity logout to the audit stream (while still authed), and
    // let the login screen explain why the session ended.
    try { await auditApi.track("session", "idle_logout", window.location.pathname, { reason: "inactivity", after_minutes: 15 }); } catch { /* ignore */ }
    try { sessionStorage.setItem("sphr_logout_reason", "idle"); } catch { /* ignore */ }
    try { await logout(); } catch { /* ignore */ }
  }, [clearAll, logout]);

  // (Re)arm the inactivity timers from a clean slate.
  const arm = useCallback(() => {
    clearAll();
    warningRef.current = false;
    setWarning(false);
    warnTimer.current = setTimeout(() => {
      // Enter the warning phase: show the prompt and count down to logout.
      warningRef.current = true;
      setWarning(true);
      setRemaining(Math.floor(WARN_MS / 1000));
      countdown.current = setInterval(() => {
        setRemaining((s) => (s > 0 ? s - 1 : 0));
      }, 1000);
      logoutTimer.current = setTimeout(doLogout, WARN_MS);
    }, IDLE_MS - WARN_MS);
  }, [clearAll, doLogout]);

  useEffect(() => {
    if (!user) { clearAll(); return; }
    arm();
    const onActivity = () => {
      // Ignore activity once the prompt is up — the user must choose explicitly.
      if (warningRef.current) return;
      arm();
    };
    ACTIVITY.forEach((e) => window.addEventListener(e, onActivity, { passive: true }));
    return () => {
      ACTIVITY.forEach((e) => window.removeEventListener(e, onActivity));
      clearAll();
    };
  }, [user, arm, clearAll]);

  if (!user || !warning) return null;

  const mm = String(Math.floor(remaining / 60)).padStart(1, "0");
  const ss = String(remaining % 60).padStart(2, "0");

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
        className="fixed inset-0 z-[70] flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
      >
        <motion.div
          initial={{ opacity: 0, scale: 0.96, y: 12 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.96 }}
          className="w-full max-w-sm overflow-hidden rounded-2xl border border-border bg-surface p-6 text-center shadow-2xl"
        >
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-warning/10 text-warning">
            <Clock size={26} />
          </div>
          <h2 className="mt-4 text-lg font-semibold">{t("idle.title")}</h2>
          <p className="mt-1 text-sm text-muted">{t("idle.body")}</p>
          <p className="my-3 font-mono text-3xl font-bold tabular-nums text-warning">{num(mm)}:{num(ss)}</p>
          <div className="flex gap-2">
            <Button variant="outline" className="flex-1" onClick={doLogout}>
              <LogOut size={15} /> {t("idle.logoutNow")}
            </Button>
            <Button className="flex-1" onClick={arm}>{t("idle.stay")}</Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}
