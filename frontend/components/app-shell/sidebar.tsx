"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard, Folder, Globe, Star, Share2, Trash2, Inbox,
  Users, ScrollText, KeyRound, HardDrive, PieChart, ShieldAlert, Sparkles, DatabaseBackup, UsersRound,
} from "lucide-react";
import { BRAND } from "@/lib/brand";
import { useAuth } from "@/lib/auth";
import { useI18n, type TKey } from "@/lib/i18n";
import { cn, formatBytes } from "@/lib/utils";
import { Logo } from "@/components/logo";
import { motion } from "motion/react";

interface NavItem {
  href: string;
  label: TKey;
  icon: React.ElementType;
  perm?: string;
  superAdmin?: boolean;
}

const primary: NavItem[] = [
  { href: "/dashboard", label: "nav.dashboard", icon: LayoutDashboard },
  { href: "/files", label: "nav.files", icon: Folder },
  ...(BRAND.ai?.enabled ? [{ href: "/ask", label: "nav.askAI" as TKey, icon: Sparkles }] : []),
  { href: "/teams", label: "nav.teams", icon: UsersRound },
  { href: "/common", label: "nav.common", icon: Globe },
  { href: "/shared", label: "nav.shared_with_me", icon: Users },
  { href: "/inherited", label: "nav.inherited", icon: Inbox },
  { href: "/starred", label: "nav.starred", icon: Star },
  { href: "/shares", label: "nav.shared", icon: Share2 },
  { href: "/trash", label: "nav.trash", icon: Trash2 },
];

const admin: NavItem[] = [
  { href: "/admin/users", label: "nav.users", icon: Users, perm: "users.read" },
  { href: "/admin/storage", label: "nav.storage", icon: PieChart, perm: "storage.manage" },
  { href: "/admin/audit", label: "nav.audit", icon: ScrollText, perm: "audit.read" },
  { href: "/admin/security", label: "nav.security", icon: ShieldAlert, perm: "audit.read" },
  { href: "/admin/backup", label: "nav.backup", icon: DatabaseBackup, superAdmin: true },
  { href: "/api-keys", label: "nav.apiKeys", icon: KeyRound, perm: "apikeys.manage" },
];

/** Desktop sidebar (hidden on small screens; the mobile drawer reuses SidebarNav). */
export function Sidebar() {
  return (
    <aside className="hidden w-64 shrink-0 flex-col border-r border-border bg-surface md:flex">
      <div className="flex h-16 items-center px-5">
        <Logo size={30} />
      </div>
      <SidebarNav />
    </aside>
  );
}

/** The nav + storage meter, shared by the desktop sidebar and the mobile drawer. */
export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  const { has, user } = useAuth();
  const { t } = useI18n();

  const link = (item: NavItem) => {
    const active = pathname === item.href || pathname.startsWith(item.href + "/");
    const Icon = item.icon;
    return (
      <Link
        key={item.href}
        href={item.href}
        onClick={onNavigate}
        className={cn(
          "group relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
          active ? "text-primary" : "text-muted hover:text-foreground",
        )}
      >
        {active && (
          <motion.span
            layoutId="nav-active"
            className="absolute inset-0 rounded-lg bg-primary/10"
            transition={{ type: "spring", stiffness: 400, damping: 32 }}
          />
        )}
        <Icon
          size={18}
          className={cn("relative z-10 transition-transform group-hover:scale-110", active && "text-primary")}
        />
        <span className="relative z-10">{t(item.label)}</span>
      </Link>
    );
  };

  const visibleAdmin = admin.filter((a) => a.superAdmin ? user?.role === "super_admin" : (!a.perm || has(a.perm)));
  const used = user?.storage_used ?? 0;
  const quota = user?.storage_quota ?? 0;
  const pct = quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : 0;

  return (
    <>
      <nav className="flex-1 space-y-0.5 overflow-y-auto px-3 py-2">
        {primary.map(link)}
        {visibleAdmin.length > 0 && (
          <>
            <div className="eyebrow px-3 pb-1 pt-5">{t("nav.administration")}</div>
            {visibleAdmin.map(link)}
          </>
        )}
      </nav>

      {/* Google-Drive-style storage meter */}
      <div className="border-t border-border p-4">
        <div className="mb-2 flex items-center gap-2 text-xs font-medium text-muted">
          <HardDrive size={14} />
          {t("nav.storage")}
        </div>
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-surface-2">
          <motion.div
            className="h-full rounded-full bg-primary"
            initial={{ width: 0 }}
            animate={{ width: quota > 0 ? `${pct}%` : "8%" }}
            transition={{ duration: 0.9, ease: [0.22, 1, 0.36, 1] }}
          />
        </div>
        <p className="mt-2 text-xs text-muted">
          <span className="font-medium text-foreground">{formatBytes(used)}</span>
          {quota > 0 ? ` of ${formatBytes(quota)}` : " used · unlimited"}
        </p>
      </div>
    </>
  );
}
