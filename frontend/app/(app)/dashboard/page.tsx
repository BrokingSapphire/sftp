"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { File, FolderOpen, HardDrive, Clock, Star, ArrowRight } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { filesApi } from "@/lib/endpoints";
import { FilePreview } from "@/components/files/file-preview";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/misc";
import { formatBytes, timeAgo } from "@/lib/utils";
import { fileIcon } from "@/components/files/icon";
import { StaggerList, StaggerItem, motion } from "@/components/motion";

export default function DashboardPage() {
  const { user } = useAuth();
  const recent = useQuery({ queryKey: ["recent"], queryFn: () => filesApi.recent() });
  const starred = useQuery({ queryKey: ["starred"], queryFn: () => filesApi.starred() });
  const [preview, setPreview] = useState<number | null>(null);

  const used = user?.storage_used ?? 0;
  const quota = user?.storage_quota ?? 0;
  const pct = quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : 0;

  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Good morning" : hour < 17 ? "Good afternoon" : "Good evening";

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      {/* Gradient hero */}
      <motion.div
        initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
        className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-primary via-primary to-[#053e42] px-6 py-7 text-white shadow-lg sm:px-8 sm:py-9"
      >
        <div aria-hidden className="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full bg-white/10 blur-2xl" />
        <div aria-hidden className="pointer-events-none absolute -bottom-24 left-1/3 h-56 w-56 rounded-full bg-[#5CC5C9]/20 blur-3xl" />
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 opacity-[0.07]"
          style={{ backgroundImage: "linear-gradient(to right,#fff 1px,transparent 1px),linear-gradient(to bottom,#fff 1px,transparent 1px)", backgroundSize: "40px 40px" }}
        />
        <div className="relative">
          <p className="font-mono text-xs uppercase tracking-[0.18em] text-white/70">{greeting}</p>
          <h1 className="mt-1.5 text-2xl font-semibold tracking-tight sm:text-3xl">
            {user?.full_name?.split(" ")[0] || user?.username}
          </h1>
          <p className="mt-1 text-sm text-white/75">Here is what is happening in your workspace.</p>
        </div>
      </motion.div>

      <StaggerList className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat icon={HardDrive} label="Storage used" value={formatBytes(used)}
          sub={quota > 0 ? `of ${formatBytes(quota)}` : "unlimited"} tint="teal" />
        <Stat icon={File} label="Recent files" value={String(recent.data?.length ?? 0)}
          sub="last 20" loading={recent.isLoading} tint="blue" />
        <Stat icon={Star} label="Starred" value={String(starred.data?.length ?? 0)}
          sub="quick access" loading={starred.isLoading} tint="amber" />
        <Stat icon={FolderOpen} label="Role" value={user?.role?.replace("_", " ") ?? ""}
          sub="access level" capitalize tint="violet" />
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
          {recent.data?.map((f, i) => (
            <button
              key={f.id}
              onClick={() => setPreview(i)}
              className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-surface-2"
            >
              {fileIcon(f.extension, 18)}
              <span className="min-w-0 flex-1 truncate text-sm font-medium">{f.name}</span>
              <span className="text-xs text-muted">{formatBytes(f.size_bytes)}</span>
              <span className="hidden w-24 text-right text-xs text-muted sm:block">{timeAgo(f.created_at)}</span>
            </button>
          ))}
        </CardContent>
      </Card>

      {preview !== null && recent.data?.[preview] && (
        <FilePreview
          files={recent.data}
          index={preview}
          onChangeIndex={setPreview}
          onClose={() => setPreview(null)}
          onChanged={() => recent.refetch()}
        />
      )}
    </div>
  );
}

const TINTS: Record<string, { icon: string; glow: string }> = {
  teal: { icon: "bg-gradient-to-br from-[#0d9488] to-[#064D51] text-white", glow: "from-teal-500/10" },
  blue: { icon: "bg-gradient-to-br from-sky-500 to-blue-600 text-white", glow: "from-sky-500/10" },
  amber: { icon: "bg-gradient-to-br from-amber-400 to-orange-500 text-white", glow: "from-amber-500/10" },
  violet: { icon: "bg-gradient-to-br from-violet-500 to-purple-600 text-white", glow: "from-violet-500/10" },
};

function Stat({
  icon: Icon, label, value, sub, loading, capitalize, tint = "teal",
}: {
  icon: React.ElementType; label: string; value: string; sub?: string; loading?: boolean; capitalize?: boolean; tint?: keyof typeof TINTS;
}) {
  const c = TINTS[tint] ?? TINTS.teal;
  return (
    <StaggerItem>
      <motion.div whileHover={{ y: -3 }} transition={{ type: "spring", stiffness: 380, damping: 30 }}>
        <Card className="relative overflow-hidden transition-shadow hover:shadow-md">
          <div aria-hidden className={`pointer-events-none absolute -right-8 -top-8 h-24 w-24 rounded-full bg-gradient-to-br ${c.glow} to-transparent blur-2xl`} />
          <CardContent className="relative flex items-start justify-between p-5">
            <div>
              <p className="eyebrow">{label}</p>
              {loading ? (
                <Skeleton className="mt-2 h-7 w-16" />
              ) : (
                <p className={`mt-1.5 text-2xl font-semibold tracking-tight ${capitalize ? "capitalize" : ""}`}>{value}</p>
              )}
              {sub && <p className="mt-0.5 text-xs text-muted">{sub}</p>}
            </div>
            <div className={`flex h-10 w-10 items-center justify-center rounded-xl shadow-sm ${c.icon}`}>
              <Icon size={20} />
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </StaggerItem>
  );
}
