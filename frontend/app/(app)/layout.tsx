"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { Sidebar } from "@/components/app-shell/sidebar";
import { Topbar } from "@/components/app-shell/topbar";
 import { WelcomeModal } from "@/components/app-shell/welcome-modal";
import { Spinner } from "@/components/ui/misc";
import { Telemetry } from "@/components/telemetry";
import { ForcePasswordChange } from "@/components/force-password-change";
import { UploadProvider } from "@/lib/upload-manager";
import { UploadPanel } from "@/components/files/upload-panel";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !user) router.replace("/login");
  }, [loading, user, router]);

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
        <WelcomeModal />
        <Sidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <Topbar />
          <main key={pathname} className="flex-1 overflow-y-auto p-6">
            {children}
          </main>
        </div>
        <UploadPanel />
      </div>
    </UploadProvider>
  );
}
