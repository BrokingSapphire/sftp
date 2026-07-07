"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Loader2, ShieldCheck, Server, Lock } from "lucide-react";
import { motion } from "motion/react";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ThemeToggle } from "@/components/theme-toggle";
import { Logo } from "@/components/logo";
import { BRAND } from "@/lib/brand";
import { ApiError } from "@/lib/api";

const schema = z.object({
  identifier: z.string().min(1, "Email or username is required"),
  password: z.string().min(1, "Password is required"),
  remember: z.boolean().optional(),
});
type FormValues = z.infer<typeof schema>;

const ease = [0.22, 1, 0.36, 1] as const;

export default function LoginPage() {
  const { login } = useAuth();
  const [submitting, setSubmitting] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { remember: true } });

  async function onSubmit(values: FormValues) {
    setSubmitting(true);
    try {
      await login(values.identifier, values.password, values.remember ?? false);
      toast.success("Welcome back");
    } catch (e) {
      // 409 → a session is already active elsewhere. Offer to take it over.
      if (e instanceof ApiError && e.status === 409) {
        if (confirm("A session is already active for this account on another device.\n\nLog it out and continue here?")) {
          try {
            await login(values.identifier, values.password, values.remember ?? false, true);
            toast.success("Signed in — other session ended");
          } catch (e2) {
            toast.error(e2 instanceof ApiError ? e2.message : "Login failed");
          }
        }
      } else {
        toast.error(e instanceof ApiError ? e.message : "Login failed");
      }
    } finally {
      setSubmitting(false);
    }
  }

  const ssoEnabled = BRAND.sso?.microsoft?.enabled ?? false;

  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      {/* ── Brand panel ── */}
      <div className="relative hidden overflow-hidden bg-[#053e42] text-white lg:flex lg:flex-col lg:justify-between lg:p-12">
        <motion.div
          aria-hidden
          className="pointer-events-none absolute -right-24 -top-24 h-96 w-96 rounded-full bg-white/5 blur-2xl"
          animate={{ scale: [1, 1.15, 1], opacity: [0.5, 0.8, 0.5] }}
          transition={{ duration: 8, repeat: Infinity, ease: "easeInOut" }}
        />
        <motion.div
          aria-hidden
          className="pointer-events-none absolute -bottom-32 -left-16 h-96 w-96 rounded-full bg-teal-400/10 blur-3xl"
          animate={{ scale: [1, 1.2, 1] }}
          transition={{ duration: 10, repeat: Infinity, ease: "easeInOut" }}
        />
        {/* faint ledger grid — old-school detail */}
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 opacity-[0.06]"
          style={{
            backgroundImage:
              "linear-gradient(to right, #fff 1px, transparent 1px), linear-gradient(to bottom, #fff 1px, transparent 1px)",
            backgroundSize: "44px 44px",
          }}
        />

        <motion.div initial={{ opacity: 0, y: -8 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, ease }}>
          <div className="flex items-center gap-2.5">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={BRAND.logo.full} alt={BRAND.company.shortName} width={34} height={34} className="drop-shadow" />
            <span className="text-lg font-semibold tracking-tight">{BRAND.company.shortName}</span>
          </div>
        </motion.div>

        <div className="relative max-w-md">
          <motion.p
            className="font-mono text-xs uppercase tracking-[0.2em] text-teal-200/80"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.2, duration: 0.6 }}
          >
            Enterprise file transfer
          </motion.p>
          <motion.h1
            className="mt-4 text-4xl font-semibold leading-tight tracking-tight"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.28, duration: 0.6, ease }}
          >
            Your files, on your terms.
          </motion.h1>
          <motion.p
            className="mt-4 text-sm leading-relaxed text-teal-100/70"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.38, duration: 0.6, ease }}
          >
            Self-hosted, on-premise file management for the whole firm — resumable
            uploads, granular access control, and a full compliance audit trail.
          </motion.p>

          <div className="mt-10 space-y-3">
            {[
              { icon: Server, text: "100% on-premise — data never leaves your network" },
              { icon: ShieldCheck, text: "Argon2id, RBAC, and an immutable audit log" },
              { icon: Lock, text: "Password-protected, expiring share links" },
            ].map((f, i) => (
              <motion.div
                key={i}
                className="flex items-center gap-3 text-sm text-teal-50/90"
                initial={{ opacity: 0, x: -12 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: 0.5 + i * 0.1, duration: 0.5, ease }}
              >
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-white/10">
                  <f.icon size={16} />
                </span>
                {f.text}
              </motion.div>
            ))}
          </div>
        </div>

        <motion.p
          className="relative font-mono text-xs text-teal-200/50"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.9 }}
        >
          © {new Date().getFullYear()} {BRAND.company.copyright}
        </motion.p>
      </div>

      {/* ── Form panel ── */}
      <div className="relative flex items-center justify-center px-6 py-12">
        <div className="absolute right-4 top-4">
          <ThemeToggle />
        </div>

        <motion.div
          className="w-full max-w-sm"
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, ease }}
        >
          <div className="mb-8 lg:hidden">
            <Logo size={32} />
          </div>

          <p className="eyebrow">Welcome back</p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight">Sign in to your workspace</h2>
          <p className="mt-1 text-sm text-muted">Enter your credentials to continue.</p>

          <form onSubmit={handleSubmit(onSubmit)} className="mt-8 space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="identifier">Email or username</label>
              <Input id="identifier" autoComplete="username" placeholder="you@company.com" {...register("identifier")} />
              {errors.identifier && <p className="text-xs text-danger">{errors.identifier.message}</p>}
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="password">Password</label>
              <Input id="password" type="password" autoComplete="current-password" placeholder="••••••••••••" {...register("password")} />
              {errors.password && <p className="text-xs text-danger">{errors.password.message}</p>}
            </div>
            <label className="flex items-center gap-2 text-sm text-muted">
              <input type="checkbox" className="accent-[var(--primary)]" {...register("remember")} />
              Remember me for 30 days
            </label>
            <Button type="submit" className="w-full" disabled={submitting}>
              {submitting && <Loader2 size={16} className="animate-spin" />}
              Sign in
            </Button>
          </form>

          {ssoEnabled && (
            <>
              <div className="my-6 flex items-center gap-3 text-xs text-muted">
                <span className="h-px flex-1 bg-border" /> OR <span className="h-px flex-1 bg-border" />
              </div>
              <a href="/api/v1/auth/sso/microsoft/login">
                <Button variant="outline" className="w-full" type="button">Continue with Microsoft</Button>
              </a>
            </>
          )}

          <p className="mt-8 font-mono text-[11px] uppercase tracking-wider text-muted">
            Self-hosted · On-premise · Zero cloud
          </p>
        </motion.div>
      </div>
    </div>
  );
}
