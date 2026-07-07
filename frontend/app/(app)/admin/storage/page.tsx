"use client";

import { useQuery } from "@tanstack/react-query";
import { HardDrive, Infinity as InfinityIcon } from "lucide-react";
import { storageApi, type MediaSlice } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge, Skeleton } from "@/components/ui/misc";
import { StaggerList, StaggerItem } from "@/components/motion";
import { formatBytes, cn } from "@/lib/utils";

const CATEGORY_COLOR: Record<string, string> = {
  images: "#a855f7",
  video: "#f43f5e",
  audio: "#f59e0b",
  documents: "#3b82f6",
  spreadsheets: "#16a34a",
  presentations: "#ea580c",
  archives: "#ca8a04",
  code: "#0d9488",
  other: "#6b7280",
};
const colorFor = (c: string) => CATEGORY_COLOR[c] ?? "#6b7280";

export default function AdminStoragePage() {
  const q = useQuery({ queryKey: ["storage-overview"], queryFn: () => storageApi.overview() });

  const media = q.data?.media ?? [];
  const total = q.data?.system_used ?? 0;
  const users = q.data?.users ?? [];

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <PageHeader icon={HardDrive} title="Storage" subtitle="Organisation-wide storage consumption and media breakdown" />

      {q.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[20rem_1fr]">
          {/* Media donut */}
          <Card>
            <CardHeader><CardTitle>Media breakdown</CardTitle></CardHeader>
            <CardContent className="flex flex-col items-center">
              <Donut slices={media} total={total} />
              <div className="mt-4 w-full space-y-1.5">
                {media.map((m) => (
                  <div key={m.category} className="flex items-center gap-2 text-sm">
                    <span className="h-2.5 w-2.5 rounded-full" style={{ background: colorFor(m.category) }} />
                    <span className="flex-1 capitalize">{m.category}</span>
                    <span className="text-muted">{formatBytes(m.total)}</span>
                  </div>
                ))}
                {media.length === 0 && <p className="text-center text-sm text-muted">No files yet.</p>}
              </div>
            </CardContent>
          </Card>

          {/* Per-user usage */}
          <Card>
            <CardHeader className="flex-row items-center justify-between">
              <CardTitle>Usage by user</CardTitle>
              <span className="text-sm text-muted">Total: <span className="font-medium text-foreground">{formatBytes(total)}</span></span>
            </CardHeader>
            <CardContent>
              <StaggerList className="space-y-3">
                {users.map((u) => (
                  <StaggerItem key={u.id} className="rounded-lg border border-border p-3">
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/15 text-xs font-semibold text-primary">
                        {(u.full_name || u.username).slice(0, 2).toUpperCase()}
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">{u.full_name || u.username}</p>
                        <p className="text-xs text-muted">{u.email} · <span className="capitalize">{u.role.replace("_", " ")}</span> · {u.file_count} files</p>
                      </div>
                      <div className="text-right text-sm">
                        <span className="font-medium">{formatBytes(u.storage_used)}</span>
                        <span className="text-muted"> / {u.unlimited ? "∞" : formatBytes(u.storage_quota)}</span>
                      </div>
                    </div>
                    <div className="mt-2 flex items-center gap-2">
                      <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-2">
                        <div
                          className={cn("h-full rounded-full", u.percent_used >= 90 ? "bg-danger" : u.percent_used >= 70 ? "bg-warning" : "bg-primary")}
                          style={{ width: u.unlimited ? "6%" : `${u.percent_used}%` }}
                        />
                      </div>
                      {u.unlimited
                        ? <Badge className="gap-1"><InfinityIcon size={11} /> Unlimited</Badge>
                        : <span className="w-9 text-right text-xs text-muted">{u.percent_used}%</span>}
                    </div>
                  </StaggerItem>
                ))}
                {users.length === 0 && <p className="py-8 text-center text-sm text-muted">No users.</p>}
              </StaggerList>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}

// Inline SVG donut (no chart dependency).
function Donut({ slices, total }: { slices: MediaSlice[]; total: number }) {
  const size = 168, stroke = 26, r = (size - stroke) / 2, C = 2 * Math.PI * r;
  let acc = 0;
  return (
    <div className="relative">
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--surface-2)" strokeWidth={stroke} />
        {total > 0 && slices.map((s) => {
          const frac = s.total / total;
          const dash = frac * C;
          const el = (
            <circle
              key={s.category}
              cx={size / 2} cy={size / 2} r={r} fill="none"
              stroke={colorFor(s.category)} strokeWidth={stroke}
              strokeDasharray={`${dash} ${C - dash}`} strokeDashoffset={-acc * C}
            />
          );
          acc += frac;
          return el;
        })}
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <HardDrive size={18} className="text-muted" />
        <span className="mt-1 text-sm font-semibold">{formatBytes(total)}</span>
        <span className="text-[10px] text-muted">total</span>
      </div>
    </div>
  );
}
