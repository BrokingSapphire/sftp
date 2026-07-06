"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AnimatePresence, motion } from "motion/react";
import { Bell, CheckCheck } from "lucide-react";
import { notificationsApi } from "@/lib/endpoints";
import { timeAgo } from "@/lib/utils";

export function NotificationBell() {
  const qc = useQueryClient();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const unread = useQuery({
    queryKey: ["notif-unread"],
    queryFn: () => notificationsApi.unreadCount(),
    refetchInterval: 20000,
  });
  const list = useQuery({
    queryKey: ["notif-list"],
    queryFn: () => notificationsApi.list(),
    enabled: open,
  });

  useEffect(() => {
    const onDoc = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false); };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  const count = unread.data?.unread ?? 0;

  async function markAll() {
    await notificationsApi.markAllRead();
    qc.invalidateQueries({ queryKey: ["notif-unread"] });
    qc.invalidateQueries({ queryKey: ["notif-list"] });
  }
  async function openItem(id: string, link?: string) {
    await notificationsApi.markRead(id).catch(() => {});
    qc.invalidateQueries({ queryKey: ["notif-unread"] });
    setOpen(false);
    if (link) router.push(link);
  }

  return (
    <div className="relative" ref={ref}>
      <button
        aria-label="Notifications"
        onClick={() => setOpen((o) => !o)}
        className="relative flex h-9 w-9 items-center justify-center rounded-lg text-foreground hover:bg-surface-2"
      >
        <Bell size={18} />
        {count > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-danger px-1 text-[10px] font-semibold text-white">
            {count > 9 ? "9+" : count}
          </span>
        )}
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -6, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -6, scale: 0.98 }}
            transition={{ duration: 0.15 }}
            className="absolute right-0 top-11 z-50 w-80 overflow-hidden rounded-xl border border-border bg-surface shadow-xl"
          >
            <div className="flex items-center justify-between border-b border-border px-3 py-2">
              <span className="text-sm font-semibold">Notifications</span>
              {count > 0 && (
                <button onClick={markAll} className="flex items-center gap-1 text-xs text-primary hover:underline">
                  <CheckCheck size={13} /> Mark all read
                </button>
              )}
            </div>
            <div className="max-h-96 overflow-y-auto">
              {list.isLoading && <p className="p-4 text-center text-sm text-muted">Loading…</p>}
              {list.data?.length === 0 && <p className="p-6 text-center text-sm text-muted">No notifications yet.</p>}
              {list.data?.map((n) => (
                <button
                  key={n.id}
                  onClick={() => openItem(n.id, n.link)}
                  className={`flex w-full flex-col items-start gap-0.5 border-b border-border/50 px-3 py-2.5 text-left transition-colors hover:bg-surface-2 ${n.is_read ? "" : "bg-primary/5"}`}
                >
                  <div className="flex w-full items-center gap-2">
                    {!n.is_read && <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-primary" />}
                    <span className="flex-1 truncate text-sm font-medium">{n.title}</span>
                    <span className="shrink-0 text-[10px] text-muted">{timeAgo(n.created_at)}</span>
                  </div>
                  <p className="line-clamp-2 text-xs text-muted">{n.body}</p>
                </button>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
