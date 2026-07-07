"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { LogOut, Search, Camera } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { avatarApi } from "@/lib/endpoints";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar } from "@/components/ui/avatar";
import { ThemeToggle } from "@/components/theme-toggle";
import { PingIndicator } from "@/components/app-shell/ping";
import { NotificationBell } from "@/components/app-shell/notification-bell";
import { LanguagePicker } from "@/components/app-shell/language-picker";
import { useI18n } from "@/lib/i18n";

export function Topbar() {
  const { user, refreshUser } = useAuth();
  const { t } = useI18n();
  const router = useRouter();
  const [q, setQ] = useState("");
  const avatarInput = useRef<HTMLInputElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const inField = /^(INPUT|TEXTAREA|SELECT)$/.test((e.target as HTMLElement)?.tagName || "");
      if (((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") || (e.key === "/" && !inField)) {
        e.preventDefault();
        searchRef.current?.focus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  function onSearch(e: React.FormEvent) {
    e.preventDefault();
    if (q.trim()) router.push(`/search?q=${encodeURIComponent(q.trim())}`);
  }

  async function onAvatar(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    if (!file.type.startsWith("image/")) return toast.error("Please choose an image");
    try { await avatarApi.upload(file); await refreshUser(); toast.success("Profile photo updated"); }
    catch { toast.error("Could not update photo"); }
  }

  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-4 border-b border-border bg-surface/80 px-6 backdrop-blur">
      <form onSubmit={onSearch} className="relative max-w-md flex-1">
        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
        <Input
          ref={searchRef}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t("action.search")}
          className="pl-9 pr-14"
        />
        <kbd className="pointer-events-none absolute right-2.5 top-1/2 hidden -translate-y-1/2 rounded border border-border bg-surface-2 px-1.5 py-0.5 font-mono text-[10px] text-muted sm:block">⌘K</kbd>
      </form>
      <div className="ml-auto flex items-center gap-2">
        <PingIndicator />
        <NotificationBell />
        <LanguagePicker />
        <ThemeToggle />
        <div className="flex items-center gap-2 rounded-lg border border-border px-2.5 py-1.5">
          <button
            onClick={() => avatarInput.current?.click()}
            title="Change profile photo"
            className="group relative h-7 w-7 shrink-0 rounded-full"
          >
            <Avatar userId={user?.id} name={user?.full_name || user?.username || "?"} hasAvatar={user?.has_avatar} size={28} />
            <span className="absolute inset-0 flex items-center justify-center rounded-full bg-black/45 opacity-0 transition-opacity group-hover:opacity-100">
              <Camera size={13} className="text-white" />
            </span>
          </button>
          <input ref={avatarInput} type="file" accept="image/*" hidden onChange={onAvatar} />
          <div className="hidden text-sm leading-tight sm:block">
            <div className="font-medium">{user?.full_name || user?.username}</div>
            <div className="text-xs capitalize text-muted">{user?.role?.replace("_", " ")}</div>
          </div>
        </div>
        <Button variant="ghost" size="icon" aria-label="Sign out" onClick={() => router.push("/logout")}>
          <LogOut size={18} />
        </Button>
      </div>
    </header>
  );
}
