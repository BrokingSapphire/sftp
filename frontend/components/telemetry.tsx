"use client";

import { useEffect, useRef } from "react";
import { usePathname } from "next/navigation";
import { auditApi } from "@/lib/endpoints";

/**
 * Global UI telemetry — records page views and every interactive click
 * (buttons, links, role=button, [data-track]) to the /activity stream so the
 * platform keeps a fine-grained record of user interaction. Mounted inside the
 * authenticated shell only.
 */
export function Telemetry() {
  const pathname = usePathname();
  const last = useRef<{ label: string; t: number }>({ label: "", t: 0 });

  // Page view on every route change.
  useEffect(() => {
    auditApi.track("view", undefined, pathname);
  }, [pathname]);

  // Delegated click capture.
  useEffect(() => {
    function onClick(e: MouseEvent) {
      const target = e.target as HTMLElement | null;
      const el = target?.closest<HTMLElement>(
        "button, a, [role='button'], [data-track]",
      );
      if (!el) return;

      const label =
        el.getAttribute("data-track") ||
        el.getAttribute("aria-label") ||
        el.getAttribute("title") ||
        el.textContent?.trim().slice(0, 60) ||
        el.tagName.toLowerCase();

      // De-dupe identical clicks fired within 400ms (double-fire guards).
      const now = Date.now();
      if (last.current.label === label && now - last.current.t < 400) return;
      last.current = { label, t: now };

      auditApi.track("click", label, window.location.pathname, {
        tag: el.tagName.toLowerCase(),
        href: el.getAttribute("href") ?? undefined,
      });
    }

    document.addEventListener("click", onClick, { capture: true });
    return () => document.removeEventListener("click", onClick, { capture: true });
  }, []);

  return null;
}
