"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ShieldAlert, ShieldCheck, Download, Trash2, Share2, KeyRound, Check, Clock } from "lucide-react";
import { securityApi, type SecurityAlert } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/misc";
import { StaggerList, StaggerItem } from "@/components/motion";
import { timeAgo, cn } from "@/lib/utils";

const severityStyle: Record<string, string> = {
  high: "text-danger bg-danger/10 border-danger/30",
  medium: "text-warning bg-warning/10 border-warning/30",
  low: "text-muted bg-surface-2 border-border",
};

const typeIcon: Record<string, React.ElementType> = {
  mass_download: Download,
  bulk_delete: Trash2,
  share_spike: Share2,
  failed_login_burst: KeyRound,
};

const typeLabel: Record<string, string> = {
  mass_download: "Mass download",
  bulk_delete: "Bulk deletion",
  share_spike: "Share spike",
  failed_login_burst: "Failed-login burst",
};

export default function SecurityPage() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["security-alerts"], queryFn: () => securityApi.list() });
  const [showResolved, setShowResolved] = useState(false);

  const alerts = (q.data ?? []).filter((a) => showResolved || !a.resolved);
  const openCount = (q.data ?? []).filter((a) => !a.resolved).length;

  async function resolve(a: SecurityAlert) {
    try {
      await securityApi.resolve(a.id);
      toast.success("Alert resolved");
      qc.invalidateQueries({ queryKey: ["security-alerts"] });
      qc.invalidateQueries({ queryKey: ["security-unresolved"] });
    } catch { toast.error("Could not resolve"); }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={ShieldAlert} title="Security" subtitle="Anomalies detected on the audit stream — exfiltration, brute force, bulk actions" />
        <label className="flex items-center gap-2 text-xs text-muted">
          <input type="checkbox" checked={showResolved} onChange={(e) => setShowResolved(e.target.checked)} />
          Show resolved
        </label>
      </div>

      <div className="flex items-center gap-2 rounded-xl border border-border bg-surface px-4 py-3">
        {openCount > 0 ? <ShieldAlert size={18} className="text-danger" /> : <ShieldCheck size={18} className="text-success" />}
        <p className="text-sm">
          {openCount > 0
            ? <><span className="font-semibold text-danger">{openCount}</span> open alert{openCount === 1 ? "" : "s"} need review.</>
            : "No open alerts — all clear."}
        </p>
      </div>

      {q.isLoading && <Skeleton className="h-48 w-full" />}

      {!q.isLoading && alerts.length === 0 && (
        <div className="flex min-h-[14rem] flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-surface text-muted">
          <ShieldCheck size={40} />
          <p className="text-sm">Nothing to show.</p>
        </div>
      )}

      <StaggerList>
        {alerts.map((a) => {
          const Icon = typeIcon[a.type] ?? ShieldAlert;
          return (
            <StaggerItem key={a.id}>
              <Card className={cn("mb-2", a.resolved && "opacity-60")}>
                <CardContent className="flex items-center gap-3 p-4">
                  <span className={cn("flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border", severityStyle[a.severity] ?? severityStyle.low)}>
                    <Icon size={18} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">{typeLabel[a.type] ?? a.type}</span>
                      <span className={cn("rounded-full border px-2 py-0.5 text-[10px] font-medium uppercase", severityStyle[a.severity] ?? severityStyle.low)}>{a.severity}</span>
                      {a.resolved && <span className="text-[10px] text-success">resolved</span>}
                    </div>
                    <p className="mt-0.5 truncate text-sm text-muted">{a.summary}</p>
                    <p className="mt-0.5 flex items-center gap-1 text-[11px] text-muted"><Clock size={11} /> {timeAgo(a.created_at)}</p>
                  </div>
                  {!a.resolved && (
                    <Button variant="outline" size="sm" onClick={() => resolve(a)}><Check size={14} /> Resolve</Button>
                  )}
                </CardContent>
              </Card>
            </StaggerItem>
          );
        })}
      </StaggerList>
    </div>
  );
}
