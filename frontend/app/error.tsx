"use client";

import { useEffect } from "react";
import Link from "next/link";
import { motion } from "motion/react";
import { AlertTriangle, Home, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { BRAND } from "@/lib/brand";

export default function Error({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    // Surface for debugging; in production this would go to a logger.
    console.error(error);
  }, [error]);

  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden px-6 text-center">
      <div className="pointer-events-none absolute -top-40 left-1/2 h-96 w-[42rem] -translate-x-1/2 rounded-full bg-danger/10 blur-3xl" />

      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
        className="relative max-w-md"
      >
        <motion.div
          initial={{ rotate: -8, scale: 0.8 }}
          animate={{ rotate: 0, scale: 1 }}
          transition={{ type: "spring", stiffness: 260, damping: 18 }}
          className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-danger/10 text-danger"
        >
          <AlertTriangle size={30} />
        </motion.div>
        <h1 className="mt-6 text-2xl font-semibold tracking-tight">Something went wrong</h1>
        <p className="mt-2 text-sm text-muted">
          An unexpected error occurred. You can try again, or head back to your dashboard.
        </p>
        {error.digest && (
          <p className="mt-3 inline-block rounded-md bg-surface-2 px-2.5 py-1 font-mono text-[11px] text-muted">
            ref: {error.digest}
          </p>
        )}
        <div className="mt-6 flex items-center justify-center gap-2">
          <Button size="sm" onClick={reset}><RotateCcw size={16} /> Try again</Button>
          <Link href="/dashboard"><Button variant="outline" size="sm"><Home size={16} /> Dashboard</Button></Link>
        </div>
      </motion.div>

      <p className="mt-12 font-mono text-[11px] uppercase tracking-wider text-muted">{BRAND.company.product}</p>
    </div>
  );
}
