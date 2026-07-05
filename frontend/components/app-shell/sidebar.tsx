"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard, Folder, Clock, Star, Share2, Trash2,
  Users, ScrollText, KeyRound, HardDrive,
} from "lucide-react";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/utils";

interface NavItem {
  href: string;
  label: string;
  icon: React.ElementType;
  perm?: string;
}

const primary: NavItem[] = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/files", label: "My Files", icon: Folder },
  { href: "/recent", label: "Recent", icon: Clock },
  { href: "/starred", label: "Starred", icon: Star },
  { href: "/shares", label: "Shared", icon: Share2 },
  { href: "/trash", label: "Trash", icon: Trash2 },
];

const admin: NavItem[] = [
  { href: "/admin/users", label: "Users", icon: Users, perm: "users.read" },
  { href: "/admin/audit", label: "Audit Log", icon: ScrollText, perm: "audit.read" },
  { href: "/api-keys", label: "API Keys", icon: KeyRound, perm: "apikeys.manage" },
];

export function Sidebar() {
  const pathname = usePathname();
  const { has } = useAuth();

  const link = (item: NavItem) => {
    const active = pathname === item.href || pathname.startsWith(item.href + "/");
    const Icon = item.icon;
    return (
      <Link
        key={item.href}
        href={item.href}
        className={cn(
          "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
          active
            ? "bg-primary/10 text-primary"
            : "text-muted hover:bg-surface-2 hover:text-foreground",
        )}
      >
        <Icon size={18} />
        {item.label}
      </Link>
    );
  };

  const visibleAdmin = admin.filter((a) => !a.perm || has(a.perm));

  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r border-border bg-surface md:flex">
      <div className="flex h-16 items-center gap-2 px-5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <HardDrive size={18} />
        </div>
        <span className="font-semibold tracking-tight">Sapphire SFTP</span>
      </div>
      <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-2">
        {primary.map(link)}
        {visibleAdmin.length > 0 && (
          <>
            <div className="px-3 pb-1 pt-4 text-xs font-semibold uppercase tracking-wider text-muted">
              Administration
            </div>
            {visibleAdmin.map(link)}
          </>
        )}
      </nav>
    </aside>
  );
}
