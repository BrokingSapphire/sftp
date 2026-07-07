"use client";

import { useEffect, useRef, useState } from "react";
import { Languages, Check } from "lucide-react";
import { LOCALES, useI18n } from "@/lib/i18n";

/** A compact language selector for the top bar. */
export function LanguagePicker() {
  const { locale, setLocale, t } = useI18n();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  const current = LOCALES.find((l) => l.code === locale) ?? LOCALES[0];

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        title={t("common.language")}
        className="flex h-9 items-center gap-1.5 rounded-lg px-2 text-sm text-muted transition-colors hover:bg-surface-2 hover:text-foreground"
      >
        <Languages size={17} />
        <span className="hidden sm:inline">{current.native}</span>
      </button>

      {open && (
        <div className="absolute right-0 z-50 mt-2 max-h-80 w-52 overflow-y-auto rounded-xl border border-border bg-surface p-1 shadow-xl">
          <p className="px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-muted">{t("common.language")}</p>
          {LOCALES.map((l) => (
            <button
              key={l.code}
              onClick={() => { setLocale(l.code); setOpen(false); }}
              className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm hover:bg-surface-2"
            >
              <span className="flex flex-col">
                <span className="font-medium">{l.native}</span>
                <span className="text-xs text-muted">{l.label}</span>
              </span>
              {l.code === locale && <Check size={15} className="text-primary" />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
