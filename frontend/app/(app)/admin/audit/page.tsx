"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Search, Filter, X, FileText, FolderPlus, Share2, UserCog, LogIn,
  KeyRound, Activity, Shield, Globe, ChevronRight,
} from "lucide-react";
import { auditApi } from "@/lib/endpoints";
import type { AuditLog } from "@/lib/types";
import { PageHeader } from "@/components/files/file-list";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/misc";
import { StaggerList, StaggerItem } from "@/components/motion";
import { AnimatePresence, motion } from "motion/react";
import { timeAgo, cn } from "@/lib/utils";

const resultColor: Record<string, string> = {
  success: "text-success bg-success/10",
  failure: "text-danger bg-danger/10",
  denied: "text-warning bg-warning/10",
};
const RESULTS = ["all", "success", "failure", "denied"] as const;

const categoryIcon: Record<string, React.ElementType> = {
  file: FileText, folder: FolderPlus, share: Share2, user: UserCog,
  auth: LogIn, apikey: KeyRound, activity: Activity, admin: Shield, http: Globe,
};

// Human-readable phrasing for an action verb.
function describe(a: AuditLog): string {
  const map: Record<string, string> = {
    "file.upload": "Uploaded a file",
    "file.upload.session": "Started a resumable upload",
    "file.delete": "Permanently deleted a file",
    "file.trash": "Moved a file to trash",
    "file.restore": "Restored a file from trash",
    "file.rename": "Renamed a file",
    "file.move": "Moved a file",
    "file.star": "Starred a file",
    "file.make-common": "Shared a file to Common",
    "file.keep": "Kept an inherited file",
    "folder.create": "Created a folder",
    "folder.rename": "Renamed a folder",
    "folder.delete": "Deleted a folder",
    "folder.color": "Changed a folder colour",
    "share.create": "Created a share link",
    "share.revoke": "Revoked a share link",
    "user.create": "Created a user",
    "user.delete": "Removed a user (files transferred)",
    "user.update": "Updated a user",
    "user.enable": "Re-enabled an account",
    "user.role": "Changed a user's role",
    "user.status": "Enabled/disabled a user",
    "user.reset-password": "Reset a user's password",
    "auth.login": "Signed in",
    "auth.logout": "Signed out",
    "auth.refresh": "Refreshed a session",
    "auth.change-password": "Changed password",
    "apikey.create": "Created an API key",
    "apikey.revoke": "Revoked an API key",
    "activity.track": "UI interaction",
  };
  if (map[a.action]) return map[a.action];
  // Fallback: humanise "resource.verb".
  const [res, ...rest] = a.action.split(".");
  const verb = rest.join(" ").replace(/[-_]/g, " ");
  return `${verb ? verb[0].toUpperCase() + verb.slice(1) : "Action"} · ${res}`;
}

