"use client";

import { useQuery } from "@tanstack/react-query";
import { auditApi } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Card } from "@/components/ui/card";
import { Badge, Skeleton } from "@/components/ui/misc";
import { timeAgo } from "@/lib/utils";

const resultColor: Record<string, string> = {
  success: "text-success",
  failure: "text-danger",
  denied: "text-warning",
};

export default function AuditPage() {
  const q = useQuery({ queryKey: ["audit"], queryFn: () => auditApi.list(150) });

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <PageHeader title="Audit log" subtitle="Immutable, compliance-grade record of every action" />
      {q.isLoading && <Skeleton className="h-96 w-full" />}
      <Card className="overflow-hidden">
        <div className="grid grid-cols-[10rem_1fr_6rem_7rem] gap-3 border-b border-border bg-surface-2 px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
          <span>Actor</span><span>Action</span><span>Result</span><span className="text-right">When</span>
        </div>
        <div className="max-h-[70vh] overflow-y-auto">
          {q.data?.length === 0 && <p className="py-16 text-center text-sm text-muted">No audit events.</p>}
          {q.data?.map((a) => (
            <div key={a.id} className="grid grid-cols-[10rem_1fr_6rem_7rem] items-center gap-3 border-b border-border/50 px-4 py-2 text-sm hover:bg-surface-2">
              <span className="truncate text-muted">{a.actor_email || "system"}</span>
              <span className="flex min-w-0 items-center gap-2">
                <Badge>{a.category}</Badge>
                <span className="truncate font-mono text-xs">{a.action}</span>
              </span>
              <span className={`text-xs font-medium capitalize ${resultColor[a.result] ?? ""}`}>{a.result}</span>
              <span className="text-right text-xs text-muted">{timeAgo(a.created_at)}</span>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}
