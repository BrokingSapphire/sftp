"use client";

/**
 * App-wide replacement for the browser's blocking window.confirm / window.prompt.
 *
 * Mount <DialogProvider> once (in Providers) and call the promise-based helpers
 * from anywhere:
 *
 *   const { confirm, prompt } = useDialogs();
 *   if (await confirm({ title: "Delete?", tone: "danger" })) …
 *   const name = await prompt({ title: "New folder", placeholder: "Name" });
 *
 * confirm resolves to a boolean; prompt resolves to the string, or null if the
 * user cancels. Enter submits, Escape / backdrop cancels.
 */

import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { motion, AnimatePresence } from "motion/react";
import { AlertTriangle, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type Tone = "primary" | "danger";

interface ConfirmOptions {
  title: string;
  message?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: Tone;
}
interface PromptOptions {
  title: string;
  message?: ReactNode;
  defaultValue?: string;
  placeholder?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  type?: "text" | "password" | "number";
  required?: boolean;
}

type ConfirmState = ConfirmOptions & { kind: "confirm"; resolve: (v: boolean) => void };
type PromptState = PromptOptions & { kind: "prompt"; resolve: (v: string | null) => void };
type DialogState = ConfirmState | PromptState;

interface DialogApi {
  confirm: (opts: ConfirmOptions) => Promise<boolean>;
  prompt: (opts: PromptOptions) => Promise<string | null>;
}

const DialogContext = createContext<DialogApi | null>(null);

export function useDialogs(): DialogApi {
  const ctx = useContext(DialogContext);
  if (!ctx) throw new Error("useDialogs must be used within <DialogProvider>");
  return ctx;
}

export function DialogProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<DialogState | null>(null);

  const confirm = useCallback(
    (opts: ConfirmOptions) => new Promise<boolean>((resolve) => setState({ ...opts, kind: "confirm", resolve })),
    [],
  );
  const prompt = useCallback(
    (opts: PromptOptions) => new Promise<string | null>((resolve) => setState({ ...opts, kind: "prompt", resolve })),
    [],
  );

  function close(result: boolean | string | null) {
    if (!state) return;
    if (state.kind === "confirm") state.resolve(result as boolean);
    else state.resolve(result as string | null);
    setState(null);
  }

  return (
    <DialogContext.Provider value={{ confirm, prompt }}>
      {children}
      <AnimatePresence>
        {state && <DialogView key="dlg" state={state} onCancel={() => close(state.kind === "confirm" ? false : null)} onConfirm={close} />}
      </AnimatePresence>
    </DialogContext.Provider>
  );
}

function DialogView({ state, onCancel, onConfirm }: { state: DialogState; onCancel: () => void; onConfirm: (v: boolean | string) => void }) {
  const isPrompt = state.kind === "prompt";
  const [value, setValue] = useState(isPrompt ? (state as PromptState).defaultValue ?? "" : "");
  const inputRef = useRef<HTMLInputElement>(null);
  const danger = !isPrompt && (state as ConfirmState).tone === "danger";

  useEffect(() => {
    if (isPrompt) {
      const t = setTimeout(() => { inputRef.current?.focus(); inputRef.current?.select(); }, 30);
      return () => clearTimeout(t);
    }
  }, [isPrompt]);

  useEffect(() => {
    function onKey(e: KeyboardEvent) { if (e.key === "Escape") onCancel(); }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onCancel]);

  function submit() {
    if (isPrompt) {
      const v = value.trim();
      if ((state as PromptState).required !== false && v === "") return; // block empty
      onConfirm(v);
    } else {
      onConfirm(true);
    }
  }

  const confirmLabel = state.confirmLabel ?? (isPrompt ? "Save" : danger ? "Delete" : "Confirm");
  const cancelLabel = state.cancelLabel ?? "Cancel";

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4" onClick={onCancel}>
      <motion.div
        initial={{ opacity: 0, scale: 0.96, y: 8 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.96, y: 8 }}
        transition={{ type: "spring", stiffness: 320, damping: 26 }}
        className="w-full max-w-sm overflow-hidden rounded-2xl border border-border bg-surface shadow-xl"
        onClick={(e) => e.stopPropagation()}
        role="dialog" aria-modal="true"
      >
        <div className="flex items-start gap-3 p-5">
          {danger && (
            <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-danger/10 text-danger">
              <AlertTriangle size={18} />
            </span>
          )}
          <div className="min-w-0 flex-1">
            <div className="flex items-start justify-between gap-2">
              <h3 className="text-base font-semibold leading-tight">{state.title}</h3>
              <button onClick={onCancel} className="-mr-1 -mt-1 text-muted hover:text-foreground"><X size={18} /></button>
            </div>
            {state.message && <p className="mt-1.5 text-sm text-muted">{state.message}</p>}
            {isPrompt && (
              <Input
                ref={inputRef}
                type={(state as PromptState).type ?? "text"}
                value={value}
                placeholder={(state as PromptState).placeholder}
                onChange={(e) => setValue(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); submit(); } }}
                className="mt-3"
              />
            )}
          </div>
        </div>
        <div className="flex justify-end gap-2 border-t border-border bg-surface-2/50 px-5 py-3">
          <Button variant="outline" size="sm" onClick={onCancel}>{cancelLabel}</Button>
          <Button variant={danger ? "danger" : "primary"} size="sm" onClick={submit}>{confirmLabel}</Button>
        </div>
      </motion.div>
    </div>
  );
}
