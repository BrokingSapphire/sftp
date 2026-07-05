"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { HardDrive, Loader2 } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ThemeToggle } from "@/components/theme-toggle";
import { ApiError } from "@/lib/api";

const schema = z.object({
  identifier: z.string().min(1, "Email or username is required"),
  password: z.string().min(1, "Password is required"),
  remember: z.boolean().optional(),
});
type FormValues = z.infer<typeof schema>;

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
      toast.error(e instanceof ApiError ? e.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  const ssoEnabled = process.env.NEXT_PUBLIC_MICROSOFT_SSO === "true";

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-4">
      <div className="pointer-events-none absolute -top-40 left-1/2 h-96 w-[42rem] -translate-x-1/2 rounded-full bg-primary/20 blur-3xl" />
      <div className="absolute right-4 top-4">
        <ThemeToggle />
      </div>

      <div className="animate-in w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-lg">
            <HardDrive size={24} />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Sapphire SFTP</h1>
          <p className="mt-1 text-sm text-muted">Enterprise file transfer platform</p>
        </div>

        <div className="rounded-2xl border border-border bg-surface p-6 shadow-xl">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
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
              <div className="my-5 flex items-center gap-3 text-xs text-muted">
                <span className="h-px flex-1 bg-border" /> OR <span className="h-px flex-1 bg-border" />
              </div>
              <a href="/api/v1/auth/sso/microsoft/login">
                <Button variant="outline" className="w-full" type="button">
                  Continue with Microsoft
                </Button>
              </a>
            </>
          )}
        </div>
        <p className="mt-6 text-center text-xs text-muted">
          Self-hosted • On-premise • Your data never leaves your network
        </p>
      </div>
    </div>
  );
}
