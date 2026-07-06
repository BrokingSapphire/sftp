"use client";

import Link from "next/link";
import { motion } from "motion/react";
import { Home, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { BRAND } from "@/lib/brand";

export default function NotFound() {
  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden px-6 text-center">
      <div className="pointer-events-none absolute -top-40 left-1/2 h-96 w-[42rem] -translate-x-1/2 rounded-full bg-primary/10 blur-3xl" />

      <motion.div
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
        className="relative"
      >
        <div className="flex items-center justify-center gap-3">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={BRAND.logo.full} alt={BRAND.company.shortName} width={44} height={44} />
        </div>
        <h1 className="mt-6 bg-gradient-to-b from-foreground to-muted bg-clip-text text-8xl font-bold tracking-tighter text-transparent">
          404
        </h1>
        <p className="mt-2 text-lg font-medium">This page took a wrong turn</p>
        <p className="mt-1 max-w-sm text-sm text-muted">
          The file, folder, or page you are looking for does not exist or may have been moved.
        </p>
        <div className="mt-6 flex items-center justify-center gap-2">
          <Link href="/dashboard"><Button size="sm"><Home size={16} /> Back to dashboard</Button></Link>
          <Link href="/files"><Button variant="outline" size="sm"><Search size={16} /> Browse files</Button></Link>
        </div>
      </motion.div>

      <p className="mt-12 font-mono text-[11px] uppercase tracking-wider text-muted">{BRAND.company.product}</p>
    </div>
  );
}
