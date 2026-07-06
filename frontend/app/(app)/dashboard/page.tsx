"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { File, FolderOpen, HardDrive, Clock, Star, ArrowRight } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { filesApi } from "@/lib/endpoints";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/misc";
import { formatBytes, timeAgo } from "@/lib/utils";
import { fileIcon } from "@/components/files/icon";
import { StaggerList, StaggerItem, motion } from "@/components/motion";

export default function DashboardPage() {
  const { user } = useAuth();
  const recent = useQuery({ queryKey: ["recent"], queryFn: () => filesApi.recent() });
  const starred = useQuery({ queryKey: ["starred"], queryFn: () => filesApi.starred() });

  const used = user?.storage_used ?? 0;
  const quota = user?.storage_quota ?? 0;
  const pct = quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : 0;

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">
          Welcome back, {user?.full_name?.split(" ")[0] || user?.username}
        </h1>
        <p className="text-sm text-muted">Here is what is happening in your workspace.</p>
      </div>

      <StaggerList className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat icon={HardDrive} label="Storage used" value={formatBytes(used)}
          sub={quota > 0 ? `of ${formatBytes(quota)}` : "unlimited"} />
        <Stat icon={File} label="Recent files" value={String(recent.data?.length ?? 0)}
          sub="last 20" loading={recent.isLoading} />
        <Stat icon={Star} label="Starred" value={String(starred.data?.length ?? 0)}
          sub="quick access" loading={starred.isLoading} />
        <Stat icon={FolderOpen} label="Role" value={user?.role?.replace("_", " ") ?? ""}
          sub="access level" capitalize />
      </StaggerList>

      {quota > 0 && (
        <Card>
          <CardHeader><CardTitle>Storage</CardTitle></CardHeader>
          <CardContent>
            <div className="mb-2 flex items-center justify-between text-sm">
              <span className="text-muted">{formatBytes(used)} used</span>
              <span className="font-medium">{pct}%</span>
            </div>
            <div className="h-2.5 w-full overflow-hidden rounded-full bg-surface-2">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: `${pct}%` }}
              />
            </div>
            <p className="mt-2 text-xs text-muted">{formatBytes(quota - used)} remaining</p>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle className="flex items-center gap-2"><Clock size={16} /> Recent uploads</CardTitle>
          <Link href="/recent" className="flex items-center gap-1 text-sm text-primary hover:underline">
            View all <ArrowRight size={14} />
          </Link>
        </CardHeader>
        <CardContent className="space-y-1">
          {recent.isLoading && [...Array(4)].map((_, i) => <Skeleton key={i} className="h-11 w-full" />)}
          {recent.data?.length === 0 && <p className="py-6 text-center text-sm text-muted">No files yet — upload something.</p>}
          {recent.data?.map((f) => (
            <a
              key={f.id}
              href={filesApi.downloadUrl(f.id)}
              className="flex items-center gap-3 rounded-lg px-3 py-2 transition-colors hover:bg-surface-2"
            >
              {fileIcon(f.extension, 18)}
              <span className="min-w-0 flex-1 truncate text-sm font-medium">{f.name}</span>
              <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
              <span className="hidden w-24 text-right text-xs text-muted sm:block">{timeAgo(f.created_at)}</span>
            </a>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

function Stat({
  icon: Icon, label, value, sub, loading, capitalize,
}: {
  icon: React.ElementType; label: string; value: string; sub?: string; loading?: boolean; capitalize?: boolean;
}) {
  return (
    <StaggerItem>
      <motion.div whileHover={{ y: -3 }} transition={{ type: "spring", stiffness: 380, damping: 30 }}>
        <Card className="transition-shadow hover:shadow-md">
          <CardContent className="flex items-start justify-between p-5">
            <div>
              <p className="eyebrow">{label}</p>
              {loading ? (
                <Skeleton className="mt-2 h-7 w-16" />
              ) : (
                <p className={`mt-1.5 text-2xl font-semibold tracking-tight ${capitalize ? "capitalize" : ""}`}>{value}</p>
              )}
              {sub && <p className="mt-0.5 text-xs text-muted">{sub}</p>}
            </div>
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Icon size={20} />
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </StaggerItem>
  );
}
