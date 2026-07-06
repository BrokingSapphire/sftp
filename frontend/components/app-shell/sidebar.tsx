"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard, Folder, Globe, Star, Share2, Trash2,
  Users, ScrollText, KeyRound, HardDrive, PieChart,
} from "lucide-react";
import { useAuth } from "@/lib/auth";
import { cn, formatBytes } from "@/lib/utils";
import { Logo } from "@/components/logo";
import { motion } from "motion/react";

interface NavItem {
  href: string;
  label: string;
  icon: React.ElementType;
  perm?: string;
}

const primary: NavItem[] = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/files", label: "My Files", icon: Folder },
  { href: "/common", label: "Common", icon: Globe },
  { href: "/starred", label: "Starred", icon: Star },
  { href: "/shares", label: "Shared", icon: Share2 },
  { href: "/trash", label: "Trash", icon: Trash2 },
];

const admin: NavItem[] = [
  { href: "/admin/users", label: "Users", icon: Users, perm: "users.read" },
  { href: "/admin/storage", label: "Storage", icon: PieChart, perm: "storage.manage" },
  { href: "/admin/audit", label: "Audit Log", icon: ScrollText, perm: "audit.read" },
  { href: "/api-keys", label: "API Keys", icon: KeyRound, perm: "apikeys.manage" },
];

export function Sidebar() {
  const pathname = usePathname();
  const { has, user } = useAuth();

  const link = (item: NavItem) => {
    const active = pathname === item.href || pathname.startsWith(item.href + "/");
    const Icon = item.icon;
    return (
      <Link
        key={item.href}
        href={item.href}
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
        <span className="relative z-10">{item.label}</span>
      </Link>
    );
  };

  const visibleAdmin = admin.filter((a) => !a.perm || has(a.perm));
  const used = user?.storage_used ?? 0;
  const quota = user?.storage_quota ?? 0;
  const pct = quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : 0;

  return (
    <aside className="hidden w-64 shrink-0 flex-col border-r border-border bg-surface md:flex">
      <div className="flex h-16 items-center px-5">
        <Logo size={30} />
      </div>

      <nav className="flex-1 space-y-0.5 overflow-y-auto px-3 py-2">
        {primary.map(link)}
        {visibleAdmin.length > 0 && (
          <>
            <div className="eyebrow px-3 pb-1 pt-5">Administration</div>
            {visibleAdmin.map(link)}
          </>
        )}
      </nav>

      {/* Google-Drive-style storage meter */}
      <div className="border-t border-border p-4">
        <div className="mb-2 flex items-center gap-2 text-xs font-medium text-muted">
          <HardDrive size={14} />
          Storage
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
    </aside>
  );
}
