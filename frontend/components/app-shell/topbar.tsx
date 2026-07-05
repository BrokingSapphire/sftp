"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { LogOut, Search, User } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ThemeToggle } from "@/components/theme-toggle";

export function Topbar() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const [q, setQ] = useState("");

  function onSearch(e: React.FormEvent) {
    e.preventDefault();
    if (q.trim()) router.push(`/files?q=${encodeURIComponent(q.trim())}`);
  }

  const initials = (user?.full_name || user?.username || "?")
    .split(" ")
    .map((s) => s[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();

  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-4 border-b border-border bg-surface/80 px-6 backdrop-blur">
      <form onSubmit={onSearch} className="relative max-w-md flex-1">
        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search files…"
          className="pl-9"
        />
      </form>
      <div className="ml-auto flex items-center gap-2">
        <ThemeToggle />
        <div className="flex items-center gap-2 rounded-lg border border-border px-2.5 py-1.5">
          <div className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/15 text-xs font-semibold text-primary">
            {initials || <User size={14} />}
          </div>
          <div className="hidden text-sm leading-tight sm:block">
            <div className="font-medium">{user?.full_name || user?.username}</div>
            <div className="text-xs capitalize text-muted">{user?.role?.replace("_", " ")}</div>
          </div>
        </div>
        <Button variant="ghost" size="icon" aria-label="Sign out" onClick={() => logout()}>
          <LogOut size={18} />
        </Button>
      </div>
    </header>
  );
}
