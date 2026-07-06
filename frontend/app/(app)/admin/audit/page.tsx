"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, Filter, X } from "lucide-react";
import { auditApi } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge, Skeleton } from "@/components/ui/misc";
import { StaggerList, StaggerItem } from "@/components/motion";
import { AnimatePresence, motion } from "motion/react";
import { timeAgo, cn } from "@/lib/utils";

const resultColor: Record<string, string> = {
  success: "text-success",
  failure: "text-danger",
  denied: "text-warning",
};

const RESULTS = ["all", "success", "failure", "denied"] as const;

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
        const hay = `${a.action} ${a.actor_email ?? ""} ${a.category} ${a.ip_address ?? ""}`.toLowerCase();
        if (!hay.includes(needle)) return false;
      }
      return true;
    });
  }, [q.data, query, result, category]);

  const active = query || result !== "all" || category !== "all";

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <PageHeader title="Audit log" subtitle="Immutable, compliance-grade record of every action" />

      {/* Filter + search toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-[220px] flex-1">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search action, user, IP…" className="pl-9" />
        </div>

        <div className="flex items-center gap-1.5 rounded-md border border-border p-0.5">
          {RESULTS.map((r) => (
            <button
              key={r}
              onClick={() => setResult(r)}
              className={cn(
                "rounded px-2.5 py-1 text-xs font-medium capitalize transition-colors",
                result === r ? "bg-primary/10 text-primary" : "text-muted hover:text-foreground",
              )}
            >
              {r}
            </button>
          ))}
        </div>

        <div className="relative flex items-center gap-1.5 rounded-md border border-border px-2">
          <Filter size={14} className="text-muted" />
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="h-9 bg-transparent pr-1 text-sm capitalize focus:outline-none"
          >
            {categories.map((c) => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>

        {active && (
          <button
            onClick={() => { setQuery(""); setResult("all"); setCategory("all"); }}
            className="flex items-center gap-1 rounded-md px-2 py-1.5 text-xs text-muted hover:text-foreground"
          >
            <X size={14} /> Clear
          </button>
        )}
      </div>

      <p className="text-xs text-muted">
        {q.isLoading ? "Loading…" : `${rows.length} event${rows.length === 1 ? "" : "s"}${active ? " (filtered)" : ""}`}
      </p>

      {q.isLoading && <Skeleton className="h-96 w-full" />}
      <Card className="overflow-hidden">
        <div className="grid grid-cols-[10rem_1fr_6rem_7rem] gap-3 border-b border-border bg-surface-2 px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
          <span>Actor</span><span>Action</span><span>Result</span><span className="text-right">When</span>
        </div>
        <div className="max-h-[68vh] overflow-y-auto">
          {!q.isLoading && rows.length === 0 && <p className="py-16 text-center text-sm text-muted">No matching events.</p>}
          <StaggerList>
            {rows.map((a) => {
              const open = expanded === a.id;
              return (
                <StaggerItem key={a.id} className="border-b border-border/50">
                  <button
                    onClick={() => setExpanded(open ? null : a.id)}
                    className="grid w-full grid-cols-[10rem_1fr_6rem_7rem] items-center gap-3 px-4 py-2 text-left text-sm transition-colors hover:bg-surface-2"
                  >
                    <span className="truncate text-muted">{a.actor_email || "system"}</span>
                    <span className="flex min-w-0 items-center gap-2">
                      <Badge>{a.category}</Badge>
                      <span className="truncate font-mono text-xs">{a.action}</span>
                    </span>
                    <span className={`text-xs font-medium capitalize ${resultColor[a.result] ?? ""}`}>{a.result}</span>
                    <span className="text-right text-xs text-muted">{timeAgo(a.created_at)}</span>
                  </button>
                  <AnimatePresence>
                    {open && (
                      <motion.div
                        initial={{ height: 0, opacity: 0 }}
                        animate={{ height: "auto", opacity: 1 }}
                        exit={{ height: 0, opacity: 0 }}
                        transition={{ duration: 0.2 }}
                        className="overflow-hidden bg-surface-2/50"
                      >
                        <div className="grid grid-cols-2 gap-x-6 gap-y-1.5 px-4 py-3 text-xs sm:grid-cols-3">
                          <Detail label="Timestamp" value={new Date(a.created_at).toLocaleString()} />
                          <Detail label="Actor" value={a.actor_email || "system"} />
                          <Detail label="Result" value={a.result} />
                          <Detail label="IP address" value={a.ip_address || "—"} />
                          <Detail label="Browser" value={a.browser || "—"} />
                          <Detail label="OS" value={a.os || "—"} />
                          <Detail label="Method" value={String(a.metadata?.method ?? "—")} />
                          <Detail label="Path" value={String(a.metadata?.path ?? "—")} mono />
                          <Detail label="Status" value={String(a.metadata?.status ?? "—")} />
                          {a.object_id && <Detail label="Object ID" value={a.object_id} mono />}
                          {a.request_id && <Detail label="Request ID" value={a.request_id} mono />}
                        </div>
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
      <p className={`mt-0.5 break-words ${mono ? "font-mono text-[11px]" : ""}`}>{value}</p>
    </div>
  );
}
