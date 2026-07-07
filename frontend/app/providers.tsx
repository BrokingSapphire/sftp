"use client";

import { useState, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "next-themes";
import { Toaster } from "sonner";
import { AuthProvider } from "@/lib/auth";
import { I18nProvider } from "@/lib/i18n";

export function Providers({ children }: { children: ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          // Keep the UI fresh: refetch on mount and when the tab regains focus,
          // and treat data as immediately stale so navigation shows current data
          // without a manual refresh.
          queries: { staleTime: 0, gcTime: 5 * 60_000, retry: 1, refetchOnWindowFocus: true, refetchOnMount: true },
        },
      }),
  );

  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <QueryClientProvider client={client}>
        <I18nProvider>
          <AuthProvider>{children}</AuthProvider>
        </I18nProvider>
        <Toaster richColors closeButton position="top-right" duration={3000} />
      </QueryClientProvider>
    </ThemeProvider>
  );
}
