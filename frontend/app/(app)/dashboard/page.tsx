"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import {
  File, FolderOpen, HardDrive, Clock, Star, ArrowRight, Upload, Sparkles, Users, ShieldCheck,
  Building2, CalendarDays, ChevronRight,
} from "lucide-react";
import { useAuth } from "@/lib/auth";
import { filesApi } from "@/lib/endpoints";
import { BRAND } from "@/lib/brand";
import { FilePreview } from "@/components/files/file-preview";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/misc";
import { formatBytes, timeAgo } from "@/lib/utils";
import { fileIcon } from "@/components/files/icon";
import { StaggerList, StaggerItem, motion } from "@/components/motion";

export default function DashboardPage() {
  const { user, has } = useAuth();
  const recent = useQuery({ queryKey: ["recent"], queryFn: () => filesApi.recent() });
  const starred = useQuery({ queryKey: ["starred"], queryFn: () => filesApi.starred() });
  const [preview, setPreview] = useState<number | null>(null);

  const used = user?.storage_used ?? 0;
  const quota = user?.storage_quota ?? 0;
  const pct = quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : 0;

  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Good morning" : hour < 17 ? "Good afternoon" : "Good evening";
  const today = new Date().toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: "numeric" });
  const orgDomain = BRAND.org?.domains?.[0];

  const quickActions = [
    { href: "/files", icon: Upload, title: "Upload files", desc: "Add documents to your private drive", show: true },
    { href: "/common", icon: FolderOpen, title: "Common area", desc: "Share files org-wide, off-quota", show: true },
    ...(BRAND.ai?.enabled ? [{ href: "/ask", icon: Sparkles, title: "Ask your files", desc: "Answers from your documents, on-prem", show: true }] : []),
    { href: "/admin/users", icon: Users, title: "Manage users", desc: "Accounts, roles and access", show: has("users.read") },
    { href: "/admin/audit", icon: ShieldCheck, title: "Audit log", desc: "Every action, immutably recorded", show: has("audit.read") },
  ].filter((a) => a.show).slice(0, 4);

  return (
    <div className="mx-auto max-w-6xl space-y-8">
      {/* Hero */}
      <motion.div
        initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
        className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-primary via-primary to-[#053e42] px-6 py-7 text-white shadow-lg sm:px-8 sm:py-9"
      >
        <div aria-hidden className="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full bg-white/10 blur-2xl" />
        <div aria-hidden className="pointer-events-none absolute -bottom-24 left-1/3 h-56 w-56 rounded-full bg-[#5CC5C9]/20 blur-3xl" />
        <div aria-hidden className="pointer-events-none absolute inset-0 opacity-[0.07]"
          style={{ backgroundImage: "linear-gradient(to right,#fff 1px,transparent 1px),linear-gradient(to bottom,#fff 1px,transparent 1px)", backgroundSize: "40px 40px" }} />
        <div className="relative flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="font-mono text-xs uppercase tracking-[0.18em] text-white/70">{greeting}</p>
            <h1 className="mt-1.5 text-2xl font-semibold tracking-tight sm:text-3xl">{user?.full_name || user?.username}</h1>
            <p className="mt-1 text-sm text-white/75">
              Welcome back to {BRAND.company.product}. Here is an overview of your workspace.
            </p>
          </div>
          <div className="hidden flex-col items-end gap-1 text-right text-xs text-white/70 sm:flex">
            <span className="flex items-center gap-1.5"><CalendarDays size={13} /> {today}</span>
            {orgDomain && <span className="flex items-center gap-1.5"><Building2 size={13} /> {orgDomain}</span>}
            <span className="mt-1 rounded-full bg-white/15 px-2.5 py-0.5 text-[11px] font-medium capitalize">{user?.role?.replace("_", " ")}</span>
          </div>
        </div>
      </motion.div>

      {/* Key metrics */}
      <section>
        <SectionHeading title="At a glance" hint="Your storage, files and access at a glance" />
        <StaggerList className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Stat icon={HardDrive} label="Storage used" value={formatBytes(used)}
            sub={quota > 0 ? `${pct}% of ${formatBytes(quota)} quota` : "Unlimited quota"} tint="teal" />
          <Stat icon={File} label="Recent files" value={String(recent.data?.length ?? 0)}
            sub="Added in the last period" loading={recent.isLoading} tint="blue" />
          <Stat icon={Star} label="Starred" value={String(starred.data?.length ?? 0)}
            sub="Pinned for quick access" loading={starred.isLoading} tint="amber" />
          <Stat icon={ShieldCheck} label="Access level" value={user?.role?.replace("_", " ") ?? "—"}
            sub="Your role and permissions" capitalize tint="violet" />
        </StaggerList>
      </section>

      {/* Quick actions */}
      <section>
        <SectionHeading title="Quick actions" hint="Jump straight to the things you do most" />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {quickActions.map((a) => (
            <Link key={a.href} href={a.href}
              className="group flex items-start gap-3 rounded-xl border border-border bg-surface p-4 shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md">
              <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
                <a.icon size={19} />
              </span>
              <div className="min-w-0">
                <p className="flex items-center gap-1 text-sm font-semibold">{a.title}<ChevronRight size={14} className="opacity-0 transition-opacity group-hover:opacity-100" /></p>
                <p className="mt-0.5 text-xs leading-snug text-muted">{a.desc}</p>
              </div>
            </Link>
          ))}
        </div>
      </section>

      {/* Recent + storage */}
      <section className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader className="flex-row items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2"><Clock size={16} /> Recent uploads</CardTitle>
              <p className="mt-0.5 text-xs text-muted">Your most recently added files across your drive</p>
            </div>
            <Link href="/recent" className="flex shrink-0 items-center gap-1 text-sm font-medium text-primary hover:underline">
              View all <ArrowRight size={14} />
            </Link>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-[1fr_5rem_6rem] gap-3 border-b border-border px-3 pb-2 text-[11px] font-medium uppercase tracking-wider text-muted">
              <span>Name</span><span className="text-right">Size</span><span className="text-right">Added</span>
            </div>
            <div className="mt-1 space-y-0.5">
              {recent.isLoading && [...Array(5)].map((_, i) => <Skeleton key={i} className="h-10 w-full" />)}
              {recent.data?.length === 0 && (
                <p className="py-10 text-center text-sm text-muted">No files yet — head to <Link href="/files" className="text-primary hover:underline">My Files</Link> to upload your first document.</p>
              )}
              {recent.data?.map((f, i) => (
                <button key={f.id} onClick={() => setPreview(i)}
                  className="grid w-full grid-cols-[1fr_5rem_6rem] items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-surface-2">
                  <span className="flex min-w-0 items-center gap-3">
                    {fileIcon(f.extension, 18)}
                    <span className="truncate text-sm font-medium">{f.name}</span>
                  </span>
                  <span className="text-right text-xs text-muted">{formatBytes(f.size_bytes)}</span>
                  <span className="text-right text-xs text-muted">{timeAgo(f.created_at)}</span>
                </button>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Storage / account panel */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><HardDrive size={16} /> Storage</CardTitle>
            <p className="mt-0.5 text-xs text-muted">Usage against your allocated quota</p>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <div className="mb-2 flex items-baseline justify-between">
                <span className="text-2xl font-semibold tracking-tight">{formatBytes(used)}</span>
                <span className="text-xs text-muted">{quota > 0 ? `of ${formatBytes(quota)}` : "unlimited"}</span>
              </div>
              <div className="h-2.5 w-full overflow-hidden rounded-full bg-surface-2">
                <motion.div className="h-full rounded-full bg-primary" initial={{ width: 0 }}
                  animate={{ width: quota > 0 ? `${pct}%` : "6%" }} transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1] }} />
              </div>
              <p className="mt-2 text-xs text-muted">
                {quota > 0 ? `${formatBytes(Math.max(0, quota - used))} remaining (${100 - pct}% free)` : "No quota limit on your account"}
              </p>
            </div>
            <div className="space-y-2 border-t border-border pt-3 text-sm">
              <InfoRow label="Account" value={user?.email ?? "—"} />
              <InfoRow label="Username" value={user?.username ?? "—"} />
              <InfoRow label="Role" value={(user?.role ?? "").replace("_", " ")} capitalize />
            </div>
          </CardContent>
        </Card>
      </section>

      {preview !== null && recent.data?.[preview] && (
        <FilePreview files={recent.data} index={preview} onChangeIndex={setPreview}
          onClose={() => setPreview(null)} onChanged={() => recent.refetch()} />
      )}
    </div>
  );
}

function SectionHeading({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="mb-3">
      <h2 className="text-sm font-semibold tracking-tight">{title}</h2>
      {hint && <p className="text-xs text-muted">{hint}</p>}
    </div>
  );
}

function InfoRow({ label, value, capitalize }: { label: string; value: string; capitalize?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-xs text-muted">{label}</span>
      <span className={`min-w-0 truncate text-right text-xs font-medium ${capitalize ? "capitalize" : ""}`}>{value}</span>
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
            <div className="min-w-0">
              <p className="eyebrow">{label}</p>
              {loading ? (
                <Skeleton className="mt-2 h-7 w-16" />
              ) : (
                <p className={`mt-1.5 truncate text-2xl font-semibold tracking-tight ${capitalize ? "capitalize" : ""}`}>{value}</p>
              )}
              {sub && <p className="mt-0.5 text-xs text-muted">{sub}</p>}
            </div>
            <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl shadow-sm ${c.icon}`}>
              <Icon size={20} />
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </StaggerItem>
  );
}