export default function AuditPage() {
  const q = useQuery({ queryKey: ["audit"], queryFn: () => auditApi.list(300) });
  const [query, setQuery] = useState("");
  const [result, setResult] = useState<(typeof RESULTS)[number]>("all");
  const [category, setCategory] = useState<string>("all");
  const [expanded, setExpanded] = useState<number | null>(null);

  const categories = useMemo(() => {
    const set = new Set<string>();
    q.data?.forEach((a) => set.add(a.category));
    return ["all", ...Array.from(set).sort()];
  }, [q.data]);

  const rows = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return (q.data ?? []).filter((a) => {
      if (result !== "all" && a.result !== result) return false;
      if (category !== "all" && a.category !== category) return false;
      if (needle) {
        const hay = `${a.action} ${describe(a)} ${a.actor_email ?? ""} ${a.category} ${a.ip_address ?? ""} ${a.object_id ?? ""}`.toLowerCase();
        if (!hay.includes(needle)) return false;
      }
      return true;
    });
  }, [q.data, query, result, category]);

  const active = query || result !== "all" || category !== "all";

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <PageHeader title="Audit log" subtitle="Immutable, compliance-grade record of every action — click a row for full detail" />

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-[220px] flex-1">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search action, user, IP, object…" className="pl-9" />
        </div>
        <div className="flex items-center gap-1.5 rounded-md border border-border p-0.5">
          {RESULTS.map((r) => (
            <button key={r} onClick={() => setResult(r)}
              className={cn("rounded px-2.5 py-1 text-xs font-medium capitalize transition-colors", result === r ? "bg-primary/10 text-primary" : "text-muted hover:text-foreground")}>
              {r}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-1.5 rounded-md border border-border px-2">
          <Filter size={14} className="text-muted" />
          <select value={category} onChange={(e) => setCategory(e.target.value)} className="h-9 bg-transparent pr-1 text-sm capitalize focus:outline-none">
            {categories.map((c) => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
        {active && (
          <button onClick={() => { setQuery(""); setResult("all"); setCategory("all"); }} className="flex items-center gap-1 rounded-md px-2 py-1.5 text-xs text-muted hover:text-foreground">
            <X size={14} /> Clear
          </button>
        )}
      </div>

      <p className="text-xs text-muted">{q.isLoading ? "Loading…" : `${rows.length} event${rows.length === 1 ? "" : "s"}${active ? " (filtered)" : ""}`}</p>

      {q.isLoading && <Skeleton className="h-96 w-full" />}
      <Card className="overflow-hidden">
        <div className="max-h-[72vh] overflow-y-auto">
          {!q.isLoading && rows.length === 0 && <p className="py-16 text-center text-sm text-muted">No matching events.</p>}
          <StaggerList>
            {rows.map((a) => {
              const open = expanded === a.id;
              const Icon = categoryIcon[a.category] ?? Globe;
              return (
                <StaggerItem key={a.id} className="border-b border-border/50">
                  <button onClick={() => setExpanded(open ? null : a.id)} className="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-surface-2">
                    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-muted"><Icon size={15} /></span>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate text-sm font-medium">{describe(a)}</span>
                        <code className="hidden truncate font-mono text-[11px] text-muted sm:inline">{a.action}</code>
                      </div>
                      <p className="truncate text-xs text-muted">
                        {a.actor_email || "system"}{a.ip_address ? ` · ${a.ip_address}` : ""}{a.browser ? ` · ${a.browser} on ${a.os}` : ""}
                      </p>
                    </div>
                    <span className={cn("hidden shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium capitalize sm:inline", resultColor[a.result] ?? "bg-surface-2 text-muted")}>{a.result}</span>
                    <span className="hidden w-20 shrink-0 text-right text-xs text-muted md:inline">{timeAgo(a.created_at)}</span>
                    <motion.span animate={{ rotate: open ? 90 : 0 }} className="shrink-0 text-muted"><ChevronRight size={15} /></motion.span>
                  </button>

                  <AnimatePresence>
                    {open && (
                      <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: "auto", opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.2 }} className="overflow-hidden bg-surface-2/40">
                        <div className="grid grid-cols-2 gap-x-6 gap-y-2 px-4 py-3 pl-14 text-xs sm:grid-cols-3">
                          <Detail label="Event ID" value={`#${a.id}`} mono />
                          <Detail label="Timestamp" value={new Date(a.created_at).toLocaleString()} />
                          <Detail label="Result" value={a.result} />
                          <Detail label="Actor" value={a.actor_email || "system"} />
                          <Detail label="Category" value={a.category} />
                          <Detail label="Action" value={a.action} mono />
                          <Detail label="Method" value={String(a.metadata?.method ?? "—")} />
                          <Detail label="Path" value={String(a.metadata?.path ?? "—")} mono />
                          <Detail label="Status" value={String(a.metadata?.status ?? "—")} />
                          <Detail label="IP address" value={a.ip_address || "—"} mono />
                          <Detail label="Browser" value={a.browser || "—"} />
                          <Detail label="OS" value={a.os || "—"} />
                          {a.object_id && <Detail label="Object ID" value={a.object_id} mono />}
                          {a.object_name && <Detail label="Object" value={a.object_name} />}
                          {a.request_id && <Detail label="Request ID" value={a.request_id} mono />}
                        </div>
                        {a.user_agent && (
                          <div className="px-4 pb-2 text-xs">
                            <p className="font-mono text-[10px] uppercase tracking-wider text-muted">User agent</p>
                            <p className="mt-0.5 break-all font-mono text-[11px] text-muted">{a.user_agent}</p>
                          </div>
                        )}
                        {a.metadata && Object.keys(a.metadata).length > 0 && (
                          <div className="px-4 pb-3 text-xs">
                            <p className="font-mono text-[10px] uppercase tracking-wider text-muted">Raw metadata</p>
                            <pre className="mt-1 overflow-x-auto rounded-md bg-[#0d1214] p-2.5 font-mono text-[11px] text-zinc-100">{JSON.stringify(a.metadata, null, 2)}</pre>
                          </div>
                        )}
                      </motion.div>
                    )}
                  </AnimatePresence>
                </StaggerItem>
              );
            })}
          </StaggerList>
        </div>
      </Card>
    </div>
  );
}

function Detail({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <p className="font-mono text-[10px] uppercase tracking-wider text-muted">{label}</p>
      <p className={`mt-0.5 break-words capitalize ${mono ? "font-mono text-[11px] normal-case" : ""}`}>{value}</p>
    </div>
  );
}
