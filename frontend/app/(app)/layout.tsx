"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { useI18n, type LocaleCode } from "@/lib/i18n";
import { Sidebar } from "@/components/app-shell/sidebar";
import { MobileNav } from "@/components/app-shell/mobile-nav";
import { Topbar } from "@/components/app-shell/topbar";
import { WelcomeModal } from "@/components/app-shell/welcome-modal";
import { Spinner } from "@/components/ui/misc";
import { Telemetry } from "@/components/telemetry";
import { IdleTimeout } from "@/components/idle-timeout";
import { ForcePasswordChange } from "@/components/force-password-change";
import { UploadProvider } from "@/lib/upload-manager";
import { UploadPanel } from "@/components/files/upload-panel";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const [mobileNav, setMobileNav] = useState(false);

  useEffect(() => {
    if (!loading && !user) router.replace("/login");
  }, [loading, user, router]);

  // Close the mobile drawer whenever the route changes.
  useEffect(() => { setMobileNav(false); }, [pathname]);

  // Apply the user's saved language on login so it follows them across devices.
  const { setLocale, locale } = useI18n();
  useEffect(() => {
    const lang = user?.language;
    if (lang && lang !== locale) setLocale(lang as LocaleCode);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.language]);

  if (loading || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner className="h-6 w-6" />
      </div>
    );
  }

  // First login / after an admin reset: force a password change before anything else.
  if (user.must_change_pw) {
    return <ForcePasswordChange />;
  }

  return (
    <UploadProvider>
      <div className="flex h-screen overflow-hidden">
        <Telemetry />
        <IdleTimeout />
        <WelcomeModal />
        <Sidebar />
        <MobileNav open={mobileNav} onClose={() => setMobileNav(false)} />
        <div className="flex min-w-0 flex-1 flex-col">
          <Topbar onMenu={() => setMobileNav(true)} />
          <main key={pathname} className="flex-1 overflow-y-auto p-4 sm:p-6">
            {children}
          </main>
        </div>
        <UploadPanel />
      </div>
    </UploadProvider>
  );
}
